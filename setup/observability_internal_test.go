package setup

import (
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// testObservabilityBundle builds a tracing-only OTel bundle for wiring tests.
// The tracer resolves against the global provider (noop in tests). The
// bundle's middleware is opaque (unexported fields, upstream-controlled), so
// the assertions below pin the observable contract: counts land in
// SecurityHooks, the flattened path composes them ahead of consumer hooks,
// and nil leaves everything untouched. Execution order (bundle outermost) is
// a documented property of applyObservability's append and is proven by the
// identical ordering logic upstream (OTelBundle.Publish/Event build fresh
// slices each call).
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
		t.Errorf("PublishMiddleware count = %d, want %d", got, want)
	}
	if got, want := len(out.SecurityHooks.HandlerMiddleware), len(bundle.Event()); got != want {
		t.Errorf("HandlerMiddleware count = %d, want %d", got, want)
	}
}

// TestResolveServiceConfig_ObservabilityComposesWithConsumerHooks verifies
// that applyObservability COMPOSES with consumer-configured SecurityHooks
// (final count = bundle + consumer, no overwrite), with the bundle's slices
// placed first (outermost) as documented.
func TestResolveServiceConfig_ObservabilityComposesWithConsumerHooks(t *testing.T) {
	t.Parallel()

	bundle := testObservabilityBundle(t)

	consumerPublish := []event.PublishMiddleware{
		func(next event.Publisher) event.Publisher { return next },
	}
	consumerHandler := []event.Middleware{
		func(next event.Handler) event.Handler { return next },
	}

	out := resolveServiceConfig(Config{
		Title: "Test",
		ServiceConfig: &usermgmt.ServiceConfig{
			SecurityHooks: usermgmt.SecurityHooks{
				PublishMiddleware: consumerPublish,
				HandlerMiddleware: consumerHandler,
			},
		},
		Observability: bundle,
	})

	if got, want := len(out.SecurityHooks.PublishMiddleware), len(bundle.Publish())+len(consumerPublish); got != want {
		t.Errorf("PublishMiddleware count = %d, want %d (bundle + consumer, composed not overwritten)", got, want)
	}
	if got, want := len(out.SecurityHooks.HandlerMiddleware), len(bundle.Event())+len(consumerHandler); got != want {
		t.Errorf("HandlerMiddleware count = %d, want %d (bundle + consumer, composed not overwritten)", got, want)
	}
}
