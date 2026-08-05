package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func GenerateToken() (string, error) { return identitymodel.GenerateToken() }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func HashToken(token string, pepper []byte) []byte { return identitymodel.HashToken(token, pepper) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func VerifyToken(token string, storedMAC, pepper []byte) bool {
	return identitymodel.VerifyToken(token, storedMAC, pepper)
}
