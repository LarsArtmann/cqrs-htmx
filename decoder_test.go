package cqrshtmx

import (
	"net/http"
	"testing"
)

// TestReadBody_NilBodyDoesNotPanic is the regression test for a nil-pointer
// panic that occurred when a hand-constructed or proxied request had
// r.Body == nil. The guard at the top of readBody (r.Body == nil → return
// nil, nil) must hold for all callers.
func TestReadBody_NilBodyDoesNotPanic(t *testing.T) {
	t.Parallel()

	r := &http.Request{Body: nil}

	body, err := readBody(r, 0)
	if err != nil {
		t.Fatalf("readBody with nil body returned error: %v", err)
	}

	if body != nil {
		t.Fatalf("readBody with nil body should return nil bytes, got %d bytes", len(body))
	}
}

// TestDecodeJSONBody_NilBodyDoesNotPanic verifies the JSON decoder path
// handles a nil body gracefully (treated as empty → zero-value T, no error).
func TestDecodeJSONBody_NilBodyDoesNotPanic(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string
	}

	r := &http.Request{Body: nil}

	result, err := decodeJSONBody[payload](r, 1024)
	if err != nil {
		t.Fatalf("decodeJSONBody with nil body returned error: %v", err)
	}

	if result.Name != "" {
		t.Fatalf("decodeJSONBody with nil body should return zero-value, got Name=%q", result.Name)
	}
}

// TestDecodeFormBody_NilBodyDoesNotPanic verifies the form decoder path
// handles a nil body gracefully (treated as empty → zero-value T, no error).
func TestDecodeFormBody_NilBodyDoesNotPanic(t *testing.T) {
	t.Parallel()

	type formData struct {
		Email string
	}

	r := &http.Request{
		Method: http.MethodPost,
		Header: http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}},
		Body:   nil,
	}

	result, err := decodeFormBody[formData](r, 1024)
	if err != nil {
		t.Fatalf("decodeFormBody with nil body returned error: %v", err)
	}

	if result.Email != "" {
		t.Fatalf("decodeFormBody with nil body should return zero-value, got Email=%q", result.Email)
	}
}
