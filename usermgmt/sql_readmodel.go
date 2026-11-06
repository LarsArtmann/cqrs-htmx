package usermgmt

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
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

func (m *SQLUserReadModel) Handle(ctx context.Context, evt event.Event) error {
	if err := m.UserReadModel.Handle(ctx, evt); err != nil {
		return err
	}
	return m.syncToSQL(ctx, evt)
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
	data, err := marshalUserViewData(user)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.sql_readmodel.user_marshal", "marshal user view data").
			WithContextAny("agg_id", aggID)
	}
	view := UserView{
		Email: user.Email, DisplayName: user.DisplayName,
		EmailVerified: user.EmailVerified, TOTPEnabled: user.TOTPEnabled,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		Data:      data,
	}
	userID := NewUserID(aggID.Get())
	if err := m.cache.Set(ctx, userID, &view); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.upsert", "upsert user view").
			WithContextAny("user_id", userID)
	}
	return nil
}

func (m *SQLUserReadModel) FindByIDSQL(ctx context.Context, userID UserID) (*UserView, error) {
	view, err := m.cache.Get(ctx, userID)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.get", "get user view by id").
			WithContextAny("user_id", userID)
	}
	return view, nil
}

func (m *SQLUserReadModel) FindByEmailSQL(ctx context.Context, email string) ([]*UserView, error) {
	views, err := m.querier.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: csvColumnEmail, Op: kv.OpEq, Value: email}},
	})
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.query_email", "query user view by email").
			WithContext("email", email)
	}
	return views, nil
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
