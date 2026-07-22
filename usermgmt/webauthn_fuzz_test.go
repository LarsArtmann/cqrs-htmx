package usermgmt

import (
	"encoding/json/jsontext"
	"testing"
)

// FuzzMarshalWebAuthnUser verifies that marshalWebAuthnUser never panics
// on arbitrary User inputs and always produces valid JSON (or an error).
func FuzzMarshalWebAuthnUser(f *testing.F) {
	f.Add("test@test.com")
	f.Add("")
	f.Add(string(make([]byte, 1024)))

	f.Fuzz(func(t *testing.T, email string) {
		user := &User{
			ID:    NewUserID("01HXKYGEG0QH8XJYQKZ3R8WZAA"),
			Email: email,
			Credentials: []WebAuthnCredential{
				{credentialCore: credentialCore{Name: "test"}},
			},
		}
		data, err := marshalWebAuthnUser(user)
		if err != nil {
			return
		}
		if !jsontext.Value(data).IsValid() {
			t.Errorf("marshalWebAuthnUser produced invalid JSON: %s", string(data))
		}
	})
}
