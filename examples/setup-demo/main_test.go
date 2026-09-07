package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-sse"
)

// TestDemoApp_EndToEnd boots the same composition as main() (minus the real
// listener) and walks the full auth flow: public routes answer, protected
// routes gate on the session, and the dev-login cookie unlocks both panels.
func TestDemoApp_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Same composition as main(): ServiceConfig escape hatch + shared SSE +
	// DataStar feed on the same hub.
	bundle, err := setup.New(setup.Config{ //nolint:exhaustruct // demo uses in-memory defaults
		Title:     "Setup Demo Test",
		LogoutURL: "/dev-logout",
		ServiceConfig: &usermgmt.ServiceConfig{
			MaxUsers: 50,
		},
		SSEPath:      "/sse",
		DataStarPath: "/ds/events",
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

	// Same demo route main() registers: push ONE action through the bundle's
	// shared fan-out hub twice — raw SSE event + DataStar signal patch.
	broadcastCount := 0
	mux.HandleFunc("POST /broadcast", func(w http.ResponseWriter, _ *http.Request) {
		broadcastCount++

		bundle.Broadcaster.Broadcast(sse.Event{
			Event: "demoBroadcast",
			Data:  `{"message":"hello from setup-demo test"}`,
		})

		if bundle.DataStarBroadcaster != nil {
			patch, perr := ds.SignalsPatch(map[string]any{"broadcasts": broadcastCount})
			if perr == nil {
				bundle.DataStarBroadcaster.Broadcast(patch)
			}
		}

		w.WriteHeader(http.StatusAccepted)
	})

	// Public routes.
	for path, want := range map[string]int{
		"/health":        http.StatusOK,
		"/":              http.StatusOK,           // login page
		"/auth/me":       http.StatusUnauthorized, // no session -> 401
		"/dashboard/":    http.StatusUnauthorized,
		"/admin/":        http.StatusUnauthorized,
		"/sse":           http.StatusUnauthorized, // shared SSE is session-gated
		"/ds/events":     http.StatusUnauthorized, // DataStar feed, same gate
		"/datastar.js":   http.StatusOK,           // SDK script is public
		"/ds-demo":       http.StatusOK,           // demo client page is public
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

	// Read actual SSE frames from the still-open connection until the
	// broadcast event arrives — status codes alone prove nothing about
	// delivery. The journal backfill may deliver registration events first,
	// so scan until the demoBroadcast payload shows up; the request context
	// bounds the wait (test fails on timeout instead of hanging).
	scanner := bufio.NewScanner(sseResp.Body)
	found := false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "hello from setup-demo") {
			found = true
			break
		}
	}
	if err := scanner.Err(); err != nil && !found {
		t.Fatalf("reading SSE frames: %v (broadcast frame never arrived)", err)
	}
	if !found {
		t.Fatal("broadcast frame not delivered on the open SSE connection")
	}

	// Dual-transport assertion (ADR-0050): the SAME broadcast action also
	// reaches the DataStar feed. The DataStar stream flushes on first frame
	// (no connected-event), so broadcast WHILE connecting: fire the request
	// in a goroutine, push the patch, then read frames off the response.
	var dsResp *http.Response
	dsErr := make(chan error, 1)

	go func() {
		dsReq, err := http.NewRequest(http.MethodGet, server.URL+"/ds/events", nil)
		if err != nil {
			dsErr <- err

			return
		}

		dsReq.Header.Set("Cookie", strings.Join(cookies, "; "))
		dsReq.Header.Set("Accept", "text/event-stream")

		dsCtx, dsCancel := context.WithTimeout(ctx, 5*time.Second)
		defer dsCancel()

		dsResp, err = (&http.Client{Timeout: 5 * time.Second}).Do(dsReq.WithContext(dsCtx))
		dsErr <- err
	}()

	time.Sleep(150 * time.Millisecond)

	second, err := http.Post(server.URL+"/broadcast", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /broadcast (second): %v", err)
	}
	_, _ = io.Copy(io.Discard, second.Body)
	second.Body.Close()

	if err := <-dsErr; err != nil {
		t.Fatalf("GET /ds/events (authed): %v", err)
	}

	if dsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /ds/events (authed): status %d, want 200", dsResp.StatusCode)
	}

	dsScanner := bufio.NewScanner(dsResp.Body)
	patchFound := false
	for dsScanner.Scan() {
		if strings.Contains(dsScanner.Text(), "datastar-patch-signals") {
			patchFound = true
			break
		}
	}
	if err := dsScanner.Err(); err != nil && !patchFound {
		t.Fatalf("reading DataStar frames: %v (patch frame never arrived)", err)
	}
	if !patchFound {
		t.Fatal("DataStar signal patch not delivered on the open /ds/events connection")
	}
}
