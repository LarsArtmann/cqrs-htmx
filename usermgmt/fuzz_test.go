package usermgmt

import (
	"strings"
	"testing"
)

func FuzzRegisterRequest_Validate(f *testing.F) {
	f.Add("u1", "test@example.com", "secret12", "Alice")
	f.Add("", "", "", "")
	f.Add("u2", "not-an-email", "short", strings.Repeat("x", 101))
	f.Add("u3", "a@b.com", strings.Repeat("x", 129), "")
	f.Add("u4", "valid@email.com", "exactly8", "OK")

	f.Fuzz(func(_ *testing.T, id, email, password, displayName string) {
		req := RegisterRequest{
			ID:          NewUserID(id),
			Email:       email,
			Password:    password,
			DisplayName: displayName,
		}
		_ = req.Validate()
	})
}

func FuzzLoginRequest_Validate(f *testing.F) {
	f.Add("test@example.com", "secret12")
	f.Add("", "")
	f.Add("no-email", "short")

	f.Fuzz(func(_ *testing.T, email, password string) {
		req := LoginRequest{
			Email:    email,
			Password: password,
		}
		_ = req.Validate()
	})
}
