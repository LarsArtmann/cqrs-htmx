package identitymodel

import (
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// S38 type-safety tails: ULID ordering (a), strict branded-ID parsing (b),
// Exists/IsActive split (c), g2 guard under custom models (d), permanent
// NotConfigured families (e), and accessor copy semantics (f).

// TestGenerateUserID_PreservesULIDOrdering pins S38a: GenerateUserID must
// route fresh ULIDs through brandid.NewID, never SyntheticUserID — the
// SHA-256 derivation destroys the timestamp and with it ULID's core
// ordering guarantee. Two IDs minted >=2ms apart must compare strictly
// increasing (same-millisecond entropy is random; cross-millisecond is not).
func TestGenerateUserID_PreservesULIDOrdering(t *testing.T) {
	first := GenerateUserID()
	time.Sleep(5 * time.Millisecond)
	second := GenerateUserID()

	if first.Get().Compare(second.Get()) >= 0 {
		t.Fatalf("GenerateUserID ordering broken: %s !< %s (timestamp destroyed?)", first.Get(), second.Get())
	}

	if first.Get().Time() == 0 {
		t.Fatal("GenerateUserID produced a zero-timestamp ULID — derivation is hashing the input again")
	}

	// Freshness: two calls must never collide.
	if first.Get() == second.Get() {
		t.Fatal("GenerateUserID returned the same ULID twice")
	}
}

// TestParseTenantID_Vectors pins S38b: the strict reconstruction path accepts
// bare and display-prefixed values, and rejects the ambiguous forms that
// would otherwise silently become distinct identities.
func TestParseTenantID_Vectors(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "bare value", in: "acme", want: "acme"},
		{name: "display prefix stripped", in: "Tenant:acme", want: "acme"},
		{name: "empty rejected", in: "", wantErr: true},
		{name: "prefix with empty value rejected", in: "Tenant:", wantErr: true},
		{name: "leading whitespace rejected", in: " acme", wantErr: true},
		{name: "trailing whitespace rejected", in: "acme ", wantErr: true},
		{name: "padded value inside prefix rejected", in: "Tenant: acme", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTenantID(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseTenantID(%q) = %q, want error", tc.in, got.Get())
				}

				fam := errorfamily.Classify(err)
				if fam != errorfamily.Rejection {
					t.Errorf("ParseTenantID(%q) family = %v, want Rejection", tc.in, fam)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseTenantID(%q): %v", tc.in, err)
			}

			if got.Get() != tc.want {
				t.Errorf("ParseTenantID(%q) = %q, want %q", tc.in, got.Get(), tc.want)
			}
		})
	}
}

// TestParseBotID_Vectors is the BotID twin of the TenantID vectors.
func TestParseBotID_Vectors(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "bare value", in: "bot-1", want: "bot-1"},
		{name: "display prefix stripped", in: "Bot:bot-1", want: "bot-1"},
		{name: "empty rejected", in: "", wantErr: true},
		{name: "prefix with empty value rejected", in: "Bot:", wantErr: true},
		{name: "leading whitespace rejected", in: " bot-1", wantErr: true},
		{name: "trailing whitespace rejected", in: "bot-1 ", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseBotID(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseBotID(%q) = %q, want error", tc.in, got.Get())
				}

				fam := errorfamily.Classify(err)
				if fam != errorfamily.Rejection {
					t.Errorf("ParseBotID(%q) family = %v, want Rejection", tc.in, fam)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseBotID(%q): %v", tc.in, err)
			}

			if got.Get() != tc.want {
				t.Errorf("ParseBotID(%q) = %q, want %q", tc.in, got.Get(), tc.want)
			}
		})
	}
}

func memberAddedEvent(t *testing.T, tenantID string) event.Event {
	t.Helper()

	payload, err := MarshalPayload(MemberAddedPayload{
		SchemaVersion: CurrentSchemaVersion,
		ActorKind:     ActorKindUserStr,
		ActorID:       GenerateUserID().Get().String(),
		TenantID:      tenantID,
		Roles:         []Role{RoleAdmin},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	actor := ActorIDFromUser(GenerateUserID())
	evt, evtErr := event.NewEvent(
		EventMemberAdded,
		DeriveMembershipID(actor, NewTenantID("agg")),
		AggregateTypeMembership,
		1,
		payload,
	)
	if evtErr != nil {
		t.Fatalf("event.NewEvent: %v", evtErr)
	}

	return evt
}

// TestFoldMembership_RejectsAmbiguousTenantID pins the S38b fold wiring: the
// fold reconstructs TenantID through ParseTenantID, so a journaled event with
// an ambiguous tenant reference is rejected at replay time instead of
// silently becoming a distinct identity.
func TestFoldMembership_RejectsAmbiguousTenantID(t *testing.T) {
	for _, tenantID := range []string{"", "  ", " padded "} {
		if _, err := FoldMembership(MembershipState{}, memberAddedEvent(t, tenantID)); err == nil {
			t.Errorf("FoldMembership with TenantID %q: want error, got nil", tenantID)
		}
	}
}

func TestFoldMembership_AcceptsPrefixedTenantID(t *testing.T) {
	state, err := FoldMembership(MembershipState{}, memberAddedEvent(t, "Tenant:acme"))
	if err != nil {
		t.Fatalf("FoldMembership: %v", err)
	}

	if state.TenantID.Get() != "acme" {
		t.Errorf("TenantID = %q, want %q (display prefix must be stripped)", state.TenantID.Get(), "acme")
	}
}

// livenessState is the shared Exists/IsActive surface every aggregate state
// exposes (S38c); the helper below keeps the per-aggregate assertions
// duplicate-free.
type livenessState interface {
	Exists() bool
	IsActive() bool
}

func assertLiveness(t *testing.T, name string, state livenessState, wantExists, wantIsActive bool) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		if got := state.Exists(); got != wantExists {
			t.Errorf("Exists() = %v, want %v", got, wantExists)
		}

		if got := state.IsActive(); got != wantIsActive {
			t.Errorf("IsActive() = %v, want %v", got, wantIsActive)
		}
	})
}

// TestUserState_IsActive pins the S38c additive split: Exists stays true for
// deleted users (they have history), IsActive is the write-path liveness gate.
func TestUserState_IsActive(t *testing.T) {
	assertLiveness(t, "zero state", UserState{}, false, false)
	assertLiveness(t, "registered", UserState{Email: "a@b.c"}, true, true)
	assertLiveness(t, "deleted user still exists", UserState{Email: "a@b.c", Deleted: true}, true, false)
}

// TestBotState_IsActive pins the S38c uniform vocabulary: BotState.Exists
// already folds the Deleted tombstone, so IsActive mirrors it exactly.
func TestBotState_IsActive(t *testing.T) {
	assertLiveness(t, "zero state", BotState{}, false, false)
	assertLiveness(t, "registered", BotState{Name: "deploy-bot"}, true, true)
	assertLiveness(t, "deleted folds into Exists", BotState{Name: "deploy-bot", Deleted: true}, false, false)
}

// TestTenantState_IsActive pins the pre-existing tenant split (S38c context):
// suspended tenants exist but are not active.
func TestTenantState_IsActive(t *testing.T) {
	suspended := TenantState{Name: "acme", Suspended: true}
	if !suspended.Exists() {
		t.Fatal("suspended tenant should still exist")
	}

	if suspended.IsActive() {
		t.Fatal("suspended tenant must not be active")
	}
}

// TestNewAuthz_CustomModelWithoutG2 pins S38d: a consumer-supplied model
// without a g2 role-definition section must construct cleanly — the default
// role hierarchy is skipped instead of being force-fed into a section the
// model never declared (AddNamedGroupingPolicy errors on a missing section).
func TestNewAuthz_CustomModelWithoutG2(t *testing.T) {
	noG2Model := strings.Replace(DefaultRBACModel, "g2 = _, _\n", "", 1)
	if strings.Contains(noG2Model, "g2 = ") {
		t.Fatal("test setup: g2 definition line not removed")
	}

	noG2Model = strings.Replace(noG2Model, " || g2(r.sub, p.sub)", "", 1)
	if strings.Contains(noG2Model, "g2(") {
		t.Fatal("test setup: g2 matcher reference not removed")
	}

	a, err := NewAuthz(EnforcerConfig{ModelString: noG2Model, Policies: DefaultPolicies()})
	if err != nil {
		t.Fatalf("NewAuthz with g2-less model: %v (g2 seeding not guarded?)", err)
	}

	ok, enforceErr := a.Enforce(string(RoleSuperAdmin), "*", "users", ActionRead)
	if enforceErr != nil {
		t.Fatalf("Enforce under g2-less model: %v", enforceErr)
	}

	if !ok {
		t.Fatal("super-admin wildcard policy must still allow reads under a g2-less model")
	}
}

// TestNotConfiguredErrors_ArePermanent pins S38e: a missing deployment
// configuration is permanent for this deployment — Rejection (400-class),
// never Transient (503), and never retryable.
func TestNotConfiguredErrors_ArePermanent(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{name: "ErrEmailVerificationNotConfigured", err: ErrEmailVerificationNotConfigured},
		{name: "ErrTOTPNotConfigured", err: ErrTOTPNotConfigured},
		{name: "ErrOAuthNotConfigured", err: ErrOAuthNotConfigured},
	}

	for _, s := range sentinels {
		t.Run(s.name, func(t *testing.T) {
			if fam := errorfamily.Classify(s.err); fam != errorfamily.Rejection {
				t.Errorf("family = %v, want Rejection", fam)
			}

			if errorfamily.IsRetryable(s.err) {
				t.Error("NotConfigured errors must not be retryable")
			}
		})
	}
}

// TestCommandAccessors_ReturnCopies pins S38f: mutating a slice returned by
// an accessor must never write through to the command — commands are
// immutable once constructed.
func TestCommandAccessors_ReturnCopies(t *testing.T) {
	t.Run("RegisterUserCmd.Roles", func(t *testing.T) {
		cmd := NewRegisterUserCmd(id.NewStreamID(), "a@b.c", "A", []Role{RoleAdmin, RoleViewer})
		roles := cmd.Roles()
		roles[0] = RoleSuperAdmin
		if cmd.Roles()[0] == RoleSuperAdmin {
			t.Fatal("mutating Roles() wrote through to the command")
		}
	})

	t.Run("RemoveCredentialCmd.CredentialID", func(t *testing.T) {
		cmd := NewRemoveCredentialCmd(id.NewStreamID(), []byte{0x01, 0x02})
		credID := cmd.CredentialID()
		credID[0] = 0xFF
		if cmd.CredentialID()[0] == 0xFF {
			t.Fatal("mutating CredentialID() wrote through to the command")
		}
	})

	t.Run("EnableTOTPCmd.Secret", func(t *testing.T) {
		cmd := NewEnableTOTPCmd(id.NewStreamID(), []byte{0xAA, 0xBB})
		secret := cmd.Secret()
		secret[0] = 0x00
		if cmd.Secret()[0] == 0x00 {
			t.Fatal("mutating Secret() wrote through to the command")
		}
	})

	t.Run("AddMemberCmd.Roles", func(t *testing.T) {
		actor := ActorIDFromUser(GenerateUserID())
		cmd := NewAddMemberCmd(actor, NewTenantID("acme"), []Role{RoleViewer})
		roles := cmd.Roles()
		roles[0] = RoleSuperAdmin
		if cmd.Roles()[0] == RoleSuperAdmin {
			t.Fatal("mutating Roles() wrote through to the command")
		}
	})

	t.Run("UpdateMemberRolesCmd.Roles", func(t *testing.T) {
		actor := ActorIDFromUser(GenerateUserID())
		cmd := NewUpdateMemberRolesCmd(actor, NewTenantID("acme"), []Role{RoleViewer})
		roles := cmd.Roles()
		roles[0] = RoleSuperAdmin
		if cmd.Roles()[0] == RoleSuperAdmin {
			t.Fatal("mutating Roles() wrote through to the command")
		}
	})

	t.Run("RegisterBotCmd.TokenHash", func(t *testing.T) {
		cmd := NewRegisterBotCmd(id.NewStreamID(), "bot", GenerateUserID(), []byte{0x01}, []string{"read"})
		hash := cmd.TokenHash()
		hash[0] = 0xFF
		if cmd.TokenHash()[0] == 0xFF {
			t.Fatal("mutating TokenHash() wrote through to the command")
		}
	})

	t.Run("RegisterBotCmd.Scopes", func(t *testing.T) {
		cmd := NewRegisterBotCmd(id.NewStreamID(), "bot", GenerateUserID(), []byte{0x01}, []string{"read"})
		scopes := cmd.Scopes()
		scopes[0] = "admin"
		if cmd.Scopes()[0] == "admin" {
			t.Fatal("mutating Scopes() wrote through to the command")
		}
	})
}
