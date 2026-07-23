package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

type Email = identitymodel.Email

func ParseEmail(raw string) (Email, error)   { return identitymodel.ParseEmail(raw) }
func MustParseEmail(raw string) Email         { return identitymodel.MustParseEmail(raw) }
