package usermgmt

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// countingViewStore is a kv.ViewStore/ViewQuerier/ViewCounter fake whose only
// job is counting SQL-shaped round-trips: the cache tests assert behavior by
// reading the counters, not by timing.
type countingViewStore struct {
	mu      sync.Mutex
	rows    map[string]*UserView
	gets    int
	sets    int
	deletes int
	queries int
}

func newCountingViewStore() *countingViewStore {
	return &countingViewStore{rows: map[string]*UserView{}}
}

func (s *countingViewStore) Get(_ context.Context, id UserID) (*UserView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gets++

	view, ok := s.rows[id.String()]
	if !ok {
		return nil, kv.ErrNotFound
	}

	cpy := *view

	return &cpy, nil
}

func (s *countingViewStore) Set(_ context.Context, id UserID, view *UserView) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sets++

	cpy := *view
	s.rows[id.String()] = &cpy

	return nil
}

func (s *countingViewStore) Delete(_ context.Context, id UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deletes++
	delete(s.rows, id.String())

	return nil
}

func (s *countingViewStore) Scan(_ context.Context, _ []byte) ([]*UserView, error) {
	return nil, nil //nolint:nilnil // not exercised by the cache tests
}

func (s *countingViewStore) Query(_ context.Context, q kv.ViewQuery) ([]*UserView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queries++

	out := []*UserView{}
	for _, view := range s.rows {
		for _, cond := range q.Conditions {
			if cond.Column == csvColumnEmail && cond.Value == view.Email {
				cpy := *view
				out = append(out, &cpy)
			}
		}
	}

	return out, nil
}

func (s *countingViewStore) Count(_ context.Context, _ kv.ViewQuery) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return int64(len(s.rows)), nil
}

func (s *countingViewStore) snapshot() (gets, sets, deletes, queries int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.gets, s.sets, s.deletes, s.queries
}

// newCachedTestModel builds a SQLUserReadModel over the counting fake with
// the view cache enabled — the seam tests never touch a real database.
func newCachedTestModel(t *testing.T, opts ...ViewCacheOption) (*SQLUserReadModel, *countingViewStore) {
	t.Helper()

	fake := newCountingViewStore()
	model, err := newSQLUserReadModelFromViewStore(fake, fake, fake, opts...)
	if err != nil {
		t.Fatalf("newSQLUserReadModelFromViewStore: %v", err)
	}

	return model, fake
}

// TestViewCache_WriteThroughOnSync_HitsAvoidSQL pins the R9 core claim: the
// projection sync path writes through (store AND cache), so FindByIDSQL
// reads after events never round-trip SQL at all.
func TestViewCache_WriteThroughOnSync_HitsAvoidSQL(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	model, fake := newCachedTestModel(t, WithViewCacheCapacity(8))

	if err := model.Handle(ctx, makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "alice@example.com",
		DisplayName:   "Alice",
		Roles:         []Role{RoleUser},
	})); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	userID := NewUserID(testAggID.Get())

	view, err := model.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("FindByIDSQL after sync: %v", err)
	}

	if view.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", view.Email)
	}

	if gets, sets, _, _ := fake.snapshot(); gets != 0 || sets != 1 {
		t.Fatalf("write-through must make post-event reads cache hits: gets=%d (want 0), sets=%d (want 1)", gets, sets)
	}

	// Repeated reads stay on the cache.
	for range 3 {
		if _, err := model.FindByIDSQL(ctx, userID); err != nil {
			t.Fatalf("FindByIDSQL repeated: %v", err)
		}
	}

	if gets, _, _, _ := fake.snapshot(); gets != 0 {
		t.Fatalf("repeated FindByIDSQL must not hit SQL, got %d store Gets", gets)
	}

	// Caller-owned copies: mutating a returned view never leaks into the cache.
	view.Email = "mutated@example.com"

	again, err := model.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("FindByIDSQL after mutation: %v", err)
	}

	if again.Email != "alice@example.com" {
		t.Errorf("cache must serve isolated copies, got %q", again.Email)
	}
}

// TestViewCache_MissPopulatesFromStore covers the cold path: a row the
// projection never synced is still readable — first Get consults the store,
// the second is a hit.
func TestViewCache_MissPopulatesFromStore(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	model, fake := newCachedTestModel(t, WithViewCacheCapacity(8))

	userID := NewUserID(testAggID.Get())
	fake.mu.Lock()
	fake.rows[userID.String()] = &UserView{Email: "seeded@example.com", DisplayName: "Seeded"}
	fake.mu.Unlock()

	first, err := model.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("first FindByIDSQL: %v", err)
	}

	if first.Email != "seeded@example.com" {
		t.Errorf("email = %q, want seeded@example.com", first.Email)
	}

	if gets, _, _, _ := fake.snapshot(); gets != 1 {
		t.Fatalf("cold read must consult the store exactly once, got %d", gets)
	}

	if _, err := model.FindByIDSQL(ctx, userID); err != nil {
		t.Fatalf("second FindByIDSQL: %v", err)
	}

	if gets, _, _, _ := fake.snapshot(); gets != 1 {
		t.Fatalf("warm read must be a cache hit, store Gets = %d (want 1)", gets)
	}
}

// TestViewCache_EmailChanged_ReplacesEntry is the invalidation test (B52):
// an EmailChanged event flows through Handle → syncToSQL → cache.Set, and
// the very next FindByIDSQL serves the NEW email without touching SQL.
func TestViewCache_EmailChanged_ReplacesEntry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	model, fake := newCachedTestModel(t, WithViewCacheCapacity(8))

	if err := model.Handle(ctx, makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "old@example.com",
		DisplayName:   "Alice",
		Roles:         []Role{RoleUser},
	})); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	if err := model.Handle(ctx, makeEvent(t, eventEmailChanged, 2, EmailChangedPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "new@example.com",
	})); err != nil {
		t.Fatalf("Handle EmailChanged: %v", err)
	}

	userID := NewUserID(testAggID.Get())

	view, err := model.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("FindByIDSQL after EmailChanged: %v", err)
	}

	if view.Email != "new@example.com" {
		t.Fatalf("stale cache entry survived EmailChanged: email = %q, want new@example.com", view.Email)
	}

	if gets, sets, _, _ := fake.snapshot(); gets != 0 || sets != 2 {
		t.Fatalf(
			"EmailChanged must replace the cache entry in the same sync: gets=%d (want 0), sets=%d (want 2)",
			gets,
			sets,
		)
	}

	// The store itself carries the new email too — write-through, not write-behind.
	fake.mu.Lock()
	storeView, inStore := fake.rows[userID.String()]
	fake.mu.Unlock()

	if !inStore || storeView.Email != "new@example.com" {
		t.Fatalf("store must carry the new email, got %+v (present=%v)", storeView, inStore)
	}
}

// TestViewCache_UserDeleted_EvictsEntryAndDoesNotCacheNegatives: deletion
// invalidates the cached entry, and the resulting not-found is served from
// the store on every read (negatives never enter the cache).
func TestViewCache_UserDeleted_EvictsEntryAndDoesNotCacheNegatives(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	model, fake := newCachedTestModel(t, WithViewCacheCapacity(8))

	if err := model.Handle(ctx, makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "gone@example.com",
		DisplayName:   "Ghost",
		Roles:         []Role{RoleUser},
	})); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	if err := model.Handle(ctx, makeEvent(t, eventUserDeleted, 2, UserDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "test",
	})); err != nil {
		t.Fatalf("Handle UserDeleted: %v", err)
	}

	userID := NewUserID(testAggID.Get())

	for range 2 {
		if _, err := model.FindByIDSQL(ctx, userID); err == nil {
			t.Fatal("FindByIDSQL after delete: expected error, got nil")
		}
	}

	if gets, _, deletes, _ := fake.snapshot(); deletes != 1 {
		t.Fatalf("delete must write through exactly once, got %d", deletes)
	} else if gets != 2 {
		t.Fatalf("negative results must NOT be cached: 2 reads after delete must consult the store twice, got %d", gets)
	}
}

// TestViewCache_DisabledByDefault_Passthrough: no options means zero
// behavior change — every read consults the store, exactly like today.
func TestViewCache_DisabledByDefault_Passthrough(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	model, fake := newCachedTestModel(t)

	userID := NewUserID(testAggID.Get())
	fake.mu.Lock()
	fake.rows[userID.String()] = &UserView{Email: "plain@example.com"}
	fake.mu.Unlock()

	for range 2 {
		if _, err := model.FindByIDSQL(ctx, userID); err != nil {
			t.Fatalf("FindByIDSQL passthrough: %v", err)
		}
	}

	if gets, _, _, _ := fake.snapshot(); gets != 2 {
		t.Fatalf("disabled cache must pass every read through: store Gets = %d (want 2)", gets)
	}
}

// TestViewCache_FindByEmailSQL_NeverCached (B51): the secondary-index lookup
// stays a store Query — it neither consults nor populates the by-ID cache.
// Caching it would need dual-key invalidation on every EmailChanged (the
// 2026-07-30 ROADMAP trap this design refuses).
func TestViewCache_FindByEmailSQL_NeverCached(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	model, fake := newCachedTestModel(t, WithViewCacheCapacity(8))

	if err := model.Handle(ctx, makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "searchable@example.com",
		DisplayName:   "Find Me",
		Roles:         []Role{RoleUser},
	})); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	views, err := model.FindByEmailSQL(ctx, "searchable@example.com")
	if err != nil {
		t.Fatalf("FindByEmailSQL: %v", err)
	}

	if len(views) != 1 || views[0].Email != "searchable@example.com" {
		t.Fatalf("FindByEmailSQL result = %+v, want the one registered user", views)
	}

	if gets, _, _, queries := fake.snapshot(); queries != 1 || gets != 0 {
		t.Fatalf(
			"FindByEmailSQL must be a pure Query (queries=%d want 1) and never a cached Get (gets=%d want 0)",
			queries,
			gets,
		)
	}
}

// TestViewCache_InvalidCapacityRejected: enabling with a non-positive
// capacity is a construction error, not a silent no-op.
func TestViewCache_InvalidCapacityRejected(t *testing.T) {
	t.Parallel()

	fake := newCountingViewStore()
	if _, err := newViewCache[UserView, UserID](fake, WithViewCacheCapacity(0)); err == nil {
		t.Fatal("capacity 0 must be rejected when the cache is enabled")
	}
}

// TestViewCacheOptions_NilConfigMeansDisabled pins the ServiceConfig
// translation: nil → no options → disabled; a set config → enabled with the
// given capacity (and TTL only when positive).
func TestViewCacheOptions_NilConfigMeansDisabled(t *testing.T) {
	t.Parallel()

	if opts := viewCacheOptions(nil); opts != nil {
		t.Fatalf("nil config must translate to nil options, got %v", opts)
	}

	fake := newCountingViewStore()

	model, err := newSQLUserReadModelFromViewStore(fake, fake, fake, viewCacheOptions(&ViewCacheConfig{Capacity: 4})...)
	if err != nil {
		t.Fatalf("newSQLUserReadModelFromViewStore: %v", err)
	}

	userID := NewUserID(testAggID.Get())
	fake.mu.Lock()
	fake.rows[userID.String()] = &UserView{Email: "threaded@example.com"}
	fake.mu.Unlock()

	for range 2 {
		if _, err := model.FindByIDSQL(t.Context(), userID); err != nil {
			t.Fatalf("FindByIDSQL: %v", err)
		}
	}

	if gets, _, _, _ := fake.snapshot(); gets != 1 {
		t.Fatalf("ViewCacheConfig{Capacity:4} must enable the cache (1 store Get for 2 reads), got %d", gets)
	}
}
