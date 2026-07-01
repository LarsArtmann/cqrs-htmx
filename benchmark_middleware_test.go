package cqrshtmx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

func BenchmarkRequestLogging(b *testing.B) {
	middleware := cqrshtmx.RequestLogging(nil, func(_ string) {})
	handler := middleware(okHandler())

	b.Run("DefaultFormatter", func(b *testing.B) {
		benchGET(b, handler, "/users")
	})

	b.Run("JSONFormatter", func(b *testing.B) {
		jsonMw := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(_ string) {})
		jsonHandler := jsonMw(okHandler())
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/users", nil)
			w := httptest.NewRecorder()
			jsonHandler.ServeHTTP(w, r)
		}
	})

	b.Run("WithContextIDs", func(b *testing.B) {
		uid := cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
		cid := cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/users", nil)
			ctx := cqrshtmx.WithUserID(r.Context(), uid)
			ctx = cqrshtmx.WithCorrelationID(ctx, cid)
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
		}
	})
}

func BenchmarkCSRFMiddleware(b *testing.B) {
	cfg := cqrshtmx.CSRFConfig{}
	middleware := cqrshtmx.CSRFMiddleware(cfg)
	handler := middleware(okHandler())

	b.Run("GET", func(b *testing.B) {
		benchGET(b, handler, "/")
	})

	b.Run("POST-ValidToken", func(b *testing.B) {
		// Obtain a masked token from context via GET
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest(http.MethodGet, "/", nil)
		var token string
		captureMw := cqrshtmx.CSRFMiddleware(cfg)
		captureHandler := captureMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token = cqrshtmx.CSRFTokenFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		captureHandler.ServeHTTP(w1, r1)

		// Get the cookie from the capture handler
		var cookie *http.Cookie
		for _, c := range w1.Result().Cookies() {
			if c.Name == "csrf_token" {
				cookie = c
				break
			}
		}

		b.ResetTimer()
		for range b.N {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
			r.Header.Set("X-CSRF-Token", token)
			r.Header.Set("Sec-Fetch-Site", "same-origin")
			if cookie != nil {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
		}
	})
}

func BenchmarkRateLimiterMiddleware(b *testing.B) {
	b.Run("Global", func(b *testing.B) {
		middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
			Limit:  10000,
			Window: time.Minute,
		})
		handler := middleware(okHandler())

		b.ResetTimer()
		benchGET(b, handler, "/")
	})

	b.Run("PerKey", func(b *testing.B) {
		middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
			Limit:        10000,
			Window:       time.Minute,
			KeyExtractor: func(_ *http.Request) string { return "client-1" },
		})
		handler := middleware(okHandler())

		b.ResetTimer()
		benchGET(b, handler, "/")
	})

	b.Run("RemoteAddr", func(b *testing.B) {
		middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
			Limit:        10000,
			Window:       time.Minute,
			KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
		})
		handler := middleware(okHandler())

		b.ResetTimer()
		for range b.N {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
		}
	})
}

func BenchmarkSecurityHeadersMiddleware(b *testing.B) {
	middleware := cqrshtmx.SecurityHeadersMiddleware(okHandler())

	for b.Loop() {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, r)
	}
}

func benchGET(b *testing.B, h http.Handler, path string) {
	b.Helper()
	for b.Loop() {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

// benchGETWithBody runs a GET-like loop that posts a JSON body to /path.
func benchGETWithBody(b *testing.B, h http.Handler, path string) {
	b.Helper()
	for b.Loop() {
		r := httptest.NewRequest(http.MethodGet, path, strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

func BenchmarkHealthHandler(b *testing.B) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
	benchGET(b, app.HealthHandler(), "/health")
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	benchGET(b, cqrshtmx.RecoveryMiddleware(okHandler()), "/")
}

func BenchmarkAppRecoveryMiddleware(b *testing.B) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
	benchGET(b, app.RecoverHandler()(okHandler()), "/")
}
