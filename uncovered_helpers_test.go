package cqrshtmx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The deprecated re-export helpers (S33/S35 honesty pass) must keep delegating
// faithfully; these tests pin the delegation so removal in v5 is the only
// future change.

func TestDeprecatedSSEHelpers_Delegate(t *testing.T) {
	t.Parallel()

	if got := NewSSEEventID("7"); !strings.Contains(got.String(), "7") {
		t.Fatalf("NewSSEEventID = %q, want it to carry 7", got)
	}

	if got := MustParseSSEEventID("42"); !strings.Contains(got.String(), "42") {
		t.Fatalf("MustParseSSEEventID = %q, want it to carry 42", got)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("MustParseSSEEventID should panic on a newline-carrying id")
		}
	}()

	_ = MustParseSSEEventID("bad\nid")
}

func TestNewSSEStream_LastEventID(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events", nil)
	request.Header.Set("Last-Event-ID", "99")

	stream := NewSSEStream(recorder, request)
	if stream == nil {
		t.Fatal("NewSSEStream returned nil")
	}

	if got := LastEventIDFromRequest(request); !strings.Contains(got.String(), "99") {
		t.Fatalf("LastEventIDFromRequest = %q, want it to carry 99", got)
	}
}

func TestGenerateUserID_FreshDistinct(t *testing.T) {
	t.Parallel()

	first := GenerateUserID()
	second := GenerateUserID()

	if first.String() == "" || first == second {
		t.Fatalf("GenerateUserID must produce distinct non-empty ids: %q vs %q", first, second)
	}
}
