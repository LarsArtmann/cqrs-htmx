package identitymodel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/casbin/casbin/v3"
	casbinmodel "github.com/casbin/casbin/v3/model"
)

// The two clauses below are the only difference between the shipped object
// matcher and the one this workstream proposes. Both helpers are deliberately
// no-ops in one direction, so the proof survives the model change: before the
// change, upgradeModel is the real edit; after it, downgradeModel is.
const (
	objectMatcherExact    = "r.obj == p.obj"
	objectMatcherKeyMatch = "keyMatch(r.obj, p.obj)"
)

func upgradeObjectMatcher(modelStr string) string {
	return strings.ReplaceAll(modelStr, objectMatcherExact, objectMatcherKeyMatch)
}

func downgradeObjectMatcher(modelStr string) string {
	return strings.ReplaceAll(modelStr, objectMatcherKeyMatch, objectMatcherExact)
}

// threeFieldModel is a consumer-shaped model without a domain column, the arity
// enforceRequest falls back to when Casbin rejects a four-argument call. It is
// written out rather than derived because it is a different model shape, not a
// variant of ours.
const threeFieldModel = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = (p.sub == "*" || g(r.sub, p.sub)) && (p.obj == "*" || %s) && (p.act == "*" || r.act == p.act)`

// matcherCase is one object-matching row: a policy pattern, a request object,
// and what each matcher must answer for it.
type matcherCase struct {
	name     string
	policy   string
	request  string
	wantFlat bool
	wantKey  bool
}

// matcherCases is the capability table. wantFlat is what the exact-equality
// matcher answers today; wantKey is what keyMatch must answer. Every row where
// wantFlat is true must also be true under keyMatch — that is the superset
// property, asserted mechanically below rather than trusted.
func matcherCases() []matcherCase {
	return []matcherCase{
		{name: "exact_hit", policy: "command/CreateItem", request: "command/CreateItem", wantFlat: true, wantKey: true},
		{name: "exact_miss", policy: "command/CreateItem", request: "command/DeleteItem", wantFlat: false, wantKey: false},
		{name: "bare_star_matches_everything", policy: "*", request: "command/CreateItem", wantFlat: true, wantKey: true},
		{name: "command_prefix", policy: "command/*", request: "command/CreateItem", wantFlat: false, wantKey: true},
		{name: "command_prefix_second_name", policy: "command/*", request: "command/DeleteItem", wantFlat: false, wantKey: true},
		{name: "query_prefix", policy: "query/*", request: "query/ListItems", wantFlat: false, wantKey: true},
		{name: "prefix_is_not_a_string_prefix", policy: "command/CreateItem", request: "command/CreateItemX", wantFlat: false, wantKey: false},
		{name: "colon_stays_literal", policy: "command:CreateItem", request: "command:CreateItem", wantFlat: true, wantKey: true},
		{name: "colon_is_not_a_wildcard", policy: "command:CreateItem", request: "command:DeleteItem", wantFlat: false, wantKey: false},
		{name: "case_sensitive", policy: "command/*", request: "Command/CreateItem", wantFlat: false, wantKey: false},
		{name: "trailing_slash_matches_prefix", policy: "command/*", request: "command/", wantFlat: false, wantKey: true},
	}
}

// newMatcherAuthz builds an Authz over modelStr with one role policy holding
// objectPattern, plus the grouping edge that makes "alice" a member of the role.
func newMatcherAuthz(t *testing.T, modelStr, objectPattern string) *Authz {
	t.Helper()

	m, err := casbinmodel.NewModelFromString(modelStr)
	if err != nil {
		t.Fatalf("parse model: %v", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}

	policy := Policy{
		Subject: RoleUser,
		Domain:  WildcardDomain,
		Object:  objectPattern,
		Action:  ActionExecute,
		Effect:  EffectAllow,
	}
	if _, err := e.AddPolicy(policyArgs(policy)...); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	hasDomainRole := strings.Contains(modelStr, "g = _, _, _")
	if hasDomainRole {
		_, err = e.AddGroupingPolicy("alice", string(RoleUser), WildcardDomain)
	} else {
		_, err = e.AddGroupingPolicy("alice", string(RoleUser))
	}

	if err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	return &Authz{enforcer: e}
}

// TestKeyMatchIsASupersetOfTheExactObjectMatcher proves the single fact the
// whole policy-row scheme rests on: swapping the object comparison for
// keyMatch never withdraws a permission the shipped matcher grants, and adds
// prefix matching. If this test cannot pass, the scheme collapses to one policy
// row per command per role and must be re-planned.
func TestKeyMatchIsASupersetOfTheExactObjectMatcher(t *testing.T) {
	t.Parallel()

	flatModel := downgradeObjectMatcher(DefaultRBACModel)
	keyModel := upgradeObjectMatcher(DefaultRBACModel)

	for _, tc := range matcherCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flat := newMatcherAuthz(t, flatModel, tc.policy)
			key := newMatcherAuthz(t, keyModel, tc.policy)

			gotFlat, _, err := flat.Check("alice", WildcardDomain, tc.request, string(ActionExecute))
			if err != nil {
				t.Fatalf("exact matcher check: %v", err)
			}

			gotKey, _, err := key.Check("alice", WildcardDomain, tc.request, string(ActionExecute))
			if err != nil {
				t.Fatalf("keyMatch matcher check: %v", err)
			}

			if gotFlat != tc.wantFlat {
				t.Errorf("exact matcher (%q vs %q) = %v, want %v", tc.request, tc.policy, gotFlat, tc.wantFlat)
			}

			if gotKey != tc.wantKey {
				t.Errorf("keyMatch matcher (%q vs %q) = %v, want %v", tc.request, tc.policy, gotKey, tc.wantKey)
			}

			if gotFlat && !gotKey {
				t.Errorf("keyMatch WITHDRAWS a permission the exact matcher grants (%q vs %q): scheme is not a superset",
					tc.request, tc.policy)
			}
		})
	}
}

// TestKeyMatchOnThreeFieldModel repeats the object-matching proof on the
// domain-less model shape enforceRequest falls back to, so the fleet's mixed
// arity cannot surprise the matcher change.
func TestKeyMatchOnThreeFieldModel(t *testing.T) {
	t.Parallel()

	flatModel := fmt.Sprintf(threeFieldModel, objectMatcherExact)
	keyModel := fmt.Sprintf(threeFieldModel, objectMatcherKeyMatch)

	for _, tc := range matcherCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flat := newMatcher3FieldEnforcer(t, flatModel, tc.policy)
			key := newMatcher3FieldEnforcer(t, keyModel, tc.policy)

			// Three arguments is the shape enforceRequest falls back to after
			// Casbin rejects the four-argument request as an arity mismatch.
			gotFlat, err := flat.Enforce("alice", tc.request, string(ActionExecute))
			if err != nil {
				t.Fatalf("exact matcher check: %v", err)
			}

			gotKey, err := key.Enforce("alice", tc.request, string(ActionExecute))
			if err != nil {
				t.Fatalf("keyMatch matcher check: %v", err)
			}

			if gotKey != tc.wantKey {
				t.Errorf("keyMatch matcher (%q vs %q) = %v, want %v", tc.request, tc.policy, gotKey, tc.wantKey)
			}

			if gotFlat && !gotKey {
				t.Errorf("keyMatch WITHDRAWS a permission on the 3-field model (%q vs %q)", tc.request, tc.policy)
			}
		})
	}
}

// newMatcher3FieldEnforcer builds a bare enforcer over a model whose request
// definition has no domain column. It is deliberately not an Authz: Authz.Check
// calls the four-argument shape unconditionally, so a domain-less model can only
// be exercised through the three-argument call root's enforceRequest falls back
// to.
func newMatcher3FieldEnforcer(t *testing.T, modelStr, objectPattern string) *casbin.Enforcer {
	t.Helper()

	m, err := casbinmodel.NewModelFromString(modelStr)
	if err != nil {
		t.Fatalf("parse 3-field model: %v", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}

	if _, err := e.AddPolicy(string(RoleUser), objectPattern, string(ActionExecute), string(EffectAllow)); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	if _, err := e.AddGroupingPolicy("alice", string(RoleUser)); err != nil {
		t.Fatalf("add grouping policy: %v", err)
	}

	return e
}
