package main

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer wires the demo handler behind httptest with cookie support.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(newHandler())
	t.Cleanup(server.Close)

	return server
}

// get is a small helper: the demo tests are plain HTTP round trips.
func get(t *testing.T, url string, headers ...[2]string) (int, string, http.Header) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	for _, h := range headers {
		req.Header.Set(h[0], h[1])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(body), resp.Header
}

// pushBatch posts one batch push envelope set and decodes the results.
func pushBatch(t *testing.T, url, body string) (int, syncPushResponse) {
	t.Helper()

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var out syncPushResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode push response %q: %v", raw, err)
	}

	return resp.StatusCode, out
}

type syncPushResponse struct {
	Results []struct {
		CommandID string `json:"commandId"`
		Status    string `json:"status"`
		Error     *struct {
			Message string `json:"message"`
			Family  string `json:"family"`
		} `json:"error"`
	} `json:"results"`
}

// syncPullResponse mirrors the pull wire shape the demo asserts on.
type syncPullResponse struct {
	BackendID  string `json:"backendId"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
	Events     []struct {
		EventID string `json:"eventId"`
		Type    string `json:"type"`
		StreamType string `json:"streamType"`
	} `json:"events"`
}

func pull(t *testing.T, url string) (int, syncPullResponse, http.Header) {
	t.Helper()

	status, body, header := get(t, url)
	if status != http.StatusOK {
		t.Fatalf("pull status = %d, body %s", status, body)
	}

	var out syncPullResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode pull body %q: %v", body, err)
	}

	return status, out, header
}

// TestSyncDemo_PushThenPull walks the protocol's write+read halves in one
// flow: a batch push with one good and one poisoned envelope, then a pull
// that returns the good command's event.
func TestSyncDemo_PushThenPull(t *testing.T) {
	server := newTestServer(t)

	body := `{"commands":[` +
		`{"commandId":"01J8Z0DEMO000000000000000","type":"CreateNote","body":"name=first-note","contentType":"application/x-www-form-urlencoded"},` +
		`{"commandId":"01J8Z0DEMO000000000000001","type":"NoSuchCommand","body":"{}"}]}`

	status, results := pushBatch(t, server.URL+"/sync/push", body)
	if status != http.StatusOK {
		t.Fatalf("push status = %d, want 200 (per-command outcomes)", status)
	}

	if len(results.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(results.Results))
	}

	if results.Results[0].Status != "confirmed" {
		t.Errorf("first outcome = %s (err %+v), want confirmed", results.Results[0].Status, results.Results[0].Error)
	}

	if results.Results[1].Status != "rejected" || results.Results[1].Error == nil || results.Results[1].Error.Family != "rejection" {
		t.Errorf("second outcome = %+v, want rejected/rejection", results.Results[1])
	}

	_, feed, header := pull(t, server.URL+"/sync/pull")
	if feed.BackendID != "sync-demo-v1" {
		t.Errorf("backendId = %q, want sync-demo-v1", feed.BackendID)
	}

	if len(feed.Events) != 1 || feed.Events[0].Type != "Note.created" || feed.Events[0].StreamType != "Note" {
		t.Errorf("events = %+v, want one Note.created on the Note stream", feed.Events)
	}

	if header.Get("ETag") == "" {
		t.Error("pull response carries no ETag")
	}
}

// TestSyncDemo_DuplicateNameRejected pins the conflict contract: the second
// CreateNote with the same name answers rejected with family "rejection" —
// the client surfaces it and drops the command, it never silently retries.
func TestSyncDemo_DuplicateNameRejected(t *testing.T) {
	server := newTestServer(t)

	body := `{"commands":[{"commandId":"01J8Z0DEMO000000000000002","type":"CreateNote",` +
		`"body":"name=dupe","contentType":"application/x-www-form-urlencoded"}]}`
	if status, results := pushBatch(t, server.URL+"/sync/push", body); status != http.StatusOK || results.Results[0].Status != "confirmed" {
		t.Fatalf("first push = %d %+v, want confirmed", status, results.Results)
	}

	_, results := pushBatch(t, server.URL+"/sync/push", body)
	if len(results.Results) != 1 || results.Results[0].Status != "rejected" {
		t.Fatalf("duplicate push = %+v, want one rejected", results.Results)
	}

	if results.Results[0].Error == nil || results.Results[0].Error.Family != "rejection" {
		t.Errorf("duplicate error = %+v, want family rejection", results.Results[0].Error)
	}
}

// TestSyncDemo_PullFilterHidesConfidentialNotes pins the permission seam:
// SecretNote events are invisible without the admin cookie, visible with it.
func TestSyncDemo_PullFilterHidesConfidentialNotes(t *testing.T) {
	server := newTestServer(t)

	body := `{"commands":[` +
		`{"commandId":"01J8Z0DEMO000000000000003","type":"CreateNote","body":"name=open-note","contentType":"application/x-www-form-urlencoded"},` +
		`{"commandId":"01J8Z0DEMO000000000000004","type":"CreateNote","body":"name=secret-note&confidential=true","contentType":"application/x-www-form-urlencoded"}]}`
	if status, results := pushBatch(t, server.URL+"/sync/push", body); status != http.StatusOK {
		t.Fatalf("push status = %d", status)
	}

	for _, r := range results.Results {
		if r.Status != "confirmed" {
			t.Fatalf("outcome = %+v, want confirmed", r)
		}
	}

	_, plain, _ := pull(t, server.URL+"/sync/pull")
	for _, evt := range plain.Events {
		if evt.StreamType == "SecretNote" {
			t.Errorf("unprivileged pull leaked a SecretNote: %+v", evt)
		}
	}

	if len(plain.Events) != 1 {
		t.Errorf("unprivileged events = %d, want 1 (the open note)", len(plain.Events))
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/sync/pull", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "demo_role", Value: "admin"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var admin syncPullResponse
	if err := json.NewDecoder(resp.Body).Decode(&admin); err != nil {
		t.Fatalf("decode admin pull: %v", err)
	}

	if len(admin.Events) != 2 {
		t.Errorf("admin events = %d, want 2 (open + confidential)", len(admin.Events))
	}
}

// TestSyncDemo_ConditionalRePull304 pins the cheap re-poll: same cursor state
// + If-None-Match answers 304 with no body.
func TestSyncDemo_ConditionalRePull304(t *testing.T) {
	server := newTestServer(t)

	_, _, header := pull(t, server.URL+"/sync/pull")
	etag := header.Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on first pull")
	}

	status, body, _ := get(t, server.URL+"/sync/pull", [2]string{"If-None-Match", etag})
	if status != http.StatusNotModified {
		t.Errorf("re-poll status = %d (body %q), want 304", status, body)
	}
}

// TestSyncDemo_AssetsAndOpenAPI pins the serving surface: embedded sync
// assets and the OpenAPI contract for both sync endpoints.
func TestSyncDemo_AssetsAndOpenAPI(t *testing.T) {
	server := newTestServer(t)

	for path, marker := range map[string]string{
		"/sync-worker.js": "cqrshtmx-sync",
		"/sync-client.js": "cqrsSync",
	} {
		status, body, _ := get(t, server.URL+path)
		if status != http.StatusOK || !strings.Contains(body, marker) {
			t.Errorf("GET %s = %d, contains %q = %v", path, status, marker, strings.Contains(body, marker))
		}
	}

	status, body, _ := get(t, server.URL+"/openapi.json")
	if status != http.StatusOK {
		t.Fatalf("openapi.json status = %d", status)
	}

	for _, want := range []string{`"/sync/pull"`, `"/sync/push"`, `"syncPull"`, `"syncPush"`} {
		if !strings.Contains(body, want) {
			t.Errorf("openapi.json missing %s", want)
		}
	}

	var doc struct {
		Paths map[string]json.RawValue `json:"paths"`
	}
	if err := json.Unmarshal([]byte(body), &doc); err != nil || len(doc.Paths) != 2 {
		t.Errorf("openapi.json paths decode err=%v count=%d, want 2 paths", err, len(doc.Paths))
	}
}

// TestSyncDemo_ServerTruthListsNotes pins the projection-side query used by
// the "server truth" panel.
func TestSyncDemo_ServerTruthListsNotes(t *testing.T) {
	server := newTestServer(t)

	body := `{"commands":[{"commandId":"01J8Z0DEMO000000000000005","type":"CreateNote",` +
		`"body":"name=truth-note","contentType":"application/x-www-form-urlencoded"}]}`
	if status, _ := pushBatch(t, server.URL+"/sync/push", body); status != http.StatusOK {
		t.Fatalf("push status = %d", status)
	}

	status, listBody, _ := get(t, server.URL+"/api/notes")
	if status != http.StatusOK || !strings.Contains(listBody, "truth-note") {
		t.Errorf("GET /api/notes = %d %s, want 200 containing truth-note", status, listBody)
	}
}

// keep bytes imported for future request-body tests without churn
var _ = bytes.MinRead
