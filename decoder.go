package cqrshtmx

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/url"
	"strings"

	form "github.com/go-playground/form/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// DefaultMaxBodySize is the default maximum request body size (10 MB).
// Used when neither App.Config.MaxBodySize nor a per-handler WithMaxBodySize is set.
const DefaultMaxBodySize int64 = 10 << 20

// decodeJSONBody decodes JSON from request body into type T.
// If maxBodySize > 0, bodies larger than maxBodySize are rejected with ErrRequestTooLarge.
// An empty body (e.g. GET request with no body) is treated as a zero-value T, not an error.
func decodeJSONBody[T any](r *http.Request, maxBodySize int64) (T, error) {
	var out T

	body, readErr := readBody(r, maxBodySize)
	if readErr != nil {
		return out, errorfamily.Wrapf(readErr, event.Rejection,
			"cqrshtmx.decode.json.read_failed", "maxBodySize=%d", maxBodySize)
	}

	if len(body) == 0 {
		return out, nil // empty body = zero-value T (GET requests, no body)
	}

	if decodeErr := json.Unmarshal(body, &out); decodeErr != nil {
		return out, errorfamily.Wrapf(decodeErr, event.Rejection,
			"cqrshtmx.decode.json.unmarshal_failed", "maxBodySize=%d: decode JSON", maxBodySize)
	}

	return out, nil
}

// readBody reads the request body, respecting maxBodySize if > 0.
// When maxBodySize <= 0, DefaultMaxBodySize is used.
func readBody(r *http.Request, maxBodySize int64) ([]byte, error) {
	if maxBodySize <= 0 {
		maxBodySize = DefaultMaxBodySize
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	closeErr := r.Body.Close()

	if err != nil {
		return nil, errorfamily.Wrapf(err, event.Rejection,
			"cqrshtmx.decode.body.read_failed", "read body (maxBodySize=%d)", maxBodySize)
	}

	if closeErr != nil {
		return nil, errorfamily.Wrapf(closeErr, event.Rejection,
			"cqrshtmx.decode.body.close_failed", "close body (maxBodySize=%d)", maxBodySize)
	}

	if maxBodySize > 0 && int64(len(body)) > maxBodySize {
		return nil, ErrRequestTooLarge
	}

	return body, nil
}

// decodeRequest decodes and maps a request using the provided decoder and mapper.
func decodeRequest[T, R any](
	r *http.Request,
	decode func(*http.Request) (T, error),
	mapper func(T) (R, error),
) (R, error) {
	req, err := decode(r)
	if err != nil {
		var zero R

		return zero, err
	}

	return mapper(req)
}

// decodeFormBody parses form data and decodes into type T.
// If maxBodySize > 0, bodies larger than maxBodySize are rejected with ErrRequestTooLarge.
func decodeFormBody[T any](r *http.Request, maxBodySize int64) (T, error) {
	var out T

	body, readErr := readBody(r, maxBodySize)
	if readErr != nil {
		return out, errorfamily.Wrapf(readErr, event.Rejection,
			"cqrshtmx.decode.form.read_failed", "maxBodySize=%d", maxBodySize)
	}

	// Restore body so ParseForm can read it.
	r.Body = io.NopCloser(bytes.NewReader(body))

	if parseErr := r.ParseForm(); parseErr != nil {
		return out, errorfamily.Wrapf(parseErr, event.Rejection,
			"cqrshtmx.decode.form.parse_failed", "maxBodySize=%d: parse form", maxBodySize)
	}

	if decodeErr := decodeFormValues(r.PostForm, &out); decodeErr != nil {
		return out, errorfamily.Wrapf(decodeErr, event.Rejection,
			"cqrshtmx.decode.form.values_failed", "maxBodySize=%d: decode form values", maxBodySize)
	}

	return out, nil
}

// newFormDecoder creates a form decoder that respects json struct tags for
// backward compatibility with the previous JSON round-trip decoder.
func newFormDecoder() *form.Decoder {
	d := form.NewDecoder()
	d.SetTagName("json")

	return d
}

func decodeFormValues(form url.Values, target any) error {
	// Normalize form keys to lowercase for case-insensitive matching,
	// preserving the previous JSON round-trip's lenient field matching.
	normalized := make(url.Values, len(form))
	for k, v := range form {
		normalized[strings.ToLower(k)] = v
	}

	if err := newFormDecoder().Decode(target, normalized); err != nil {
		return errorfamily.Wrapf(err, event.Rejection,
			"cqrshtmx.decode.form.decode_failed", "decode form values for target=%T", target)
	}

	return nil
}
