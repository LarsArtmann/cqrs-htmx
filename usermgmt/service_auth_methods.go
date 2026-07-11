package usermgmt

// HasWebAuthn reports whether a WebAuthn provider is configured.
func (s *Service) HasWebAuthn() bool { return s.webauthn != nil }

// HasOAuth2 reports whether an OAuth2 provider is configured.
func (s *Service) HasOAuth2() bool { return s.oauth2 != nil }

// HasTOTP reports whether a TOTP provider is configured.
func (s *Service) HasTOTP() bool { return s.totp != nil }
