package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPingDispatchesAndReturns204(t *testing.T) {
	handler, promProvider, otelProvider, err := newHandler(slog.Default())
	if err != nil {
		t.Fatalf("newHandler: %v", err)
	}
	defer otelProvider.Shutdown(context.Background())
	defer promProvider.Shutdown(context.Background())

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Post(server.URL+"/ping", "application/json", bytes.NewReader([]byte(`{"msg":"hello"}`)))
	if err != nil {
		t.Fatalf("POST /ping: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestMetricsEndpointReturnsPrometheusFormat(t *testing.T) {
	handler, promProvider, otelProvider, err := newHandler(slog.Default())
	if err != nil {
		t.Fatalf("newHandler: %v", err)
	}
	defer otelProvider.Shutdown(context.Background())
	defer promProvider.Shutdown(context.Background())

	server := httptest.NewServer(handler)
	defer server.Close()

	// Dispatch a command first so metrics are recorded.
	resp, err := http.Post(server.URL+"/ping", "application/json", bytes.NewReader([]byte(`{"msg":"hello"}`)))
	if err != nil {
		t.Fatalf("POST /ping: %v", err)
	}
	resp.Body.Close()

	// Check /metrics endpoint.
	resp2, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	body, _ := io.ReadAll(resp2.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "cqrs_operation") {
		t.Fatalf("metrics output does not contain cqrs_operation metrics:\n%s", bodyStr[:min(len(bodyStr), 500)])
	}
}
