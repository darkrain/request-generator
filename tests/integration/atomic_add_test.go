package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	dbpkg "github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func setupAtomicAddRouter(t *testing.T, sqlDB *sql.DB, operation actions.AtomicAddOperation) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	items := pg.NewTable("public", "atomic_items", "", id, title)

	atomicModule := &module.BaseModule{
		Name:       "atomic-items",
		Path:       "/admin",
		Table:      items,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
		},
		Actions: []actions.ModuleAction{actions.AddModuleAction{
			Mode:       actions.AddModeAtomic,
			Atomic:     &actions.AtomicAddConfig{Operation: operation},
			Columns:    []pg.Column{title},
			Permission: []actions.Role{actions.RoleAll},
			Auth:       true,
		}},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) },
		*group,
		[]*module.BaseModule{atomicModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()
	return engine
}

func atomicCreateOperation(items, related pg.Table, primaryKey, title, relatedItemID pg.Column) actions.AtomicAddOperation {
	return func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicInput) (actions.AtomicRecord, error) {
		titleValue, err := input.RequireString("title")
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		item, err := executor.Insert(ctx, actions.AtomicInsert{
			Table: items, PrimaryKey: primaryKey,
			Fields: []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString(titleValue)}},
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		_, err = executor.Insert(ctx, actions.AtomicInsert{
			Table: related, PrimaryKey: primaryKey,
			Fields: []actions.AtomicInsertField{{Column: relatedItemID, Value: actions.AtomicInt(item.Value)}},
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		return actions.AtomicRecord{Value: item.Value, PrimaryKey: "id", Fields: []actions.AtomicField{{Name: "nick", Value: actions.AtomicString("atomic-item")}}}, nil
	}
}

func TestAtomicAdd_CommitsTypedRecordAndInterpolatesRoute(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	itemID := pg.IntegerColumn("item_id")
	items := pg.NewTable("public", "atomic_items", "", id, title)
	related := pg.NewTable("public", "atomic_related", "", id, itemID)
	engine := setupAtomicAddRouter(t, sqlDB, atomicCreateOperation(items, related, id, title, itemID))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."atomic_items" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."atomic_related" ("item_id") VALUES ($1) RETURNING "id"`)).WithArgs(int64(41)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
	mock.ExpectCommit()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"value":41,"primary_key":"id","nick":"atomic-item"}`, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())

	var record map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &record))
	route := strings.ReplaceAll("/profiles/{nick}", "{nick}", record["nick"].(string))
	require.Equal(t, "/profiles/atomic-item", route)
}

func TestAtomicAdd_PersistsTranslationsInsideTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	itemID := pg.IntegerColumn("item_id")
	name := pg.StringColumn("name")
	items := pg.NewTable("public", "atomic_items", "", id, title, name)
	related := pg.NewTable("public", "atomic_related", "", id, itemID)
	atomicModule := &module.BaseModule{
		Name: "atomic-items", Path: "/admin", Table: items, PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
			{Column: name, FieldName: "name", Title: "Name", Type: fields.ModuleFieldTypeObject, FormType: fields.ModuleFieldFormTypeText, Translatable: true},
		},
		Actions: []actions.ModuleAction{actions.AddModuleAction{
			Mode: actions.AddModeAtomic, Atomic: &actions.AtomicAddConfig{Operation: atomicCreateOperation(items, related, id, title, itemID)}, Columns: []pg.Column{title, name}, Permission: []actions.Role{actions.RoleAll}, Auth: true,
		}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(func(*module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) }, *group, []*module.BaseModule{atomicModule}, func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}, createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}))
	generator.Run()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO public").WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectQuery("INSERT INTO public").WithArgs(int64(41)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO translations (entity, entity_id, field, lang, value) VALUES ($1, $2, $3, $4, $5)`)).WithArgs("atomic_items", int64(41), "name", "en", "Hello").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello", "name": map[string]interface{}{"en": "Hello"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicAdd_RollsBackWhenTranslationInsertFails(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	itemID := pg.IntegerColumn("item_id")
	name := pg.StringColumn("name")
	items := pg.NewTable("public", "atomic_items", "", id, title, name)
	related := pg.NewTable("public", "atomic_related", "", id, itemID)
	atomicModule := &module.BaseModule{
		Name: "atomic-items", Path: "/admin", Table: items, PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
			{Column: name, FieldName: "name", Title: "Name", Type: fields.ModuleFieldTypeObject, FormType: fields.ModuleFieldFormTypeText, Translatable: true},
		},
		Actions: []actions.ModuleAction{actions.AddModuleAction{
			Mode: actions.AddModeAtomic, Atomic: &actions.AtomicAddConfig{Operation: atomicCreateOperation(items, related, id, title, itemID)}, Columns: []pg.Column{title, name}, Permission: []actions.Role{actions.RoleAll}, Auth: true,
		}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(func(*module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) }, *group, []*module.BaseModule{atomicModule}, func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}, createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}))
	generator.Run()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO public").WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectQuery("INSERT INTO public").WithArgs(int64(41)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
	mock.ExpectExec("INSERT INTO translations").WithArgs("atomic_items", int64(41), "name", "en", "Hello").WillReturnError(errors.New("translation insert failed"))
	mock.ExpectRollback()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello", "name": map[string]interface{}{"en": "Hello"}})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicAdd_RollsBackOnPrimaryDuplicateAndRelatedFailure(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		setup func(sqlmock.Sqlmock)
	}{
		{
			name: "duplicate primary",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("INSERT INTO public").WithArgs("duplicate").WillReturnError(errors.New("duplicate key"))
				mock.ExpectRollback()
			},
		},
		{
			name: "related insert failure",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("INSERT INTO public").WithArgs("related-fail").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
				mock.ExpectQuery("INSERT INTO public").WithArgs(int64(42)).WillReturnError(errors.New("related insert failed"))
				mock.ExpectRollback()
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			id := pg.IntegerColumn("id")
			title := pg.StringColumn("title")
			itemID := pg.IntegerColumn("item_id")
			items := pg.NewTable("public", "atomic_items", "", id, title)
			related := pg.NewTable("public", "atomic_related", "", id, itemID)
			engine := setupAtomicAddRouter(t, sqlDB, atomicCreateOperation(items, related, id, title, itemID))
			testCase.setup(mock)

			titleValue := "duplicate"
			if testCase.name == "related insert failure" {
				titleValue = "related-fail"
			}
			w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": titleValue})
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAtomicAdd_RollsBackInvalidResultBeforeCommit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	engine := setupAtomicAddRouter(t, sqlDB, func(context.Context, actions.AtomicExecutor, actions.AtomicInput) (actions.AtomicRecord, error) {
		return actions.AtomicRecord{
			Value:      41,
			PrimaryKey: "id",
			Fields: []actions.AtomicField{{
				Name:  "broken",
				Value: actions.AtomicValue{JSON: []byte("{")},
			}},
		}, nil
	})
	mock.ExpectBegin()
	mock.ExpectRollback()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NotEqual(t, http.StatusOK, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicAdd_SelectOneAndInsertsShareTransaction(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		selectErr error
		status    int
	}{
		{name: "success", status: http.StatusOK},
		{name: "select failure", selectErr: errors.New("agency context not found"), status: http.StatusBadRequest},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			id := pg.IntegerColumn("id")
			title := pg.StringColumn("title")
			ownerID := pg.IntegerColumn("owner_id")
			items := pg.NewTable("public", "atomic_items", "", id, title)
			related := pg.NewTable("public", "atomic_related", "", id, ownerID)
			owners := pg.NewTable("public", "atomic_owners", "", id, ownerID)
			operation := func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicInput) (actions.AtomicRecord, error) {
				owner, err := executor.SelectOne(ctx, actions.AtomicSelect{
					Table:  owners,
					Fields: []actions.AtomicSelectField{{Name: "owner_id", Column: ownerID, Kind: actions.AtomicValueKindInt}},
					Where:  ownerID.EQ(pg.Int(1)),
				})
				if err != nil {
					return actions.AtomicRecord{}, err
				}
				ownerValue, ok := owner.Field("owner_id")
				if !ok || ownerValue.Int == nil {
					return actions.AtomicRecord{}, errors.New("owner id missing")
				}
				titleValue, err := input.RequireString("title")
				if err != nil {
					return actions.AtomicRecord{}, err
				}
				item, err := executor.Insert(ctx, actions.AtomicInsert{Table: items, PrimaryKey: id, Fields: []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString(titleValue)}}})
				if err != nil {
					return actions.AtomicRecord{}, err
				}
				_, err = executor.Insert(ctx, actions.AtomicInsert{Table: related, PrimaryKey: id, Fields: []actions.AtomicInsertField{{Column: ownerID, Value: actions.AtomicInt(*ownerValue.Int)}}})
				if err != nil {
					return actions.AtomicRecord{}, err
				}
				return actions.AtomicRecord{Value: item.Value, PrimaryKey: "id"}, nil
			}
			engine := setupAtomicAddRouter(t, sqlDB, operation)
			mock.ExpectBegin()
			selectExpectation := mock.ExpectQuery("SELECT")
			if testCase.selectErr != nil {
				selectExpectation.WillReturnError(testCase.selectErr)
				mock.ExpectRollback()
			} else {
				selectExpectation.WillReturnRows(sqlmock.NewRows([]string{"owner_id"}).AddRow(9))
				mock.ExpectQuery("INSERT INTO public").WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
				mock.ExpectQuery("INSERT INTO public").WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
				mock.ExpectCommit()
			}

			w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
			require.Equal(t, testCase.status, w.Code, w.Body.String())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAtomicAdd_SelectManyReadsInsideTransactionAndRollsBack(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	userID := pg.IntegerColumn("user_id")
	items := pg.NewTable("public", "atomic_items", "", id, title)
	recipients := pg.NewTable("public", "atomic_recipients", "", id, userID)
	operation := func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicInput) (actions.AtomicRecord, error) {
		titleValue, err := input.RequireString("title")
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		item, err := executor.Insert(ctx, actions.AtomicInsert{
			Table:      items,
			PrimaryKey: id,
			Fields:     []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString(titleValue)}},
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		records, err := executor.SelectMany(ctx, actions.AtomicSelectMany{
			AtomicSelect: actions.AtomicSelect{
				Table:  recipients,
				Fields: []actions.AtomicSelectField{{Name: "user_id", Column: userID, Kind: actions.AtomicValueKindInt}},
				Where:  userID.GT(pg.Int(0)),
			},
			OrderBy: []pg.OrderByClause{userID.ASC()},
			Limit:   2,
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		if len(records) != 2 {
			return actions.AtomicRecord{}, errors.New("expected two recipients")
		}
		firstUserID, ok := records[0].Int("user_id")
		if !ok || firstUserID != 10 {
			return actions.AtomicRecord{}, errors.New("first recipient is unavailable")
		}
		return actions.AtomicRecord{Value: item.Value, PrimaryKey: "id"}, errors.New("force rollback after select many")
	}
	engine := setupAtomicAddRouter(t, sqlDB, operation)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."atomic_items" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectQuery(`ORDER BY atomic_recipients\.user_id ASC[\s\S]*LIMIT \$2`).WithArgs(int64(0), int64(2)).WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(10).AddRow(20))
	mock.ExpectRollback()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func atomicInsertThenUpdateOperation(items, counters pg.Table, primaryKey, title pg.Column, counterID, unreadCount pg.ColumnInteger) actions.AtomicAddOperation {
	return func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicInput) (actions.AtomicRecord, error) {
		titleValue, err := input.RequireString("title")
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		item, err := executor.Insert(ctx, actions.AtomicInsert{
			Table:      items,
			PrimaryKey: primaryKey,
			Fields:     []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString(titleValue)}},
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		updated, err := executor.Update(ctx, actions.AtomicUpdate{
			Table: counters,
			Fields: []actions.AtomicUpdateField{{
				Column:    unreadCount,
				Operation: actions.AtomicUpdateIncrement,
				Value:     actions.AtomicInt(1),
			}},
			Where: counterID.EQ(pg.Int(9)),
		})
		if err != nil {
			return actions.AtomicRecord{}, err
		}
		if updated != 1 {
			return actions.AtomicRecord{}, errors.New("counter was not updated")
		}
		return actions.AtomicRecord{Value: item.Value, PrimaryKey: primaryKey.Name()}, nil
	}
}

func TestAtomicAdd_InsertThenUpdateCommitsInOneTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	counterID := pg.IntegerColumn("id")
	unreadCount := pg.IntegerColumn("unread_count")
	items := pg.NewTable("public", "atomic_items", "", id, title)
	counters := pg.NewTable("public", "atomic_counters", "", counterID, unreadCount)
	engine := setupAtomicAddRouter(t, sqlDB, atomicInsertThenUpdateOperation(items, counters, id, title, counterID, unreadCount))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."atomic_items" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectExec(`UPDATE public\.atomic_counters`).WithArgs(int64(1), int64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"value":41,"primary_key":"id"}`, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicAdd_InsertThenUpdateRollsBackOnUpdateError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	counterID := pg.IntegerColumn("id")
	unreadCount := pg.IntegerColumn("unread_count")
	items := pg.NewTable("public", "atomic_items", "", id, title)
	counters := pg.NewTable("public", "atomic_counters", "", counterID, unreadCount)
	engine := setupAtomicAddRouter(t, sqlDB, atomicInsertThenUpdateOperation(items, counters, id, title, counterID, unreadCount))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."atomic_items" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectExec(`UPDATE public\.atomic_counters`).WithArgs(int64(1), int64(9)).WillReturnError(errors.New("counter update failed"))
	mock.ExpectRollback()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicAdd_RejectsHooksAtConfigurationTime(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		before     func(*gin.Context) error
		after      func(*gin.Context)
		roleBefore []actions.RoleHook
		roleAfter  []actions.RoleAfterHook
	}{
		{name: "action before hook", before: func(*gin.Context) error { return nil }},
		{name: "action after hook", after: func(*gin.Context) {}},
		{name: "module before hook", roleBefore: []actions.RoleHook{{Role: actions.RoleAll, Hook: func(*gin.Context) error { return nil }}}},
		{name: "module after hook", roleAfter: []actions.RoleAfterHook{{Role: actions.RoleAll, Hook: func(*gin.Context) {}}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			id := pg.IntegerColumn("id")
			title := pg.StringColumn("title")
			table := pg.NewTable("public", "atomic_items", "", id, title)
			badModule := &module.BaseModule{
				Name: "atomic-items", Path: "/admin", Table: table, PrimaryKey: id,
				RoleBeforeHook: testCase.roleBefore, RoleAfterHook: testCase.roleAfter,
				Actions: []actions.ModuleAction{actions.AddModuleAction{Mode: actions.AddModeAtomic, BeforeAction: testCase.before, AfterAction: testCase.after, Atomic: &actions.AtomicAddConfig{Operation: func(context.Context, actions.AtomicExecutor, actions.AtomicInput) (actions.AtomicRecord, error) {
					return actions.AtomicRecord{}, nil
				}}}},
			}
			engine := gin.New()
			group := engine.Group("")
			generator := module.NewGenerator(func(*module.BaseModule) dbpkg.DBExecutor { return nil }, *group, []*module.BaseModule{badModule}, nil, nil)
			require.Panics(t, generator.Run)
		})
	}
}

func TestAtomicAddPublishesTypedRealtimeAfterCommit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	chatID := pg.IntegerColumn("chat_id")
	recipientUserIDs := pg.IntegerColumn("recipient_user_ids")
	items := pg.NewTable("public", "atomic_items", "", id, title)
	broker := module.NewMemoryBroker(module.MemoryBrokerOptions{MaxEvents: 10})
	atomicModule := &module.BaseModule{
		Name:       "atomic-items",
		Path:       "/admin",
		Table:      items,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
			{Column: chatID, Title: "Chat", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: recipientUserIDs, Title: "Recipients", Type: fields.ModuleFieldTypeArray, FormType: fields.ModuleFieldFormTypeOnlyView},
		},
		Actions: []actions.ModuleAction{actions.AddModuleAction{
			Mode: actions.AddModeAtomic,
			Atomic: &actions.AtomicAddConfig{
				Operation: func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicInput) (actions.AtomicRecord, error) {
					titleValue, err := input.RequireString("title")
					if err != nil {
						return actions.AtomicRecord{}, err
					}
					record, err := executor.Insert(ctx, actions.AtomicInsert{
						Table:      items,
						PrimaryKey: id,
						Fields:     []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString(titleValue)}},
					})
					if err != nil {
						return actions.AtomicRecord{}, err
					}
					record.Fields = []actions.AtomicField{
						{Name: "chat_id", Value: actions.AtomicInt(42)},
						{Name: "recipient_user_ids", Value: actions.AtomicValue{Ints: []int64{2, 3}}},
					}
					return record, nil
				},
				ResultFields: []actions.AtomicResultField{
					{Name: "chat_id", Kind: actions.AtomicValueKindInt},
					{Name: "recipient_user_ids", Kind: actions.AtomicValueKindInts},
				},
				Publish: []actions.AtomicRealtimePublishConfig{{
					Recipients: []actions.AtomicRealtimeRecipient{{
						UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "recipient_user_ids"},
					}},
					Correlation: &actions.AtomicRealtimeCorrelation{
						Field:  "chat_id",
						Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"},
					},
				}},
			},
			Columns:    []pg.Column{title},
			Permission: []actions.Role{actions.RoleAll},
			Auth:       true,
			Realtime:   &actions.RealtimeEventConfig{CorrelationField: "chat_id"},
		}},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) },
		*group,
		[]*module.BaseModule{atomicModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Realtime = module.RealtimeConfig{Enabled: true, Broker: broker}
	generator.Run()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."atomic_items" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("hello").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectCommit()

	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())

	events, resync, err := broker.Replay(context.Background(), "0", 10)
	require.NoError(t, err)
	require.False(t, resync)
	require.Len(t, events, 1)
	require.Equal(t, []string{"user:2", "user:3"}, events[0].Topics)
	require.Equal(t, "chat_id", events[0].Correlation.Field)
	require.Equal(t, renderer.TypedValueNumber, events[0].Correlation.Value.Type)
	require.Equal(t, float64(42), events[0].Correlation.Value.Number)
}

func TestAtomicAddDoesNotPublishRealtimeOnRollback(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	chatID := pg.IntegerColumn("chat_id")
	recipientUserID := pg.IntegerColumn("recipient_user_id")
	items := pg.NewTable("public", "atomic_items", "", id, title)
	broker := module.NewMemoryBroker(module.MemoryBrokerOptions{MaxEvents: 10})
	atomicModule := &module.BaseModule{
		Name: "atomic-items", Path: "/admin", Table: items, PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
			{Column: chatID, Title: "Chat", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: recipientUserID, Title: "Recipient", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
		},
		Actions: []actions.ModuleAction{actions.AddModuleAction{
			Mode: actions.AddModeAtomic,
			Atomic: &actions.AtomicAddConfig{
				Operation: func(context.Context, actions.AtomicExecutor, actions.AtomicInput) (actions.AtomicRecord, error) {
					return actions.AtomicRecord{}, errors.New("forced rollback")
				},
				ResultFields: []actions.AtomicResultField{{Name: "chat_id", Kind: actions.AtomicValueKindInt}, {Name: "recipient_user_id", Kind: actions.AtomicValueKindInt}},
				Publish: []actions.AtomicRealtimePublishConfig{{
					Recipients:  []actions.AtomicRealtimeRecipient{{UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "recipient_user_id"}}},
					Correlation: &actions.AtomicRealtimeCorrelation{Field: "chat_id", Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"}},
				}},
			},
			Columns: []pg.Column{title}, Permission: []actions.Role{actions.RoleAll}, Auth: true,
			Realtime: &actions.RealtimeEventConfig{CorrelationField: "chat_id"},
		}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(func(_ *module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) }, *group, []*module.BaseModule{atomicModule}, func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}, createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}))
	generator.Realtime = module.RealtimeConfig{Enabled: true, Broker: broker}
	generator.Run()

	mock.ExpectBegin()
	mock.ExpectRollback()
	w := executeJSONRequest(engine, http.MethodPut, "/admin/atomic-items", map[string]interface{}{"title": "hello"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
	events, _, err := broker.Replay(context.Background(), "0", 10)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestStandardAdd_RemainsUnchanged(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	id := pg.IntegerColumn("id")
	title := pg.StringColumn("title")
	table := pg.NewTable("public", "standard_items", "", id, title)
	standard := &module.BaseModule{
		Name: "standard-items", Path: "/admin", Table: table, PrimaryKey: id,
		Fields:  []fields.ModuleField{{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText}},
		Actions: []actions.ModuleAction{actions.AddModuleAction{Columns: []pg.Column{title}, Permission: []actions.Role{actions.RoleAll}, Auth: true}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(func(*module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) }, *group, []*module.BaseModule{standard}, func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}, createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}))
	generator.Run()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."standard_items" ("title") VALUES ($1) RETURNING "id"`)).WithArgs("ordinary").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectCommit()
	w := executeJSONRequest(engine, http.MethodPut, "/admin/standard-items", map[string]interface{}{"title": "ordinary"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"value":9,"primary_key":"id"}`, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}
