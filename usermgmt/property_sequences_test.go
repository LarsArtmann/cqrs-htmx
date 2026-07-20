package usermgmt

import (
	"slices"
	"strconv"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"pgregory.net/rapid"
)

// Property-based tests for event SEQUENCE folds (the rapid/Hypothesis-style
// sweet spot). The single-event property tests in property_test.go verify each
// fold branch in isolation; these tests verify invariants that only emerge over
// a RANDOM STREAM of events, with shrinking to find the minimal failing case.
//
// Aggregate under test: User. Sibling generators for Tenant/Membership/Bot
// live in the bottom half of this file.

// foldAllUser folds a sequence of events left-to-right starting from the zero
// state, stopping at the first error. Returns the state reached so far and the
// error (nil on success). This is the reference implementation the App's replay
// path uses; the property tests below assert algebraic laws about it.
func foldAllUser(events []event.Event) (UserState, error) {
	state := UserState{}

	var err error

	for _, evt := range events {
		state, err = foldUser(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

// versionOf converts a slice position/length into an event.Version. Slice sizes
// are bounded by memory and cannot approach uint64 range, so the int->Version
// cast is safe; the nolint centralizes the gosec G115 suppression for all
// sequence-property generators that mint sequential versions.
func versionOf(n int) event.Version {
	return event.Version(uint64(n)) //nolint:gosec // bounded slice index
}

// genUserEventStream generates a VALID user event stream: exactly one
// UserRegistered first, then 0-6 follow-up events (EmailChanged or
// DisplayNameChanged), optionally terminated by a single UserDeleted. Versions
// are sequential starting at 1. All events share propAggID (same aggregate).
func genUserEventStream() *rapid.Generator[[]event.Event] {
	return rapid.Custom(func(t *rapid.T) []event.Event {
		const maxFollowups = 6

		regEmail := rapidEmail().Draw(t, "regEmail")
		regDisplay := rapid.StringMatching(`[A-Z][a-z]{3,10}`).Draw(t, "regDisplay")

		events := []event.Event{
			mustPropEvent(eventUserRegistered, 1, UserRegisteredPayload{
				SchemaVersion: currentSchemaVersion,
				Email:         regEmail,
				DisplayName:   regDisplay,
				Roles:         []Role{RoleUser},
			}),
		}

		followups := rapid.IntRange(0, maxFollowups).Draw(t, "followups")

		for i := range followups {
			newEmail := rapidEmail().Draw(t, "email-"+strconv.Itoa(i))
			newDisplay := rapid.StringMatching(`[A-Z][a-z]{3,10}`).Draw(t, "display-"+strconv.Itoa(i))

			switch rapid.IntRange(0, 2).Draw(t, "kind-"+strconv.Itoa(i)) {
			case 0:
				events = append(events, mustPropEvent(eventEmailChanged,
					event.Version(len(events)+1), EmailChangedPayload{
						SchemaVersion: currentSchemaVersion, Email: newEmail,
					}))
			case 1:
				events = append(events, mustPropEvent(eventDisplayNameChanged,
					event.Version(len(events)+1), DisplayNameChangedPayload{
						SchemaVersion: currentSchemaVersion, DisplayName: newDisplay,
					}))
			default:
				// terminal: UserDeleted ends the stream for this aggregate.
				reason := rapid.StringMatching(`[a-z ]{5,20}`).Draw(t, "reason-"+strconv.Itoa(i))

				events = append(events, mustPropEvent(eventUserDeleted,
					event.Version(len(events)+1), UserDeletedPayload{
						SchemaVersion: currentSchemaVersion, Reason: reason,
					}))

				return events
			}
		}

		return events
	})
}

// TestFoldUserSequence_Associativity is the crown-jewel property: folding a
// whole stream equals folding a prefix, then folding the suffix starting from
// the prefix's result. This catches any fold branch that reads or writes state
// outside the event payload (e.g. a stale closure or a mutated shared slice).
// rapid shrinks any counterexample to the minimal failing sequence.
func TestFoldUserSequence_Associativity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genUserEventStream().Draw(t, "stream")
		split := rapid.IntRange(0, len(events)).Draw(t, "split")

		whole, err := foldAllUser(events)
		if err != nil {
			t.Fatalf("foldAllUser(whole): %v", err)
		}

		prefix, err := foldAllUser(events[:split])
		if err != nil {
			t.Fatalf("foldAllUser(prefix): %v", err)
		}

		chunked, err := foldAllUserFrom(prefix, events[split:])
		if err != nil {
			t.Fatalf("foldAllUser(suffix): %v", err)
		}

		if !userStateEqual(whole, chunked) {
			t.Errorf("associativity broken at split=%d:\nwhole=%+v\nchunked=%+v",
				split, whole, chunked)
		}
	})
}

// TestFoldUserSequence_EmailLastWriteWins: the final email equals the email set
// by the LAST email-setting event in the stream (registration or EmailChanged).
func TestFoldUserSequence_EmailLastWriteWins(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genUserEventStream().Draw(t, "stream")

		state, err := foldAllUser(events)
		if err != nil {
			t.Fatalf("foldAllUser: %v", err)
		}

		want := lastEmailIn(events)
		if state.Email != want {
			t.Errorf("email=%q want last-write-wins %q", state.Email, want)
		}
	})
}

// TestFoldUserSequence_DisplayNameLastWriteWins: same as email, for the display
// name field.
func TestFoldUserSequence_DisplayNameLastWriteWins(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genUserEventStream().Draw(t, "stream")

		state, err := foldAllUser(events)
		if err != nil {
			t.Fatalf("foldAllUser: %v", err)
		}

		want := lastDisplayNameIn(events)
		if state.DisplayName != want {
			t.Errorf("displayName=%q want last-write-wins %q", state.DisplayName, want)
		}
	})
}

// TestFoldUserSequence_TombstoneTerminal: once a UserDeleted appears in the
// stream, the final state is tombstoned (Deleted==true). The generator never
// re-registers after a delete, so this invariant must always hold.
func TestFoldUserSequence_TombstoneTerminal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genUserEventStream().Draw(t, "stream")

		state, err := foldAllUser(events)
		if err != nil {
			t.Fatalf("foldAllUser: %v", err)
		}

		if slices.IndexFunc(events, isUserDeletedEvent) >= 0 && !state.Deleted {
			t.Error("stream contains UserDeleted but final state.Deleted is false")
		}
	})
}

// TestFoldUserSequence_EmailChangeResetsVerification: any EmailChanged in the
// stream (with no later EmailVerified) leaves EmailVerified==false. The
// generator never emits EmailVerified, so after an EmailChanged the flag must
// be false.
func TestFoldUserSequence_EmailChangeResetsVerification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genUserEventStream().Draw(t, "stream")

		state, err := foldAllUser(events)
		if err != nil {
			t.Fatalf("foldAllUser: %v", err)
		}

		hasEmailChange := slices.IndexFunc(events,
			func(e event.Event) bool { return e.Type() == eventEmailChanged }) >= 0
		if hasEmailChange && state.EmailVerified {
			t.Error("EmailChanged must reset EmailVerified to false")
		}
	})
}

// TestFoldUserSequence_UnknownEventHalts: injecting an event with an unknown
// type into a valid stream makes foldAllUser return an error, and the state
// folded BEFORE the unknown event is preserved (returned alongside the error).
func TestFoldUserSequence_UnknownEventHalts(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		events := genUserEventStream().Draw(t, "stream")
		pos := rapid.IntRange(0, len(events)).Draw(t, "pos")

		// Build a bogus event at the boundary version.
		bogus := mustPropEvent("Bogus.Unknown", versionOf(pos+1),
			EmailChangedPayload{SchemaVersion: currentSchemaVersion, Email: "x@x.com"})

		withBogus := slices.Concat(events[:pos], []event.Event{bogus}, events[pos:])

		beforeState, beforeErr := foldAllUser(events[:pos])
		gotState, gotErr := foldAllUser(withBogus)

		if gotErr == nil {
			t.Fatal("expected error from unknown event, got nil")
		}

		// State up to the unknown event must match the clean prefix fold.
		if !userStateEqual(beforeState, gotState) {
			t.Errorf("state before unknown event must be preserved:\nbefore=%+v\ngot=%+v",
				beforeState, gotState)
		}

		if beforeErr != nil {
			t.Fatalf("prefix fold unexpectedly failed: %v", beforeErr)
		}
	})
}

// --- helpers used by the sequence property tests ---

// foldAllUserFrom folds events starting from an explicit initial state.
func foldAllUserFrom(initial UserState, events []event.Event) (UserState, error) {
	state := initial

	var err error

	for _, evt := range events {
		state, err = foldUser(state, evt)
		if err != nil {
			return state, err
		}
	}

	return state, nil
}

// lastEmailIn returns the email set by the last UserRegistered or EmailChanged
// event in the stream, or "" if none (impossible for generator output, which
// always starts with registration).
func lastEmailIn(events []event.Event) string {
	email := ""

	for _, evt := range events {
		switch evt.Type() {
		case eventUserRegistered:
			if p, err := unmarshalPayload[UserRegisteredPayload](evt); err == nil {
				email = p.Email
			}
		case eventEmailChanged:
			if p, err := unmarshalPayload[EmailChangedPayload](evt); err == nil {
				email = p.Email
			}
		}
	}

	return email
}

// lastDisplayNameIn returns the display name set by the last UserRegistered or
// DisplayNameChanged event, or "" if none.
func lastDisplayNameIn(events []event.Event) string {
	display := ""

	for _, evt := range events {
		switch evt.Type() {
		case eventUserRegistered:
			if p, err := unmarshalPayload[UserRegisteredPayload](evt); err == nil {
				display = p.DisplayName
			}
		case eventDisplayNameChanged:
			if p, err := unmarshalPayload[DisplayNameChangedPayload](evt); err == nil {
				display = p.DisplayName
			}
		}
	}

	return display
}

// userStateEqual compares two UserStates for value equality across the fields
// the sequence generator exercises (email, display, deleted, verified). It
// intentionally ignores Credentials/ExternalAccounts/TOTP, which the generator
// never touches.
func userStateEqual(a, b UserState) bool {
	return a.Email == b.Email &&
		a.DisplayName == b.DisplayName &&
		a.Deleted == b.Deleted &&
		a.EmailVerified == b.EmailVerified
}

// isUserDeletedEvent reports whether an event is a UserDeleted tombstone.
// Extracted so the slices.IndexFunc call site stays on one line (golines).
func isUserDeletedEvent(e event.Event) bool {
	return e.Type() == eventUserDeleted
}
