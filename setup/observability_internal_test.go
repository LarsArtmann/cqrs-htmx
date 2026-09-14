package setup

import (
	"reflect"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
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

	out = resolveServiceConfig(Config{Title: "Test", ServiceConfig: &usermgmt.ServiceConfig{}})
	if len(out.SecurityHooks.PublishMiddleware) != 0 {
		t.Errorf("nil bundle must not add PublishMiddleware on the override path, got %d", len(out.SecurityHooks.PublishMiddleware))
	}
	if len(out.SecurityHooks.HandlerMiddleware) != 0 {
		t.Errorf("nil bundle must not add HandlerMiddleware on the override path, got %d", len(out.SecurityHooks.HandlerMiddleware))
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
// composition order when the consumer also configured SecurityHooks manually:
// the bundle's middleware runs OUTERMOST (the OTel span wraps any
// signing/encryption work), consumer hooks run inside. New validation rejects
// this combination; the direct call documents the order applyObservability
// produces.
func TestResolveServiceConfig_ObservabilityPrependsOverConsumerHooks(t *testing.T) {
	t.Parallel()

	bundle := testObservabilityBundle(t)
	consumerPublish := []event.PublishMiddleware{
		func(next event.Publisher) event.Publisher { return next },
	}
	consumerHandler := []event.Middleware{
		func(next event.Handler) event.Handler { return next },
	}

	out := resolveServiceConfig(Config{
		Title:         "Test",
		Observability: bundle,
		ServiceConfig: &usermgmt.ServiceConfig{
			SecurityHooks: usermgmt.SecurityHooks{
				PublishMiddleware: consumerPublish,
				HandlerMiddleware: consumerHandler,
			},
		},
	})

	bundlePublish := bundle.Publish()
	bundleHandler := bundle.Event()

	if got, want := len(out.SecurityHooks.PublishMiddleware), len(bundlePublish)+len(consumerPublish); got != want {
		t.Fatalf("PublishMiddleware count = %d, want %d", got, want)
	}
	if got, want := len(out.SecurityHooks.HandlerMiddleware), len(bundleHandler)+len(consumerHandler); got != want {
		t.Fatalf("HandlerMiddleware count = %d, want %d", got, want)
	}

	if funcPointer(out.SecurityHooks.PublishMiddleware[0]) != funcPointer(bundlePublish[0]) {
		t.Error("expected the bundle's publish middleware outermost (index 0)")
	}
	if funcPointer(out.SecurityHooks.PublishMiddleware[1]) != funcPointer(consumerPublish[0]) {
		t.Error("expected the consumer publish middleware inside the bundle's (index 1)")
	}
	if funcPointer(out.SecurityHooks.HandlerMiddleware[0]) != funcPointer(bundleHandler[0]) {
		t.Error("expected the bundle's handler middleware outermost (index 0)")
	}
	if funcPointer(out.SecurityHooks.HandlerMiddleware[1]) != funcPointer(consumerHandler[0]) {
		t.Error("expected the consumer handler middleware inside the bundle's (index 1)")
	}
}
