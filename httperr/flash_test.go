package httperr_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	errorfamily "github.com/larsartmann/go-error-family"
)

func htmxRequest() *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/items", nil)

	r.Header.Set(htmx.HeaderRequest, "true")

	return r
}

func TestFlashErrorHandler_HTMXPathReturns200WithFlashHeaders(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	err := errorfamily.NewRejection("test.rejection", "name may not be empty")

	httperr.FlashErrorHandler("#flash")(w, htmxRequest(), err)

	if w.Code != http.StatusOK {
		t.Fatalf("HTMX path: status = %d, want 200 (the swap rides the success path)", w.Code)
	}

	if got := w.Header().Get("Hx-Retarget"); got != "#flash" {
		t.Errorf("Hx-Retarget = %q, want %q", got, "#flash")
	}

	if got := w.Header().Get("Hx-Reswap"); got != "innerHTML" {
		t.Errorf("Hx-Reswap = %q, want %q", got, "innerHTML")
	}

	if body := w.Body.String(); !strings.Contains(body, "name may not be empty") {
		t.Errorf("flash body = %q, want it to carry the client-fault detail", body)
	}
}

func TestFlashErrorHandler_EmptyTargetDefaultsToFlashRegion(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()

	httperr.FlashErrorHandler("")(w, htmxRequest(), errorfamily.NewRejection("test.rejection", "boom"))

	if got := w.Header().Get("Hx-Retarget"); got != httperr.DefaultFlashTarget {
		t.Errorf("Hx-Retarget = %q, want default %q", got, httperr.DefaultFlashTarget)
	}
}

func TestFlashErrorHandler_NonHTMXGetsRealStatus(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	err := errorfamily.NewRejection("test.rejection", "name may not be empty")

	httperr.FlashErrorHandler("#flash")(w, httptest.NewRequest(http.MethodPost, "/items", nil), err)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("non-HTMX status = %d, want 400 (real mapped status)", w.Code)
	}

	if got := w.Header().Get("Hx-Retarget"); got != "" {
		t.Errorf("non-HTMX path must not set Hx-Retarget, got %q", got)
	}

	if got := w.Header().Get("Hx-Reswap"); got != "" {
		t.Errorf("non-HTMX path must not set Hx-Reswap, got %q", got)
	}
}

func TestFlashErrorHandler_NonHTMXConflictIs409(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()

	httperr.FlashErrorHandler("#flash")(
		w,
		httptest.NewRequest(http.MethodPost, "/items", nil),
		errorfamily.NewConflict("test.conflict", "battle exists"),
	)

	if w.Code != http.StatusConflict {
		t.Fatalf("non-HTMX status = %d, want 409", w.Code)
	}
}

func TestFlashErrorHandler_ServerFaultDetailIsRedactedOnBothPaths(t *testing.T) {
	t.Parallel()

	err := errorfamily.NewInfrastructure(
		"test.infra",
		"postgres://ops:secret@db.internal:5432/prod: connection refused",
	)

	flash := httptest.NewRecorder()

	httperr.FlashErrorHandler("#flash")(flash, htmxRequest(), err)

	if flash.Code != http.StatusOK {
		t.Fatalf("flash status = %d, want 200", flash.Code)
	}

	if body := flash.Body.String(); strings.Contains(body, "postgres://") || strings.Contains(body, "secret") {
		t.Errorf("flash body leaked server-fault detail: %q", body)
	}

	plain := httptest.NewRecorder()

	httperr.FlashErrorHandler("#flash")(plain, httptest.NewRequest(http.MethodPost, "/items", nil), err)

	if plain.Code != http.StatusServiceUnavailable {
		t.Fatalf("non-HTMX status = %d, want 503", plain.Code)
	}

	if body := plain.Body.String(); strings.Contains(body, "postgres://") || strings.Contains(body, "secret") {
		t.Errorf("plain body leaked server-fault detail: %q", body)
	}
}

func TestFlashErrorHandler_HeaderConstantsAreTheWireNames(t *testing.T) {
	t.Parallel()

	// Byte-exactness: the handler must write the typed constants' wire names,
	// not hand-written strings that could drift from the htmx vocabulary.
	if htmx.HeaderRetarget != "Hx-Retarget" {
		t.Fatalf("htmx.HeaderRetarget = %q, want the canonical wire name", htmx.HeaderRetarget)
	}

	if htmx.HeaderReswap != "Hx-Reswap" {
		t.Fatalf("htmx.HeaderReswap = %q, want the canonical wire name", htmx.HeaderReswap)
	}

	w := httptest.NewRecorder()

	httperr.FlashErrorHandler("#ops-flash")(w, htmxRequest(), errorfamily.NewRejection("test.rejection", "nope"))

	if got := w.Header().Get(htmx.HeaderRetarget); got != "#ops-flash" {
		t.Errorf("header via constant = %q, want %q", got, "#ops-flash")
	}
}
