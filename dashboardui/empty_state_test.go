package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateEmptyStateCopy_UnknownKeyRejected(t *testing.T) {
	t.Parallel()

	_, err := mustTestDashboardWithConfig(t, Config{}).config.withDefaults()
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}

	bad := Config{EmptyStateCopy: map[EmptyStateKey]EmptyStateCopy{
		"commandz": {Title: "typo"},
	}}

	if _, err := bad.withDefaults(); err == nil {
		t.Fatal("unknown EmptyStateCopy key must be rejected")
	}
}

func TestResolveEmptyStates_MergeSemantics(t *testing.T) {
	t.Parallel()

	t.Run("nil overrides resolve to upstream defaults", func(t *testing.T) {
		t.Parallel()

		resolved := Config{}.resolveEmptyStates()

		for key, def := range defaultEmptyStates {
			if resolved[key] != def {
				t.Errorf("resolved[%s] = %+v, want upstream default %+v", key, resolved[key], def)
			}
		}
	})

	t.Run("message-only override keeps the default title", func(t *testing.T) {
		t.Parallel()

		resolved := Config{EmptyStateCopy: map[EmptyStateKey]EmptyStateCopy{
			EmptyStateCommands: {Message: "Nothing dispatched yet."},
		}}.resolveEmptyStates()

		got := resolved[EmptyStateCommands]

		if got.Title != "No commands recorded" {
			t.Errorf("Title = %q, want the upstream default", got.Title)
		}

		if got.Message != "Nothing dispatched yet." {
			t.Errorf("Message = %q, want the override", got.Message)
		}
	})

	t.Run("full override replaces both fields", func(t *testing.T) {
		t.Parallel()

		resolved := Config{EmptyStateCopy: map[EmptyStateKey]EmptyStateCopy{
			EmptyStateQueries: {Title: "Quiet", Message: "No queries yet."},
		}}.resolveEmptyStates()

		if got := resolved[EmptyStateQueries]; got.Title != "Quiet" || got.Message != "No queries yet." {
			t.Errorf("resolved = %+v, want the full override", got)
		}
	})

	t.Run("overrides do not mutate the defaults map", func(t *testing.T) {
		t.Parallel()

		_ = Config{EmptyStateCopy: map[EmptyStateKey]EmptyStateCopy{
			EmptyStateCommands: {Title: "BRANDED"},
		}}.resolveEmptyStates()

		if defaultEmptyStates[EmptyStateCommands].Title != "No commands recorded" {
			t.Error("resolveEmptyStates must not mutate defaultEmptyStates")
		}
	})
}

func TestEmptyStateCopy_EndToEndRender(t *testing.T) {
	t.Parallel()

	d := mustTestDashboardWithConfig(t, Config{
		EmptyStateCopy: map[EmptyStateKey]EmptyStateCopy{
			EmptyStateCommands: {Title: "NOTHING DISPATCHED", Message: "Your commands show up here."},
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/commands", nil)
	d.commandsIndexHandler(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !strings.Contains(body, "NOTHING DISPATCHED") || !strings.Contains(body, "Your commands show up here.") {
		t.Error("overridden command empty-state copy must render")
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/events", nil)
	d.eventsIndexHandler(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on /events, got %d", w2.Code)
	}

	if !strings.Contains(w2.Body.String(), "No events yet") {
		t.Error("non-overridden panels must keep the upstream default copy")
	}
}
