package usermgmt

import (
	"testing"
)

// BenchmarkMarshalWebAuthnUser measures the JSON serialization overhead
// at the module boundary. This is the cost of the v4 extraction design —
// the Service marshals a User to []byte, the Provider unmarshals it.
// For WebAuthn ceremonies (not hot paths), this should be negligible (~µs).
func BenchmarkMarshalWebAuthnUser(b *testing.B) {
	user := &User{
		ID:    NewUserID("01HXKYGEG0QH8XJYQKZ3R8WZAA"),
		Email: "benchmark@test.com",
		Credentials: []WebAuthnCredential{
			{CredentialCore: CredentialCore{Name: "key1"}},
			{CredentialCore: CredentialCore{Name: "key2"}},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = marshalWebAuthnUser(user)
	}
}

// BenchmarkMarshalWebAuthnUser_NoCreds measures serialization for a user
// with zero credentials (the common case during registration).
func BenchmarkMarshalWebAuthnUser_NoCreds(b *testing.B) {
	user := &User{
		ID:    NewUserID("01HXKYGEG0QH8XJYQKZ3R8WZAA"),
		Email: "benchmark@test.com",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = marshalWebAuthnUser(user)
	}
}
