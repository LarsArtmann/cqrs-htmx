package authztest

import (
	"context"
	"fmt"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Governor is the durable write side of an authorization system: the path that
// gives a subject roles inside a tenant, and takes them away again.
//
// It is string-shaped so this package never imports a specific identity model.
// The adapter for this repository's own stack is four lines:
//
//	type governor struct{ svc *usermgmt.Service }
//
//	func (g governor) GrantRoles(ctx context.Context, subject, tenant string, roles []string) error {
//	    return g.svc.GrantRoles(ctx, usermgmt.ActorID(subject), usermgmt.TenantID(tenant), roles)
//	}
type Governor interface {
	// GrantRoles gives subject exactly roles inside tenant. It must converge:
	// the suite grants before every case, so the same call arrives more than
	// once and a case has to be runnable on its own.
	GrantRoles(ctx context.Context, subject, tenant string, roles []string) error

	// RevokeRoles removes every role subject holds inside tenant.
	RevokeRoles(ctx context.Context, subject, tenant string) error
}

// GrantSuite proves the three properties a posture matrix cannot see: a grant is
// scoped to the tenant it was made in, revoking it removes authority, and it
// outlives the process that made it.
//
// [Suite] answers "does this route refuse the wrong caller?". This answers "does
// the authority itself behave?" — which is where the 2026-11-02 consumer audit
// found the expensive defects: a grant that evaporated on every restart, and a
// grant that only worked in whichever domain the check happened to ask about.
//
// Point it at a service whose projections have drained. Under AsyncStartup,
// poll readiness first, or every case reads policy state that has not been
// replayed yet.
type GrantSuite struct {
	// Checker answers the authorization questions.
	Checker cqrshtmx.PermissionChecker

	// Governor is the durable write side.
	Governor Governor

	// Subject is the actor every case grants to.
	Subject string

	// Tenant is the tenant the grant lands in.
	Tenant string

	// ForeignTenant is a DIFFERENT tenant the grant must not reach. Required:
	// with only one domain the isolation case has nothing to compare against
	// and would pass by proving nothing.
	ForeignTenant string

	// Roles is what the cases grant.
	Roles []string

	// ResourceType is the object type the granted roles must cover.
	ResourceType string

	// ResourceID is the specific instance, when the question is about one.
	// Empty asks about the collection, which is what most RBAC grants answer.
	ResourceID string

	// Action is the verb the granted roles must permit.
	Action string

	// Fresh builds a checker from durable state alone — a second process, in
	// effect. Required by [GrantSuite.ProveDurability] only.
	Fresh func(t *testing.T) cqrshtmx.PermissionChecker
}

// Verify checks the wiring every case depends on, so a half-filled suite fails
// loudly instead of proving nothing.
func (s GrantSuite) Verify() error {
	if err := s.verifyWiring(); err != nil {
		return err
	}

	if s.ForeignTenant == s.Tenant {
		return errorfamily.NewRejection(
			"authztest.foreign_tenant_not_foreign",
			fmt.Sprintf("GrantSuite.ForeignTenant equals Tenant (%q), so tenant isolation cannot be proven", s.Tenant),
		)
	}

	return nil
}

// verifyWiring checks the fields all three cases need, in the order a reader
// fills them in.
func (s GrantSuite) verifyWiring() error {
	required := []struct {
		name  string
		empty bool
		code  string
	}{
		{"Checker", s.Checker == nil, "authztest.checker_nil"},
		{"Governor", s.Governor == nil, "authztest.governor_nil"},
		{"Subject", s.Subject == "", "authztest.subject_empty"},
		{"Tenant", s.Tenant == "", "authztest.tenant_empty"},
		{"ForeignTenant", s.ForeignTenant == "", "authztest.foreign_tenant_empty"},
		{"ResourceType", s.ResourceType == "", "authztest.resource_empty"},
		{"Action", s.Action == "", "authztest.action_empty"},
	}

	for _, field := range required {
		if field.empty {
			return errorfamily.NewRejection(field.code, "GrantSuite."+field.name+" is required")
		}
	}

	if len(s.Roles) == 0 {
		return errorfamily.NewRejection(
			"authztest.roles_empty",
			"GrantSuite.Roles is empty — declare the roles these cases grant",
		)
	}

	return nil
}

// ProveTenantIsolation proves the grant is scoped to its tenant: it authorizes
// inside Tenant and does not authorize inside ForeignTenant.
func (s GrantSuite) ProveTenantIsolation(t *testing.T) {
	t.Helper()
	failOn(t, s.checkTenantIsolation(t.Context()))
}

// ProveRevocation proves RevokeRoles removes authority that was really there:
// the grant is observed to authorize first, so a deny afterwards cannot be an
// accident of the grant never having worked.
func (s GrantSuite) ProveRevocation(t *testing.T) {
	t.Helper()
	failOn(t, s.checkRevocation(t.Context()))
}

// ProveDurability proves the grant survives a fresh checker built from durable
// state alone — a restart, a projection rebuild, or a second instance.
func (s GrantSuite) ProveDurability(t *testing.T) {
	t.Helper()

	if s.Fresh == nil {
		t.Fatal("ProveDurability needs GrantSuite.Fresh: build a checker from durable state alone (a second process)")
	}

	failOn(t, s.checkDurability(t.Context(), func() cqrshtmx.PermissionChecker { return s.Fresh(t) }))
}

// checkTenantIsolation is ProveTenantIsolation with the verdict as a value, so
// the failure paths are testable without a failing *testing.T.
func (s GrantSuite) checkTenantIsolation(ctx context.Context) error {
	if err := s.Verify(); err != nil {
		return err
	}

	if err := s.grant(ctx); err != nil {
		return err
	}

	if err := s.requireAllowed(s.Checker, s.Tenant); err != nil {
		return err
	}

	decision, err := s.decide(s.Checker, s.ForeignTenant)
	if err != nil {
		return err
	}

	if decision.Allowed {
		return errorfamily.NewRejection(
			"authztest.grant_leaks_across_tenants",
			fmt.Sprintf(
				"a grant in tenant %q also authorized in tenant %q — the grant is not tenant-scoped",
				s.Tenant, s.ForeignTenant,
			),
		)
	}

	return nil
}

// checkRevocation is ProveRevocation with the verdict as a value.
func (s GrantSuite) checkRevocation(ctx context.Context) error {
	if err := s.Verify(); err != nil {
		return err
	}

	if err := s.grant(ctx); err != nil {
		return err
	}

	if err := s.requireAllowed(s.Checker, s.Tenant); err != nil {
		return err
	}

	if err := s.Governor.RevokeRoles(ctx, s.Subject, s.Tenant); err != nil {
		return errorfamily.WrapTransient(
			err, "authztest.revoke.failed",
			fmt.Sprintf("revoke roles of %q in tenant %q", s.Subject, s.Tenant),
		)
	}

	decision, err := s.decide(s.Checker, s.Tenant)
	if err != nil {
		return err
	}

	if decision.Allowed {
		return errorfamily.NewRejection(
			"authztest.revoke_ineffective",
			fmt.Sprintf(
				"the roles %v still authorize after RevokeRoles — the revoke did not reach the authoritative state",
				s.Roles,
			),
		)
	}

	return nil
}

// checkDurability is ProveDurability with the verdict as a value.
//
// fresh is a builder, not a checker: the second process has to be consulted
// AFTER this case grants, otherwise it reconstructs the state that existed
// before the grant and the case proves the opposite of what it claims.
func (s GrantSuite) checkDurability(
	ctx context.Context, fresh func() cqrshtmx.PermissionChecker,
) error {
	if err := s.Verify(); err != nil {
		return err
	}

	if fresh == nil {
		return errorfamily.NewRejection(
			"authztest.checker_nil",
			"GrantSuite.Fresh is required: build a checker from durable state alone",
		)
	}

	if err := s.grant(ctx); err != nil {
		return err
	}

	return s.requireAllowed(fresh(), s.Tenant)
}

// grant performs the suite's grant, so every case starts from the same state
// and can be run on its own.
func (s GrantSuite) grant(ctx context.Context) error {
	if err := s.Governor.GrantRoles(ctx, s.Subject, s.Tenant, s.Roles); err != nil {
		return errorfamily.WrapTransient(
			err, "authztest.grant.failed",
			fmt.Sprintf("grant roles %v to %q in tenant %q", s.Roles, s.Subject, s.Tenant),
		)
	}

	return nil
}

// requireAllowed asserts the question is permitted, naming the reason when it is
// not: "your grant is missing" and "you were explicitly refused" need different
// fixes, and an error here must say which one happened.
func (s GrantSuite) requireAllowed(checker cqrshtmx.PermissionChecker, domain string) error {
	if checker == nil {
		return errorfamily.NewRejection(
			"authztest.checker_nil",
			fmt.Sprintf("the checker for domain %q is nil", domain),
		)
	}

	decision, err := s.decide(checker, domain)
	if err != nil {
		return err
	}

	if !decision.Allowed {
		return errorfamily.NewRejection(
			"authztest.grant_not_effective",
			fmt.Sprintf(
				"the roles %v of %q do not authorize %s on %q in tenant %q: reason=%s",
				s.Roles, s.Subject, s.Action, s.resourceLabel(), domain, decision.Reason,
			),
		)
	}

	return nil
}

// resourceLabel renders the declared resource for a failure message.
func (s GrantSuite) resourceLabel() string {
	return cqrshtmx.Resource{Type: s.ResourceType, ID: s.ResourceID}.String()
}

// decide asks the question the suite declares, in domain.
func (s GrantSuite) decide(
	checker cqrshtmx.PermissionChecker, domain string,
) (cqrshtmx.Decision, error) {
	question := cqrshtmx.Request{
		Subject:  s.Subject,
		Domain:   domain,
		Resource: cqrshtmx.Resource{Type: s.ResourceType, ID: s.ResourceID},
		Action:   s.Action,
	}

	decision, err := cqrshtmx.Decide(checker, question)
	if err != nil {
		return cqrshtmx.Decision{}, errorfamily.WrapTransient(
			err, "authztest.decide.failed",
			fmt.Sprintf("decide %s on %q in tenant %q", s.Action, question.Resource.String(), domain),
		)
	}

	return decision, nil
}

// failOn turns a verdict into a test failure.
func failOn(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}
