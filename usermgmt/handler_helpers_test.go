package usermgmt

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMux(t *testing.T) (*Service, *http.ServeMux) {
	t.Helper()
	svc := newTestServiceWithAuthz(t)
	h := NewAuthHandler(svc, HandlerConfig{Secure: new(bool)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return svc, mux
}

func registerUser(t *testing.T, mux *http.ServeMux) *RegisterResponse {
	t.Helper()
	w := postJSON(t, mux, "/auth/register",
		fmt.Sprintf(`{"id":%q,"email":"a@b.com","display_name":"Test"}`, NewUserID("u1").Get().String()))
	if w.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeJSON[RegisterResponse](t, w)
	return &resp
}

func assertCookie(
	t *testing.T,
	w *httptest.ResponseRecorder,
	name string,
	check func(*http.Cookie) bool,
) {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name && check(c) {
			return
		}
	}
	t.Errorf("expected cookie %q matching condition", name)
}

func assertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("expected %d, got %d: %s", expected, w.Code, w.Body.String())
	}
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return result
}

// newJar returns a cookie jar for real-client flow tests (the HTTP-level
// equivalent of a browser's credentials: "same-origin" fetch posture).
func newJar(t *testing.T) http.CookieJar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return jar
}

// postJSONTo posts a JSON body via a real client (cookie jar attached) and
// returns the undrained response.
func postJSONTo(t *testing.T, client *http.Client, url, body string) (*http.Response, error) {
	t.Helper()
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp, nil
}

// decodeJSONBody decodes a live *http.Response body into target.
func decodeJSONBody(t *testing.T, resp *http.Response, target any) error {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
