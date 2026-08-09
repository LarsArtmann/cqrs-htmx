package main

import (
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/samber/do/v2"
)

// newTestContainer creates a production container then overrides the TOTP
// provider with a stub. This demonstrates the canonical test-container pattern:
// production wiring + targeted overrides via do.OverrideNamedValue.
//
// do.Override* is safe in _test.go files (DO-3 rule explicitly allows it).
func newTestContainer(t *testing.T) (*Container, func()) {
	t.Helper()

	container, cleanup := NewContainer(AppConfig{
		TOTPIssuer: "test",
	})

	// Override the TOTP provider with a no-op stub for tests.
	// This avoids real TOTP secret generation during unit tests.
	// OverrideNamed (not OverrideNamedValue) preserves the interface type
	// so that InvokeNamed[usermgmt.TOTPProvider] resolves correctly.
	do.OverrideNamed(container.injector, "auth.totp", func(_ do.Injector) (usermgmt.TOTPProvider, error) {
		return stubTOTP{}, nil
	})

	return container, cleanup
}

// stubTOTP satisfies usermgmt.TOTPProvider without doing any real TOTP work.
type stubTOTP struct{}

func (stubTOTP) GenerateSecret(_ string) ([]byte, string, string, error) {
	return []byte("test"), "test-base32", "otpauth://test", nil
}
func (stubTOTP) ValidateCode(_ []byte, _ string) bool { return true }

// TestContainerResolvesService verifies that the container can resolve the
// usermgmt.Service with the overridden TOTP provider.
func TestContainerResolvesService(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	svc, err := container.Service()
	if err != nil {
		t.Fatalf("resolve Service: %v", err)
	}
	if svc == nil {
		t.Fatal("Service is nil")
	}
}

// TestContainerResolvesApp verifies that the cqrshtmx.App resolves correctly.
func TestContainerResolvesApp(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	app, err := container.App()
	if err != nil {
		t.Fatalf("resolve App: %v", err)
	}
	if app == nil {
		t.Fatal("App is nil")
	}
}

// TestContainerServiceIsSingleton verifies that the container returns the
// same *usermgmt.Service instance on every invocation (lazy singleton).
func TestContainerServiceIsSingleton(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	svc1, err := container.Service()
	if err != nil {
		t.Fatalf("resolve Service: %v", err)
	}
	svc2, err := container.Service()
	if err != nil {
		t.Fatalf("resolve Service (2nd): %v", err)
	}
	if svc1 != svc2 {
		t.Fatal("Service is not a singleton — got different instances")
	}
}

// TestContainerOverrideTOTP verifies that the named override actually replaces
// the production TOTP provider.
func TestContainerOverrideTOTP(t *testing.T) {
	container, cleanup := newTestContainer(t)
	defer cleanup()

	totp, err := do.InvokeNamed[usermgmt.TOTPProvider](container.injector, "auth.totp")
	if err != nil {
		t.Fatalf("resolve TOTP provider: %v", err)
	}

	if _, ok := totp.(stubTOTP); !ok {
		t.Fatalf("expected stubTOTP, got %T", totp)
	}
}

// TestContainerCleanupCallsShutdown verifies that the cleanup function
// properly shuts down the container without panicking.
func TestContainerCleanupCallsShutdown(t *testing.T) {
	container, cleanup := NewContainer(AppConfig{
		TOTPIssuer: "shutdown-test",
	})

	// Force Service creation so the lifecycle wrapper is tracked.
	if _, err := container.Service(); err != nil {
		t.Fatalf("resolve Service: %v", err)
	}

	// cleanup should call injector.Shutdown() which calls
	// serviceLifecycle.Shutdown() which calls svc.Close().
	// If Close() panics or errors, the test fails.
	cleanup()
}
