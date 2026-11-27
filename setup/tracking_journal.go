package setup

import (
	"context"
	"encoding/json/v2"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// The signals journal is where client signals live. It is neither the domain
// event journal nor the query log: a click is not an event a projection folds
// over, and it is not a query a user asked. It is composed on go-cqrs-lite's
// adapter core over an engine stream-log backend, so every durable write goes
// through the engine and no engine code was added.
//
// Layout: one collection ([SignalsCollection]), one stream per subject. A
// signal with a user goes to "user:<id>", one with only a device key to
// "client:<id>", the rest to "anonymous". Reading one subject's history is
// then one stream read, and appending to it needs no concurrency protocol:
// signals are append-only facts nobody updates.
//
// Every stored envelope carries received_at, the server's clock at the
// beacon. The signals are the one journal that grows without a domain
// reason, so an app sweeps it by age; an engine's journal sweeper keeps every
// row that has no typed time, and it takes that time from a dated envelope.

const (
	// SignalsCollection is the engine collection that holds client signals.
	SignalsCollection = "signals"

	// anonymousSignalStream holds the signals of visitors known by neither a
	// user nor a device key.
	anonymousSignalStream = "anonymous"

	userSignalStreamPrefix   = "user:"
	clientSignalStreamPrefix = "client:"
)

// SignalsJournal is the durable home of client signals: append-only,
// subject-keyed, and separate from the event and query journals. [New] builds
// one when the bundle's event store is a go-cqrs-lite system event adapter,
// or over [Config.TrackingBackend]. It is a [cqrshtmx.TrackingSink].
type SignalsJournal struct {
	core system.AdapterCore[cqrshtmx.TrackedEvent]
}

var _ cqrshtmx.TrackingSink = (*SignalsJournal)(nil)

// NewSignalsJournal builds a journal over an engine stream-log backend, for
// an app that mounts [cqrshtmx.TrackHandler] itself. Signals are stored as
// JSON envelopes on every engine, so the same code path serves the memory
// engine and the persistent ones. A nil backend is rejected here, at boot,
// instead of failing the first beacon.
func NewSignalsJournal(backend metaengine.StreamLogBackend) (*SignalsJournal, error) {
	if backend == nil {
		return nil, errorfamily.NewRejection(
			"setup.invalid_config",
			"NewSignalsJournal needs an engine backend: pass the stream-log backend of your go-cqrs-lite deployment (sys.EventStore().(*system.EventAdapter).Backend, or an engine such as metaengine.NewMemoryEngine())",
		)
	}

	return newSignalsJournal(backend), nil
}

func newSignalsJournal(backend metaengine.StreamLogBackend) *SignalsJournal {
	return &SignalsJournal{
		core: system.AdapterCore[cqrshtmx.TrackedEvent]{
			Backend:    backend,
			Collection: SignalsCollection,
			Noun:       "signal",
			Serialize:  true,
			Encode:     encodeSignal,
			Decode:     decodeSignal,
			IDOf:       func(cqrshtmx.TrackedEvent) string { return "" },
		},
	}
}

// Record stores a batch, one stream append per subject it names. The batch
// is encoded whole before anything is written, and on an engine that supports
// transactions the appends are one transaction: an error means the batch was
// not stored.
func (j *SignalsJournal) Record(ctx context.Context, events []cqrshtmx.TrackedEvent) error {
	if err := j.ready(); err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	values, err := j.core.ToAny(events)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "setup.signals.encode", "encode the signal batch")
	}

	appendAll := func(ctx context.Context) error {
		for _, batch := range groupSignals(events, values) {
			err := j.core.Backend.StreamAppend(ctx, SignalsCollection, batch.stream, batch.values)
			if err != nil {
				return errorfamily.WrapInfrastructuref(err, "setup.signals.append",
					"append %d signal(s) to stream %q", len(batch.values), batch.stream)
			}
		}

		return nil
	}

	if transactional, ok := j.core.Backend.(metaengine.Transactional); ok {
		err = transactional.RunInTx(ctx, appendAll)
		if err != nil {
			// WrapOnce keeps an append error's own code; only an engine
			// failure of the transaction itself gains the store code.
			return errorfamily.WrapOnce(err, errorfamily.Infrastructure,
				"setup.signals.store", "store the signal batch in one transaction")
		}

		return nil
	}

	return appendAll(ctx)
}

// ForUser returns the signals recorded for one user, oldest first.
func (j *SignalsJournal) ForUser(ctx context.Context, userID id.UserID) ([]cqrshtmx.TrackedEvent, error) {
	return j.loadStream(ctx, userSignalStream(userID))
}

// ForClient returns the signals recorded for one device, oldest first.
func (j *SignalsJournal) ForClient(ctx context.Context, clientID id.ClientID) (
	[]cqrshtmx.TrackedEvent,
	error,
) {
	return j.loadStream(ctx, clientSignalStream(clientID))
}

// All returns every recorded signal, oldest first.
func (j *SignalsJournal) All(ctx context.Context) ([]cqrshtmx.TrackedEvent, error) {
	if err := j.ready(); err != nil {
		return nil, err
	}

	events, err := j.core.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "setup.signals.read_all", "read the signals journal")
	}

	return events, nil
}

// ready answers for a journal that cannot reach an engine: one the bundle
// never built (a nil [Bundle.TrackingJournal]) or a zero value. Both are a
// classified error, never a nil-pointer panic far from the misconfiguration.
func (j *SignalsJournal) ready() error {
	if j != nil && j.core.Backend != nil {
		return nil
	}

	return errorfamily.NewRejection(
		"setup.tracking_without_home",
		"this bundle built no signals journal: set Config.TrackingBackend, or pass the event store go-cqrs-lite's system builds as Config.EventStore",
	)
}

func (j *SignalsJournal) loadStream(ctx context.Context, stream string) ([]cqrshtmx.TrackedEvent, error) {
	if err := j.ready(); err != nil {
		return nil, err
	}

	events, err := j.core.LoadStream(ctx, stream)
	if err != nil {
		return nil, errorfamily.WrapInfrastructuref(err, "setup.signals.read", "read the signal stream %q", stream)
	}

	return events, nil
}

func userSignalStream(userID id.UserID) string { return userSignalStreamPrefix + userID.String() }

func clientSignalStream(clientID id.ClientID) string {
	return clientSignalStreamPrefix + clientID.String()
}

// signalStream names the stream a signal belongs to: the user's when the
// server saw one, else the device's, else the shared anonymous stream.
func signalStream(event cqrshtmx.TrackedEvent) string {
	switch {
	case !event.UserID.IsZero():
		return userSignalStream(event.UserID)
	case !event.ClientID.IsZero():
		return clientSignalStream(event.ClientID)
	default:
		return anonymousSignalStream
	}
}

// signalBatch is the signals of one Record call that share a subject.
type signalBatch struct {
	stream string
	values []any
}

// groupSignals splits an encoded batch by subject, in order of first
// appearance, so every subject is appended to exactly once.
func groupSignals(events []cqrshtmx.TrackedEvent, values []any) []signalBatch {
	var batches []signalBatch

	position := make(map[string]int)

	for index, event := range events {
		stream := signalStream(event)

		if at, seen := position[stream]; seen {
			batches[at].values = append(batches[at].values, values[index])

			continue
		}

		position[stream] = len(batches)
		batches = append(batches, signalBatch{stream: stream, values: []any{values[index]}})
	}

	return batches
}

// signalEnvelope is the stored form of one signal: the signal's own fields
// and received_at. A signal names its time at, so without the second stamp
// an engine would file it as an undated row and never sweep it. The envelope
// carries no event identity (id, stream_type, version): a signal is not an
// event, and an envelope with those keys would meet the engine's unique
// event indexes.
type signalEnvelope struct {
	cqrshtmx.TrackedEvent

	ReceivedAt time.Time `json:"received_at"`
}

// encodeSignal renders one signal as its JSON envelope: the same field names
// [cqrshtmx.TrackedEvent] has on any JSON surface, plus the sweep's stamp.
func encodeSignal(event cqrshtmx.TrackedEvent) (string, error) {
	encoded, err := json.Marshal(signalEnvelope{TrackedEvent: event, ReceivedAt: event.At})
	if err != nil {
		return "", errorfamily.WrapInfrastructure(err, "setup.signals.encode", "encode one signal")
	}

	return string(encoded), nil
}

// decodeSignal parses one envelope back into a signal. A value an engine
// handed back as a decoded JSON object reaches this as re-marshaled bytes.
func decodeSignal(envelope string) (cqrshtmx.TrackedEvent, error) {
	var stored signalEnvelope

	if err := json.Unmarshal([]byte(envelope), &stored); err != nil {
		return cqrshtmx.TrackedEvent{},
			errorfamily.WrapInfrastructure(err, "setup.signals.decode", "decode one signal envelope")
	}

	return stored.TrackedEvent, nil
}
