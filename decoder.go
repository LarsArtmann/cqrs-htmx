package cqrshtmx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// decodeJSONBody decodes JSON from request body into type T.
func decodeJSONBody[T any](r *http.Request) (out T, err error) {
	if decodeErr := json.NewDecoder(r.Body).Decode(&out); decodeErr != nil {
		return out, fmt.Errorf("%w: decode JSON: %w", ErrDecodeFailed, decodeErr)
	}

	return out, nil
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
func decodeFormBody[T any](r *http.Request) (out T, err error) {
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
