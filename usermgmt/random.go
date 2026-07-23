package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

func randomBase64URLString(n int, purpose string) (string, error) {
	return identitymodel.RandomBase64URLString(n, purpose)
}
