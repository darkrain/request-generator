package integration

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	dbpkg "github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/locale"
	"github.com/gin-gonic/gin"
	"github.com/go-jet/jet/v2/postgres"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeListDB struct{}

func (fakeListDB) List(
	_ *log.Entry,
	_ pg.Table,
	_ pg.Column,
	_ []fields.ModuleField,
	_ []fields.ModuleField,
	_ int64,
	_ int64,
	_ []pg.Column,
	_ string,
	_ map[string]string,
	_ pg.BoolExpression,
	_ []actions.ModuleActionJoin,
	_ *actions.SortOption,
	_ *dbpkg.TranslationContext,
) ([]interface{}, int64, error) {
	return []interface{}{}, 0, nil
}

func (fakeListDB) View(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, _ pg.BoolExpression, _ []actions.ModuleActionJoin, _ *dbpkg.TranslationContext) (interface{}, error) {
	return map[string]interface{}{}, nil
}

func (fakeListDB) Add(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, _ map[string]interface{}, _ *dbpkg.TranslationContext) (interface{}, error) {
	return nil, nil
}

func (fakeListDB) Update(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, _ map[string]interface{}, _ pg.BoolExpression, _ *dbpkg.TranslationContext) (interface{}, error) {
	return nil, nil
}

func (fakeListDB) Delete(_ *log.Entry, _ pg.Table, _ pg.BoolExpression, _ *dbpkg.TranslationContext) error {
	return nil
}

func (fakeListDB) RawRequest(_ *log.Entry, _ string, _ ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (fakeListDB) RawDB() *sql.DB {
	return nil
}

func TestListAction_MetadataUsesTranslationContextAndFieldExtra(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := postgres.IntegerColumn("id")
	status := postgres.StringColumn("status")
	table := postgres.NewTable("public", "metadata_items", "", id, status)

	hookTranslation := ""
	testModule := &module.BaseModule{
		Name:       "metadata-items",
		Label:      "metadata.items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{
				Column:   id,
				Title:    "metadata.fields.id",
				Type:     fields.ModuleFieldTypeInt,
				FormType: fields.ModuleFieldFormTypeNumber,
			},
			{
				Column:   status,
				Title:    "metadata.fields.status",
				Type:     fields.ModuleFieldTypeString,
				FormType: fields.ModuleFieldFormTypeSelect,
				Group:    "base",
				Order:    1,
				Extra: &fields.FieldExtra{
					List: map[string]interface{}{
						"filter_group": "advanced",
						"filter_order": 42,
					},
				},
			},
		},
		RoleBeforeHook: []actions.RoleHook{
			{
				Role: actions.RoleAll,
				Hook: func(c *gin.Context) error {
					hookTranslation = module.Translate(c, "metadata.hook", "hook fallback")
					return nil
				},
			},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Columns:    []pg.Column{id, status},
				Filter:     []pg.Column{status},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "metadata.actions.list",
				ExtraFunc: func(c *gin.Context) interface{} {
					return map[string]interface{}{
						"title":        module.Translate(c, "metadata.title", "fallback title"),
						"plural_one":   module.Plural(c, "metadata.item_count", 1, "fallback one"),
						"plural_other": module.Plural(c, "metadata.item_count", 2, "fallback many"),
					}
				},
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeListDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Locales = []locale.Lang{locale.EN}

	translationsPath := filepath.Join(t.TempDir(), "en.json")
	err := os.WriteFile(translationsPath, []byte(`{
		"metadata": {
			"title": "Translated title",
			"hook": "Translated hook",
			"item_count": {
				"one": "One item",
				"other": "Many items"
			}
		}
	}`), 0o600)
	require.NoError(t, err)
	require.NoError(t, generator.LoadTranslationsFile(locale.EN, translationsPath))

	generator.Run()

	w := executeRequest(engine, http.MethodGet, "/admin/metadata-items?addFilters=true", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Extra   map[string]interface{} `json:"extra"`
		Filters map[string]struct {
			Group string                 `json:"group"`
			Order int                    `json:"order"`
			Extra map[string]interface{} `json:"extra"`
		} `json:"filters"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	assert.Equal(t, "Translated hook", hookTranslation)
	assert.Equal(t, "Translated title", response.Extra["title"])
	assert.Equal(t, "One item", response.Extra["plural_one"])
	assert.Equal(t, "Many items", response.Extra["plural_other"])

	statusFilter, ok := response.Filters["status"]
	require.True(t, ok)
	assert.Equal(t, "advanced", statusFilter.Group)
	assert.Equal(t, 42, statusFilter.Order)
	assert.Equal(t, "advanced", statusFilter.Extra["filter_group"])
	assert.Equal(t, float64(42), statusFilter.Extra["filter_order"])
}
