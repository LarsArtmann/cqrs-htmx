package cqrshtmx

import (
	"errors"
	"fmt"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// WildcardDomain is the authorization domain that matches every tenant. A
// policy or grant in this domain applies installation-wide.
//
// It mirrors identitymodel.WildcardDomain. The literal cannot be imported from
// there: this package is deliberately domain-free and must not pull the Casbin
// dependency into every consumer, so the string is duplicated by design and
// pinned by a test.
const WildcardDomain = "*"

// Resource identifies the object an authorization question is about.
//
// Type and ID are separate fields even though today's RBAC model flattens them
// into one Casbin object string: a relationship-based engine needs to know the
// object's kind to traverse edges, and splitting the field later would break
// every caller. Most RBAC callers set only Type (for example "orders"), which
// renders exactly as before.
type Resource struct {
	// Type is the kind of object: "orders", "invoice", "tenant".
	Type string

	// ID is the specific instance. Empty means the collection or "any".
	ID string
}

// String renders the resource the way the RBAC model stores it: "type:id",
// "type" when no ID is given, or "id" when no type is given. A zero Resource
// renders as the empty string, which the wildcard matcher treats as "any".
func (r Resource) String() string {
	switch {
	case r.Type == "":
		return r.ID
	case r.ID == "":
		return r.Type
	default:
		return r.Type + ":" + r.ID
	}
}

// Request is one authorization question: may Subject perform Action on
// Resource within Domain?
//
// It is a plain value so it can be logged, asserted in tests, and passed across
// module boundaries without either side importing the other.
type Request struct {
	// Subject is the acting identity — a user ID, bot ID, or service name.
	Subject string

	// Domain is the tenant/scope the question is asked in. Empty means the
	// wildcard domain (see WildcardDomain).
	Domain string

	// Resource is the object being acted on.
	Resource Resource

	// Action is the verb: "read", "write", "execute", or a domain verb.
	Action string
}

// DecisionReason explains why a Decision came out the way it did. It is the
// part consumers actually switch on — a bare boolean cannot tell "you are not
// logged in" from "you are logged in but not allowed".
type DecisionReason string

const (
	// ReasonAllowed means a policy rule permitted the request.
	ReasonAllowed DecisionReason = "allowed"

	// ReasonDeniedByPolicy means a policy rule matched and denied the request.
	ReasonDeniedByPolicy DecisionReason = "denied_by_policy"

	// ReasonNoMatchingPolicy means no rule spoke to the request at all. For a
	// default-deny engine this is a denial; the distinction is what tells an
	// operator "your grant is missing" from "you were explicitly refused".
	ReasonNoMatchingPolicy DecisionReason = "no_matching_policy"

	// ReasonNotAuthenticated means the question had no subject to ask about.
	ReasonNotAuthenticated DecisionReason = "not_authenticated"
)

// Decision is the answer to a [Request].
type Decision struct {
	// Allowed is the verdict.
	Allowed bool

	// Reason explains the verdict.
	Reason DecisionReason

	// MatchedRules carries the policy rows the engine matched, when the engine
	// exposes them. Nil is normal: a bare Enforcer cannot name its rules.
	MatchedRules []string
}

// PermissionChecker is the authorization seam.
//
// It is primitive-shaped on purpose: the identity model's Authz engine, a
// consumer's own store, and a test stub all satisfy it without importing this
// package, and this package never imports an engine. That is what keeps Casbin
// out of every consumer's dependency graph.
type PermissionChecker interface {
	// Check reports whether subject may perform action on resource within
	// domain. matchedRules carries the policy rows that decided it, when the
	// engine can name them; nil is fine.
	Check(subject, domain, resource, action string) (allowed bool, matchedRules []string, err error)
}

// Decide asks checker the question in req and normalizes the answer.
//
// A request with no subject is answered without consulting the engine:
// [ReasonNotAuthenticated] with Allowed false. An engine failure is returned as
// an error, never as a silent denial — a broken policy engine must not look
// like a permissions problem.
//
// An empty Request.Domain is resolved to [WildcardDomain].
func Decide(checker PermissionChecker, req Request) (Decision, error) {
	if req.Subject == "" {
		return Decision{Allowed: false, Reason: ReasonNotAuthenticated, MatchedRules: nil}, nil
	}

	if checker == nil {
		return Decision{}, errorfamily.NewInfrastructure(
			"enforcer_nil",
			"authorization checker is nil",
		).WithCause(ErrEnforcerNil)
	}

	domain := req.Domain
	if domain == "" {
		domain = WildcardDomain
	}

	allowed, matched, err := checker.Check(req.Subject, domain, req.Resource.String(), req.Action)
	if err != nil {
		return Decision{}, errorfamily.NewTransient(
			"enforce_failed",
			fmt.Sprintf("authorization check failed for subject=%s domain=%s resource=%s action=%s",
				req.Subject, domain, req.Resource.String(), req.Action),
		).WithCause(err)
	}

	if allowed {
		return Decision{Allowed: true, Reason: ReasonAllowed, MatchedRules: matched}, nil
	}

	reason := ReasonNoMatchingPolicy
	if len(matched) > 0 {
		reason = ReasonDeniedByPolicy
	}

	return Decision{Allowed: false, Reason: reason, MatchedRules: matched}, nil
}

// IsForbidden reports whether err represents a denied permission, whichever
// module produced it.
//
// It matches on the shared error code rather than sentinel identity, because the
// same denial exists as several separate sentinel values (this package's
// ErrForbidden, identity-model's, usermgmt's alias of it, adminui's) and
// errors.Is only ever matches one of them. Every level of the unwrap chain is
// inspected, so a module that re-codes the denial with its own error code — a
// wrap that necessarily replaces the outer code — is still recognised by its
// cause.
func IsForbidden(err error) bool {
	return carriesCode(err, CodeForbidden)
}

// IsUnauthorized reports whether err represents a missing or invalid
// authentication, whichever module produced it. See [IsForbidden] for why this
// matches on the code and walks the chain.
func IsUnauthorized(err error) bool {
	return carriesCode(err, CodeUnauthorized)
}

// carriesCode reports whether err or any error it wraps declares code.
func carriesCode(err error, code string) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if errorfamily.Code(e) == code {
			return true
		}
	}

	return false
}

// enforcerChecker adapts a variadic [Enforcer] to [PermissionChecker] using the
// four-argument request shape the shipped RBAC model declares
// (subject, domain, object, action).
type enforcerChecker struct {
	enforcer Enforcer
}

func (c enforcerChecker) Check(subject, domain, resource, action string) (bool, []string, error) {
	if c.enforcer == nil {
		return false, nil, errorfamily.NewInfrastructure(
			"enforcer_nil",
			"casbin enforcer is required for authorization",
		).WithCause(ErrEnforcerNil)
	}

	ok, err := enforceRequest(c.enforcer, subject, domain, resource, action)
	if err != nil {
		return false, nil, errorfamily.NewTransient(
			"enforce_failed",
			fmt.Sprintf("casbin enforce failed for subject=%s domain=%s resource=%s action=%s",
				subject, domain, resource, action),
		).WithCause(err)
	}

	return ok, nil, nil
}

// AuthorizeRequest returns a HandlerOption that authorizes the request with a
// caller-supplied question instead of the static resource/action pair
// [Authorize] takes.
//
// The resolver runs before the body is decoded, so it can only inspect the
// request (path values, query, headers, session). Leave Request.Subject empty
// to use the authenticated user from the request context; set it explicitly to
// authorize a bot or service identity.
//
//	app.Command("DeleteOrder",
//	    cqrshtmx.AuthorizeRequest(func(r *http.Request) (cqrshtmx.Request, error) {
//	        return cqrshtmx.Request{
//	            Domain:   tenantFromHost(r.Host),
//	            Resource: cqrshtmx.Resource{Type: "orders", ID: chi.URLParam(r, "id")},
//	            Action:   "delete",
//	        }, nil
//	    }),
//	)
func AuthorizeRequest(resolve func(*http.Request) (Request, error)) HandlerOption {
	return func(config *handlerConfig) {
		config.authzRequest = resolve
		config.authMode = authAuthorized
	}
}

// AuthorizeIn returns a HandlerOption that checks permissions within a fixed
// domain: the tenant-scoped counterpart of [Authorize].
//
// [Authorize] asks the question in the wildcard domain, which only global
// policies answer. Use AuthorizeIn whenever the answer depends on tenant
// membership.
func AuthorizeIn(domain, resource, action string) HandlerOption {
	return func(config *handlerConfig) {
		config.domain = domain
		config.resource = resource
		config.action = action
		config.authMode = authAuthorized
	}
}

// authorizeWithRequest resolves and answers a config.authzRequest question,
// returning the same error shape the rest of the authorization path uses.
func (a *App) authorizeWithRequest(r *http.Request, config *handlerConfig) error {
	req, err := config.authzRequest(r)
	if err != nil {
		return errorfamily.Wrapf(
			err, errorfamily.Classify(err), "cqrshtmx.handler.authz_request",
			"resolve authorization request for %s %s", r.Method, r.URL.Path,
		)
	}

	if req.Subject == "" {
		req.Subject = UserIDFromContext(r.Context()).String()
	}

	decision, err := Decide(enforcerChecker{enforcer: a.enforcer}, req)
	if err != nil {
		return err
	}

	if decision.Allowed {
		return nil
	}

	return decisionError(decision, req)
}

// decisionError turns a denial into the classified error the error handler maps
// to 401 or 403.
func decisionError(decision Decision, req Request) error {
	sentinel := ErrForbidden
	if decision.Reason == ReasonNotAuthenticated {
		sentinel = ErrUnauthorized
	}

	return errorfamily.NewRejection(
		errorfamily.Code(sentinel),
		fmt.Sprintf("subject=%q domain=%q resource=%q action=%q: %s",
			req.Subject, req.Domain, req.Resource.String(), req.Action, decision.Reason),
	).WithCause(sentinel)
}
