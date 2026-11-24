package adminui

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestPanel_AssetsServeCSS(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/admin-tw.css", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("CSS asset: status %d, want 200", rec.Code)
	}
}

func TestPanel_AssetsServeJS(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/admin.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("JS asset: status %d, want 200", rec.Code)
	}
}

func TestPanel_AssetsServeSyncWorker(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/sync-worker.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("sync-worker asset: status %d, want 200", rec.Code)
	}
}

func TestPanel_AssetsServeSyncClient(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/sync-client.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("sync-client asset: status %d, want 200", rec.Code)
	}
}

// TestPanel_SyncWorkerDelegationMatchesRoot verifies that adminui's delegation
// to the root SyncWorkerHandler serves byte-identical content. This catches
// drift if someone accidentally replaces the delegation with a stale embedded copy.
func TestPanel_SyncWorkerDelegationMatchesRoot(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	// Get bytes from adminui panel route
	panelRec := httptest.NewRecorder()
	h.ServeHTTP(panelRec, httptest.NewRequest(http.MethodGet, "/admin/-/sync-worker.js", nil))

	// Get bytes from root handler directly
	rootRec := httptest.NewRecorder()
	cqrshtmx.SyncWorkerHandler().ServeHTTP(rootRec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !bytes.Equal(panelRec.Body.Bytes(), rootRec.Body.Bytes()) {
		t.Errorf(
			"sync-worker.js delegation mismatch: adminui %d bytes vs root %d bytes",
			panelRec.Body.Len(), rootRec.Body.Len(),
		)
	}
}

// TestPanel_SyncClientDelegationMatchesRoot verifies that adminui's delegation
// to the root SyncClientHandler serves byte-identical content.
func TestPanel_SyncClientDelegationMatchesRoot(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	// Get bytes from adminui panel route
	panelRec := httptest.NewRecorder()
	h.ServeHTTP(panelRec, httptest.NewRequest(http.MethodGet, "/admin/-/sync-client.js", nil))

	// Get bytes from root handler directly
	rootRec := httptest.NewRecorder()
	cqrshtmx.SyncClientHandler().ServeHTTP(rootRec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !bytes.Equal(panelRec.Body.Bytes(), rootRec.Body.Bytes()) {
		t.Errorf(
			"sync-client.js delegation mismatch: adminui %d bytes vs root %d bytes",
			panelRec.Body.Len(), rootRec.Body.Len(),
		)
	}
}

func TestPanel_AssetsNotFound(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/nonexistent.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown asset: status %d, want 404", rec.Code)
	}
}

func TestPanel_HTMXScriptHandler(t *testing.T) {
	user := mustUser(t, "admin@example.com")
	h, _ := newTestPanel(t, user)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/-/htmx.js", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("htmx.js: status %d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header")
	}
}

func TestAssetHandler_NotFound(t *testing.T) {
	t.Parallel()
	h, err := newAssetHandler(assetsFS, "admin.js", "application/javascript")
	if err != nil {
		t.Fatalf("newAssetHandler: %v", err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	// newAssetHandler embeds the file — it should serve it successfully
	if rec.Code != http.StatusOK {
		t.Errorf("newAssetHandler: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "application/javascript" {
		t.Errorf("newAssetHandler Content-Type = %q, want application/javascript", rec.Header().Get("Content-Type"))
	}
}
