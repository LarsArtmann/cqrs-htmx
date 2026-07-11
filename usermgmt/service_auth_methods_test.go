package usermgmt

import (
	"context"
	"testing"
)

// mockOAuth2WithNames is an OAuth2Provider that also implements Names().
type mockOAuth2WithNames struct {
	names []string
}

func (m *mockOAuth2WithNames) BeginLogin(_ context.Context, _, _ string) (string, string, error) {
	return "", "", nil
}

func (m *mockOAuth2WithNames) FinishLogin(_ context.Context, _, _, _ string) ([]byte, error) {
	return nil, nil
}

func (m *mockOAuth2WithNames) Names() []string {
	return m.names
}

// mockOAuth2WithoutNames implements only the OAuth2Provider interface.
type mockOAuth2WithoutNames struct{}

func (m *mockOAuth2WithoutNames) BeginLogin(_ context.Context, _, _ string) (string, string, error) {
	return "", "", nil
}

func (m *mockOAuth2WithoutNames) FinishLogin(_ context.Context, _, _, _ string) ([]byte, error) {
	return nil, nil
}

func TestService_ConfiguredOAuth2Providers_Nil(t *testing.T) {
	svc, err := NewService(ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if got := svc.ConfiguredOAuth2Providers(); got != nil {
		t.Errorf("expected nil for no oauth2, got %v", got)
	}
}

func TestService_ConfiguredOAuth2Providers_WithNames(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		OAuth2: &mockOAuth2WithNames{names: []string{"github", "google"}},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	got := svc.ConfiguredOAuth2Providers()
	want := []string{"github", "google"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestService_ConfiguredOAuth2Providers_WithoutNames(t *testing.T) {
	svc, err := NewService(ServiceConfig{
		OAuth2: &mockOAuth2WithoutNames{},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if got := svc.ConfiguredOAuth2Providers(); got != nil {
		t.Errorf("expected nil for provider without Names(), got %v", got)
	}
}
