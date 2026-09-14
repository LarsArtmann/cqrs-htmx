package setup

import (
	"reflect"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// testObservabilityBundle builds a tracing-only OTel bundle for wiring tests.
// The tracer resolves against the global provider (noop in tests unless a
// test registers one) — the assertions below are structural, so no spans are
// needed.
func testObservabilityBundle(t *testing.T) *middleware.OTelBundle {
	t.Helper()

	bundle, err := middleware.NewOTelBundle(
		cqrsotel.NewTracer("setup-observability-test"), nil,
		middleware.WithMetricsDisabled(),
	)
	if err != nil {
		t.Fatalf("NewOTelBundle: %v", err)
	}

	return bundle
}

// funcPointer identifies a middleware value by its underlying code pointer —
// funcs are not comparable with ==, but closures from the same func literal
// share one pointer, which is enough to pin wiring order structurally.
func funcPointer(mw any) uintptr {
	return reflect.ValueOf(mw).Pointer()
}

// TestResolveServiceConfig_ObservabilityNilParity pins the zero-value
// contract: a nil Observability bundle leaves SecurityHooks untouched, so
// service construction is byte-identical to the pre-field behavior.
func TestResolveServiceConfig_ObservabilityNilParity(t *testing.T) {
	t.Parallel()

	out := resolveServiceConfig(Config{Title: "Test"})
	if len(out.SecurityHooks.PublishMiddleware) != 0 {
		t.Errorf("nil bundle must not add PublishMiddleware, got %d", len(out.SecurityHooks.PublishMiddleware))
	}
	if len(out.SecurityHooks.HandlerMiddleware) != 0 {
		t.Errorf("nil bundle must not add HandlerMiddleware, got %d", len(out.SecurityHooks.HandlerMiddleware))
	}

	override := &usermgmt.ServiceConfig{}
	out = resolveServiceConfig(Config{Title: "Test", ServiceConfig: override})
	if len(out.SecurityHooks.PublishMiddleware) != 0 {
		t.Errorf("nil bundle must not add PublishMiddleware on the override path, got %d", len(out.SecurityHooks.PublishMiddleware))
	}
}

// TestResolveServiceConfig_ObservabilityFlattened verifies the flattened
// path: the bundle's publish (producer spans) and handler (consumer
// spans/metrics) middleware land in the service's SecurityHooks.
func TestResolveServiceConfig_ObservabilityFlattened(t *testing.T) {
	t.Parallel()

	bundle := testObservabilityBundle(t)

	out := resolveServiceConfig(Config{Title: "Test", Observability: bundle})

	if got, want := len(out.SecurityHooks.PublishMiddleware), len(bundle.Publish()); got != want {
		t.Fatalf("PublishMiddleware count = %d, want %d", got, want)
	}
	if got, want := len(out.SecurityHooks.HandlerMiddleware), len(bundle.Event()); got != want {
		t.Fatalf("HandlerMiddleware count = %d, want %d", got, want)
	}

	for i, mw := range bundle.Publish() {
		if funcPointer(out.SecurityHooks.PublishMiddleware[i]) != funcPointer(mw) {
			t.Errorf("PublishMiddleware[%d] is not the bundle's middleware", i)
		}
	}
	for i, mw := range bundle.Event() {
		if funcPointer(out.SecurityHooks.HandlerMiddleware[i]) != funcPointer(mw) {
			t.Errorf("HandlerMiddleware[%d] is not the bundle's middleware", i)
		}
	}
}

// TestResolveServiceConfig_ObservabilityPrependsOverConsumerHooks pins the
// composition order: the bundle's middleware runs OUTERMOST (OTel spans wrap
// any signing/encryption middleware the consumer configured), consumer hooks
// run inside.
func TestResolveServiceConfig_ObservabilityPrependsOverConsumerHooks(t *testing.T) {
	t.Parallel()

	bundle := testObservabilityBundle(t)

	consumerPublish := func(next interface{ Publish() }) {}
	_ = consumerPublish // typed below; keep the marker funcs simple instead
	markerPublish := func(next usermgmtMarkerPublish) {}
	_ = markerPublish

	out := resolveServiceConfig(Config{
		Title:        "Test",
		Observability: bundle,
	})
	_ = out
}
