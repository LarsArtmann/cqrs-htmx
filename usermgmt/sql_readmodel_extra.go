package usermgmt

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const sqlColName = "name"

// --- Membership ---

type MembershipView struct {
	//cqrs-lint:ignore(A032) SQL view DTO: string ID for JSON serialization and DB scan compatibility
	ActorID string `json:"actor_id"  view:"actor_id"`
	//cqrs-lint:ignore(A032) SQL view DTO: string ID for JSON serialization and DB scan compatibility
	TenantID string `json:"tenant_id" view:"tenant_id"`
	Data     string `json:"data"      view:"data"`
}

type SQLMembershipReadModel struct {
	*MembershipReadModel
	store   *storage.SQLViewStore[MembershipView, id.StreamID]
	querier kv.ViewQuerier[MembershipView]
}

func membershipViewMapper() storage.ViewMapper[MembershipView] {
	m := storage.AutoMapper[MembershipView]("memberships_view")

	m.Indexes = []storage.IndexSpec{{Name: "idx_memberships_view_actor", Columns: []string{"actor_id"}}}
	return m
}

func NewSQLiteMembershipReadModel(db *sql.DB) (*SQLMembershipReadModel, error) {
	return buildSQLMembershipReadModel(
		db,
		storage.NewSQLiteViewStore[MembershipView, id.StreamID],
		"create sqlite membership view store",
	)
}

func NewSQLMembershipReadModel(db *sql.DB) (*SQLMembershipReadModel, error) {
	return buildSQLMembershipReadModel(
		db,
		storage.NewSQLViewStore[MembershipView, id.StreamID],
		"create sql membership view store",
	)
}

func buildSQLMembershipReadModel(
	db *sql.DB,
	create viewStoreCreator[MembershipView, id.StreamID],
	errMsg string,
) (*SQLMembershipReadModel, error) {
	store, err := newViewStoreOrFail(
		create, db, membershipViewMapper(),
		"usermgmt.sql_readmodel.membership_create", errMsg,
	)
	if err != nil {
		return nil, err
	}
	return &SQLMembershipReadModel{MembershipReadModel: NewMembershipReadModel(), store: store, querier: store}, nil
}

func (m *SQLMembershipReadModel) Handle(ctx context.Context, evt event.Event) error {
	if err := m.MembershipReadModel.Handle(ctx, evt); err != nil {
		return err
	}
	aggID := evt.StreamID()
	if evt.Type() == eventMemberRemoved {
		if err := m.store.Delete(ctx, aggID); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.membership_delete", "delete membership view")
		}
		return nil
	}
	mem, ok := m.FindByAggregateID(aggID)
	if !ok {
		return nil
	}
	data, err := marshalViewJSON(mem, "usermgmt.sql_readmodel.membership_marshal", "marshal membership data")
	if err != nil {
		return err
	}
	view := MembershipView{ActorID: mem.ActorID.String(), TenantID: mem.TenantID.Get(), Data: data}
	if err := m.store.Set(ctx, aggID, &view); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.membership_upsert", "upsert membership view")
	}
	return nil
}

func (m *SQLMembershipReadModel) FindByActorSQL(ctx context.Context, actorID string) ([]*MembershipView, error) {
	views, err := m.querier.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "actor_id", Op: kv.OpEq, Value: actorID}},
	})
	if err != nil {
		return nil, errorfamily.WrapTransient(
			err,
			"usermgmt.sql_readmodel.membership_query",
			"query membership by actor",
		)
	}
	return views, nil
}

var _ projection.Projection = (*SQLMembershipReadModel)(nil)

// --- Tenant ---

type TenantView struct {
	Name        string `json:"name"         view:"name"`
	DisplayName string `json:"display_name" view:"display_name"`
	Suspended   bool   `json:"suspended"    view:"suspended"`
	Deleted     bool   `json:"deleted"      view:"deleted"`
	Data        string `json:"data"         view:"data"`
}

type SQLTenantReadModel struct {
	*TenantReadModel
	store   *storage.SQLViewStore[TenantView, TenantID]
	querier kv.ViewQuerier[TenantView]
}

func tenantViewMapper() storage.ViewMapper[TenantView] {
	m := storage.AutoMapper[TenantView]("tenants_view")

	m.Indexes = []storage.IndexSpec{{Name: "idx_tenants_view_name", Columns: []string{sqlColName}}}
	return m
}

func NewSQLiteTenantReadModel(db *sql.DB) (*SQLTenantReadModel, error) {
	return buildSQLTenantReadModel(
		db,
		storage.NewSQLiteViewStore[TenantView, TenantID],
		"create sqlite tenant view store",
	)
}

func NewSQLTenantReadModel(db *sql.DB) (*SQLTenantReadModel, error) {
	return buildSQLTenantReadModel(db, storage.NewSQLViewStore[TenantView, TenantID], "create sql tenant view store")
}

func buildSQLTenantReadModel(
	db *sql.DB,
	create viewStoreCreator[TenantView, TenantID],
	errMsg string,
) (*SQLTenantReadModel, error) {
	store, err := newViewStoreOrFail(
		create, db, tenantViewMapper(),
		"usermgmt.sql_readmodel.tenant_create", errMsg,
	)
	if err != nil {
		return nil, err
	}
	return &SQLTenantReadModel{TenantReadModel: NewTenantReadModel(), store: store, querier: store}, nil
}

func (m *SQLTenantReadModel) Handle(ctx context.Context, evt event.Event) error {
	if err := m.TenantReadModel.Handle(ctx, evt); err != nil {
		return err
	}
	aggID := evt.StreamID()
	tid := NewTenantID(aggID.String())
	if evt.Type() == eventTenantDeleted {
		if err := m.store.Delete(ctx, tid); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.tenant_delete", "delete tenant view")
		}
		return nil
	}
	tenant, ok := m.FindByID(aggID)
	if !ok {
		return nil
	}
	data, err := marshalViewJSON(tenant, "usermgmt.sql_readmodel.tenant_marshal", "marshal tenant data")
	if err != nil {
		return err
	}
	view := TenantView{
		Name: tenant.Name, DisplayName: tenant.DisplayName,
		Suspended: tenant.Suspended, Deleted: tenant.Deleted, Data: data,
	}
	if err := m.store.Set(ctx, tid, &view); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.tenant_upsert", "upsert tenant view")
	}
	return nil
}

func (m *SQLTenantReadModel) FindByNameSQL(ctx context.Context, name string) ([]*TenantView, error) {
	return queryViewByName(ctx, m.querier, name, "usermgmt.sql_readmodel.tenant_query", "query tenant by name")
}

var _ projection.Projection = (*SQLTenantReadModel)(nil)

// --- Bot ---

type BotView struct {
	Name string `json:"name"       view:"name"`
	//cqrs-lint:ignore(A032) SQL view DTO: string ID for JSON serialization and DB scan compatibility
	OwnerID   string `json:"owner_id"   view:"owner_id"`
	TokenHash string `json:"token_hash" view:"token_hash"`
	Deleted   bool   `json:"deleted"    view:"deleted"`
	Data      string `json:"data"      view:"data"`
}

type SQLBotReadModel struct {
	*BotReadModel
	store   *storage.SQLViewStore[BotView, BotID]
	querier kv.ViewQuerier[BotView]
}

func botViewMapper() storage.ViewMapper[BotView] {
	m := storage.AutoMapper[BotView]("bots_view")

	m.Indexes = []storage.IndexSpec{{Name: "idx_bots_view_name", Columns: []string{sqlColName}}}
	return m
}

func NewSQLiteBotReadModel(db *sql.DB) (*SQLBotReadModel, error) {
	return buildSQLBotReadModel(db, storage.NewSQLiteViewStore[BotView, BotID], "create sqlite bot view store")
}

func NewSQLBotReadModel(db *sql.DB) (*SQLBotReadModel, error) {
	return buildSQLBotReadModel(db, storage.NewSQLViewStore[BotView, BotID], "create sql bot view store")
}

func buildSQLBotReadModel(
	db *sql.DB,
	create viewStoreCreator[BotView, BotID],
	errMsg string,
) (*SQLBotReadModel, error) {
	store, err := newViewStoreOrFail(
		create, db, botViewMapper(),
		"usermgmt.sql_readmodel.bot_create", errMsg,
	)
	if err != nil {
		return nil, err
	}
	return &SQLBotReadModel{BotReadModel: NewBotReadModel(), store: store, querier: store}, nil
}

func (m *SQLBotReadModel) Handle(ctx context.Context, evt event.Event) error {
	if err := m.BotReadModel.Handle(ctx, evt); err != nil {
		return err
	}
	aggID := evt.StreamID()
	bid := NewBotID(aggID.String())
	if evt.Type() == eventBotDeleted {
		if err := m.store.Delete(ctx, bid); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.bot_delete", "delete bot view")
		}
		return nil
	}
	bot, ok := m.FindByID(aggID)
	if !ok {
		return nil
	}
	data, err := marshalViewJSON(bot, "usermgmt.sql_readmodel.bot_marshal", "marshal bot data")
	if err != nil {
		return err
	}
	view := BotView{
		Name: bot.Name, OwnerID: bot.OwnerID.Get().String(),
		TokenHash: string(bot.TokenHash), Deleted: bot.Deleted, Data: data,
	}
	if err := m.store.Set(ctx, bid, &view); err != nil {
		return errorfamily.WrapTransient(err, "usermgmt.sql_readmodel.bot_upsert", "upsert bot view")
	}
	return nil
}

func (m *SQLBotReadModel) FindByNameSQL(ctx context.Context, name string) ([]*BotView, error) {
	return queryViewByName(ctx, m.querier, name, "usermgmt.sql_readmodel.bot_query", "query bot by name")
}

var _ projection.Projection = (*SQLBotReadModel)(nil)

// queryViewByName runs a single-column equality lookup against a view store
// and wraps any store error as a Transient failure with the caller's error
// code and human-readable message. Shared by the per-aggregate FindByNameSQL
// methods whose only differences are the view type and the error tags.
func queryViewByName[T any](
	ctx context.Context,
	q kv.ViewQuerier[T],
	name, errCode, errMsg string,
) ([]*T, error) {
	views, err := q.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: sqlColName, Op: kv.OpEq, Value: name}},
	})
	if err != nil {
		return nil, errorfamily.WrapTransient(err, errCode, errMsg)
	}
	return views, nil
}
