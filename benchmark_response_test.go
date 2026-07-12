package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

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
			for range b.N {
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
