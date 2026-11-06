package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	healthdashboard "github.com/larsartmann/go-health-dashboard"
	gohealth "github.com/larsartmann/go-health"
)

// A3's verification bar: the bridge must forward go-health-dashboard
// options verbatim — proven on the CSP nonce pair, the one option class the
// audit flagged. A dashboard rendered under this repo's own recommended
// nonce-CSP posture (RecommendedSecurityMiddleware mints per-request
// nonces) must carry the nonce on its inline scripts and in its CSP header,
// or the browser silently refuses to execute the page.
func TestNewDashboard_ForwardsCSPNonceOptions(t *testing.T) {
	t.Parallel()

	probe := gohealth.NewChecks(map[string]gohealth.CheckFunc{
		"always-healthy": func(context.Context) error { return nil },
	})
	t.Cleanup(probe.Shutdown)

	t.Run("fixed nonce via WithNonce", func(t *testing.T) {
		t.Parallel()

		dash := NewDashboard(probe, healthdashboard.WithNonce("aGVsbG8gd29ybGQ="))

		rec := httptest.NewRecorder()
		dash.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if body := rec.Body.String(); !strings.Contains(body, `nonce="aGVsbG8gd29ybGQ="`) {
			t.Errorf("dashboard body missing the nonce attribute on inline scripts\nbody head:\n%s",
				firstLines(body))
		}

		// The dashboard does not set its own CSP header — the host does, via
		// RecommendedCSP. Pin that the recommended header actually admits the
		// same nonce, so the documented pairing cannot drift.
		csp := healthdashboard.RecommendedCSP("aGVsbG8gd29ybGQ=")
		if !strings.Contains(csp, "'nonce-aGVsbG8gd29ybGQ='") {
			t.Errorf("RecommendedCSP(nonce) missing the nonce source\ncsp: %s", csp)
		}
	})

	t.Run("per-request nonce via WithNonceExtractor", func(t *testing.T) {
		t.Parallel()

		dash := NewDashboard(probe, healthdashboard.WithNonceExtractor(func(*http.Request) string {
			return "cmVxdWVzdC1ub25jZQ=="
		}))

		rec := httptest.NewRecorder()
		dash.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if !strings.Contains(rec.Body.String(), `nonce="cmVxdWVzdC1ub25jZQ=="`) {
			t.Errorf("dashboard body missing the extracted nonce\nbody head:\n%s",
				firstLines(rec.Body.String()))
		}
	})
}

func firstLines(s string) string {
	lines := strings.SplitN(s, "\n", 15)
	if len(lines) < 15 {
		return s
	}
	return strings.Join(lines, "\n")
}
