package integration

import (
	"context"
	"database/sql"
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

const (
	jsonArrayPayload     = `[{"cid":"bafy-test","kind":"image"}]`
	postgresArrayPayload = `{"vip","verified"}`
)

func setupArrayStorageRouter(t *testing.T, sqlDB *sql.DB, storage fields.ModuleFieldArrayStorage, atomic bool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	id := pg.IntegerColumn("id")
	media := pg.StringColumn("media")
	items := pg.NewTable("public", "array_items", "", id, media)
	add := actions.AddModuleAction{
		Columns:    []pg.Column{media},
		Permission: []actions.Role{actions.RoleAll},
		Auth:       true,
	}
	if atomic {
		add.Mode = actions.AddModeAtomic
		add.Atomic = &actions.AtomicAddConfig{Operation: func(ctx context.Context, executor actions.AtomicExecutor, input actions.AtomicInput) (actions.AtomicRecord, error) {
			mediaValue, ok := input.Field("media")
			if !ok {
				return actions.AtomicRecord{}, context.Canceled
			}
			record, err := executor.Insert(ctx, actions.AtomicInsert{
				Table:      items,
				PrimaryKey: id,
				Fields:     []actions.AtomicInsertField{{Column: media, Value: mediaValue}},
			})
			if err != nil {
				return actions.AtomicRecord{}, err
			}
			record.Fields = []actions.AtomicField{{Name: "media", Value: mediaValue}}
			return record, nil
		}}
	}

	arrayModule := &module.BaseModule{
		Name:       "array-items",
		Path:       "/admin",
		Table:      items,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: media, Title: "Media", Type: fields.ModuleFieldTypeArray, FormType: fields.ModuleFieldFormTypeOnlyView, ArrayStorage: storage},
		},
		Actions: []actions.ModuleAction{
			add,
			actions.UpdateModuleAction{
				Columns:    []pg.Column{media},
				By:         []pg.Column{id},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return dbpkg.NewDB(sqlDB) },
		*group,
		[]*module.BaseModule{arrayModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()
	return engine
}

func expectArrayStorageView(mock sqlmock.Sqlmock, value string) {
	rows := sqlmock.NewRows([]string{"id", "media"}).AddRow(17, value)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	rows = sqlmock.NewRows([]string{"id", "media"}).AddRow(17, value)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
}

func TestArrayStorageJSONStandardAddAndUpdate(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	engine := setupArrayStorageRouter(t, sqlDB, fields.ModuleFieldArrayStorageJSON, false)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."array_items" ("media") VALUES ($1) RETURNING "id"`)).
		WithArgs(jsonArrayPayload).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(17))
	mock.ExpectCommit()

	add := executeJSONRequest(engine, http.MethodPut, "/admin/array-items", map[string]interface{}{
		"media": []interface{}{map[string]interface{}{"cid": "bafy-test", "kind": "image"}},
	})
	require.Equal(t, http.StatusOK, add.Code, add.Body.String())
	require.JSONEq(t, `{"value":17,"primary_key":"id"}`, add.Body.String())

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public."array_items" SET "media" = $1 WHERE "id" = $2`)).
		WithArgs(jsonArrayPayload, "17").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectArrayStorageView(mock, jsonArrayPayload)

	update := executeJSONRequest(engine, http.MethodPost, "/admin/array-items/id/17", map[string]interface{}{
		"media": []interface{}{map[string]interface{}{"cid": "bafy-test", "kind": "image"}},
	})
	require.Equal(t, http.StatusOK, update.Code, update.Body.String())
	require.JSONEq(t, `{"media":[{"cid":"bafy-test","kind":"image"}]}`, update.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestArrayStorageJSONAtomicAddNormalizesInputAndReturnsJSONArray(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	engine := setupArrayStorageRouter(t, sqlDB, fields.ModuleFieldArrayStorageJSON, true)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."array_items" ("media") VALUES ($1) RETURNING "id"`)).
		WithArgs(jsonArrayPayload).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(17))
	mock.ExpectCommit()

	response := executeJSONRequest(engine, http.MethodPut, "/admin/array-items", map[string]interface{}{
		"media": []interface{}{map[string]interface{}{"cid": "bafy-test", "kind": "image"}},
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.JSONEq(t, `{"value":17,"primary_key":"id","media":[{"cid":"bafy-test","kind":"image"}]}`, response.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestArrayStoragePostgresStandardAddAndUpdateRemainCompatible(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	engine := setupArrayStorageRouter(t, sqlDB, "", false)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."array_items" ("media") VALUES ($1) RETURNING "id"`)).
		WithArgs(postgresArrayPayload).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(17))
	mock.ExpectCommit()

	add := executeJSONRequest(engine, http.MethodPut, "/admin/array-items", map[string]interface{}{"media": []interface{}{"vip", "verified"}})
	require.Equal(t, http.StatusOK, add.Code, add.Body.String())

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public."array_items" SET "media" = $1 WHERE "id" = $2`)).
		WithArgs(postgresArrayPayload, "17").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectArrayStorageView(mock, postgresArrayPayload)

	update := executeJSONRequest(engine, http.MethodPost, "/admin/array-items/id/17", map[string]interface{}{"media": []interface{}{"vip", "verified"}})
	require.Equal(t, http.StatusOK, update.Code, update.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestArrayStoragePostgresAtomicAddRemainsCompatible(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	engine := setupArrayStorageRouter(t, sqlDB, "", true)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."array_items" ("media") VALUES ($1) RETURNING "id"`)).
		WithArgs(postgresArrayPayload).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(17))
	mock.ExpectCommit()

	response := executeJSONRequest(engine, http.MethodPut, "/admin/array-items", map[string]interface{}{"media": []interface{}{"vip", "verified"}})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.JSONEq(t, `{"value":17,"primary_key":"id","media":["vip","verified"]}`, response.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}
