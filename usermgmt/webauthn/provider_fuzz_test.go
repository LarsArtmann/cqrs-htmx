package webauthn

import (
	"encoding/json"
	"testing"
)

// FuzzParseUser verifies that parseUser never panics on arbitrary JSON input
// and either returns valid data or an error.
func FuzzParseUser(f *testing.F) {
	f.Add([]byte(`{"id":"test","email":"a@b.com","display_name":"Test","credentials":[]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"id":"` + string(make([]byte, 512)) + `"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		user, err := parseUser(data)
		if err != nil {
			return
		}
		_ = user
	})
}

// FuzzParseSession verifies that parseSession never panics on arbitrary input.
func FuzzParseSession(f *testing.F) {
	f.Add([]byte(`{"challenge":"dGVzdA==","user_id":"dGVzdA==","expires":"2099-01-01T00:00:00Z"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`not json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		session, err := parseSession(data)
		if err != nil {
			return
		}
		roundtrip, err := json.Marshal(session)
		if err != nil {
			t.Errorf("failed to marshal parsed session: %v", err)
		}
		_ = roundtrip
	})
}
