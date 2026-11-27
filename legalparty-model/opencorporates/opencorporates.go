// Package opencorporates resolves company records from OpenCorporates
// (api.opencorporates.com), the aggregator of official company registries.
//
// Authentication: every API call needs a token, sent here as the
// X-API-TOKEN header. Open data plans allow 200 requests per month and 50
// per day with a hard cap of 5 queries per second; exceeding limits
// answers 403 (mapped to a transient error).
//
// ATTRIBUTION (contract, not decoration): OpenCorporates data is ODbL
// licensed. Displaying facts requires a hyperlink reading "from
// OpenCorporates" pointing at the record page — RegistryFacts.SourceURL
// carries it; API-derived distributions must show "Powered by
// OpenCorporates". Share-alike applies to open-data accounts; paid plans
// drop it. See https://opencorporates.com/terms-of-use-2/.
//
// Registry references use OpenCorporates' own identifiers: Registry is the
// jurisdiction code ("de", "us_mi") and Number the company_number
// ("B1204_HRB707415" for Germany — register court and section are encoded
// in the number). German records often leave company_type null and stuff
// the whole address into street_address; the typed projection includes an
// address only when locality is present, because a partially typed address
// would pretend more structure than the registry carries.
package opencorporates

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/legalparty-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// DefaultBaseURL is the pinned API version root (0.4).
const DefaultBaseURL = "https://api.opencorporates.com/v0.4"

// Client resolves company records from api.opencorporates.com.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// Option configures the client.
type Option func(*Client)

// WithBaseURL overrides the API root (tests point it at an httptest server).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// NewClient builds an OpenCorporates client; the API token is required.
// A nil httpClient falls back to one with a 30-second timeout.
func NewClient(httpClient *http.Client, apiToken string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiToken) == "" {
		return nil, errorfamily.NewInfrastructure("legalparty.registry.token_required",
			"OpenCorporates requires an API token (X-API-TOKEN)")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	c := &Client{baseURL: DefaultBaseURL, token: apiToken, http: httpClient}
	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// ResolveByRegister fetches the company record for the register entry.
func (c *Client) ResolveByRegister(
	ctx context.Context, ref legalparty.RegisterRef,
) (legalparty.RegistryFacts, error) {
	if err := ref.Validate(); err != nil {
		return legalparty.RegistryFacts{}, errorfamily.WrapRejection(err, "legalparty.registry.reference_invalid",
			"validate the register reference for the OpenCorporates lookup")
	}

	endpoint := c.baseURL + "/companies/" + url.PathEscape(ref.Registry) + "/" +
		url.PathEscape(ref.Number) + "?sparse=true"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return legalparty.RegistryFacts{}, errorfamily.WrapInfrastructure(err,
			"legalparty.registry.request_failed", "build OpenCorporates request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-TOKEN", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return legalparty.RegistryFacts{}, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable", "call OpenCorporates: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := readBody(resp)
	if err != nil {
		return legalparty.RegistryFacts{}, err
	}

	var env companyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return legalparty.RegistryFacts{}, errorfamily.WrapCorruption(err,
			"legalparty.registry.bad_response", "decode OpenCorporates company")
	}

	return companyToFacts(ref, env.Results.Company), nil
}

type companyEnvelope struct {
	APIVersion string  `json:"api_version"`
	Results    results `json:"results"`
}

type results struct {
	Company company `json:"company"`
}

type company struct {
	Name              string   `json:"name"`
	CompanyNumber     string   `json:"company_number"`
	JurisdictionCode  string   `json:"jurisdiction_code"`
	CompanyType       *string  `json:"company_type"`
	CurrentStatus     *string  `json:"current_status"`
	OpencorporatesURL string   `json:"opencorporates_url"`
	RegisteredAddress *address `json:"registered_address"`
}

type address struct {
	StreetAddress string  `json:"street_address"`
	Locality      *string `json:"locality"`
	Region        *string `json:"region"`
	PostalCode    *string `json:"postal_code"`
	Country       string  `json:"country"`
}

func readBody(resp *http.Response) ([]byte, error) {
	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errorfamily.WrapCorruption(err,
				"legalparty.registry.bad_response", "read OpenCorporates response body")
		}

		return body, nil
	case http.StatusNotFound:
		return nil, legalparty.ErrRegistryNotFound
	case http.StatusUnauthorized:
		return nil, errorfamily.NewInfrastructure("legalparty.registry.invalid_token",
			"OpenCorporates rejected the API token (check X-API-TOKEN)")
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable", "OpenCorporates rate limited the request (403); back off")
	case http.StatusServiceUnavailable:
		return nil, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable",
			"OpenCorporates is overloaded or the company is temporarily redacted (503)")
	default:
		return nil, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable", "OpenCorporates answered "+resp.Status)
	}
}

func companyToFacts(ref legalparty.RegisterRef, c company) legalparty.RegistryFacts {
	facts := legalparty.RegistryFacts{
		Source:    legalparty.RegistrySourceOpenCorporates,
		Register:  &ref,
		LegalName: c.Name,
		Status:    deref(c.CurrentStatus),
		Country:   jurisdictionCountry(c.JurisdictionCode),
		SourceURL: c.OpencorporatesURL,
		FetchedAt: time.Now().UTC(),
	}
	if c.CompanyType != nil {
		facts.LegalFormLabel = *c.CompanyType
	}
	if a := typedAddress(c.RegisteredAddress, facts.Country); a != nil {
		facts.RegisteredAddr = a
	}

	return facts
}

// jurisdictionCountry maps OpenCorporates' jurisdiction code ("de",
// "us_mi") to an ISO 3166-1 alpha-2 code, leaving it unset when it does
// not parse.
func jurisdictionCountry(code string) legalparty.CountryCode {
	if i := strings.IndexByte(code, '_'); i > 0 {
		code = code[:i]
	}

	cc, err := legalparty.ParseCountryCode(code)
	if err != nil {
		return ""
	}

	return cc
}

// typedAddress projects a structured registered address; nil when the
// locality is missing (German records keep the full address inside
// street_address) or the parts do not validate.
func typedAddress(a *address, country legalparty.CountryCode) *legalparty.RegisteredAddress {
	if a == nil || a.Locality == nil || *a.Locality == "" {
		return nil
	}

	postal, region := "", ""
	if a.PostalCode != nil {
		postal = *a.PostalCode
	}
	if a.Region != nil {
		region = *a.Region
	}

	built, err := legalparty.NewRegisteredAddress(a.StreetAddress, *a.Locality, postal, region, country)
	if err != nil {
		return nil
	}

	return &built
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
