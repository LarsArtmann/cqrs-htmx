package cqrshtmx

import (
	"errors"
	"net/http"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// stubCarrier is a minimal HTTPStatusCarrier for exercising carrierStatus in
// isolation, without coupling to errorfamily internals. A status of 0 means
// "not set", mirroring how *errorfamily.Error behaves when no override is
// pinned (the root cause of the go-error-family v0.8.0 carrierStatus bug).
type stubCarrier struct {
	status int
	cause  error
}

func (e *stubCarrier) Error() string   { return "stub carrier" }
func (e *stubCarrier) Unwrap() error   { return e.cause }
func (e *stubCarrier) HTTPStatus() int { return e.status }

// TestCarrierStatus is the dedicated regression guard for the carrierStatus
// bug: go-error-family v0.8.0 added HTTPStatus() int to *errorfamily.Error
// (returning 0 when unset), which made every errorfamily error satisfy
// HTTPStatusCarrier. carrierStatus must treat that 0 as "not set" so MapError
// falls through to the family-based default (e.g. Rejection -> 400) instead of
// short-circuiting to 500.
func TestCarrierStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "nil error is not a carrier",
			err:        nil,
			wantStatus: 0,
			wantOK:     false,
		},
		{
			name:       "plain non-carrier error",
			err:        errors.New("boom"),
			wantStatus: 0,
			wantOK:     false,
		},
		{
			// The core bug: a carrier whose HTTPStatus()==0 means "not set".
			name:       "zero status carrier means not set",
			err:        &stubCarrier{status: 0, cause: nil},
			wantStatus: 0,
			wantOK:     false,
		},
		{
			name:       "valid explicit status is returned",
			err:        &stubCarrier{status: http.StatusTeapot, cause: nil},
			wantStatus: http.StatusTeapot,
			wantOK:     true,
		},
		{
			// Non-zero but implausible status degrades to 500 rather than
			// leaking a nonsensical code.
			name:       "out-of-range status falls back to 500",
			err:        &stubCarrier{status: 999, cause: nil},
			wantStatus: http.StatusInternalServerError,
			wantOK:     true,
		},
		{
			// Chain-walking: skip a zero-status carrier and keep searching for
			// a real override deeper in the chain (e.g. WithHTTPStatus wrapping
			// a bare errorfamily.Error whose HTTPStatus()==0).
			name:       "zero carrier unwraps to a deeper override",
			err:        &stubCarrier{status: 0, cause: &stubCarrier{status: http.StatusNotFound, cause: nil}},
			wantStatus: http.StatusNotFound,
			wantOK:     true,
		},
		{
			// Integration with the real triggering type: a bare errorfamily
			// Rejection has HTTPStatus()==0, so it must NOT be treated as a
			// carrier and must fall through to the family default.
			name:       "bare errorfamily Rejection is not treated as a carrier",
			err:        errorfamily.NewRejection("bad_input", "email is required"),
			wantStatus: 0,
			wantOK:     false,
		},
		{
			// WithHTTPStatus pins a concrete status; that IS recognised as a
			// carrier even when wrapping a zero-status errorfamily error.
			name:       "WithHTTPStatus override is recognised",
			err:        WithHTTPStatus(errorfamily.NewRejection("bad_input", "nope"), http.StatusNotFound),
			wantStatus: http.StatusNotFound,
			wantOK:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotStatus, gotOK := carrierStatus(tc.err)
			if gotStatus != tc.wantStatus || gotOK != tc.wantOK {
				t.Errorf("carrierStatus(%v) = (%d, %t), want (%d, %t)",
					tc.err, gotStatus, gotOK, tc.wantStatus, tc.wantOK)
			}
		})
	}
}
