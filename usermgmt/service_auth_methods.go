package usermgmt

// HasWebAuthn reports whether a WebAuthn provider is configured.
func (s *Service) HasWebAuthn() bool { return s.webauthn != nil }

// HasOAuth2 reports whether an OAuth2 provider is configured.
func (s *Service) HasOAuth2() bool { return s.oauth2 != nil }

// HasTOTP reports whether a TOTP provider is configured.
func (s *Service) HasTOTP() bool { return s.totp != nil }

// ConfiguredOAuth2Providers returns the sorted names of all configured OAuth2
// providers. Returns nil when no OAuth2 provider is configured or when the
// provider does not support name enumeration (backward compatible with custom
// implementations that don't implement Names()).
func (s *Service) ConfiguredOAuth2Providers() []string {
	if s.oauth2 == nil {
		return nil
	}
	type providerNamer interface {
		Names() []string
	}
	if namer, ok := s.oauth2.(providerNamer); ok {
		return namer.Names()
	}
	return nil
}
