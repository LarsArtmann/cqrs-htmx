package cqrshtmx_test

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/gomega"
)

// --- Request/response helpers ---

type testResponse struct {
	*httptest.ResponseRecorder
}

func newPostJSONRequest(body string, opts ...func(*http.Request)) *http.Request {
	r := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/users", strings.NewReader(body),
	)
	r.Header.Set("Content-Type", "application/json")
	for _, o := range opts {
		o(r)
	}
	return r
}

func newPostRequest(path, body string, opts ...func(*http.Request)) *http.Request {
	r := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, path, strings.NewReader(body),
	)
	for _, o := range opts {
		o(r)
	}
	return r
}

func withHTMX(r *http.Request) {
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
}

func withUserHeader(header string, uid cqrshtmx.UserID) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(header, uid.String()) }
}

func withHeader(key, value string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(key, value) }
}

func serve(handler http.Handler, r *http.Request) *testResponse {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return &testResponse{w}
}

func (r *testResponse) code() int { return r.Code }

func assertStatusCode(handler http.Handler, r *http.Request, expected int) {
	w := serve(handler, r)
	ExpectWithOffset(1, w.code()).To(Equal(expected))
}

func assertHTMXErrorRedirect(err error, loginRedirect, expectedRedirect string) {
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	cqrshtmx.DefaultErrorHandlerWithRedirect(w, r, err, loginRedirect)
	ExpectWithOffset(1, w.Header().Get("HX-Redirect")).To(Equal(expectedRedirect))
}

// --- HTTP interface mock helpers ---

type mockPusher struct {
	http.ResponseWriter
	pushFunc func(target string, opts *http.PushOptions) error
}

func (m *mockPusher) Push(target string, opts *http.PushOptions) error {
	return m.pushFunc(target, opts)
}

func newPusherRecorder(p *mockPusher) *pusherRecorder {
	return &pusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		pusher:           p,
	}
}

type pusherRecorder struct {
	*httptest.ResponseRecorder
	pusher *mockPusher
}

func (r *pusherRecorder) Push(target string, opts *http.PushOptions) error {
	return r.pusher.Push(target, opts)
}

type hijackRecorder struct {
	*httptest.ResponseRecorder
}

func newHijackRecorder() *hijackRecorder {
	return &hijackRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (h *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

type errorEnforcer struct {
	err error
}

func (e *errorEnforcer) Enforce(...any) (bool, error) {
	return false, e.err
}

func newFailingEnforcer(err error) cqrshtmx.Enforcer {
	return &errorEnforcer{err: err}
}
