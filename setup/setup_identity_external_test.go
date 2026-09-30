package setup_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// ADR-0054: the two identity-external flags. DisableAuth drops the auth
// handler while the service stays (own-login-endpoint mode); DisableService
// builds the shell — no usermgmt.Service anywhere, the bundle is exactly the
// lifecycle, the readiness composition, and the consumer-provided stores.
// Every surface whose session gate would dereference the missing service is
// rejected at New (a half-mounted gated endpoint is a latent panic, not a
// feature).

var errShellCheckFailed = errors.New("shell check failed")

// newShellConfig returns a minimal valid shell config; tests mutate from it.
func newShellConfig() setup.Config {
	return setup.Config{
		Title:            "Shell",
		DisableService:   true,
		DisableAdmin:     true,
		DisableDashboard: true,
		DisableLogin:     true,
	}
}

// --- DisableAuth: no auth handler, service stays ---

func TestNew_DisableAuth_AuthNilServicePresent(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:        "Own Login",
		DisableAuth:  true,
		DisableLogin: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Auth != nil {
		t.Error("DisableAuth must leave Bundle.Auth nil — Mount registers no /auth/* routes")
	}

	if bundle.Service == nil {
		t.Error("DisableAuth must keep the service — the own-login mode mints sessions against it")
	}
}

func TestMount_DisableAuth_AuthRoutes404HealthServed(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(setup.Config{
		Title:        "Own Login",
		DisableAuth:  true,
		DisableLogin: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	for _, route := range []string{
		"/auth/register",
		"/auth/login",
		"/auth/logout",
		"/auth/me",
	} {
		resp, err := server.Client().Get(server.URL + route)
		if err != nil {
			t.Fatalf("GET %s: %v", route, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("DisableAuth must leave %s unmounted, got status %d, want 404", route, resp.StatusCode)
		}
	}

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("DisableAuth must keep the readiness endpoint served, got %d", resp.StatusCode)
	}
}

func TestDisableAuth_RejectsEnabledLogin(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{Title: "Dead Form", DisableAuth: true})
	if err == nil {
		t.Fatal("expected error for DisableAuth with the login page enabled — its form posts into the void")
	}

	if !strings.Contains(err.Error(), "DisableLogin") {
		t.Errorf("rejection must point at DisableLogin, got: %v", err)
	}
}

func TestDisableAuth_RejectsAuthHandlerConfig(t *testing.T) {
	t.Parallel()

	_, err := setup.New(setup.Config{
		Title:             "Nothing To Configure",
		DisableAuth:       true,
		DisableLogin:      true,
		AuthHandlerConfig: &usermgmt.HandlerConfig{},
	})
	if err == nil {
		t.Fatal("expected error for AuthHandlerConfig with DisableAuth — it configures endpoints that will not exist")
	}
}

// --- DisableService: the shell ---

func TestNew_ShellBundle_NilServiceNilAuthDefaultStores(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(newShellConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Service != nil {
		t.Error("shell bundle must have no usermgmt.Service")
	}

	if bundle.Auth != nil {
		t.Error("shell bundle must have no auth handler")
	}

	if bundle.Admin != nil || bundle.Dashboard != nil || bundle.Login != nil {
		t.Error("shell bundle must have no panels")
	}

	if bundle.Stores == nil {
		t.Fatal("shell bundle must expose Stores")
	}

	if bundle.Stores.EventStore == nil || bundle.Stores.EventBus == nil {
		t.Error("shell bundle must default both stores (memory store, watermill bus)")
	}
}

func TestNew_ShellBundle_CustomStoresRoundTrip(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := watermill.NewEventBus()

	cfg := newShellConfig()
	cfg.EventStore = store
	cfg.EventBus = bus

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Stores.EventStore != store {
		t.Error("shell bundle must expose the consumer's EventStore verbatim")
	}

	if bundle.Stores.EventBus != bus {
		t.Error("shell bundle must expose the consumer's EventBus verbatim")
	}
}

func TestMount_Shell_ProbeSurfacesAndNothingElse(t *testing.T) {
	t.Parallel()

	cfg := newShellConfig()
	cfg.LivePath = "/livez"

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	bundle.Mount(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Every identity and session-gated surface stays unmounted.
	for _, route := range []string{
		"/auth/register",
		"/auth/login",
		"/admin/",
		"/dashboard/",
		"/sse",
		"/debug",
	} {
		resp, err := server.Client().Get(server.URL + route)
		if err != nil {
			t.Fatalf("GET %s: %v", route, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("shell bundle must leave %s unmounted, got status %d, want 404", route, resp.StatusCode)
		}
	}

	// The shell's reason to exist: health serves, liveness serves.
	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("shell /health must serve 200 with no checks configured, got %d (body: %s)", resp.StatusCode, body)
	}

	resp, err = server.Client().Get(server.URL + "/livez")
	if err != nil {
		t.Fatalf("GET /livez: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("shell /livez must serve 200, got %d", resp.StatusCode)
	}
}

func TestMount_Shell_HealthConsumerChecksOnly(t *testing.T) {
	t.Parallel()

	cfg := newShellConfig()
	cfg.HealthChecks = []cqrshtmx.NamedCheck{
		cqrshtmx.NewNamedCheck("sqlite", func() error { return nil }),
		cqrshtmx.NewNamedCheck("blob-dir", func() error { return errShellCheckFailed }),
	}

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	mux := http.NewServeMux()
	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("a failing consumer check must flip /health to 503, got %d (body: %s)", resp.StatusCode, body)
	}

	for _, name := range []string{"sqlite", "blob-dir"} {
		if !strings.Contains(string(body), "\""+name+"\"") {
			t.Errorf("shell health body must report the consumer check %q, got: %s", name, body)
		}
	}

	for _, builtin := range []string{"projections", "sse-hub"} {
		if strings.Contains(string(body), "\""+builtin+"\"") {
			t.Errorf("shell health must not report the built-in %q — no service, no hub", builtin)
		}
	}
}

func TestClose_Shell_Idempotent(t *testing.T) {
	t.Parallel()

	bundle, err := setup.New(newShellConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// --- Shell validation rejections: every surface that would reference the
// missing service fails fast at New. ---

func TestDisableService_RejectsEnabledPanels(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		flag string
		mut  func(c setup.Config) setup.Config
	}{
		{"DisableAdmin", func(c setup.Config) setup.Config { c.DisableAdmin = false; return c }},
		{"DisableDashboard", func(c setup.Config) setup.Config { c.DisableDashboard = false; return c }},
		{"DisableLogin", func(c setup.Config) setup.Config { c.DisableLogin = false; return c }},
	} {
		_, err := setup.New(tc.mut(newShellConfig()))
		if err == nil {
			t.Errorf("%s=false must be rejected in shell mode", tc.flag)
		} else if !strings.Contains(err.Error(), tc.flag) {
			t.Errorf("%s=false rejection must name the flag, got: %v", tc.flag, err)
		}
	}
}

func TestDisableService_RejectsServiceConstructionFields(t *testing.T) {
	t.Parallel()

	cfg := newShellConfig()
	cfg.SessionTTL = 1

	_, err := setup.New(cfg)
	if err == nil || !strings.Contains(err.Error(), "SessionTTL") {
		t.Errorf("service-construction fields must be rejected in shell mode naming the field, got: %v", err)
	}
}

func TestDisableService_RejectsEventFeeds(t *testing.T) {
	t.Parallel()

	cfg := newShellConfig()
	cfg.SSEPath = "/sse"

	_, err := setup.New(cfg)
	if err == nil || !strings.Contains(err.Error(), "SSEPath") {
		t.Errorf("SSEPath must be rejected in shell mode (nil-service session gate), got: %v", err)
	}
}

func TestDisableService_RejectsMachineEndpoints(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		field string
		mut   func(c setup.Config) setup.Config
	}{
		{"EventCatalogPath", func(c setup.Config) setup.Config { c.EventCatalogPath = "/catalog"; return c }},
		{"ProjectionStatusPath", func(c setup.Config) setup.Config { c.ProjectionStatusPath = "/projections"; return c }},
		{"DebugPath", func(c setup.Config) setup.Config { c.DebugPath = "/debug"; return c }},
	} {
		_, err := setup.New(tc.mut(newShellConfig()))
		if err == nil || !strings.Contains(err.Error(), tc.field) {
			t.Errorf("%s must be rejected in shell mode, got: %v", tc.field, err)
		}
	}
}

func TestDisableService_RejectsServiceConfig(t *testing.T) {
	t.Parallel()

	cfg := newShellConfig()
	cfg.ServiceConfig = &usermgmt.ServiceConfig{}

	_, err := setup.New(cfg)
	if err == nil || !strings.Contains(err.Error(), "ServiceConfig") {
		t.Errorf("ServiceConfig must be rejected in shell mode, got: %v", err)
	}
}

func TestDisableService_RejectsAdoptedService(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService fixture: %v", err)
	}
	defer func() { _ = svc.Close() }()

	cfg := newShellConfig()
	cfg.Service = svc

	_, err = setup.New(cfg)
	if err == nil || !strings.Contains(err.Error(), "Service") {
		t.Errorf("an adopted service must be rejected in shell mode, got: %v", err)
	}
}

// TestDisableService_AcceptsStoreInputs pins the two exceptions: EventStore
// and EventBus are the shell's OWN inputs — the fields the service paths
// reject as conflicts are exactly the fields the shell consumes.
func TestDisableService_AcceptsStoreInputs(t *testing.T) {
	t.Parallel()

	cfg := newShellConfig()
	cfg.EventStore = memory.NewMemoryStore()
	cfg.EventBus = watermill.NewEventBus()

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("EventStore/EventBus are the shell's inputs and must be accepted: %v", err)
	}
	defer func() { _ = bundle.Close() }()
}

// --- NewShell: the prunable shell constructor ---

func TestNewShell_ForcesShellSemantics(t *testing.T) {
	t.Parallel()

	// The caller forgot DisableService — NewShell IS the shell constructor,
	// the flag is implied. Panels stay disabled (newShellConfig), so only
	// the flag itself is unset and validation must still pass.
	cfg := newShellConfig()
	cfg.DisableService = false

	bundle, err := setup.NewShell(cfg)
	if err != nil {
		t.Fatalf("NewShell: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	if bundle.Service != nil || bundle.Auth != nil {
		t.Error("NewShell must build no service and no auth handler")
	}
}

func TestNewShell_ValidationStillApplies(t *testing.T) {
	t.Parallel()

	// Shell rules hold: panels are rejected exactly as with
	// New(Config{DisableService: true}).
	_, err := setup.NewShell(setup.Config{Title: "Panels", DisableAdmin: false})
	if err == nil || !strings.Contains(err.Error(), "DisableAdmin") {
		t.Errorf("NewShell must apply the shell validation, got: %v", err)
	}
}

func TestNewShell_ParityWithNewDisableService(t *testing.T) {
	t.Parallel()

	viaFlag, err := setup.New(newShellConfig())
	if err != nil {
		t.Fatalf("New(DisableService): %v", err)
	}
	defer func() { _ = viaFlag.Close() }()

	viaCtor, err := setup.NewShell(newShellConfig())
	if err != nil {
		t.Fatalf("NewShell: %v", err)
	}
	defer func() { _ = viaCtor.Close() }()

	if (viaFlag.Service == nil) != (viaCtor.Service == nil) ||
		(viaFlag.Auth == nil) != (viaCtor.Auth == nil) ||
		(viaFlag.Stores == nil) != (viaCtor.Stores == nil) {
		t.Error("NewShell and New(DisableService) must build the same bundle shape")
	}
}
