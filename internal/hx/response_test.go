package hx_test

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
)

func TestWriteAll(t *testing.T) {
	w := httptest.NewRecorder()
	hx.WriteAll(w, []byte("hello"))

	if got := w.Body.String(); got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}
}

func TestWriteAllString(t *testing.T) {
	t.Run("string writer path", func(t *testing.T) {
		w := httptest.NewRecorder()
		hx.WriteAllString(w, "hello")

		if got := w.Body.String(); got != "hello" {
			t.Fatalf("body = %q, want %q", got, "hello")
		}
	})

	t.Run("plain writer fallback", func(t *testing.T) {
		buf := &bytesOnlyWriter{}
		hx.WriteAllString(buf, "hello")

		if got := string(buf.data); got != "hello" {
			t.Fatalf("body = %q, want %q", got, "hello")
		}
	})
}

// bytesOnlyWriter implements io.Writer but not io.StringWriter, forcing the
// []byte conversion path in WriteAllString.
type bytesOnlyWriter struct{ data []byte }

func (b *bytesOnlyWriter) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)

	return len(p), nil
}

func TestMarshalJSONForResponse(t *testing.T) {
	t.Run("marshals the payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		hx.MarshalJSONForResponse(w, map[string]string{"status": "ok"})

		var got map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if got["status"] != "ok" {
			t.Fatalf("status = %q, want ok", got["status"])
		}
	})

	t.Run("falls back when the payload cannot be marshalled", func(t *testing.T) {
		w := httptest.NewRecorder()
		hx.MarshalJSONForResponse(w, make(chan int))

		if got := w.Body.String(); got != `{"error":"failed to serialize response"}` {
			t.Fatalf("body = %q", got)
		}
	})
}

func TestMarshalJSONOrFallback(t *testing.T) {
	if got := hx.MarshalJSONOrFallback(map[string]int{"n": 1}, "fallback"); got != `{"n":1}` {
		t.Fatalf("got %q", got)
	}

	if got := hx.MarshalJSONOrFallback(make(chan int), "fallback"); got != "fallback" {
		t.Fatalf("got %q, want fallback", got)
	}
}

func TestHashTag(t *testing.T) {
	if got := hx.HashTag([]byte("abc")); got != hx.HashTag([]byte("abc")) {
		t.Fatalf("HashTag is not deterministic: %q", got)
	}

	if hx.HashTag([]byte("abc")) == hx.HashTag([]byte("abd")) {
		t.Fatal("HashTag must change when the content changes")
	}
}

func TestServeImmutableJSON(t *testing.T) {
	data := []byte(`{"a":1}`)
	tag := `"` + hx.HashTag(data) + `"`

	serve := func(ifNoneMatch string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()

		r := httptest.NewRequest(http.MethodGet, "/spec.json", nil)
		if ifNoneMatch != "" {
			r.Header.Set("If-None-Match", ifNoneMatch)
		}

		hx.ServeImmutableJSON(w, r, tag, data)

		return w
	}

	t.Run("serves the body with revalidating cache headers", func(t *testing.T) {
		w := serve("")

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}

		if got := w.Header().Get("Content-Type"); got != hx.ContentTypeJSON {
			t.Fatalf("content type = %q", got)
		}

		if got := w.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("cache control = %q", got)
		}

		if got := w.Header().Get("ETag"); got != tag {
			t.Fatalf("etag = %q, want %q", got, tag)
		}

		if w.Body.String() != string(data) {
			t.Fatalf("body = %q", w.Body.String())
		}
	})

	t.Run("returns 304 for a matching If-None-Match", func(t *testing.T) {
		w := serve(tag)

		if w.Code != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", w.Code)
		}

		if w.Body.Len() != 0 {
			t.Fatalf("body = %q, want empty", w.Body.String())
		}
	})
}

func TestNewImmutableJSONHandler(t *testing.T) {
	data := []byte(`{"b":2}`)
	w := httptest.NewRecorder()
	hx.NewImmutableJSONHandler(data).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK || w.Body.String() != string(data) {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}

func TestServeJS(t *testing.T) {
	js := []byte("// js")
	tag := `"app-1"`
	handler := hx.ServeJS(js, tag)

	t.Run("GET serves the body", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/app.js", nil))

		if w.Code != http.StatusOK || w.Body.String() != string(js) {
			t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
		}

		if got := w.Header().Get("Content-Type"); got != "text/javascript; charset=utf-8" {
			t.Fatalf("content type = %q", got)
		}
	})

	t.Run("HEAD is allowed", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/app.js", nil))

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	})

	t.Run("POST is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/app.js", nil))

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", w.Code)
		}
	})

	t.Run("matching If-None-Match yields 304", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		r.Header.Set("If-None-Match", tag)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusNotModified {
			t.Fatalf("status = %d, want 304", w.Code)
		}
	})
}

func TestRequestIDContext(t *testing.T) {
	if !hx.RequestIDFromContext(t.Context()).IsZero() {
		t.Fatal("expected zero request ID for an empty context")
	}

	id := hx.NewRequestID()
	ctx := hx.WithRequestID(t.Context(), id)

	if got := hx.RequestIDFromContext(ctx); got != id {
		t.Fatalf("request ID = %q, want %q", got, id)
	}
}

func TestIsPanicError(t *testing.T) {
	if hx.IsPanicError(nil) {
		t.Fatal("nil is not a panic error")
	}

	if hx.IsPanicError(errors.New("boom")) {
		t.Fatal("a plain error is not a panic error")
	}
}
