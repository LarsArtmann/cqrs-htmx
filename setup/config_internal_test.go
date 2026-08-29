package setup

import (
	"log/slog"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// TestResolveServiceConfig_Flattened verifies the default mapping: every
// flattened service-construction field lands in the corresponding
// usermgmt.ServiceConfig slot, and the in-memory audit log default is applied.
func TestResolveServiceConfig_Flattened(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)
	projectionFailed := func(projectionName, lastError string) {}

	cfg := Config{
		EventStore:         nil,
		TOTP:               nil,
		Logger:             logger,
		OnProjectionFailed: projectionFailed,
		AsyncStartup:       true,
		SessionTTL:         0,
	}

	out := resolveServiceConfig(cfg)

	if out.EventStore != nil {
		t.Error("EventStore must pass through nil (usermgmt applies the in-memory default)")
	}

	if out.Logger != logger {
		t.Error("Logger must map to ServiceConfig.Logger")
	}

	if out.OnProjectionFailed == nil {
		t.Error("OnProjectionFailed must map to ServiceConfig.OnProjectionFailed")
	}

	if !out.AsyncStartup {
		t.Error("AsyncStartup must map to ServiceConfig.AsyncStartup")
	}

	if out.AuditLog == nil {
		t.Error("flattened path must default AuditLog to an in-memory audit log")
	}
}

// TestResolveServiceConfig_Override verifies the escape hatch: a caller-supplied
// ServiceConfig is copied verbatim (copy, not alias — later mutations of the
// caller's struct must not leak into the bundle), with only the AuditLog
// default applied when nil.
func TestResolveServiceConfig_Override(t *testing.T) {
	t.Parallel()

	pepper := usermgmt.TokenPepper("test-pepper-bytes-32-chars-long!")
	override := usermgmt.ServiceConfig{
		MaxUsers:    1,
		TokenPepper: pepper,
		AuditLog:    nil,
	}

	cfg := Config{ServiceConfig: &override}
	out := resolveServiceConfig(cfg)

	if out.MaxUsers != 1 {
		t.Errorf("MaxUsers override: got %d, want 1", out.MaxUsers)
	}

	if string(out.TokenPepper) != string(pepper) {
		t.Error("TokenPepper override must pass through unchanged")
	}

	if out.AuditLog == nil {
		t.Error("override path must still default AuditLog when nil")
	}

	if override.AuditLog != nil {
		t.Error("caller's ServiceConfig must not be mutated by resolveServiceConfig")
	}

	out.MaxUsers = 99
	if override.MaxUsers != 1 {
		t.Error(
			"resolveServiceConfig must return a copy: mutating it must not affect the caller's ServiceConfig",
		)
	}
}

// TestResolveServiceConfig_OverrideExplicitAuditLog verifies an explicit
// AuditLog inside the override wins over the bundle default.
func TestResolveServiceConfig_OverrideExplicitAuditLog(t *testing.T) {
	t.Parallel()

	audit := usermgmt.NewAuditLog()
	cfg := Config{ServiceConfig: &usermgmt.ServiceConfig{AuditLog: audit}}

	if out := resolveServiceConfig(cfg); out.AuditLog != audit {
		t.Error("explicit ServiceConfig.AuditLog must be preserved, not replaced by the default")
	}
}
