package opencorporates

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/legalparty-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// fixture mirrors the live response shape captured for a German UG
// (api.opencorporates.com/v0.4, de/B1204_HRB707415 shape; company_type null
// and the full address inside street_address are the German-record norm).
const germanCompanyFixture = `{
  "api_version": "0.4.8",
  "results": {
    "company": {
      "name": "Mundaneum UG (haftungsbeschraenkt)",
      "company_number": "B1204_HRB707415",
      "jurisdiction_code": "de",
      "incorporation_date": null,
      "company_type": null,
      "registry_url": null,
      "inactive": false,
      "current_status": "Currently Registered",
      "opencorporates_url": "https://opencorporates.com/companies/de/B1204_HRB707415",
      "native_company_number": "Freiburg HRB 707415",
      "registered_address_in_full": "Langenbach 15 a, 77709 Wolfach.",
      "registered_address": {
        "street_address": "Langenbach 15 a, 77709 Wolfach.",
        "locality": null,
        "region": null,
        "postal_code": null,
        "country": "Germany"
      },
      "previous_names": [],
      "alternative_names": []
    }
  }
}`

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(server.Client(), "test-token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("client must build: %v", err)
	}

	return client
}

func testRef(t *testing.T) legalparty.RegisterRef {
	t.Helper()

	ref, err := legalparty.NewRegisterRef("de", "B1204_HRB707415")
	if err != nil {
		t.Fatalf("ref must build: %v", err)
	}

	return ref
}

func TestNewClient_RequiresToken(t *testing.T) {
	if _, err := NewClient(nil, "  "); err == nil {
		t.Fatal("empty token must reject at construction")
	}
}

func TestResolveByRegister_GermanRecord(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/companies/de/B1204_HRB707415") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-API-TOKEN"); got != "test-token" {
			t.Errorf("token header: got %q", got)
		}
		if got := r.URL.Query().Get("sparse"); got != "true" {
			t.Errorf("sparse param: got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(germanCompanyFixture))
	})

	facts, err := client.ResolveByRegister(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}

	if facts.Source != legalparty.RegistrySourceOpenCorporates {
		t.Errorf("source: got %q", facts.Source)
	}
	if facts.LegalName != "Mundaneum UG (haftungsbeschraenkt)" {
		t.Errorf("legal name: got %q", facts.LegalName)
	}
	if facts.LegalFormLabel != "" {
		t.Errorf("null company_type must map to empty, got %q", facts.LegalFormLabel)
	}
	if facts.Status != "Currently Registered" {
		t.Errorf("status: got %q", facts.Status)
	}
	if facts.Country != "DE" {
		t.Errorf("jurisdiction de must map to DE, got %q", facts.Country)
	}
	if facts.SourceURL != "https://opencorporates.com/companies/de/B1204_HRB707415" {
		t.Errorf("attribution url: got %q", facts.SourceURL)
	}
	if facts.Register == nil || facts.Register.Number != "B1204_HRB707415" {
		t.Errorf("query ref must echo: got %+v", facts.Register)
	}
	if facts.RegisteredAddr != nil {
		t.Errorf("locality-less German address must stay untyped, got %+v", facts.RegisteredAddr)
	}
	if facts.FetchedAt.IsZero() {
		t.Error("fetched at must be stamped")
	}

	if err := facts.Validate(); err != nil {
		t.Errorf("mapped facts must validate: %v", err)
	}
}

func TestResolveByRegister_StructuredAddress(t *testing.T) {
	fixture := `{"api_version":"0.4.8","results":{"company":{
		"name":"OpenCorporates Ltd","company_number":"07444723","jurisdiction_code":"gb",
		"current_status":"Active","company_type":"Private Limited Company",
		"opencorporates_url":"https://opencorporates.com/companies/gb/07444723",
		"registered_address":{"street_address":"Aston House\nCornwall Avenue","locality":"London","region":null,"postal_code":"N3 1LF","country":"United Kingdom"}
	}}}`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixture))
	})

	ref, _ := legalparty.NewRegisterRef("gb", "07444723")
	facts, err := client.ResolveByRegister(context.Background(), ref)
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}

	if facts.LegalFormLabel != "Private Limited Company" {
		t.Errorf("company_type must map, got %q", facts.LegalFormLabel)
	}
	if facts.Country != "GB" {
		t.Errorf("jurisdiction gb must map to GB, got %q", facts.Country)
	}
	if facts.RegisteredAddr == nil || facts.RegisteredAddr.Locality != "London" ||
		facts.RegisteredAddr.PostalCode != "N3 1LF" {
		t.Errorf("structured address must project: got %+v", facts.RegisteredAddr)
	}
}

func TestResolveByRegister_SubdivisionJurisdiction(t *testing.T) {
	fixture := strings.Replace(
		strings.Replace(germanCompanyFixture, `"jurisdiction_code": "de"`, `"jurisdiction_code": "us_mi"`, 1),
		"Mundaneum UG (haftungsbeschraenkt)", "Michigan Widget LLC", 1)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixture))
	})

	facts, err := client.ResolveByRegister(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}
	if facts.Country != "US" {
		t.Errorf("us_mi must map to US, got %q", facts.Country)
	}
}

func TestResolveByRegister_StatusMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"404 is not found", http.StatusNotFound, legalparty.ErrRegistryNotFound},
		{"401 is a token problem", http.StatusUnauthorized, nil},
		{"403 is transient", http.StatusForbidden, legalparty.ErrRegistryUnavailable},
		{"503 is transient", http.StatusServiceUnavailable, legalparty.ErrRegistryUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
			})

			_, err := client.ResolveByRegister(context.Background(), testRef(t))
			if tt.want == nil {
				var coded interface{ ErrorCode() string }
				if !errors.As(err, &coded) || coded.ErrorCode() != "legalparty.registry.invalid_token" {
					t.Fatalf("status %d must map to invalid_token, got %v", tt.status, err)
				}

				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("status %d must map to %v, got %v", tt.status, tt.want, err)
			}
		})
	}
}

// TestResolveByRegister_InvalidReference pins the input guard: a reference the
// canonical vocabulary rejects never reaches the API, comes back as the
// adapter's own rejection, and keeps the canonical rejection as its cause.
func TestResolveByRegister_InvalidReference(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("must not call the API for an invalid register reference")
	})

	_, err := client.ResolveByRegister(context.Background(), legalparty.RegisterRef{Number: "B1204_HRB707415"})
	if err == nil {
		t.Fatal("a reference without a registry must reject without an API call")
	}
	if code := codeOf(t, err); code != "legalparty.registry.reference_invalid" {
		t.Errorf("error code: got %q, want the adapter's invalid-reference rejection", code)
	}
	if code := codeOf(t, errors.Unwrap(err)); code != "legalparty.register.registry_required" {
		t.Errorf("cause code: got %q, want go-legalparty's own register rejection", code)
	}
	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Errorf("family: got %v, want a rejection (bad input is never retried)", family)
	}
}

// codeOf returns the machine-readable code the error carries.
func codeOf(t *testing.T, err error) string {
	t.Helper()
	coded, ok := errors.AsType[errorfamily.Coded](err)
	if !ok {
		t.Fatalf("error %v carries no code", err)
	}

	return coded.ErrorCode()
}
