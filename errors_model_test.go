package cqrshtmx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestWithHTTPStatus_OverridesFamilyDefault(t *testing.T) {
	wrapped := cqrshtmx.WithHTTPStatus(cqrshtmx.ErrValidationFailed, http.StatusNotFound)
	got := cqrshtmx.MapError(wrapped)
	if got != http.StatusNotFound {
		t.Errorf("WithHTTPStatus: got %d, want %d", got, http.StatusNotFound)
	}
}

func TestWithHTTPStatus_PreservesSentinelIdentity(t *testing.T) {
	wrapped := cqrshtmx.WithHTTPStatus(cqrshtmx.ErrValidationFailed, http.StatusNotFound)
	if !errors.Is(wrapped, cqrshtmx.ErrValidationFailed) {
		t.Error("WithHTTPStatus: errors.Is must still match the wrapped sentinel")
	}
}

func TestWithHTTPStatus_NilReturnsNil(t *testing.T) {
	if cqrshtmx.WithHTTPStatus(nil, 404) != nil {
		t.Error("WithHTTPStatus(nil, ...) must return nil")
	}
}

func TestWithHTTPStatus_PreservesFamily(t *testing.T) {
	wrapped := cqrshtmx.WithHTTPStatus(cqrshtmx.ErrValidationFailed, http.StatusNotFound)
	if errorfamily.Classify(wrapped) != event.Rejection {
		t.Error("WithHTTPStatus must preserve the wrapped error's family")
	}
}

func TestSafeDetail_RedactsServerFaults(t *testing.T) {
	err := errorfamily.NewTransient("db.down", "connection refused at 10.0.0.5:5432")
	detail := cqrshtmx.SafeDetail(err, http.StatusServiceUnavailable, false)
	if detail == "connection refused at 10.0.0.5:5432" {
		t.Error("SafeDetail must redact 5xx internal detail")
	}
	if detail == "" {
		t.Error("SafeDetail must provide a replacement message")
	}
}

func TestSafeDetail_PreservesClientErrors(t *testing.T) {
	err := errorfamily.NewRejection("bad_input", "email is required")
	detail := cqrshtmx.SafeDetail(err, http.StatusBadRequest, false)
	if !strings.Contains(detail, "email is required") {
		t.Errorf("SafeDetail must preserve 4xx detail, got %q", detail)
	}
}

func TestSafeDetail_IncludeInternalOverridesRedaction(t *testing.T) {
	err := errorfamily.NewTransient("db.down", "connection refused")
	detail := cqrshtmx.SafeDetail(err, http.StatusServiceUnavailable, true)
	if !strings.Contains(detail, "connection refused") {
		t.Errorf("SafeDetail with includeInternal must show raw detail, got %q", detail)
	}
}

func TestStructuredError_MetadataForRejection(t *testing.T) {
	se := cqrshtmx.NewStructuredError(cqrshtmx.ErrValidationFailed, nil)
	if se.Message == "" {
		t.Error("StructuredError.Message must be populated")
	}
	if se.Fix == "" {
		t.Error("StructuredError.Fix must be populated for Rejection")
	}
}

func TestStructuredError_MetadataForTransient(t *testing.T) {
	se := cqrshtmx.NewStructuredError(cqrshtmx.ErrDispatchFailed, nil)
	if se.Message == "" {
		t.Error("StructuredError.Message must be populated for Transient")
	}
	if se.Why == "" {
		t.Error("StructuredError.Why must be populated for Transient")
	}
	if se.Fix == "" {
		t.Error("StructuredError.Fix must be populated for Transient")
	}
}

func TestProblemDetailsErrorHandler_ShapeAndContentType(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	cqrshtmx.ProblemDetailsErrorHandler(w, r, cqrshtmx.ErrValidationFailed)

	if ct := w.Header().Get("Content-Type"); ct != cqrshtmx.ContentTypeProblem {
		t.Errorf("Content-Type: got %q, want %q", ct, cqrshtmx.ContentTypeProblem)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var payload cqrshtmx.StructuredError
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("body must be valid StructuredError JSON: %v", err)
	}
	if payload.Type == "" || payload.Title == "" {
		t.Error("payload Type and Title must be non-empty")
	}
}

func TestProblemDetailsErrorHandler_RedactsServerFaults(t *testing.T) {
	err := errorfamily.NewTransient("db.down", "secret internal detail")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	cqrshtmx.ProblemDetailsErrorHandler(w, r, err)

	var payload cqrshtmx.StructuredError
	_ = json.Unmarshal(w.Body.Bytes(), &payload)
	if payload.Detail == "secret internal detail" {
		t.Error("ProblemDetailsErrorHandler must redact 5xx detail")
	}
}

func TestProblemDetailsErrorHandler_AuthRedirectForHTMX(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	cqrshtmx.ProblemDetailsErrorHandler(w, r, cqrshtmx.ErrUnauthorized)

	if w.Code != http.StatusSeeOther {
		t.Errorf("HTMX auth: got %d, want %d", w.Code, http.StatusSeeOther)
	}
	if w.Header().Get("HX-Redirect") == "" {
		t.Error("HTMX auth must set HX-Redirect")
	}
}

func TestProblemDetailsErrorHandler_IncludesCodeField(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	cqrshtmx.ProblemDetailsErrorHandler(w, r, cqrshtmx.ErrValidationFailed)

	var decoded map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}
	if decoded["code"] != "validation_failed" {
		t.Errorf("code field: got %v, want %q", decoded["code"], "validation_failed")
	}
}

func TestProblemDetailsErrorHandler_OmitsCodeWhenAbsent(t *testing.T) {
	err := errors.New("plain unclassified error")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	cqrshtmx.ProblemDetailsErrorHandler(w, r, err)

	var decoded map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &decoded); e != nil {
		t.Fatalf("body must be valid JSON: %v", e)
	}
	if _, hasCode := decoded["code"]; hasCode {
		t.Error("code field should be omitted for errors without a Code() method")
	}
}

func TestConfig_IncludeInternalDetails(t *testing.T) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", erroringCommandHandler("db down"))
	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands:               disp,
		IncludeInternalDetails: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
		newPostRequest("/", `{"email":"t@t.com","name":"T"}`))
	if !strings.Contains(w.Body.String(), "db down") {
		t.Errorf("IncludeInternalDetails=true: expected raw detail, got %q", w.Body.String())
	}
}

func TestConfig_DefaultRedactsInternalDetails(t *testing.T) {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", erroringCommandHandler("db down"))
	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
	})
	if err != nil {
		t.Fatal(err)
	}
	w := serve(app.Command("CreateUser", decodeCreateUserJSON()),
		newPostRequest("/", `{"email":"t@t.com","name":"T"}`))
	if strings.Contains(w.Body.String(), "db down") {
		t.Errorf("default: expected redacted body, got %q", w.Body.String())
	}
}

func TestErrorCode_ReturnsDeepestCodeThroughWrapChain(t *testing.T) {
	// Build a 3-level deep error chain:
	// inner (domain-specific code) → middle (dispatch wrapper) → outer (another wrapper)
	inner := errorfamily.NewRejection("usermgmt.email_exists", "email already registered")
	middle := errorfamily.Wrapf(
		inner, errorfamily.Classify(inner),
		"cqrshtmx.dispatch.command_failed", "dispatch command RegisterUser",
	)
	outer := errorfamily.Wrapf(
		middle, errorfamily.Classify(middle),
		"cqrshtmx.http.request_failed", "request failed",
	)

	tests := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"single error", inner, "usermgmt.email_exists"},
		{"two-level chain", middle, "usermgmt.email_exists"},
		{"three-level chain", outer, "usermgmt.email_exists"},
		{"plain error", errors.New("no code"), ""},
		{"nil error", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cqrshtmx.ErrorCode(tt.err)
			if got != tt.wantCode {
				t.Errorf("ErrorCode() = %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestErrorCode_PrefersDeepestOverOutermost(t *testing.T) {
	// The outermost error has code "outer.code", the innermost has "inner.deep".
	// ErrorCode must return the deepest code, not the first one found.
	inner := errorfamily.NewConflict("inner.deep", "domain conflict")
	outer := errorfamily.Wrapf(inner, errorfamily.Conflict, "outer.code", "wrapped outer")

	got := cqrshtmx.ErrorCode(outer)
	if got != "inner.deep" {
		t.Errorf("ErrorCode() = %q, want %q (deepest code, not outermost)", got, "inner.deep")
	}
}

func TestErrorCode_WithTransientFamily(t *testing.T) {
	inner := errorfamily.NewTransient("db.connection_failed", "db down")
	outer := errorfamily.Wrapf(inner, errorfamily.Transient, "cqrshtmx.dispatch.command_failed", "dispatch failed")

	got := cqrshtmx.ErrorCode(outer)
	if got != "db.connection_failed" {
		t.Errorf("ErrorCode() = %q, want %q", got, "db.connection_failed")
	}
}

func TestErrorRecorder_Standalone(t *testing.T) {
	rec := cqrshtmx.NewErrorRecorder()

	// Initially nil
	if rec.DispatchError() != nil {
		t.Error("DispatchError() should be nil on a fresh recorder")
	}

	// Set an error
	testErr := errorfamily.NewRejection("test.error", "something went wrong")
	rec.SetDispatchError(testErr)

	if !errors.Is(rec.DispatchError(), testErr) {
		t.Error("DispatchError() should return the set error")
	}
}

func TestStatusRecorder_EMBEDSErrorRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	rec := cqrshtmx.NewStatusRecorder(w)

	// ErrorRecorder methods are promoted via embedding
	testErr := errorfamily.NewRejection("test.embedded", "embedded test")
	rec.SetDispatchError(testErr)

	if !errors.Is(rec.DispatchError(), testErr) {
		t.Error("StatusRecorder should promote ErrorRecorder methods via embedding")
	}

	// StatusRecorder still records status
	rec.WriteHeader(http.StatusTeapot)
	if rec.Status() != http.StatusTeapot {
		t.Errorf("Status() = %d, want %d", rec.Status(), http.StatusTeapot)
	}
}
