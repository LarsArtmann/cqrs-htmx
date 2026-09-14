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

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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

// TestOtelHTTPRootSpanAndTraceparentExtraction proves the two §2.5 claims:
// (1) wrapping the handler in otelhttp produces a server-kind ROOT span, and
// (2) an incoming W3C traceparent header is extracted, so the root span nests
// under the caller's trace. Uses an isolated SpanRecorder provider (not the
// global one) so the assertion is deterministic regardless of test order.
func TestOtelHTTPRootSpanAndTraceparentExtraction(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	handler, promProvider, otelProvider, err := newHandler(
		slog.New(slog.DiscardHandler),
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithPropagators(cqrsotel.NewTextMapPropagator()),
	)
	if err != nil {
		t.Fatalf("newHandler: %v", err)
	}
	defer otelProvider.Shutdown(context.Background())
	defer promProvider.Shutdown(context.Background())

	server := httptest.NewServer(handler)
	defer server.Close()

	const (
		parentTraceID = "11111111111111111111111111111111"
		parentSpanID  = "2222222222222222"
	)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/ping", bytes.NewReader([]byte(`{"msg":"hello"}`)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("traceparent", "00-"+parentTraceID+"-"+parentSpanID+"-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /ping: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	var root sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended() {
		if span.SpanKind() == trace.SpanKindServer {
			root = span

			break
		}
	}
	if root == nil {
		t.Fatalf("no server-kind span recorded; spans: %+v", recorder.Ended())
	}

	wantTrace, _ := trace.TraceIDFromHex(parentTraceID)
	wantSpan, _ := trace.SpanIDFromHex(parentSpanID)
	if root.SpanContext().TraceID() != wantTrace {
		t.Errorf(
			"root span trace ID = %s, want %s (traceparent not extracted)",
			root.SpanContext().TraceID(),
			wantTrace,
		)
	}
	if got := root.Parent().SpanID(); got != wantSpan {
		t.Errorf("root span parent = %s, want %s (traceparent not extracted)", got, wantSpan)
	}
}
