package usermgmt

import (
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
	t       *testing.T
	aggID   id.StreamID
	version event.Version
}

// newUserStream returns a stream whose UserRegistered (version 1) is built but
// not applied; the test decides which read model receives it.
func newUserStream(t *testing.T, email string) *userStream {
	t.Helper()
	stream := &userStream{t: t, aggID: id.NewStreamID()}
	stream.registered(email)
	return stream
}

func (s *userStream) next(eventType event.Type, payload any) event.Event {
	s.t.Helper()
	s.version = s.version.Increment()
	return makeUserEvent(s.t, s.aggID, eventType, s.version, payload)
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
		ExternalAccountCore: ExternalAccountCore{
			Provider: provider,
			Subject:  subject,
		},
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
		CredentialCore: CredentialCore{
			ID:        credentialID,
			PublicKey: []byte("key-" + string(credentialID)),
		},
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

// applyEvents feeds events to the model in the given order.
func applyEvents(t *testing.T, m *UserReadModel, events ...event.Event) {
	t.Helper()
	for _, evt := range events {
		if err := m.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle %s on stream %s: %v", evt.Type(), evt.StreamID().Get(), err)
		}
	}
}

// assertAnswers checks that one lookup answers with exactly the want user.
func assertAnswers(t *testing.T, label string, lookup func() (*User, bool), want UserID) {
	t.Helper()
	got, ok := lookup()
	if !ok {
		t.Fatalf("%s: not found, want user %s", label, want.Get())
	}
	if got == nil {
		t.Fatalf("%s: found flag is true but the user is nil", label)
	}
	if got.ID.Get() != want.Get() {
		t.Errorf("%s: answered with user %s, want %s", label, got.ID.Get(), want.Get())
	}
}

// assertMissing checks that one lookup answers not-found without a user.
func assertMissing(t *testing.T, label string, lookup func() (*User, bool)) {
	t.Helper()
	got, ok := lookup()
	if ok || got != nil {
		t.Errorf("%s: answered with %#v (found=%t), want not found", label, got, ok)
	}
}

func mergedUserIDs(users []*User) []string {
	ids := make([]string, 0, len(users))
	for _, id := range users {
		ids = append(ids, id.Get())
	}
	return ids
}

// mergedScenario registers a survivor and two losers on the model and merges
// both losers into the survivor in the order the service appends the events.
// The survivor shares its email with one loser and owns the older mapping; the
// other loser brings a distinct email and an external subject of its own.
func mergedScenario(t *testing.T, m *UserReadModel) (survivor, distinct, duplicate *userStream) {
	t.Helper()
	survivor = newUserStream(t, "same@example.com")
	distinct = newUserStream(t, "other@example.com")
	duplicate = newUserStream(t, "same@example.com")

	applyEvents(t, m,
		survivor.registered("same@example.com"),
	)
	applyEvents(t, m, survivor.linked("pocket-id", "subject-survivor"))
	applyEvents(t, m,
		distinct.registered("other@example.com"),
		distinct.linked("pocket-id", "subject-distinct"),
		distinct.verified("other@example.com"),
	)
	applyEvents(t, m, duplicate.registered("same@example.com"))

	// Adoption lands on the survivor as ordinary lifecycle events, then
	// UserMerged closes the survivor's append and each loser is retired.
	applyEvents(t, m,
		survivor.credential([]byte("credential-distinct")),
		survivor.linked("pocket-id", "subject-distinct"),
		survivor.verified("same@example.com"),
	)
	mergeInto(t, m, survivor, distinct, duplicate)
	return survivor, distinct, duplicate
}

// mergeInto appends UserMerged on the survivor and UserMergedInto on each loser.
func mergeInto(t *testing.T, m *UserReadModel, survivor *userStream, losers ...*userStream) {
	t.Helper()
	loserStreams := make([]id.StreamID, 0, len(losers))
	for _, loser := range losers {
		loserStreams = append(loserStreams, loser.aggID)
	}
	applyEvents(t, m, survivor.merged(loserStreams...))
	for _, loser := range losers {
		applyEvents(t, m, loser.mergedInto(survivor.aggID))
	}
}

func assertMergedScenario(t *testing.T, m *UserReadModel, survivor, distinct, duplicate *userStream) {
	t.Helper()

	if got := m.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1: merged rows must leave the live set", got)
	}
	if got := mergedUserIDs(m.AllUsers()); !slices.Equal(got, []string{survivor.aggID.Get()}) {
		t.Errorf("AllUsers() = %v, want only the survivor %s", got, survivor.aggID.Get())
	}

	assertAnswers(t, "FindByID(survivor)", func() (*User, bool) { return m.FindByID(survivor.aggID) }, survivor.userID())
	assertAnswers(t, "FindByID(distinct loser)", func() (*User, bool) { return m.FindByID(distinct.aggID) }, survivor.userID())
	assertAnswers(t, "FindByID(duplicate loser)", func() (*User, bool) { return m.FindByID(duplicate.aggID) }, survivor.userID())
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
	survivor, distinct, duplicate := mergedScenario(t, m)

	assertMergedScenario(t, m, survivor, distinct, duplicate)

	row, ok := m.FindByID(survivor.aggID)
	if !ok {
		t.Fatalf("survivor row missing")
	}
	wantMerged := []string{distinct.aggID.Get(), duplicate.aggID.Get()}
	if got := mergedUserIDs(row.MergedUserIDs); !slices.Equal(got, wantMerged) {
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
		t.Errorf("survivor MergedInto = %s, want zero: the survivor is live", row.MergedInto.Get())
	}
}

func TestUserReadModel_Merge_AcceptsEitherEventOrder(t *testing.T) {
	// The two merge events live on different streams, so nothing orders one
	// stream's UserMerged before the other's UserMergedInto.
	t.Run("retire first", func(t *testing.T) {
		m := NewUserReadModel()
		survivor, distinct, duplicate := mergedScenario(t, m)
		assertMergedScenario(t, m, survivor, distinct, duplicate)
	})

	t.Run("record first", func(t *testing.T) {
		m := NewUserReadModel()
		survivor := newUserStream(t, "same@example.com")
		distinct := newUserStream(t, "other@example.com")
		duplicate := newUserStream(t, "same@example.com")

		applyEvents(t, m,
			distinct.registered("other@example.com"),
			distinct.linked("pocket-id", "subject-distinct"),
			duplicate.registered("same@example.com"),
		)
		applyEvents(t, m,
			survivor.registered("same@example.com"),
			survivor.linked("pocket-id", "subject-survivor"),
		)
		applyEvents(t, m, survivor.merged(distinct.aggID, duplicate.aggID))
		applyEvents(t, m,
			distinct.mergedInto(survivor.aggID),
			duplicate.mergedInto(survivor.aggID),
		)

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
		older = newUserStream(t, "dupe@example.com")
		newer = newUserStream(t, "dupe@example.com")
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
		mergeInto(t, m, newer, older)

		assertAnswers(t, "FindByEmail", func() (*User, bool) {
			return m.FindByEmail("dupe@example.com")
		}, newer.userID())
		assertAnswers(t, "FindByExternalAccount", func() (*User, bool) {
			return m.FindByExternalAccount("pocket-id", "subject-dupe")
		}, newer.userID())
		assertAnswers(t, "FindByID(retired owner)", func() (*User, bool) { return m.FindByID(older.aggID) }, newer.userID())
	})

	t.Run("duplicate merged away", func(t *testing.T) {
		m, older, newer := setup(t)
		mergeInto(t, m, older, newer)

		assertAnswers(t, "FindByEmail", func() (*User, bool) {
			return m.FindByEmail("dupe@example.com")
		}, older.userID())
		assertAnswers(t, "FindByExternalAccount", func() (*User, bool) {
			return m.FindByExternalAccount("pocket-id", "subject-dupe")
		}, older.userID())
		assertAnswers(t, "FindByID(retired duplicate)", func() (*User, bool) { return m.FindByID(newer.aggID) }, older.userID())
	})
}

func TestUserReadModel_Merge_FollowsATransitiveChain(t *testing.T) {
	m := NewUserReadModel()
	first := newUserStream(t, "first@example.com")
	second := newUserStream(t, "second@example.com")
	third := newUserStream(t, "third@example.com")
	applyEvents(t, m,
		first.registered("first@example.com"),
		second.registered("second@example.com"),
		third.registered("third@example.com"),
	)

	mergeInto(t, m, second, first)
	mergeInto(t, m, third, second)

	if got := m.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1", got)
	}
	for label, stream := range map[string]*userStream{"first": first, "second": second, "third": third} {
		assertAnswers(t, "FindByID("+label+")", func() (*User, bool) { return m.FindByID(stream.aggID) }, third.userID())
	}
	for _, email := range []string{"first@example.com", "second@example.com", "third@example.com"} {
		assertAnswers(t, "FindByEmail("+email+")", func() (*User, bool) {
			return m.FindByEmail(email)
		}, third.userID())
	}

	// The pointers name the direct survivor of each merge, so the chain stays
	// as long as the journal recorded it.
	if got, want := len(m.mergedInto), 2; got != want {
		t.Errorf("len(mergedInto) = %d, want %d", got, want)
	}
	if got, want := m.mergedInto[first.aggID], second.aggID; got != want {
		t.Errorf("mergedInto[first] = %s, want the direct survivor %s", got.Get(), want.Get())
	}
}

func TestUserReadModel_Merge_PointerCycleAnswersNotFound(t *testing.T) {
	m := NewUserReadModel()
	first := newUserStream(t, "first@example.com")
	second := newUserStream(t, "second@example.com")
	applyEvents(t, m,
		first.registered("first@example.com"),
		second.registered("second@example.com"),
	)

	// The deciders refuse to merge a merged-away identity, so a cycle needs a
	// hand-planted pointer map. Stale rows under the cycle make sure the
	// answer comes from the resolver and not from the rows being absent.
	m.mergedInto[first.aggID] = second.aggID
	m.mergedInto[second.aggID] = first.aggID

	type result struct {
		first, second id.StreamID
	}
	done := make(chan result, 1)
	go func() {
		done <- result{first: m.resolve(first.aggID), second: m.resolve(second.aggID)}
	}()

	select {
	case got := <-done:
		if !got.first.IsZero() || !got.second.IsZero() {
			t.Errorf("resolve on a cycle answered %s and %s, want zero: no stream on a cycle is live",
				got.first.Get(), got.second.Get())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resolve did not return on a pointer cycle")
	}
}

func TestUserReadModel_Merge_StaleIndexEntryAnswersNotFound(t *testing.T) {
	m := NewUserReadModel()
	stream := newUserStream(t, "ghost@example.com")

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
	survivor, distinct, duplicate := mergedScenario(t, m)
	before := m.AllUsers()

	// At-least-once delivery can hand the merge events over twice.
	applyEvents(t, m,
		survivor.merged(distinct.aggID, duplicate.aggID),
		distinct.mergedInto(survivor.aggID),
		duplicate.mergedInto(survivor.aggID),
	)

	after := m.AllUsers()
	if len(after) != 1 {
		t.Fatalf("len(AllUsers()) = %d, want 1", len(after))
	}
	if got, want := mergedUserIDs(after[0].MergedUserIDs), mergedUserIDs(before[0].MergedUserIDs); !slices.Equal(got, want) {
		t.Errorf("MergedUserIDs = %v after re-delivery, want %v listed once each", got, want)
	}
	if got, want := len(m.mergedInto), 2; got != want {
		t.Errorf("len(mergedInto) = %d, want %d", got, want)
	}
	assertMergedScenario(t, m, survivor, distinct, duplicate)
}

func TestUserReadModel_Merge_DeletedSurvivorReleasesInheritedAliases(t *testing.T) {
	m := NewUserReadModel()
	survivor, _, _ := mergedScenario(t, m)
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
	fresh := newUserStream(t, "other@example.com")
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
	survivor, distinct, _ := mergedScenario(t, m)

	// After a hydrate the drain can re-deliver events that precede the merge
	// on a stream whose pointer is already known. A retired stream is
	// terminal, so the earlier events must not bring its row back.
	applyEvents(t, m,
		distinct.registered("other@example.com"),
		distinct.linked("pocket-id", "subject-late"),
		distinct.verified("other@example.com"),
	)

	if got := m.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1: a re-delivered event resurrected the retired row", got)
	}
	if got := mergedUserIDs(m.AllUsers()); !slices.Equal(got, []string{survivor.aggID.Get()}) {
		t.Errorf("AllUsers() = %v, want only the survivor", got)
	}
	assertMissing(t, "FindByEmail(subject of the retired stream)", func() (*User, bool) {
		return m.FindByExternalAccount("pocket-id", "subject-late")
	})
}

func TestUserReadModel_Merge_IntoUnknownSurvivorLeavesNoRow(t *testing.T) {
	m := NewUserReadModel()
	loser := newUserStream(t, "loser@example.com")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.unhandled(); len(got) != 0 {
				t.Errorf("subscribed without a handler: %v", got)
			}
		})
	}
}
