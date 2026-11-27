package setup_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Client signals need a durable home before the bundle mounts anything that
// collects them. These tests pin the three ways a home is named, what the
// mounted endpoint attributes, and the one log line an app without a home
// gets instead of silence.

const (
	trackEndpoint   = "/-/track"
	trackScript     = "/-/tracker.js"
	pageview        = `{"kind":"pageview","path":"/items"}`
	anonymousStream = "anonymous"
)

var errSignalsDiskFull = errors.New("disk full")

// appendRecorder wraps an engine and remembers every stream append in call
// order, so a test can prove one append per subject. It adds no capability:
// the journal sees exactly the interfaces the wrapped engine has.
type appendRecorder struct {
	metaengine.StreamLogBackend

	mu      sync.Mutex
	streams []string
	values  []any // every value handed to the engine, in call order
	failAt  int   // the 1-based append that fails; 0 = none
}

func (r *appendRecorder) StreamAppend(ctx context.Context, collection, stream string, values []any) error {
	r.mu.Lock()
	r.streams = append(r.streams, stream)
	r.values = append(r.values, values...)
	call := len(r.streams)
	r.mu.Unlock()

	if call == r.failAt {
		return errSignalsDiskFull
	}

	return r.StreamLogBackend.StreamAppend(ctx, collection, stream, values)
}

func (r *appendRecorder) appended() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.streams...)
}

// storedValues returns every envelope the journal handed to the engine.
func (r *appendRecorder) storedValues() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored := make([]string, 0, len(r.values))

	for _, value := range r.values {
		envelope, _ := value.(string)
		stored = append(stored, envelope)
	}

	return stored
}

// transactionalRecorder is an appendRecorder on an engine that supports
// transactions.
type transactionalRecorder struct {
	appendRecorder

	transactions int
}

func (r *transactionalRecorder) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	r.transactions++

	return fn(ctx)
}

// decodingBackend hands stored envelopes back as decoded JSON objects, the
// way the SQL engines return TEXT values, so the journal is read through the
// same path a persistent engine takes.
type decodingBackend struct{ metaengine.StreamLogBackend }

func (b decodingBackend) StreamRead(ctx context.Context, collection, stream string) ([]any, error) {
	values, err := b.StreamLogBackend.StreamRead(ctx, collection, stream)

	return decodeEnvelopes(values, err)
}

func (b decodingBackend) JournalReadAll(ctx context.Context, collection string) ([]any, error) {
	values, err := b.StreamLogBackend.JournalReadAll(ctx, collection)

	return decodeEnvelopes(values, err)
}

func decodeEnvelopes(values []any, err error) ([]any, error) {
	if err != nil {
		return nil, err
	}

	decoded := make([]any, 0, len(values))

	for _, value := range values {
		envelope, ok := value.(string)
		if !ok {
			decoded = append(decoded, value)

			continue
		}

		var object map[string]any
		if err := json.Unmarshal([]byte(envelope), &object); err != nil {
			return nil, err
		}

		decoded = append(decoded, object)
	}

	return decoded, nil
}

// memoryStreamLog is a real engine narrowed to the interface the journal
// writes through.
func memoryStreamLog(t *testing.T) metaengine.StreamLogBackend {
	t.Helper()

	engine, ok := metaengine.NewMemoryEngine().(metaengine.StreamLogBackend)
	if !ok {
		t.Fatal("the memory engine must store streams")
	}

	return engine
}

// trackingSinkSpy is a consumer sink that remembers every batch.
type trackingSinkSpy struct {
	mu      sync.Mutex
	batches [][]cqrshtmx.TrackedEvent
}

func (s *trackingSinkSpy) Record(_ context.Context, events []cqrshtmx.TrackedEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.batches = append(s.batches, events)

	return nil
}

func (s *trackingSinkSpy) recorded() [][]cqrshtmx.TrackedEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([][]cqrshtmx.TrackedEvent(nil), s.batches...)
}

// trackingBundle builds a bundle whose logger discards (lines == nil) or
// writes text lines into the buffer, and closes it when the test ends.
func trackingBundle(t *testing.T, config setup.Config, lines *bytes.Buffer) *setup.Bundle {
	t.Helper()

	if lines == nil {
		config.Logger = slog.New(slog.DiscardHandler)
	} else {
		config.Logger = slog.New(slog.NewTextHandler(lines, nil))
	}

	bundle, err := setup.New(config)
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}

	t.Cleanup(func() { _ = bundle.Close() })

	return bundle
}

// beacon is one request the way tracker.js sends it.
func beacon(events string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, trackEndpoint, strings.NewReader(`{"events":[`+events+`]}`))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func serveBundle(bundle *setup.Bundle, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	bundle.Handler(http.NewServeMux()).ServeHTTP(recorder, request)

	return recorder
}

// trackingProbePaths are the tracking routes the path table can mount, in
// manifest order.
func trackingProbePaths() []string {
	return []string{trackEndpoint, "/beacons", trackScript, "/beacons.js"}
}

func postureAt(rows []cqrshtmx.RoutePosture, pattern string) (cqrshtmx.RoutePosture, bool) {
	for _, row := range rows {
		if row.Pattern == pattern {
			return row, true
		}
	}

	return cqrshtmx.RoutePosture{}, false
}

func fullSignal() cqrshtmx.TrackedEvent {
	userID := id.NewUserID()

	return cqrshtmx.TrackedEvent{
		Kind:      cqrshtmx.TrackedClick,
		Path:      "/items/42",
		Referrer:  "https://g.example/search",
		Name:      "add-to-cart",
		Target:    "button#buy",
		VisibleMS: 1500,
		At:        time.Date(2026, time.November, 27, 6, 30, 0, 0, time.UTC),
		UserID:    userID,
		ActorID:   cqrshtmx.ActorIDFromUser(userID),
		ClientID:  id.NewClientID(),
		RequestID: id.NewRequestID(),
	}
}

func requireSignalsEqual(t *testing.T, got, want cqrshtmx.TrackedEvent) {
	t.Helper()

	want.At = got.At // JSON round trips restore the instant, not the struct

	if got != want {
		t.Fatalf("signal round trip changed the record:\n got  %+v\n want %+v", got, want)
	}
}

func TestNewSignalsJournal_RejectsANilBackend(t *testing.T) {
	t.Parallel()

	journal, err := setup.NewSignalsJournal(nil)
	if err == nil {
		t.Fatal("NewSignalsJournal(nil) must fail at boot, got a journal")
	}

	if journal != nil || errorfamily.Code(err) != "setup.invalid_config" {
		t.Fatalf("NewSignalsJournal(nil) = (%v, %v); want a nil journal and setup.invalid_config", journal, err)
	}
}

func TestSignalsJournal_AJournalWithoutEngineAnswersWithError(t *testing.T) {
	t.Parallel()

	var journal *setup.SignalsJournal

	userID := id.NewUserID()

	_, userErr := journal.ForUser(t.Context(), userID)
	_, clientErr := journal.ForClient(t.Context(), id.NewClientID())
	_, allErr := journal.All(t.Context())
	recordErr := journal.Record(t.Context(), nil)

	for name, err := range map[string]error{
		"ForUser": userErr, "ForClient": clientErr, "All": allErr, "Record": recordErr,
	} {
		if errorfamily.Code(err) != "setup.tracking_without_home" {
			t.Errorf(
				"%s on a nil journal: code %q (%v); want setup.tracking_without_home",
				name,
				errorfamily.Code(err),
				err,
			)
		}
	}

	zero := &setup.SignalsJournal{}

	if _, err := zero.All(t.Context()); errorfamily.Code(err) != "setup.tracking_without_home" {
		t.Errorf("All on a zero journal: code %q (%v); want setup.tracking_without_home", errorfamily.Code(err), err)
	}
}

func TestSignalsJournal_RecordAppendsOncePerSubject(t *testing.T) {
	t.Parallel()

	engine := &transactionalRecorder{appendRecorder: appendRecorder{StreamLogBackend: memoryStreamLog(t)}}
	journal, err := setup.NewSignalsJournal(engine)
	if err != nil {
		t.Fatalf("NewSignalsJournal: %v", err)
	}

	ctx := t.Context()
	alice, bob := id.NewUserID(), id.NewUserID()
	device := id.NewClientID()

	batch := []cqrshtmx.TrackedEvent{
		{Kind: cqrshtmx.TrackedPageview, Path: "/a", UserID: alice},
		{Kind: cqrshtmx.TrackedClick, Path: "/a", Name: "buy", ClientID: device},
		{Kind: cqrshtmx.TrackedImpression, Path: "/a", Name: "hero"},
		{Kind: cqrshtmx.TrackedEngagement, Path: "/a", VisibleMS: 900, UserID: alice},
		{Kind: cqrshtmx.TrackedPageview, Path: "/b", UserID: bob},
	}

	if err := journal.Record(ctx, batch); err != nil {
		t.Fatalf("Record: %v", err)
	}

	want := []string{
		"user:" + alice.String(),
		"client:" + device.String(),
		anonymousStream,
		"user:" + bob.String(),
	}
	if got := strings.Join(engine.appended(), ","); got != strings.Join(want, ",") {
		t.Fatalf("stream appends = %v; want one per subject in first-appearance order %v", engine.appended(), want)
	}

	if engine.transactions != 1 {
		t.Errorf("transactions used = %d; want the whole batch inside one", engine.transactions)
	}

	aliceSignals, err := journal.ForUser(ctx, alice)
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}

	if len(aliceSignals) != 2 || aliceSignals[0].Kind != cqrshtmx.TrackedPageview || aliceSignals[1].VisibleMS != 900 {
		t.Errorf("ForUser(alice) = %+v; want the page view then the engagement", aliceSignals)
	}

	deviceSignals, err := journal.ForClient(ctx, device)
	if err != nil || len(deviceSignals) != 1 || deviceSignals[0].Name != "buy" {
		t.Errorf("ForClient(device) = %+v, %v; want the click only", deviceSignals, err)
	}

	everything, err := journal.All(ctx)
	if err != nil || len(everything) != len(batch) {
		t.Fatalf("All = %d signals, %v; want the whole batch in journal order", len(everything), err)
	}
}

func TestSignalsJournal_AnEmptyBatchWritesNothing(t *testing.T) {
	t.Parallel()

	engine := &transactionalRecorder{appendRecorder: appendRecorder{StreamLogBackend: memoryStreamLog(t)}}
	journal, err := setup.NewSignalsJournal(engine)
	if err != nil {
		t.Fatalf("NewSignalsJournal: %v", err)
	}

	if err := journal.Record(t.Context(), nil); err != nil {
		t.Fatalf("Record(nil): %v", err)
	}

	if appended := engine.appended(); len(appended) != 0 || engine.transactions != 0 {
		t.Errorf("an empty batch reached the engine: appends %v, transactions %d", appended, engine.transactions)
	}
}

func TestSignalsJournal_AFailingAppendIsClassifiedAndKeepsItsCause(t *testing.T) {
	t.Parallel()

	// Two subjects, and the second append fails: without a transaction the
	// engine has already stored the first stream, so the error must say so.
	engine := &appendRecorder{StreamLogBackend: memoryStreamLog(t), failAt: 2}
	journal, err := setup.NewSignalsJournal(engine)
	if err != nil {
		t.Fatalf("NewSignalsJournal: %v", err)
	}

	batch := []cqrshtmx.TrackedEvent{
		{Kind: cqrshtmx.TrackedPageview, Path: "/a", UserID: id.NewUserID()},
		{Kind: cqrshtmx.TrackedPageview, Path: "/a", UserID: id.NewUserID()},
	}

	err = journal.Record(t.Context(), batch)
	if err == nil || !errors.Is(err, errSignalsDiskFull) {
		t.Fatalf("Record with a failing engine = %v; want an error that keeps the engine's cause", err)
	}

	if got := errorfamily.Code(
		err,
	); got != "setup.signals.append" ||
		errorfamily.Classify(err) != errorfamily.Infrastructure {
		t.Errorf("Record error is %s/%s; want Infrastructure/setup.signals.append", errorfamily.Classify(err), got)
	}
}

func TestSignalsJournal_RoundTripsThroughAnEngineThatDecodesEnvelopes(t *testing.T) {
	t.Parallel()

	journal, err := setup.NewSignalsJournal(decodingBackend{StreamLogBackend: memoryStreamLog(t)})
	if err != nil {
		t.Fatalf("NewSignalsJournal: %v", err)
	}

	want := fullSignal()

	ctx := t.Context()

	if err := journal.Record(ctx, []cqrshtmx.TrackedEvent{want}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	byUser, err := journal.ForUser(ctx, want.UserID)
	if err != nil || len(byUser) != 1 {
		t.Fatalf("ForUser = %d signals, %v; want one", len(byUser), err)
	}

	requireSignalsEqual(t, byUser[0], want)

	everything, err := journal.All(ctx)
	if err != nil || len(everything) != 1 {
		t.Fatalf("All = %d signals, %v; want one", len(everything), err)
	}

	requireSignalsEqual(t, everything[0], want)
}

func TestSignalsJournal_StoredSignalsCarryTheStampASweepAgesThemBy(t *testing.T) {
	t.Parallel()

	engine := &appendRecorder{StreamLogBackend: memoryStreamLog(t)}
	journal, err := setup.NewSignalsJournal(engine)
	if err != nil {
		t.Fatalf("NewSignalsJournal: %v", err)
	}

	signal := fullSignal()

	ctx := t.Context()

	if err := journal.Record(ctx, []cqrshtmx.TrackedEvent{signal}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	stored := engine.storedValues()
	if len(stored) != 1 {
		t.Fatalf("the engine received %d envelopes; want one", len(stored))
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(stored[0]), &envelope); err != nil {
		t.Fatalf("the stored envelope is not JSON: %v\n%s", err, stored[0])
	}

	// A journal sweeper keeps every row without a typed time, and an engine
	// takes that time from the first dated stamp of the envelope.
	stamp, isText := envelope["received_at"].(string)

	receivedAt, err := time.Parse(time.RFC3339Nano, stamp)
	if !isText || err != nil {
		t.Fatalf("received_at = %q; want an RFC 3339 stamp a sweeper can read (%v)", stamp, err)
	}

	if !receivedAt.Equal(signal.At) {
		t.Errorf("received_at = %s; want the server's clock at the beacon, %s", receivedAt, signal.At)
	}

	// A signal is not an event: an envelope with event identity would meet
	// the engine's unique event indexes.
	for _, key := range []string{"id", "stream_type", "version"} {
		if _, present := envelope[key]; present {
			t.Errorf("the stored signal carries %q; only event envelopes may", key)
		}
	}

	// The stamp is storage detail: reading the signal back gives the signal.
	byUser, err := journal.ForUser(ctx, signal.UserID)
	if err != nil || len(byUser) != 1 {
		t.Fatalf("ForUser = %d signals, %v; want one", len(byUser), err)
	}

	requireSignalsEqual(t, byUser[0], signal)
}

func TestNew_TrackingIsOffWithoutADurableHome(t *testing.T) {
	t.Parallel()

	var lines bytes.Buffer

	// DisableLogin removes the site-root catch-all, so an unmounted path is
	// an honest 404 instead of the login page's 200.
	bundle := trackingBundle(t, setup.Config{Title: "No Signals Home", DisableLogin: true}, &lines)

	if bundle.TrackingJournal() != nil {
		t.Error("TrackingJournal must be nil when nothing durable is configured")
	}

	assertMountedExactly(t, bundle, nil)

	if rows := trackingRows(bundle); len(rows) != 0 {
		t.Errorf("tracking rows without a home: %v", rows)
	}

	if logged := lines.String(); !strings.Contains(logged, "client tracking is off") ||
		!strings.Contains(logged, "TrackingBackend") {
		t.Errorf("the bundle must say once why tracking is off and how to turn it on; logged:\n%s", logged)
	}
}

func TestNew_TrackingPathDashSilencesTheOffLine(t *testing.T) {
	t.Parallel()

	var lines bytes.Buffer

	bundle := trackingBundle(t, setup.Config{TrackingPath: "-"}, &lines)

	if bundle.TrackingJournal() != nil || strings.Contains(lines.String(), "client tracking is off") {
		t.Errorf(
			"TrackingPath \"-\" opts out silently; journal %v, logged:\n%s",
			bundle.TrackingJournal(),
			lines.String(),
		)
	}
}

func TestNew_TrackingBackendMountsTheEndpointAndTheScript(t *testing.T) {
	t.Parallel()

	bundle := trackingBundle(t, setup.Config{TrackingBackend: memoryStreamLog(t)}, nil)

	device := id.NewClientID()
	request := beacon(pageview)
	request.Header.Set("X-Client-Id", device.String())

	recorder := serveBundle(bundle, request)
	if recorder.Code != http.StatusAccepted ||
		strings.TrimSpace(recorder.Body.String()) != `{"accepted":1,"dropped":0}` {
		t.Fatalf("POST %s = %d %q; want 202 with the receipt", trackEndpoint, recorder.Code, recorder.Body.String())
	}

	journal := bundle.TrackingJournal()
	if journal == nil {
		t.Fatal("TrackingJournal must be built over Config.TrackingBackend")
	}

	signals, err := journal.ForClient(t.Context(), device)
	if err != nil || len(signals) != 1 {
		t.Fatalf("ForClient = %d signals, %v; want the page view", len(signals), err)
	}

	signal := signals[0]
	if signal.Kind != cqrshtmx.TrackedPageview || signal.Path != "/items" || signal.ClientID != device ||
		!signal.UserID.IsZero() || !signal.ActorID.IsZero() || signal.RequestID.IsZero() || signal.At.IsZero() {
		t.Errorf("recorded signal = %+v; want the path, the device key and a server request ID", signal)
	}

	script := serveBundle(bundle, httptest.NewRequest(http.MethodGet, trackScript, nil))
	if script.Code != http.StatusOK || !strings.Contains(script.Body.String(), "cqrsTracker") {
		t.Errorf("GET %s = %d; want 200 with the tracker", trackScript, script.Code)
	}

	assertTrackingPostures(t, bundle.RoutePostures())
}

func assertTrackingPostures(t *testing.T, rows []cqrshtmx.RoutePosture) {
	t.Helper()

	for pattern, want := range map[string]cqrshtmx.RoutePosture{
		trackEndpoint: {Method: http.MethodPost, Posture: cqrshtmx.PosturePublic},
		trackScript:   {Method: http.MethodGet, Posture: cqrshtmx.PosturePublic},
	} {
		row, ok := postureAt(rows, pattern)
		if !ok || row.Method != want.Method || row.Posture != want.Posture || row.Why == "" {
			t.Errorf("posture for %s = %+v (found %t); want %+v with a reason", pattern, row, ok, want)
		}
	}
}

func TestNew_TrackingUsesTheSystemEventStoresEngine(t *testing.T) {
	t.Parallel()

	engine := memoryStreamLog(t)

	bundle := trackingBundle(t, setup.Config{
		EventStore: system.NewEventAdapter(engine, "events"),
	}, nil)

	if bundle.TrackingJournal() == nil {
		t.Fatal("the system event adapter's engine must be the default home of signals")
	}

	device := id.NewClientID()
	request := beacon(pageview)
	request.Header.Set("X-Client-Id", device.String())

	if status := serveBundle(bundle, request).Code; status != http.StatusAccepted {
		t.Fatalf("POST %s = %d; want 202", trackEndpoint, status)
	}

	stored, err := engine.StreamRead(t.Context(), setup.SignalsCollection, "client:"+device.String())
	if err != nil || len(stored) != 1 {
		t.Fatalf("engine stream = %d values, %v; want the signal appended to the shared engine", len(stored), err)
	}

	if envelope, ok := stored[0].(string); !ok || !strings.Contains(envelope, `"path":"/items"`) {
		t.Errorf("stored value = %#v; want a JSON envelope naming the path", stored[0])
	}
}

func TestNew_TrackingAttributesASignedInVisitor(t *testing.T) {
	t.Parallel()

	bundle := trackingBundle(t, setup.Config{TrackingBackend: memoryStreamLog(t)}, nil)

	ctx := t.Context()
	registered, err := bundle.Service.Register(ctx, usermgmt.RegisterRequest{
		ID: usermgmt.SyntheticUserID("tracking-visitor"), Email: "tracking-visitor@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	device := id.NewClientID()
	request := beacon(pageview)
	request.Header.Set("X-Client-Id", device.String())
	request.AddCookie(&http.Cookie{Name: "session", Value: registered.Session.Token})

	if status := serveBundle(bundle, request).Code; status != http.StatusAccepted {
		t.Fatalf("POST %s = %d; want 202", trackEndpoint, status)
	}

	byUser, err := bundle.TrackingJournal().ForUser(ctx, registered.User.ID)
	if err != nil || len(byUser) != 1 {
		t.Fatalf("ForUser = %d signals, %v; want the visitor's page view", len(byUser), err)
	}

	signal := byUser[0]
	if signal.UserID != registered.User.ID || signal.ClientID != device ||
		signal.ActorID != cqrshtmx.ActorIDFromUser(registered.User.ID) {
		t.Errorf(
			"signal identity = user %s actor %s device %s; want the session's user, the derived actor, the header's device",
			signal.UserID,
			signal.ActorID,
			signal.ClientID,
		)
	}

	// A signed-in visitor's signal lives in the user's stream, so the device
	// view no longer carries it: one signal, one subject.
	if byDevice, err := bundle.TrackingJournal().ForClient(ctx, device); err != nil || len(byDevice) != 0 {
		t.Errorf("ForClient = %d signals, %v; want none once the user is known", len(byDevice), err)
	}
}

func TestNew_TrackingSinkReceivesSignalsAndBuildsNoJournal(t *testing.T) {
	t.Parallel()

	sink := &trackingSinkSpy{}
	bundle := trackingBundle(t, setup.Config{TrackingSink: sink}, nil)

	request := beacon(pageview)
	request.Header.Set("X-Client-Id", id.NewClientID().String())

	if status := serveBundle(bundle, request).Code; status != http.StatusAccepted {
		t.Fatalf("POST %s = %d; want 202", trackEndpoint, status)
	}

	if bundle.TrackingJournal() != nil {
		t.Error("TrackingJournal must be nil when the consumer's sink stores the signals")
	}

	batches := sink.recorded()
	if len(batches) != 1 || len(batches[0]) != 1 || batches[0][0].Path != "/items" {
		t.Errorf("sink batches = %+v; want one batch with the page view", batches)
	}

	assertTrackingPostures(t, bundle.RoutePostures())
}

func TestNew_TrackingKeepsTheEndpointsOwnRefusals(t *testing.T) {
	t.Parallel()

	bundle := trackingBundle(t, setup.Config{TrackingBackend: memoryStreamLog(t)}, nil)

	request := beacon(pageview)
	request.Header.Set("Sec-Fetch-Site", "cross-site")

	if status := serveBundle(bundle, request).Code; status != http.StatusForbidden {
		t.Fatalf("a cross-site beacon = %d; want 403 from the handler's own guard", status)
	}

	if signals, err := bundle.TrackingJournal().All(t.Context()); err != nil || len(signals) != 0 {
		t.Errorf("a refused beacon stored %d signals (%v); want none", len(signals), err)
	}
}

func TestNew_TrackingPathOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      setup.Config
		wantJournal bool
		wantRows    []string
	}{
		{
			name:        "TrackingPath - mounts nothing but still builds the journal",
			config:      setup.Config{TrackingPath: "-", TrackerScriptPath: trackScript},
			wantJournal: true,
		},
		{
			name:        "TrackerScriptPath - serves the endpoint without the script",
			config:      setup.Config{TrackerScriptPath: "-"},
			wantJournal: true,
			wantRows:    []string{"POST " + trackEndpoint},
		},
		{
			name: "custom paths are honored",
			config: setup.Config{
				TrackingPath: "/beacons", TrackerScriptPath: "/beacons.js",
			},
			wantJournal: true,
			wantRows:    []string{"POST /beacons", "GET /beacons.js"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := tc.config
			config.TrackingBackend = memoryStreamLog(t)
			config.DisableLogin = true

			bundle := trackingBundle(t, config, nil)

			if (bundle.TrackingJournal() != nil) != tc.wantJournal {
				t.Errorf("journal built = %t; want %t", bundle.TrackingJournal() != nil, tc.wantJournal)
			}

			if rows := trackingRows(bundle); strings.Join(rows, ",") != strings.Join(tc.wantRows, ",") {
				t.Errorf("tracking manifest rows = %v; want %v", rows, tc.wantRows)
			}

			assertMountedExactly(t, bundle, tc.wantRows)
		})
	}
}

// trackingRows lists the bundle's manifest rows for the tracking routes of
// the path table below, the way the manifest orders them.
func trackingRows(bundle *setup.Bundle) []string {
	var rows []string

	for _, row := range bundle.RoutePostures() {
		if slices.Contains(trackingProbePaths(), row.Pattern) {
			rows = append(rows, row.Method+" "+row.Pattern)
		}
	}

	return rows
}

// assertMountedExactly probes every tracking route of the table and expects
// a mounted route (any status but 404) exactly for the listed rows and 404
// for the rest. The bundle must not carry the site-root catch-all
// (DisableLogin), which answers every GET with the login page.
func assertMountedExactly(t *testing.T, bundle *setup.Bundle, mountedRows []string) {
	t.Helper()

	for _, path := range trackingProbePaths() {
		method := http.MethodPost
		if strings.HasSuffix(path, ".js") {
			method = http.MethodGet
		}

		mounted := slices.Contains(mountedRows, method+" "+path)

		request := httptest.NewRequest(method, path, strings.NewReader(`{"events":[]}`))
		request.Header.Set("Content-Type", "application/json")

		status := serveBundle(bundle, request).Code

		if mounted == (status == http.StatusNotFound) {
			t.Errorf("%s %s = %d; mounted %t", method, path, status, mounted)
		}
	}
}

func TestNew_TrackingRejectsConfigsItCannotHonor(t *testing.T) {
	t.Parallel()

	// The shell reports its unbuildable panels first (validateDisableService
	// runs before validateTracking), so the shell cases disable them to
	// reach the tracking verdict.
	shell := setup.Config{DisableService: true, DisableAdmin: true, DisableDashboard: true, DisableLogin: true}

	tests := []struct {
		name   string
		config setup.Config
		want   string
	}{
		{
			name:   "a sink and a backend",
			config: setup.Config{TrackingSink: &trackingSinkSpy{}, TrackingBackend: memoryStreamLog(t)},
			want:   "two homes",
		},
		{
			name: "a backend behind the shell",
			config: func() setup.Config {
				c := shell
				c.TrackingBackend = memoryStreamLog(t)

				return c
			}(),
			want: "DisableService mounts no tracking endpoint",
		},
		{
			name: "a sink behind the shell",
			config: func() setup.Config {
				c := shell
				c.TrackingSink = &trackingSinkSpy{}

				return c
			}(),
			want: "DisableService mounts no tracking endpoint",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := setup.New(tc.config)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("New = %v; want a rejection naming %q", err, tc.want)
			}
		})
	}
}

func TestNewShell_MountsNoTracking(t *testing.T) {
	t.Parallel()

	// A system event store would be the default home on the service path.
	// The shell builds nothing from it and ignores the paths; its storage
	// fields are rejected earlier (TestNew_TrackingRejectsConfigsItCannotHonor).
	bundle, err := setup.NewShell(setup.Config{
		DisableService:    true,
		DisableAdmin:      true,
		DisableDashboard:  true,
		DisableLogin:      true,
		EventStore:        system.NewEventAdapter(memoryStreamLog(t), "events"),
		TrackingPath:      "/beacons",
		TrackerScriptPath: "/beacons.js",
	})
	if err != nil {
		t.Fatalf("NewShell: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	assertMountedExactly(t, bundle, nil)

	if bundle.TrackingJournal() != nil || len(trackingRows(bundle)) != 0 {
		t.Errorf("the shell mounts no tracking: journal %v, rows %v", bundle.TrackingJournal(), trackingRows(bundle))
	}
}
