package setup_test

import (
	"context"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// TestNew_ObservabilityBundleWiredToEndToEnd proves the full path: a config
// with an Observability bundle builds a working bundle whose service publishes
// and handles events THROUGH the wired middleware (tracing-only bundle, so no
// metrics reader is needed). The middleware must not block or break the
// command flow — registration succeeds and the read model converges.
func TestNew_ObservabilityBundleWiredToEndToEnd(t *testing.T) {
	t.Parallel()

	bundle, err := middleware.NewOTelBundle(
		cqrsotel.NewTracer("setup-observability-e2e"), nil,
		middleware.WithMetricsDisabled(),
	)
	if err != nil {
		t.Fatalf("NewOTelBundle: %v", err)
	}

	b, err := setup.New(setup.Config{
		Title:         "Observability Test",
		Observability: bundle,
	})
	if err != nil {
		t.Fatalf("New with Observability bundle: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	reg, err := b.Service.Register(context.Background(), usermgmt.RegisterRequest{
		Email: "obs-e2e@example.com", DisplayName: "Obs E2E",
	})
	if err != nil {
		t.Fatalf("Register through OTel-wired bus: %v", err)
	}
	if reg.Session == nil || reg.Session.Token == "" {
		t.Fatal("expected a session from registration through the OTel-wired bus")
	}

	user, err := b.Service.GetUser(context.Background(), reg.User.ID)
	if err != nil {
		t.Fatalf("GetUser after OTel-wired registration: %v", err)
	}
	if user.Email != "obs-e2e@example.com" {
		t.Errorf("user email = %q, want %q (read model must converge through wired middleware)", user.Email, "obs-e2e@example.com")
	}
}
