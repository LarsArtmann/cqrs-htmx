package dashboardui

import (
	"strings"
	"testing"
)

func TestPushCursor(t *testing.T) {
	cases := []struct {
		name    string
		history string
		cursor  string
		want    string
	}{
		{"empty_history_empty_cursor", "", "", ""},
		{"empty_history_real_cursor", "", "abc", "abc"},
		{"empty_cursor_skipped", "x,y", "", "x,y"},
		{"append_to_existing", "x,y", "z", "x,y,z"},
		{"single_to_double", "x", "y", "x,y"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pushCursor(tc.history, tc.cursor)
			if got != tc.want {
				t.Errorf("pushCursor(%q, %q) = %q, want %q", tc.history, tc.cursor, got, tc.want)
			}
		})
	}
}

func TestPopCursor(t *testing.T) {
	cases := []struct {
		name        string
		history     string
		wantLast    string
		wantRemain  string
	}{
		{"empty", "", "", ""},
		{"single", "abc", "abc", ""},
		{"multiple", "x,y,z", "z", "x,y"},
		{"two", "x,y", "y", "x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			last, remain := popCursor(tc.history)
			if last != tc.wantLast {
				t.Errorf("last = %q, want %q", last, tc.wantLast)
			}

			if remain != tc.wantRemain {
				t.Errorf("remain = %q, want %q", remain, tc.wantRemain)
			}
		})
	}
}

// TestRenderPagination_PreviousUsesCursorHistory verifies that the Previous
// link navigates back using the cursor history, not just to page 1.
func TestRenderPagination_PreviousUsesCursorHistory(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{
		HasNext:     true,
		NextCursor:  "cursor3",
		PageSize:    50,
		HasPrev:     true,
		After:       "cursor2",
		PrevHistory: "cursor1",
	}, "")

	// Previous should pop cursor1 from history and use it as the new after.
	if !strings.Contains(html, "after=cursor1") {
		t.Errorf("Previous link should use after=cursor1 from history, got: %s", html)
	}

	// Previous link should NOT have after=cursor2 (the current page's cursor).
	prevSection := html
	if idx := strings.Index(html, "Next"); idx > 0 {
		prevSection = html[:idx]
	}

	if strings.Contains(prevSection, "after=cursor2") {
		t.Errorf("Previous link should not use current cursor, got: %s", prevSection)
	}
}

// TestRenderPagination_NextPushesCursor verifies that the Next link pushes
// the current cursor onto the history stack.
func TestRenderPagination_NextPushesCursor(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{
		HasNext:     true,
		NextCursor:  "cursor2",
		PageSize:    50,
		HasPrev:     true,
		After:       "cursor1",
		PrevHistory: "",
	}, "")

	// Next should use cursor2 as after and push cursor1 into prev.
	if !strings.Contains(html, "after=cursor2") {
		t.Errorf("Next link should use after=cursor2, got: %s", html)
	}

	if !strings.Contains(html, "prev=cursor1") {
		t.Errorf("Next link should push cursor1 into prev history, got: %s", html)
	}
}

// TestRenderPagination_PreviousToFirstPage verifies that when there is no
// history, Previous navigates to the first page (empty after).
func TestRenderPagination_PreviousToFirstPage(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{
		HasNext:     false,
		NextCursor:  "",
		PageSize:    50,
		HasPrev:     true,
		After:       "cursor1",
		PrevHistory: "",
	}, "")

	// Previous should go to page 1: after should be absent (empty).
	prevSection := html
	if idx := strings.Index(html, "Next"); idx > 0 {
		prevSection = html[:idx]
	}

	if strings.Contains(prevSection, "after=cursor1") {
		t.Errorf("Previous to page 1 should not carry current cursor, got: %s", prevSection)
	}

	if !strings.Contains(prevSection, "limit=50") {
		t.Errorf("Previous link should include limit, got: %s", prevSection)
	}
}

// TestRenderPagination_FiltersPreserved verifies extra filter params survive
// in both Previous and Next links.
func TestRenderPagination_FiltersPreserved(t *testing.T) {
	html := renderPagination("/d", "/events", paginationState{
		HasNext:     true,
		NextCursor:  "c2",
		PageSize:    50,
		HasPrev:     true,
		After:       "c1",
		PrevHistory: "",
	}, "type=user.created")

	if !strings.Contains(html, "type=user.created") {
		t.Errorf("filter params should be preserved, got: %s", html)
	}
}
