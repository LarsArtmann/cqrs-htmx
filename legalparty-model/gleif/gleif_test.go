package gleif

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/legalparty-model/v4"
)

// fixture mirrors the live record fetched 2026-11-20 for LEI
// 984500663D3E060B3G67 (api.gleif.org/api/v1/lei-records/984500663D3E060B3G67).
const leiRecordFixture = `{
  "data": {
    "type": "lei-records",
    "id": "984500663D3E060B3G67",
    "attributes": {
      "lei": "984500663D3E060B3G67",
      "entity": {
        "legalName": { "name": "ANDHOFF Wohnbau GmbH", "language": "de" },
        "legalAddress": {
          "language": "de",
          "addressLines": ["Kurfürstendamm 11"],
          "addressNumber": null,
          "mailRouting": null,
          "city": "Berlin",
          "region": "DE-BE",
          "country": "DE",
          "postalCode": "10719"
        },
        "registeredAt": { "id": "RA000217", "other": null },
        "registeredAs": "HRB 127525",
        "jurisdiction": "DE",
        "legalForm": { "id": "2HBR", "other": null },
        "status": "ACTIVE"
      },
      "registration": {
        "initialRegistrationDate": "2026-11-20T06:46:57Z",
        "status": "ISSUED",
        "managingLou": "529900T8BM49AURSDO55"
      }
    }
  }
}`

const leiLiveDE = "984500663D3E060B3G67"

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return NewClient(server.Client(), WithBaseURL(server.URL))
}

func TestResolveByLEI(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/lei-records/"+leiLiveDE) {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.api+json" {
			t.Errorf("Accept header: got %q", got)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(leiRecordFixture))
	})

	facts, err := client.ResolveByLEI(context.Background(), legalparty.LEI(leiLiveDE))
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}

	if facts.Source != legalparty.RegistrySourceGLEIF {
		t.Errorf("source: got %q", facts.Source)
	}
	if facts.LEI != legalparty.LEI(leiLiveDE) {
		t.Errorf("lei: got %q", facts.LEI)
	}
	if facts.LegalName != "ANDHOFF Wohnbau GmbH" {
		t.Errorf("legal name: got %q", facts.LegalName)
	}
	if facts.LegalFormCode != "2HBR" {
		t.Errorf("ELF code: got %q", facts.LegalFormCode)
	}
	if facts.Status != "ACTIVE" {
		t.Errorf("status: got %q", facts.Status)
	}
	if facts.Country != "DE" {
		t.Errorf("country: got %q", facts.Country)
	}
	if facts.Register == nil || facts.Register.Registry != "RA000217" || facts.Register.Number != "HRB 127525" {
		t.Errorf("register ref: got %+v", facts.Register)
	}
	if facts.SourceURL == "" || !strings.Contains(facts.SourceURL, leiLiveDE) {
		t.Errorf("source url: got %q", facts.SourceURL)
	}
	if facts.FetchedAt.IsZero() {
		t.Error("fetched at must be stamped")
	}
	if facts.RegisteredAddr == nil || facts.RegisteredAddr.Locality != "Berlin" {
		t.Errorf("address: got %+v", facts.RegisteredAddr)
	}

	if err := facts.Validate(); err != nil {
		t.Errorf("mapped facts must validate: %v", err)
	}
}

func TestResolveByLEI_StatusLiteralNull(t *testing.T) {
	fixture := strings.Replace(leiRecordFixture, `"status": "ACTIVE"`, `"status": "NULL"`, 1)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixture))
	})

	facts, err := client.ResolveByLEI(context.Background(), legalparty.LEI(leiLiveDE))
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}
	if facts.Status != "" {
		t.Errorf("literal NULL status must map to empty, got %q", facts.Status)
	}
}

func TestResolveByLEI_SubdivisionJurisdiction(t *testing.T) {
	fixture := strings.Replace(leiRecordFixture, `"jurisdiction": "DE"`, `"jurisdiction": "US-IA"`, 1)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixture))
	})

	facts, err := client.ResolveByLEI(context.Background(), legalparty.LEI(leiLiveDE))
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}
	if facts.Country != "US" {
		t.Errorf("subdivision jurisdiction must map to the country: got %q", facts.Country)
	}
}

const leiListFixture = `{
  "data": [
    {
      "type": "lei-records",
      "id": "984500663D3E060B3G67",
      "attributes": {
        "lei": "984500663D3E060B3G67",
        "entity": {
          "legalName": { "name": "ANDHOFF Wohnbau GmbH", "language": "de" },
          "legalAddress": {
            "addressLines": ["Kurfürstendamm 11"],
            "city": "Berlin",
            "region": "DE-BE",
            "country": "DE",
            "postalCode": "10719"
          },
          "registeredAt": { "id": "RA000217", "other": null },
          "registeredAs": "HRB 127525",
          "jurisdiction": "DE",
          "legalForm": { "id": "2HBR", "other": null },
          "status": "ACTIVE"
        },
        "registration": { "status": "ISSUED" }
      }
    }
  ]
}`

func TestResolveByRegister(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter[entity.registeredAs]"); got != "HRB 127525" {
			t.Errorf("filter: got %q", got)
		}

		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(leiListFixture))
	})

	ref, err := legalparty.NewRegisterRef("de", "HRB 127525")
	if err != nil {
		t.Fatalf("ref must build: %v", err)
	}

	facts, err := client.ResolveByRegister(context.Background(), ref)
	if err != nil {
		t.Fatalf("resolve must succeed: %v", err)
	}
	if facts.LegalName != "ANDHOFF Wohnbau GmbH" {
		t.Errorf("legal name: got %q", facts.LegalName)
	}
}

func TestResolveByRegister_NoMatch(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data": []}`))
	})

	ref, _ := legalparty.NewRegisterRef("de", "HRB 1")
	if _, err := client.ResolveByRegister(context.Background(), ref); !errors.Is(err, legalparty.ErrRegistryNotFound) {
		t.Fatalf("empty result must be ErrRegistryNotFound, got %v", err)
	}
}

func TestResolveByLEI_StatusMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"404 is not found", http.StatusNotFound, legalparty.ErrRegistryNotFound},
		{"429 is transient", http.StatusTooManyRequests, legalparty.ErrRegistryUnavailable},
		{"500 is transient", http.StatusInternalServerError, legalparty.ErrRegistryUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})

			_, err := client.ResolveByLEI(context.Background(), legalparty.LEI(leiLiveDE))
			if !errors.Is(err, tt.want) {
				t.Fatalf("status %d must map to %v, got %v", tt.status, tt.want, err)
			}
		})
	}
}

func TestResolveByLEI_InvalidInput(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("must not call the API for an invalid LEI")
	})

	if _, err := client.ResolveByLEI(context.Background(), "not-a-lei"); err == nil {
		t.Fatal("invalid LEI must reject without an API call")
	}
}
