package cqrshtmx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
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
	if event.Classify(wrapped) != event.Rejection {
		t.Error("WithHTTPStatus must preserve the wrapped error's family")
	}
}

func TestSafeDetail_RedactsServerFaults(t *testing.T) {
	err := event.NewTransient("db.down", "connection refused at 10.0.0.5:5432")
	detail := cqrshtmx.SafeDetail(err, http.StatusServiceUnavailable, false)
	if detail == "connection refused at 10.0.0.5:5432" {
		t.Error("SafeDetail must redact 5xx internal detail")
	}
	if detail == "" {
		t.Error("SafeDetail must provide a replacement message")
	}
}

func TestSafeDetail_PreservesClientErrors(t *testing.T) {
	err := event.NewRejection("bad_input", "email is required")
	detail := cqrshtmx.SafeDetail(err, http.StatusBadRequest, false)
	if !strings.Contains(detail, "email is required") {
		t.Errorf("SafeDetail must preserve 4xx detail, got %q", detail)
	}
}

func TestSafeDetail_IncludeInternalOverridesRedaction(t *testing.T) {
	err := event.NewTransient("db.down", "connection refused")
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
	err := event.NewTransient("db.down", "secret internal detail")
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
