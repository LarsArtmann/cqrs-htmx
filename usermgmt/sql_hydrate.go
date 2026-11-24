package usermgmt

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"slices"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Hydrator is implemented by SQL-backed read models that can rebuild their
// in-memory maps from their SQL view store. When a CheckpointStore is
// configured, NewEventSourcedSetup and StartProjections hydrate every Hydrator
// projection before the projection host starts: checkpoints make the drain
// skip already-processed events, so without hydration the in-memory maps
// (which serve all reads) would stay empty after a restart.
type Hydrator interface {
	Hydrate(ctx context.Context) error
}

// hydrationAwareCheckpointStore only persists and honors checkpoints for
// projections that implement Hydrator. Checkpointing a projection whose state
// lives purely in memory (casbin-projection, the default AuditLog) would
// permanently starve it: after a restart the checkpoint makes the drain skip
// every pre-checkpoint event, and with no SQL state to hydrate from, the
// projection stays empty forever. For such projections Load always reports a
// zero checkpoint (full replay from the journal start) and Save is a no-op, so
// they rebuild completely on every start.
//
// Unknown projection names get the same treatment — the safe default is a
// full replay, never silently skipped events.
// cqrs-lint:ignore(C035) hydratable is written once in newHydrationAwareCheckpointStore before any projection worker exists and only read afterwards (construction-time immutable; concurrent Save/Load calls are read-only map accesses)
type hydrationAwareCheckpointStore struct {
	inner      event.CheckpointStore
	hydratable map[string]bool
}

func newHydrationAwareCheckpointStore(
	inner event.CheckpointStore,
	projections []projection.Projection,
) event.CheckpointStore {
	hydratable := make(map[string]bool, len(projections))
	for _, p := range projections {
		if _, ok := p.(Hydrator); ok {
			hydratable[p.Name()] = true
		}
	}
	return &hydrationAwareCheckpointStore{inner: inner, hydratable: hydratable}
}

func (s *hydrationAwareCheckpointStore) Save(ctx context.Context, name string, cp event.Checkpoint) error {
	if !s.hydratable[name] {
		return nil
	}
	return s.inner.Save(ctx, name, cp)
}

func (s *hydrationAwareCheckpointStore) Load(ctx context.Context, name string) (event.Checkpoint, error) {
	if !s.hydratable[name] {
		return event.Checkpoint{}, nil //nolint:exhaustruct // zero-value is the "no checkpoint, replay all" contract
	}
	return s.inner.Load(ctx, name)
}

// hydrateProjections hydrates every Hydrator projection, in order, before the
// projection host starts. A hydrate failure aborts construction: serving
// reads from a half-hydrated read model would return wrong results, which is
// worse than failing to start.
func hydrateProjections(
	ctx context.Context,
	logger *slog.Logger,
	projections []projection.Projection,
) error {
	for _, p := range projections {
		h, ok := p.(Hydrator)
		if !ok {
			continue
		}

		if err := h.Hydrate(ctx); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.hydrate", "hydrate "+p.Name()+" from sql")
		}

		logger.Info("hydrated read model from sql", slog.String("projection", p.Name()))
	}

	return nil
}

// userAlias exists to strip User's custom MarshalJSON (which only adds a
// derived credential_count field) so userViewData controls the wire shape.
type userAlias User

// userViewData is the JSON shape of UserView.Data. identitymodel.User hides
// TOTPSecret from JSON (json:"-"), which would silently break TOTP login
// after a checkpointed restart: the hydrated user would report TOTPEnabled
// true but carry no secret. userViewData re-exposes the secret under
// totp_secret. This shape never leaves the read model's SQL table.
type userViewData struct {
	*userAlias
	TOTPSecret []byte `json:"totp_secret,omitempty"`
}

func marshalUserViewData(u *User) (string, error) {
	data, err := json.Marshal(userViewData{
		userAlias:  (*userAlias)(u),
		TOTPSecret: u.TOTPSecret,
	})
	if err != nil {
		return "", errorfamily.WrapInfrastructure(
			err, "usermgmt.sql_readmodel.user_marshal", "marshal user view data")
	}
	return string(data), nil
}

func unmarshalUserViewData(data string) (*User, error) {
	//nolint:exhaustruct // unmarshal target: fields arrive from JSON, not literals
	alias := userAlias{}
	//nolint:exhaustruct // unmarshal target: fields arrive from JSON, not literals
	view := userViewData{userAlias: &alias}
	if err := json.Unmarshal([]byte(data), &view); err != nil {
		return nil, errorfamily.WrapCorruption(
			err, "usermgmt.sql_readmodel.user_unmarshal", "unmarshal user view data")
	}
	u := User(alias)
	u.TOTPSecret = view.TOTPSecret
	return &u, nil
}

// hydrateStreamID parses a branded ID's raw value into the StreamID that
// keys a read model map. View rows do not carry the SQL key column, so the
// entity's own ID is the source of truth.
func hydrateStreamID(raw, errCode, msg string) (id.StreamID, error) {
	aggID, err := id.ParseStreamID(raw)
	if err != nil {
		return id.StreamID{}, errorfamily.WrapCorruption(err, errCode, msg)
	}
	return aggID, nil
}

// Hydrate rebuilds the users, emails and externalAccounts maps and the merge
// pointers from the SQL view store. It implements Hydrator; NewEventSourcedSetup
// calls it before the projection host starts when a CheckpointStore is
// configured. userIndex states how the rows are read back.
func (m *SQLUserReadModel) Hydrate(ctx context.Context) error {
	views, err := m.store.Scan(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.user_hydrate_scan", "scan user views")
	}

	index := newUserIndex(len(views))
	for _, view := range views {
		if err := index.add(view); err != nil {
			return err
		}
	}
	index.applyMerges()

	m.mu.Lock()
	m.users = index.users
	m.emails = index.emails
	m.externalAccounts = index.externalAccounts
	m.mergedInto = index.mergedInto
	m.mu.Unlock()

	return nil
}

// userIndex accumulates the in-memory maps of the user read model from view
// rows. Scan orders rows by key and a user's key is its ULID, so rows arrive
// in registration order: letting every row, live or merged, claim its email
// and external accounts keep-first rebuilds the ownership the event path had
// built (the 2026-10-04 live rule: the oldest registration owns the lookup).
// applyMerges then replays the merges in the order they happened.
type userIndex struct {
	users            map[id.StreamID]*User
	emails           map[string]id.StreamID
	externalAccounts map[externalAccountKey]id.StreamID
	mergedInto       map[id.StreamID]id.StreamID
	// live holds the live rows in row order; the claimant searches walk it.
	live   []liveUserRow
	merges []mergeRecord
}

// liveUserRow is one live row with the stream it is stored under.
type liveUserRow struct {
	aggID id.StreamID
	user  *User
}

// mergeRecord is one merged row as its tombstone describes it.
type mergeRecord struct {
	loser    id.StreamID
	survivor id.StreamID
	retired  *User
	mergedAt time.Time
}

func newUserIndex(size int) *userIndex {
	return &userIndex{
		users:            make(map[id.StreamID]*User, size),
		emails:           make(map[string]id.StreamID, size),
		externalAccounts: make(map[externalAccountKey]id.StreamID),
		mergedInto:       make(map[id.StreamID]id.StreamID),
		live:             make([]liveUserRow, 0, size),
		merges:           make([]mergeRecord, 0),
	}
}

// add folds one view row into the index.
func (ix *userIndex) add(view *UserView) error {
	user, err := unmarshalUserViewData(view.Data)
	if err != nil {
		return err
	}

	aggID, err := hydrateStreamID(
		user.ID.Get().String(),
		"usermgmt.sql_readmodel.user_hydrate_id", "parse user id from view",
	)
	if err != nil {
		return err
	}

	if !view.Tombstoned {
		ix.users[aggID] = user
		ix.live = append(ix.live, liveUserRow{aggID: aggID, user: user})
	} else if err := ix.addMerged(view, user, aggID); err != nil {
		return err
	}

	// A merged row claims like a live one: the entry then belongs to its
	// stream, which is what the merge re-points to the survivor.
	claimFirst(ix.emails, user.Email, aggID)
	for _, ea := range user.ExternalAccounts {
		claimFirst(ix.externalAccounts, externalAccountKey{provider: ea.Provider, subject: ea.Subject}, aggID)
	}
	return nil
}

// addMerged records the pointer of a tombstone. A tombstone without a pointer
// belongs to a deleted user, which the event path removed from every map.
func (ix *userIndex) addMerged(view *UserView, user *User, aggID id.StreamID) error {
	if user.MergedInto.IsZero() {
		return nil
	}
	survivor, err := aggIDFromUser(user.MergedInto)
	if err != nil {
		return err
	}
	mergedAt, err := time.Parse(time.RFC3339, view.UpdatedAt)
	if err != nil {
		return errorfamily.WrapCorruption(
			err, "usermgmt.sql_readmodel.user_hydrate_merged_at", "parse merge time from view").
			WithContextAny("agg_id", aggID)
	}
	ix.mergedInto[aggID] = survivor
	ix.merges = append(ix.merges, mergeRecord{
		loser: aggID, survivor: survivor, retired: user, mergedAt: mergedAt,
	})
	return nil
}

// claimFirst gives an index entry to its first owner and leaves it with that
// owner afterwards.
func claimFirst[K comparable](index map[K]id.StreamID, key K, owner id.StreamID) {
	if _, exists := index[key]; !exists {
		index[key] = owner
	}
}

// applyMerges settles every merge in the order the merges happened, so a
// chain settles hop by hop as the events did.
func (ix *userIndex) applyMerges() {
	slices.SortStableFunc(ix.merges, func(a, b mergeRecord) int {
		return a.mergedAt.Compare(b.mergedAt)
	})
	for _, merge := range ix.merges {
		ix.settle(merge)
	}
}

// settle replays one merge. While a live row ends the survivor chain, the
// entries the merged row owns move to its direct survivor, exactly as the
// event handler re-points them. An inherited external account moves only
// while that row still holds it: unlinking it on the survivor removed the
// lookup, and a restart must not bring the subject back. When no live row
// ends the chain the survivor was deleted and the event path evicted every
// entry it owned. A live row that has held the same email or account since
// after the merge is then the row that claimed the released entry.
func (ix *userIndex) settle(merge mergeRecord) {
	survivor := ix.liveEnd(merge.survivor)
	if survivor == nil {
		ix.release(merge)
		evict(ix.emails, merge.loser)
		evict(ix.externalAccounts, merge.loser)
		return
	}
	repoint(ix.emails, merge.loser, merge.survivor)
	for key, owner := range ix.externalAccounts {
		if owner != merge.loser {
			continue
		}
		if holdsExternalAccount(survivor, key) {
			ix.externalAccounts[key] = merge.survivor
		} else {
			delete(ix.externalAccounts, key)
		}
	}
}

// release hands the entries of a merged row on to a live row that claimed
// them after the merge, or drops them when none did.
func (ix *userIndex) release(merge mergeRecord) {
	if email := merge.retired.Email; ix.emails[email] == merge.loser {
		if claimant, ok := ix.emailClaimant(email, merge.mergedAt); ok {
			ix.emails[email] = claimant
		} else {
			delete(ix.emails, email)
		}
	}
	for _, ea := range merge.retired.ExternalAccounts {
		key := externalAccountKey{provider: ea.Provider, subject: ea.Subject}
		if ix.externalAccounts[key] != merge.loser {
			continue
		}
		if claimant, ok := ix.accountClaimant(key, merge.mergedAt); ok {
			ix.externalAccounts[key] = claimant
		} else {
			delete(ix.externalAccounts, key)
		}
	}
}

// emailClaimant is the first live row, in row order, that holds email and was
// registered after at.
func (ix *userIndex) emailClaimant(email string, at time.Time) (id.StreamID, bool) {
	for _, row := range ix.live {
		if row.user.Email == email && row.user.CreatedAt.After(at) {
			return row.aggID, true
		}
	}
	return id.StreamID{}, false
}

// accountClaimant is the first live row, in row order, that holds the account
// and linked it after at.
func (ix *userIndex) accountClaimant(key externalAccountKey, at time.Time) (id.StreamID, bool) {
	for _, row := range ix.live {
		for _, ea := range row.user.ExternalAccounts {
			if (externalAccountKey{provider: ea.Provider, subject: ea.Subject} == key) && ea.LinkedAt.After(at) {
				return row.aggID, true
			}
		}
	}
	return id.StreamID{}, false
}

// liveEnd follows the merge pointers from survivor and returns the live row
// at their end. A chain without a live row answers nil. The walk is bounded
// by the number of pointers, so corrupt rows cannot loop forever.
func (ix *userIndex) liveEnd(survivor id.StreamID) *User {
	for range len(ix.mergedInto) + 1 {
		if user, ok := ix.users[survivor]; ok {
			return user
		}
		next, merged := ix.mergedInto[survivor]
		if !merged {
			return nil
		}
		survivor = next
	}
	return nil
}

// holdsExternalAccount reports whether user still has the account linked.
func holdsExternalAccount(user *User, key externalAccountKey) bool {
	return slices.ContainsFunc(user.ExternalAccounts, func(ea ExternalAccount) bool {
		return externalAccountKey{provider: ea.Provider, subject: ea.Subject} == key
	})
}

// Hydrate rebuilds the memberships and byActor maps from the SQL view store.
// Membership keys are re-derived from the actor+tenant pair, exactly like the
// aggregate ID was derived when the events were written.
func (m *SQLMembershipReadModel) Hydrate(ctx context.Context) error {
	views, err := m.store.Scan(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.membership_hydrate_scan", "scan membership views")
	}

	memberships := make(map[id.StreamID]*Membership, len(views))
	byActor := make(map[string][]id.StreamID)

	for _, view := range views {
		mem, err := unmarshalViewJSON[Membership](
			view.Data, "usermgmt.sql_readmodel.membership_hydrate", "unmarshal membership view data")
		if err != nil {
			return err
		}

		aggID := deriveMembershipID(mem.ActorID, mem.TenantID)
		memberships[aggID] = &mem

		actorKey := mem.ActorID.String()
		if !slices.Contains(byActor[actorKey], aggID) {
			byActor[actorKey] = append(byActor[actorKey], aggID)
		}
	}

	m.mu.Lock()
	m.memberships = memberships
	m.byActor = byActor
	m.mu.Unlock()

	return nil
}

// Hydrate rebuilds the tenants map from the SQL view store. A tombstone row
// (merged away) is not live: its Data carries merged_into, so the pointer
// map keeps its trace and the live map stays without it.
func (m *SQLTenantReadModel) Hydrate(ctx context.Context) error {
	views, err := m.store.Scan(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.tenant_hydrate_scan", "scan tenant views")
	}

	tenants := make(map[id.StreamID]*Tenant, len(views))
	mergedInto := make(map[id.StreamID]id.StreamID)

	for _, view := range views {
		tenant, err := unmarshalViewJSON[Tenant](
			view.Data, "usermgmt.sql_readmodel.tenant_hydrate", "unmarshal tenant view data")
		if err != nil {
			return err
		}

		aggID, err := hydrateStreamID(
			tenant.ID.Get(),
			"usermgmt.sql_readmodel.tenant_hydrate_id", "parse tenant id from view",
		)
		if err != nil {
			return err
		}

		if !tenant.MergedInto.IsZero() {
			survivor, err := hydrateStreamID(
				tenant.MergedInto.Get(),
				"usermgmt.sql_readmodel.tenant_hydrate_merged_into", "parse survivor id from view",
			)
			if err != nil {
				return err
			}
			mergedInto[aggID] = survivor
			continue
		}

		tenants[aggID] = &tenant
	}

	m.mu.Lock()
	m.tenants = tenants
	m.mergedInto = mergedInto
	m.mu.Unlock()

	return nil
}

// Hydrate rebuilds the bots and byTokenHash maps from the SQL view store.
func (m *SQLBotReadModel) Hydrate(ctx context.Context) error {
	views, err := m.store.Scan(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.bot_hydrate_scan", "scan bot views")
	}

	bots := make(map[id.StreamID]*Bot, len(views))
	byTokenHash := make(map[string]*Bot, len(views))

	for _, view := range views {
		bot, err := unmarshalViewJSON[Bot](
			view.Data, "usermgmt.sql_readmodel.bot_hydrate", "unmarshal bot view data")
		if err != nil {
			return err
		}

		aggID, err := hydrateStreamID(
			bot.ID.Get(),
			"usermgmt.sql_readmodel.bot_hydrate_id", "parse bot id from view",
		)
		if err != nil {
			return err
		}

		bots[aggID] = &bot
		byTokenHash[string(bot.TokenHash)] = &bot
	}

	m.mu.Lock()
	m.bots = bots
	m.byTokenHash = byTokenHash
	m.mu.Unlock()

	return nil
}
