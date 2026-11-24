package adminui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// ReadOnly refusal receipts: Config.ReadOnly skips the route table's
// write:true rows at REGISTRATION, so a mutating request finds no POST route
// and the mux's method catch-all answers 405 — the refusal is structural,
// not a handler check that can be forgotten or bypassed.

// readonlyPanel builds a ReadOnly super-admin panel behind middleware that
// authenticates admin as a super_admin platform role holder, so every probe
// exercises the ROUTE layer's refusal, not the guard's.
func readonlyPanel(t *testing.T) (http.Handler, *Handler) {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{AuditLog: usermgmt.NewAuditLog()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	admin := identitymodel.NewUser(identitymodel.SyntheticUserID("01HADMINUSER"), "admin@example.com", "")
	if err := svc.GrantRoles(t.Context(),
		identitymodel.ActorIDFromUser(admin.ID),
		identitymodel.NewTenantID(identitymodel.WildcardDomain),
		[]identitymodel.Role{identitymodel.RoleSuperAdmin},
	); err != nil {
		t.Fatalf("GrantRoles: %v", err)
	}

	panel, err := New(Config{Service: svc, ReadOnly: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return userInject(panel.Handler(), admin), panel
}

// userInject wraps next with a fixed-user context injection (the session
// middleware's shape).
func userInject(next http.Handler, user *identitymodel.User) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user != nil {
			r = r.WithContext(usermgmt.WithUser(r.Context(), user))
		}
		next.ServeHTTP(w, r)
	})
}

// TestReadOnly_WritesRefused probes every write row the manifest would
// otherwise mount (patterns asserted against the FULL panel's manifest, so
// the probe list cannot drift): each answers 405 to a fully privileged
// caller, because no write route exists to reach.
func TestReadOnly_WritesRefused(t *testing.T) {
	t.Parallel()

	// The write patterns are taken from a FULL panel's manifest (Write:true
	// rows), so a new write route is automatically added to this probe set.
	full := newManifestPanel(t, Config{})
	writePatterns := make([]string, 0)
	for _, route := range full.Routes() {
		if route.Write {
			writePatterns = append(writePatterns, concreteProbePath(route.Pattern))
		}
	}
	if len(writePatterns) == 0 {
		t.Fatal("full manifest shows no write rows — probe set is empty")
	}

	handler, _ := readonlyPanel(t)

	for _, path := range writePatterns {
		t.Run("POST "+path, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin"+path, nil))
			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("POST %s answered %d, want 405 (structural read-only refusal)", path, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "Method not allowed") {
				t.Errorf("POST %s body lost the method-not-allowed page", path)
			}
		})
	}
}

// TestReadOnly_SafePOSTStillMounted pins the refusal's precision: the search
// POST fallback twin is a SAFE read (write:false) and stays mounted in
// ReadOnly mode — a blanket POST block would be a different, wrong contract.
func TestReadOnly_SafePOSTStillMounted(t *testing.T) {
	t.Parallel()

	handler, _ := readonlyPanel(t)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/users/search", nil))
	if rec.Code == http.StatusMethodNotAllowed {
		t.Fatal("POST /users/search (safe search twin) must stay mounted in ReadOnly mode")
	}
}

// TestReadOnly_ManifestTruth pins that the ReadOnly manifest simply omits the
// write rows — same rows, same order, minus every Write:true entry — and that
// no row left in it claims to be a write.
func TestReadOnly_ManifestTruth(t *testing.T) {
	t.Parallel()

	full := newManifestPanel(t, Config{})
	ro := newManifestPanel(t, Config{ReadOnly: true})

	fullRows := full.Routes()
	want := make([]string, 0, len(fullRows))
	for _, route := range fullRows {
		if !route.Write {
			want = append(want, route.String()+" "+string(route.Posture)+" "+route.Panel)
		}
	}

	got := ro.Routes()
	lines := make([]string, 0, len(got))
	for _, route := range got {
		if route.Write {
			t.Errorf("%s: ReadOnly manifest carries a write row", route)
		}
		lines = append(lines, route.String()+" "+string(route.Posture)+" "+route.Panel)
	}

	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ReadOnly manifest is not the full manifest minus write rows:\n--- want ---\n%s\n--- got ---\n%s",
			strings.Join(want, "\n"), strings.Join(lines, "\n"))
	}
}

// TestReadOnly_NoticeRender pins the sidebar footer notice: a ReadOnly panel
// renders the "Read-only" chip on a page; the default panel does not.
func TestReadOnly_NoticeRender(t *testing.T) {
	t.Parallel()

	handler, _ := readonlyPanel(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard answered %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Read-only") {
		t.Error("ReadOnly panel page lost the sidebar read-only notice chip")
	}

	full := newTestPanel(t, nil)
	rec2 := httptest.NewRecorder()
	full.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	if strings.Contains(rec2.Body.String(), "Read-only") {
		t.Error("default panel renders a read-only notice it does not earn")
	}
}
