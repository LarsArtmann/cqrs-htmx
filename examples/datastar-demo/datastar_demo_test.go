package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/go-datastar/datastartest"
)

// Smoke test: keeps the DataStar todo demo runnable — catches wiring drift
// (broken mounts, renamed routes) that a build alone cannot see.

func TestSmoke_IndexAndTodoRoundTrip(t *testing.T) {
	t.Parallel()

	cqrs := NewCQRS()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("POST /api/todos", handleCreateTodo(cqrs))
	mux.HandleFunc("GET /api/todos", handleListTodos(cqrs))
	mux.Handle("GET /datastar.js", ds.ScriptHandler())

	srv := httptest.NewServer(mux)
	defer srv.Close()

	index, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = index.Body.Close() }()
	if index.StatusCode != http.StatusOK {
		t.Fatalf("GET /: got %d, want 200", index.StatusCode)
	}

	created, err := http.Post(srv.URL+"/api/todos", "application/json", strings.NewReader(`{"text":"smoke-todo"}`))
	if err != nil {
		t.Fatalf("POST /api/todos: %v", err)
	}
	defer func() { _ = created.Body.Close() }()
	if created.StatusCode < 200 || created.StatusCode > 299 {
		t.Fatalf("POST /api/todos: got %d, want 2xx", created.StatusCode)
	}

	list, err := http.Get(srv.URL + "/api/todos")
	if err != nil {
		t.Fatalf("GET /api/todos: %v", err)
	}
	defer func() { _ = list.Body.Close() }()
	if list.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/todos: got %d, want 200", list.StatusCode)
	}
}

// TestSmoke_VersionedScriptServing pins the versioned-serving contract: the
// index carries the client version as a cache-busting query parameter, and
// the script handler answers conditional requests with 304 (the caching
// setup ScriptTagVersioned exists to complement).
func TestSmoke_VersionedScriptServing(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveIndex)
	mux.Handle("GET /datastar.js", ds.ScriptHandler())

	srv := httptest.NewServer(mux)
	defer srv.Close()

	index, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	indexBody, err := io.ReadAll(index.Body)
	_ = index.Body.Close()
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	datastartest.RequireScriptTagVersion(t, string(indexBody), "/datastar.js")

	first, err := http.Get(srv.URL + "/datastar.js")
	if err != nil {
		t.Fatalf("GET /datastar.js: %v", err)
	}
	etag := first.Header.Get("ETag")
	_ = first.Body.Close()
	if etag == "" {
		t.Fatal("GET /datastar.js: no ETag header to harvest")
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/datastar.js", nil)
	if err != nil {
		t.Fatalf("build conditional request: %v", err)
	}
	req.Header.Set("If-None-Match", etag)
	conditional, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("conditional GET /datastar.js: %v", err)
	}
	_ = conditional.Body.Close()
	if conditional.StatusCode != http.StatusNotModified {
		t.Fatalf("conditional GET with If-None-Match: %s: got %d, want 304", etag, conditional.StatusCode)
	}
}
