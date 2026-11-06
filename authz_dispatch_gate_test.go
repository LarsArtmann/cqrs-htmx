package cqrshtmx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// gateCheck records one authorization question and answers it with a fixed
// verdict. It is the only way to prove a posture skipped the engine: a
// deny-everything engine that was never called denies nothing.
type gateCheck struct {
	allow bool
	err   error

	calls    int
	subject  string
	domain   string
	resource string
	action   string
}

func (c *gateCheck) Check(subject, domain, resource, action string) (bool, []string, error) {
	c.calls++
	c.subject, c.domain, c.resource, c.action = subject, domain, resource, action

	return c.allow, nil, c.err
}

// gatedQuery is the query counterpart of gatedCmd.
type gatedQuery struct{}

func (gatedQuery) Type() query.Type { return "ListItems" }

// refundCmd describes its own question, proving the PolicyContext seam reaches
// the engine instead of the derived command/Name ask.
type refundCmd struct {
	aggID    id.StreamID
	cmdID    id.CommandID
	tenantID string
	invoice  string
}

func (c *refundCmd) Type() command.Type    { return "RefundInvoice" }
func (c *refundCmd) StreamID() id.StreamID { return c.aggID }
func (c *refundCmd) ID() id.CommandID      { return c.cmdID }

func (c *refundCmd) PolicyContext(context.Context) Request {
	return Request{
		Domain:   c.tenantID,
		Resource: Resource{Type: "invoices", ID: c.invoice},
		Action:   "refund",
	}
}

func authedCtx() context.Context {
	return WithUserID(context.Background(), MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2"))
}

// TestDispatchGateAllowsWithPolicy proves the allowed path end to end: a
// service-level dispatch with an authenticated actor and a permitting engine
// reaches the handler, and the question the engine saw is the derived one.
func TestDispatchGateAllowsWithPolicy(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: true}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	ran := false

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		ran = true

		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(authedCtx(), &gatedCmd{}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if !ran {
		t.Fatal("handler did not run")
	}

	if checker.calls != 1 {
		t.Fatalf("checker calls = %d, want 1", checker.calls)
	}

	if checker.resource != "command/CreateItem" || checker.action != ActionExecute {
		t.Fatalf("question = %s/%s, want command/CreateItem/%s", checker.resource, checker.action, ActionExecute)
	}

	if checker.domain != WildcardDomain {
		t.Fatalf("domain = %q, want wildcard", checker.domain)
	}
}

// TestDispatchGateDeniesWithoutPolicy proves the closing of the hole: the same
// service-level dispatch that succeeded before the gate now fails with a
// recognisable denial, and the handler never runs.
func TestDispatchGateDeniesWithoutPolicy(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	ran := false

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		ran = true

		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := disp.Dispatch(authedCtx(), &gatedCmd{})
	if err == nil {
		t.Fatal("dispatch allowed, want denial")
	}

	if !IsForbidden(err) {
		t.Fatalf("IsForbidden = false for %v", err)
	}

	if IsUnauthorized(err) {
		t.Fatal("a policy denial must not read as a missing identity")
	}

	if MapError(err) != http.StatusForbidden {
		t.Fatalf("MapError = %d, want %d", MapError(err), http.StatusForbidden)
	}

	if ran {
		t.Fatal("handler ran despite the denial")
	}
}

// TestDispatchGateRequiresIdentity proves the unauthenticated path answers 401
// without consulting the engine: there is no subject to ask about.
func TestDispatchGateRequiresIdentity(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: true}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := disp.Dispatch(context.Background(), &gatedCmd{})
	if !IsUnauthorized(err) {
		t.Fatalf("IsUnauthorized = false for %v", err)
	}

	if MapError(err) != http.StatusUnauthorized {
		t.Fatalf("MapError = %d, want %d", MapError(err), http.StatusUnauthorized)
	}

	if checker.calls != 0 {
		t.Fatalf("engine consulted %d times with no subject, want 0", checker.calls)
	}
}

// TestDispatchGateEngineFailureIsNotADenial proves a broken policy engine
// surfaces as infrastructure (503), never as a silent 403.
func TestDispatchGateEngineFailureIsNotADenial(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{err: errors.New("casbin exploded")}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	err := disp.Dispatch(authedCtx(), &gatedCmd{})
	if err == nil {
		t.Fatal("dispatch succeeded despite an engine failure")
	}

	if IsForbidden(err) || IsUnauthorized(err) {
		t.Fatalf("engine failure classified as a permission denial: %v", err)
	}

	if MapError(err) != http.StatusServiceUnavailable {
		t.Fatalf("MapError = %d, want %d", MapError(err), http.StatusServiceUnavailable)
	}
}

// TestDispatchGateCoversTheQueryDispatcher proves the query mirror asks
// "query/Name" with the read action.
func TestDispatchGateCoversTheQueryDispatcher(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: true}
	disp := query.NewDispatcher()
	disp.Use(AuthorizeQuery(checker))

	ran := false

	if err := query.RegisterTyped(disp, "ListItems", func(context.Context, gatedQuery) (int, error) {
		ran = true

		return 7, nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	got, err := query.DispatchTyped[int](authedCtx(), disp, gatedQuery{})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if got != 7 || !ran {
		t.Fatalf("result = %d ran = %v, want 7/true", got, ran)
	}

	if checker.resource != "query/ListItems" || checker.action != ActionRead {
		t.Fatalf("question = %s/%s, want query/ListItems/%s", checker.resource, checker.action, ActionRead)
	}
}

// TestPolicyContextOverlaysTheQuestion proves a command can describe its own
// question: the engine sees the overlay, not the derived command/Name.
func TestPolicyContextOverlaysTheQuestion(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: true}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	if err := command.RegisterTyped(disp, "RefundInvoice", func(context.Context, *refundCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	cmd := &refundCmd{tenantID: "tenant-1", invoice: "inv-9"}
	if err := disp.Dispatch(authedCtx(), cmd); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if checker.domain != "tenant-1" {
		t.Errorf("domain = %q, want tenant-1", checker.domain)
	}

	if checker.resource != "invoices:inv-9" {
		t.Errorf("resource = %q, want invoices:inv-9", checker.resource)
	}

	if checker.action != "refund" {
		t.Errorf("action = %q, want refund", checker.action)
	}
}

// TestDispatchGateIsOutermostInTheChain proves a denial never reaches inner
// middleware: retry, circuit breaker and metrics must not see an authorization
// decision as a retryable or countable failure.
func TestDispatchGateIsOutermostInTheChain(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}
	disp := command.NewDispatcher()

	inner := 0
	retryShaped := func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			inner++

			return next(ctx, cmd)
		}
	}

	// First added is outermost: the gate wraps the counting middleware.
	disp.Use(AuthorizeCommand(checker), retryShaped)

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(authedCtx(), &gatedCmd{}); err == nil {
		t.Fatal("dispatch allowed, want denial")
	}

	if inner != 0 {
		t.Fatalf("inner middleware ran %d times behind a denial, want 0", inner)
	}
}

// TestRouteGateReportsTheDenialOnce proves a route gate above the dispatch gate
// keeps ownership of the denial: the dispatch gate is never reached, so a single
// denial is reported rather than two.
func TestRouteGateReportsTheDenialOnce(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	app, err := New(Config{Commands: disp, Enforcer: &countingDenyEnforcer{}})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	route := app.Command("CreateItem",
		Authorize("items", "create"),
		DecodeJSON(func(_ struct{}) (command.Command, error) { return &gatedCmd{}, nil }),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{}`))
	req = req.WithContext(WithUserID(req.Context(), MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")))
	route(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("route status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if checker.calls != 0 {
		t.Fatalf("dispatch gate consulted %d times behind a route denial, want 0", checker.calls)
	}
}

// TestDispatchGateServesHTTPHandlers proves the gate is live on the App's own
// dispatch paths, not only on direct dispatcher calls.
func TestDispatchGateServesHTTPHandlers(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}
	disp := command.NewDispatcher()
	disp.Use(AuthorizeCommand(checker))

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	app, err := New(Config{Commands: disp})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	route := CommandTyped[*gatedCmd](app, "CreateItem",
		DecodeJSONTyped[*gatedCmd](),
		WithSuccessStatus(http.StatusCreated),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{}`))
	req = req.WithContext(authedCtx())
	route(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("typed route status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// --- declaration surface (DispatchGate) ---

// newDeclaredGate builds a gate with one command posture already registered.
func newDeclaredGate(
	t *testing.T,
	checker PermissionChecker,
	posture Posture,
	why string,
) (*DispatchGate, *command.Dispatcher) {
	t.Helper()

	gate, err := NewDispatchGate(checker)
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	disp := command.NewDispatcher()

	if err := RegisterCommand(gate, disp, "CreateItem", posture, why,
		func(context.Context, *gatedCmd) error { return nil }); err != nil {
		t.Fatalf("register command: %v", err)
	}

	disp.Use(gate.CommandMiddleware())

	return gate, disp
}

// TestDispatchGateValidateAcceptsACompleteDeclaration proves the happy path of
// the boot check.
func TestDispatchGateValidateAcceptsACompleteDeclaration(t *testing.T) {
	t.Parallel()

	gate, _ := newDeclaredGate(t, &gateCheck{allow: true}, PostureRoleGated, "creates an item")

	if err := gate.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	manifest := gate.Manifest()
	if len(manifest) != 1 || manifest[0].String() != "command/CreateItem" {
		t.Fatalf("manifest = %v, want [command/CreateItem]", manifest)
	}
}

// TestDispatchGateValidateFailsClosed proves an unusable declaration set is a
// boot error rather than a runtime surprise.
func TestDispatchGateValidateFailsClosed(t *testing.T) {
	t.Parallel()

	t.Run("no_declarations", func(t *testing.T) {
		t.Parallel()

		gate, err := NewDispatchGate(&gateCheck{allow: true})
		if err != nil {
			t.Fatalf("new gate: %v", err)
		}

		if err := gate.Validate(); err == nil {
			t.Fatal("Validate accepted an empty declaration set")
		}
	})

	t.Run("ungated_posture", func(t *testing.T) {
		t.Parallel()

		gate, _ := newDeclaredGate(t, &gateCheck{allow: true}, PostureUngated, "delegates")

		err := gate.Validate()
		if err == nil {
			t.Fatal("Validate accepted PostureUngated, which a gate cannot enforce")
		}

		if !strings.Contains(err.Error(), "command/CreateItem") {
			t.Errorf("error should name the offending declaration, got: %v", err)
		}
	})

	t.Run("missing_reason", func(t *testing.T) {
		t.Parallel()

		gate, _ := newDeclaredGate(t, &gateCheck{allow: true}, PostureRoleGated, "  ")

		if err := gate.Validate(); err == nil {
			t.Fatal("Validate accepted a posture with no reason")
		}
	})
}

// TestDispatchGateRejectsContradictoryDeclarations proves one type cannot carry
// two postures: that is the drift the declaration surface exists to prevent.
func TestDispatchGateRejectsContradictoryDeclarations(t *testing.T) {
	t.Parallel()

	gate, err := NewDispatchGate(&gateCheck{allow: true})
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	disp := command.NewDispatcher()

	if err := RegisterCommand(gate, disp, "CreateItem", PostureRoleGated, "gated",
		func(context.Context, *gatedCmd) error { return nil }); err != nil {
		t.Fatalf("first registration: %v", err)
	}

	err = RegisterCommand(gate, command.NewDispatcher(), "CreateItem", PosturePublic, "open",
		func(context.Context, *gatedCmd) error { return nil })
	if err == nil {
		t.Fatal("gate accepted a second, contradictory posture for the same type")
	}

	if !strings.Contains(err.Error(), "declared twice") {
		t.Errorf("error should name the contradiction, got: %v", err)
	}
}

// TestRegisterCommandTypeDriftIsCaughtAtDispatch documents where a mismatched
// declaration actually surfaces: the gate cannot tell what a handler's Type()
// answers without a live value (a zero value may be a nil pointer), so the
// dispatcher itself reports the missing handler instead. This test exists so the
// guarantee's boundary is stated rather than assumed.
func TestRegisterCommandTypeDriftIsCaughtAtDispatch(t *testing.T) {
	t.Parallel()

	gate, err := NewDispatchGate(&gateCheck{allow: true})
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	disp := command.NewDispatcher()
	disp.Use(gate.CommandMiddleware())

	if err := RegisterCommand(gate, disp, "WrongName", PostureRoleGated, "typo",
		func(context.Context, *gatedCmd) error { return nil }); err != nil {
		t.Fatalf("register: %v", err)
	}

	// The handler answers to CreateItem; nothing is registered under that name.
	err = disp.Dispatch(authedCtx(), &gatedCmd{})
	if err == nil {
		t.Fatal("a drifted registration dispatched cleanly")
	}

	if IsForbidden(err) || IsUnauthorized(err) {
		t.Fatalf("a missing handler must not read as a permission decision: %v", err)
	}
}

// TestDispatchGatePublicPostureSkipsTheEngine proves a public posture costs one
// map lookup and no engine call, and works with no identity at all.
func TestDispatchGatePublicPostureSkipsTheEngine(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}
	_, disp := newDeclaredGate(t, checker, PosturePublic, "health-style probe")

	if err := disp.Dispatch(context.Background(), &gatedCmd{}); err != nil {
		t.Fatalf("public dispatch: %v", err)
	}

	if checker.calls != 0 {
		t.Fatalf("engine consulted %d times for a public posture, want 0", checker.calls)
	}
}

// TestDispatchGateAuthenticatedPostureNeedsAnIdentity proves an authenticated
// posture refuses a zero subject but asks nothing else, and that a present
// identity is admitted without an engine call.
func TestDispatchGateAuthenticatedPostureNeedsAnIdentity(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}
	_, disp := newDeclaredGate(t, checker, PostureAuthenticated, "any signed-in user")

	err := disp.Dispatch(context.Background(), &gatedCmd{})
	if !IsUnauthorized(err) {
		t.Fatalf("anonymous dispatch = %v, want unauthorized", err)
	}

	if err := disp.Dispatch(authedCtx(), &gatedCmd{}); err != nil {
		t.Fatalf("authenticated dispatch: %v", err)
	}

	if checker.calls != 0 {
		t.Fatalf("engine consulted %d times for an authenticated posture, want 0", checker.calls)
	}
}

// TestDispatchGateUndeclaredTypeFailsClosed proves the half of the guarantee the
// boot check cannot reach: a handler registered straight on the dispatcher, with
// no declaration, is denied and named at dispatch time.
func TestDispatchGateUndeclaredTypeFailsClosed(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: true}
	gate, disp := newDeclaredGate(t, checker, PostureRoleGated, "creates an item")

	// Registered behind the gate's back — the dispatcher cannot be enumerated,
	// so only the dispatch-time check can see this.
	if err := disp.Register("Rogue", func(context.Context, command.Command) error { return nil }); err != nil {
		t.Fatalf("register rogue handler: %v", err)
	}

	err := disp.Dispatch(authedCtx(), rogueCmd{})
	if !IsForbidden(err) {
		t.Fatalf("undeclared dispatch = %v, want a forbidden denial", err)
	}

	if !strings.Contains(err.Error(), "no dispatch posture declared") {
		t.Errorf("error should name the missing declaration, got: %v", err)
	}

	if checker.calls != 0 {
		t.Fatalf("engine consulted %d times for an undeclared type, want 0", checker.calls)
	}

	if got := len(gate.Manifest()); got != 1 {
		t.Fatalf("manifest grew to %d rows, want 1", got)
	}
}

// rogueCmd is the type the behind-the-gate registration answers to.
type rogueCmd struct {
	aggID id.StreamID
	cmdID id.CommandID
}

func (rogueCmd) Type() command.Type      { return command.Type("Rogue") }
func (r rogueCmd) StreamID() id.StreamID { return r.aggID }
func (r rogueCmd) ID() id.CommandID      { return r.cmdID }

// --- audit mode ---

// TestAuditOnlyReportsWithoutDenying proves the migration posture: the denial is
// measured at full fidelity and the dispatch still runs.
func TestAuditOnlyReportsWithoutDenying(t *testing.T) {
	t.Parallel()

	checker := &gateCheck{allow: false}

	var outcomes []DispatchOutcome

	gate, err := NewDispatchGate(checker, AuditOnly(), OnDecision(func(o DispatchOutcome) {
		outcomes = append(outcomes, o)
	}))
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	disp := command.NewDispatcher()
	disp.Use(gate.CommandMiddleware())

	ran := false

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		ran = true

		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(authedCtx(), &gatedCmd{}); err != nil {
		t.Fatalf("audit dispatch returned %v, want nil", err)
	}

	if !ran {
		t.Fatal("audit mode blocked the handler")
	}

	if len(outcomes) != 1 {
		t.Fatalf("observer called %d times, want 1", len(outcomes))
	}

	if !outcomes[0].Audit {
		t.Error("outcome should be marked as an audit-only denial")
	}

	if outcomes[0].Decision.Allowed {
		t.Error("outcome should record the would-be denial")
	}

	if outcomes[0].Decision.Reason != ReasonNoMatchingPolicy {
		t.Errorf("reason = %q, want %q", outcomes[0].Decision.Reason, ReasonNoMatchingPolicy)
	}
}

// TestEnforceModeReportsOneAuditFreeDenial proves the default posture reports the
// same outcome shape, without the audit flag, and does refuse.
func TestEnforceModeReportsOneAuditFreeDenial(t *testing.T) {
	t.Parallel()

	var outcomes []DispatchOutcome

	gate, err := NewDispatchGate(&gateCheck{allow: false}, OnDecision(func(o DispatchOutcome) {
		outcomes = append(outcomes, o)
	}))
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	disp := command.NewDispatcher()
	disp.Use(gate.CommandMiddleware())

	if err := command.RegisterTyped(disp, "CreateItem", func(context.Context, *gatedCmd) error {
		return nil
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := disp.Dispatch(authedCtx(), &gatedCmd{}); err == nil {
		t.Fatal("enforce mode allowed the dispatch")
	}

	if len(outcomes) != 1 {
		t.Fatalf("observer called %d times, want 1", len(outcomes))
	}

	if outcomes[0].Audit {
		t.Error("enforce mode must not mark the outcome as audit-only")
	}
}

// TestNewDispatchGateRejectsNilChecker proves the gate refuses to exist without
// an engine: a nil checker would silently allow everything.
func TestNewDispatchGateRejectsNilChecker(t *testing.T) {
	t.Parallel()

	if _, err := NewDispatchGate(nil); err == nil {
		t.Fatal("NewDispatchGate accepted a nil checker")
	}
}
