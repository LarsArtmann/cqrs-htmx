package usermgmt

import (
	"slices"
	"strconv"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"pgregory.net/rapid"
)

// Property-based tests for event SEQUENCE folds of the Tenant, Membership, and
// Bot aggregates. Mirrors property_sequences_test.go (User) but exercises each
// aggregate's terminal/toggle semantics and — for Tenant — the documented
// struct-level IsValid() invariant that must hold for ANY valid event stream.
//
// Helpers mustPropTenantEvent / mustPropBotEvent / mustPropMembershipEvent and
// the prop*AggID fixtures are defined in property_extras_test.go and reused.

// --- Tenant sequence properties ---

// genTenantEventStream generates a valid tenant stream: TenantCreated first,
// then 0-5 suspend/reactivate toggles, optionally terminated by TenantDeleted.
func genTenantEventStream() *rapid.Generator[[]event.Event] {
	return rapid.Custom(func(t *rapid.T) []event.Event {
		const maxToggles = 5

		name := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "name")
		display := rapid.StringMatching(`[A-Z][a-z]{3,10}`).Draw(t, "display")

		events := []event.Event{
			mustPropTenantEvent(eventTenantCreated, 1, TenantCreatedPayload{
				SchemaVersion: currentSchemaVersion, Name: name, DisplayName: display,
			}),
		}

		toggles := rapid.IntRange(0, maxToggles).Draw(t, "toggles")

		for i := range toggles {
			reason := rapid.StringMatching(`[a-z ]{5,20}`).Draw(t, "reason-"+strconv.Itoa(i))
			kind := rapid.IntRange(0, 2).Draw(t, "kind-"+strconv.Itoa(i))

			switch kind {
			case 0: // suspend
				events = append(events, mustPropTenantEvent(eventTenantSuspended,
					versionOf(len(events)+1), TenantSuspendedPayload{
						SchemaVersion: currentSchemaVersion, Reason: reason,
					}))
			case 1: // reactivate
				events = append(events, mustPropTenantEvent(eventTenantReactivated,
					versionOf(len(events)+1), TenantReactivatedPayload{
						SchemaVersion: currentSchemaVersion,
					}))
			default: // terminal delete
				events = append(events, mustPropTenantEvent(eventTenantDeleted,
					versionOf(len(events)+1), TenantDeletedPayload{
						SchemaVersion: currentSchemaVersion, Reason: reason,
					}))

				return events
			}
		}

		return events
	})
}

// foldAllTenant folds a tenant event stream left-to-right from the zero state.
func foldAllTenant(events []event.Event) (TenantState, error) {
	state := TenantState{}

	var err error

	for _, evt := range events {
		state, err = foldTenant(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

// TestFoldTenantSequence_IsValidAlwaysHolds asserts the documented invariant:
// "struct-level invariants that cannot be violated by any sequence of events".
// This is the highest-value property test in the file — if any fold branch
// ever leaves the struct inconsistent, rapid shrinks to the minimal stream.
func TestFoldTenantSequence_IsValidAlwaysHolds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genTenantEventStream().Draw(t, "stream")

		state, err := foldAllTenant(events)
		if err != nil {
			t.Fatalf("foldAllTenant: %v", err)
		}

		if !state.IsValid() {
			t.Errorf("IsValid invariant violated for stream %v: %+v", eventTypes(events), state)
		}
	})
}

// TestFoldTenantSequence_Associativity: fold(whole) == fold(suffix, fold(prefix)).
func TestFoldTenantSequence_Associativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genTenantEventStream().Draw(t, "stream")
		split := rapid.IntRange(0, len(events)).Draw(t, "split")

		whole, err := foldAllTenant(events)
		if err != nil {
			t.Fatalf("whole: %v", err)
		}

		prefix, err := foldAllTenant(events[:split])
		if err != nil {
			t.Fatalf("prefix: %v", err)
		}

		chunked, err := foldTenantSlice(prefix, events[split:])
		if err != nil {
			t.Fatalf("chunked: %v", err)
		}

		if whole != chunked {
			t.Errorf("associativity broken at split=%d: whole=%+v chunked=%+v", split, whole, chunked)
		}
	})
}

// TestFoldTenantSequence_DeletedClearsSuspend: a TenantDeleted event always
// clears both Suspended and SuspendReason, regardless of prior state.
func TestFoldTenantSequence_DeletedClearsSuspend(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genTenantEventStream().Draw(t, "stream")

		state, err := foldAllTenant(events)
		if err != nil {
			t.Fatalf("foldAllTenant: %v", err)
		}

		if state.Deleted && (state.Suspended || state.SuspendReason != "") {
			t.Errorf("deleted tenant must not be suspended: %+v", state)
		}
	})
}

// --- Membership sequence properties ---

// genMembershipEventStream: MemberAdded, then 0-4 MemberRolesChanged, optionally
// MemberRemoved (terminal).
func genMembershipEventStream() *rapid.Generator[[]event.Event] {
	return rapid.Custom(func(t *rapid.T) []event.Event {
		const maxChanges = 4

		roles := rapid.SliceOfN(rapidRole(), 1, 3).Draw(t, "roles")

		events := []event.Event{
			mustPropMembershipEvent(eventMemberAdded, 1, MemberAddedPayload{
				SchemaVersion: currentSchemaVersion,
				ActorKind:     actorKindUserStr,
				ActorID:       "actor-1",
				TenantID:      "tenant-1",
				Roles:         roles,
			}),
		}

		changes := rapid.IntRange(0, maxChanges).Draw(t, "changes")
		terminal := rapid.Bool().Draw(t, "terminal")

		for i := range changes {
			newRoles := rapid.SliceOfN(rapidRole(), 1, 3).Draw(t, "roles-"+strconv.Itoa(i))

			events = append(events, mustPropMembershipEvent(eventMemberRolesChanged,
				versionOf(len(events)+1), MemberRolesChangedPayload{
					SchemaVersion: currentSchemaVersion,
					ActorKind:     actorKindUserStr,
					ActorID:       "actor-1",
					TenantID:      "tenant-1",
					Roles:         newRoles,
				}))
		}

		if terminal {
			events = append(events, mustPropMembershipEvent(eventMemberRemoved,
				versionOf(len(events)+1), MemberRemovedPayload{
					SchemaVersion: currentSchemaVersion,
					ActorID:       "actor-1",
					TenantID:      "tenant-1",
				}))
		}

		return events
	})
}

// foldAllMembership folds a membership event stream from the zero state.
func foldAllMembership(events []event.Event) (MembershipState, error) {
	state := MembershipState{}

	var err error

	for _, evt := range events {
		state, err = foldMembership(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

// TestFoldMembershipSequence_Associativity: chunked fold equals whole fold.
func TestFoldMembershipSequence_Associativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genMembershipEventStream().Draw(t, "stream")
		split := rapid.IntRange(0, len(events)).Draw(t, "split")

		whole, err := foldAllMembership(events)
		if err != nil {
			t.Fatalf("whole: %v", err)
		}

		prefix, err := foldAllMembership(events[:split])
		if err != nil {
			t.Fatalf("prefix: %v", err)
		}

		chunked, err := foldMembershipSlice(prefix, events[split:])
		if err != nil {
			t.Fatalf("chunked: %v", err)
		}

		if !membershipStateEqual(whole, chunked) {
			t.Errorf("associativity broken at split=%d: whole=%+v chunked=%+v", split, whole, chunked)
		}
	})
}

// TestFoldMembershipSequence_RemovedTerminal: if MemberRemoved is present, the
// final state has Removed==true and no roles.
func TestFoldMembershipSequence_RemovedTerminal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genMembershipEventStream().Draw(t, "stream")

		state, err := foldAllMembership(events)
		if err != nil {
			t.Fatalf("foldAllMembership: %v", err)
		}

		removed := slices.IndexFunc(events,
			func(e event.Event) bool { return e.Type() == eventMemberRemoved }) >= 0
		if removed && (!state.Removed || len(state.Roles) != 0) {
			t.Errorf("MemberRemoved must leave Removed=true with no roles: %+v", state)
		}
	})
}

// --- Bot sequence properties ---

// genBotEventStream: BotRegistered, optionally BotDeleted (terminal).
func genBotEventStream() *rapid.Generator[[]event.Event] {
	return rapid.Custom(func(t *rapid.T) []event.Event {
		name := rapid.StringMatching(`[a-z]{3,10}-bot`).Draw(t, "name")
		scopes := rapid.SliceOfN(rapid.StringMatching(`[a-z]{3,8}`), 1, 3).Draw(t, "scopes")
		deleted := rapid.Bool().Draw(t, "deleted")

		events := []event.Event{
			mustPropBotEvent(eventBotRegistered, 1, BotRegisteredPayload{
				SchemaVersion: currentSchemaVersion,
				Name:          name,
				OwnerID:       NewUserID(id.NewStreamID().String()),
				TokenHash:     []byte{0x01, 0x02, 0x03},
				Scopes:        scopes,
			}),
		}

		if deleted {
			reason := rapid.StringMatching(`[a-z ]{5,15}`).Draw(t, "reason")

			events = append(events, mustPropBotEvent(eventBotDeleted,
				versionOf(len(events)+1), BotDeletedPayload{
					SchemaVersion: currentSchemaVersion, Reason: reason,
				}))
		}

		return events
	})
}

// foldAllBot folds a bot event stream from the zero state.
func foldAllBot(events []event.Event) (BotState, error) {
	state := BotState{}

	var err error

	for _, evt := range events {
		state, err = foldBot(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

// TestFoldBotSequence_DeletedTerminal: if BotDeleted is present, Deleted==true.
func TestFoldBotSequence_DeletedTerminal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genBotEventStream().Draw(t, "stream")

		state, err := foldAllBot(events)
		if err != nil {
			t.Fatalf("foldAllBot: %v", err)
		}

		hasDelete := slices.IndexFunc(events,
			func(e event.Event) bool { return e.Type() == eventBotDeleted }) >= 0
		if hasDelete && !state.Deleted {
			t.Error("BotDeleted must leave state.Deleted == true")
		}
	})
}

// TestFoldBotSequence_Associativity: chunked fold equals whole fold.
func TestFoldBotSequence_Associativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genBotEventStream().Draw(t, "stream")
		split := rapid.IntRange(0, len(events)).Draw(t, "split")

		whole, err := foldAllBot(events)
		if err != nil {
			t.Fatalf("whole: %v", err)
		}

		prefix, err := foldAllBot(events[:split])
		if err != nil {
			t.Fatalf("prefix: %v", err)
		}

		chunked, err := foldBotSlice(prefix, events[split:])
		if err != nil {
			t.Fatalf("chunked: %v", err)
		}

		if !botStateEqual(whole, chunked) {
			t.Errorf("associativity broken at split=%d: whole=%+v chunked=%+v", split, whole, chunked)
		}
	})
}

// --- shared helpers ---

func foldTenantSlice(initial TenantState, events []event.Event) (TenantState, error) {
	state := initial

	var err error

	for _, evt := range events {
		state, err = foldTenant(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

func foldMembershipSlice(initial MembershipState, events []event.Event) (MembershipState, error) {
	state := initial

	var err error

	for _, evt := range events {
		state, err = foldMembership(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

func foldBotSlice(initial BotState, events []event.Event) (BotState, error) {
	state := initial

	var err error

	for _, evt := range events {
		state, err = foldBot(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

// eventTypes returns the type strings of a stream, for readable failure output.
func eventTypes(events []event.Event) []string {
	out := make([]string, len(events))

	for i, evt := range events {
		out[i] = string(evt.Type())
	}

	return out
}

// membershipStateEqual compares membership states by value, treating nil and
// empty role slices as equal (fold uses make([]Role, ...) which differs from nil).
func membershipStateEqual(a, b MembershipState) bool {
	if a.ActorID != b.ActorID || a.TenantID != b.TenantID || a.Removed != b.Removed {
		return false
	}

	return slices.Equal(a.Roles, b.Roles)
}

// botStateEqual compares bot states; BotState contains []byte (TokenHash) and
// []string (Scopes), so it is not comparable with ==.
func botStateEqual(a, b BotState) bool {
	return a.Name == b.Name &&
		a.OwnerID == b.OwnerID &&
		bytesEqual(a.TokenHash, b.TokenHash) &&
		slices.Equal(a.Scopes, b.Scopes) &&
		a.Deleted == b.Deleted
}

// bytesEqual compares two byte slices (nil-safe). Local to avoid importing bytes
// solely for this helper.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
