package integration

import (
	"database/sql"
	"errors"
	"net/http"
	"testing"

	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	dbpkg "github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scopedActionDB struct {
	table      pg.Table
	lastInput  map[string]interface{}
	whereArgs  []interface{}
	listCalled bool
}

type scopeFieldConverters struct {
	source func(*gin.Context, interface{}) (interface{}, error)
	target func(*gin.Context, interface{}) (interface{}, error)
}

func (db *scopedActionDB) List(
	_ *log.Entry,
	table pg.Table,
	_ pg.Column,
	_ []fields.ModuleField,
	_ []fields.ModuleField,
	_ int64,
	_ int64,
	_ []pg.Column,
	_ string,
	_ map[string]string,
	where pg.BoolExpression,
	_ []actions.ModuleActionJoin,
	_ *actions.SortOption,
	_ *dbpkg.TranslationContext,
) ([]interface{}, int64, error) {
	db.listCalled = true
	db.captureWhere(table, where)
	return []interface{}{}, 0, nil
}

func (db *scopedActionDB) View(_ *log.Entry, table pg.Table, _ pg.Column, _ []fields.ModuleField, where pg.BoolExpression, _ []actions.ModuleActionJoin, _ *dbpkg.TranslationContext) (interface{}, error) {
	db.captureWhere(table, where)
	return map[string]interface{}{"id": 88, "owner_id": 123, "title": "changed"}, nil
}

func (db *scopedActionDB) Add(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, input map[string]interface{}, _ *dbpkg.TranslationContext) (interface{}, error) {
	db.lastInput = input
	return input, nil
}

func (db *scopedActionDB) Update(_ *log.Entry, table pg.Table, _ pg.Column, _ []fields.ModuleField, input map[string]interface{}, where pg.BoolExpression, _ *dbpkg.TranslationContext) (interface{}, error) {
	db.lastInput = input
	db.captureWhere(table, where)
	return nil, nil
}

func (db *scopedActionDB) Delete(_ *log.Entry, table pg.Table, where pg.BoolExpression, _ *dbpkg.TranslationContext) error {
	db.captureWhere(table, where)
	return nil
}

func (db *scopedActionDB) RawRequest(_ *log.Entry, _ string, _ ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (db *scopedActionDB) RawDB() *sql.DB {
	return nil
}

func (db *scopedActionDB) captureWhere(table pg.Table, where pg.BoolExpression) {
	if where == nil {
		db.whereArgs = nil
		return
	}
	_, args := pg.SELECT(pg.Raw("1")).FROM(table).WHERE(where).Sql()
	db.whereArgs = args
}

func setupRelationScopeActionsRouter(t *testing.T, scopeCheck func(*gin.Context, module.RelationScope) error, converters ...scopeFieldConverters) (*gin.Engine, *scopedActionDB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var converter scopeFieldConverters
	if len(converters) > 0 {
		converter = converters[0]
	}

	id := pg.IntegerColumn("id")
	ownerID := pg.IntegerColumn("owner_id")
	title := pg.StringColumn("title")
	ownerTable := pg.NewTable("public", "owners", "", id)
	itemsTable := pg.NewTable("public", "scoped_items", "", id, ownerID, title)
	db := &scopedActionDB{table: itemsTable}

	itemsModule := &module.BaseModule{
		Name:       "scoped-items",
		Path:       "/admin",
		Table:      itemsTable,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
			{Column: ownerID, Title: "Owner", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber, Convert: converter.source},
			{Column: title, Title: "Title", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
		},
		Relations: []module.ModuleRelation{{
			Name:         "owner",
			TargetModule: "owners",
			SourceField:  ownerID,
			TargetField:  id,
			ScopeCheck:   scopeCheck,
		}},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Columns: []pg.Column{id, ownerID, title}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
			actions.AddModuleAction{Columns: []pg.Column{title}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
			actions.UpdateModuleAction{Columns: []pg.Column{title}, By: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
			actions.DeleteModuleAction{By: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
		},
	}
	ownersModule := &module.BaseModule{
		Name:       "owners",
		Path:       "/admin",
		Table:      ownerTable,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber, Convert: converter.target},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return db },
		*group,
		[]*module.BaseModule{itemsModule, ownersModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()
	return engine, db
}

func TestRelationScopeActions_ListAddUpdateDelete(t *testing.T) {
	var checkedScopes []module.RelationScope
	engine, db := setupRelationScopeActionsRouter(t, func(_ *gin.Context, scope module.RelationScope) error {
		checkedScopes = append(checkedScopes, scope)
		return nil
	})

	w := executeRequest(engine, http.MethodGet, "/admin/scoped-items?scope[relation]=owner&scope[id]=123", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, db.listCalled)
	assert.Contains(t, db.whereArgs, int64(123))

	w = executeJSONRequest(engine, http.MethodPut, "/admin/scoped-items?scope[relation]=owner&scope[id]=123", map[string]interface{}{"title": "new"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "new", db.lastInput["title"])
	assert.Equal(t, int64(123), db.lastInput["owner_id"])

	w = executeJSONRequest(engine, http.MethodPost, "/admin/scoped-items/id/88?scope[relation]=owner&scope[id]=123", map[string]interface{}{"title": "changed"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "changed", db.lastInput["title"])
	assert.NotContains(t, db.lastInput, "owner_id")
	assert.Contains(t, db.whereArgs, "88")
	assert.Contains(t, db.whereArgs, int64(123))

	w = executeRequest(engine, http.MethodDelete, "/admin/scoped-items/delete/id/88?scope[relation]=owner&scope[id]=123", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, db.whereArgs, "88")
	assert.Contains(t, db.whereArgs, int64(123))

	require.Len(t, checkedScopes, 4)
	for _, scope := range checkedScopes {
		assert.Equal(t, module.RelationScope{Relation: "owner", ID: int64(123)}, scope)
	}
}

func TestRelationScopeActions_RejectSourceFieldAndDeniedScope(t *testing.T) {
	engine, _ := setupRelationScopeActionsRouter(t, func(_ *gin.Context, _ module.RelationScope) error {
		return nil
	})

	w := executeJSONRequest(engine, http.MethodPut, "/admin/scoped-items?scope[relation]=owner&scope[id]=123", map[string]interface{}{"title": "new", "owner_id": 999})
	require.Equal(t, http.StatusBadRequest, w.Code)

	deniedEngine, _ := setupRelationScopeActionsRouter(t, func(_ *gin.Context, _ module.RelationScope) error {
		return errors.New("scope access denied")
	})
	w = executeRequest(deniedEngine, http.MethodGet, "/admin/scoped-items?scope[relation]=owner&scope[id]=123", nil)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRelationScopeActions_UnscopedBehaviorUnchanged(t *testing.T) {
	engine, db := setupRelationScopeActionsRouter(t, func(_ *gin.Context, _ module.RelationScope) error {
		return errors.New("should not be called")
	})

	w := executeRequest(engine, http.MethodGet, "/admin/scoped-items", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, db.whereArgs)

	w = executeJSONRequest(engine, http.MethodPut, "/admin/scoped-items", map[string]interface{}{"title": "new"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "new", db.lastInput["title"])
	assert.NotContains(t, db.lastInput, "owner_id")
}

func TestRelationScopeActions_AddDoesNotReconvertCanonicalScopeID(t *testing.T) {
	type canonicalID struct{ value string }

	engine, db := setupRelationScopeActionsRouter(t, func(_ *gin.Context, scope module.RelationScope) error {
		assert.Equal(t, canonicalID{value: "owner-123"}, scope.ID)
		return nil
	}, scopeFieldConverters{
		source: func(_ *gin.Context, value interface{}) (interface{}, error) {
			if _, ok := value.(string); !ok {
				return nil, errors.New("source converter accepts only transport strings")
			}
			return value, nil
		},
		target: func(_ *gin.Context, value interface{}) (interface{}, error) {
			raw, ok := value.(string)
			if !ok {
				return nil, errors.New("target converter accepts only transport strings")
			}
			return canonicalID{value: raw}, nil
		},
	})

	w := executeJSONRequest(engine, http.MethodPut, "/admin/scoped-items?scope[relation]=owner&scope[id]=owner-123", map[string]interface{}{"title": "new"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, canonicalID{value: "owner-123"}, db.lastInput["owner_id"])
}
