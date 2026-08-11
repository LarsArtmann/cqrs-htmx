package usermgmt

import (
	"testing"
)

// TestParseUserID_Invalid verifies that ParseUserID rejects invalid ULIDs.
func TestParseUserID_Invalid(t *testing.T) {
	t.Parallel()

	_, err := ParseUserID("not-a-ulid")
	if err == nil {
		t.Error("expected error for invalid ULID")
	}
}

// TestMustParseUserID_Panic verifies that MustParseUserID panics on invalid input.
func TestMustParseUserID_Panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid ULID")
		}
	}()
	_ = MustParseUserID("invalid")
}

// TestParseActorID_RoundTrip verifies that ParseActorID correctly reconstructs
// ActorID from PrefixedString output.
func TestParseActorID_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		kind ActorKind
	}{
		{"user", "01HXKYGEG0QH8XJYQKZ3R8WZAA", ActorUser},
		{"bot", "my-bot-name", ActorBot},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original := NewActorID(tc.kind, tc.raw)
			prefixed := original.PrefixedString()
			parsed, err := ParseActorID(prefixed)
			if err != nil {
				t.Fatalf("ParseActorID(%q): %v", prefixed, err)
			}
			if parsed.Kind() != tc.kind {
				t.Errorf("kind: got %v, want %v", parsed.Kind(), tc.kind)
			}
			if parsed.String() != tc.raw {
				t.Errorf("raw: got %q, want %q", parsed.String(), tc.raw)
			}
		})
	}
}

// TestParseActorID_NoPrefix verifies that a string without a kind prefix
// is rejected (the prefixed format "kind:raw" is mandatory).
func TestParseActorID_NoPrefix(t *testing.T) {
	t.Parallel()

	_, err := ParseActorID("just-an-id")
	if err == nil {
		t.Error("expected error for unprefixed string, got nil")
	}
}

// TestMustParseEmail_Panic verifies that MustParseEmail panics on invalid input.
func TestMustParseEmail_Panic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid email")
		}
	}()
	_ = MustParseEmail("not-an-email")
}
