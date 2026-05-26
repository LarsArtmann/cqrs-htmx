package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func BenchmarkMapError(b *testing.B) {
	b.Run("Unauthorized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cqrshtmx.MapError(cqrshtmx.ErrUnauthorized)
		}
	})
	b.Run("Forbidden", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cqrshtmx.MapError(cqrshtmx.ErrForbidden)
		}
	})
	b.Run("Rejection", func(b *testing.B) {
		err := event.NewRejection("test.rejection", "rejected")
		for i := 0; i < b.N; i++ {
			cqrshtmx.MapError(err)
		}
	})
	b.Run("Conflict", func(b *testing.B) {
		err := event.NewConflict("test.conflict", "conflict")
		for i := 0; i < b.N; i++ {
			cqrshtmx.MapError(err)
		}
	})
	b.Run("Transient", func(b *testing.B) {
		err := event.NewTransient("test.transient", "transient")
		for i := 0; i < b.N; i++ {
			cqrshtmx.MapError(err)
		}
	})
	b.Run("Nil", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cqrshtmx.MapError(nil)
		}
	})
}

func BenchmarkParseHTMXRequest(b *testing.B) {
	handler := cqrshtmx.HTMXMiddleware(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}),
	)

	b.Run("AllHeaders", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("HX-Request", "true")
			r.Header.Set("HX-Boosted", "true")
			r.Header.Set("HX-Target", "main")
			r.Header.Set("HX-Trigger", "btn")
			r.Header.Set("HX-Trigger-Name", "action")
			r.Header.Set("HX-Prompt", "yes")
			r.Header.Set("HX-Current-URL", "https://example.com/page")
			r.Header.Set("HX-History-Restore-Request", "true")
			handler.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
	b.Run("NoHeaders", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(httptest.NewRecorder(), r)
		}
	})
}

func BenchmarkCommandDispatch(b *testing.B) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", func(_ context.Context, _ command.Command) error {
		return nil
	})

	app, _ := cqrshtmx.New(cqrshtmx.Config{Commands: disp})
	handler := app.Command("CreateUser", decodeCreateUserJSON())

	for b.Loop() {
		r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkQueryDispatch(b *testing.B) {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return map[string]string{testEmailKey: testEmailValue}, nil
	})

	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	handler := app.Query(
		"GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.Render(encodeJSONResult),
	)

	for b.Loop() {
		r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkRequestLogging(b *testing.B) {
	middleware := cqrshtmx.RequestLogging(nil, func(_ string) {})
	handler := middleware(okHandler())

	b.Run("DefaultFormatter", func(b *testing.B) {
		benchGET(b, handler, "/users")
	})

	b.Run("JSONFormatter", func(b *testing.B) {
		jsonMw := cqrshtmx.RequestLogging(cqrshtmx.JSONLogFormatter, func(_ string) {})
		jsonHandler := jsonMw(okHandler())
		for i := 0; i < b.N; i++ {
			r := httptest.NewRequest(http.MethodGet, "/users", nil)
			w := httptest.NewRecorder()
			jsonHandler.ServeHTTP(w, r)
		}
	})

	b.Run("WithContextIDs", func(b *testing.B) {
		uid := cqrshtmx.MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
		cid := cqrshtmx.MustParseCorrelationID("01HK1549P84T9XF8R94E960633")
		for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
		}
	})

	b.Run("PerKey", func(b *testing.B) {
		middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
			Limit:        10000,
			Window:       time.Minute,
			KeyExtractor: func(_ *http.Request) string { return "client-1" },
		})
		handler := middleware(okHandler())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
		}
	})

	b.Run("RemoteAddr", func(b *testing.B) {
		middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
			Limit:        10000,
			Window:       time.Minute,
			KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
		})
		handler := middleware(okHandler())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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
	for i := 0; i < b.N; i++ {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
	}
}

func BenchmarkResponseJSON(b *testing.B) {
	data := map[string]string{"s": "ok", "message": "hello world"}
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		cqrshtmx.NewResponse(w, r).JSON(data)
	}
}

func BenchmarkResponseWriteString(b *testing.B) {
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		cqrshtmx.NewResponse(w, r).WriteString("hello world")
	}
}

func BenchmarkResponseBody(b *testing.B) {
	body := []byte("hello world")
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		cqrshtmx.NewResponse(w, r).Body(body)
	}
}

func BenchmarkHealthHandler(b *testing.B) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
	handler := app.HealthHandler()
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	handler := cqrshtmx.RecoveryMiddleware(okHandler())
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkAppRecoveryMiddleware(b *testing.B) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
	handler := app.RecoveryMiddleware()(okHandler())
	for b.Loop() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)
	}
}

func BenchmarkRenderJSON(b *testing.B) {
	disp := query.NewDispatcher()
	_ = disp.Register("GetUser", func(_ context.Context, _ query.Query) (any, error) {
		return map[string]string{testNameKey: testNameKey}, nil
	})
	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	handler := app.Query(
		"GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.RenderJSON[map[string]string](),
	)
	for b.Loop() {
		r := httptest.NewRequest(http.MethodGet, "/user", strings.NewReader(`{}`))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
