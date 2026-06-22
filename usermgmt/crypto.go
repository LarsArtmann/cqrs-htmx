package usermgmt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// GenerateToken generates a 256-bit random API token encoded as base64url.
// The raw token is returned to the caller ONCE at creation time; only the
// HMAC hash is persisted. Returns an error if crypto/rand fails.
func GenerateToken() (string, error) {
	raw := make([]byte, 32) // 256 bits
	if _, err := rand.Read(raw); err != nil {
		return "", event.WrapInfrastructure(err, "usermgmt.crypto.rand_failed", "generate API token")
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashToken computes the HMAC-SHA256 of the token using the pepper as key.
// The result is what gets stored in the event store — the raw token is never
// persisted. Pepper provides DB-leak defense: an attacker with the database
// cannot brute-force tokens without also having the server-side pepper.
func HashToken(token string, pepper []byte) []byte {
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(token))
	return mac.Sum(nil)
}

// VerifyToken checks whether the token matches the stored HMAC using a
// constant-time comparison to prevent timing attacks.
func VerifyToken(token string, storedMAC, pepper []byte) bool {
	expectedMAC := HashToken(token, pepper)
	return hmac.Equal(storedMAC, expectedMAC)
}
