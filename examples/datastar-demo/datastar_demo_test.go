package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
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
