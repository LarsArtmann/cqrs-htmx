package cqrshtmx

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// syncProtocolEnv wires a memory journal + an App whose single command type
// appends an event on success and rejects names containing "conflict".
type syncProtocolEnv struct {
	app     *App
	journal *memory.MemoryStore
	mux     *http.ServeMux
}

func newSyncProtocolEnv(t *testing.T) *syncProtocolEnv {
	t.Helper()

	store := memory.NewMemoryStore()
	cmdDisp := command.NewDispatcher()

	err := cmdDisp.Register("AppendNote", func(ctx context.Context, cmd command.Command) error {
		note, ok := cmd.(*syncTestCmd)
		if !ok {
			return errorfamily.NewCorruption("sync_protocol.bad_type", "unexpected command type")
		}

		if strings.Contains(note.Name, "conflict") {
			return errorfamily.NewConflict("note.conflict", "note already exists: "+note.Name)
		}

		// jsontext payload + json stamp = GENUINELY-JSON bytes. (A map payload
		// would still be CBOR-encoded despite the stamp — the WithEncoding
		// metadata-only trap — and land on the degraded opaque path.)
		evt, err := event.New(
			event.Type("NoteAppended"),
			note.StreamID(),
			"note",
			event.Version(1),
			jsontext.Value(`{"name":`+strconv.Quote(note.Name)+`}`),
			event.WithEncoding(codec.EncodingJSON),
		)
		if err != nil {
			return err
		}

		return store.AppendBatch(ctx, id.StreamRef{ID: note.StreamID(), Type: "note"}, []event.Event{evt})
	})
	if err != nil {
		t.Fatalf("register AppendNote: %v", err)
	}

	app := MustNew(Config{Commands: cmdDisp})

	_ = app.Command("AppendNote", DecodeJSON(func(req struct {
		Name string `json:"name"`
	},
	) (command.Command, error) {
		cmd := newSyncTestCmd("AppendNote")
		cmd.Name = req.Name

		return cmd, nil
	}))

	mux := http.NewServeMux()
	mux.Handle("POST /sync/push", app.SyncPushHandler())
	mux.Handle("GET /sync/pull", SyncPullHandler(store, WithSyncPullBackendID("test-backend")))

	return &syncProtocolEnv{app: app, journal: store, mux: mux}
}

func (e *syncProtocolEnv) serve(method, target, body string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", ContentTypeJSON)

	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)

	return rec
}

// TestSyncProtocol_OfflineQueueBatchPushAndCatchUpPull walks the full
// ADR-0056 loop a real offline client performs:
//
//  1. Two commands queue offline; on reconnect they leave as ONE batch push.
//  2. One command conflicts with authoritative state (rejected, family
//     conflict) — the other confirms (its event lands in the journal).
//  3. The client pulls from cursor zero, pages through, and observes the
//     confirmed command's event with its payload.
//  4. A conditional re-poll at the same cursor is a 304 (cache-aware).
func TestSyncProtocol_OfflineQueueBatchPushAndCatchUpPull(t *testing.T) {
	t.Parallel()

	env := newSyncProtocolEnv(t)

	// --- 1. Batch push (one request, two envelopes) ---
	pushRec := env.serve(http.MethodPost, "/sync/push", `{"commands":[
		{"commandId":"offline-1","type":"AppendNote","body":"{\"name\":\"grocery-list\"}"},
		{"commandId":"offline-2","type":"AppendNote","body":"{\"name\":\"conflict-dup\"}"}
	]}`)
	if pushRec.Code != http.StatusOK {
		t.Fatalf("push status = %d, body: %s", pushRec.Code, pushRec.Body.String())
	}

	var pushResp SyncPushResponse
	if err := json.Unmarshal(pushRec.Body.Bytes(), &pushResp); err != nil {
		t.Fatalf("unmarshal push response: %v", err)
	}

	if len(pushResp.Results) != 2 {
		t.Fatalf("push results = %d, want 2", len(pushResp.Results))
	}

	outcomes := make(map[string]SyncPushResult, len(pushResp.Results))
	for _, res := range pushResp.Results {
		outcomes[res.CommandID] = res
	}

	if res := outcomes["offline-1"]; res.Status != SyncPushStatusConfirmed {
		t.Errorf("offline-1 = %+v, want confirmed", res)
	}

	if res := outcomes["offline-2"]; res.Status != SyncPushStatusRejected ||
		res.Error == nil || res.Error.Family != "conflict" {
		t.Errorf("offline-2 = %+v, want rejected/conflict (client surfaces, drops from queue)", res)
	}

	// --- 2. Bootstrap pull from cursor zero ---
	firstPull := env.serve(http.MethodGet, "/sync/pull?limit=1", "")
	if firstPull.Code != http.StatusOK {
		t.Fatalf("pull status = %d, body: %s", firstPull.Code, firstPull.Body.String())
	}

	var page SyncPullResponse
	if err := json.Unmarshal(firstPull.Body.Bytes(), &page); err != nil {
		t.Fatalf("unmarshal pull page 1: %v", err)
	}

	if page.BackendID != "test-backend" {
		t.Errorf("backendId = %q, want test-backend", page.BackendID)
	}

	// Only offline-1 appends an event (offline-2 conflicts), so page 1 holds
	// the full journal and reports caught-up.
	if len(page.Events) != 1 || page.HasMore {
		t.Fatalf("page 1: events=%d hasMore=%v, want 1/false", len(page.Events), page.HasMore)
	}

	if page.Events[0].Type != "NoteAppended" {
		t.Errorf("first event type = %q, want NoteAppended", page.Events[0].Type)
	}

	if page.Events[0].PayloadEncoding != "json" || string(page.Events[0].Payload) != `{"name":"grocery-list"}` {
		t.Errorf("first event payload: encoding=%q payload=%q, want json grocery-list",
			page.Events[0].PayloadEncoding, string(page.Events[0].Payload))
	}

	// --- 3. Page to caught-up ---
	caughtUpCursor := env.pullToCaughtUp(t, page)

	// --- 4. Conditional re-poll at the caught-up cursor: 304, no body ---
	tag := firstPull.Header().Get("ETag")
	rePoll := env.serve(http.MethodGet, "/sync/pull?limit=1&after="+caughtUpCursor, "")
	rePollSameCursor := env.serve(http.MethodGet, "/sync/pull?limit=1&after="+caughtUpCursor, "",
		"If-None-Match", rePoll.Header().Get("ETag"))

	if tag == "" || rePoll.Header().Get("ETag") == "" {
		t.Fatal("pull responses must carry ETags")
	}

	if rePollSameCursor.Code != http.StatusNotModified {
		t.Errorf("conditional re-poll = %d, want 304 (body: %s)",
			rePollSameCursor.Code, rePollSameCursor.Body.String())
	}
}

// pullToCaughtUp pages from the given page's cursor until the journal reports
// caught-up, then proves it: one more pull at the resting cursor returns zero
// events with an empty nextCursor. It returns the cursor a client persists
// (the last NON-empty one — an empty page never moves the cursor).
func (e *syncProtocolEnv) pullToCaughtUp(t *testing.T, page SyncPullResponse) string {
	t.Helper()

	for page.HasMore {
		next := e.serve(http.MethodGet, "/sync/pull?limit=1&after="+page.NextCursor, "")
		if next.Code != http.StatusOK {
			t.Fatalf("pull page status = %d", next.Code)
		}

		if err := json.Unmarshal(next.Body.Bytes(), &page); err != nil {
			t.Fatalf("unmarshal pull page: %v", err)
		}
	}

	restRec := e.serve(http.MethodGet, "/sync/pull?limit=1&after="+page.NextCursor, "")
	if restRec.Code != http.StatusOK {
		t.Fatalf("caught-up pull status = %d, body: %s", restRec.Code, restRec.Body.String())
	}

	rest := pullSyncBody(t, restRec)
	if len(rest.Events) != 0 || rest.HasMore {
		t.Fatalf("caught-up pull: events=%d hasMore=%v, want 0/false",
			len(rest.Events), rest.HasMore)
	}

	if rest.NextCursor != "" {
		t.Errorf("caught-up nextCursor = %q, want empty (keep previous cursor)", rest.NextCursor)
	}

	return page.NextCursor
}
