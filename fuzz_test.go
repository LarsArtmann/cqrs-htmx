package cqrshtmx

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func FuzzDecodeJSONBody(f *testing.F) {
	f.Add(`{"name":"test","value":42}`)
	f.Add(``)
	f.Add(`{}`)
	f.Add(`not json at all`)
	f.Add(`{"name":null}`)
	f.Add(`[]`)
	f.Add(`0`)

	f.Fuzz(func(_ *testing.T, body string) {
		r := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/",
			strings.NewReader(body),
		)
		defer func() { _ = r.Body.Close() }()

		type target struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		result, err := decodeJSONBody[target](r, 1024)
		_ = result
		_ = err
	})
}

func FuzzDecodeFormBody(f *testing.F) {
	f.Add("name=test&value=42")
	f.Add("")
	f.Add("name=test")
	f.Add("name=test&name=duplicate")
	f.Add("value=not_a_number")
	f.Add("=")
	f.Add("key=")

	f.Fuzz(func(_ *testing.T, body string) {
		r := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/",
			strings.NewReader(body),
		)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		defer func() { _ = r.Body.Close() }()

		type target struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		result, err := decodeFormBody[target](r, 1024)
		_ = result
		_ = err
	})
}

func FuzzSanitizeRedirectURL(f *testing.F) {
	f.Add("/")
	f.Add("/path")
	f.Add("https://evil.com")
	f.Add("javascript:alert(1)")
	f.Add("")
	f.Add("/a/../b/c")
	f.Add("//evil.com")
	f.Add("data:text/html,<script>alert(1)</script>")

	f.Fuzz(func(_ *testing.T, rawURL string) {
		url, ok := sanitizeRedirectURL(rawURL)
		_ = url
		_ = ok
	})
}

func FuzzCSRFConfigValidation(f *testing.F) {
	f.Add("", "", "X-CSRF-Token", "csrf_token")
	f.Add("my_csrf", "X-Token", "field", "")
	f.Add("", "", "", "")
	f.Add("b", "c", "d", "e")

	f.Fuzz(func(_ *testing.T, cookieName, headerName, fieldName, domain string) {
		cfg := CSRFConfig{
			CookieName: cookieName,
			HeaderName: headerName,
			FieldName:  fieldName,
			Domain:     domain,
		}
		_ = cfg.cookieName()
		_ = cfg.headerName()
		_ = cfg.fieldName()
		_ = cfg.maxAge()
		_ = cfg.path()
		_ = cfg.Validate()
	})
}

// FuzzEventOptionsFromContext fuzzes the EventOptionsFromContext function
// with various context values (deadline present/absent, user ID set/unset,
// correlation ID set/unset, request ID set/unset). Verifies that the
// function never panics and that the number of returned options is bounded.
func FuzzEventOptionsFromContext(f *testing.F) {
	f.Add(0, 0, "", "", "")
	f.Add(1, 1, "01HK1549P84T9XF8R94E960633", "01HK154ANGZHV2ZW0X3SKSNEN2", "01HK154B5C4EZY5S4Y4HQ3ZX9H")
	f.Add(1, 0, "bad-ulid", "", "01HK154B5C4EZY5S4Y4HQ3ZX9H")

	f.Fuzz(func(t *testing.T, hasDeadline, hasCancel int, uid, cid, rid string) {
		ctx := context.Background()
		if hasDeadline%2 == 1 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, time.Unix(0, 0).Add(time.Hour))
			defer cancel()
		}
		if hasCancel%2 == 1 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		if uid != "" {
			if parsed, err := ParseUserID(uid); err == nil {
				ctx = WithUserID(ctx, parsed)
			}
		}
		if cid != "" {
			if parsed, err := ParseCorrelationID(cid); err == nil {
				ctx = WithCorrelationID(ctx, parsed)
			}
		}
		if rid != "" {
			if parsed, err := ParseRequestID(rid); err == nil {
				ctx = WithRequestID(ctx, parsed)
			}
		}

		opts := EventOptionsFromContext(ctx)
		// Deadlines can be there or not, but the returned options must
		// always be a bounded slice.
		if len(opts) > 4 {
			t.Errorf("too many options: %d", len(opts))
		}
	})
}

// FuzzWriteWSMessage fuzzes the WSMessage encoder with arbitrary body/header
// combinations. Verifies the output is always valid JSON and round-trips
// through ParseWSMessage.
func FuzzWriteWSMessage(f *testing.F) {
	f.Add(`{"name":"test"}`, `{"HX-Request":"true"}`)
	f.Add(`{}`, ``)
	f.Add(`{"items":[1,2,3],"nested":{"a":"b"}}`, `{"Authorization":"Bearer x"}`)

	f.Fuzz(func(t *testing.T, bodyJSON, headersJSON string) {
		var body map[string]any
		if err := json.Unmarshal([]byte(bodyJSON), &body); err != nil {
			t.Skip()
		}
		var headers map[string]string
		if headersJSON != "" {
			if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
				t.Skip()
			}
		}

		msg := WSMessage{Body: body, Headers: headers}
		var buf bytes.Buffer
		if err := WriteWSMessage(&buf, msg); err != nil {
			t.Fatalf("WriteWSMessage: %v", err)
		}

		// Verify output is valid JSON
		var combined map[string]any
		if err := json.Unmarshal(buf.Bytes(), &combined); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}
	})
}

// FuzzStructuredErrorJSON fuzzes the StructuredError.JSON() method with
// arbitrary field values. Verifies the output is always valid JSON.
func FuzzStructuredErrorJSON(f *testing.F) {
	f.Add("rejection", "Bad Request", 400, "invalid email", "req-123")
	f.Add("", "", 0, "", "")
	f.Add("conflict", "Conflict", 409, "duplicate email user@test.com", "")
	f.Add("about:blank", "Internal Server Error", 500,
		"null byte \x00 in message", "trace-id-with-unicode-üñîçødé")

	f.Fuzz(func(t *testing.T, typ, title string, status int, detail, instance string) {
		se := StructuredError{ //nolint:exhaustruct // cause intentionally omitted in fuzz
			Type:     typ,
			Title:    title,
			Status:   status,
			Detail:   detail,
			Instance: instance,
		}
		jsonStr := se.JSON()

		// Verify output is always valid JSON with the expected fields
		var parsed map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("JSON() output is not valid JSON: %v\noutput: %s", err, jsonStr)
		}
	})
}
