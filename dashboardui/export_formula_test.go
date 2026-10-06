package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestNeutralizeCSVFormula pins M03/F09: every OWASP CSV-injection trigger
// character in leading position gets the single-quote text prefix; everything
// else passes through byte-for-byte.
func TestNeutralizeCSVFormula(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"a=b", "a=b"},   // trigger only matters in leading position
		{"1-2", "1-2"},   // hyphen not leading
		{"x@y", "x@y"},   // at not leading
		{"=1+1", "'=1+1"}, // formula
		{"+cmd|' /C calc'!A0", "'+cmd|' /C calc'!A0"},
		{"-2+3+cmd|' /C calc'!A0", "'-2+3+cmd|' /C calc'!A0"},
		{"@SUM(A1:A2)", "'@SUM(A1:A2)"},
		{"\tTAB", "'\tTAB"},
		{"\rCR", "'\rCR"},
	}
	for _, tt := range tests {
		if got := neutralizeCSVFormula(tt.in); got != tt.want {
			t.Errorf("neutralizeCSVFormula(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestEventsExport_CSVFormulaInjection pins M03/F10: hostile event metadata
// (a type crafted as a spreadsheet formula) must reach the CSV export
// neutralized — operators open these exports in Excel/Sheets, and an
// unneutralized leading =, +, -, or @ executes as a formula there.
func TestEventsExport_CSVFormulaInjection(t *testing.T) {
	streamID := id.NewStreamID()
	hyperlinkPayload, _ := event.New(
		event.Type(`=HYPERLINK("http://evil.example","click")`),
		streamID,
		"TestAgg",
		1,
		map[string]string{"k": "v"},
	)
	sumPayload, _ := event.New(
		event.Type("=SUM(A1:A2)"),
		streamID,
		"TestAgg",
		2,
		map[string]string{"k": "v"},
	)

	d := MustNew(Config{
		Journal: &fakeSeekableJournal{events: []event.Event{hyperlinkPayload, sumPayload}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/events?format=csv", nil)
	d.eventsIndexHandler(w, r)

	body := w.Body.String()

	for _, want := range []string{
		`'=HYPERLINK(""http://evil.example"",""click"")"`, // quoted cell, neutralized
		`'=SUM(A1:A2)`, // plain cell, neutralized
	} {
		if !strings.Contains(body, want) {
			t.Errorf("CSV export must neutralize formula cells; body missing %q\nbody:\n%s", want, body)
		}
	}

	// No cell may start (post-CSV-quoting) with a formula trigger.
	for _, raw := range []string{"\"=", "\n=", ",=", "\"+", "\n+", ",+", "\"-", "\n-", ",-", "\"@", "\n@", ",@"} {
		if strings.Contains(body, raw) {
			t.Errorf("CSV export leaked an unneutralized formula start %q\nbody:\n%s", raw, body)
		}
	}
}
