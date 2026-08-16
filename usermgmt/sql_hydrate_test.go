package usermgmt

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func TestSQLUserReadModel_HydrateRoundTrip(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)

	rm, err := NewSQLiteUserReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel: %v", err)
	}

	events := []event.Event{
		makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "hydrate@example.com",
			DisplayName:   "Hydrate",
			Roles:         []Role{RoleUser},
		}),
		makeEvent(t, eventCredentialAdded, 2, CredentialAddedPayload{
			SchemaVersion: currentSchemaVersion,
			CredentialCore: CredentialCore{
				ID:              []byte("cred-hydrate-1"),
				PublicKey:       []byte("pk"),
				AttestationType: "none",
			},
		}),
		makeEvent(t, eventTOTPEnabled, 3, TOTPEnabledPayload{
			SchemaVersion: currentSchemaVersion,
			Secret:        []byte("totp-secret-bytes"),
		}),
		makeEvent(t, eventExternalAccountLinked, 4, ExternalAccountLinkedPayload{
			SchemaVersion: currentSchemaVersion,
			ExternalAccountCore: ExternalAccountCore{
				Provider: "github",
				Subject:  "42",
				Email:    "hydrate@example.com",
			},
		}),
	}
	for _, evt := range events {
		if err := rm.Handle(ctx, evt); err != nil {
			t.Fatalf("Handle %s: %v", evt.Type(), err)
		}
	}

	fresh, err := NewSQLiteUserReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel (fresh): %v", err)
	}
	if err := fresh.Hydrate(ctx); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}

	user, ok := fresh.FindByID(testAggID)
	if !ok {
		t.Fatal("FindByID after hydrate: user not found")
	}
	if user.Email != "hydrate@example.com" {
		t.Errorf("email = %q, want hydrate@example.com", user.Email)
	}
	if len(user.Credentials) != 1 || string(user.Credentials[0].ID) != "cred-hydrate-1" {
		t.Errorf("credentials after hydrate = %+v, want exactly cred-hydrate-1", user.Credentials)
	}
	if !user.TOTPEnabled {
		t.Error("TOTPEnabled = false after hydrate, want true")
	}
	if string(user.TOTPSecret) != "totp-secret-bytes" {
		t.Errorf("TOTPSecret = %q, want totp-secret-bytes (secret must survive the round trip)", user.TOTPSecret)
	}
	if len(user.ExternalAccounts) != 1 || user.ExternalAccounts[0].Provider != "github" {
		t.Errorf("external accounts after hydrate = %+v, want one github account", user.ExternalAccounts)
	}

	if _, ok := fresh.FindByEmail("hydrate@example.com"); !ok {
		t.Error("FindByEmail after hydrate: email index not rebuilt")
	}
	if _, ok := fresh.FindByExternalAccount("github", "42"); !ok {
		t.Error("FindByExternalAccount after hydrate: provider+subject index not rebuilt")
	}
}

func TestSQLMembershipReadModel_HydrateRoundTrip(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)

	rm, err := NewSQLiteMembershipReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteMembershipReadModel: %v", err)
	}

	actor := ActorIDFromUser(NewUserID("01JXMEMBER0000000000000001"))
	tenantID := NewTenantID("01JXTENANT0000000000000000A")
	aggID := deriveMembershipID(actor, tenantID)

	evt := makeEventFor(t, eventMemberAdded, 1, aggID, aggregateTypeMembership, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     actor.Kind().String(),
		ActorID:       actor.String(),
		TenantID:      tenantID.Get(),
		Roles:         []Role{RoleUser},
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle MemberAdded: %v", err)
	}

	fresh, err := NewSQLiteMembershipReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteMembershipReadModel (fresh): %v", err)
	}
	if err := fresh.Hydrate(ctx); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}

	mem, ok := fresh.FindByAggregateID(aggID)
	if !ok {
		t.Fatal("FindByAggregateID after hydrate: membership not found")
	}
	if mem.ActorID != actor {
		t.Errorf("ActorID = %v, want %v (kind must survive the JSON round trip)", mem.ActorID, actor)
	}
	if mem.TenantID.Get() != tenantID.Get() {
		t.Errorf("TenantID = %s, want %s", mem.TenantID.Get(), tenantID.Get())
	}

	byActor := fresh.FindByActor(actor.String())
	if len(byActor) != 1 {
		t.Errorf("FindByActor after hydrate = %d memberships, want 1 (byActor index rebuilt)", len(byActor))
	}
}

func TestSQLTenantReadModel_HydrateRoundTrip(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)

	rm, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteTenantReadModel: %v", err)
	}

	tenantAggID := id.NewStreamID()
	evt := makeEventFor(t, eventTenantCreated, 1, tenantAggID, aggregateTypeTenant, TenantCreatedPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "hydrate-org",
		DisplayName:   "Hydrate Org",
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle TenantCreated: %v", err)
	}

	fresh, err := NewSQLiteTenantReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteTenantReadModel (fresh): %v", err)
	}
	if err := fresh.Hydrate(ctx); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}

	tenant, ok := fresh.FindByID(tenantAggID)
	if !ok {
		t.Fatal("FindByID after hydrate: tenant not found")
	}
	if tenant.Name != "hydrate-org" || tenant.DisplayName != "Hydrate Org" {
		t.Errorf("tenant after hydrate = %+v, want name hydrate-org / display Hydrate Org", tenant)
	}
	if _, ok := fresh.FindByName("hydrate-org"); !ok {
		t.Error("FindByName after hydrate: tenant not found")
	}
}

func TestSQLBotReadModel_HydrateRoundTrip(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)

	rm, err := NewSQLiteBotReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteBotReadModel: %v", err)
	}

	botAggID := id.NewStreamID()
	evt := makeEventFor(t, eventBotRegistered, 1, botAggID, aggregateTypeBot, BotRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "hydrate-bot",
		OwnerID:       NewUserID("01JXOWNER000000000000000001"),
		TokenHash:     []byte("bot-token-hash"),
		Scopes:        []string{"visits:read"},
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle BotRegistered: %v", err)
	}

	fresh, err := NewSQLiteBotReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteBotReadModel (fresh): %v", err)
	}
	if err := fresh.Hydrate(ctx); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}

	bot, ok := fresh.FindByID(botAggID)
	if !ok {
		t.Fatal("FindByID after hydrate: bot not found")
	}
	if bot.Name != "hydrate-bot" || string(bot.TokenHash) != "bot-token-hash" {
		t.Errorf("bot after hydrate = %+v, want name hydrate-bot and token hash preserved", bot)
	}

	byHash, ok := fresh.FindByTokenHash([]byte("bot-token-hash"))
	if !ok {
		t.Fatal("FindByTokenHash after hydrate: bot not found (byTokenHash index rebuilt)")
	}
	if byHash.Name != "hydrate-bot" {
		t.Errorf("FindByTokenHash name = %q, want hydrate-bot", byHash.Name)
	}
}

// At-least-once re-delivery guards: after a crash between a SQL view write
// and its checkpoint save, the post-hydrate drain re-applies events that are
// already reflected in the hydrated state. These handlers must converge, not
// duplicate.

func TestUserReadModel_CredentialAddedIsReplaySafe(t *testing.T) {
	rm := NewUserReadModel()
	if err := rm.Handle(t.Context(), makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "cred-dup@example.com",
	})); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	evt := makeEvent(t, eventCredentialAdded, 2, CredentialAddedPayload{
		SchemaVersion:  currentSchemaVersion,
		CredentialCore: CredentialCore{ID: []byte("cred-dup"), PublicKey: []byte("pk")},
	})

	for range 2 {
		if err := rm.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle CredentialAdded: %v", err)
		}
	}

	user, ok := rm.FindByID(testAggID)
	if !ok {
		t.Fatal("user not found")
	}
	if len(user.Credentials) != 1 {
		t.Errorf("credentials after duplicate delivery = %d, want 1", len(user.Credentials))
	}
}

func TestUserReadModel_ExternalAccountLinkedIsReplaySafe(t *testing.T) {
	rm := NewUserReadModel()
	if err := rm.Handle(t.Context(), makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "dup@example.com",
	})); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	evt := makeEvent(t, eventExternalAccountLinked, 2, ExternalAccountLinkedPayload{
		SchemaVersion:       currentSchemaVersion,
		ExternalAccountCore: ExternalAccountCore{Provider: "google", Subject: "sub-1"},
	})
	for range 2 {
		if err := rm.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle ExternalAccountLinked: %v", err)
		}
	}

	user, ok := rm.FindByID(testAggID)
	if !ok {
		t.Fatal("user not found")
	}
	if len(user.ExternalAccounts) != 1 {
		t.Errorf("external accounts after duplicate delivery = %d, want 1", len(user.ExternalAccounts))
	}
}

func TestMembershipReadModel_MemberAddedIsReplaySafe(t *testing.T) {
	rm := NewMembershipReadModel()
	actor := ActorIDFromUser(NewUserID("01JXMEMBER0000000000000002"))
	tenantID := NewTenantID("01JXTENANT0000000000000000B")
	aggID := deriveMembershipID(actor, tenantID)

	evt := makeEventFor(t, eventMemberAdded, 1, aggID, aggregateTypeMembership, MemberAddedPayload{
		SchemaVersion: currentSchemaVersion,
		ActorKind:     actor.Kind().String(),
		ActorID:       actor.String(),
		TenantID:      tenantID.Get(),
		Roles:         []Role{RoleUser},
	})
	for range 2 {
		if err := rm.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle MemberAdded: %v", err)
		}
	}

	if got := len(rm.FindByActor(actor.String())); got != 1 {
		t.Errorf("FindByActor after duplicate delivery = %d, want 1", got)
	}
	if _, ok := rm.FindByAggregateID(aggID); !ok {
		t.Error("membership not found after duplicate delivery")
	}
}

// openRestartDB opens the file-backed SQLite database shared by the restart
// phases below. Unlike newSQLTestDB's shared-cache memory DB, a file DB
// survives closing and reopening — the point of a restart test.
func openRestartDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite %s: %v", dbPath, err)
	}
	db.SetMaxOpenConns(1)
	if err := OptimizeSQLiteDB(context.Background(), db); err != nil {
		t.Fatalf("OptimizeSQLiteDB: %v", err)
	}

	return db
}

// mustAppend appends events to a stream without concurrency checks, the way
// bulk replay and migrations write events.
func mustAppend(t *testing.T, store event.Store, aggType event.StreamType, events []event.Event) {
	t.Helper()

	ref := id.NewStreamRef(aggType, events[0].StreamID())
	if err := store.AppendBatch(t.Context(), ref, events); err != nil {
		t.Fatalf("AppendBatch %s: %v", aggType, err)
	}
}

// openRestartPhase opens the shared restart database and builds fresh event
// and checkpoint stores over it, the way a restarted process would. The
// caller owns closing the database.
func openRestartPhase(
	t *testing.T,
	dbPath string,
) (*sql.DB, event.Store, event.CheckpointStore) {
	t.Helper()

	db := openRestartDB(t, dbPath)

	store, err := storage.NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	cpStore, err := storage.NewSQLiteCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}

	return db, store, cpStore
}

// TestEventSourcedSetup_CheckpointRestartHydratesReadModels is the restart
// equivalence proof for the CheckpointStore + Hydrate combination:
//
// Phase 1 drains a seeded journal into SQL view stores and checkpoints.
// Phase 2 restarts on the same database: hydrate loads the in-memory maps
// from SQL and the checkpoints make the drain a no-op — the read models must
// answer exactly as they did before the restart, including the TOTP secret.
// Phase 3 simulates a crash between the SQL write and the checkpoint save by
// deleting the checkpoints: the full drain re-applies every event on top of
// the hydrated state and must converge (no duplicated credentials, external
// accounts, or byActor entries).
func TestEventSourcedSetup_CheckpointRestartHydratesReadModels(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "restart.db")

	tenantAggID := id.NewStreamID()
	botAggID := id.NewStreamID()
	actor := ActorIDFromUser(NewUserID("01JXRESTART0000000000000001"))
	tenantID := NewTenantID(tenantAggID.String())
	membershipAggID := deriveMembershipID(actor, tenantID)

	userEvents := []event.Event{
		makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "restart@example.com",
			DisplayName:   "Restart",
			Roles:         []Role{RoleUser},
		}),
		makeEvent(t, eventCredentialAdded, 2, CredentialAddedPayload{
			SchemaVersion: currentSchemaVersion,
			CredentialCore: CredentialCore{
				ID:              []byte("cred-restart"),
				PublicKey:       []byte("pk"),
				AttestationType: "none",
			},
		}),
		makeEvent(t, eventTOTPEnabled, 3, TOTPEnabledPayload{
			SchemaVersion: currentSchemaVersion,
			Secret:        []byte("restart-totp-secret"),
		}),
		makeEvent(t, eventExternalAccountLinked, 4, ExternalAccountLinkedPayload{
			SchemaVersion: currentSchemaVersion,
			ExternalAccountCore: ExternalAccountCore{
				Provider: "github",
				Subject:  "restart-1",
			},
		}),
	}
	membershipEvent := makeEventFor(
		t,
		eventMemberAdded,
		1,
		membershipAggID,
		aggregateTypeMembership,
		MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      tenantID.Get(),
			Roles:         []Role{RoleUser},
		},
	)
	tenantEvent := makeEventFor(t, eventTenantCreated, 1, tenantAggID, aggregateTypeTenant, TenantCreatedPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "restart-org",
		DisplayName:   "Restart Org",
	})
	botEvent := makeEventFor(t, eventBotRegistered, 1, botAggID, aggregateTypeBot, BotRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Name:          "restart-bot",
		OwnerID:       NewUserID(testAggID.String()),
		TokenHash:     []byte("restart-token-hash"),
		Scopes:        []string{"visits:read"},
	})

	newSetup := func(db *sql.DB, store event.Store, cpStore event.CheckpointStore) *EventSourcedSetup {
		t.Helper()

		setup, err := NewEventSourcedSetup(EventSourcedConfig{
			EventStore:       store,
			ReadModelDB:      db,
			ReadModelDialect: "sqlite",
			CheckpointStore:  cpStore,
		})
		if err != nil {
			t.Fatalf("NewEventSourcedSetup: %v", err)
		}
		t.Cleanup(func() { _ = setup.Close() })

		return setup
	}

	assertState := func(setup *EventSourcedSetup, phase string) {
		t.Helper()
		assertRestartState(t, setup, actor, tenantID, phase)
	}

	// Phase 1: seed the journal, then drain it (synchronous startup) so the
	// view stores and checkpoints fill.
	db1, store1, cp1 := openRestartPhase(t, dbPath)
	if err := storage.SQLiteInitSchema(context.Background(), db1); err != nil {
		t.Fatalf("init event schema: %v", err)
	}
	if _, err := db1.Exec(storage.SQLiteCheckpointSchema()); err != nil {
		t.Fatalf("exec checkpoint schema: %v", err)
	}

	mustAppend(t, store1, aggregateTypeUser, userEvents)
	mustAppend(t, store1, aggregateTypeMembership, []event.Event{membershipEvent})
	mustAppend(t, store1, aggregateTypeTenant, []event.Event{tenantEvent})
	mustAppend(t, store1, aggregateTypeBot, []event.Event{botEvent})

	setup1 := newSetup(db1, store1, cp1)
	assertState(setup1, "phase 1 (initial drain)")

	if cp, err := cp1.Load(t.Context(), "user-read-model"); err != nil {
		t.Fatalf("load checkpoint: %v", err)
	} else if cp.IsZero() {
		t.Fatal("phase 1: user-read-model checkpoint was not saved")
	}

	// Close phase 1 fully: setup (projection host, bus, store handle) and the
	// database connection — a real process restart.
	if err := setup1.Close(); err != nil {
		t.Fatalf("close setup1: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close db1: %v", err)
	}

	// Phase 2: restart on the same file. Hydrate loads the in-memory maps
	// from SQL; the saved checkpoints make the drain replay nothing.
	db2, store2, cp2 := openRestartPhase(t, dbPath)

	setup2 := newSetup(db2, store2, cp2)
	assertState(setup2, "phase 2 (hydrate + checkpoint resume)")

	if err := setup2.Close(); err != nil {
		t.Fatalf("close setup2: %v", err)
	}
	if err := db2.Close(); err != nil {
		t.Fatalf("close db2: %v", err)
	}

	// Phase 3: crash simulation — drop the checkpoints (as if every drain
	// batch had written its SQL views but died before saving the checkpoint)
	// and restart. The full drain re-applies all events onto the hydrated
	// state and must converge without duplicating anything.
	db3, store3, cp3 := openRestartPhase(t, dbPath)
	if _, err := db3.Exec("DELETE FROM checkpoints"); err != nil {
		t.Fatalf("delete checkpoints: %v", err)
	}

	setup3 := newSetup(db3, store3, cp3)
	assertState(setup3, "phase 3 (hydrate + full re-apply after lost checkpoints)")
}

// assertRestartState asserts the full expected identity state — read models,
// secondary indexes, TOTP secret, and the in-memory casbin role assignment —
// after any restart phase (initial drain, checkpoint resume, or full re-apply).
func assertRestartState(
	t *testing.T,
	setup *EventSourcedSetup,
	actor ActorID,
	tenantID TenantID,
	phase string,
) {
	t.Helper()

	user, ok := setup.ReadModel.FindByID(testAggID)
	if !ok {
		t.Fatalf("%s: user not found", phase)
	}
	if user.Email != "restart@example.com" {
		t.Errorf("%s: email = %q, want restart@example.com", phase, user.Email)
	}
	if len(user.Credentials) != 1 {
		t.Errorf("%s: credentials = %d, want 1", phase, len(user.Credentials))
	}
	if len(user.ExternalAccounts) != 1 {
		t.Errorf("%s: external accounts = %d, want 1", phase, len(user.ExternalAccounts))
	}
	if string(user.TOTPSecret) != "restart-totp-secret" {
		t.Errorf("%s: TOTPSecret = %q, want restart-totp-secret", phase, user.TOTPSecret)
	}
	if _, ok := setup.ReadModel.FindByExternalAccount("github", "restart-1"); !ok {
		t.Errorf("%s: external account index not rebuilt", phase)
	}

	if got := len(setup.MembershipReadModel.FindByActor(actor.String())); got != 1 {
		t.Errorf("%s: FindByActor = %d memberships, want 1", phase, got)
	}
	if _, ok := setup.TenantReadModel.FindByName("restart-org"); !ok {
		t.Errorf("%s: tenant restart-org not found", phase)
	}
	if _, ok := setup.BotReadModel.FindByTokenHash([]byte("restart-token-hash")); !ok {
		t.Errorf("%s: bot token hash not found", phase)
	}

	// casbin-projection is in-memory only, so it must NOT be checkpointed:
	// it replays the full journal on every start and the role assignment
	// from the MemberAdded event must survive the restart.
	roles, err := setup.casbinProjection.authz.RolesForActor(actor, tenantID)
	if err != nil {
		t.Fatalf("%s: RolesForActor: %v", phase, err)
	}
	if !slices.Contains(roles, RoleUser) {
		t.Errorf("%s: casbin roles for actor = %v, want %s (checkpoint must not starve casbin)",
			phase, roles, RoleUser)
	}
}

// hydratableStub adapts the shared stubProjection with a Hydrate method, so
// the wrapper sees it as checkpoint-eligible.
type hydratableStub struct{ *stubProjection }

func (hydratableStub) Hydrate(context.Context) error { return nil }

// mapCheckpointStore is a minimal in-memory event.CheckpointStore.
type mapCheckpointStore struct {
	data map[string]event.Checkpoint
}

func (s *mapCheckpointStore) Save(_ context.Context, name string, cp event.Checkpoint) error {
	s.data[name] = cp
	return nil
}

func (s *mapCheckpointStore) Load(_ context.Context, name string) (event.Checkpoint, error) {
	return s.data[name], nil
}

func TestHydrationAwareCheckpointStore_OnlyHydratableProjectionsKeepCheckpoints(t *testing.T) {
	inner := &mapCheckpointStore{data: map[string]event.Checkpoint{}}
	projections := []projection.Projection{
		&stubProjection{name: "user-read-model"},
		&stubProjection{name: "casbin-projection"},
		hydratableStub{&stubProjection{name: "audit-log"}},
	}

	store := newHydrationAwareCheckpointStore(inner, projections)

	cp := event.Checkpoint{EventID: id.NewEventID()}

	// Non-hydratable: Save is a no-op and Load always reports zero, so the
	// projection replays the full journal on every start.
	if err := store.Save(t.Context(), "casbin-projection", cp); err != nil {
		t.Fatalf("save casbin-projection: %v", err)
	}
	if got, err := store.Load(t.Context(), "casbin-projection"); err != nil {
		t.Fatalf("load casbin-projection: %v", err)
	} else if !got.IsZero() {
		t.Errorf("casbin-projection checkpoint = %v, want zero (never checkpointed)", got.EventID)
	}
	if _, saved := inner.data["casbin-projection"]; saved {
		t.Error("casbin-projection checkpoint leaked through to the inner store")
	}

	// Hydratable: checkpoints pass through untouched.
	if err := store.Save(t.Context(), "audit-log", cp); err != nil {
		t.Fatalf("save audit-log: %v", err)
	}
	if got, err := store.Load(t.Context(), "audit-log"); err != nil {
		t.Fatalf("load audit-log: %v", err)
	} else if got.EventID != cp.EventID {
		t.Errorf("audit-log checkpoint = %v, want %v", got.EventID, cp.EventID)
	}

	// Unknown names fail safe: zero checkpoint, no persistence.
	if err := store.Save(t.Context(), "unknown-projection", cp); err != nil {
		t.Fatalf("save unknown: %v", err)
	}
	if got, err := store.Load(t.Context(), "unknown-projection"); err != nil {
		t.Fatalf("load unknown: %v", err)
	} else if !got.IsZero() {
		t.Errorf("unknown checkpoint = %v, want zero", got.EventID)
	}
}
