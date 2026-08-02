package datastar_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

func TestScriptHandlerReturnsJS(t *testing.T) {
	t.Parallel()

	handler := ds.ScriptHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/javascript; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotEmpty(t, body, "embedded datastar.js should not be empty")
	require.Contains(t, string(body), "Datastar", "JS body should contain Datastar identifier")
}

func TestScriptHandlerETag(t *testing.T) {
	t.Parallel()

	handler := ds.ScriptHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()

	etag := resp.Header.Get("ETag")
	require.Equal(t, `"datastar-1.0.2"`, etag)
}

func TestScriptHandlerIfNoneMatchReturns304(t *testing.T) {
	t.Parallel()

	handler := ds.ScriptHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	etag := `"datastar-1.0.2"`

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("If-None-Match", etag)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNotModified, resp.StatusCode)
}

func TestScriptHandlerCacheControl(t *testing.T) {
	t.Parallel()

	handler := ds.ScriptHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.Equal(t, "public, max-age=31536000, immutable", resp.Header.Get("Cache-Control"))
}

func TestScriptHandlerRejectsPOST(t *testing.T) {
	t.Parallel()

	handler := ds.ScriptHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Post(server.URL, "text/plain", nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestScriptHandlerWithCustomJS(t *testing.T) {
	t.Parallel()

	customJS := []byte("// custom datastar build")
	handler := ds.ScriptHandlerWith(customJS, "2.0.0")

	req := httptest.NewRequest(http.MethodGet, "/datastar.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, `"datastar-2.0.0"`, w.Header().Get("ETag"))

	body, err := io.ReadAll(w.Body)
	require.NoError(t, err)
	require.Equal(t, customJS, body)
}

func TestScriptTag(t *testing.T) {
	t.Parallel()

	tag := ds.ScriptTag("/static/datastar.js")
	require.Equal(t, `<script type="module" src="/static/datastar.js"></script>`, tag)
}

func TestVersion(t *testing.T) {
	t.Parallel()

	require.Equal(t, "1.0.2", ds.Version())
}
