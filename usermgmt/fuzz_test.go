package usermgmt

import (
	"strings"
	"testing"
)

func FuzzRegisterRequest_Validate(f *testing.F) {
	f.Add("u1", "test@example.com", "Alice")
	f.Add("", "", "")
	f.Add("u2", "not-an-email", strings.Repeat("x", 101))
	f.Add("u3", "a@b.com", "")
	f.Add("u4", "valid@email.com", "OK")

	f.Fuzz(func(_ *testing.T, id, email, displayName string) {
		req := RegisterRequest{
			ID:          NewUserID(id),
			Email:       email,
			DisplayName: displayName,
		}
		_ = req.Validate()
	})
}
