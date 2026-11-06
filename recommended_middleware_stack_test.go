package cqrshtmx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/httputil"
)

var noncePattern = regexp.MustCompile(`nonce-[A-Za-z0-9+/=_-]+`)

// canonicalHeaders records the security-relevant response headers of one
// request through a chain, normalizing the per-request nonce out of the CSP
// header (the nonce differs per request by design).
func canonicalHeaders(t *testing.T, chain func(http.Handler) http.Handler) map[string]string {
	t.Helper()

	rec := httptest.NewRecorder()
	chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	headers := map[string]string{}
	for name := range rec.Header() {
		headers[name] = rec.Header().Get(name)
	}

	return headers
}

// The stack form and the classic middleware form of the recommended security
// baseline are behaviorally equivalent: identical header sets and values on
// the normal path (the nonce differing per request by design, normalized
// here).
func TestRecommendedSecurityStack_EquivalentToMiddleware(t *testing.T) {
	stack, err := cqrshtmx.RecommendedSecurityStack()
	if err != nil {
		t.Fatalf("RecommendedSecurityStack: %v", err)
	}

	fromStack := canonicalHeaders(t, stack.Middleware())
	fromClassic := canonicalHeaders(t, cqrshtmx.RecommendedSecurityMiddleware())

	if len(fromStack) != len(fromClassic) {
		t.Fatalf("header set differs: stack has %d (%v), classic has %d (%v)",
			len(fromStack), keysOf(fromStack), len(fromClassic), keysOf(fromClassic))
	}

	for name, classicValue := range fromClassic {
		stackValue, ok := fromStack[name]
		if !ok {
			t.Errorf("stack form lost header %q (classic: %q)", name, classicValue)
			continue
		}
		if name == "Content-Security-Policy" {
			// Both carry a per-request nonce; compare the nonce-stripped rest.
			if stripNonce(classicValue) != stripNonce(stackValue) {
				t.Errorf("CSP differs beyond the nonce:\nclassic: %s\nstack:   %s", classicValue, stackValue)
			}
			continue
		}
		if stackValue != classicValue {
			t.Errorf("header %q: stack %q != classic %q", name, stackValue, classicValue)
		}
	}

	for _, want := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Content-Security-Policy",
	} {
		if _, ok := fromStack[want]; !ok {
			t.Errorf("stack form is missing the baseline header %q", want)
		}
	}
}

// The recommended stack enforces the fleet ordering contract: recovery is the
// outermost entry, and the stack stays valid after consumer extensions.
func TestRecommendedSecurityStack_RecoveryFirstAndExtendable(t *testing.T) {
	stack, err := cqrshtmx.RecommendedSecurityStack()
	if err != nil {
		t.Fatalf("RecommendedSecurityStack: %v", err)
	}

	names := stack.Names()
	if names[0] != httputil.MiddlewareRecovery {
		t.Errorf(
			"outermost entry = %q, want %q (recovery must catch panics from all layers)",
			names[0],
			httputil.MiddlewareRecovery,
		)
	}
	if !slices.Contains(names, httputil.MiddlewareSecurityHeaders) ||
		!slices.Contains(names, httputil.MiddlewareNonce) {
		t.Errorf("baseline entries incomplete: %v", names)
	}

	inner := func(next http.Handler) http.Handler { return next }
	if err := stack.Add("session", inner); err != nil {
		t.Fatalf("Add(session): %v", err)
	}
	if err := stack.Validate(); err != nil {
		t.Errorf("Validate after extension: %v", err)
	}
}

// Consumers extending the recommended stack inherit duplicate-name detection:
// adding a second middleware under an existing name (the copy-paste
// duplication class) fails at construction with a Rejection carrying the
// conflicting name.
func TestRecommendedSecurityStack_RejectsDuplicateNames(t *testing.T) {
	stack, err := cqrshtmx.RecommendedSecurityStack()
	if err != nil {
		t.Fatalf("RecommendedSecurityStack: %v", err)
	}

	inner := func(next http.Handler) http.Handler { return next }
	if err := stack.Add(httputil.MiddlewareCSRF, inner); err != nil {
		t.Fatalf("first Add(csrf): %v", err)
	}

	second := stack.Add(httputil.MiddlewareCSRF, inner)
	if second == nil {
		t.Fatal("duplicate Add succeeded — duplicate detection did not fire")
	}
	if errorfamily.Classify(second) != errorfamily.Rejection {
		t.Errorf("duplicate Add family = %v, want Rejection", errorfamily.Classify(second))
	}
	if contextual, ok := errors.AsType[errorfamily.Contextual](second); ok {
		if name := contextual.ErrorContext()["name"]; name != httputil.MiddlewareCSRF {
			t.Errorf("duplicate Add context name = %q, want %q", name, httputil.MiddlewareCSRF)
		}
	} else {
		t.Error("duplicate Add error carries no context — the conflicting name should ride as context")
	}

	// A duplicate of the BASELINE names is caught the same way — the class
	// where a consumer wraps the recommended chain in their own recovery.
	if err := stack.Add(httputil.MiddlewareRecovery, cqrshtmx.RecoveryMiddleware); err == nil {
		t.Fatal("duplicate Add(recovery) succeeded — baseline names must be reserved")
	}
}

func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}

func stripNonce(csp string) string {
	return noncePattern.ReplaceAllString(csp, "nonce-X")
}
