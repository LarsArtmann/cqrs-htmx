package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const beacon = `{"events":[{"kind":"pageview","path":"/checkout"}]}`

// TestDemoApp_EndToEnd boots the same composition as main() (minus the
// listener) and walks the tracking loop: a page carries the tracker tag, a
// beacon is stored and listed, a signed-in visitor's signals carry the user
// key, and one subject's stream can be read on its own.
func TestDemoApp_EndToEnd(t *testing.T) {
	app, err := newDemo(t.Context())
	if err != nil {
		t.Fatalf("newDemo: %v", err)
	}

	t.Cleanup(func() { _ = app.bundle.Close() })

	server := httptest.NewServer(app.bundle.Handler(app.mux))
	t.Cleanup(server.Close)

	status, shop := get(t, server.Client(), server.URL+"/")
	if status != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", status)
	}

	for _, want := range []string{`<script src="/-/tracker.js" defer></script>`, `data-track="add-to-cart"`, `data-track-impression="card:coffee"`} {
		if !strings.Contains(shop, want) {
			t.Errorf("the shop page lacks %s", want)
		}
	}

	status, script := get(t, server.Client(), server.URL+trackerScript)
	if status != http.StatusOK || !strings.Contains(script, "cqrsTracker") {
		t.Errorf("GET %s = %d with the tracker: %t, want 200 and the tracker", trackerScript, status, strings.Contains(script, "cqrsTracker"))
	}

	device := id.NewClientID()

	if status = postBeacon(t, server.Client(), server.URL+"/-/track", device.String(), ""); status != http.StatusAccepted {
		t.Fatalf("POST /-/track = %d, want 202", status)
	}

	status, signals := get(t, server.Client(), server.URL+"/signals")
	if status != http.StatusOK || !strings.Contains(signals, device.String()) || !strings.Contains(signals, "/checkout") {
		t.Errorf("GET /signals = %d; want the device key and the path listed:\n%s", status, signals)
	}

	// The demo shortcut signs the shopper in; their next signal is theirs.
	shopper := httptest.NewRecorder()
	app.devLogin(shopper, httptest.NewRequest(http.MethodGet, "/dev-login", nil))

	cookies := shopper.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("dev-login set %d cookie(s), want the session cookie", len(cookies))
	}

	if status = postBeacon(t, server.Client(), server.URL+"/-/track", device.String(), cookies[0].String()); status != http.StatusAccepted {
		t.Fatalf("POST /-/track signed in = %d, want 202", status)
	}

	status, own := get(t, server.Client(), server.URL+"/signals?user="+app.shopper.ID.String())
	if status != http.StatusOK || !strings.Contains(own, "user "+app.shopper.ID.String()) || !strings.Contains(own, device.String()) {
		t.Errorf("GET /signals?user=<shopper> = %d; want the shopper's stream naming the subject:\n%s", status, own)
	}

	// One signal, one subject: once the visitor is known, their signals
	// live in the user's stream, so the device's stream holds only the
	// anonymous page view.
	status, deviceSignals := get(t, server.Client(), server.URL+"/signals?device="+device.String())
	if status != http.StatusOK || strings.Count(deviceSignals, "/checkout") != 1 {
		t.Errorf("GET /signals?device=<key> = %d with %d rows; want the device's one anonymous signal",
			status, strings.Count(deviceSignals, "/checkout"))
	}

	if status, _ = get(t, server.Client(), server.URL+"/signals?user=not-an-id"); status != http.StatusBadRequest {
		t.Errorf("GET /signals?user=not-an-id = %d, want 400", status)
	}
}

// get fetches url without following redirects and returns status and body.
func get(t *testing.T, client *http.Client, url string) (int, string) {
	t.Helper()

	response, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}

	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatalf("GET %s: read body: %v", url, err)
	}

	return response.StatusCode, string(body)
}

// postBeacon sends one signal the way the tracker does: the device key in
// X-Client-Id and, for a signed-in visitor, the session cookie.
func postBeacon(t *testing.T, client *http.Client, url, device, cookie string) int {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(beacon))
	if err != nil {
		t.Fatalf("build the beacon: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Client-Id", device)

	if cookie != "" {
		request.Header.Set("Cookie", cookie)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}

	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()

	return response.StatusCode
}
