package setup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	appkit "github.com/larsartmann/go-appkit"
)

// TestRunWithAppkit_ServerTimingHeaderDefaultAndOptOut (GitHub issue #43):
// Config.AppkitServerTiming forwards to appkit.ServiceConfig.ServerTiming,
// the tri-state knob over go-appkit >= v0.10.0's OUTERMOST W3C Server-Timing
// middleware. Pinned through a real appkit listener, one boot per state:
// nil rides the appkit default (header on), &true matches nil, &false omits
// the middleware entirely so the header is absent.
func TestRunWithAppkit_ServerTimingHeaderDefaultAndOptOut(t *testing.T) {
	off, on := false, true

	for _, tc := range []struct {
		name       string
		knob       *bool
		wantHeader bool
	}{
		{name: "nil_is_the_appkit_default_header_on", knob: nil, wantHeader: true},
		{name: "false_omits_the_header", knob: &off, wantHeader: false},
		{name: "true_matches_the_default", knob: &on, wantHeader: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle := MustNew(Config{
				Title:              "server-timing-" + tc.name,
				AppkitServerTiming: tc.knob,
			})
			defer func() { _ = bundle.Close() }()

			hello := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)

				_, _ = io.WriteString(w, "server-timing-ok")
			})

			addr := freeLocalAddr(t)
			stop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
				return bundle.runWithAppkit(ctx, addr, h, testDrainDelay, appkit.LogLevelError)
			}, addr, hello)
			defer stop()

			client := &http.Client{Timeout: 10 * time.Second}

			resp, err := client.Get("http://" + addr + "/")
			if err != nil {
				t.Fatalf("get: %v", err)
			}

			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			if resp.StatusCode != http.StatusOK || string(body) != "server-timing-ok" {
				t.Fatalf("status = %d body = %q, want 200 server-timing-ok", resp.StatusCode, body)
			}

			// One header line per middleware entry; join before asserting so a
			// future extra entry cannot dodge the contains check.
			got := strings.Join(resp.Header.Values("Server-Timing"), ", ")

			// appkit's own middleware renders the total entry with a desc
			// segment between total and dur ("total;desc=\"Total request\";dur=…"),
			// so assert the two halves, not the adjacent form.
			if tc.wantHeader && (!strings.Contains(got, "total;") || !strings.Contains(got, "dur=")) {
				t.Errorf("Server-Timing = %q, want it to contain total; and dur=", got)
			}

			if !tc.wantHeader && got != "" {
				t.Errorf("Server-Timing = %q, want the header absent with AppkitServerTiming=false", got)
			}
		})
	}
}
