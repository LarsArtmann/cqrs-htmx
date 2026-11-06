package huma

import (
	"context"
	"time"

	"github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Option configures how a registered operation runs the pipeline.
//
// The options theme mirrors the endpoint builder's HandlerOption values, minus
// the stages Huma owns: there is no decoder option (Huma decodes the typed
// input), no render option (Huma writes the typed output) and no HTMX option
// (a JSON API has no HTMX headers).
type Option func(*operationConfig)

// authMode is the authorization posture of one operation.
type authMode int

const (
	// authNone means the operation is public: the adapter does not consult the
	// policy engine at all.
	authNone authMode = iota
	// authRequired means the caller must be authenticated, no policy question.
	authRequired
	// authAuthorized means the caller must hold a permission.
	authAuthorized
)

// operationConfig is the resolved option set of one registration.
type operationConfig struct {
	mode     authMode
	domain   string
	resource string
	action   string
	// resolve derives the authorization question from the typed input. Nil
	// means the question is the static domain/resource/action triple, unless
	// the mode is authRequired or authNone.
	resolve func(ctx context.Context, in any) (cqrshtmx.Request, error)
	// guard runs after the command or query is built and validated.
	guard func(ctx context.Context, cmdOrQuery any) error
	// validateCommand and validateQuery run between build and guard.
	validateCommand func(command.Command) error
	validateQuery   func(query.Query) error
	// timeout overrides the adapter-wide timeout when positive.
	timeout time.Duration
}

// newOperationConfig applies opts in order; later options win.
func newOperationConfig(opts []Option) *operationConfig {
	cfg := &operationConfig{
		mode:            authNone,
		domain:          "",
		resource:        "",
		action:          "",
		resolve:         nil,
		guard:           nil,
		validateCommand: nil,
		validateQuery:   nil,
		timeout:         0,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return cfg
}

// Authorize requires the caller to hold action on resource in the wildcard
// domain ("*"). It is the transport-neutral twin of cqrshtmx.Authorize: only
// global policies and grants answer it, so use AuthorizeIn whenever the answer
// depends on tenant membership.
func Authorize(resource, action string) Option {
	return func(cfg *operationConfig) {
		cfg.mode = authAuthorized
		cfg.domain = ""
		cfg.resource = resource
		cfg.action = action
		cfg.resolve = nil
	}
}

// AuthorizeIn requires the caller to hold action on resource inside domain,
// the tenant-scoped counterpart of Authorize.
func AuthorizeIn(domain, resource, action string) Option {
	return func(cfg *operationConfig) {
		cfg.mode = authAuthorized
		cfg.domain = domain
		cfg.resource = resource
		cfg.action = action
		cfg.resolve = nil
	}
}

// AuthorizeOn defers the whole authorization question to resolve, which sees
// the typed input (path parameters, query values, body) before anything is
// dispatched. It is the typed twin of cqrshtmx.AuthorizeRequest.
//
//	RegisterCommand(api, pipe, op, build,
//	    AuthorizeOn(func(_ context.Context, in *RevokeInput) (cqrshtmx.Request, error) {
//	        return cqrshtmx.Request{
//	            Domain:   in.Tenant,
//	            Resource: cqrshtmx.Resource{Type: "tokens", ID: in.TokenID},
//	            Action:   "revoke",
//	        }, nil
//	    }),
//	)
//
// Leaving Request.Subject empty authorizes the authenticated user from the
// request context; set it explicitly to act as a bot or service identity.
func AuthorizeOn[I any](resolve func(ctx context.Context, in *I) (cqrshtmx.Request, error)) Option {
	return func(cfg *operationConfig) {
		cfg.mode = authAuthorized
		cfg.resolve = func(ctx context.Context, in any) (cqrshtmx.Request, error) {
			typed, ok := in.(*I)
			if !ok {
				return cqrshtmx.Request{}, //nolint:exhaustruct_v5 // zero-value return on a wiring bug
					errorfamily.NewCorruption("huma_adapter_input_type",
						"[cqrs-htmx/huma] AuthorizeOn option was applied to an operation with a different input type")
			}

			return resolve(ctx, typed)
		}
	}
}

// RequireAuth requires an authenticated caller without asking the policy
// engine. The twin of cqrshtmx.RequireAuth.
func RequireAuth() Option {
	return func(cfg *operationConfig) {
		cfg.mode = authRequired
	}
}

// Guard runs guard after the command or query is built and validated, before
// dispatch. It is the twin of cqrshtmx.RequestGuard for authorization models
// that do not fit a policy engine (ownership checks, API keys, feature flags).
//
//	Guard(func(_ context.Context, cmdOrQuery any) error {
//	    if !ownsVisit(cmdOrQuery.(*visitCmd).VisitID) {
//	        return httperr.ErrForbidden
//	    }
//	    return nil
//	})
func Guard(guard func(ctx context.Context, cmdOrQuery any) error) Option {
	return func(cfg *operationConfig) {
		cfg.guard = guard
	}
}

// ValidateCommand validates a built command before dispatch. The twin of
// cqrshtmx.ValidateCommand: a validator error is a rejection carrying the
// cqrshtmx.validate.failed code, so the client sees HTTP 400 on both
// transports instead of the fail-open 503 an unclassified error would produce.
func ValidateCommand(validate func(command.Command) error) Option {
	return func(cfg *operationConfig) {
		cfg.validateCommand = validate
	}
}

// ValidateQuery validates a built query before dispatch. The twin of
// cqrshtmx.ValidateQuery.
func ValidateQuery(validate func(query.Query) error) Option {
	return func(cfg *operationConfig) {
		cfg.validateQuery = validate
	}
}

// WithTimeout bounds this operation's dispatch, overriding Config.Timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *operationConfig) {
		cfg.timeout = timeout
	}
}

// validateCommandValue applies the validation stage the endpoint pipeline
// applies, with the same wrapping code, so a validator failure is a 400 with
// code cqrshtmx.validate.failed whichever transport ran it.
func validateCommandValue(cmd command.Command, validate func(command.Command) error) error {
	if validate == nil {
		return nil
	}

	if err := validate(cmd); err != nil {
		return errorfamily.WrapRejection(err, validationCode, "request validation failed")
	}

	return nil
}

// validateQueryValue is the query-side mirror of validateCommandValue.
func validateQueryValue(qry query.Query, validate func(query.Query) error) error {
	if validate == nil {
		return nil
	}

	if err := validate(qry); err != nil {
		return errorfamily.WrapRejection(err, validationCode, "request validation failed")
	}

	return nil
}

// validationCode is the error code the endpoint pipeline's validation stage
// attaches (root package, options_validate.go). Kept identical so a client can
// branch on one code regardless of which transport served the request.
const validationCode = "cqrshtmx.validate.failed"
