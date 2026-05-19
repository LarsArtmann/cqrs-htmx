package cqrshtmx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// decodeJSONBody decodes JSON from request body into type T.
// If maxBodySize > 0, bodies larger than maxBodySize are rejected with ErrRequestTooLarge.
func decodeJSONBody[T any](r *http.Request, maxBodySize int64) (out T, err error) {
	body, readErr := readBody(r, maxBodySize)
	if readErr != nil {
		return out, readErr
	}

	if decodeErr := json.Unmarshal(body, &out); decodeErr != nil {
		return out, fmt.Errorf("%w: decode JSON: %w", ErrDecodeFailed, decodeErr)
	}

	return out, nil
}

// readBody reads the request body, respecting maxBodySize if > 0.
func readBody(r *http.Request, maxBodySize int64) ([]byte, error) {
	var body []byte
	var err error

	if maxBodySize > 0 {
		body, err = io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	} else {
		body, err = io.ReadAll(r.Body)
	}
	_ = r.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrDecodeFailed, err)
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
		return out, readErr
	}

	// Restore body so ParseForm can read it.
	r.Body = io.NopCloser(bytes.NewReader(body))

	if parseErr := r.ParseForm(); parseErr != nil {
		return out, fmt.Errorf("%w: parse form: %w", ErrDecodeFailed, parseErr)
	}

	if decodeErr := decodeFormValues(r.Form, &out); decodeErr != nil {
		return out, fmt.Errorf("%w: decode form values: %w", ErrDecodeFailed, decodeErr)
	}

	return out, nil
}

func decodeFormValues(form url.Values, target any) error {
	jsonMap := make(map[string]any, len(form))
	for key, values := range form {
		if len(values) == 1 {
			jsonMap[key] = values[0]
		} else {
			jsonMap[key] = values
		}
	}

	encoded, err := json.Marshal(jsonMap)
	if err != nil {
		return fmt.Errorf("%w: marshal form values for keys=%v: %w", ErrDecodeFailed, form, err)
	}

	if err := json.Unmarshal(encoded, target); err != nil {
		return fmt.Errorf(
			"%w: unmarshal form values for target=%T: %w",
			ErrDecodeFailed, target, err)
	}

	return nil
}
