package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func BenchmarkCommandDispatch(b *testing.B) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", noOpCommandHandler)

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
		for range b.N {
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
		for range b.N {
			_ = disp.Dispatch(ctx, cmd)
		}
	})
}

func BenchmarkQueryDispatchTypedVsDispatch(b *testing.B) {
	b.Run("DispatchTyped", func(b *testing.B) {
		disp := query.NewDispatcher()
		_ = registerListUsersQuery(disp, []string{"a", "b"})
		q := newTestListUsersQuery()
		ctx := context.Background()
		b.ResetTimer()
		for range b.N {
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
		for range b.N {
			_, _ = disp.Dispatch(ctx, q)
		}
	})
}
