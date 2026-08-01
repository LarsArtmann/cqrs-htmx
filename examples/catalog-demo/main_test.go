package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver"
)

func TestBuildCatalog_Registration(t *testing.T) {
	t.Parallel()

	cat := buildCatalog()

	if cat == nil {
		t.Fatal("buildCatalog() returned nil")
	}

	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	if len(svc.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(svc.Commands))
	}
	if len(svc.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(svc.Events))
	}
	if len(svc.Queries) != 1 {
		t.Errorf("expected 1 query, got %d", len(svc.Queries))
	}
}

func TestCatalog_Validate(t *testing.T) {
	t.Parallel()

	cat := buildCatalog()

	if violations := cat.Validate(); len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("validation violation: %s", v.Message)
		}
	}
}

func TestHTTPHandlers_ServeSpecs(t *testing.T) {
	t.Parallel()

	cat := buildCatalog()

	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "Order Service",
		Version:     "1.0.0",
	})

	mux := http.NewServeMux()
	mux.Handle("/openapi.json", ds.OpenAPISpec())
	mux.Handle("/asyncapi.json", ds.AsyncAPISpec())
	mux.Handle("/diagram.d2", docserver.D2Handler(cat))
	mux.Handle("/health", docserver.HealthCheckHandler(cat))

	server := httptest.NewServer(mux)
	defer server.Close()

	for _, path := range []string{"/openapi.json", "/asyncapi.json", "/health"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}

	resp, err := http.Get(server.URL + "/diagram.d2")
	if err != nil {
		t.Fatalf("GET /diagram.d2: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /diagram.d2: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestOpenAPISpec_ValidJSON(t *testing.T) {
	t.Parallel()

	cat := buildCatalog()

	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "Order Service",
		Version:     "1.0.0",
	})

	server := httptest.NewServer(ds.OpenAPISpec())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET openapi.json: %v", err)
	}
	defer resp.Body.Close()

	var parsed map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		t.Fatalf("openapi.json is not valid JSON: %v", err)
	}

	title, _ := parsed["info"].(map[string]any)
	if title == nil {
		t.Fatal("openapi.json missing 'info' section")
	}
	if title["title"] != "Order Service" {
		t.Errorf("expected info.title 'Order Service', got %v", title["title"])
	}
}
