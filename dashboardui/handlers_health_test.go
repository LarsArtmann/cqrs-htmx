package dashboardui

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestHealthz_ReturnsOK(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/healthz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d; want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("healthz Content-Type = %q; want application/json", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("healthz body unmarshal: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("healthz status = %q; want %q", body["status"], "ok")
	}
}

func TestReadyz_ReadyWhenConfigured(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/readyz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("readyz status = %d; want 200; body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("readyz body unmarshal: %v", err)
	}

	if body["ready"] != true {
		t.Errorf("readyz ready = %v; want true", body["ready"])
	}
}

func TestReadyz_NotReadyAfterClose(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	d.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/readyz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz after close status = %d; want 503", rec.Code)
	}
}

func TestVersionz_ReturnsCapabilities(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	d, _ := New(Config{EventSource: store, Journal: store, ReadOnly: true})
	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/-/versionz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("versionz status = %d; want 200; body=%s", rec.Code, rec.Body.String())
	}

	var body versionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("versionz body unmarshal: %v", err)
	}

	if body.Module != modulePath {
		t.Errorf("versionz module = %q; want %q", body.Module, modulePath)
	}

	if !body.Capabilities.EventSource {
		t.Error("versionz capabilities should report EventSource=true")
	}

	if !body.ReadOnly {
		t.Error("versionz readOnly should be true")
	}
}
