package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPingRetriesThenSucceeds(t *testing.T) {
	handler := newHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	body := []byte(`{"msg":"hello"}`)

	start := time.Now()
	resp, err := http.Post(server.URL+"/ping", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp.Body.Close()
	elapsed1 := time.Since(start)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("first call: expected 204, got %d", resp.StatusCode)
	}

	// First call retries twice (2 transient failures then success), so it
	// should take at least some time for the backoff delays.
	if elapsed1 < 50*time.Millisecond {
		t.Fatalf("first call completed in %v — expected retry backoff delay", elapsed1)
	}
}

func TestPingImmediateSuccessOnSecondCall(t *testing.T) {
	handler := newHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	body := []byte(`{"msg":"hello"}`)

	// First call: retries twice then succeeds (flaky service recovers after 2 calls).
	resp1, err := http.Post(server.URL+"/ping", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("first POST failed: %v", err)
	}
	resp1.Body.Close()

	if resp1.StatusCode != http.StatusNoContent {
		t.Fatalf("first call: expected 204, got %d", resp1.StatusCode)
	}

	// Second call: service already recovered, should succeed immediately.
	start := time.Now()
	resp2, err := http.Post(server.URL+"/ping", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("second POST failed: %v", err)
	}
	resp2.Body.Close()
	elapsed2 := time.Since(start)

	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("second call: expected 204, got %d", resp2.StatusCode)
	}

	// Second call should be near-instant (no retries needed).
	if elapsed2 > 50*time.Millisecond {
		t.Fatalf("second call took %v — expected immediate success", elapsed2)
	}
}

func TestPingResponseBodyEmpty(t *testing.T) {
	handler := newHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	body := []byte(`{"msg":"hello"}`)
	resp, err := http.Post(server.URL+"/ping", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if len(respBody) > 0 {
		t.Fatalf("expected empty body, got %q", string(respBody))
	}
}
