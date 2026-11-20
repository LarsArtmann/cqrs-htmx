// Package gleif resolves LEI records from the Global Legal Entity
// Identifier Foundation's public API (api.gleif.org).
//
// The API needs no key and allows 60 requests per minute per user; 404 and
// 429 answers are plain HTML, so the client branches on the status code
// only. Data is published under CC BY 4.0 — attribute "Source: GLEIF" when
// displaying facts (RegistryFacts.SourceURL carries the record link).
//
// Register-reference lookups use GLEIF's exact-match filter over the
// entity's registeredAs value, so the number must be spelled the way the
// national register writes it ("HRB 127525", not "HRB127525").
package gleif

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/legalparty-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// DefaultBaseURL is the public GLEIF API root.
const DefaultBaseURL = "https://api.gleif.org/api/v1"

// elfCodePattern is the ISO 20275 ELF code shape; 8888 (code requested, not
// yet listed) and 9999 (no separate legal form) are reserved placeholders
// that carry no form information.
var elfCodePattern = regexp.MustCompile(`^[A-Z0-9]{4}$`)

// Client resolves LEI records from api.gleif.org.
type Client struct {
	baseURL string
	http    *http.Client
}

// Option configures the client.
type Option func(*Client)

// WithBaseURL overrides the API root (tests point it at an httptest server).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// NewClient builds a GLEIF client. A nil httpClient falls back to one with
// a 30-second timeout.
func NewClient(httpClient *http.Client, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	c := &Client{baseURL: DefaultBaseURL, http: httpClient}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ResolveByLEI fetches the registry record for the LEI holder.
func (c *Client) ResolveByLEI(ctx context.Context, lei legalparty.LEI) (legalparty.RegistryFacts, error) {
	if _, err := legalparty.ParseLEI(string(lei)); err != nil {
		return legalparty.RegistryFacts{}, err
	}

	record, err := c.fetchRecord(ctx, c.baseURL+"/lei-records/"+url.PathEscape(string(lei)))
	if err != nil {
		return legalparty.RegistryFacts{}, err
	}

	return recordToFacts(record, c.baseURL+"/lei-records/"+string(record.Attributes.LEI)), nil
}

// ResolveByRegister fetches the registry record for a national register
// entry, matching GLEIF's registeredAs value exactly.
func (c *Client) ResolveByRegister(
	ctx context.Context, ref legalparty.RegisterRef,
) (legalparty.RegistryFacts, error) {
	if err := ref.Validate(); err != nil {
		return legalparty.RegistryFacts{}, err
	}

	q := url.Values{}
	q.Set("filter[entity.registeredAs]", ref.Number)
	q.Set("page[size]", "5")

	list, err := c.fetchList(ctx, c.baseURL+"/lei-records?"+q.Encode())
	if err != nil {
		return legalparty.RegistryFacts{}, err
	}
	if len(list.Data) == 0 {
		return legalparty.RegistryFacts{}, legalparty.ErrRegistryNotFound
	}

	first := list.Data[0]

	return recordToFacts(first, c.baseURL+"/lei-records/"+string(first.Attributes.LEI)), nil
}

type recordEnvelope struct {
	Data record `json:"data"`
}

type listEnvelope struct {
	Data []record `json:"data"`
}

type record struct {
	Attributes attributes `json:"attributes"`
}

type attributes struct {
	LEI          string       `json:"lei"`
	Entity       entity       `json:"entity"`
	Registration registration `json:"registration"`
}

type entity struct {
	LegalName    nameObject `json:"legalName"`
	RegisteredAt refObject  `json:"registeredAt"`
	RegisteredAs string     `json:"registeredAs"`
	Jurisdiction string     `json:"jurisdiction"`
	LegalForm    legalForm  `json:"legalForm"`
	Status       string     `json:"status"`
	LegalAddress address    `json:"legalAddress"`
}

type nameObject struct {
	Name string `json:"name"`
}

type refObject struct {
	ID    string  `json:"id"`
	Other *string `json:"other"`
}

type legalForm struct {
	ID    string  `json:"id"`
	Other *string `json:"other"`
}

type address struct {
	AddressLines []string `json:"addressLines"`
	City         string   `json:"city"`
	Region       *string  `json:"region"`
	PostalCode   *string  `json:"postalCode"`
	Country      string   `json:"country"`
}

type registration struct {
	Status string `json:"status"`
}

func (c *Client) fetchRecord(ctx context.Context, endpoint string) (record, error) {
	body, err := c.get(ctx, endpoint)
	if err != nil {
		return record{}, err
	}

	var env recordEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return record{}, errorfamily.WrapCorruption(err,
			"legalparty.registry.bad_response", "decode GLEIF lei-record")
	}

	return env.Data, nil
}

func (c *Client) fetchList(ctx context.Context, endpoint string) (listEnvelope, error) {
	body, err := c.get(ctx, endpoint)
	if err != nil {
		return listEnvelope{}, err
	}

	var env listEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return listEnvelope{}, errorfamily.WrapCorruption(err,
			"legalparty.registry.bad_response", "decode GLEIF lei-record list")
	}

	return env, nil
}

func (c *Client) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err,
			"legalparty.registry.request_failed", "build GLEIF request")
	}
	req.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable", "call GLEIF: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	return readBody(resp)
}

func readBody(resp *http.Response) ([]byte, error) {
	switch {
	case resp.StatusCode == http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errorfamily.WrapCorruption(err,
				"legalparty.registry.bad_response", "read GLEIF response body")
		}

		return body, nil
	case resp.StatusCode == http.StatusNotFound:
		return nil, legalparty.ErrRegistryNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable", "GLEIF rate limited the request (60/min); back off")
	default:
		return nil, errorfamily.WrapTransient(legalparty.ErrRegistryUnavailable,
			"legalparty.registry.unavailable", "GLEIF answered "+resp.Status)
	}
}

func recordToFacts(rec record, sourceURL string) legalparty.RegistryFacts {
	e := rec.Attributes.Entity

	facts := legalparty.RegistryFacts{
		Source:    legalparty.RegistrySourceGLEIF,
		LEI:       legalparty.LEI(rec.Attributes.LEI),
		LegalName: e.LegalName.Name,
		Status:    normalizeStatus(e.Status),
		Country:   jurisdictionCountry(e.Jurisdiction),
		SourceURL: sourceURL,
		FetchedAt: time.Now().UTC(),
	}
	if elfCodePattern.MatchString(e.LegalForm.ID) && e.LegalForm.ID != "8888" && e.LegalForm.ID != "9999" {
		facts.LegalFormCode = e.LegalForm.ID
	}
	if e.LegalForm.Other != nil {
		facts.LegalFormLabel = *e.LegalForm.Other
	}
	if e.RegisteredAt.ID != "" && e.RegisteredAs != "" {
		ref, err := legalparty.NewRegisterRef(e.RegisteredAt.ID, e.RegisteredAs)
		if err == nil {
			facts.Register = &ref
		}
	}
	if a := typedAddress(e.LegalAddress); a != nil {
		facts.RegisteredAddr = a
	}

	return facts
}

// normalizeStatus maps GLEIF's literal "NULL" (emitted for annulled
// records) to an absent status.
func normalizeStatus(status string) string {
	if status == "NULL" {
		return ""
	}

	return status
}

// jurisdictionCountry maps GLEIF's jurisdiction ("DE", "US-IA") to an
// ISO 3166-1 alpha-2 code, leaving it unset when it does not parse.
func jurisdictionCountry(jurisdiction string) legalparty.CountryCode {
	code := jurisdiction
	if i := strings.IndexByte(code, '-'); i > 0 {
		code = code[:i]
	}

	cc, err := legalparty.ParseCountryCode(code)
	if err != nil {
		return ""
	}

	return cc
}

// typedAddress projects GLEIF's structured legal address; nil when the
// parts do not validate.
func typedAddress(a address) *legalparty.RegisteredAddress {
	street := strings.Join(a.AddressLines, ", ")
	if street == "" || a.City == "" {
		return nil
	}

	postal, region := "", ""
	if a.PostalCode != nil {
		postal = *a.PostalCode
	}
	if a.Region != nil {
		region = *a.Region
	}

	country := legalparty.CountryCode("")
	if cc, err := legalparty.ParseCountryCode(a.Country); err == nil {
		country = cc
	}

	built, err := legalparty.NewRegisteredAddress(street, a.City, postal, region, country)
	if err != nil {
		return nil
	}

	return &built
}
