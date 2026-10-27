package adminui

import (
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newAccentService builds the minimal in-memory service an accent test needs.
func newAccentService(t *testing.T) *usermgmt.Service {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		AuditLog: usermgmt.NewAuditLog(),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	return svc
}

// TestConfig_AccentColor_RejectsAtNew pins S36/w02.3: an AccentColor outside
// the shared CSS-color grammar (dashboardui parity) is rejected at New, not
// silently spliced into the inline <style> block. Pre-fix, the value reached
// layout.templ's raw `:root{--accent:...}` interpolation unvalidated, so a
// value like `red;} body{background:url(x)}` injected page rules.
func TestConfig_AccentColor_RejectsAtNew(t *testing.T) {
	rejects := []string{
		"red; background:url(javascript:alert(1))",
		"red;} body{background:url(x)}",
		"url(x)",
		"#12g",            // non-hex digit
		"#12345",          // bad hex length
		"hsl(1deg 2% 3%)", // letters in components (deg)
		"currentcolor",    // computed reference, not a literal
		strings.Repeat("a", maxAccentColorLen+1),
		"\"quoted\"",
	}
	for _, accent := range rejects {
		_, err := New(Config{Service: newAccentService(t), AccentColor: accent})
		if err == nil {
			t.Errorf("New(AccentColor=%q): expected rejection, got nil", accent)
		}
	}
}

// TestConfig_AccentColor_AcceptsGrammar pins the accepted side: every hex
// length, numeric function forms, and named colors pass New and land in the
// page's style block.
func TestConfig_AccentColor_AcceptsGrammar(t *testing.T) {
	accepts := []string{
		"#0ea5e9",
		"#abc",
		"#abcd",
		"#aabbccdd",
		"rgb(14 165 233)",
		"rgba(14,165,233,.9)",
		"hsl(199 89 48)",
		"hsla(199,89%,48%,1)",
		"skyblue",
		"TRANSPARENT",
	}
	for _, accent := range accepts {
		config := Config{Service: newAccentService(t), AccentColor: accent}
		panel, err := New(config)
		if err != nil {
			t.Errorf("New(AccentColor=%q): %v", accent, err)

			continue
		}
		if panel.config.AccentColor != accent {
			t.Errorf("New(AccentColor=%q): value mutated to %q", accent, panel.config.AccentColor)
		}
	}
}

// TestConfig_AccentColor_DefaultStillValid guards the default itself: if
// DefaultAccentColor ever drifts outside the grammar, every default panel
// would fail at New — the loudest possible signal, caught here first.
func TestConfig_AccentColor_DefaultStillValid(t *testing.T) {
	if !isValidAccentColor(DefaultAccentColor) {
		t.Fatalf("DefaultAccentColor %q fails its own grammar", DefaultAccentColor)
	}
}
