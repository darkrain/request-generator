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
	require.JSONEq(t, `{"value":41,"primary_key":"id","fields":[{"name":"nick","value":{"string":"atomic-item"}}]}`, w.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())

	route, err := (actions.AtomicRecord{Fields: []actions.AtomicField{{Name: "nick", Value: actions.AtomicString("atomic-item")}}}).InterpolateRoute("/profiles/{nick}")
	require.NoError(t, err)
	require.Equal(t, "/profiles/atomic-item", route)
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
