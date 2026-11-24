package usermgmt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// viewStoreCreator is the constructor signature shared by
// storage.NewSQLiteViewStore and storage.NewSQLViewStore.
type viewStoreCreator[V any, K fmt.Stringer] func(
	*sql.DB, storage.ViewMapper[V], ...storage.ViewStoreOption, //nolint:staticcheck // ADR-0123 v5
) (*storage.SQLViewStore[V, K], error) //nolint:staticcheck // ADR-0123 v5

// newViewStoreOrFail calls create to build a SQL view store and wraps any
// error as a Transient failure with the caller's error code and message.
// Shared by all New(NewSQLite|NewSQL)XReadModel constructor pairs so the
// error wrapping logic lives in exactly one place.
func newViewStoreOrFail[V any, K fmt.Stringer](
	create viewStoreCreator[V, K],
	db *sql.DB,
	mapper storage.ViewMapper[V], //nolint:staticcheck // ADR-0123 v5
	errCode, errMsg string,
) (*storage.SQLViewStore[V, K], error) { //nolint:staticcheck // ADR-0123 v5
	store, err := create(db, mapper)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, errCode, errMsg)
	}
	return store, nil
}

type UserView struct {
	Email         string `json:"email"          view:"email"`
	DisplayName   string `json:"display_name"   view:"display_name"`
	EmailVerified bool   `json:"email_verified" view:"email_verified"`
	TOTPEnabled   bool   `json:"totp_enabled"   view:"totp_enabled"`
	CreatedAt     string `json:"created_at"     view:"created_at"`
	UpdatedAt     string `json:"updated_at"     view:"updated_at"`
	Data          string `json:"data"           view:"data"`
	Tombstoned    bool   `json:"tombstoned"     view:"tombstoned"`
}

type SQLUserReadModel struct {
	*UserReadModel
	store   *storage.SQLViewStore[UserView, UserID] //nolint:staticcheck // ADR-0123 v5
	querier kv.ViewQuerier[UserView]
	counter kv.ViewCounter[UserView]
	// cache fronts the store on the FindByID path (write-through on the
	// projection sync path). Disabled by default; see [WithViewCacheCapacity].
	// kv.ViewStore since the kv v4.4.0 migration: the enabled path is a
	// kv.Cache, the disabled path is the raw store itself.
	cache kv.ViewStore[UserView, UserID]
}

func userViewMapper() storage.ViewMapper[UserView] { //nolint:staticcheck // ADR-0123 v5
	m := storage.AutoMapperWithTombstone[UserView]("users_view", "tombstoned") //nolint:staticcheck // ADR-0123 v5
	m.Indexes = []storage.IndexSpec{                                           //nolint:staticcheck // ADR-0123 v5
		{
			Name:    "idx_users_view_email",
			Columns: []string{csvColumnEmail},
		},
	}
	return m
}

func NewSQLiteUserReadModel(db *sql.DB, opts ...ViewCacheOption) (*SQLUserReadModel, error) {
	return buildSQLUserReadModel(
		db,
		storage.NewSQLiteViewStore[UserView, UserID], //nolint:staticcheck // ADR-0123 v5
		"create sqlite user view store",
		opts...,
	)
}

func NewSQLUserReadModel(db *sql.DB, opts ...ViewCacheOption) (*SQLUserReadModel, error) {
	return buildSQLUserReadModel(
		db,
		storage.NewSQLViewStore[UserView, UserID], //nolint:staticcheck // ADR-0123 v5
		"create sql user view store",
		opts...,
	)
}

// buildSQLUserReadModel constructs a SQLUserReadModel from a SQLite or generic
// SQL view store. The dialect is selected by passing the matching
// storage.New(NewSQLite|NewSQL)ViewStore as create.
func buildSQLUserReadModel(
	db *sql.DB,
	create viewStoreCreator[UserView, UserID],
	errMsg string,
	opts ...ViewCacheOption,
) (*SQLUserReadModel, error) {
	store, err := newViewStoreOrFail(
		create, db, userViewMapper(),
		"usermgmt.sql_readmodel.create", errMsg,
	)
	if err != nil {
		return nil, err
	}
	return newSQLUserReadModel(store, opts...)
}

func newSQLUserReadModel(
	store *storage.SQLViewStore[UserView, UserID], //nolint:staticcheck // ADR-0123 v5
	opts ...ViewCacheOption,
) (*SQLUserReadModel, error) {
	cache, err := newViewCache[UserView, UserID](store, opts...)
	if err != nil {
		return nil, err
	}
	return &SQLUserReadModel{
		UserReadModel: NewUserReadModel(), store: store, querier: store, counter: store, cache: cache,
	}, nil
}

// newSQLUserReadModelFromViewStore assembles the model over view-store
// interfaces so tests (and future non-SQL backends) can substitute fakes;
// the cache wraps the same interface, keeping the seam v5-neutral. The
// concrete store field stays nil here — Hydrate's Scan is a SQL-path-only
// concern and interface-built models never hydrate.
func newSQLUserReadModelFromViewStore(
	store kv.ViewStore[UserView, UserID],
	querier kv.ViewQuerier[UserView],
	counter kv.ViewCounter[UserView],
	opts ...ViewCacheOption,
) (*SQLUserReadModel, error) {
	cache, err := newViewCache[UserView, UserID](store, opts...)
	if err != nil {
		return nil, err
	}
	return &SQLUserReadModel{
		UserReadModel: NewUserReadModel(), querier: querier, counter: counter, cache: cache,
	}, nil
}

// Handle applies evt to the in-memory model and mirrors it into SQL. A
// UserMergedInto writes its soft tombstone first: the in-memory handler
// removes the row the tombstone is built from, and a failed write must leave
// memory untouched so the retry starts from the pre-merge state.
func (m *SQLUserReadModel) Handle(ctx context.Context, evt event.Event) error {
	aggID := evt.StreamID()
	if evt.Type() == eventUserMergedInto {
		if err := m.retireToSQL(ctx, aggID, evt); err != nil {
			return err
		}
		return m.UserReadModel.Handle(ctx, evt)
	}
	if err := m.UserReadModel.Handle(ctx, evt); err != nil {
		return err
	}
	// A retired stream owns no row. Its next event is one the journal holds
	// before the merge, delivered again after a hydrate; the pointer-following
	// FindByID would write the survivor under the loser's key.
	if m.Retired(aggID) {
		return nil
	}
	return m.syncToSQL(ctx, evt)
}

// retireToSQL writes the soft tombstone of the stream UserMergedInto retires.
// The row keeps the retired user's own snapshot with Tombstoned set, and its
// Data carries merged_into and the time of the merge, so a restart can
// rebuild the pointer and every lookup the survivor inherited (D4, D19).
func (m *SQLUserReadModel) retireToSQL(ctx context.Context, aggID id.StreamID, evt event.Event) error {
	p, err := decodePayload[identitymodel.UserMergedIntoPayload](evt, "UserMergedInto")
	if err != nil {
		return err
	}
	userID := NewUserID(aggID.Get())
	retired, ok := m.row(aggID)
	if !ok {
		retired, err = m.storedUser(ctx, userID)
		if err != nil {
			return err
		}
	}
	// nil answers a row that is already a tombstone: the event is delivered
	// again and the tombstone stays as it was first written.
	if retired == nil {
		return nil
	}
	retired.MergedInto = p.SurvivorID
	view, err := userViewOf(retired, evt.OccurredAt())
	if err != nil {
		return err
	}
	view.Tombstoned = true
	if err := m.cache.Set(ctx, userID, view); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.retire", "write merged user tombstone").
			WithContextAny("user_id", userID)
	}
	return nil
}

// storedUser reads the row the store holds for a user that memory has no row
// for. No row answers a user that carries nothing but its ID: a merge pointer
// without a tombstone would be lost at the next restart.
func (m *SQLUserReadModel) storedUser(ctx context.Context, userID UserID) (*User, error) {
	view, err := m.cache.Get(ctx, userID)
	if errors.Is(err, kv.ErrNotFound) {
		return &User{ID: userID}, nil //nolint:exhaustruct // a pointer-only tombstone: every other field is unknown
	}
	if err != nil {
		return nil, errorfamily.WrapTransient(
			err, "usermgmt.sql_readmodel.retire_get", "read user view before writing its tombstone").
			WithContextAny("user_id", userID)
	}
	if view.Tombstoned {
		return nil, nil
	}
	user, err := unmarshalUserViewData(view.Data)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (m *SQLUserReadModel) syncToSQL(ctx context.Context, evt event.Event) error {
	aggID := evt.StreamID()
	if evt.Type() == eventUserDeleted {
		userID := NewUserID(aggID.Get())
		if err := m.cache.Delete(ctx, userID); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.delete", "delete user view").
				WithContextAny("user_id", userID)
		}
		return nil
	}
	user, ok := m.FindByID(aggID)
	if !ok {
		return nil
	}
	view, err := userViewOf(user, user.UpdatedAt)
	if err != nil {
		return err
	}
	userID := NewUserID(aggID.Get())
	if err := m.cache.Set(ctx, userID, view); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.upsert", "upsert user view").
			WithContextAny("user_id", userID)
	}
	return nil
}

// userViewOf is the row stored for a user; at is the time of its last change.
func userViewOf(user *User, at time.Time) (*UserView, error) {
	data, err := marshalUserViewData(user)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "usermgmt.sql_readmodel.user_marshal", "marshal user view data").
			WithContextAny("user_id", user.ID)
	}
	return &UserView{ //nolint:exhaustruct // a live row is never tombstoned; the zero value is the contract
		Email: user.Email, DisplayName: user.DisplayName,
		EmailVerified: user.EmailVerified, TOTPEnabled: user.TOTPEnabled,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: at.Format(time.RFC3339),
		Data:      data,
	}, nil
}

func (m *SQLUserReadModel) FindByIDSQL(ctx context.Context, userID UserID) (*UserView, error) {
	view, err := m.cache.Get(ctx, userID)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.get", "get user view by id").
			WithContextAny("user_id", userID)
	}
	return view, nil
}

// FindByEmailSQL lists the live rows that hold an email. A merged row is not
// a user any more: the live row at the end of its merge chain answers in its
// place, once, so the result never carries a tombstone (D21). Rows that are
// live duplicates of each other are all returned, as before.
func (m *SQLUserReadModel) FindByEmailSQL(ctx context.Context, email string) ([]*UserView, error) {
	rows, err := m.querier.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: csvColumnEmail, Op: kv.OpEq, Value: email}},
	})
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.query_email", "query user view by email").
			WithContext("email", email)
	}
	live := make([]*UserView, 0, len(rows))
	seen := make(map[UserID]struct{}, len(rows))
	for _, row := range rows {
		if row.Tombstoned {
			if row, err = m.liveAfterMerge(ctx, row); err != nil {
				return nil, err
			}
			if row == nil {
				continue
			}
		}
		rowUserID, err := viewUserID(row)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[rowUserID]; dup {
			continue
		}
		seen[rowUserID] = struct{}{}
		live = append(live, row)
	}
	return live, nil
}

// sqlMergeChainLimit bounds how many pointers one lookup over stored rows
// follows. A chain is one pointer per merged user; the bound only exists so
// corrupt rows cannot loop forever.
const sqlMergeChainLimit = 10000

// liveAfterMerge follows the merge pointer of a tombstone to the live row at
// end of its chain. A tombstone without a pointer, and a chain without a
// live row, answer nil.
func (m *SQLUserReadModel) liveAfterMerge(ctx context.Context, tombstone *UserView) (*UserView, error) {
	row := tombstone
	for range sqlMergeChainLimit {
		user, err := unmarshalUserViewData(row.Data)
		if err != nil {
			return nil, err
		}
		if user.MergedInto.IsZero() {
			return nil, nil
		}
		survivor, err := aggIDFromUser(user.MergedInto)
		if err != nil {
			return nil, err
		}
		next, err := m.cache.Get(ctx, NewUserID(survivor.Get()))
		if errors.Is(err, kv.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, errorfamily.WrapTransient(
				err, "usermgmt.sql_readmodel.query_email", "read the survivor of a merged user view").
				WithContextAny("user_id", user.MergedInto)
		}
		if !next.Tombstoned {
			return next, nil
		}
		row = next
	}
	return nil, nil
}

// viewUserID reads the user a stored row belongs to. The row carries no key
// column, so the ID inside Data is the source of truth.
func viewUserID(view *UserView) (UserID, error) {
	user, err := unmarshalUserViewData(view.Data)
	if err != nil {
		return UserID{}, err
	}
	return user.ID, nil
}

func (m *SQLUserReadModel) CountSQL(ctx context.Context) (int64, error) {
	count, err := m.counter.Count(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "tombstoned", Op: kv.OpEq, Value: false}},
	})
	if err != nil {
		return 0, errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.count", "count users")
	}
	return count, nil
}

var _ projection.Projection = (*SQLUserReadModel)(nil)
