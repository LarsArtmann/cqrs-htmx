package cqrshtmx

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/httputil"
)

// CommandDecoder decodes an HTTP request into a command.Command.
type CommandDecoder func(r *http.Request) (command.Command, error)

// QueryDecoder decodes an HTTP request into a query.Query.
type QueryDecoder func(r *http.Request) (query.Query, error)

// RenderFunc renders a query result as an HTTP response.
type RenderFunc func(w http.ResponseWriter, r *http.Request, result any) error

// TemplComponent matches templ.Component's interface without importing templ.
// Any type with Render(ctx, w) error satisfies this, including templ-generated components.
type TemplComponent interface {
	Render(ctx context.Context, w io.Writer) error
}

// HandlerOption configures a command or query handler.
type HandlerOption func(*handlerConfig)

// authMode defines the authorization mode for a handler.
type authMode int

const (
	authNone authMode = iota
	authRequired
	authAuthorized
)

func (m authMode) String() string {
	switch m {
	case authNone:
		return "none"
	case authRequired:
		return "required"
	case authAuthorized:
		return "authorized"
	default:
		return "unknown"
	}
}

type handlerConfig struct {
	authMode       authMode
	domain         string
	resource       string
	action         string
	authzRequest   func(*http.Request) (Request, error)
	commandDecoder CommandDecoder
	queryDecoder   QueryDecoder
	render         RenderFunc
	redirect       string
	trigger        string
	triggerDetail  map[string]any
	pushURL        string
	csrfConfig     *httputil.CSRFConfig
	maxBodySize    int64
	timeout        time.Duration
	successStatus  int
	requireMethod  string
	// requireMethods is the multi-method generalization of requireMethod
	// (RequireAnyMethod): when set, the request method must be one of these.
	requireMethods []string
	// requiredContentType, when set (RequireContentType), rejects requests
	// whose declared Content-Type is missing with a body (400) or different
	// from the required media type (415) — the RFC 10008 media-type ladder.
	requiredContentType string
	// acceptQuery carries the media types advertised via the Accept-Query
	// response header (RFC 10008 discovery). JSON body-query decoders seed it
	// with application/json so the advertisement cannot drift from what the
	// handler actually decodes; AcceptQuery appends consumer types.
	acceptQuery  []string
	onError      func(*http.Request, error)
	requestGuard RequestGuardFunc
	openapiMeta  *openapi.Operation
}

// methodAllowed reports whether the HTTP method passes the method guard:
// the single-method RequireMethod when set, then the multi-method
// RequireAnyMethod set. No guard configured means every method is allowed
// (the consumer's mux pattern owns method routing in that case).
func (c *handlerConfig) methodAllowed(method string) bool {
	if c.requireMethod != "" {
		return method == c.requireMethod
	}
	if len(c.requireMethods) > 0 {
		for _, allowed := range c.requireMethods {
			if method == allowed {
				return true
			}
		}
		return false
	}
	return true
}

// allowedMethodsHeader renders the configured methods for the Allow header
// and the 405 error message: the RequireMethod spelling for a single method
// (byte-identical to the pre-RequireAnyMethod error), the configured order
// for a set.
func (c *handlerConfig) allowedMethodsHeader() string {
	if c.requireMethod != "" {
		return c.requireMethod
	}
	return strings.Join(c.requireMethods, ", ")
}

// hasNoExplicitBody returns true if the handler has no render function and
// no HTMX response fields that would produce body content.
// When true, the handler should return 204 No Content.
func (c *handlerConfig) hasNoExplicitBody() bool {
	return c.render == nil &&
		c.redirect == "" &&
		c.trigger == "" &&
		c.pushURL == "" &&
		len(c.triggerDetail) == 0
}

// decodeAndSet creates a HandlerOption that decodes a request body and sets
// the result on the handlerConfig. It collapses the 4 Decode* variants.
func decodeAndSet[T, R any](
	bodyDec func(*http.Request, int64) (T, error),
	mapper func(T) (R, error),
	setter func(*handlerConfig, func(*http.Request) (R, error)),
) HandlerOption {
	return func(config *handlerConfig) {
		setter(config, func(r *http.Request) (R, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return bodyDec(r, config.maxBodySize)
			}, mapper)
		})
	}
}

// decodeAndSetWithRequest is the *http.Request-aware variant of decodeAndSet.
// The mapper receives both the request and the decoded body, enabling request-scoped
// data (cookies, headers, path values) in the command/query mapping step.
func decodeAndSetWithRequest[T, R any](
	bodyDec func(*http.Request, int64) (T, error),
	mapper func(r *http.Request, body T) (R, error),
	setter func(*handlerConfig, func(*http.Request) (R, error)),
) HandlerOption {
	return func(config *handlerConfig) {
		setter(config, func(r *http.Request) (R, error) {
			decoded, err := bodyDec(r, config.maxBodySize)
			if err != nil {
				var zero R

				return zero, err
			}

			return mapper(r, decoded)
		})
	}
}
