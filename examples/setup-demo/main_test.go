package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-sse"
)

// TestDemoApp_EndToEnd boots the same composition as main() (minus the real
// listener) and walks the full auth flow: public routes answer, protected
// routes gate on the session, and the dev-login cookie unlocks both panels.
func TestDemoApp_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Same composition as main(): ServiceConfig escape hatch + shared SSE.
	bundle, err := setup.New(setup.Config{ //nolint:exhaustruct // demo uses in-memory defaults
		Title:     "Setup Demo Test",
		LogoutURL: "/dev-logout",
		ServiceConfig: &usermgmt.ServiceConfig{
			MaxUsers: 50,
		},
		SSEPath: "/sse",
	})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	defer func() { _ = bundle.Close() }()

	token := seed(ctx, bundle)

	mux := http.NewServeMux()
	mux.HandleFunc("/dev-login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct // test cookie
			Name: cookieName, Value: token, Path: "/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: dayInSeconds,
		})
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
	})

	server := httptest.NewServer(bundle.Handler(mux))
	defer server.Close()

	// Same demo route main() registers: push a custom event through the
	// bundle's shared fan-out hub.
	mux.HandleFunc("POST /broadcast", func(w http.ResponseWriter, _ *http.Request) {
		bundle.Broadcaster.Broadcast(sse.Event{
			Event: "demoBroadcast",
			Data:  `{"message":"hello from setup-demo test"}`,
		})
		w.WriteHeader(http.StatusAccepted)
	})

	// Public routes.
	for path, want := range map[string]int{
		"/health":     http.StatusOK,
		"/":           http.StatusOK,           // login page
		"/auth/me":    http.StatusUnauthorized, // no session -> 401
		"/dashboard/": http.StatusUnauthorized,
		"/admin/":     http.StatusUnauthorized,
		"/sse":        http.StatusUnauthorized, // shared SSE is session-gated
	} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode != want {
			t.Fatalf("GET %s (unauthenticated): status %d, want %d", path, resp.StatusCode, want)
		}
	}

	// Dev-login sets the session cookie via redirect. Capture the redirect
	// response itself (no cookie jar: a following client would drop the
	// cookie on the redirect hop and report the final response's cookies).
	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := noRedirect.Get(server.URL + "/dev-login")
	if err != nil {
		t.Fatalf("GET /dev-login: %v", err)
	}

	var cookies []string

	for _, c := range resp.Cookies() {
		cookies = append(cookies, c.Name+"="+c.Value)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Authenticated routes: both panels render with the session cookie.
	// The user read model projects UserRegistered asynchronously, so the
	// first attempt may race the projection — poll until it catches up
	// (this is the documented read-your-writes contract, not a hack).
	authedReq := func(path string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}

		req.Header.Set("Cookie", strings.Join(cookies, "; "))

		return req
	}

	var adminResp *http.Response

	deadline := time.Now().Add(10 * time.Second)

	for range 200 {
		adminResp, err = http.DefaultClient.Do(authedReq("/admin/"))
		if err != nil {
			t.Fatalf("GET /admin/ (authed): %v", err)
		}

		adminBody, _ := io.ReadAll(adminResp.Body)
		adminResp.Body.Close()

		if adminResp.StatusCode == http.StatusOK {
			break
		}

		if adminResp.StatusCode != http.StatusUnauthorized || time.Now().After(deadline) {
			t.Fatalf("GET /admin/ (authed): status %d, want 200. Body: %s",
				adminResp.StatusCode, adminBody)
		}

		time.Sleep(25 * time.Millisecond)
	}

	dashResp, err := (&http.Client{Timeout: 10 * time.Second}).Do(authedReq("/dashboard/"))
	if err != nil {
		t.Fatalf("GET /dashboard/ (authed): %v", err)
	}
	defer dashResp.Body.Close()

	if dashResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /dashboard/ (authed): status %d, want 200", dashResp.StatusCode)
	}

	// Authenticated SSE connects (200 + text/event-stream) and the shared
	// hub delivers a broadcast pushed through POST /broadcast.
	sseReq, err := http.NewRequest(http.MethodGet, server.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("new SSE request: %v", err)
	}
	sseReq.Header.Set("Cookie", strings.Join(cookies, "; "))
	sseReq.Header.Set("Accept", "text/event-stream")

	sseCtx, sseCancel := context.WithTimeout(ctx, 5*time.Second)
	defer sseCancel()
	sseResp, err := (&http.Client{Timeout: 5 * time.Second}).Do(sseReq.WithContext(sseCtx))
	if err != nil {
		t.Fatalf("GET /sse (authed): %v", err)
	}
	defer sseResp.Body.Close()

	if sseResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sse (authed): status %d, want 200", sseResp.StatusCode)
	}

	broadcastResp, err := http.Post(server.URL+"/broadcast", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /broadcast: %v", err)
	}
	_, _ = io.Copy(io.Discard, broadcastResp.Body)
	broadcastResp.Body.Close()

	if broadcastResp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /broadcast: status %d, want 202", broadcastResp.StatusCode)
	}
}
