package main

import (
	"bytes"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Smoke tests: every example must stay runnable — these catch wiring drift
// (broken mounts, renamed routes, decoder/render option misuse) that a build
// alone cannot see.

func TestSmoke_IndexAndHealth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(newHandler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /: got %d, want 200", resp.StatusCode)
	}

	health, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer func() { _ = health.Body.Close() }()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: got %d, want 200", health.StatusCode)
	}
}

func TestSmoke_CommandAndQueryRoundTrip(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(newHandler())
	defer srv.Close()

	// The page's form is the contract: urlencoded, like native htmx submits.
	resp, err := http.Post(srv.URL+"/api/items", "application/x-www-form-urlencoded", strings.NewReader("name=smoke-item"))
	if err != nil {
		t.Fatalf("POST /api/items: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/items: got %d, want 201", resp.StatusCode)
	}

	list, err := http.Get(srv.URL + "/api/items")
	if err != nil {
		t.Fatalf("GET /api/items: %v", err)
	}
	defer func() { _ = list.Body.Close() }()

	var items []item
	if err := json.UnmarshalRead(list.Body, &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	found := false
	for _, it := range items {
		if it.Name == "smoke-item" {
			found = true
		}
	}
	if !found {
		t.Fatalf("smoke-item not in list: %+v", items)
	}
}

func TestSmoke_ActorMetadataPropagation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(newHandler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/audit", "application/json", strings.NewReader(`{"action":"smoke-action"}`))
	if err != nil {
		t.Fatalf("POST /api/audit: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /api/audit: got %d, want 202", resp.StatusCode)
	}

	list, err := http.Get(srv.URL + "/api/audit")
	if err != nil {
		t.Fatalf("GET /api/audit: %v", err)
	}
	defer func() { _ = list.Body.Close() }()

	body := &bytes.Buffer{}
	if _, err := body.ReadFrom(list.Body); err != nil {
		t.Fatalf("read audit: %v", err)
	}

	var entries []auditEntry
	if err := json.Unmarshal(body.Bytes(), &entries); err != nil {
		t.Fatalf("decode audit: %v (%s)", err, body.String())
	}
	if len(entries) == 0 {
		t.Fatalf("audit trail empty — actor metadata was not propagated: %s", body.String())
	}

	last := entries[len(entries)-1]
	if last.Action != "smoke-action" {
		t.Errorf("action = %q, want smoke-action", last.Action)
	}
	if !strings.HasPrefix(last.Actor, "user:") {
		t.Errorf("actor = %q, want a kind-prefixed user actor (user:...)", last.Actor)
	}
	if last.User == "" {
		t.Error("user metadata is empty — UserID did not propagate from the request context")
	}
}

// TestSmoke_PartialRenderShowcase proves the three partial-render helpers
// serve distinct shapes from the same endpoints: RenderPartialOrFull (full
// page vs list fragment), OOBHTML (out-of-band #item-count block riding the
// partial response), and RenderIf (bare count when HX-Target is #item-count).
func TestSmoke_PartialRenderShowcase(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(newHandler())
	defer srv.Close()

	// Seed one item so the list/count assertions are deterministic — db is
	// package-level shared state and parallel tests race on its contents.
	seed, err := http.Post(srv.URL+"/api/items", "application/x-www-form-urlencoded", strings.NewReader("name=partial-demo-item"))
	if err != nil {
		t.Fatalf("seed item: %v", err)
	}
	defer func() { _ = seed.Body.Close() }()
	if seed.StatusCode != http.StatusCreated {
		t.Fatalf("seed item: got %d, want 201", seed.StatusCode)
	}

	get := func(path string, header map[string]string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatalf("build request %s: %v", path, err)
		}
		for k, v := range header {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer func() { _ = resp.Body.Close() }()
		body := &bytes.Buffer{}
		if _, err := body.ReadFrom(resp.Body); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return resp.StatusCode, body.String()
	}

	t.Run("DirectVisitRendersFullPage", func(t *testing.T) {
		status, body := get("/demo/items", nil)
		if status != http.StatusOK {
			t.Fatalf("got %d, want 200", status)
		}
		if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Errorf("direct visit did not render the full page:\n%s", body)
		}
		if !strings.Contains(body, `id="item-count"`) || !strings.Contains(body, "<ul") {
			t.Errorf("full page is missing the badge or the item list:\n%s", body)
		}
	})

	t.Run("HTMXRequestRendersPartialWithOOB", func(t *testing.T) {
		status, body := get("/demo/items", map[string]string{"HX-Request": "true"})
		if status != http.StatusOK {
			t.Fatalf("got %d, want 200", status)
		}
		if strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Errorf("HTMX request rendered the full page, want only the fragment:\n%s", body)
		}
		if !strings.Contains(body, "<ul") {
			t.Errorf("partial response is missing the item list:\n%s", body)
		}
		if !strings.Contains(body, `hx-swap-oob="true"`) || !strings.Contains(body, `id="item-count"`) {
			t.Errorf("partial response is missing the OOBHTML #item-count block:\n%s", body)
		}
	})

	t.Run("HistoryRestoreRendersFullPage", func(t *testing.T) {
		_, body := get("/demo/items", map[string]string{
			"HX-Request":                 "true",
			"HX-History-Restore-Request": "true",
		})
		if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Errorf("history-restore request must render the full page:\n%s", body)
		}
	})

	t.Run("RenderIfCountTargetAnswersBareCount", func(t *testing.T) {
		status, body := get("/demo/items/count", map[string]string{
			"HX-Request": "true",
			"HX-Target":  "#item-count",
		})
		if status != http.StatusOK {
			t.Fatalf("got %d, want 200", status)
		}
		trimmed := strings.TrimSpace(body)
		if !strings.HasSuffix(trimmed, " items") {
			t.Errorf("RenderIf match branch should answer the bare count, got %q", trimmed)
		}
		if strings.Contains(body, "<ul") || strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Errorf("count-only response should not contain the list or page:\n%s", body)
		}
	})

	t.Run("RenderIfNoTargetRendersFullPage", func(t *testing.T) {
		_, body := get("/demo/items/count", nil)
		if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Errorf("no-target request should render the full page:\n%s", body)
		}
	})
}
