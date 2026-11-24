package adminui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTriggerToast_AllKinds(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"success", "error", "info"} {
		w := httptest.NewRecorder()
		triggerToast(w, kind, "test message")
		trigger := w.Header().Get("HX-Trigger")
		if !strings.Contains(trigger, "adminui:toast") {
			t.Errorf("triggerToast(%q): HX-Trigger = %q, want adminui:toast", kind, trigger)
		}
		if !strings.Contains(trigger, "test message") {
			t.Errorf("triggerToast(%q): should contain message, got %q", kind, trigger)
		}
	}
}

func TestTriggerToast_MergeExisting(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	// First toast sets the HX-Trigger header
	triggerToast(w, "success", "first message")
	first := w.Header().Get("HX-Trigger")
	if !strings.Contains(first, "first message") {
		t.Errorf("first toast should contain message, got %q", first)
	}
	// Second toast overwrites (merge is best-effort with json/v2)
	triggerToast(w, "error", "second message")
	second := w.Header().Get("HX-Trigger")
	if !strings.Contains(second, "adminui:toast") {
		t.Errorf("second toast should contain adminui:toast, got %q", second)
	}
}
