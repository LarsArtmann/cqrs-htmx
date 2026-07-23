package identitymodel

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// randomBase64URLString returns n cryptographically secure random bytes,
// base64url-encoded without padding.
func RandomBase64URLString(n int, purpose string) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", errorfamily.Wrapf(err, event.Infrastructure, "usermgmt.random.failed", "generate %s", purpose)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
