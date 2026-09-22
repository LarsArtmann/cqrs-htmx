package usermgmt

import (
	"context"
	"database/sql"
	"fmt"

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
	ActorID string `json:"actor_id" view:"actor_id"`
	//cqrs-lint:ignore(A032) SQL view DTO: string ID for JSON serialization and DB scan compatibility
	TenantID string `json:"tenant_id" view:"tenant_id"`
	Data     string `json:"data"      view:"data"`
}

type SQLMembershipReadModel struct {
	*MembershipReadModel
	store   *storage.SQLViewStore[MembershipView, id.StreamID] //nolint:staticcheck // ADR-0123 v5
	querier kv.ViewQuerier[MembershipView]
}

func membershipViewMapper() storage.ViewMapper[MembershipView] { //nolint:staticcheck // ADR-0123 v5
	m := storage.AutoMapper[MembershipView]("memberships_view") //nolint:staticcheck // ADR-0123 v5

	m.Indexes = []storage.IndexSpec{ //nolint:staticcheck // ADR-0123 v5
		{Name: "idx_memberships_view_actor", Columns: []string{"actor_id"}},
	}
	return m
}

func NewSQLiteMembershipReadModel(db *sql.DB) (*SQLMembershipReadModel, error) {
	return buildSQLMembershipReadModel(
		db,
		storage.NewSQLiteViewStore[MembershipView, id.StreamID], //nolint:staticcheck // ADR-0123 v5
		"create sqlite membership view store",
	)
}

func NewSQLMembershipReadModel(db *sql.DB) (*SQLMembershipReadModel, error) {
	return buildSQLMembershipReadModel(
		db,
		storage.NewSQLViewStore[MembershipView, id.StreamID], //nolint:staticcheck // ADR-0123 v5
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
	deleted, err := deleteViewOnTombstone(ctx, m.store, evt, eventMemberRemoved, aggID,
		"usermgmt.sql_readmodel.membership_delete", "delete membership view")
	if deleted {
		return err
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
	return upsertView(ctx, m.store, aggID, view,
		"usermgmt.sql_readmodel.membership_upsert", "upsert membership view")
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
	store   *storage.SQLViewStore[TenantView, TenantID] //nolint:staticcheck // ADR-0123 v5
	querier kv.ViewQuerier[TenantView]
}

func tenantViewMapper() storage.ViewMapper[TenantView] { //nolint:staticcheck // ADR-0123 v5
	m := storage.AutoMapper[TenantView]("tenants_view") //nolint:staticcheck // ADR-0123 v5

	m.Indexes = []storage.IndexSpec{ //nolint:staticcheck // ADR-0123 v5
		{Name: "idx_tenants_view_name", Columns: []string{sqlColName}},
	}
	return m
}

func NewSQLiteTenantReadModel(db *sql.DB) (*SQLTenantReadModel, error) {
	return buildSQLTenantReadModel(
		db,
		storage.NewSQLiteViewStore[TenantView, TenantID], //nolint:staticcheck // ADR-0123 v5
		"create sqlite tenant view store",
	)
}

func NewSQLTenantReadModel(db *sql.DB) (*SQLTenantReadModel, error) {
	return buildSQLTenantReadModel(
		db,
		storage.NewSQLViewStore[TenantView, TenantID], //nolint:staticcheck // ADR-0123 v5
		"create sql tenant view store",
	)
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
	deleted, err := deleteViewOnTombstone(ctx, m.store, evt, eventTenantDeleted, tid,
		"usermgmt.sql_readmodel.tenant_delete", "delete tenant view")
	if deleted {
		return err
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
	return upsertView(ctx, m.store, tid, view,
		"usermgmt.sql_readmodel.tenant_upsert", "upsert tenant view")
}

func (m *SQLTenantReadModel) FindByNameSQL(ctx context.Context, name string) ([]*TenantView, error) {
	return queryViewByName(ctx, m.querier, name, "usermgmt.sql_readmodel.tenant_query", "query tenant by name")
}

var _ projection.Projection = (*SQLTenantReadModel)(nil)

// --- Bot ---

type BotView struct {
	Name string `json:"name" view:"name"`
	//cqrs-lint:ignore(A032) SQL view DTO: string ID for JSON serialization and DB scan compatibility
	OwnerID   string `json:"owner_id"   view:"owner_id"`
	TokenHash string `json:"token_hash" view:"token_hash"`
	Deleted   bool   `json:"deleted"    view:"deleted"`
	Data      string `json:"data"       view:"data"`
}

type SQLBotReadModel struct {
	*BotReadModel
	store   *storage.SQLViewStore[BotView, BotID] //nolint:staticcheck // ADR-0123 v5
	querier kv.ViewQuerier[BotView]
}

func botViewMapper() storage.ViewMapper[BotView] { //nolint:staticcheck // ADR-0123 v5
	m := storage.AutoMapper[BotView]("bots_view") //nolint:staticcheck // ADR-0123 v5

	m.Indexes = []storage.IndexSpec{ //nolint:staticcheck // ADR-0123 v5
		{Name: "idx_bots_view_name", Columns: []string{sqlColName}},
	}
	return m
}

func NewSQLiteBotReadModel(db *sql.DB) (*SQLBotReadModel, error) {
	return buildSQLBotReadModel(
		db,
		storage.NewSQLiteViewStore[BotView, BotID], //nolint:staticcheck // ADR-0123 v5
		"create sqlite bot view store",
	)
}

func NewSQLBotReadModel(db *sql.DB) (*SQLBotReadModel, error) {
	return buildSQLBotReadModel(
		db,
		storage.NewSQLViewStore[BotView, BotID], //nolint:staticcheck // ADR-0123 v5
		"create sql bot view store",
	)
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
	deleted, err := deleteViewOnTombstone(ctx, m.store, evt, eventBotDeleted, bid,
		"usermgmt.sql_readmodel.bot_delete", "delete bot view")
	if deleted {
		return err
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
	return upsertView(ctx, m.store, bid, view,
		"usermgmt.sql_readmodel.bot_upsert", "upsert bot view")
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

// deleteViewOnTombstone removes a view row when evt carries the aggregate's
// tombstone event, wrapping any store failure as a Transient error with the
// caller's error code and human-readable message. It reports whether evt was
// the tombstone; when true the Handle caller returns immediately. Shared by
// the per-aggregate Handle methods whose only differences are the tombstone
// event, the view key, and the error tags.
func deleteViewOnTombstone[V any, K fmt.Stringer](
	ctx context.Context,
	store *storage.SQLViewStore[V, K], //nolint:staticcheck // ADR-0123 v5
	evt event.Event,
	tombstone event.Type,
	key K,
	errCode, errMsg string,
) (bool, error) {
	if evt.Type() != tombstone {
		return false, nil
	}
	if err := store.Delete(ctx, key); err != nil {
		return true, errorfamily.WrapTransient(err, errCode, errMsg)
	}
	return true, nil
}

// upsertView persists a view row, wrapping any store failure as a Transient
// error with the caller's error code and human-readable message. Shared by
// the per-aggregate Handle methods whose only differences are the view type
// and the error tags.
func upsertView[V any, K fmt.Stringer](
	ctx context.Context,
	store *storage.SQLViewStore[V, K], //nolint:staticcheck // ADR-0123 v5
	key K,
	view V,
	errCode, errMsg string,
) error {
	if err := store.Set(ctx, key, &view); err != nil {
		return errorfamily.WrapTransient(err, errCode, errMsg)
	}
	return nil
}
