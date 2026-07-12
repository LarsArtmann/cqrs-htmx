package cqrshtmx

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
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
	resource       string
	action         string
	commandDecoder CommandDecoder
	queryDecoder   QueryDecoder
	render         RenderFunc
	redirect       string
	trigger        string
	triggerDetail  map[string]any
	pushURL        string
	csrfConfig     *CSRFConfig
	maxBodySize    int64
	timeout        time.Duration
	successStatus  int
	requireMethod  string
	onError        func(*http.Request, error)
	requestGuard   RequestGuardFunc
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
	return func(cfg *handlerConfig) {
		setter(cfg, func(r *http.Request) (R, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return bodyDec(r, cfg.maxBodySize)
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
	return func(cfg *handlerConfig) {
		setter(cfg, func(r *http.Request) (R, error) {
			decoded, err := bodyDec(r, cfg.maxBodySize)
			if err != nil {
				var zero R
				return zero, err
			}
			return mapper(r, decoded)
		})
	}
}
