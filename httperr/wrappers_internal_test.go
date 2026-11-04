package httperr

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// White-box pins for the wrapper types WithHTTPStatus and WithRetryAfter build.
// They live here because those types are unexported members of this package.

func TestErrorWrapperMethods_PreserveCauseMessage(t *testing.T) {
	t.Parallel()

	cause := errorfamily.NewRejection("test.detail", "the client-facing message")

	statusErr := WithHTTPStatus(cause, http.StatusTeapot)

	statusWrapped, ok := errors.AsType[*httpStatusError](statusErr)
	if !ok {
		t.Fatal("WithHTTPStatus must wrap in httpStatusError")
	}

	if !strings.Contains(statusWrapped.Error(), "the client-facing message") {
		t.Fatalf("httpStatusError.Error() = %q, want it to preserve the cause message", statusWrapped.Error())
	}

	retryErr := WithRetryAfter(cause, 30*time.Second)

	retryWrapped, ok := errors.AsType[*retryAfterError](retryErr)
	if !ok {
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
