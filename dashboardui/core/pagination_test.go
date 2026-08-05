package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPushCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		history  string
		cursor   string
		expected string
	}{
		{"empty history, empty cursor", "", "", ""},
		{"empty history, non-empty cursor", "", "abc", "abc"},
		{"non-empty history, empty cursor", "abc", "", "abc"},
		{"both non-empty", "abc", "def", "abc,def"},
		{"multi-entry history", "abc,def", "ghi", "abc,def,ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := PushCursor(tt.history, tt.cursor)
			if got != tt.expected {
				t.Errorf("PushCursor(%q, %q) = %q, want %q", tt.history, tt.cursor, got, tt.expected)
			}
		})
	}
}

func TestPopCursor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		history       string
		wantLast      string
		wantRemaining string
	}{
		{"empty", "", "", ""},
		{"single entry", "abc", "abc", ""},
		{"two entries", "abc,def", "def", "abc"},
		{"three entries", "abc,def,ghi", "ghi", "abc,def"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			last, remaining := PopCursor(tt.history)
			if last != tt.wantLast {
				t.Errorf("PopCursor(%q) last = %q, want %q", tt.history, last, tt.wantLast)
			}

			if remaining != tt.wantRemaining {
				t.Errorf("PopCursor(%q) remaining = %q, want %q", tt.history, remaining, tt.wantRemaining)
			}
		})
	}
}

func TestPaginationQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		after        string
		prevHistory  string
		pageSize     int
		extraParams  string
		containsAll  []string
	}{
		{
			name:        "first page no filters",
			after:       "",
			prevHistory: "",
			pageSize:    50,
			extraParams: "",
			containsAll: []string{"limit=50"},
		},
		{
			name:        "with cursor and prev",
			after:       "evt123",
			prevHistory: "abc,def",
			pageSize:    25,
			extraParams: "",
			containsAll: []string{"after=evt123", "prev=abc,def", "limit=25"},
		},
		{
			name:        "with extra params",
			after:       "",
			prevHistory: "",
			pageSize:    10,
			extraParams: "type=User&streamID=abc",
			containsAll: []string{"limit=10", "type=User&streamID=abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			q := PaginationQuery(tt.after, tt.prevHistory, tt.pageSize, tt.extraParams)

			for _, s := range tt.containsAll {
				if !contains(q, s) {
					t.Errorf("PaginationQuery result %q should contain %q", q, s)
				}
			}
		})
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		(len(haystack) > 0 && len(needle) > 0 && indexOf(haystack, needle) >= 0))
}

func indexOf(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}

	return -1
}

func TestComputePageStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state PageState
		want  int
	}{
		{
			name:  "first page no prev",
			state: PageState{PageSize: 50},
			want:  1,
		},
		{
			name:  "after one page",
			state: PageState{After: "cur1", PageSize: 50},
			want:  51,
		},
		{
			name:  "after two pages via history",
			state: PageState{After: "cur2", PrevHistory: "cur1", PageSize: 25},
			want:  51,
		},
		{
			name:  "page size zero defaults",
			state: PageState{After: "cur1"},
			want:  51, // defaultPageSize = 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ComputePageStart(tt.state)
			if got != tt.want {
				t.Errorf("ComputePageStart() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWithCountInfo(t *testing.T) {
	t.Parallel()

	t.Run("last page sets total", func(t *testing.T) {
		t.Parallel()

		s := PageState{
			PageSize:    50,
			After:       "cur1",
			HasNext:     false,
		}.WithCountInfo(10)

		if s.PageLen != 10 {
			t.Errorf("PageLen = %d, want 10", s.PageLen)
		}

		if s.PageStart != 51 {
			t.Errorf("PageStart = %d, want 51", s.PageStart)
		}

		if s.TotalCount != "60" {
			t.Errorf("TotalCount = %q, want %q", s.TotalCount, "60")
		}
	})

	t.Run("has next does not set total", func(t *testing.T) {
		t.Parallel()

		s := PageState{
			PageSize: 50,
			HasNext:  true,
		}.WithCountInfo(50)

		if s.TotalCount != "" {
			t.Errorf("TotalCount should be empty when HasNext, got %q", s.TotalCount)
		}
	})

	t.Run("zero items does not set total", func(t *testing.T) {
		t.Parallel()

		s := PageState{
			PageSize: 50,
			HasNext:  false,
		}.WithCountInfo(0)

		if s.TotalCount != "" {
			t.Errorf("TotalCount should be empty with 0 items, got %q", s.TotalCount)
		}
	})
}

func TestParsePageSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		query       string
		defaultSize int
		want        int
	}{
		{"no limit param", "/?", 50, 50},
		{"valid limit", "/?limit=25", 50, 25},
		{"limit below 1", "/?limit=0", 50, 50},
		{"limit negative", "/?limit=-5", 50, 50},
		{"limit non-numeric", "/?limit=abc", 50, 50},
		{"limit above max", "/?limit=500", 50, maxPageSize},
		{"custom default", "/?", 30, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, tt.query, nil)
			got := ParsePageSize(r, tt.defaultSize)
			if got != tt.want {
				t.Errorf("ParsePageSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseCursorParams(t *testing.T) {
	t.Parallel()

	t.Run("no params", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/?", nil)
		after, prev, hasPrev := ParseCursorParams(r)

		if after != "" || prev != "" || hasPrev {
			t.Errorf("ParseCursorParams() = (%q, %q, %v), want empty", after, prev, hasPrev)
		}
	})

	t.Run("with after and prev", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/?after=cur1&prev=abc,def", nil)
		after, prev, hasPrev := ParseCursorParams(r)

		if after != "cur1" {
			t.Errorf("after = %q, want %q", after, "cur1")
		}

		if prev != "abc,def" {
			t.Errorf("prev = %q, want %q", prev, "abc,def")
		}

		if !hasPrev {
			t.Error("hasPrev should be true when after is set")
		}
	})
}

func TestDefaultPageSize(t *testing.T) {
	t.Parallel()

	if got := DefaultPageSize(); got != defaultPageSize {
		t.Errorf("DefaultPageSize() = %d, want %d", got, defaultPageSize)
	}
}

func TestMaxPageSize(t *testing.T) {
	t.Parallel()

	if got := MaxPageSize(); got != maxPageSize {
		t.Errorf("MaxPageSize() = %d, want %d", got, maxPageSize)
	}
}
