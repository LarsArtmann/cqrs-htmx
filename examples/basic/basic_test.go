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

	resp, err := http.Post(srv.URL+"/api/items", "application/json", strings.NewReader(`{"name":"smoke-item"}`))
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
