package usermgmt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestWriteDispatchError_IncludesCodeField(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/register", nil)

	writeDispatchError(w, r, ErrValidation)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}
	if body[errorKey] != ErrValidation.Error() {
		t.Errorf("error message: got %q, want %q", body[errorKey], ErrValidation.Error())
	}
	if body[codeKey] != "usermgmt.validation" {
		t.Errorf("code: got %q, want %q", body[codeKey], "usermgmt.validation")
	}
}

func TestWriteDispatchError_IncludesRequestID(t *testing.T) {
	rid := cqrshtmx.NewRequestID()
	ctx := cqrshtmx.WithRequestID(context.Background(), rid)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/register", nil).WithContext(ctx)

	writeDispatchError(w, r, ErrValidation)

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}
	if body[requestIDKey] != rid.String() {
		t.Errorf("request_id: got %q, want %q", body[requestIDKey], rid.String())
	}
}

func TestWriteDispatchError_DerivesConflictStatus(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/register", nil)

	writeDispatchError(w, r, ErrEmailExists)

	if w.Code != http.StatusConflict {
		t.Errorf("status for conflict error: got %d, want %d", w.Code, http.StatusConflict)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}
	if body[codeKey] != "usermgmt.email_exists" {
		t.Errorf("code: got %q, want %q", body[codeKey], "usermgmt.email_exists")
	}
}

func TestWriteDispatchError_NilRequestOmitsRequestID(t *testing.T) {
	w := httptest.NewRecorder()

	writeDispatchError(w, nil, ErrValidation)

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body must be valid JSON: %v", err)
	}
	if _, hasRID := body[requestIDKey]; hasRID {
		t.Error("request_id should be omitted when request is nil")
	}
	if body[codeKey] != "usermgmt.validation" {
		t.Errorf("code: got %q, want %q", body[codeKey], "usermgmt.validation")
	}
}
