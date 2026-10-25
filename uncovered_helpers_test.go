package cqrshtmx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
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

func TestErrorWrapperMethods_PreserveCauseMessage(t *testing.T) {
	t.Parallel()

	cause := errorfamily.NewRejection("test.detail", "the client-facing message")

	statusErr := WithHTTPStatus(cause, http.StatusTeapot)

	var statusWrapped *httpStatusError
	if !errors.As(statusErr, &statusWrapped) {
		t.Fatal("WithHTTPStatus must wrap in httpStatusError")
	}

	if !strings.Contains(statusWrapped.Error(), "the client-facing message") {
		t.Fatalf("httpStatusError.Error() = %q, want it to preserve the cause message", statusWrapped.Error())
	}

	retryErr := WithRetryAfter(cause, 30*time.Second)

	var retryWrapped *retryAfterError
	if !errors.As(retryErr, &retryWrapped) {
		t.Fatal("WithRetryAfter must wrap in retryAfterError")
	}

	if !strings.Contains(retryWrapped.Error(), "the client-facing message") {
		t.Fatalf("retryAfterError.Error() = %q, want it to preserve the cause message", retryWrapped.Error())
	}
}

func TestStructuredError_ErrorReturnsDetail(t *testing.T) {
	t.Parallel()

	structured := StructuredError{Detail: "plain detail"}

	if structured.Error() != "plain detail" {
		t.Fatalf("Error() = %q, want the Detail field", structured.Error())
	}
}
