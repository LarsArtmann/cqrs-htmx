package cqrshtmx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
