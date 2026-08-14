package integration_test

// This file provides compile-time verification that the three auth strategy
// sub-module Providers satisfy the core usermgmt interfaces. The assertions
// live in integration_test (not in the sub-modules themselves) to avoid
// forcing each sub-module to import core usermgmt as a dependency — that
// would defeat the entire purpose of the v4 extraction.

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
)

// Compile-time interface satisfaction assertions.
// If any of these fail, the sub-module Provider has drifted from the
// usermgmt interface contract.
var (
	_ identitymodel.TOTPProvider     = (*totp.Provider)(nil)
	_ identitymodel.WebAuthnProvider = (*webauthn.Provider)(nil)
	_ identitymodel.OAuth2Provider   = (*oauth2.Provider)(nil)
)
