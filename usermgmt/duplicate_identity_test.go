package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Regression for the live 2026-10-04 browser-history outage: duplicate
// registrations sharing an email and an OAuth subject made newest-wins
// hydration re-point every login at the newest duplicate while the data
// (visits, agent tokens) stayed attributed to the original user — the
// dashboard rendered empty ("WebUI shows no data"). Identity lookups must
// resolve to the OLDEST registration, and deleting a duplicate must never
// evict the original's mapping.

// makeUserEvent builds an event on an explicit stream so two users can be
// replayed independently (makeEvent is pinned to the shared testAggID).
func makeUserEvent(
	t *testing.T,
	aggID id.StreamID,
	eventType event.Type,
	version event.Version,
	payload any,
) event.Event {
	t.Helper()

	payloadBytes, err := marshalPayload(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", eventType, err)
	}

	evt, err := event.NewEvent(eventType, aggID, aggregateTypeUser, version, payloadBytes)
	if err != nil {
		t.Fatalf("makeUserEvent %s: %v", eventType, err)
	}

	return evt
}

func TestUserReadModel_DuplicateIdentityKeepsOldest(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)

	rm, err := NewSQLiteUserReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel: %v", err)
	}

	original := id.NewStreamID()
	duplicate := id.NewStreamID()

	const (
		email    = "dupe@example.com"
		provider = "pocket-id"
		subject  = "sub-88bec46e"
	)

	events := []event.Event{
		makeUserEvent(t, original, eventUserRegistered, 1, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         email,
			DisplayName:   "Original",
			Roles:         []Role{RoleUser},
		}),
		makeUserEvent(t, original, eventExternalAccountLinked, 2, ExternalAccountLinkedPayload{
			SchemaVersion: currentSchemaVersion,
			ExternalAccountCore: ExternalAccountCore{
				Provider: provider, Subject: subject, Email: email,
			},
		}),
		makeUserEvent(t, duplicate, eventUserRegistered, 1, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         email,
			DisplayName:   "Duplicate",
			Roles:         []Role{RoleUser},
		}),
		makeUserEvent(t, duplicate, eventExternalAccountLinked, 2, ExternalAccountLinkedPayload{
			SchemaVersion: currentSchemaVersion,
			ExternalAccountCore: ExternalAccountCore{
				Provider: provider, Subject: subject, Email: email,
			},
		}),
	}
	for _, evt := range events {
		if err := rm.Handle(ctx, evt); err != nil {
			t.Fatalf("Handle %s: %v", evt.Type(), err)
		}
	}

	originalID := NewUserID(original.Get())

	assertOldestWins := func(stage string) {
		t.Helper()

		user, ok := rm.FindByEmail(email)
		if !ok {
			t.Fatalf("%s: FindByEmail(%q) not found", stage, email)
		}
		if user.ID.Get() != originalID.Get() {
			t.Errorf("%s: FindByEmail resolved to newest duplicate %s, want original %s",
				stage, user.ID.Get(), originalID.Get())
		}

		user, ok = rm.FindByExternalAccount(provider, subject)
		if !ok {
			t.Fatalf("%s: FindByExternalAccount(%s, %s) not found", stage, provider, subject)
		}
		if user.ID.Get() != originalID.Get() {
			t.Errorf("%s: FindByExternalAccount resolved to newest duplicate %s, want original %s",
				stage, user.ID.Get(), originalID.Get())
		}
	}

	assertOldestWins("live event path")

	// Hydration path: a fresh read model rebuilding its maps from SQL views
	// must land on the same (original) user.
	fresh, err := NewSQLiteUserReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel (fresh): %v", err)
	}
	if err := fresh.Hydrate(ctx); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}
	rm = fresh

	assertOldestWins("hydrate path")

	// Deleting the duplicate must not evict the original's mappings.
	if err := rm.Handle(ctx, makeUserEvent(t, duplicate, eventUserDeleted, 3, UserDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "dedupe",
	})); err != nil {
		t.Fatalf("Handle UserDeleted: %v", err)
	}

	assertOldestWins("after duplicate delete")

	if _, ok := rm.FindByID(duplicate); ok {
		t.Error("duplicate user still resolvable after UserDeleted")
	}
}
