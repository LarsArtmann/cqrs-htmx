package usermgmt

import (
	"context"
	"database/sql"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

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
	store   *storage.SQLViewStore[UserView, UserID]
	querier kv.ViewQuerier[UserView]
	counter kv.ViewCounter[UserView]
}

func userViewMapper() storage.ViewMapper[UserView] {
	m := storage.AutoMapperWithTombstone[UserView]("users_view", "tombstoned")
	m.Indexes = []storage.IndexSpec{
		{Name: "idx_users_view_email", Columns: []string{csvColumnEmail}},
	}
	return m
}

func NewSQLiteUserReadModel(db *sql.DB) (*SQLUserReadModel, error) {
	store, err := storage.NewSQLiteViewStore[UserView, UserID](db, userViewMapper())
	if err != nil {
		return nil, event.WrapTransient(err, "usermgmt.sql_readmodel.create", "create sqlite user view store")
	}
	return newSQLUserReadModel(store), nil
}

func NewSQLUserReadModel(db *sql.DB) (*SQLUserReadModel, error) {
	store, err := storage.NewSQLViewStore[UserView, UserID](db, userViewMapper())
	if err != nil {
		return nil, event.WrapTransient(err, "usermgmt.sql_readmodel.create", "create sql user view store")
	}
	return newSQLUserReadModel(store), nil
}

func newSQLUserReadModel(store *storage.SQLViewStore[UserView, UserID]) *SQLUserReadModel {
	return &SQLUserReadModel{UserReadModel: NewUserReadModel(), store: store, querier: store, counter: store}
}

func (m *SQLUserReadModel) Handle(ctx context.Context, evt event.Event) error {
	if err := m.UserReadModel.Handle(ctx, evt); err != nil {
		return err
	}
	return m.syncToSQL(ctx, evt)
}

func (m *SQLUserReadModel) syncToSQL(ctx context.Context, evt event.Event) error {
	aggID := evt.AggregateID()
	if evt.Type() == eventUserDeleted {
		userID := NewUserID(aggID.String())
		if err := m.store.Delete(ctx, userID); err != nil {
			return event.WrapTransient(err, "usermgmt.sql_readmodel.delete", "delete user view")
		}
		return nil
	}
	user, ok := m.FindByID(aggID)
	if !ok {
		return nil
	}
	data, err := marshalViewJSON(user, "usermgmt.sql_readmodel.marshal", "marshal user data")
	if err != nil {
		return err
	}
	view := UserView{
		Email: user.Email, DisplayName: user.DisplayName,
		EmailVerified: user.EmailVerified, TOTPEnabled: user.TOTPEnabled,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		Data:      data,
	}
	userID := NewUserID(aggID.String())
	if err := m.store.Set(ctx, userID, &view); err != nil {
		return event.WrapTransient(err, "usermgmt.sql_readmodel.upsert", "upsert user view")
	}
	return nil
}

func (m *SQLUserReadModel) FindByIDSQL(ctx context.Context, userID UserID) (*UserView, error) {
	view, err := m.store.Get(ctx, userID)
	if err != nil {
		return nil, event.WrapTransient(err, "usermgmt.sql_readmodel.get", "get user view by id")
	}
	return view, nil
}

func (m *SQLUserReadModel) FindByEmailSQL(ctx context.Context, email string) ([]*UserView, error) {
	views, err := m.querier.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: csvColumnEmail, Op: kv.OpEq, Value: email}},
	})
	if err != nil {
		return nil, event.WrapTransient(err, "usermgmt.sql_readmodel.query_email", "query user view by email")
	}
	return views, nil
}

func (m *SQLUserReadModel) CountSQL(ctx context.Context) (int64, error) {
	count, err := m.counter.Count(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "tombstoned", Op: kv.OpEq, Value: false}},
	})
	if err != nil {
		return 0, event.WrapTransient(err, "usermgmt.sql_readmodel.count", "count users")
	}
	return count, nil
}

var _ event.Projection = (*SQLUserReadModel)(nil)
