package cqrshtmx_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The tracking endpoint takes strings from anyone. These tests pin the three
// promises it makes to the app that mounts it: identity and time come from the
// server and never from the body, a refused request is refused before its
// body is read, and a well-formed envelope always answers 202.

const pageviewBody = `{"events":[{"kind":"pageview","path":"/"}]}`

// mixedBody carries three valid signals and three invalid ones.
const mixedBody = `{"events":[
	{"kind":"pageview","path":"/items/42?token=secret#top","referrer":"https://bob:pw@g.example/search?q=shoes#frag"},
	{"kind":"click","path":"/items/42","name":"add-to-cart","target":"button#buy"},
	{"kind":"not-a-signal","path":"/items/42"},
	{"kind":"pageview","path":"no-leading-slash"},
	{"kind":"pageview","path":"//evil.example/"},
	{"kind":"impression","path":"/items/42","name":"card","target":"article.card","visibleMs":1500.9}
]}`

// recordingSink remembers every batch the endpoint recorded and the state of
// the context each batch was written under.
type recordingSink struct {
	mu       sync.Mutex
	batches  [][]cqrshtmx.TrackedEvent
	ctxErrs  []error
	deadline bool
	err      error
	onRecord func()
}

func (s *recordingSink) Record(ctx context.Context, events []cqrshtmx.TrackedEvent) error {
	if s.onRecord != nil {
		s.onRecord()
	}

	s.mu.Lock()
	s.batches = append(s.batches, events)
	s.ctxErrs = append(s.ctxErrs, ctx.Err())
	_, s.deadline = ctx.Deadline()
	s.mu.Unlock()

	return s.err
}

func (s *recordingSink) recorded() ([][]cqrshtmx.TrackedEvent, []error, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.batches, s.ctxErrs, s.deadline
}

// spyBody counts reads so a test can prove a request's body was never touched.
type spyBody struct {
	io.Reader

	reads int
}

func (b *spyBody) Read(p []byte) (int, error) {
	b.reads++

	return b.Reader.Read(p)
}

// trackRequest is one JSON beacon the way tracker.js sends it.
func trackRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/-/track", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

// attributedRequest is a beacon whose context already carries the identity a
// session middleware would have put there. It returns that identity.
func attributedRequest(body string) (*http.Request, cqrshtmx.TrackedEvent) {
	userID := id.NewUserID()
	actorID := cqrshtmx.ActorIDFromUser(userID)
	clientID := id.NewClientID()
	requestID := id.NewRequestID()

	request := trackRequest(body)
	ctx := cqrshtmx.WithUserID(request.Context(), userID)
	ctx = cqrshtmx.WithActorID(ctx, actorID)
	ctx = cqrshtmx.WithClientID(ctx, clientID)
	ctx = cqrshtmx.WithRequestID(ctx, requestID)

	identity := cqrshtmx.TrackedEvent{UserID: userID, ActorID: actorID, ClientID: clientID, RequestID: requestID}

	return request.WithContext(ctx), identity
}

func serveTrack(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

// recorded serves one request through a fresh handler over a fresh sink and
// returns the signals of the one batch the sink received. It fails the test
// unless the endpoint answered 202 and wrote exactly once.
func recorded(t *testing.T, request *http.Request, opts ...cqrshtmx.TrackOption) []cqrshtmx.TrackedEvent {
	t.Helper()

	sink := &recordingSink{}
	recorder := serveTrack(cqrshtmx.TrackHandler(sink, opts...), request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body %s", recorder.Code, recorder.Body)
	}

	batches, _, _ := sink.recorded()
	if len(batches) != 1 {
		t.Fatalf("Record called %d time(s), want one batch", len(batches))
	}

	return batches[0]
}

// recordedOne is recorded for a beacon that stores exactly one signal.
func recordedOne(t *testing.T, request *http.Request) cqrshtmx.TrackedEvent {
	t.Helper()

	events := recorded(t, request)
	if len(events) != 1 {
		t.Fatalf("recorded %d signals, want one: %+v", len(events), events)
	}

	return events[0]
}

// requireReceipt fails the test unless the body is the given receipt.
func requireReceipt(t *testing.T, recorder *httptest.ResponseRecorder, accepted, dropped int) {
	t.Helper()

	want := `{"accepted":` + strconv.Itoa(accepted) + `,"dropped":` + strconv.Itoa(dropped) + `}`

	if got := strings.TrimSpace(recorder.Body.String()); got != want {
		t.Errorf("receipt = %s, want %s", got, want)
	}
}

func TestTrackHandler_AttributesEverySignalToTheRequestAndTheServerClock(t *testing.T) {
	t.Parallel()

	request, identity := attributedRequest(mixedBody)

	before := time.Now()
	events := recorded(t, request)
	after := time.Now()

	for index, event := range events {
		if event.UserID != identity.UserID || event.ActorID != identity.ActorID ||
			event.ClientID != identity.ClientID || event.RequestID != identity.RequestID {
			t.Errorf("signal %d identity = user %s actor %s client %s request %s, want the request's identity",
				index, event.UserID, event.ActorID, event.ClientID, event.RequestID)
		}

		if event.At.Before(before) || event.At.After(after) {
			t.Errorf("signal %d At = %s, want the server's receive time", index, event.At)
		}

		if events[0].At != event.At {
			t.Errorf("signal %d At = %s, want one time per batch (%s)", index, event.At, events[0].At)
		}
	}
}

func TestTrackHandler_KeepsValidSignalsAndDropsTheRest(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	recorder := serveTrack(cqrshtmx.TrackHandler(sink), trackRequest(mixedBody))

	requireReceipt(t, recorder, 3, 3)

	batches, ctxErrs, hasDeadline := sink.recorded()
	if len(batches) != 1 || len(batches[0]) != 3 {
		t.Fatalf("recorded %d batch(es) of %d signal(s), want one batch of three", len(batches), len(batches[0]))
	}

	if ctxErrs[0] != nil || !hasDeadline {
		t.Errorf("sink context err = %v, deadline set = %t; want an uncancelled context with a deadline",
			ctxErrs[0], hasDeadline)
	}

	pageview, click, impression := batches[0][0], batches[0][1], batches[0][2]

	if pageview.Kind != cqrshtmx.TrackedPageview || pageview.Path != "/items/42" {
		t.Errorf("pageview = %s %q, want pageview /items/42 (query and fragment cut)", pageview.Kind, pageview.Path)
	}

	if want := "https://g.example/search"; pageview.Referrer != want {
		t.Errorf("pageview referrer = %q, want %q (userinfo, query and fragment dropped)", pageview.Referrer, want)
	}

	if click.Kind != cqrshtmx.TrackedClick || click.Name != "add-to-cart" ||
		click.Target != "button#buy" || click.Referrer != "" {
		t.Errorf("click = %+v, want kind click with its name and target and no referrer", click)
	}

	if impression.Kind != cqrshtmx.TrackedImpression || impression.VisibleMS != 1500 {
		t.Errorf("impression = %+v, want visibleMs 1500 (floored)", impression)
	}
}

func TestTrackHandler_IgnoresIdentityAndTimeSentInTheBody(t *testing.T) {
	t.Parallel()

	const body = `{"events":[{"kind":"pageview","path":"/spoof","userId":"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"actorId":"user:01ARZ3NDEKTSV4RRFFQ69G5FAV","clientId":"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"requestId":"01ARZ3NDEKTSV4RRFFQ69G5FAV","at":"1999-01-01T00:00:00Z"}],"at":"1999-01-01T00:00:00Z"}`

	event := recordedOne(t, trackRequest(body))

	if !event.UserID.IsZero() || !event.ActorID.IsZero() ||
		!event.ClientID.IsZero() || !event.RequestID.IsZero() {
		t.Errorf("an anonymous request recorded identity: user %s actor %s client %s request %s",
			event.UserID, event.ActorID, event.ClientID, event.RequestID)
	}

	if event.At.Year() < 2000 {
		t.Errorf("At = %s, want the server's time, not the body's", event.At)
	}
}

func TestTrackHandler_CapsEveryClientString(t *testing.T) {
	t.Parallel()

	long := func(n int) string { return strings.Repeat("é", n/2+1) }

	tests := []struct {
		name   string
		signal string
		field  func(cqrshtmx.TrackedEvent) string
		want   string
	}{
		{
			name:   "path",
			signal: `{"kind":"pageview","path":"/` + long(5000) + `"}`,
			field:  func(e cqrshtmx.TrackedEvent) string { return e.Path },
			want:   "/" + strings.Repeat("é", 1023),
		},
		{
			name:   "referrer (stored percent-encoded, the cut never splits an escape)",
			signal: `{"kind":"pageview","path":"/","referrer":"https://g.example/` + long(5000) + `"}`,
			field:  func(e cqrshtmx.TrackedEvent) string { return e.Referrer },
			want:   "https://g.example/" + strings.Repeat("%C3%A9", 338),
		},
		{
			name:   "name",
			signal: `{"kind":"click","path":"/","name":"` + long(600) + `"}`,
			field:  func(e cqrshtmx.TrackedEvent) string { return e.Name },
			want:   strings.Repeat("é", 100),
		},
		{
			name:   "target",
			signal: `{"kind":"click","path":"/","target":"` + long(1200) + `"}`,
			field:  func(e cqrshtmx.TrackedEvent) string { return e.Target },
			want:   strings.Repeat("é", 256),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			event := recordedOne(t, trackRequest(`{"events":[`+tc.signal+`]}`))

			if got := tc.field(event); got != tc.want {
				t.Errorf("%s = %d bytes, want the %d-byte cap at a rune boundary", tc.name, len(got), len(tc.want))
			}
		})
	}
}

// field renders one optional wire field of a signal, or nothing.
func field(wire string) string {
	if wire == "" {
		return ""
	}

	return "," + wire
}

func TestTrackHandler_ClampsVisibleTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		wire string
		want int64
	}{
		{name: "negative", wire: `"visibleMs":-5`, want: 0},
		{name: "zero", wire: `"visibleMs":0`, want: 0},
		{name: "absent", wire: ``, want: 0},
		{name: "one day", wire: `"visibleMs":86400000`, want: 86400000},
		{name: "beyond one day", wire: `"visibleMs":90000000`, want: 86400000},
		{name: "far beyond int64", wire: `"visibleMs":1e300`, want: 86400000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := `{"events":[{"kind":"engagement","path":"/"` + field(tc.wire) + `}]}`

			if got := recordedOne(t, trackRequest(body)).VisibleMS; got != tc.want {
				t.Errorf("visibleMs = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTrackHandler_ReducesOrDropsReferrers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		referrer string
		want     string
	}{
		{referrer: `https://g.example/a/b?x=1&y=2#z`, want: "https://g.example/a/b"},
		{referrer: `http://localhost:8080/page`, want: "http://localhost:8080/page"},
		{referrer: `https://user:secret@g.example/`, want: "https://g.example/"},
		{referrer: `/relative/path`, want: ""},
		{referrer: `not a url %zz`, want: ""},
		{referrer: `javascript:alert(1)`, want: ""},
		{referrer: ``, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.referrer, func(t *testing.T) {
			t.Parallel()

			body, err := json.Marshal(map[string]any{"events": []map[string]string{
				{"kind": "pageview", "path": "/", "referrer": tc.referrer},
			}})
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}

			if got := recordedOne(t, trackRequest(string(body))).Referrer; got != tc.want {
				t.Errorf("referrer %q stored as %q, want %q", tc.referrer, got, tc.want)
			}
		})
	}
}

func TestTrackHandler_ReadsTheDeviceKeyFromTheHeaderWithoutEnrichment(t *testing.T) {
	t.Parallel()

	clientID := id.NewClientID()

	tests := []struct {
		name   string
		header string
		want   id.ClientID
	}{
		{name: "valid ULID", header: clientID.String(), want: clientID},
		{name: "not a ULID", header: "device-7", want: id.ClientID{}},
		{name: "absent", header: "", want: id.ClientID{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := trackRequest(pageviewBody)
			request.Header.Set(cqrshtmx.HeaderClientID, tc.header)

			if got := recordedOne(t, request).ClientID; got != tc.want {
				t.Errorf("client ID = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestTrackHandler_RefusesBeforeReadingTheBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		method    string
		configure func(*http.Request)
		want      int
		sentinel  error
		allow     string
	}{
		{
			name: "GET", method: http.MethodGet, want: http.StatusMethodNotAllowed,
			sentinel: httperr.ErrMethodNotAllowed, allow: http.MethodPost,
		},
		{
			name: "HEAD", method: http.MethodHead, want: http.StatusMethodNotAllowed,
			sentinel: httperr.ErrMethodNotAllowed, allow: http.MethodPost,
		},
		{
			name: "PUT", method: http.MethodPut, want: http.StatusMethodNotAllowed,
			sentinel: httperr.ErrMethodNotAllowed, allow: http.MethodPost,
		},
		{
			name: "cross-site fetch", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") },
			want:      http.StatusForbidden, sentinel: httperr.ErrForbidden,
		},
		{
			name: "same-site fetch from a sibling subdomain", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-site") },
			want:      http.StatusForbidden, sentinel: httperr.ErrForbidden,
		},
		{
			name: "foreign origin", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") },
			want:      http.StatusForbidden, sentinel: httperr.ErrForbidden,
		},
		{
			name: "same host, different port", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Set("Origin", "https://example.com:8443") },
			want:      http.StatusForbidden, sentinel: httperr.ErrForbidden,
		},
		{
			name: "text/plain", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") },
			want:      http.StatusUnsupportedMediaType, sentinel: httperr.ErrUnsupportedMediaType,
		},
		{
			name: "form-encoded", method: http.MethodPost,
			configure: func(r *http.Request) {
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			},
			want: http.StatusUnsupportedMediaType, sentinel: httperr.ErrUnsupportedMediaType,
		},
		{
			name: "no content type", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Del("Content-Type") },
			want:      http.StatusUnsupportedMediaType, sentinel: httperr.ErrUnsupportedMediaType,
		},
		{
			name: "malformed content type", method: http.MethodPost,
			configure: func(r *http.Request) { r.Header.Set("Content-Type", "application/json;;;") },
			want:      http.StatusUnsupportedMediaType, sentinel: httperr.ErrUnsupportedMediaType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spy := &spyBody{Reader: strings.NewReader(pageviewBody)}
			request := httptest.NewRequest(tc.method, "/-/track", spy)
			request.Header.Set("Content-Type", "application/json; charset=utf-8")

			if tc.configure != nil {
				tc.configure(request)
			}

			sink := &recordingSink{}

			var handled error

			capture := func(w http.ResponseWriter, _ *http.Request, err error) {
				handled = err

				w.WriteHeader(http.StatusTeapot)
			}

			recorder := serveTrack(cqrshtmx.TrackHandler(sink, cqrshtmx.WithTrackErrorHandler(capture)), request)

			if recorder.Code != http.StatusTeapot {
				t.Errorf("status = %d, want the custom handler's 418; body %s", recorder.Code, recorder.Body)
			}

			if !errors.Is(handled, tc.sentinel) || cqrshtmx.MapError(handled) != tc.want {
				t.Errorf("error = %v (maps to %d), want %v mapping to %d",
					handled, cqrshtmx.MapError(handled), tc.sentinel, tc.want)
			}

			if got := recorder.Header().Get("Allow"); got != tc.allow {
				t.Errorf("Allow = %q, want %q", got, tc.allow)
			}

			if spy.reads != 0 {
				t.Errorf("body was read %d time(s); a refused request must not cost a read", spy.reads)
			}

			if batches, _, _ := sink.recorded(); len(batches) != 0 {
				t.Errorf("sink recorded %d batch(es) for a refused request", len(batches))
			}
		})
	}
}

func TestTrackHandler_AcceptsFirstPartyEvidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*http.Request)
	}{
		{name: "no evidence (a non-browser client)", configure: nil},
		{
			name:      "same-origin fetch",
			configure: func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "same-origin") },
		},
		{
			name:      "user-initiated navigation",
			configure: func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "none") },
		},
		{name: "same origin", configure: func(r *http.Request) { r.Header.Set("Origin", "https://example.com") }},
		{
			name: "same host behind TLS termination",
			configure: func(r *http.Request) {
				r.Header.Set("Origin", "https://example.com")
				r.Header.Set("Sec-Fetch-Site", "same-origin")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := trackRequest(pageviewBody)

			if tc.configure != nil {
				tc.configure(request)
			}

			if got := recordedOne(t, request); got.Kind != cqrshtmx.TrackedPageview {
				t.Errorf("recorded %+v, want the page view", got)
			}
		})
	}
}

func TestTrackHandler_RefusesUndecodableAndOversizedBodies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		options  []cqrshtmx.TrackOption
		want     int
		sentinel error
	}{
		{
			name: "not JSON", body: `events=1`,
			want: http.StatusBadRequest, sentinel: httperr.ErrDecodeFailed,
		},
		{
			name: "a JSON array", body: `[]`,
			want: http.StatusBadRequest, sentinel: httperr.ErrDecodeFailed,
		},
		{
			name: "events is not an array", body: `{"events":"pageview"}`,
			want: http.StatusBadRequest, sentinel: httperr.ErrDecodeFailed,
		},
		{
			name: "a signal with the wrong field type", body: `{"events":[{"kind":"pageview","path":7}]}`,
			want: http.StatusBadRequest, sentinel: httperr.ErrDecodeFailed,
		},
		{
			name: "truncated", body: `{"events":[`,
			want: http.StatusBadRequest, sentinel: httperr.ErrDecodeFailed,
		},
		{
			name: "body over the limit",
			body: `{"events":[` + strings.Repeat(`{"kind":"pageview","path":"/"},`, 20) +
				`{"kind":"pageview","path":"/"}]}`,
			options: []cqrshtmx.TrackOption{cqrshtmx.WithTrackBodyLimit(64)},
			want:    http.StatusRequestEntityTooLarge, sentinel: httperr.ErrRequestTooLarge,
		},
		{
			name: "batch over the limit",
			body: `{"events":[{"kind":"pageview","path":"/"},` +
				`{"kind":"click","path":"/"},{"kind":"click","path":"/"}]}`,
			options: []cqrshtmx.TrackOption{cqrshtmx.WithTrackBatchLimit(2)},
			want:    http.StatusRequestEntityTooLarge, sentinel: httperr.ErrRequestTooLarge,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sink := &recordingSink{}
			recorder := serveTrack(cqrshtmx.TrackHandler(sink, tc.options...), trackRequest(tc.body))

			if recorder.Code != tc.want {
				t.Errorf("status = %d, want %d; body %s", recorder.Code, tc.want, recorder.Body)
			}

			if !strings.Contains(recorder.Body.String(), `"status":`+strconv.Itoa(tc.want)) {
				t.Errorf("body = %s, want a JSON error carrying status %d", recorder.Body, tc.want)
			}

			if batches, _, _ := sink.recorded(); len(batches) != 0 {
				t.Errorf("sink recorded %d batch(es); nothing of a refused envelope may be stored", len(batches))
			}
		})
	}
}

func TestTrackHandler_KeepsTheBatchJustUnderTheLimit(t *testing.T) {
	t.Parallel()

	body := `{"events":[{"kind":"pageview","path":"/"},{"kind":"click","path":"/"}]}`

	sink := &recordingSink{}
	recorder := serveTrack(cqrshtmx.TrackHandler(sink, cqrshtmx.WithTrackBatchLimit(2)), trackRequest(body))

	requireReceipt(t, recorder, 2, 0)
}

func TestTrackHandler_Answers202ForAnEmptyAndForAWhollyInvalidBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		receipt string
	}{
		{name: "empty batch", body: `{"events":[]}`, receipt: `{"accepted":0,"dropped":0}`},
		{name: "absent events", body: `{}`, receipt: `{"accepted":0,"dropped":0}`},
		{name: "null events", body: `{"events":null}`, receipt: `{"accepted":0,"dropped":0}`},
		{name: "null signal", body: `{"events":[null]}`, receipt: `{"accepted":0,"dropped":1}`},
		{
			name:    "all invalid",
			body:    `{"events":[{"kind":"x","path":"/"},{"kind":"pageview","path":"relative"}]}`,
			receipt: `{"accepted":0,"dropped":2}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sink := &recordingSink{}
			recorder := serveTrack(cqrshtmx.TrackHandler(sink), trackRequest(tc.body))

			if recorder.Code != http.StatusAccepted {
				t.Errorf("status = %d, want 202; body %s", recorder.Code, recorder.Body)
			}

			if got := strings.TrimSpace(recorder.Body.String()); got != tc.receipt {
				t.Errorf("receipt = %s, want %s", got, tc.receipt)
			}

			if batches, _, _ := sink.recorded(); len(batches) != 0 {
				t.Errorf("sink recorded %d batch(es); an empty batch is not written", len(batches))
			}
		})
	}
}

func TestTrackHandler_ASinkFailureIsLoggedNeverReturned(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{err: errors.New("journal is read-only")}
	recorder := serveTrack(cqrshtmx.TrackHandler(sink), trackRequest(pageviewBody))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 even when the sink fails; body %s", recorder.Code, recorder.Body)
	}

	requireReceipt(t, recorder, 1, 0)
}

func TestTrackHandler_WithoutASinkAcceptsAndDiscards(t *testing.T) {
	t.Parallel()

	recorder := serveTrack(cqrshtmx.TrackHandler(nil), trackRequest(pageviewBody))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body %s", recorder.Code, recorder.Body)
	}

	requireReceipt(t, recorder, 1, 0)
}

func TestTrackHandler_SinkWriteOutlivesTheRequestsCancellation(t *testing.T) {
	t.Parallel()

	request := trackRequest(pageviewBody)
	ctx, cancel := context.WithCancel(request.Context())
	t.Cleanup(cancel)

	// The browser goes away while the batch is being written.
	sink := &recordingSink{onRecord: cancel}
	recorder := serveTrack(cqrshtmx.TrackHandler(sink), request.WithContext(ctx))

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body %s", recorder.Code, recorder.Body)
	}

	_, ctxErrs, hasDeadline := sink.recorded()
	if len(ctxErrs) != 1 || ctxErrs[0] != nil || !hasDeadline {
		t.Errorf("sink context err = %v (deadline %t); want uncancelled with a deadline", ctxErrs, hasDeadline)
	}
}

func TestTrackedEvent_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.November, 26, 23, 0, 0, 0, time.UTC)
	userID := id.NewUserID()

	event := cqrshtmx.TrackedEvent{
		Kind: cqrshtmx.TrackedEngagement, Path: "/items", Referrer: "https://g.example/", Name: "card",
		Target: "article", VisibleMS: 4200, At: at, UserID: userID, ActorID: cqrshtmx.ActorIDFromUser(userID),
		ClientID: id.NewClientID(), RequestID: id.NewRequestID(),
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded cqrshtmx.TrackedEvent

	err = json.Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("unmarshal %s: %v", encoded, err)
	}

	if decoded != event {
		t.Errorf("round trip changed the event:\n got %+v\nwant %+v", decoded, event)
	}

	var wire map[string]jsontext.Value

	err = json.Unmarshal(encoded, &wire)
	if err != nil {
		t.Fatalf("unmarshal into a map: %v", err)
	}

	for key, want := range map[string]string{
		"kind": `"engagement"`, "visibleMs": `4200`, "at": `"2026-11-26T23:00:00Z"`,
	} {
		if string(wire[key]) != want {
			t.Errorf("%s = %s, want %s", key, wire[key], want)
		}
	}

	anonymous, err := json.Marshal(cqrshtmx.TrackedEvent{Kind: cqrshtmx.TrackedPageview, Path: "/", At: at})
	if err != nil {
		t.Fatalf("marshal an anonymous event: %v", err)
	}

	const want = `{"kind":"pageview","path":"/","at":"2026-11-26T23:00:00Z"}`

	if got := string(anonymous); got != want {
		t.Errorf("anonymous event = %s, want %s (zero identity, zero dwell and empty strings omitted)", got, want)
	}
}

func TestTrackerScriptHandler(t *testing.T) {
	t.Parallel()

	handler := cqrshtmx.TrackerScriptHandler()

	first := serveTrack(handler, httptest.NewRequest(http.MethodGet, "/-/tracker.js", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", first.Code)
	}

	if got := first.Header().Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}

	if got := first.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want the one-year immutable posture", got)
	}

	etag := first.Header().Get("ETag")
	if !regexp.MustCompile(`^"cqrs-tracker-[0-9a-f]+"$`).MatchString(etag) {
		t.Errorf("ETag = %q, want a content-derived strong tag", etag)
	}

	script := first.Body.String()
	if !strings.Contains(script, "window.cqrsTracker") || strings.Contains(script, "/* place") {
		t.Errorf("the served script is not the tracker (%d bytes)", len(script))
	}

	revalidate := httptest.NewRequest(http.MethodGet, "/-/tracker.js", nil)
	revalidate.Header.Set("If-None-Match", etag)

	recorder := serveTrack(handler, revalidate)
	if recorder.Code != http.StatusNotModified || recorder.Body.Len() != 0 {
		t.Errorf("conditional GET = %d with %d bytes, want a bodiless 304", recorder.Code, recorder.Body.Len())
	}

	// net/http discards a HEAD response's body on the wire; the handler level
	// only owes the same status and validator as GET.
	head := serveTrack(handler, httptest.NewRequest(http.MethodHead, "/-/tracker.js", nil))
	if head.Code != http.StatusOK || head.Header().Get("ETag") != etag {
		t.Errorf("HEAD = %d with ETag %q, want 200 and the GET validator %q",
			head.Code, head.Header().Get("ETag"), etag)
	}

	post := serveTrack(handler, httptest.NewRequest(http.MethodPost, "/-/tracker.js", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST = %d, want 405", post.Code)
	}

	if got := cqrshtmx.AssetETag("cqrs-tracker", []byte(script)); got != etag {
		t.Errorf("ETag %s is not AssetETag of the served bytes (%s)", etag, got)
	}
}

func TestTrackingVersion_MatchesTheEmbeddedScript(t *testing.T) {
	t.Parallel()

	recorder := serveTrack(cqrshtmx.TrackerScriptHandler(), httptest.NewRequest(http.MethodGet, "/-/tracker.js", nil))

	match := regexp.MustCompile(`const VERSION = "([0-9]+\.[0-9]+\.[0-9]+)"`).FindStringSubmatch(recorder.Body.String())
	if match == nil {
		t.Fatalf("tracker.js declares no `const VERSION = \"x.y.z\"`")
	}

	if match[1] != cqrshtmx.TrackingVersion() {
		t.Errorf("tracker.js VERSION = %s, TrackingVersion() = %s; the two must move together",
			match[1], cqrshtmx.TrackingVersion())
	}
}

func TestTrackerScriptTag(t *testing.T) {
	t.Parallel()

	if got, want := cqrshtmx.TrackerScriptTag(
		"/-/tracker.js",
	), `<script src="/-/tracker.js" defer></script>`; got != want {
		t.Errorf("TrackerScriptTag = %s, want %s", got, want)
	}
}
