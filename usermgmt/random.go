package usermgmt

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

// randomBase64URLString returns n cryptographically secure random bytes,
// base64url-encoded without padding. purpose decorates the returned error with
// domain context (e.g. "session token" → "generate session token: ...").
//
// Shared by session-token, verification-token, and OAuth2-state generation so
// the entropy source, length, and encoding stay consistent across all of them.
func randomBase64URLString(n int, purpose string) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", errorfamily.Wrapf(err, event.Infrastructure, "usermgmt.random.failed", "generate %s", purpose)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
