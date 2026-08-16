package usermgmt

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"slices"

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
	//nolint:wrapcheck // deliberate passthrough: this store implements the same interface contract
	return s.inner.Save(ctx, name, cp)
}

func (s *hydrationAwareCheckpointStore) Load(ctx context.Context, name string) (event.Checkpoint, error) {
	if !s.hydratable[name] {
		return event.Checkpoint{}, nil //nolint:exhaustruct // zero-value is the "no checkpoint, replay all" contract
	}
	//nolint:wrapcheck // deliberate passthrough: this store implements the same interface contract
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

// Hydrate rebuilds the users, emails, and externalAccounts maps from the SQL
// view store. It implements Hydrator; NewEventSourcedSetup calls it before
// the projection host starts when a CheckpointStore is configured.
func (m *SQLUserReadModel) Hydrate(ctx context.Context) error {
	views, err := m.store.Scan(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.user_hydrate_scan", "scan user views")
	}

	users := make(map[id.StreamID]*User, len(views))
	emails := make(map[string]id.StreamID, len(views))
	externalAccounts := make(map[externalAccountKey]id.StreamID)

	for _, view := range views {
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

		users[aggID] = user
		emails[user.Email] = aggID

		for _, ea := range user.ExternalAccounts {
			externalAccounts[externalAccountKey{provider: ea.Provider, subject: ea.Subject}] = aggID
		}
	}

	m.mu.Lock()
	m.users = users
	m.emails = emails
	m.externalAccounts = externalAccounts
	m.mu.Unlock()

	return nil
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

// Hydrate rebuilds the tenants map from the SQL view store.
func (m *SQLTenantReadModel) Hydrate(ctx context.Context) error {
	views, err := m.store.Scan(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.tenant_hydrate_scan", "scan tenant views")
	}

	tenants := make(map[id.StreamID]*Tenant, len(views))

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

		tenants[aggID] = &tenant
	}

	m.mu.Lock()
	m.tenants = tenants
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
