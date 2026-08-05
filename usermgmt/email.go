package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
type Email = identitymodel.Email

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func ParseEmail(raw string) (Email, error) { return identitymodel.ParseEmail(raw) }

// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.
func MustParseEmail(raw string) Email { return identitymodel.MustParseEmail(raw) }
