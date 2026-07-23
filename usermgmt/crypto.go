package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

func GenerateToken() (string, error)                 { return identitymodel.GenerateToken() }
func HashToken(token string, pepper []byte) []byte   { return identitymodel.HashToken(token, pepper) }
func VerifyToken(token string, storedMAC, pepper []byte) bool {
	return identitymodel.VerifyToken(token, storedMAC, pepper)
}
