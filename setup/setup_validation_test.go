package setup_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/adminui/v4"
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// Tests in this file cover config validation: every Config field that the
// setup.New/MustNew pair refuses must return a non-nil error. The cases fall
// into three shapes:
//
//   - syntactic: path strings that must start with "/", LoginRedirect that must
//     start with "/" or be an http(s) URL, root-path rejection (/ is reserved
//     for the login page).
//   - conflict: two paths resolving to the same route (admin vs dashboard, any
//     path vs health).
//   - cross-field: AdminMode=ModeTenantAdmin without TenantID.

// --- Path syntax validation: must start with "/" ---

func TestNew_ConfigValidation_InvalidAdminPath(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:     "Invalid Path",
		AdminPath: "admin",
	})
	if err == nil {
		t.Fatal("expected error for AdminPath not starting with /")
	}
}

func TestNew_ConfigValidation_InvalidDashboardPath(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Invalid Path",
		DashboardPath: "dashboard",
	})
	if err == nil {
		t.Fatal("expected error for DashboardPath not starting with /")
	}
}

func TestNew_ConfigValidation_InvalidHealthPath(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:      "Invalid Path",
		HealthPath: "health",
	})
	if err == nil {
		t.Fatal("expected error for HealthPath not starting with /")
	}
}

// --- Root path rejection: "/" is reserved for the login page ---

func TestNew_ConfigValidation_RootPathRejected(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cfg  setup.Config
	}{
		{"AdminPath root", setup.Config{AdminPath: "/"}},
		{"DashboardPath root", setup.Config{DashboardPath: "/"}},
		{"HealthPath root", setup.Config{HealthPath: "/"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := setup.New(tc.cfg)
			if err == nil {
				t.Fatal("expected error — the site root is reserved for the login page")
			}
		})
	}
}

// --- Value range validation ---

func TestNew_ConfigValidation_NegativeSSEMaxReplayRejected(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{Title: "Negative Replay", SSEMaxReplay: -1})
	if err == nil {
		t.Fatal("expected error for negative SSEMaxReplay (0 = transport default, positive = cap)")
	}
}

// --- Path conflict validation ---

func TestNew_ConfigValidation_AdminAndDashboardPathsConflict(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Conflicting Paths",
		AdminPath:     "/panel/",
		DashboardPath: "/panel",
	})
	if err == nil {
		t.Fatal("expected error when AdminPath and DashboardPath resolve to the same route")
	}
}

func TestNew_ConfigValidation_HealthPathConflictsWithAdmin(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:      "Health Conflicts With Admin",
		AdminPath:  "/health/",
		HealthPath: "/health",
	})
	if err == nil {
		t.Fatal("expected error when HealthPath and AdminPath resolve to the same route")
	}
}

func TestNew_ConfigValidation_HealthPathConflictsWithDashboard(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Health Conflicts With Dashboard",
		DashboardPath: "/observe/",
		HealthPath:    "/observe",
	})
	if err == nil {
		t.Fatal("expected error when HealthPath and DashboardPath resolve to the same route")
	}
}

// --- LoginRedirect validation ---

func TestNew_ConfigValidation_InvalidLoginRedirect(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Invalid Redirect",
		LoginRedirect: "admin/dashboard",
	})
	if err == nil {
		t.Fatal("expected error for LoginRedirect not starting with / or http")
	}
}

func TestNew_ConfigValidation_ValidLoginRedirect_HTTPS(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:         "HTTPS Redirect",
		LoginRedirect: "https://example.com/dashboard",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()
}

// --- Cross-field validation: tenant mode requires TenantID ---

// TestNew_AdminTenantMode_RequiresTenantID covers the admin creation failure
// path: adminui.New rejects tenant mode without a TenantID, and setup must
// return a wrapped error (running cleanup on the already-created service).
func TestNew_AdminTenantMode_RequiresTenantID(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:     "Tenant Mode Without TenantID",
		AdminMode: adminui.ModeTenantAdmin,
	})
	if err == nil {
		t.Fatal("expected error for tenant admin mode without TenantID")
	}

	if !strings.Contains(err.Error(), "admin") {
		t.Fatalf("error should identify the admin panel as the failing component, got: %v", err)
	}
}

// --- MustNew panic on invalid config ---

func TestMustNew_PanicsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustNew should panic on invalid config")
		}
	}()

	_ = setup.MustNew(setup.Config{
		AdminPath: "invalid",
	})
}

// --- Observability is a flattened-path-only convenience ---

// testObservabilityBundleExternal builds a tracing-only OTel bundle for
// validation tests (structure only; no spans are recorded).
func testObservabilityBundleExternal(t *testing.T) *middleware.OTelBundle {
	t.Helper()

	bundle, err := middleware.NewOTelBundle(
		cqrsotel.NewTracer("setup-validation-test"), nil,
		middleware.WithMetricsDisabled(),
	)
	if err != nil {
		t.Fatalf("NewOTelBundle: %v", err)
	}

	return bundle
}

// newAdoptedServiceExternal constructs a service for adopt-mode conflict tests.
func newAdoptedServiceExternal(t *testing.T) *usermgmt.Service {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	return svc
}

func TestNew_ConfigValidation_ObservabilityWithAdoptedService(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Observability + Adopted Service",
		Service:       newAdoptedServiceExternal(t),
		Observability: testObservabilityBundleExternal(t),
	})
	if err == nil {
		t.Fatal("expected error for Observability alongside an adopted Service")
	}

	if !strings.Contains(err.Error(), "Observability") {
		t.Fatalf("error should name Observability as the conflict, got: %v", err)
	}
}

func TestNew_ConfigValidation_ObservabilityWithServiceConfigOverride(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Observability + ServiceConfig",
		ServiceConfig: &usermgmt.ServiceConfig{},
		Observability: testObservabilityBundleExternal(t),
	})
	if err == nil {
		t.Fatal("expected error for Observability alongside ServiceConfig")
	}

	if !strings.Contains(err.Error(), "Observability") {
		t.Fatalf("error should name Observability as the conflict, got: %v", err)
	}
}

// --- AdminExtraNav / AdminDashboardURL passthrough to adminui ---

// TestNew_AdminExtraNav_RejectsEmptyLabel proves the passthrough reaches
// adminui.New's validation: an invalid entry must fail bundle creation with
// the failing field named.
func TestNew_AdminExtraNav_RejectsEmptyLabel(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:         "Empty ExtraNav Label",
		AdminExtraNav: []adminui.NavEntry{{Href: "/support"}},
	})
	if err == nil {
		t.Fatal("expected error for AdminExtraNav entry with empty Label")
	}

	if !strings.Contains(err.Error(), "ExtraNav") {
		t.Fatalf("error should name ExtraNav as the failing field, got: %v", err)
	}
}

// TestNew_AdminExtraNavAndDashboardURL_Accepted proves valid values flow
// through without changing bundle construction.
func TestNew_AdminExtraNavAndDashboardURL_Accepted(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:             "Extra Nav OK",
		AdminExtraNav:     []adminui.NavEntry{{Label: "Support", Href: "/support"}},
		AdminDashboardURL: "/dashboard/",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = bundle.Close() }()
}

// --- DashboardExtraNav passthrough to dashboardui ---

// TestNew_DashboardExtraNav_ReachesDashboard proves the knob is wired: an
// entry set on setup.Config must land in the dashboard's own config, or a
// bundle-based consumer cannot put a sibling surface in the panel sidebar.
func TestNew_DashboardExtraNav_ReachesDashboard(t *testing.T) {
	t.Parallel()

	bundle := setup.MustNew(setup.Config{
		Title:             "Dashboard Extra Nav",
		DashboardExtraNav: []dashboardui.NavEntry{{Label: "Event Catalog", Href: "/event-catalog"}},
	})
	defer func() { _ = bundle.Close() }()

	got := bundle.Dashboard.Config().ExtraNav
	if len(got) != 1 {
		t.Fatalf("dashboard ExtraNav = %+v, want the single entry from setup.Config", got)
	}

	if got[0].Label != "Event Catalog" || got[0].Href != "/event-catalog" {
		t.Errorf("dashboard ExtraNav[0] = %+v, want Event Catalog at /event-catalog", got[0])
	}
}

// TestNew_DashboardExtraNav_RejectsEmptyHref proves the passthrough reaches
// dashboardui's validation rather than being silently dropped.
func TestNew_DashboardExtraNav_RejectsEmptyHref(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:             "Empty Dashboard ExtraNav Href",
		DashboardExtraNav: []dashboardui.NavEntry{{Label: "Broken"}},
	})
	if err == nil {
		t.Fatal("expected error for DashboardExtraNav entry with an empty Href")
	}

	if !strings.Contains(err.Error(), "ExtraNav") {
		t.Fatalf("error should name ExtraNav as the failing field, got: %v", err)
	}
}
