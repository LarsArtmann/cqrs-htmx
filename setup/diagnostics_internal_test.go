package setup

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// warnLogger returns a logger and the buffer it writes to, filtered to warnings
// and above so the diagnostic under test is the only thing that lands there.
func warnLogger() (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}

	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})), buf
}

func TestNew_WarnsWhenAdminPanelHasNoAdmin(t *testing.T) {
	logger, logBuf := warnLogger()

	bundle, err := New(Config{
		Logger:           logger,
		DisableDashboard: true,
		DisableLogin:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer bundle.Close() //nolint:errcheck // test cleanup

	if bundle.Admin == nil {
		t.Fatal("admin panel should be mounted by default")
	}

	out := logBuf.String()
	if !strings.Contains(out, "nobody holds admin") {
		t.Fatalf("expected the unreachable-panel warning, got %q", out)
	}

	if !strings.Contains(out, "GrantRoles") {
		t.Fatalf("warning must carry the copy-paste fix, got %q", out)
	}
}

func TestNew_NoWarningWhenGlobalAdminExists(t *testing.T) {
	bundle, err := New(Config{
		Logger:           slog.New(slog.DiscardHandler),
		DisableDashboard: true,
		DisableLogin:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer bundle.Close() //nolint:errcheck // test cleanup

	actor := identitymodel.ActorIDFromUser(identitymodel.SyntheticUserID("setup-panel-admin"))
	domain := identitymodel.NewTenantID(identitymodel.WildcardDomain)

	if err := bundle.Service.GrantRoles(
		context.Background(), actor, domain,
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	// Re-run the diagnostic with a capturing logger now that the grant exists.
	logger, logBuf := warnLogger()
	bundle.config.Logger = logger

	bundle.warnOnUnreachableAdminPanel()

	if strings.Contains(logBuf.String(), "nobody holds admin") {
		t.Fatalf("warning fired despite an existing global admin: %q", logBuf.String())
	}
}

func TestNew_NoWarningWhenAdminDisabled(t *testing.T) {
	logger, logBuf := warnLogger()

	bundle, err := New(Config{
		Logger:           logger,
		DisableAdmin:     true,
		DisableDashboard: true,
		DisableLogin:     true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer bundle.Close() //nolint:errcheck // test cleanup

	// New already ran the diagnostic; run it again to be explicit about the
	// DisableAdmin path.
	bundle.warnOnUnreachableAdminPanel()

	if logBuf.Len() != 0 {
		t.Fatalf("no warning expected without a panel, got %q", logBuf.String())
	}
}

func TestAsyncStartup_CoversBothWiringPaths(t *testing.T) {
	if (Config{}).asyncStartup() {
		t.Fatal("zero config must not report async startup")
	}

	if !(Config{AsyncStartup: true}).asyncStartup() {
		t.Fatal("flattened AsyncStartup must be honoured")
	}

	if !(Config{ServiceConfig: &usermgmt.ServiceConfig{AsyncStartup: true}}).asyncStartup() {
		t.Fatal("ServiceConfig.AsyncStartup must be honoured")
	}
}
