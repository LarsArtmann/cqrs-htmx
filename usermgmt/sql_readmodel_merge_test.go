package usermgmt

import (
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// The SQL half of the merge wave: a merge writes a tombstone row before the
// in-memory row goes away, a restart rebuilds pointers and lookups from those
// rows, and nothing a retired stream receives afterwards brings it back.
// The in-memory contract these tests restart from lives in
// es_readmodel_merge_test.go; both files share the userStream builders.

// mergeTestBase timestamps every event in this file deterministically: a
// user's registration time is its stream's pinned time, and a tombstone's
// UpdatedAt is its UserMergedInto time. One-minute gaps keep the RFC3339
// (second-precision) view columns strictly ordered.
var mergeTestBase = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

func mergeTestTime(minutes int) time.Time {
	return mergeTestBase.Add(time.Duration(minutes) * time.Minute)
}

// mustSQLUserModel builds a SQLite-backed user read model over a fresh test DB.
func mustSQLUserModel(t *testing.T, db *sql.DB, opts ...ViewCacheOption) *SQLUserReadModel {
	t.Helper()
	m, err := NewSQLiteUserReadModel(db, opts...)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel: %v", err)
	}
	return m
}

// applyToSQL feeds events through the SQL model's Handle so every event also
// lands in the view store (applyEvents only takes the in-memory model).
func applyToSQL(t *testing.T, m *SQLUserReadModel, events ...event.Event) {
	t.Helper()
	for _, evt := range events {
		if err := m.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle %s on stream %s: %v", evt.Type(), evt.StreamID().Get(), err)
		}
	}
}

// sqlMergeScenario is mergeScenario applied through the SQL model, with every
// stream pinned to a deterministic time (survivor oldest, duplicate newest).
func sqlMergeScenario(t *testing.T, m *SQLUserReadModel) (survivor, distinct, duplicate *userStream) {
	t.Helper()
	survivor = newUserStream(t).atTime(mergeTestTime(0))
	distinct = newUserStream(t).atTime(mergeTestTime(1))
	duplicate = newUserStream(t).atTime(mergeTestTime(2))
	applyToSQL(t, m,
		survivor.registered("same@example.com"),
		survivor.linked("pocket-id", "subject-survivor"),
		distinct.registered("other@example.com"),
		distinct.linked("pocket-id", "subject-distinct"),
		distinct.verified("other@example.com"),
		duplicate.registered("same@example.com"),
		survivor.credential([]byte("credential-distinct")),
		survivor.linked("pocket-id", "subject-distinct"),
		survivor.verified("same@example.com"),
	)
	return survivor, distinct, duplicate
}

// completeSQLMerge applies the merge events (survivor record, then each
// retirement) at a deterministic merge time.
func completeSQLMerge(
	t *testing.T, m *SQLUserReadModel, survivor *userStream, losers ...*userStream,
) {
	t.Helper()
	survivor.atTime(mergeTestTime(5))
	for _, loser := range losers {
		loser.atTime(mergeTestTime(5))
	}
	applyToSQL(t, m, append(
		[]event.Event{survivorMergeEvent(survivor, losers...)},
		retireEvents(survivor, losers...)...,
	)...)
}

// hydratedUserModel rebuilds a fresh model over the same rows, the way a
// restart with a checkpoint store does.
func hydratedUserModel(t *testing.T, db *sql.DB) *SQLUserReadModel {
	t.Helper()
	fresh := mustSQLUserModel(t, db)
	if err := fresh.Hydrate(t.Context()); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}
	return fresh
}

// viewUser decodes the domain user a stored view row carries.
func viewUser(t *testing.T, view *UserView) *User {
	t.Helper()
	user, err := unmarshalUserViewData(view.Data)
	if err != nil {
		t.Fatalf("unmarshalUserViewData: %v", err)
	}
	return user
}

func TestSQLUserReadModel_Merge_TombstoneRowAndSurvivorRecord(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)
	survivor, distinct, duplicate := sqlMergeScenario(t, m)

	completeSQLMerge(t, m, survivor, distinct, duplicate)

	for _, loser := range []*userStream{distinct, duplicate} {
		view, err := m.FindByIDSQL(t.Context(), loser.userID())
		if err != nil {
			t.Fatalf("FindByIDSQL(loser %s): %v", loser.aggID.Get(), err)
		}
		if !view.Tombstoned {
			t.Errorf("loser %s row Tombstoned = false, want the merge tombstone", loser.aggID.Get())
		}
		if user := viewUser(t, view); user.MergedInto != survivor.userID() {
			t.Errorf("loser %s Data merged_into = %s, want survivor %s",
				loser.aggID.Get(), user.MergedInto.Get().String(), survivor.userID().Get().String())
		}
	}

	count, err := m.CountSQL(t.Context())
	if err != nil {
		t.Fatalf("CountSQL: %v", err)
	}
	if count != 1 {
		t.Errorf("CountSQL = %d, want 1: tombstones are not live rows", count)
	}

	view, err := m.FindByIDSQL(t.Context(), survivor.userID())
	if err != nil {
		t.Fatalf("FindByIDSQL(survivor): %v", err)
	}
	user := viewUser(t, view)
	want := []string{distinct.userID().Get().String(), duplicate.userID().Get().String()}
	if got := userIDStrings(user.MergedUserIDs); !slices.Equal(got, want) {
		t.Errorf("survivor Data merged_user_ids = %v, want %v", got, want)
	}
}

func TestSQLUserReadModel_Merge_HydrateRebuildsSameAnswers(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)
	survivor, distinct, duplicate := sqlMergeScenario(t, m)
	completeSQLMerge(t, m, survivor, distinct, duplicate)

	fresh := hydratedUserModel(t, db)

	assertMergedScenario(t, fresh.UserReadModel, survivor, distinct, duplicate)

	for _, loser := range []*userStream{distinct, duplicate} {
		if !fresh.Retired(loser.aggID) {
			t.Errorf("Retired(loser %s) = false after hydrate, want true", loser.aggID.Get())
		}
		target, ok := fresh.MergeTarget(loser.aggID)
		if !ok || target.Get() != survivor.aggID.Get() {
			t.Errorf("MergeTarget(loser %s) = %s,%v after hydrate, want %s,true",
				loser.aggID.Get(), target.Get(), ok, survivor.aggID.Get())
		}
	}
	if fresh.Retired(survivor.aggID) {
		t.Error("Retired(survivor) = true after hydrate, want false")
	}
	if _, ok := fresh.MergeTarget(survivor.aggID); ok {
		t.Error("MergeTarget(survivor) reported a target after hydrate, want none")
	}

	row, ok := fresh.FindByID(survivor.aggID)
	if !ok {
		t.Fatal("survivor row missing after hydrate")
	}
	want := []string{distinct.userID().Get().String(), duplicate.userID().Get().String()}
	if got := userIDStrings(row.MergedUserIDs); !slices.Equal(got, want) {
		t.Errorf("survivor MergedUserIDs after hydrate = %v, want %v", got, want)
	}
}

func TestSQLUserReadModel_Merge_UnlinkOfInheritedAccountSurvivesRestart(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)
	survivor, distinct, _ := sqlMergeScenario(t, m)
	completeSQLMerge(t, m, survivor, distinct)

	survivor.atTime(mergeTestTime(6))
	applyToSQL(t, m, survivor.next(eventExternalAccountUnlinked, ExternalAccountUnlinkedPayload{
		SchemaVersion: currentSchemaVersion,
		Provider:      "pocket-id",
		Subject:       "subject-distinct",
	}))

	if got, ok := m.FindByExternalAccount("pocket-id", "subject-distinct"); ok {
		t.Fatalf("live model still resolves subject-distinct to %s before restart", got.ID.Get().String())
	}

	fresh := hydratedUserModel(t, db)

	// The survivor no longer holds the inherited account, so the restart must
	// not hand the subject back to the merged row's entry (the G5 gate).
	assertMissing(t, "FindByExternalAccount(unlinked inherited subject) after restart", func() (*User, bool) {
		return fresh.FindByExternalAccount("pocket-id", "subject-distinct")
	})
	assertAnswers(t, "FindByExternalAccount(survivor's own subject) after restart", func() (*User, bool) {
		return fresh.FindByExternalAccount("pocket-id", "subject-survivor")
	}, survivor.userID())
}

func TestSQLUserReadModel_Merge_FindByEmailSQLLiveRowsOnly(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)
	survivor, distinct, duplicate := sqlMergeScenario(t, m)
	completeSQLMerge(t, m, survivor, distinct, duplicate)

	for _, email := range []string{"same@example.com", "other@example.com"} {
		rows, err := m.FindByEmailSQL(t.Context(), email)
		if err != nil {
			t.Fatalf("FindByEmailSQL(%q): %v", email, err)
		}
		if len(rows) != 1 {
			t.Fatalf("FindByEmailSQL(%q) = %d rows, want exactly the survivor", email, len(rows))
		}
		if rows[0].Tombstoned {
			t.Errorf("FindByEmailSQL(%q) returned a tombstone row", email)
		}
		user := viewUser(t, rows[0])
		if user.ID != survivor.userID() {
			t.Errorf("FindByEmailSQL(%q) answered user %s, want survivor %s",
				email, user.ID.Get().String(), survivor.userID().Get().String())
		}
	}

	// The duplicate loser's tombstone holds the shared email too: it must
	// resolve to the survivor once, not add a second row.
	rows, err := m.FindByEmailSQL(t.Context(), "same@example.com")
	if err != nil {
		t.Fatalf("FindByEmailSQL(shared email): %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("shared email resolved to %d rows, want 1 after dedupe", len(rows))
	}

	if _, err := m.FindByEmailSQL(t.Context(), "nobody@example.com"); err != nil {
		t.Fatalf("FindByEmailSQL(unknown): %v", err)
	}
}

func TestSQLUserReadModel_Merge_ViewCacheServesTombstone(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db, WithViewCacheCapacity(16))
	survivor, distinct, _ := sqlMergeScenario(t, m)
	completeSQLMerge(t, m, survivor, distinct)

	view, err := m.FindByIDSQL(t.Context(), distinct.userID())
	if err != nil {
		t.Fatalf("FindByIDSQL(loser) through the cache: %v", err)
	}
	if !view.Tombstoned {
		t.Fatal("cached loser view Tombstoned = false, want the tombstone the merge wrote")
	}
	if got := viewUser(t, view).Email; got != "other@example.com" {
		t.Errorf("cached tombstone email = %q, want the retired user's own email", got)
	}

	// A later change on the survivor must not leak into the cached tombstone:
	// the cache stores its own copy (G8).
	survivor.atTime(mergeTestTime(6))
	applyToSQL(t, m, survivor.next(eventDisplayNameChanged, DisplayNameChangedPayload{
		SchemaVersion: currentSchemaVersion,
		DisplayName:   "Renamed",
	}))

	view, err = m.FindByIDSQL(t.Context(), distinct.userID())
	if err != nil {
		t.Fatalf("FindByIDSQL(loser) after survivor change: %v", err)
	}
	if user := viewUser(t, view); user.DisplayName != "other@example.com" {
		t.Errorf("cached tombstone display_name = %q, want the retired user's own", user.DisplayName)
	}
}

func TestSQLUserReadModel_Merge_RedeliveredEventsAfterHydrateStayRetired(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)
	survivor, distinct, duplicate := sqlMergeScenario(t, m)
	completeSQLMerge(t, m, survivor, distinct, duplicate)

	fresh := hydratedUserModel(t, db)

	// The drain can replay events a retired stream recorded before its merge;
	// none of them may rebuild the row or overwrite the tombstone (D18+G9).
	applyToSQL(t, fresh,
		duplicate.registered("same@example.com"),
		distinct.linked("pocket-id", "subject-distinct"),
		distinct.verified("other@example.com"),
	)

	for _, loser := range []*userStream{distinct, duplicate} {
		view, err := fresh.FindByIDSQL(t.Context(), loser.userID())
		if err != nil {
			t.Fatalf("FindByIDSQL(loser %s) after re-delivery: %v", loser.aggID.Get(), err)
		}
		if !view.Tombstoned {
			t.Errorf("loser %s tombstone lost after re-delivered pre-merge events", loser.aggID.Get())
		}
	}
	assertMergedScenario(t, fresh.UserReadModel, survivor, distinct, duplicate)
	count, err := fresh.CountSQL(t.Context())
	if err != nil {
		t.Fatalf("CountSQL after re-delivery: %v", err)
	}
	if count != 1 {
		t.Errorf("CountSQL = %d after re-delivery, want 1", count)
	}
}

func TestSQLUserReadModel_Merge_DeletedSurvivorReleasesAliasesToClaimant(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)

	survivor := newUserStream(t).atTime(mergeTestTime(0))
	loser := newUserStream(t).atTime(mergeTestTime(1))
	claimant := newUserStream(t).atTime(mergeTestTime(7))

	applyToSQL(t, m,
		survivor.registered("kept@example.com"),
		loser.registered("release@example.com"),
		loser.linked("pocket-id", "subject-release"),
		survivor.linked("pocket-id", "subject-release"),
	)
	completeSQLMerge(t, m, survivor, loser)

	survivor.atTime(mergeTestTime(6))
	applyToSQL(t, m, survivor.deleted())

	applyToSQL(t, m,
		claimant.registered("release@example.com"),
		claimant.linked("pocket-id", "subject-release"),
	)

	fresh := hydratedUserModel(t, db)

	// The survivor is gone, so the merged row's email and subject pass to the
	// row that claimed them after the merge — not back to the tombstone.
	assertAnswers(t, "FindByEmail(released email) after restart", func() (*User, bool) {
		return fresh.FindByEmail("release@example.com")
	}, claimant.userID())
	assertAnswers(t, "FindByExternalAccount(released subject) after restart", func() (*User, bool) {
		return fresh.FindByExternalAccount("pocket-id", "subject-release")
	}, claimant.userID())
	assertMissing(t, "FindByEmail(deleted survivor email) after restart", func() (*User, bool) {
		return fresh.FindByEmail("kept@example.com")
	})
}

func TestSQLUserReadModel_Merge_TransitiveChainAcrossRestart(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)

	final := newUserStream(t).atTime(mergeTestTime(0))
	middle := newUserStream(t).atTime(mergeTestTime(1))
	origin := newUserStream(t).atTime(mergeTestTime(2))

	applyToSQL(t, m,
		final.registered("chain-a@example.com"),
		middle.registered("chain-b@example.com"),
		origin.registered("chain-c@example.com"),
	)

	// origin -> middle, then middle -> final.
	final.atTime(mergeTestTime(5))
	middle.atTime(mergeTestTime(5))
	applyToSQL(t, m, append([]event.Event{survivorMergeEvent(middle, origin)}, retireEvents(middle, origin)...)...)
	final.atTime(mergeTestTime(6))
	middle.atTime(mergeTestTime(6))
	applyToSQL(t, m, append([]event.Event{survivorMergeEvent(final, middle)}, retireEvents(final, middle)...)...)

	fresh := hydratedUserModel(t, db)

	// Pointers stay direct: origin names middle, middle names final.
	if target, ok := fresh.MergeTarget(origin.aggID); !ok || target.Get() != middle.aggID.Get() {
		t.Errorf("MergeTarget(origin) = %s,%v, want middle %s,true",
			target.Get(), ok, middle.aggID.Get())
	}
	if target, ok := fresh.MergeTarget(middle.aggID); !ok || target.Get() != final.aggID.Get() {
		t.Errorf("MergeTarget(middle) = %s,%v, want final %s,true",
			target.Get(), ok, final.aggID.Get())
	}

	assertAnswers(t, "FindByID(origin) through the chain", func() (*User, bool) {
		return fresh.FindByID(origin.aggID)
	}, final.userID())
	assertAnswers(t, "FindByID(middle) through the chain", func() (*User, bool) {
		return fresh.FindByID(middle.aggID)
	}, final.userID())
	for _, email := range []string{"chain-a@example.com", "chain-b@example.com", "chain-c@example.com"} {
		assertAnswers(t, "FindByEmail("+email+") through the chain", func() (*User, bool) {
			return fresh.FindByEmail(email)
		}, final.userID())
	}

	rows, err := fresh.FindByEmailSQL(t.Context(), "chain-c@example.com")
	if err != nil {
		t.Fatalf("FindByEmailSQL(chain-c): %v", err)
	}
	if len(rows) != 1 || viewUser(t, rows[0]).ID != final.userID() {
		t.Errorf("FindByEmailSQL(chain-c) = %d rows, want the final survivor once", len(rows))
	}
}

func TestSQLUserReadModel_Merge_KeepFirstOwnershipAcrossRestart(t *testing.T) {
	setup := func(t *testing.T) (db *sql.DB, m *SQLUserReadModel, older, newer *userStream) {
		t.Helper()
		db = newSQLTestDB(t)
		m = mustSQLUserModel(t, db)
		older = newUserStream(t).atTime(mergeTestTime(0))
		newer = newUserStream(t).atTime(mergeTestTime(1))
		applyToSQL(t, m,
			older.registered("dupe@example.com"),
			older.linked("pocket-id", "subject-dupe"),
			newer.registered("dupe@example.com"),
			newer.linked("pocket-id", "subject-dupe"),
		)
		return db, m, older, newer
	}

	t.Run("older retires into newer", func(t *testing.T) {
		db, m, older, newer := setup(t)
		completeSQLMerge(t, m, newer, older)

		fresh := hydratedUserModel(t, db)

		// The entry followed its owner: the loser's mapping now resolves to
		// the survivor (G6).
		assertAnswers(t, "FindByEmail after restart", func() (*User, bool) {
			return fresh.FindByEmail("dupe@example.com")
		}, newer.userID())
		assertAnswers(t, "FindByExternalAccount after restart", func() (*User, bool) {
			return fresh.FindByExternalAccount("pocket-id", "subject-dupe")
		}, newer.userID())
	})

	t.Run("newer retires into older", func(t *testing.T) {
		db, m, older, newer := setup(t)
		completeSQLMerge(t, m, older, newer)

		fresh := hydratedUserModel(t, db)

		// The survivor already owned the mapping; the merge keeps it.
		assertAnswers(t, "FindByEmail after restart", func() (*User, bool) {
			return fresh.FindByEmail("dupe@example.com")
		}, older.userID())
		assertAnswers(t, "FindByExternalAccount after restart", func() (*User, bool) {
			return fresh.FindByExternalAccount("pocket-id", "subject-dupe")
		}, older.userID())
	})
}

func TestSQLUserReadModel_Merge_PointerOnlyTombstone(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)

	survivor := newUserStream(t).atTime(mergeTestTime(0))
	ghost := newUserStream(t)
	unknown := newUserStream(t)
	stranger := newUserStream(t)

	applyToSQL(t, m, survivor.registered("host@example.com"))

	// A retirement whose stream never registered: the tombstone carries
	// nothing but the pointer, and must survive a restart anyway.
	applyToSQL(t, m, ghost.mergedInto(survivor.aggID))
	// The same shape pointing at a survivor no row holds.
	applyToSQL(t, m, stranger.mergedInto(unknown.aggID))

	fresh := hydratedUserModel(t, db)

	if !fresh.Retired(ghost.aggID) {
		t.Error("Retired(pointer-only loser) = false after hydrate, want true")
	}
	if target, ok := fresh.MergeTarget(ghost.aggID); !ok || target.Get() != survivor.aggID.Get() {
		t.Errorf("MergeTarget(pointer-only loser) = %s,%v, want the live survivor", target.Get(), ok)
	}
	assertAnswers(t, "FindByID(pointer-only loser) resolves to its survivor", func() (*User, bool) {
		return fresh.FindByID(ghost.aggID)
	}, survivor.userID())

	if target, ok := fresh.MergeTarget(stranger.aggID); !ok || target.Get() != unknown.aggID.Get() {
		t.Errorf("MergeTarget(row-less retirement) = %s,%v, want its named survivor", target.Get(), ok)
	}
	assertMissing(t, "FindByID(unknown survivor)", func() (*User, bool) {
		return fresh.FindByID(unknown.aggID)
	})
	assertMissing(t, "FindByID(row-less retirement with no live chain end)", func() (*User, bool) {
		return fresh.FindByID(stranger.aggID)
	})
	assertAnswers(t, "FindByID(survivor beside pointer-only tombstones)", func() (*User, bool) {
		return fresh.FindByID(survivor.aggID)
	}, survivor.userID())
}

func TestSQLUserReadModel_Merge_RedeliveredRetirementKeepsFirstTombstone(t *testing.T) {
	db := newSQLTestDB(t)
	m := mustSQLUserModel(t, db)
	survivor, distinct, _ := sqlMergeScenario(t, m)
	completeSQLMerge(t, m, survivor, distinct)

	first, err := m.FindByIDSQL(t.Context(), distinct.userID())
	if err != nil {
		t.Fatalf("FindByIDSQL(loser): %v", err)
	}

	// The same retirement delivered again must not rewrite the tombstone with
	// an emptier row: the first write is the one that keeps the snapshot.
	applyToSQL(t, m, distinct.mergedInto(survivor.aggID))

	again, err := m.FindByIDSQL(t.Context(), distinct.userID())
	if err != nil {
		t.Fatalf("FindByIDSQL(loser) after re-delivery: %v", err)
	}
	if user := viewUser(t, again); user.Email != "other@example.com" {
		t.Errorf("tombstone email after re-delivered retirement = %q, want the first snapshot's", user.Email)
	}
	if again.UpdatedAt != first.UpdatedAt {
		t.Errorf("tombstone updated_at changed from %s to %s on re-delivery", first.UpdatedAt, again.UpdatedAt)
	}
}
