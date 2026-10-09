package cqrshtmx

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

		evt, err := event.New(
			event.Type("NoteAppended"),
			note.StreamID(),
			"note",
			event.Version(1),
			map[string]any{"name": note.Name},
			event.WithEncoding("json"),
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
	}) (command.Command, error) {
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

	if page.Events[0].PayloadEncoding != "json" || len(page.Events[0].Payload) == 0 {
		t.Errorf("first event payload: encoding=%q payload=%q, want json with bytes",
			page.Events[0].PayloadEncoding, string(page.Events[0].Payload))
	}

	// --- 3. Page to caught-up ---
	for page.HasMore {
		next := env.serve(http.MethodGet, "/sync/pull?limit=1&after="+page.NextCursor, "")
		if next.Code != http.StatusOK {
			t.Fatalf("pull page status = %d", next.Code)
		}

		if err := json.Unmarshal(next.Body.Bytes(), &page); err != nil {
			t.Fatalf("unmarshal pull page: %v", err)
		}
	}

	if len(page.Events) != 0 {
		t.Fatalf("final page events = %d, want 0 (caught up)", len(page.Events))
	}

	// --- 4. Conditional re-poll at the caught-up cursor: 304, no body ---
	tag := firstPull.Header().Get("ETag")
	rePoll := env.serve(http.MethodGet, "/sync/pull?limit=1&after="+page.NextCursor, "")
	rePollSameCursor := env.serve(http.MethodGet, "/sync/pull?limit=1&after="+page.NextCursor,
		"If-None-Match", rePoll.Header().Get("ETag"))

	if tag == "" || rePoll.Header().Get("ETag") == "" {
		t.Fatal("pull responses must carry ETags")
	}

	if rePollSameCursor.Code != http.StatusNotModified {
		t.Errorf("conditional re-poll = %d, want 304 (body: %s)",
			rePollSameCursor.Code, rePollSameCursor.Body.String())
	}
}
