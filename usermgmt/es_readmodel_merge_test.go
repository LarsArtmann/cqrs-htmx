package usermgmt

import (
	"fmt"
	"slices"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// A merge retires a user stream in favour of a survivor. These tests pin the
// read-model half of invariant 5 of the merge wave: every lookup by a merged
// ID, email or external subject answers with the survivor, a merged row leaves
// the live set, and a retired stream stays retired whatever is delivered to it.

// userStream builds the events of one user stream with running versions, so a
// test can list them in the order the journal holds them.
type userStream struct {
	t     *testing.T
	aggID id.StreamID
	// at pins the occurred-at of every event this stream builds from here on;
	// zero leaves the construction clock in charge.
	at      time.Time
	version event.Version
}

// newUserStream returns an empty stream on a fresh aggregate ID.
func newUserStream(t *testing.T) *userStream {
	t.Helper()
	return &userStream{t: t, aggID: id.NewStreamID()}
}

func (s *userStream) next(eventType event.Type, payload any) event.Event {
	s.t.Helper()
	s.version = s.version.Increment()
	var opts []event.Option
	if !s.at.IsZero() {
		opts = append(opts, event.WithOccurredAt(s.at))
	}
	return makeUserEvent(s.t, s.aggID, eventType, s.version, payload, opts...)
}

// atTime pins the occurred-at of every later event on this stream, so merge
// and registration times sort deterministically across a hydrate.
func (s *userStream) atTime(at time.Time) *userStream {
	s.at = at
	return s
}

func (s *userStream) registered(email string) event.Event {
	return s.next(eventUserRegistered, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         email,
		DisplayName:   email,
	})
}

func (s *userStream) linked(provider, subject string) event.Event {
	return s.next(eventExternalAccountLinked, ExternalAccountLinkedPayload{
		SchemaVersion: currentSchemaVersion,
		Provider:      provider,
		Subject:       subject,
	})
}

func (s *userStream) verified(email string) event.Event {
	return s.next(eventEmailVerified, EmailVerifiedPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         email,
	})
}

func (s *userStream) credential(credentialID []byte) event.Event {
	return s.next(eventCredentialAdded, CredentialAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ID:            credentialID,
		PublicKey:     []byte("key-" + string(credentialID)),
	})
}

// merged builds UserMerged, the event the survivor's stream records.
func (s *userStream) merged(losers ...id.StreamID) event.Event {
	loserIDs := make([]UserID, 0, len(losers))
	for _, loser := range losers {
		loserIDs = append(loserIDs, NewUserID(loser.Get()))
	}
	return s.next(eventUserMerged, identitymodel.UserMergedPayload{
		SchemaVersion: currentSchemaVersion,
		LoserIDs:      loserIDs,
		Reason:        "dedupe",
	})
}

// mergedInto builds UserMergedInto, the event that retires this stream.
func (s *userStream) mergedInto(survivor id.StreamID) event.Event {
	return s.next(eventUserMergedInto, identitymodel.UserMergedIntoPayload{
		SchemaVersion: currentSchemaVersion,
		SurvivorID:    NewUserID(survivor.Get()),
		Reason:        "dedupe",
	})
}

func (s *userStream) deleted() event.Event {
	return s.next(eventUserDeleted, UserDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "test",
	})
}

func (s *userStream) userID() UserID { return NewUserID(s.aggID.Get()) }

// survivorMergeEvent builds the survivor's UserMerged naming every loser.
func survivorMergeEvent(survivor *userStream, losers ...*userStream) event.Event {
	loserStreams := make([]id.StreamID, 0, len(losers))
	for _, loser := range losers {
		loserStreams = append(loserStreams, loser.aggID)
	}
	return survivor.merged(loserStreams...)
}

// retireEvents builds each loser's UserMergedInto naming the survivor.
func retireEvents(survivor *userStream, losers ...*userStream) []event.Event {
	events := make([]event.Event, 0, len(losers))
	for _, loser := range losers {
		events = append(events, loser.mergedInto(survivor.aggID))
	}
	return events
}

// applyEvents feeds events to the model in the given order.
func applyEvents(t *testing.T, m *UserReadModel, events ...event.Event) {
	t.Helper()
	for _, evt := range events {
		if err := m.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle %s on stream %s: %v", evt.Type(), evt.StreamID().Get(), err)
		}
	}
}

// completeMerge applies the merge events in the order the service appends them
// (the survivor's record, then each loser's retirement) and returns them, so a
// test can hand the same event values over a second time.
func completeMerge(t *testing.T, m *UserReadModel, survivor *userStream, losers ...*userStream) []event.Event {
	t.Helper()
	events := append([]event.Event{survivorMergeEvent(survivor, losers...)}, retireEvents(survivor, losers...)...)
	applyEvents(t, m, events...)
	return events
}

// assertAnswers checks that one lookup answers with exactly the want user.
func assertAnswers(t *testing.T, label string, lookup func() (*User, bool), want UserID) {
	t.Helper()
	got, ok := lookup()
	if !ok || got == nil {
		t.Fatalf("%s: not found, want user %s", label, want.Get().String())
	}
	if got.ID.Get() != want.Get() {
		t.Errorf("%s: answered with user %s, want %s", label, got.ID.Get().String(), want.Get().String())
	}
}

// assertMissing checks that one lookup answers not-found without a user.
func assertMissing(t *testing.T, label string, lookup func() (*User, bool)) {
	t.Helper()
	got, ok := lookup()
	switch {
	case got != nil:
		t.Errorf("%s: answered with user %s, want not found", label, got.ID.Get().String())
	case ok:
		t.Errorf("%s: found flag is true without a user, want not found", label)
	}
}

func userIDs(users []*User) []string {
	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID.Get().String())
	}
	return ids
}

func userIDStrings(ids []UserID) []string {
	strings := make([]string, 0, len(ids))
	for _, id := range ids {
		strings = append(strings, id.Get().String())
	}
	return strings
}

// mergeScenario registers a survivor and two losers on the model and adopts
// one loser's assets onto the survivor; the merge events themselves are left
// to the test. The survivor shares its email with one loser and owns the older
// mapping; the other loser brings a distinct email and an external subject.
func mergeScenario(t *testing.T, m *UserReadModel) (survivor, distinct, duplicate *userStream) {
	t.Helper()
	survivor = newUserStream(t)
	distinct = newUserStream(t)
	duplicate = newUserStream(t)

	applyEvents(t, m,
		survivor.registered("same@example.com"),
		survivor.linked("pocket-id", "subject-survivor"),
		distinct.registered("other@example.com"),
		distinct.linked("pocket-id", "subject-distinct"),
		distinct.verified("other@example.com"),
		duplicate.registered("same@example.com"),
		// Adoption lands on the survivor as ordinary lifecycle events.
		survivor.credential([]byte("credential-distinct")),
		survivor.linked("pocket-id", "subject-distinct"),
		survivor.verified("same@example.com"),
	)
	return survivor, distinct, duplicate
}

func assertMergedScenario(t *testing.T, m *UserReadModel, survivor, distinct, duplicate *userStream) {
	t.Helper()

	if got := m.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1: merged rows must leave the live set", got)
	}
	if got, want := userIDs(m.AllUsers()), []string{survivor.aggID.Get()}; !slices.Equal(got, want) {
		t.Errorf("AllUsers() = %v, want only the survivor %v", got, want)
	}

	assertAnswers(
		t,
		"FindByID(survivor)",
		func() (*User, bool) { return m.FindByID(survivor.aggID) },
		survivor.userID(),
	)
	assertAnswers(
		t,
		"FindByID(distinct loser)",
		func() (*User, bool) { return m.FindByID(distinct.aggID) },
		survivor.userID(),
	)
	assertAnswers(
		t,
		"FindByID(duplicate loser)",
		func() (*User, bool) { return m.FindByID(duplicate.aggID) },
		survivor.userID(),
	)
	assertAnswers(t, "FindByUserID(distinct loser)", func() (*User, bool) {
		return m.FindByUserID(distinct.userID())
	}, survivor.userID())

	assertAnswers(t, "FindByEmail(survivor email)", func() (*User, bool) {
		return m.FindByEmail("same@example.com")
	}, survivor.userID())
	assertAnswers(t, "FindByEmail(distinct loser email)", func() (*User, bool) {
		return m.FindByEmail("other@example.com")
	}, survivor.userID())
	assertMissing(t, "FindByEmail(unknown)", func() (*User, bool) {
		return m.FindByEmail("nobody@example.com")
	})

	assertAnswers(t, "FindByExternalAccount(survivor subject)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-survivor")
	}, survivor.userID())
	assertAnswers(t, "FindByExternalAccount(distinct loser subject)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-distinct")
	}, survivor.userID())
	assertMissing(t, "FindByExternalAccount(unknown)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-unknown")
	})
}

func TestUserReadModel_Merge_ResolvesEveryLookupToSurvivor(t *testing.T) {
	m := NewUserReadModel()
	survivor, distinct, duplicate := mergeScenario(t, m)

	completeMerge(t, m, survivor, distinct, duplicate)

	assertMergedScenario(t, m, survivor, distinct, duplicate)

	row, ok := m.FindByID(survivor.aggID)
	if !ok {
		t.Fatal("survivor row missing")
	}
	wantMerged := []string{distinct.aggID.Get(), duplicate.aggID.Get()}
	if got := userIDStrings(row.MergedUserIDs); !slices.Equal(got, wantMerged) {
		t.Errorf("MergedUserIDs = %v, want %v", got, wantMerged)
	}
	if len(row.Credentials) != 1 {
		t.Errorf("len(Credentials) = %d, want 1 adopted credential", len(row.Credentials))
	}
	if len(row.ExternalAccounts) != 2 {
		t.Errorf("len(ExternalAccounts) = %d, want the own and the adopted account", len(row.ExternalAccounts))
	}
	if !row.EmailVerified {
		t.Error("EmailVerified = false, want the adopted verification")
	}
	if !row.MergedInto.IsZero() {
		t.Errorf("survivor MergedInto = %s, want zero: the survivor is live", row.MergedInto.Get().String())
	}
}

func TestUserReadModel_Merge_AcceptsEitherEventOrder(t *testing.T) {
	// The two merge events live on different streams, so nothing orders one
	// stream's UserMerged before the other's UserMergedInto.
	t.Run("survivor records first", func(t *testing.T) {
		m := NewUserReadModel()
		survivor, distinct, duplicate := mergeScenario(t, m)

		completeMerge(t, m, survivor, distinct, duplicate)

		assertMergedScenario(t, m, survivor, distinct, duplicate)
	})

	t.Run("losers retire first", func(t *testing.T) {
		m := NewUserReadModel()
		survivor, distinct, duplicate := mergeScenario(t, m)

		applyEvents(t, m, retireEvents(survivor, distinct, duplicate)...)
		applyEvents(t, m, survivorMergeEvent(survivor, distinct, duplicate))

		assertMergedScenario(t, m, survivor, distinct, duplicate)
	})
}

func TestUserReadModel_Merge_KeepFirstMappingFollowsItsOwner(t *testing.T) {
	// Two users share an email and an external subject. The index keeps the
	// OLDER registration (keep-first), so which one a merge retires decides
	// who owns the mapping afterwards.
	setup := func(t *testing.T) (m *UserReadModel, older, newer *userStream) {
		t.Helper()
		m = NewUserReadModel()
		older = newUserStream(t)
		newer = newUserStream(t)
		applyEvents(t, m,
			older.registered("dupe@example.com"),
			older.linked("pocket-id", "subject-dupe"),
			newer.registered("dupe@example.com"),
			newer.linked("pocket-id", "subject-dupe"),
		)
		return m, older, newer
	}

	t.Run("owner merged away", func(t *testing.T) {
		m, older, newer := setup(t)
		completeMerge(t, m, newer, older)

		assertAnswers(t, "FindByEmail", func() (*User, bool) {
			return m.FindByEmail("dupe@example.com")
		}, newer.userID())
		assertAnswers(t, "FindByExternalAccount", func() (*User, bool) {
			return m.FindByExternalAccount("pocket-id", "subject-dupe")
		}, newer.userID())
		assertAnswers(
			t,
			"FindByID(retired owner)",
			func() (*User, bool) { return m.FindByID(older.aggID) },
			newer.userID(),
		)
	})

	t.Run("duplicate merged away", func(t *testing.T) {
		m, older, newer := setup(t)
		completeMerge(t, m, older, newer)

		assertAnswers(t, "FindByEmail", func() (*User, bool) {
			return m.FindByEmail("dupe@example.com")
		}, older.userID())
		assertAnswers(t, "FindByExternalAccount", func() (*User, bool) {
			return m.FindByExternalAccount("pocket-id", "subject-dupe")
		}, older.userID())
		assertAnswers(
			t,
			"FindByID(retired duplicate)",
			func() (*User, bool) { return m.FindByID(newer.aggID) },
			older.userID(),
		)
	})
}

func TestUserReadModel_Merge_FollowsATransitiveChain(t *testing.T) {
	m := NewUserReadModel()
	first := newUserStream(t)
	second := newUserStream(t)
	third := newUserStream(t)
	applyEvents(t, m,
		first.registered("first@example.com"),
		second.registered("second@example.com"),
		third.registered("third@example.com"),
	)

	completeMerge(t, m, second, first)
	completeMerge(t, m, third, second)

	if got := m.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1", got)
	}
	for label, stream := range map[string]*userStream{"first": first, "second": second, "third": third} {
		assertAnswers(
			t,
			"FindByID("+label+")",
			func() (*User, bool) { return m.FindByID(stream.aggID) },
			third.userID(),
		)
	}
	for _, email := range []string{"first@example.com", "second@example.com", "third@example.com"} {
		assertAnswers(t, "FindByEmail("+email+")", func() (*User, bool) {
			return m.FindByEmail(email)
		}, third.userID())
	}

	// Each pointer names the direct survivor of one merge, so the chain is as
	// long as the journal recorded it.
	if got, want := len(m.mergedInto), 2; got != want {
		t.Errorf("len(mergedInto) = %d, want %d", got, want)
	}
	if got, want := m.mergedInto[first.aggID], second.aggID; got != want {
		t.Errorf("mergedInto[first] = %s, want the direct survivor %s", got.Get(), want.Get())
	}
}

func TestUserReadModel_Merge_PointerCycleAnswersNotFound(t *testing.T) {
	m := NewUserReadModel()
	first := newUserStream(t)
	second := newUserStream(t)
	applyEvents(t, m,
		first.registered("first@example.com"),
		second.registered("second@example.com"),
	)

	// The deciders refuse to merge a merged-away identity, so a cycle needs a
	// hand-planted pointer map. Both rows are still present, which makes sure
	// the answer comes from the resolver and not from the rows being absent.
	m.mergedInto[first.aggID] = second.aggID
	m.mergedInto[second.aggID] = first.aggID

	done := make(chan []string, 1)
	go func() {
		failures := make([]string, 0)
		for label, aggID := range map[string]id.StreamID{"first": first.aggID, "second": second.aggID} {
			if got := m.resolve(aggID); !got.IsZero() {
				failures = append(
					failures,
					fmt.Sprintf("resolve(%s) = %s, want zero: no stream on a cycle is live", label, got.Get()),
				)
			}
			if user, ok := m.FindByID(aggID); ok || user != nil {
				failures = append(failures, fmt.Sprintf("FindByID(%s) answered with a user, want not found", label))
			}
		}
		done <- failures
	}()

	select {
	case failures := <-done:
		for _, failure := range failures {
			t.Error(failure)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lookups did not return on a pointer cycle")
	}
}

func TestUserReadModel_Merge_StaleIndexEntryAnswersNotFound(t *testing.T) {
	m := NewUserReadModel()
	stream := newUserStream(t)

	// An index entry can outlive its row (a view write that lost its row).
	// The lookup must answer not-found, never a nil user marked found.
	m.emails["ghost@example.com"] = stream.aggID
	m.externalAccounts[externalAccountKey{provider: "pocket-id", subject: "ghost"}] = stream.aggID

	assertMissing(t, "FindByEmail(stale entry)", func() (*User, bool) {
		return m.FindByEmail("ghost@example.com")
	})
	assertMissing(t, "FindByExternalAccount(stale entry)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "ghost")
	})
}

func TestUserReadModel_Merge_RedeliveredMergeEventsChangeNothing(t *testing.T) {
	m := NewUserReadModel()
	survivor, distinct, duplicate := mergeScenario(t, m)

	// At-least-once delivery can hand the same event values over twice.
	mergeEvents := completeMerge(t, m, survivor, distinct, duplicate)
	before := m.AllUsers()

	applyEvents(t, m, mergeEvents...)

	after := m.AllUsers()
	if len(after) != 1 || len(before) != 1 {
		t.Fatalf("len(AllUsers()) = %d before and %d after re-delivery, want 1", len(before), len(after))
	}
	if got, want := userIDStrings(
		after[0].MergedUserIDs,
	), userIDStrings(
		before[0].MergedUserIDs,
	); !slices.Equal(
		got,
		want,
	) {
		t.Errorf("MergedUserIDs = %v after re-delivery, want %v listed once each", got, want)
	}
	if got, want := len(m.mergedInto), 2; got != want {
		t.Errorf("len(mergedInto) = %d, want %d", got, want)
	}
	assertMergedScenario(t, m, survivor, distinct, duplicate)
}

func TestUserReadModel_Merge_DeletedSurvivorReleasesInheritedAliases(t *testing.T) {
	m := NewUserReadModel()
	survivor, distinct, duplicate := mergeScenario(t, m)
	completeMerge(t, m, survivor, distinct, duplicate)
	applyEvents(t, m, survivor.deleted())

	assertMissing(t, "FindByID(survivor)", func() (*User, bool) { return m.FindByID(survivor.aggID) })
	assertMissing(t, "FindByEmail(survivor email)", func() (*User, bool) {
		return m.FindByEmail("same@example.com")
	})
	assertMissing(t, "FindByEmail(inherited email)", func() (*User, bool) {
		return m.FindByEmail("other@example.com")
	})
	assertMissing(t, "FindByExternalAccount(inherited subject)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-distinct")
	})
	if got := m.Count(); got != 0 {
		t.Errorf("Count() = %d, want 0", got)
	}

	// The inherited aliases were only the deleted survivor's to hold: a new
	// user must be able to claim them.
	fresh := newUserStream(t)
	applyEvents(t, m,
		fresh.registered("other@example.com"),
		fresh.linked("pocket-id", "subject-distinct"),
	)
	assertAnswers(t, "FindByEmail(new owner)", func() (*User, bool) {
		return m.FindByEmail("other@example.com")
	}, fresh.userID())
	assertAnswers(t, "FindByExternalAccount(new owner)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-distinct")
	}, fresh.userID())
}

func TestUserReadModel_Merge_RetiredStreamStaysRetired(t *testing.T) {
	m := NewUserReadModel()
	survivor := newUserStream(t)
	loser := newUserStream(t)
	registration := loser.registered("loser@example.com")
	link := loser.linked("pocket-id", "subject-loser")
	applyEvents(t, m,
		survivor.registered("survivor@example.com"),
		registration,
		link,
	)
	completeMerge(t, m, survivor, loser)

	// After a hydrate the drain can hand events that precede the merge to a
	// stream whose pointer is already known. A retired stream is terminal,
	// so its earlier events must not bring its row back.
	applyEvents(t, m, registration, link)

	if got, want := userIDs(m.AllUsers()), []string{survivor.aggID.Get()}; !slices.Equal(got, want) {
		t.Errorf(
			"AllUsers() = %v, want only the survivor %v: a re-delivered event resurrected the retired row",
			got,
			want,
		)
	}
	assertAnswers(t, "FindByID(loser)", func() (*User, bool) { return m.FindByID(loser.aggID) }, survivor.userID())
	// The retired stream's subject stays resolvable and names the survivor.
	assertAnswers(t, "FindByExternalAccount(loser subject)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-loser")
	}, survivor.userID())
}

func TestUserReadModel_Merge_IntoUnknownSurvivorLeavesNoRow(t *testing.T) {
	m := NewUserReadModel()
	loser := newUserStream(t)
	applyEvents(t, m,
		loser.registered("loser@example.com"),
		loser.linked("pocket-id", "subject-loser"),
		loser.mergedInto(id.NewStreamID()),
	)

	assertMissing(t, "FindByID(loser)", func() (*User, bool) { return m.FindByID(loser.aggID) })
	assertMissing(t, "FindByEmail(loser)", func() (*User, bool) {
		return m.FindByEmail("loser@example.com")
	})
	assertMissing(t, "FindByExternalAccount(loser)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-loser")
	})
}

// unhandledEventTypes lists the event types a read model subscribes to
// without a handler. handleEvent ignores an unhandled type silently, so a
// missing handler would drop events with no signal at all.
func unhandledEventTypes[T any](declared []event.Type, core *readModelCore[T]) []event.Type {
	unhandled := make([]event.Type, 0)
	for _, eventType := range declared {
		if _, ok := core.handlers[eventType]; !ok {
			unhandled = append(unhandled, eventType)
		}
	}
	return unhandled
}

func TestReadModels_HandleEveryDeclaredEventType(t *testing.T) {
	tests := []struct {
		name      string
		unhandled func() []event.Type
	}{
		{"user", func() []event.Type {
			model := NewUserReadModel()
			return unhandledEventTypes(model.EventTypes(), &model.readModelCore)
		}},
		{"membership", func() []event.Type {
			model := NewMembershipReadModel()
			return unhandledEventTypes(model.EventTypes(), &model.readModelCore)
		}},
		{"bot", func() []event.Type {
			model := NewBotReadModel()
			return unhandledEventTypes(model.EventTypes(), &model.readModelCore)
		}},
		{"tenant", func() []event.Type {
			model := NewTenantReadModel()
			return unhandledEventTypes(model.EventTypes(), &model.readModelCore)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.unhandled(); len(got) != 0 {
				t.Errorf("subscribed without a handler: %v", got)
			}
		})
	}
}
