package integration

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	dbpkg "github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func setupAtomicUpdateRouter(
	t *testing.T,
	sqlDB *sql.DB,
	broker module.RealtimeBroker,
	operation actions.AtomicUpdateActionOperation,
	scopeCheck func(*gin.Context, module.RelationScope) error,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	id := pg.IntegerColumn("id")
	ownerID := pg.IntegerColumn("owner_id")
	title := pg.StringColumn("title")
	chatID := pg.IntegerColumn("chat_id")
	recipientUserID := pg.IntegerColumn("recipient_user_id")
	items := pg.NewTable("public", "atomic_update_items", "", id, ownerID, title, chatID, recipientUserID)
	owners := pg.NewTable("public", "atomic_update_owners", "", id)

	itemsModule := &module.BaseModule{
		Name:       "atomic-update-items",
		Path:       "/admin",
		Table:      items,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
			{Column: ownerID, Title: "Owner", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
			{Column: chatID, Title: "Chat", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: recipientUserID, Title: "Recipient", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
		},
		Relations: []module.ModuleRelation{{
			Name:         "owner",
			TargetModule: "atomic-update-owners",
			SourceField:  ownerID,
			TargetField:  id,
			ScopeCheck:   scopeCheck,
		}},
		RoleWhere: []actions.RoleWhere{{
			Role: actions.RoleAll,
			Where: func(*gin.Context) pg.BoolExpression {
				return ownerID.EQ(pg.Int(7))
			},
		}},
		Actions: []actions.ModuleAction{actions.UpdateModuleAction{
			Mode:       actions.UpdateModeAtomic,
			Columns:    []pg.Column{title},
			By:         []pg.Column{id},
			Permission: []actions.Role{actions.RoleAll},
			Auth:       true,
			Realtime:   &actions.RealtimeEventConfig{CorrelationField: "chat_id"},
			Atomic: &actions.AtomicUpdateConfig{
				Operation: operation,
				ResultFields: []actions.AtomicResultField{
					{Name: "chat_id", Kind: actions.AtomicValueKindInt},
					{Name: "recipient_user_id", Kind: actions.AtomicValueKindInt},
				},
				Publish: []actions.AtomicRealtimePublishConfig{{
					Recipients:  []actions.AtomicRealtimeRecipient{{UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "recipient_user_id"}}},
					Correlation: &actions.AtomicRealtimeCorrelation{Field: "chat_id", Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"}},
				}},
			},
		}},
	}
	ownersModule := &module.BaseModule{
		Name:       "atomic-update-owners",
		Path:       "/admin",
		Table:      owners,
		PrimaryKey: id,
		Fields:     []fields.ModuleField{{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber}},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(*module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) },
		*group,
		[]*module.BaseModule{itemsModule, ownersModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 7, Role: "admin"}),
	)
	generator.Realtime = module.RealtimeConfig{Enabled: true, Broker: broker}
	generator.Run()
	return engine
}

func TestAtomicUpdateCommitsTypedSelectorAndPublishesRealtime(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	items := pg.NewTable("public", "atomic_update_items", "", id, title)
	counterID := pg.IntegerColumn("id")
	unreadCount := pg.IntegerColumn("unread_count")
	counters := pg.NewTable("public", "atomic_update_counters", "", counterID, unreadCount)
	broker := module.NewMemoryBroker(module.MemoryBrokerOptions{MaxEvents: 10})

	var received actions.AtomicUpdateInput
	operation := func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicUpdateInput) (actions.AtomicRecord, error) {
		received = input
		recordID, ok := input.Selector.Value.Int, input.Selector.ByKey == "id"
		if !ok || recordID == nil {
			return actions.AtomicRecord{}, errors.New("typed selector is unavailable")
		}
		titleValue, err := input.Input.RequireString("title")
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		if _, err := executor.Update(ctx, actions.AtomicUpdate{
			Table:  items,
			Fields: []actions.AtomicUpdateField{{Column: title, Operation: actions.AtomicUpdateSet, Value: actions.AtomicString(titleValue)}},
			Where:  id.EQ(pg.Int(*recordID)),
		}); err != nil {
			return actions.AtomicRecord{}, err
		}
		if _, err := executor.Update(ctx, actions.AtomicUpdate{
			Table:  counters,
			Fields: []actions.AtomicUpdateField{{Column: unreadCount, Operation: actions.AtomicUpdateIncrement, Value: actions.AtomicInt(1)}},
			Where:  counterID.EQ(pg.Int(55)),
		}); err != nil {
			return actions.AtomicRecord{}, err
		}
		return actions.AtomicRecord{
			Value:      *recordID,
			PrimaryKey: "id",
			Fields: []actions.AtomicField{
				{Name: "chat_id", Value: actions.AtomicInt(42)},
				{Name: "recipient_user_id", Value: actions.AtomicInt(19)},
			},
		}, nil
	}
	engine := setupAtomicUpdateRouter(t, sqlDB, broker, operation, func(*gin.Context, module.RelationScope) error { return nil })

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WithArgs(int64(41), int64(7), int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public.atomic_update_items SET title = $1::text WHERE atomic_update_items.id = $2;`)).WithArgs("updated", int64(41)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE public\.atomic_update_counters`).WithArgs(int64(1), int64(55)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	w := executeJSONRequest(engine, http.MethodPost, "/admin/atomic-update-items/id/41", map[string]interface{}{"title": "updated"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"value":41,"primary_key":"id","chat_id":42,"recipient_user_id":19}`, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, "id", received.Selector.ByKey)
	require.NotNil(t, received.Selector.Value.Int)
	require.Equal(t, int64(41), *received.Selector.Value.Int)
	require.Equal(t, "updated", mustAtomicInputString(t, received.Input, "title"))

	events, resync, err := broker.Replay(context.Background(), "0", 10)
	require.NoError(t, err)
	require.False(t, resync)
	require.Len(t, events, 1)
	require.Equal(t, "atomic-update-items", events[0].Module)
	require.Equal(t, "update", events[0].Action)
	require.Equal(t, []string{"user:19"}, events[0].Topics)
	require.NotNil(t, events[0].Correlation)
	require.Equal(t, "chat_id", events[0].Correlation.Field)
}

func TestAtomicUpdateRollsBackAndDoesNotPublishRealtime(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	items := pg.NewTable("public", "atomic_update_items", "", id, title)
	broker := module.NewMemoryBroker(module.MemoryBrokerOptions{MaxEvents: 10})
	operation := func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicUpdateInput) (actions.AtomicRecord, error) {
		recordID, _ := input.Selector.Value.Int, input.Selector.ByKey == "id"
		titleValue, err := input.Input.RequireString("title")
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		_, err = executor.Update(ctx, actions.AtomicUpdate{
			Table:  items,
			Fields: []actions.AtomicUpdateField{{Column: title, Operation: actions.AtomicUpdateSet, Value: actions.AtomicString(titleValue)}},
			Where:  id.EQ(pg.Int(*recordID)),
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		return actions.AtomicRecord{}, errors.New("dependent update failed")
	}
	engine := setupAtomicUpdateRouter(t, sqlDB, broker, operation, func(*gin.Context, module.RelationScope) error { return nil })

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	w := executeJSONRequest(engine, http.MethodPost, "/admin/atomic-update-items/id/41", map[string]interface{}{"title": "updated"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "dependent update failed")
	require.NoError(t, mock.ExpectationsWereMet())

	events, _, err := broker.Replay(context.Background(), "0", 10)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestAtomicUpdateRejectsRelationScopeBeforeOpeningTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	called := false
	operation := func(context.Context, actions.AtomicExecutor, actions.AtomicUpdateInput) (actions.AtomicRecord, error) {
		called = true
		return actions.AtomicRecord{}, nil
	}
	engine := setupAtomicUpdateRouter(t, sqlDB, module.NewMemoryBroker(module.MemoryBrokerOptions{}), operation, func(*gin.Context, module.RelationScope) error {
		return errors.New("scope access denied")
	})

	w := executeJSONRequest(engine, http.MethodPost, "/admin/atomic-update-items/id/41?scope[relation]=owner&scope[id]=99", map[string]interface{}{"title": "updated"})
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	require.False(t, called)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicUpdateRejectsInvalidTypedSelectorBeforeOpeningTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	called := false
	operation := func(context.Context, actions.AtomicExecutor, actions.AtomicUpdateInput) (actions.AtomicRecord, error) {
		called = true
		return actions.AtomicRecord{}, nil
	}
	engine := setupAtomicUpdateRouter(t, sqlDB, module.NewMemoryBroker(module.MemoryBrokerOptions{}), operation, func(*gin.Context, module.RelationScope) error { return nil })

	w := executeJSONRequest(engine, http.MethodPost, "/admin/atomic-update-items/id/not-a-number", map[string]interface{}{"title": "updated"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "expected integer")
	require.False(t, called)
	require.NoError(t, mock.ExpectationsWereMet())
}

func mustAtomicInputString(t *testing.T, input actions.AtomicInput, name string) string {
	t.Helper()
	value, ok := input.String(name)
	require.True(t, ok)
	return value
}
