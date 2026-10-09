package cqrshtmx

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// syncTestCmd is a minimal command.Command for push tests. The type is
// per-instance because tests register it under different dispatcher names.
type syncTestCmd struct {
	cmdType  command.Type
	streamID id.StreamID
	cmdID    id.CommandID
	Name     string
}

func (c *syncTestCmd) Type() command.Type    { return c.cmdType }
func (c *syncTestCmd) StreamID() id.StreamID { return c.streamID }
func (c *syncTestCmd) ID() id.CommandID      { return c.cmdID }

func newSyncTestCmd(cmdType command.Type) *syncTestCmd {
	return &syncTestCmd{
		cmdType:  cmdType,
		streamID: id.NewStreamID(),
		cmdID:    id.NewCommandID(),
	}
}

// newSyncPushApp builds an App with one registered SyncTest command whose
// handler records dispatches and rejects names containing "reject" (Rejection)
// or "conflict" (Conflict).
func newSyncPushApp(t *testing.T) (*App, *[]string) {
	t.Helper()

	cmdDisp := command.NewDispatcher()

	var dispatched []string

	err := cmdDisp.Register("SyncTest", func(_ context.Context, cmd command.Command) error {
		syncCmd, ok := cmd.(*syncTestCmd)
		if !ok {
			return errorfamily.NewCorruption("sync_test.bad_type", "unexpected command type")
		}

		switch {
		case strings.Contains(syncCmd.Name, "conflict"):
			return errorfamily.NewConflict("sync_test.conflict", "state moved on: "+syncCmd.Name)
		case strings.Contains(syncCmd.Name, "reject"):
			return errorfamily.NewRejection("sync_test.rejected", "invalid: "+syncCmd.Name)
		default:
			dispatched = append(dispatched, syncCmd.Name)

			return nil
		}
	})
	if err != nil {
		t.Fatalf("register SyncTest: %v", err)
	}

	app := MustNew(Config{Commands: cmdDisp})

	// The endpoint registration is what makes the type pushable.
	_ = app.Command("SyncTest", DecodeJSON(func(req struct {
		Name string `json:"name"`
	},
	) (command.Command, error) {
		cmd := newSyncTestCmd("SyncTest")
		cmd.Name = req.Name

		return cmd, nil
	}))

	return app, &dispatched
}

func postSyncPush(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/sync/push", strings.NewReader(body))
	req.Header.Set("Content-Type", ContentTypeJSON)
	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

func decodeSyncPushResults(t *testing.T, rec *httptest.ResponseRecorder) []SyncPushResult {
	t.Helper()

	var resp SyncPushResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal push response: %v\nbody: %s", err, rec.Body.String())
	}

	return resp.Results
}

func TestSyncPushHandler_ConfirmsAndDispatchesEachEnvelope(t *testing.T) {
	t.Parallel()

	app, dispatched := newSyncPushApp(t)
	rec := postSyncPush(app.SyncPushHandler(), `{"commands":[
		{"commandId":"cmd-1","type":"SyncTest","body":"{\"name\":\"alpha\"}"},
		{"commandId":"cmd-2","type":"SyncTest","body":"{\"name\":\"beta\"}"}
	]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	results := decodeSyncPushResults(t, rec)
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}

	for _, res := range results {
		if res.Status != SyncPushStatusConfirmed || res.Error != nil {
			t.Errorf("result %s: status=%s error=%+v, want confirmed", res.CommandID, res.Status, res.Error)
		}
	}

	if len(*dispatched) != 2 || (*dispatched)[0] != "alpha" || (*dispatched)[1] != "beta" {
		t.Errorf("dispatched = %v, want [alpha beta]", *dispatched)
	}
}

func TestSyncPushHandler_ClassifiedRejectionsAreIsolatedPerCommand(t *testing.T) {
	t.Parallel()

	app, _ := newSyncPushApp(t)
	rec := postSyncPush(app.SyncPushHandler(), `{"commands":[
		{"commandId":"ok-1","type":"SyncTest","body":"{\"name\":\"fine\"}"},
		{"commandId":"bad-1","type":"SyncTest","body":"{\"name\":\"reject-me\"}"},
		{"commandId":"clash-1","type":"SyncTest","body":"{\"name\":\"conflict-here\"}"},
		{"commandId":"bad-type","type":"NoSuchCommand","body":"{}"}
	]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (per-command outcomes)", rec.Code)
	}

	results := decodeSyncPushResults(t, rec)
	if len(results) != 4 {
		t.Fatalf("results = %d, want 4", len(results))
	}

	byID := make(map[string]SyncPushResult, len(results))
	for _, res := range results {
		byID[res.CommandID] = res
	}

	if res := byID["ok-1"]; res.Status != SyncPushStatusConfirmed {
		t.Errorf("ok-1: status=%s, want confirmed", res.Status)
	}

	if res := byID["bad-1"]; res.Status != SyncPushStatusRejected || res.Error.Family != "rejection" {
		t.Errorf("bad-1: %+v, want rejected/rejection", res)
	}

	if res := byID["clash-1"]; res.Status != SyncPushStatusRejected || res.Error.Family != "conflict" {
		t.Errorf("clash-1: %+v, want rejected/conflict", res)
	}

	if res := byID["bad-type"]; res.Status != SyncPushStatusRejected {
		t.Errorf("bad-type: %+v, want rejected", res)
	}
}

func TestSyncPushHandler_RejectsMalformedBatches(t *testing.T) {
	t.Parallel()

	app, _ := newSyncPushApp(t)
	handler := app.SyncPushHandler()

	if rec := postSyncPush(handler, `{`); rec.Code != http.StatusBadRequest {
		t.Errorf("undecodable body: status = %d, want 400", rec.Code)
	}

	if rec := postSyncPush(handler, `{"commands":[]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty batch: status = %d, want 400", rec.Code)
	}

	oversized := `{"commands":[` + strings.Repeat(
		`{"commandId":"x","type":"SyncTest","body":"{}"},`,
		MaxSyncPushBatch+1,
	) + `{"commandId":"x","type":"SyncTest","body":"{}"}]}`
	if rec := postSyncPush(handler, oversized); rec.Code != http.StatusBadRequest {
		t.Errorf("oversized batch: status = %d, want 400", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/sync/push", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: status = %d, want 405", rec.Code)
	}
}

func TestSyncPushHandler_RequiresCommandID(t *testing.T) {
	t.Parallel()

	app, _ := newSyncPushApp(t)
	rec := postSyncPush(app.SyncPushHandler(), `{"commands":[{"type":"SyncTest","body":"{\"name\":\"x\"}"}]}`)

	results := decodeSyncPushResults(t, rec)
	if len(results) != 1 || results[0].Status != SyncPushStatusRejected || results[0].CommandID != "" {
		t.Fatalf("results = %+v, want one rejected with empty commandId", results)
	}
}

func TestSyncPushHandler_StampesCommandIDOnInnerRequest(t *testing.T) {
	t.Parallel()

	cmdDisp := command.NewDispatcher()
	if err := cmdDisp.Register("HeaderProbe", func(_ context.Context, _ command.Command) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	app := MustNew(Config{Commands: cmdDisp})

	var seenCommandID, seenContentType string

	_ = app.Command("HeaderProbe", DecodeJSONWithRequest(func(r *http.Request, _ struct{}) (command.Command, error) {
		seenCommandID = r.Header.Get(CommandIDHeader)
		seenContentType = r.Header.Get("Content-Type")

		return newSyncTestCmd("HeaderProbe"), nil
	}))

	rec := postSyncPush(
		app.SyncPushHandler(),
		`{"commands":[{"commandId":"probe-42","type":"HeaderProbe","body":"{}","contentType":"application/x-www-form-urlencoded"}]}`,
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	if seenCommandID != "probe-42" {
		t.Errorf("inner X-Command-Id = %q, want probe-42", seenCommandID)
	}

	if seenContentType != "application/x-www-form-urlencoded" {
		t.Errorf("inner Content-Type = %q, want the envelope's contentType", seenContentType)
	}
}

func TestSyncPushHandler_EnforcesPerEnvelopeAuthorization(t *testing.T) {
	t.Parallel()

	cmdDisp := command.NewDispatcher()
	if err := cmdDisp.Register("Gated", func(_ context.Context, _ command.Command) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	app := MustNew(Config{
		Commands:        cmdDisp,
		UserIDExtractor: func(*http.Request) (UserID, error) { return UserID{}, ErrUnauthorized },
	})

	_ = app.Command("Gated",
		Authorize("resource", "action"),
		DecodeJSON(func(_ struct{}) (command.Command, error) {
			return newSyncTestCmd("Gated"), nil
		}),
	)

	rec := postSyncPush(app.SyncPushHandler(), `{"commands":[{"commandId":"g-1","type":"Gated","body":"{}"}]}`)

	results := decodeSyncPushResults(t, rec)
	if len(results) != 1 || results[0].Status != SyncPushStatusRejected {
		t.Fatalf("results = %+v, want rejected (authz enforced)", results)
	}
}

func TestSyncPushHandler_AfterDispatchHookSeesInnerRequest(t *testing.T) {
	t.Parallel()

	cmdDisp := command.NewDispatcher()
	if err := cmdDisp.Register("Hooked", func(_ context.Context, _ command.Command) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	var hookCommandIDs []string

	app := MustNew(Config{
		Commands: cmdDisp,
		Queries:  query.NewDispatcher(),
		AfterDispatch: func(_ context.Context, r *http.Request, _ error) {
			hookCommandIDs = append(hookCommandIDs, r.Header.Get(CommandIDHeader))
		},
	})

	_ = app.Command("Hooked", DecodeJSON(func(_ struct{}) (command.Command, error) {
		return newSyncTestCmd("Hooked"), nil
	}))

	rec := postSyncPush(app.SyncPushHandler(), `{"commands":[
		{"commandId":"h-1","type":"Hooked","body":"{}"},
		{"commandId":"h-2","type":"Hooked","body":"{}"}
	]}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}

	if len(hookCommandIDs) != 2 || hookCommandIDs[0] != "h-1" || hookCommandIDs[1] != "h-2" {
		t.Errorf("hook command IDs = %v, want [h-1 h-2]", hookCommandIDs)
	}
}

func TestSyncPushHandler_TransientFailuresKeepRetryableFamily(t *testing.T) {
	t.Parallel()

	cmdDisp := command.NewDispatcher()
	err := cmdDisp.Register("Flaky", func(_ context.Context, _ command.Command) error {
		return errorfamily.NewTransient("flaky.down", "database unavailable")
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	app := MustNew(Config{Commands: cmdDisp})

	_ = app.Command("Flaky", DecodeJSON(func(_ struct{}) (command.Command, error) {
		return newSyncTestCmd("Flaky"), nil
	}))

	rec := postSyncPush(app.SyncPushHandler(), `{"commands":[{"commandId":"f-1","type":"Flaky","body":"{}"}]}`)

	results := decodeSyncPushResults(t, rec)
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}

	if results[0].Error == nil || results[0].Error.Family != "transient" {
		t.Fatalf("outcome = %+v, want rejected with family transient (client re-queues)", results[0])
	}

	// 5xx detail is redacted from the client message.
	if strings.Contains(results[0].Error.Message, "database unavailable") {
		t.Errorf("transient message leaks internals: %q", results[0].Error.Message)
	}
}
