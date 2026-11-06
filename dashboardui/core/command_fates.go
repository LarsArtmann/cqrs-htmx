package core

import (
	"context"
	"encoding/json/v2"
	"time"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Command fate labels — the terminal or attempt-level outcome of a dispatched
// command, derived from commandlifecycle event types.
const (
	// FateRejected: a business rule refused the command (no state changed,
	// no retry, no dead letter).
	FateRejected = "rejected"

	// FateFailed: one processing attempt failed with a retryable error.
	FateFailed = "failed"

	// FateRetried: another attempt was scheduled (Attempt is the 1-indexed
	// retry number).
	FateRetried = "retried"

	// FateDeadLettered: all retry attempts are exhausted.
	FateDeadLettered = "dead-lettered"

	// FateCompleted: the command was processed successfully.
	FateCompleted = "completed"
)

// CommandFate is one row of the command-fates view: what happened to a
// dispatched command, decoded from its command.rejected/failed/retried/
// dead-lettered/completed lifecycle event.
type CommandFate struct {
	// Time is when the fate was recorded (the payload's timestamp).
	Time time.Time

	// CommandType is the type of the command the fate belongs to.
	CommandType string

	// CommandID identifies the command. Payload value when the payload
	// carries one; otherwise the lifecycle stream ID (streams are keyed by
	// command ID, so the fallback is the same identity).
	CommandID string

	// Fate is one of the Fate* labels, or the raw event type for lifecycle
	// events this build does not know (they stay visible rather than
	// silently dropped).
	Fate string

	// Attempt is the 1-indexed attempt or retry number the fate describes
	// (0 when the payload carries none).
	Attempt int

	// Error is the recorded error message ("" for completed/retried).
	Error string

	// Family is the errorfamily classification stamped at rejection time
	// ("" for non-rejection fates).
	Family string

	// ErrorCode is the machine-readable error code when the rejecting error
	// chain declared one.
	ErrorCode string

	// EventID is the lifecycle event's ID, for drill-down into the event
	// browser.
	EventID id.EventID
}

// LoadCommandFates reads the newest command fates from the journal — every
// commandlifecycle event except command.received (arrival is not a fate; it
// would double every row with no new information). The scan pages the whole
// journal keeping a rolling tail (journals read oldest-first, so a zero-cursor
// window is the HEAD — the gotcha the events page already documents), then
// decodes payloads. Fates this build cannot decode (schema drift, or an event
// type newer than this dashboard) stay visible with the raw event type as
// their Fate. The second return value is the total number of fate events in
// the journal; the returned slice is newest-first.
func LoadCommandFates(
	ctx context.Context,
	cfg Config,
	limit int,
) ([]CommandFate, int, error) {
	var (
		fates []CommandFate
		total int
	)

	keep := func(batch []event.Event) {
		fates, total = keepFates(fates, batch, limit, total)
	}

	if cfg.SeekableJournal != nil {
		var after id.EventID

		for {
			batch, err := cfg.SeekableJournal.ReadFrom(ctx, after, tailScanBatch)
			if err != nil {
				return nil, 0, errorfamily.WrapInfrastructure(err,
					"dashboardui.command_fates.read_failed", "read journal for command fates").
					WithContextAny("limit", limit)
			}

			keep(batch)

			if len(batch) < tailScanBatch {
				break
			}

			next := batch[len(batch)-1].ID()
			if next == after {
				return nil, 0, errorfamily.NewInfrastructure("dashboardui.command_fates.cursor_stalled",
					"journal cursor did not advance while reading command fates")
			}

			after = next
		}
	} else if cfg.Journal != nil {
		all, err := cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, 0, errorfamily.WrapInfrastructure(err,
				"dashboardui.command_fates.read_all_failed", "read journal for command fates").
				WithContextAny("limit", limit)
		}

		keep(all)
	}

	// Newest first for display: the scan is oldest-first.
	for i, j := 0, len(fates)-1; i < j; i, j = i+1, j-1 {
		fates[i], fates[j] = fates[j], fates[i]
	}

	return fates, total, nil
}

// keepFates folds one journal batch into the rolling fates window,
// counting every fate event (the total is the whole journal's fate count;
// the window only ever holds the newest limit rows).
func keepFates(fates []CommandFate, batch []event.Event, limit, total int) ([]CommandFate, int) {
	for _, evt := range batch {
		if evt.StreamType() != commandlifecycle.StreamTypeCommandLifecycle {
			continue
		}

		fate, ok := decodeCommandFate(evt)
		if !ok {
			continue
		}

		total++

		fates = append(fates, fate)

		if len(fates) > limit {
			trimmed := make([]CommandFate, 0, limit)
			trimmed = append(trimmed, fates[len(fates)-limit:]...)
			fates = trimmed
		}
	}

	return fates, total
}

// decodeCommandFate maps one lifecycle event to a CommandFate row. ok is
// false only for command.received events, which are deliberately not fates.
// A payload that fails to decode still yields a row — with the raw event
// type as the Fate — so schema drift stays visible instead of hiding rows.
func decodeCommandFate(evt event.Event) (CommandFate, bool) {
	switch evt.Type() {
	case commandlifecycle.TypeReceived:
		var zero CommandFate

		return zero, false
	case commandlifecycle.TypeFailed:
		return failedFate(evt), true
	case commandlifecycle.TypeRejected:
		return rejectedFate(evt), true
	case commandlifecycle.TypeRetried:
		return retriedFate(evt), true
	case commandlifecycle.TypeDeadLettered:
		return deadLetteredFate(evt), true
	case commandlifecycle.TypeCompleted:
		return completedFate(evt), true
	default:
		// An event type newer than this build: keep it visible with the raw
		// type as the fate, so nothing on a CommandLifecycle stream silently
		// disappears from the view.
		row := newCommandFateRow(evt, string(evt.Type()))
		row.CommandType = string(evt.Type())

		return row, true
	}
}

// newCommandFateRow seeds a row with the fate label and the stream-keyed
// command identity; the per-fate decoders fill in the payload fields.
//
// The stream ID is read in its bare (Get) form on purpose: a lifecycle stream
// is keyed by the command ID, so the bare value is exactly what the payload
// carries (upstream records CommandKey(cmd.ID().String())), and StreamID's
// String() adds a second brand prefix ("StreamMarker:…") that no other view
// renders. Row and payload therefore print the same identity (gotcha 25).
func newCommandFateRow(evt event.Event, fate string) CommandFate {
	return CommandFate{
		Fate:      fate,
		CommandID: evt.StreamID().Get(),
		EventID:   evt.ID(),
	}
}

// payloadCommandID prefers the payload's command ID and falls back to the
// stream-keyed identity (FailedPayload.CommandID is empty on payloads
// recorded before the payload carried one).
func payloadCommandID(payloadID commandlifecycle.CommandKey, fallback string) string {
	if payloadID != "" {
		return string(payloadID)
	}

	return fallback
}

func failedFate(evt event.Event) CommandFate {
	row := newCommandFateRow(evt, FateFailed)

	var p commandlifecycle.FailedPayload
	if decodeLifecyclePayload(evt.Payload(), evt.Encoding(), &p) != nil {
		return row
	}

	row.CommandType = p.CommandType
	row.Attempt = p.Attempt
	row.Error = p.Error
	row.Time = p.FailedAt
	row.CommandID = payloadCommandID(p.CommandID, row.CommandID)

	return row
}

func rejectedFate(evt event.Event) CommandFate {
	row := newCommandFateRow(evt, FateRejected)

	var p commandlifecycle.RejectedPayload
	if decodeLifecyclePayload(evt.Payload(), evt.Encoding(), &p) != nil {
		return row
	}

	row.CommandType = p.CommandType
	row.Attempt = p.Attempt
	row.Error = p.Error
	row.Family = p.Family
	row.ErrorCode = p.ErrorCode
	row.Time = p.RejectedAt
	row.CommandID = payloadCommandID(p.CommandID, row.CommandID)

	return row
}

func retriedFate(evt event.Event) CommandFate {
	row := newCommandFateRow(evt, FateRetried)

	var p commandlifecycle.RetriedPayload
	if decodeLifecyclePayload(evt.Payload(), evt.Encoding(), &p) != nil {
		return row
	}

	row.CommandType = p.CommandType
	row.Attempt = p.Attempt
	row.Time = p.RetriedAt

	return row
}

func deadLetteredFate(evt event.Event) CommandFate {
	row := newCommandFateRow(evt, FateDeadLettered)

	var p commandlifecycle.DeadLetteredPayload
	if decodeLifecyclePayload(evt.Payload(), evt.Encoding(), &p) != nil {
		return row
	}

	row.CommandType = p.CommandType
	row.Attempt = p.Attempts
	row.Error = p.Error
	row.Time = p.DeadLetteredAt

	return row
}

func completedFate(evt event.Event) CommandFate {
	row := newCommandFateRow(evt, FateCompleted)

	var p commandlifecycle.CompletedPayload
	if decodeLifecyclePayload(evt.Payload(), evt.Encoding(), &p) != nil {
		return row
	}

	row.CommandType = p.CommandType
	row.Time = p.CompletedAt
	row.CommandID = payloadCommandID(p.CommandID, row.CommandID)

	return row
}

// decodeLifecyclePayload decodes a lifecycle payload by the event's
// encoding — the same JSON/CBOR switch the typed payload renderers use
// (commandlifecycle streams are CBOR when the recorder runs the default
// codec). Raw and unknown encodings decode to an error; callers keep the
// row visible undecoded rather than dropping it.
func decodeLifecyclePayload(payload []byte, encoding codec.Encoding, dst any) error {
	switch encoding {
	case codec.EncodingJSON, "":
		return json.Unmarshal(payload, dst)
	case codec.EncodingCBOR:
		return codec.CBORCodec{}.Decode(payload, dst)
	case codec.EncodingRaw:
		return errorfamily.NewCorruption("dashboardui.command_fates.encoding_unsupported",
			"no lifecycle payload decode for raw payload encoding")
	default:
		return errorfamily.NewCorruption("dashboardui.command_fates.encoding_unsupported",
			"no lifecycle payload decode for the event's encoding")
	}
}
