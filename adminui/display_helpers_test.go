package adminui

import (
	"testing"

	"github.com/larsartmann/templ-components/display"
)

func TestRoleBadgeType_AllBranches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		role string
		want display.BadgeType
	}{
		{"super_admin", display.BadgePrimary},
		{"admin", display.BadgeInfo},
		{"owner", display.BadgeSuccess},
		{"viewer", display.BadgeWarning},
		{"unknown", display.BadgeNeutral},
		{"", display.BadgeNeutral},
	}
	for _, tc := range tests {
		got := roleBadgeType(tc.role)
		if got != tc.want {
			t.Errorf("roleBadgeType(%q) = %v, want %v", tc.role, got, tc.want)
		}
	}
}

func TestInitials_EdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"", "?"},
		{"a", "A"},
		{"ab", "AB"},
		{"john.doe@example.com", "JD"},
		{"john@example.com", "JO"},
		{"John Doe", "JD"},
		{"john_doe", "JD"},
		{"john-doe", "JD"},
		{"x@y.com", "X"},
		{"  spaces  ", "SP"},
	}
	for _, tc := range tests {
		got := initials(tc.input)
		if got != tc.want {
			t.Errorf("initials(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCapList_Truncation(t *testing.T) {
	t.Parallel()
	// Under limit — returns full list
	short := make([]int, 5)
	out, total := capList(short)
	if len(out) != 5 || total != 5 {
		t.Errorf("short list: got len=%d total=%d, want 5/5", len(out), total)
	}

	// Over limit — returns truncated
	long := make([]int, MaxListRows+10)
	out, total = capList(long)
	if len(out) != MaxListRows || total != MaxListRows+10 {
		t.Errorf("long list: got len=%d total=%d, want %d/%d", len(out), total, MaxListRows, MaxListRows+10)
	}
}
