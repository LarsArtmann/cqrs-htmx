package legalparty

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// errorCode extracts the machine-readable code from a domain error.
func errorCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	var coded errorfamily.Coded
	if !errors.As(err, &coded) {
		t.Fatalf("error %v carries no code", err)
	}

	return coded.ErrorCode()
}

// TestParseLegalPartyID_RoundTrip pins the strict reconstruction path: a
// fresh id round-trips, and the display prefix carries no identity.
func TestParseLegalPartyID_RoundTrip(t *testing.T) {
	id := GenerateLegalPartyID()

	parsed, err := ParseLegalPartyID(id.String())
	if err != nil {
		t.Fatalf("fresh id must parse: %v", err)
	}
	if parsed.Get() != id.Get() {
		t.Fatalf("round trip changed identity: got %q, want %q", parsed.Get(), id.Get())
	}

	prefixed, err := ParseLegalPartyID("LegalParty:" + id.Get())
	if err != nil {
		t.Fatalf("display prefix must parse: %v", err)
	}
	if prefixed.Get() != id.Get() {
		t.Fatalf("prefix strip changed identity: got %q, want %q", prefixed.Get(), id.Get())
	}
}

func TestParseLegalPartyID_Strict(t *testing.T) {
	valid := GenerateLegalPartyID().Get()

	tests := []struct {
		name     string
		in       string
		wantCode string
	}{
		{"valid ulid", valid, ""},
		{"empty", "", "legalparty.legal_party_id.invalid"},
		{"whitespace-padded", " " + valid + " ", "legalparty.legal_party_id.invalid"},
		{"not a ulid", "hello-world", "legalparty.legal_party_id.invalid"},
		{"truncated", valid[:20], "legalparty.legal_party_id.invalid"},
		{"other brand prefix", "Tenant:" + valid, "legalparty.legal_party_id.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseLegalPartyID(tt.in)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("ParseLegalPartyID(%q) must accept: %v", tt.in, err)
				}

				return
			}
			if err == nil {
				t.Fatalf("ParseLegalPartyID(%q) must reject, got %v", tt.in, parsed)
			}
			if got := errorCode(t, err); got != tt.wantCode {
				t.Fatalf("error code: got %q, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestGenerateLegalPartyID_Unique(t *testing.T) {
	first := GenerateLegalPartyID()
	second := GenerateLegalPartyID()
	if first.Get() == second.Get() {
		t.Fatal("two generated ids must differ")
	}
}
