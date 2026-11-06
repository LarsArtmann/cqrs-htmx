package httperr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// anInfrastructureError carries internal detail an attacker must never see.
func anInfrastructureError() error {
	return errorfamily.Newf(errorfamily.Infrastructure, "test.db_unreachable",
		"dial tcp 10.0.0.7:5432: connection refused (pool exhausted, 42 conns)")
}

func TestSafeDetailFor_AudienceControlsServerFaultDetail(t *testing.T) {
	t.Parallel()

	err := anInfrastructureError()

	public := SafeDetailFor(err, http.StatusServiceUnavailable, AudiencePublic)
	if public == err.Error() {
		t.Errorf("public audience leaked the raw 5xx detail: %q", public)
	}

	if !strings.Contains(public, errorfamily.Classify(err).DefaultMessage()) {
		t.Errorf("public detail %q is not the family default message", public)
	}

	operator := SafeDetailFor(err, http.StatusServiceUnavailable, AudienceOperator)
	if operator != err.Error() {
		t.Errorf("operator detail = %q, want the raw error text", operator)
	}
}

func TestSafeDetailFor_ClientFaultIdenticalForBothAudiences(t *testing.T) {
	t.Parallel()

	err := errorfamily.NewRejection("test.email_invalid", "email is not a valid address")

	public := SafeDetailFor(err, http.StatusBadRequest, AudiencePublic)
	operator := SafeDetailFor(err, http.StatusBadRequest, AudienceOperator)

	if public != err.Error() || operator != err.Error() {
		t.Errorf("client-fault detail must be the raw text for both audiences: public=%q operator=%q",
			public, operator)
	}
}

func TestSafeDetail_DelegatesToAudience(t *testing.T) {
	t.Parallel()

	err := anInfrastructureError()

	if SafeDetail(err, 503, true) != err.Error() {
		t.Error("includeInternal=true must behave like AudienceOperator")
	}

	if SafeDetail(err, 503, false) == err.Error() {
		t.Error("includeInternal=false must behave like AudiencePublic")
	}
}

// The RFC 9457 body: the SAME server fault renders redacted for a public
// reader and actionable for an operator — including the machine context on
// 5xx, which public readers never receive.
func TestProblemDetailsAudienceHandler_RFC9457Body(t *testing.T) {
	t.Parallel()

	err := errorfamily.New(errorfamily.Infrastructure, "test.db_unreachable",
		"dial tcp 10.0.0.7:5432: connection refused").
		WithContextMap(map[string]string{"pool_size": "42", "db": "users"})

	render := func(t *testing.T, audience Audience) StructuredError {
		t.Helper()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/things", nil)

		ProblemDetailsAudienceErrorHandler(func(*http.Request) Audience { return audience }, "")(
			rec, req, err)

		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
			t.Fatalf("Content-Type = %q, want application/problem+json", ct)
		}

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}

		var payload StructuredError
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal problem body: %v\nbody: %s", err, rec.Body.String())
		}

		return payload
	}

	t.Run("public", func(t *testing.T) {
		t.Parallel()

		payload := render(t, AudiencePublic)
		if strings.Contains(payload.Detail, "10.0.0.7") {
			t.Errorf("public body leaked the internal address: %q", payload.Detail)
		}

		if payload.Context != nil {
			t.Errorf("public body leaked 5xx machine context: %v", payload.Context)
		}
	})

	t.Run("operator", func(t *testing.T) {
		t.Parallel()

		payload := render(t, AudienceOperator)
		if !strings.Contains(payload.Detail, "10.0.0.7") {
			t.Errorf("operator detail lost the internal address: %q", payload.Detail)
		}

		if payload.Context["pool_size"] != "42" || payload.Context["db"] != "users" {
			t.Errorf("operator body lost the 5xx machine context: %v", payload.Context)
		}
	})
}

func TestProblemDetailsAudienceHandler_NilResolverFailsPublic(t *testing.T) {
	t.Parallel()

	err := anInfrastructureError()

	rec := httptest.NewRecorder()
	ProblemDetailsAudienceErrorHandler(nil, "")(rec, httptest.NewRequest(http.MethodGet, "/", nil), err)

	if strings.Contains(rec.Body.String(), "10.0.0.7") {
		t.Errorf("nil resolver must fail safe (public): body leaked internals: %s", rec.Body.String())
	}
}
