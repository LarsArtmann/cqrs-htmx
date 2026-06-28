package cqrshtmx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	form "github.com/go-playground/form/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// DefaultMaxBodySize is the default maximum request body size (10 MB).
// Used when neither App.Config.MaxBodySize nor a per-handler WithMaxBodySize is set.
const DefaultMaxBodySize int64 = 10 << 20

// decodeJSONBody decodes JSON from request body into type T.
// If maxBodySize > 0, bodies larger than maxBodySize are rejected with ErrRequestTooLarge.
func decodeJSONBody[T any](r *http.Request, maxBodySize int64) (out T, err error) {
	body, readErr := readBody(r, maxBodySize)
	if readErr != nil {
		return out, event.Wrapf(readErr, event.Rejection,
			"cqrshtmx.decode.json.read_failed", "maxBodySize=%d", maxBodySize)
	}

	if decodeErr := json.Unmarshal(body, &out); decodeErr != nil {
		return out, event.Wrapf(decodeErr, event.Rejection,
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
		return nil, event.Wrapf(err, event.Rejection,
			"cqrshtmx.decode.body.read_failed", "read body (maxBodySize=%d)", maxBodySize)
	}
	if closeErr != nil {
		return nil, event.Wrapf(closeErr, event.Rejection,
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
func decodeFormBody[T any](r *http.Request, maxBodySize int64) (out T, err error) {
	body, readErr := readBody(r, maxBodySize)
	if readErr != nil {
		return out, event.Wrapf(readErr, event.Rejection,
			"cqrshtmx.decode.form.read_failed", "maxBodySize=%d", maxBodySize)
	}

	// Restore body so ParseForm can read it.
	r.Body = io.NopCloser(bytes.NewReader(body))

	if parseErr := r.ParseForm(); parseErr != nil {
		return out, event.Wrapf(parseErr, event.Rejection,
			"cqrshtmx.decode.form.parse_failed", "maxBodySize=%d: parse form", maxBodySize)
	}

	if decodeErr := decodeFormValues(r.PostForm, &out); decodeErr != nil {
		return out, event.Wrapf(decodeErr, event.Rejection,
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
		return event.Wrapf(err, event.Rejection,
			"cqrshtmx.decode.form.decode_failed", "decode form values for target=%T", target)
	}
	return nil
}
