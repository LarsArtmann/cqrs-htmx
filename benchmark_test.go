package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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
	registerGetUserEmail(disp)

	app, _ := cqrshtmx.New(cqrshtmx.Config{Queries: disp})
	handler := app.Query(
		"GetUser",
		decodeGetUserJSONQuery(),
		cqrshtmx.Render(encodeJSONResult),
	)
	benchGETWithBody(b, handler, "/user")
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
	benchGET(b, app.HealthHandler(), "/health")
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	benchGET(b, cqrshtmx.RecoveryMiddleware(okHandler()), "/")
}

func BenchmarkAppRecoveryMiddleware(b *testing.B) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: command.NewDispatcher()})
	benchGET(b, app.RecoverHandler()(okHandler()), "/")
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
	benchGETWithBody(b, handler, "/user")
}

func BenchmarkDecodePagination(b *testing.B) {
	benchCases := []struct {
		name string
		url  string
	}{
		{"no_params", "/items"},
		{"with_page", "/items?page=5"},
		{"with_size", "/items?page_size=50"},
		{"with_both", "/items?page=3&page_size=25"},
		{"with_extras", "/items?page=3&page_size=25&filter=active&sort=name&order=asc"},
		{"invalid_page", "/items?page=abc"},
		{"zero_page", "/items?page=0"},
		{"huge_size", "/items?page_size=10000"},
	}
	for _, bc := range benchCases {
		b.Run(bc.name, func(b *testing.B) {
			r := httptest.NewRequest(http.MethodGet, bc.url, nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cqrshtmx.DecodePagination(r)
			}
		})
	}
}

// benchmarkCreateUserCmd is a typed command used by the RegisterTyped
// benchmarks. It satisfies command.Command.
type benchmarkCreateUserCmd struct {
	*command.BasicCommand
}

func (c *benchmarkCreateUserCmd) Email() string { return "bench@example.com" }

func newBenchmarkCreateUserCmd() *benchmarkCreateUserCmd {
	core, err := command.New("CreateUser", id.NewAggregateID())
	if err != nil {
		panic(err)
	}
	return &benchmarkCreateUserCmd{BasicCommand: core}
}

func BenchmarkCommandRegisterTypedVsRegister(b *testing.B) {
	b.Run("RegisterTyped", func(b *testing.B) {
		disp := command.NewDispatcher()
		if err := command.RegisterTyped(
			disp, "CreateUser",
			func(_ context.Context, _ *benchmarkCreateUserCmd) error { return nil },
		); err != nil {
			b.Fatal(err)
		}
		cmd := newBenchmarkCreateUserCmd()
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = disp.Dispatch(ctx, cmd)
		}
	})

	b.Run("Register_with_type_assertion", func(b *testing.B) {
		disp := command.NewDispatcher()
		_ = disp.Register("CreateUser", func(_ context.Context, c command.Command) error {
			_, _ = c.(*benchmarkCreateUserCmd)
			return nil
		})
		cmd := newBenchmarkCreateUserCmd()
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = disp.Dispatch(ctx, cmd)
		}
	})
}

func BenchmarkQueryDispatchTypedVsDispatch(b *testing.B) {
	b.Run("DispatchTyped", func(b *testing.B) {
		disp := query.NewDispatcher()
		_ = query.RegisterTyped(
			disp, "ListUsers",
			func(_ context.Context, _ *testListUsersQuery) ([]string, error) {
				return []string{"a", "b"}, nil
			},
		)
		q := newTestListUsersQuery()
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = query.DispatchTyped[[]string](ctx, disp, q)
		}
	})

	b.Run("Dispatch_with_type_assertion", func(b *testing.B) {
		disp := query.NewDispatcher()
		_ = disp.Register("ListUsers", func(_ context.Context, _ query.Query) (any, error) {
			return []string{"a", "b"}, nil
		})
		q := newTestListUsersQuery()
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = disp.Dispatch(ctx, q)
		}
	})
}
