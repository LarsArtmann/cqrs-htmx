package usermgmt

import (
	"testing"
)

func TestOAuth2ProviderConfig_Validate(t *testing.T) {
	base := OAuth2ProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid config should not error: %v", err)
	}
}

func TestOAuth2ProviderConfig_Validate_MissingClientID(t *testing.T) {
	cfg := OAuth2ProviderConfig{
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing ClientID")
	}
}

func TestOAuth2ProviderConfig_Validate_MissingClientSecret(t *testing.T) {
	cfg := OAuth2ProviderConfig{
		ClientID:    "id",
		RedirectURL: "http://localhost/callback",
		IssuerURL:   "https://accounts.google.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing ClientSecret")
	}
}

func TestOAuth2ProviderConfig_Validate_MissingRedirectURL(t *testing.T) {
	cfg := OAuth2ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		IssuerURL:    "https://accounts.google.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing RedirectURL")
	}
}

func TestOAuth2ProviderConfig_Validate_OAuth2EndpointsRequired(t *testing.T) {
	cfg := OAuth2ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when IssuerURL is empty and no explicit endpoints")
	}
}

func TestOAuth2ProviderConfig_Validate_OAuth2EndpointsOK(t *testing.T) {
	cfg := OAuth2ProviderConfig{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		AuthURL:      "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		UserInfoURL:  "https://example.com/userinfo",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid OAuth2 config should not error: %v", err)
	}
}
