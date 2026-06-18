package cataloghtmx_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cataloghtmx "github.com/larsartmann/cqrs-htmx/catalog/v2"
)

func unmarshalJSONBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return doc
}

func tryUnmarshalJSONBody(w *httptest.ResponseRecorder) map[string]any {
	var doc map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &doc)
	return doc
}

func setupTestCatalog() *cataloghtmx.Builder {
	return buildTestCatalog("Test API", "test-svc")
}

// Test

func TestOpenAPIHandler_StatusAndContentType(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestOpenAPIHandler_ValidJSON(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	doc := unmarshalJSONBody(t, w)

	if doc["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", doc["openapi"])
	}
}

func TestOpenAPIHandler_ContainsPaths(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	doc := tryUnmarshalJSONBody(w)

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths in OpenAPI document")
	}

	if len(paths) == 0 {
		t.Error("expected at least one path")
	}
}

func TestOpenAPIHandler_YAML(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(cat, cataloghtmx.WithFormat(cataloghtmx.FormatYAML))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/yaml") {
		t.Errorf("expected application/yaml, got %q", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi:") {
		t.Error("expected YAML to contain 'openapi:' key")
	}
}

func TestAsyncAPIHandler_StatusAndContentType(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.AsyncAPIHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestAsyncAPIHandler_ValidJSON(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.AsyncAPIHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	doc := unmarshalJSONBody(t, w)

	if doc["asyncapi"] != "3.0.0" {
		t.Errorf("expected asyncapi 3.0.0, got %v", doc["asyncapi"])
	}
}

func TestD2Handler_StatusAndContentType(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.D2Handler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected text/plain, got %q", ct)
	}
}

func TestD2Handler_ContainsService(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.D2Handler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "test_svc") && !strings.Contains(body, "Test API") {
		t.Error("expected D2 diagram to contain service identifier")
	}
}

func TestHealthCheckHandler_Healthy(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.HealthCheckHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "healthy") {
		t.Errorf("expected 'healthy' in body, got %s", body)
	}
}

func TestHealthCheckHandler_NilCatalog(t *testing.T) {
	t.Parallel()

	handler := cataloghtmx.HealthCheckHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestGenerateEventCatalog(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()

	dir := t.TempDir()
	if err := cataloghtmx.GenerateEventCatalog(cat, dir); err != nil {
		t.Fatalf("GenerateEventCatalog failed: %v", err)
	}
}

func TestWithBasePath(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(
		cat,
		cataloghtmx.WithBasePath("/v2/api"),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	doc := tryUnmarshalJSONBody(w)

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected paths in OpenAPI document")
	}

	for path := range paths {
		if !strings.HasPrefix(path, "/v2/api") {
			t.Errorf("expected path to start with /v2/api, got %s", path)
		}
	}
}

func TestWithDescription(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.OpenAPIHandler(
		cat,
		cataloghtmx.WithDescription("My awesome API"),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	doc := unmarshalJSONBody(t, w)

	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("expected info section in OpenAPI document")
	}

	if info["description"] != "My awesome API" {
		t.Errorf("expected description 'My awesome API', got %v", info["description"])
	}
}

func TestHealthCheckHandler_HealthyBodyContent(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()
	handler := cataloghtmx.HealthCheckHandler(cat)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for catalog with services, got %d", w.Code)
	}

	body := unmarshalJSONBody(t, w)

	if body["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", body["status"])
	}

	services, ok := body["services"].(float64)
	if !ok {
		t.Fatalf("expected 'services' count in body, got %T", body["services"])
	}

	if services < 1 {
		t.Errorf("expected at least 1 service, got %v", services)
	}
}

func TestHealthCheckHandler_NilCatalogBody(t *testing.T) {
	t.Parallel()

	handler := cataloghtmx.HealthCheckHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	body := unmarshalJSONBody(t, w)

	if body["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %v", body["status"])
	}

	if body["message"] != "catalog has no services" {
		t.Errorf("expected message 'catalog has no services', got %v", body["message"])
	}
}

func TestGenerateEventCatalog_BadOutputDir(t *testing.T) {
	t.Parallel()

	cat := setupTestCatalog().Build()

	// A path under /proc should be unwritable on Linux, triggering the error path.
	err := cataloghtmx.GenerateEventCatalog(cat, "/proc/cannot-write-here")
	if err == nil {
		t.Fatal("expected error writing to unwritable directory")
	}

	if !strings.Contains(err.Error(), "failed to generate EventCatalog files") {
		t.Errorf("expected wrapped error message, got %v", err)
	}
}
