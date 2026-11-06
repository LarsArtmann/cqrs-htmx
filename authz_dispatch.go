package cqrshtmx

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// DispatchKind distinguishes the two dispatcher surfaces a declaration covers.
type DispatchKind string

const (
	// DispatchCommandKind is the command dispatcher.
	DispatchCommandKind DispatchKind = "command"

	// DispatchQueryKind is the query dispatcher.
	DispatchQueryKind DispatchKind = "query"
)

// The dispatch-boundary action vocabulary. A command is an intent to change
// something ("execute"); a query is an intent to read ("read"). A consumer that
// needs a domain verb exits through [PolicyContext], not by widening the ask.
const (
	// ActionExecute is the action a command dispatch asks for.
	ActionExecute = "execute"

	// ActionRead is the action a query dispatch asks for.
	ActionRead = "read"
)

// DispatchObject renders the policy object for a dispatch: "command/Name" or
// "query/Name".
//
// The delimiter is a slash because the shipped Casbin matcher spells it that way
// safely: under keyMatch only "*" is special, so a slash stays a literal and
// "command/*" is usable as a prefix. A colon would become a [^/]+ wildcard under
// keyMatch2 and a dot would become a regexp any-character, either of which
// silently widens the row. See ADR 0063 and the spelling table in
// identity-model/authz_matcher_test.go.
func DispatchObject(kind DispatchKind, name string) string {
	return string(kind) + "/" + name
}

func dispatchAction(kind DispatchKind) string {
	if kind == DispatchQueryKind {
		return ActionRead
	}

	return ActionExecute
}

// PolicyContext is implemented by a command or query that can describe its own
// authorization question more precisely than the derived "command/Name" ask.
//
// It is a closed interface on purpose (ADR 0058): the question is a
// [Request] — subject, domain, resource, action — never a map of payload
// fields. A map would let the policy file's shape drift with the payload schema
// and turn this into attribute-based authorization without the engine that makes
// attribute-based authorization sound.
//
// Returning a zero Request (or not implementing the interface at all) keeps the
// derived question: the gate's DispatchObject, [ActionExecute] or [ActionRead],
// and the authenticated subject from the context.
//
//	err := gate.RegisterCommand(disp, cqrshtmx.PostureRoleGated, "moves money",
//	    func(ctx context.Context, cmd *refundCmd) error { ... })
//
//	func (c *refundCmd) PolicyContext(_ context.Context) cqrshtmx.Request {
//	    return cqrshtmx.Request{
//	        Domain:   c.TenantID,
//	        Resource: cqrshtmx.Resource{Type: "invoices", ID: c.InvoiceID},
//	        Action:   "refund",
//	    }
//	}
type PolicyContext interface {
	// PolicyContext returns the authorization question this dispatch asks. An
	// empty Subject leaves the authenticated subject from the context in place.
	PolicyContext(ctx context.Context) Request
}

// DispatchOutcome is one gate verdict, handed to the observation seam so
// consumers can wire metrics, OTel spans, or audit without the library choosing
// a backend.
type DispatchOutcome struct {
	// Kind is the dispatcher surface the dispatch used.
	Kind DispatchKind

	// Name is the command or query type name, without the object prefix.
	Name string

	// Posture is the posture that was enforced.
	Posture Posture

	// Request is the question that was asked.
	Request Request

	// Decision is the answer. Reason is the field to branch on.
	Decision Decision

	// Audit means the denial did not stop the dispatch.
	Audit bool
}

// DispatchGateOption configures a dispatch gate.
type DispatchGateOption func(*dispatchGate)

// AuditOnly makes the gate observe denials instead of enforcing them: each
// denial is reported through [OnDecision] and logged, and the dispatch proceeds.
//
// It is a migration tool, not a posture (ADR 0063 §5). A module may sit in audit
// mode for at most one wave; the maturity gate records that it is there so it
// cannot become permanent. Audit mode still reports "would have denied" at the
// same fidelity as enforcement, so the wave's measurement is the gate's real
// output rather than a guess.
func AuditOnly() DispatchGateOption {
	return func(g *dispatchGate) { g.auditOnly = true }
}

// OnDecision installs an observer for every verdict the gate reaches, allowed or
// denied. Nil (the default) costs nothing.
func OnDecision(observe func(DispatchOutcome)) DispatchGateOption {
	return func(g *dispatchGate) { g.observe = observe }
}

// DispatchGate authorizes command and query dispatches at the boundary itself,
// so a dispatch that never travels through an HTTP route is still authorized.
//
// It is the declaring half of the feature: [DispatchGate.RegisterCommand] and
// [DispatchGate.RegisterQuery] bind a handler to the posture it requires, and
// [DispatchGate.Validate] checks the declaration set at boot. Its middlewares
// are the enforcing half. A gate with no declarations is still useful — every
// dispatch asks for "command/Name" — which is the smallest adoption step:
//
//	gate := cqrshtmx.NewDispatchGate(cqrshtmx.OnDecision(metrics.Record))
//	cmdDisp.Use(gate.CommandMiddleware())
//	qryDisp.Use(gate.QueryMiddleware())
//
// The two middlewares must be the OUTERMOST entries in the chain so that retry,
// circuit-breaker, and metrics middleware never observe an authorization
// decision as a retryable failure. See docs/guides/dispatch-middleware-ordering.md.
type DispatchGate struct {
	*dispatchGate

	// mu guards the write path only. Dispatches never take it.
	mu sync.Mutex

	// rows is an immutable declaration table published by Declare. Readers load
	// the pointer atomically and read a table that is never mutated in place, so
	// a dispatch pays no lock. A gate is declared at boot and read on every
	// dispatch; a mutex on that read path would serialise every command in the
	// process on one shared lock for the privilege of re-reading constants.
	rows atomic.Pointer[map[dispatchKey]dispatchRow]
}

// dispatchKey identifies one declared type in one dispatcher surface.
type dispatchKey struct {
	kind DispatchKind
	name string
}

// dispatchRow is one resolved declaration. The policy object is rendered once, at
// declaration time, because it is a fixed function of the type: rendering it per
// dispatch allocates a string on every call for a value that cannot change.
type dispatchRow struct {
	decl   DispatchPosture
	object string
}

// DispatchPosture is one declaration: what a command or query type requires of
// its caller, wherever it is dispatched from.
type DispatchPosture struct {
	// Kind is the dispatcher surface the declaration applies to.
	Kind DispatchKind

	// Name is the command or query type name, without the object prefix.
	Name string

	// Posture is what the type requires of its caller.
	Posture Posture

	// Why is the reason this posture is correct. It is required: a posture
	// without a reason is a guess, and a guess reviewed later is
	// indistinguishable from a decision.
	Why string
}

// String renders the declaration as its policy identity: "command/CreateItem".
func (d DispatchPosture) String() string { return DispatchObject(d.Kind, d.Name) }

// dispatchGate is the shared runtime: checker, options, and the verdict path.
type dispatchGate struct {
	checker   PermissionChecker
	auditOnly bool
	observe   func(DispatchOutcome)
	logger    *slog.Logger
}

// NewDispatchGate builds a declaration surface and the enforcing middlewares
// that go with it. A nil checker is an error rather than a deferral: the fastest
// way to be sure a gate cannot silently allow everything is to refuse to build
// one without an engine.
func NewDispatchGate(checker PermissionChecker, opts ...DispatchGateOption) (*DispatchGate, error) {
	if checker == nil {
		return nil, errorfamily.NewInfrastructure(
			"enforcer_nil",
			"dispatch gate requires a permission checker",
		).WithCause(ErrEnforcerNil)
	}

	return &DispatchGate{
		dispatchGate: newDispatchGate(checker, opts...),
	}, nil
}

func newDispatchGate(checker PermissionChecker, opts ...DispatchGateOption) *dispatchGate {
	g := &dispatchGate{checker: checker, logger: slog.Default()}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// RegisterCommand registers a typed command handler together with the posture it
// requires. Registering is declaring: the dispatcher exposes no enumeration API
// (only Use/Register/Dispatch/Close), so a gate that did not learn the posture
// at registration time could never learn it at all.
//
// It is a package-level function rather than a method because Go methods cannot
// carry type parameters, the same constraint that makes [CommandTyped] a
// function. Registration order relative to `Use` does not matter, but the
// handler must be registered before anything dispatches the type.
//
//	err := cqrshtmx.RegisterCommand(gate, cmdDisp, "CreateItem",
//	    cqrshtmx.PostureRoleGated, "creates an item",
//	    func(ctx context.Context, cmd *createItemCmd) error { ... })
func RegisterCommand[T command.Command](
	gate *DispatchGate,
	disp *command.Dispatcher,
	cmdType command.Type,
	posture Posture,
	why string,
	handler func(ctx context.Context, cmd T) error,
) error {
	if err := command.RegisterTyped(disp, cmdType, handler); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Classify(err),
			"cqrshtmx.dispatch_gate.register_failed",
			"registering command type %s", cmdType)
	}

	return gate.Declare(DispatchPosture{
		Kind:    DispatchCommandKind,
		Name:    string(cmdType),
		Posture: posture,
		Why:     why,
	})
}

// RegisterQuery registers a typed query handler together with the posture it
// requires. See [RegisterCommand] for why registration is the declaration.
func RegisterQuery[Q query.Query, R any](
	gate *DispatchGate,
	disp *query.Dispatcher,
	qryType query.Type,
	posture Posture,
	why string,
	handler func(ctx context.Context, q Q) (R, error),
) error {
	if err := query.RegisterTyped(disp, qryType, handler); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Classify(err),
			"cqrshtmx.dispatch_gate.register_failed",
			"registering query type %s", qryType)
	}

	return gate.Declare(DispatchPosture{
		Kind:    DispatchQueryKind,
		Name:    string(qryType),
		Posture: posture,
		Why:     why,
	})
}

// Declare records one posture without registering a handler, for wiring that
// registers through the raw dispatcher (a `command.Handler` that cannot be typed,
// or a registration owned by another module).
//
// An amendment that would change an existing declaration is rejected: a posture
// two code paths disagree about is exactly the drift this surface exists to
// prevent.
func (g *DispatchGate) Declare(row DispatchPosture) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	current := g.table()

	if existing, ok := current[dispatchKey{kind: row.Kind, name: row.Name}]; ok && existing.decl != row {
		return errorfamily.NewRejection(
			"invalid_argument",
			fmt.Sprintf("%s is declared twice with different postures (%q vs %q)",
				row, existing.decl.Posture, row.Posture),
		)
	}

	next := make(map[dispatchKey]dispatchRow, len(current)+1)
	maps.Copy(next, current)
	next[dispatchKey{kind: row.Kind, name: row.Name}] = dispatchRow{decl: row, object: row.String()}

	g.rows.Store(&next)

	return nil
}

// table returns the published declaration table. The returned map is immutable:
// Declare copies before it writes, so a reader never observes a half-built map
// and never needs a lock.
func (g *DispatchGate) table() map[dispatchKey]dispatchRow {
	if published := g.rows.Load(); published != nil {
		return *published
	}

	return nil
}

// Manifest returns the declared postures, sorted by kind then name. It is the
// artifact the L5 maturity level is measured by and the input to the
// policy-row linter.
func (g *DispatchGate) Manifest() []DispatchPosture {
	table := g.table()

	out := make([]DispatchPosture, 0, len(table))
	for _, resolved := range table {
		out = append(out, resolved.decl)
	}

	sortDispatchPostures(out)

	return out
}

// Validate checks the declaration set and returns one error naming every
// problem. Call it once after wiring, before serving.
//
// What it can and cannot prove is worth stating: it validates the declarations
// the gate was told about (valid posture, a reason, no contradictory
// duplicates, at least one declaration when a protected posture is used), but it
// cannot detect a handler registered directly on the dispatcher with no
// declaration — the dispatcher cannot be enumerated. That case is covered at
// dispatch time instead: an undeclared type is denied (fail-closed) and named in
// the denial, and the `check-permission-surface` gate flags the raw
// registration.
func (g *DispatchGate) Validate() error {
	rows := g.Manifest()

	var problems []string

	if len(rows) == 0 {
		problems = append(problems,
			"no dispatch postures declared: every dispatch will be denied; either declare the "+
				"postures this dispatcher serves or do not install the gate")
	}

	for _, row := range rows {
		if !row.Posture.Valid() || row.Posture == PostureUngated {
			problems = append(problems, fmt.Sprintf(
				"%s declares posture %q, which a dispatch gate cannot enforce",
				row, row.Posture))
		}

		if strings.TrimSpace(row.Why) == "" {
			problems = append(problems, fmt.Sprintf("%s has no reason; a posture without a reason is a guess", row))
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return errorfamily.NewRejection("invalid_argument",
		"dispatch posture validation failed: "+strings.Join(problems, "; "))
}

// CommandMiddleware returns the command-dispatcher middleware enforcing this
// gate. Install it outermost.
func (g *DispatchGate) CommandMiddleware() command.Middleware {
	return commandMiddlewareFor(g.dispatchGate, g.enforcedPosture)
}

// QueryMiddleware returns the query-dispatcher middleware enforcing this gate.
// Install it outermost.
func (g *DispatchGate) QueryMiddleware() query.Middleware {
	return queryMiddlewareFor(g.dispatchGate, g.enforcedPosture)
}

// enforcedPosture returns the posture a declared gate applies to a type. A type
// the gate was never told about is gated: fail-closed is the only default that
// cannot silently ship an open door.
func (g *DispatchGate) enforcedPosture(kind DispatchKind, name string) (dispatchRow, bool) {
	row, ok := g.table()[dispatchKey{kind: kind, name: name}]

	return row, ok
}

// AuthorizeCommand returns a command middleware that asks "may this actor run
// command/Name?" for every dispatch, wherever it comes from.
//
// It is the declaration-free form of [DispatchGate]: every command is gated, no
// posture table is needed. Use [NewDispatchGate] when some commands are public or
// only authenticated, or when you want the declaration set validated at boot.
//
//	cmdDisp.Use(cqrshtmx.AuthorizeCommand(svc.Authz()))
func AuthorizeCommand(checker PermissionChecker, opts ...DispatchGateOption) command.Middleware {
	g := newDispatchGate(checker, opts...)

	return commandMiddlewareFor(g, allGated)
}

// AuthorizeQuery returns the query counterpart of [AuthorizeCommand]: every
// query asks "may this actor run query/Name?".
func AuthorizeQuery(checker PermissionChecker, opts ...DispatchGateOption) query.Middleware {
	g := newDispatchGate(checker, opts...)

	return queryMiddlewareFor(g, allGated)
}

// allGated is the posture lookup of the declaration-free form: every type is
// role-gated, and nothing is claimed about a declaration set.
func allGated(DispatchKind, string) (dispatchRow, bool) {
	return dispatchRow{decl: DispatchPosture{Posture: PostureRoleGated}}, true
}

// postureLookup resolves the declaration for one dispatch.
type postureLookup func(kind DispatchKind, name string) (dispatchRow, bool)

func commandMiddlewareFor(g *dispatchGate, lookup postureLookup) command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			if err := g.authorize(ctx, DispatchCommandKind, string(cmd.Type()), cmd, lookup); err != nil {
				return err
			}

			return next(ctx, cmd)
		}
	}
}

func queryMiddlewareFor(g *dispatchGate, lookup postureLookup) query.Middleware {
	return func(next query.Handler) query.Handler {
		return func(ctx context.Context, qry query.Query) (any, error) {
			if err := g.authorize(ctx, DispatchQueryKind, string(qry.Type()), qry, lookup); err != nil {
				return nil, err
			}

			return next(ctx, qry)
		}
	}
}

// authorize is the one verdict path both dispatchers share.
func (g *dispatchGate) authorize(
	ctx context.Context,
	kind DispatchKind,
	name string,
	payload any,
	lookup postureLookup,
) error {
	row, declared := lookup(kind, name)

	posture := row.decl.Posture
	if !declared {
		posture = PostureRoleGated
	}

	object := row.object
	if object == "" {
		object = DispatchObject(kind, name)
	}

	req := deriveDispatchRequest(ctx, object, dispatchAction(kind), DispatchSubject(ctx), payload)

	if !declared {
		return g.settle(ctx, kind, name, PostureRoleGated, req,
			Decision{Allowed: false, Reason: ReasonNoMatchingPolicy, MatchedRules: nil},
			errorfamily.NewRejection("forbidden", fmt.Sprintf(
				"no dispatch posture declared for %s; register it through DispatchGate.RegisterCommand",
				req.Resource.String())).WithCause(ErrForbidden))
	}

	// A public posture asks nothing of the caller and never reaches the engine:
	// declaring it public is the whole permission decision.
	if posture == PosturePublic {
		return g.settle(ctx, kind, name, posture, req,
			Decision{Allowed: true, Reason: ReasonAllowed, MatchedRules: nil}, nil)
	}

	if req.Subject == "" {
		return g.settle(ctx, kind, name, posture, req,
			Decision{Allowed: false, Reason: ReasonNotAuthenticated, MatchedRules: nil},
			denyDispatch(req, Decision{Allowed: false, Reason: ReasonNotAuthenticated, MatchedRules: nil}))
	}

	// An authenticated posture wants an identity and nothing more.
	if posture == PostureAuthenticated {
		return g.settle(ctx, kind, name, posture, req,
			Decision{Allowed: true, Reason: ReasonAllowed, MatchedRules: nil}, nil)
	}

	if g.checker == nil {
		return errorfamily.NewInfrastructure(
			"enforcer_nil",
			"dispatch gate has no checker for "+req.Resource.String(),
		).WithCause(ErrEnforcerNil)
	}

	decision, err := Decide(g.checker, req)
	if err != nil {
		return err
	}

	var denial error
	if !decision.Allowed {
		denial = denyDispatch(req, decision)
	}

	return g.settle(ctx, kind, name, posture, req, decision, denial)
}

// settle reports the verdict, applies audit mode, and returns the denial the
// dispatch should see (nil when it may proceed).
//
// The outcome value is built only when somebody is watching. Without an observer
// and without a denial there is nothing to report, and building a seven-field
// record per dispatch is exactly the kind of cost a hot path should not pay for
// a nil callback.
func (g *dispatchGate) settle(
	ctx context.Context,
	kind DispatchKind,
	name string,
	posture Posture,
	req Request,
	decision Decision,
	denial error,
) error {
	if denial == nil {
		if g.observe != nil {
			g.report(DispatchOutcome{
				Kind: kind, Name: name, Posture: posture, Request: req,
				Decision: decision, Audit: false,
			})
		}

		return nil
	}

	if !g.auditOnly {
		if g.observe != nil {
			g.report(DispatchOutcome{
				Kind: kind, Name: name, Posture: posture, Request: req,
				Decision: decision, Audit: false,
			})
		}

		return denial
	}

	g.report(DispatchOutcome{
		Kind: kind, Name: name, Posture: posture, Request: req,
		Decision: decision, Audit: true,
	})

	if g.logger != nil {
		g.logger.WarnContext(ctx, "dispatch authorization denied (audit only)",
			slog.String("kind", string(kind)),
			slog.String("type", name),
			slog.String("posture", string(posture)),
			slog.String("subject", req.Subject),
			slog.String("domain", req.Domain),
			slog.String("resource", req.Resource.String()),
			slog.String("action", req.Action),
			slog.String("reason", string(decision.Reason)),
		)
	}

	return nil
}

func (g *dispatchGate) report(outcome DispatchOutcome) {
	if g.observe != nil {
		g.observe(outcome)
	}
}

// DispatchSubject returns the identity a dispatch is authorized as: the
// authenticated user when the context carries one, otherwise the effective
// actor (a bot, a service, or a system identity).
//
// The order matters. A session-derived context carries both, and the user ID is
// the spelling the route layer uses, so route policy rows and dispatch policy
// rows describe the same subject. Only a caller with no user — a bootstrap, a
// scheduled job, an admin CLI — falls through to the actor, which is what makes
// those paths expressible at all: put an actor in the context and grant it
// `command/*`.
func DispatchSubject(ctx context.Context) string {
	if userID := UserIDFromContext(ctx); !userID.IsZero() {
		return userID.String()
	}

	if actorID := ActorIDFromContext(ctx); !actorID.IsZero() {
		// A human actor is spelled the same way a route subject is, so one
		// policy row answers both layers. Machine actors keep their kind
		// prefix, because "scheduler" and a bot token hash must not collide
		// with a user id in one subject namespace.
		if actorID.Kind() == id.ActorUser {
			return actorID.Raw()
		}

		return actorID.PrefixedString()
	}

	return ""
}

// deriveDispatchRequest composes the question for one dispatch, letting a
// [PolicyContext] implementation overlay the parts it knows better.
func deriveDispatchRequest(
	ctx context.Context,
	object, action, subject string,
	payload any,
) Request {
	req := Request{
		Subject:  subject,
		Domain:   WildcardDomain,
		Resource: Resource{Type: object, ID: ""},
		Action:   action,
	}

	describer, ok := payload.(PolicyContext)
	if !ok {
		return req
	}

	overlay := describer.PolicyContext(ctx)
	if overlay.Subject != "" {
		req.Subject = overlay.Subject
	}

	if overlay.Domain != "" {
		req.Domain = overlay.Domain
	}

	if overlay.Resource.Type != "" || overlay.Resource.ID != "" {
		req.Resource = overlay.Resource
	}

	if overlay.Action != "" {
		req.Action = overlay.Action
	}

	return req
}

// denyDispatch renders a denial in the vocabulary ADR 0057 established, so
// IsForbidden/IsUnauthorized and every existing classifier keep working and a
// dispatch denial reads the same as a route denial.
func denyDispatch(req Request, decision Decision) error {
	if decision.Reason == ReasonNotAuthenticated {
		return errorfamily.NewRejection("unauthorized", "dispatch "+req.Resource.String()).
			WithCause(ErrUnauthorized)
	}

	return errorfamily.NewRejection("forbidden",
		fmt.Sprintf("subject=%s domain=%s resource=%s action=%s reason=%s",
			req.Subject, req.Domain, req.Resource.String(), req.Action, decision.Reason)).
		WithCause(ErrForbidden)
}

// sortDispatchPostures orders declarations by kind then name, so manifests and
// their test fixtures are stable.
func sortDispatchPostures(rows []DispatchPosture) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].less(rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func (d DispatchPosture) less(other DispatchPosture) bool {
	if d.Kind != other.Kind {
		return d.Kind < other.Kind
	}

	return d.Name < other.Name
}
