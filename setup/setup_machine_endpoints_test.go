package setup_test

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
)

// M6 of the 2026-09-17 gap-bundle plan: the opt-in machine endpoints
// (EventCatalogPath / ProjectionStatusPath / DebugPath) are session-gated JSON
// surfaces — 401 without an authenticated session, real payloads with one —
// while zero paths mount nothing (no surprise routes).

const (
	machineCatalogPath  = "/events/catalog"
	machineProjPath     = "/health/projections"
	machineDebugPath    = "/debug"
	machineRegisterBody = `{"email":"machine@example.com","display_name":"Machine"}`
)

func newMachineBundle(t *testing.T, cfg setup.Config) *httptest.Server {
	t.Helper()

	bundle, err := setup.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	mux := http.NewServeMux()
	bundle.Mount(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

// registerAndLogin creates a session via the real registration flow and
// returns the session cookie for authenticated requests.
func registerAndLogin(t *testing.T, server *httptest.Server) *http.Cookie {
	t.Helper()

	resp, err := server.Client().Post(
		server.URL+"/auth/register", "application/json", strings.NewReader(machineRegisterBody))
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("registration failed: %d %s", resp.StatusCode, body)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}

	t.Fatal("registration did not set the session cookie")

	return nil
}

// getWithSession performs an authenticated GET carrying the session cookie,
// following no redirects.
func getWithSession(t *testing.T, server *httptest.Server, path string, cookie *http.Cookie) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.AddCookie(cookie)

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}

	return resp
}

func TestMount_MachineEndpoints_Unauthenticated401(t *testing.T) {
	t.Parallel()

	server := newMachineBundle(t, setup.Config{
		Title:                "Machine 401",
		EventCatalogPath:     machineCatalogPath,
		ProjectionStatusPath: machineProjPath,
		DebugPath:            machineDebugPath,
	})

	for _, path := range []string{machineCatalogPath, machineProjPath, machineDebugPath} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET %s unauthenticated = %d, want 401 (event metadata is not public)", path, resp.StatusCode)
		}
	}
}

func TestMount_MachineEndpoints_AuthenticatedServeJSON(t *testing.T) {
	t.Parallel()

	server := newMachineBundle(t, setup.Config{
		Title:                "Machine JSON",
		Version:              "v9.9.9-machine",
		EventCatalogPath:     machineCatalogPath,
		ProjectionStatusPath: machineProjPath,
		DebugPath:            machineDebugPath,
	})

	client := registerAndLogin(t, server)

	getBody := func(path string) (int, string, string) {
		t.Helper()

		resp := getWithSession(t, server, path, client)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		return resp.StatusCode, resp.Header.Get("Content-Type"), string(body)
	}

	status, contentType, body := getBody(machineCatalogPath)
	if status != http.StatusOK || !strings.Contains(contentType, "application/json") {
		t.Fatalf("catalog: status=%d content-type=%q, want 200 JSON", status, contentType)
	}
	if !strings.Contains(body, `"UserRegistered"`) {
		t.Errorf("catalog must list the domain events, missing UserRegistered in %.200s", body)
	}

	status, contentType, body = getBody(machineProjPath)
	if status != http.StatusOK || !strings.Contains(contentType, "application/json") {
		t.Fatalf("projection status: status=%d content-type=%q, want 200 JSON", status, contentType)
	}
	if !strings.Contains(body, "user-read-model") {
		t.Errorf("projection status must list the read models, missing user-read-model in %.200s", body)
	}

	status, contentType, body = getBody(machineDebugPath)
	if status != http.StatusOK || !strings.Contains(contentType, "application/json") {
		t.Fatalf("debug: status=%d content-type=%q, want 200 JSON", status, contentType)
	}

	var debugPayload struct {
		Version   string `json:"version"`
		GoVersion string `json:"goVersion"`
		Title     string `json:"title"`
	}
	if err := json.Unmarshal([]byte(body), &debugPayload); err != nil {
		t.Fatalf("debug payload must be JSON: %v (%.100s)", err, body)
	}

	if debugPayload.Version != "v9.9.9-machine" {
		t.Errorf("debug version = %q, want the Config.Version stamp", debugPayload.Version)
	}

	if !strings.HasPrefix(debugPayload.GoVersion, "go1") {
		t.Errorf("debug goVersion = %q, want a go runtime version", debugPayload.GoVersion)
	}
}

func TestMount_MachineEndpoints_ZeroPathsMountNothing(t *testing.T) {
	t.Parallel()

	server := newMachineBundle(t, setup.Config{Title: "No Machine Routes"})

	for _, path := range []string{machineCatalogPath, machineProjPath, machineDebugPath} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
			t.Errorf("GET %s must not serve JSON without a configured path", path)
		}
	}
}

func TestNew_MachineEndpoints_PathValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cfg  setup.Config
		want string
	}{
		{
			name: "missing leading slash rejected",
			cfg:  setup.Config{Title: "Bad Shape", EventCatalogPath: "events"},
			want: "must start with",
		},
		{
			name: "root path rejected",
			cfg:  setup.Config{Title: "Root", DebugPath: "/"},
			want: "reserved for the login page",
		},
		{
			name: "collision with health rejected",
			cfg: setup.Config{
				Title:                "Collision",
				ProjectionStatusPath: "/health",
			},
			want: "routes would conflict",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := setup.New(tc.cfg)
			if err == nil {
				t.Fatalf("expected rejection, got nil")
			}

			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("rejection must mention %q, got: %v", tc.want, err)
			}
		})
	}
}
