package integration

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	dbpkg "github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	"github.com/go-jet/jet/v2/postgres"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRendererDB struct{}

func (fakeRendererDB) List(
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

func (fakeRendererDB) View(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, _ pg.BoolExpression, _ []actions.ModuleActionJoin, _ *dbpkg.TranslationContext) (interface{}, error) {
	return map[string]interface{}{"id": 1, "status": "active"}, nil
}

func (fakeRendererDB) Add(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, _ map[string]interface{}, _ *dbpkg.TranslationContext) (interface{}, error) {
	return nil, nil
}

func (fakeRendererDB) Update(_ *log.Entry, _ pg.Table, _ pg.Column, _ []fields.ModuleField, _ map[string]interface{}, _ pg.BoolExpression, _ *dbpkg.TranslationContext) (interface{}, error) {
	return nil, nil
}

func (fakeRendererDB) Delete(_ *log.Entry, _ pg.Table, _ pg.BoolExpression, _ *dbpkg.TranslationContext) error {
	return nil
}

func (fakeRendererDB) RawRequest(_ *log.Entry, _ string, _ ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (fakeRendererDB) RawDB() *sql.DB {
	return nil
}

func setupUniversalRendererRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	id := postgres.IntegerColumn("id")
	status := postgres.StringColumn("status")
	table := postgres.NewTable("public", "renderer_items", "", id, status)

	testModule := &module.BaseModule{
		Name:       "renderer-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
			{Column: status, Title: "Status", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeSelect},
		},
		Render: renderer.Universal{
			List: &renderer.ListPage{
				ID: "renderer-items",
				Layout: &renderer.Layout{
					Type:  renderer.LayoutOneColumn,
					Gap:   renderer.SpacingMD,
					Align: renderer.AlignStretch,
				},
			},
			Form: &renderer.FormPage{
				ID:     "renderer-items-form",
				Layout: renderer.LayoutTwoColumn,
			},
			Record: &renderer.RecordPage{
				Layout: &renderer.Layout{Type: renderer.LayoutThreeColumn},
			},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Columns:    []pg.Column{id, status},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "List",
			},
			actions.AddModuleAction{
				Columns:    []pg.Column{status},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "Add",
			},
			actions.ViewModuleAction{
				Columns:    []pg.Column{id, status},
				By:         []pg.Column{id},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "View",
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()
	return engine
}

func TestUniversalRendererMetadata_ListResponse(t *testing.T) {
	engine := setupUniversalRendererRouter(t)

	w := executeRequest(engine, http.MethodGet, "/admin/renderer-items", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Renderer *renderer.Identity `json:"renderer"`
		ListPage *renderer.ListPage `json:"list_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	require.NotNil(t, response.Renderer)
	assert.Equal(t, renderer.Name, response.Renderer.Name)
	assert.Equal(t, renderer.Version, response.Renderer.Version)
	require.NotNil(t, response.ListPage)
	assert.Equal(t, "renderer-items", response.ListPage.ID)
	assert.Equal(t, renderer.LayoutOneColumn, response.ListPage.Layout.Type)
	assert.Equal(t, renderer.SpacingMD, response.ListPage.Layout.Gap)
}

func TestUniversalRendererMetadata_DefrecResponse(t *testing.T) {
	engine := setupUniversalRendererRouter(t)

	w := executeRequest(engine, http.MethodGet, "/admin/renderer-items/defrec/", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Renderer *renderer.Identity `json:"renderer"`
		FormPage *renderer.FormPage `json:"form_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	require.NotNil(t, response.Renderer)
	assert.Equal(t, renderer.Name, response.Renderer.Name)
	assert.Equal(t, renderer.Version, response.Renderer.Version)
	require.NotNil(t, response.FormPage)
	assert.Equal(t, "renderer-items-form", response.FormPage.ID)
	assert.Equal(t, renderer.LayoutTwoColumn, response.FormPage.Layout)
}

func TestUniversalRendererMetadata_ViewResponse(t *testing.T) {
	engine := setupUniversalRendererRouter(t)

	w := executeRequest(engine, http.MethodGet, "/admin/renderer-items/view/id/1", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Renderer   *renderer.Identity   `json:"renderer"`
		RecordPage *renderer.RecordPage `json:"record_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	require.NotNil(t, response.Renderer)
	assert.Equal(t, renderer.Name, response.Renderer.Name)
	assert.Equal(t, renderer.Version, response.Renderer.Version)
	require.NotNil(t, response.RecordPage)
	assert.Equal(t, renderer.LayoutThreeColumn, response.RecordPage.Layout.Type)
}

func TestUniversalRendererMetadata_ListAndResourceGridAreMutuallyExclusive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := postgres.IntegerColumn("id")
	table := postgres.NewTable("public", "invalid_renderer_items", "", id)

	testModule := &module.BaseModule{
		Name:       "invalid-renderer-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
		},
		Render: renderer.Universal{
			List:         &renderer.ListPage{ID: "invalid-renderer-items"},
			ResourceGrid: &renderer.ResourceGridPage{Endpoint: "/invalid-renderer-items"},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Columns:    []pg.Column{id},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "List",
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)

	assert.PanicsWithValue(
		t,
		"invalid renderer config in module invalid-renderer-items: renderer.Universal: List and ResourceGrid are mutually exclusive for one list route",
		func() { generator.Run() },
	)
}

func TestUniversalRendererMetadata_RenderFuncBuildsTypedRuntimeMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := postgres.IntegerColumn("id")
	table := postgres.NewTable("public", "dynamic_renderer_items", "", id)

	testModule := &module.BaseModule{
		Name:       "dynamic-renderer-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
		},
		RenderFunc: func(c *gin.Context, base renderer.Universal) (renderer.Universal, error) {
			base.List = &renderer.ListPage{
				ID: "dynamic-" + c.Query("scope"),
				Layout: &renderer.Layout{
					Type: renderer.LayoutOneColumn,
					Gap:  renderer.SpacingLG,
				},
			}
			return base, nil
		},
		Navigation: []module.NavigationEntry{
			{ActionName: "list", Title: "Dynamic", Group: "Admin", Order: 1, Show: true, Path: "/admin/dynamic-renderer-items"},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Columns:    []pg.Column{id},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "List",
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()

	w := executeRequest(engine, http.MethodGet, "/admin/dynamic-renderer-items?scope=models", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var listResponse struct {
		Renderer *renderer.Identity `json:"renderer"`
		ListPage *renderer.ListPage `json:"list_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResponse))
	require.NotNil(t, listResponse.Renderer)
	require.NotNil(t, listResponse.ListPage)
	assert.Equal(t, "dynamic-models", listResponse.ListPage.ID)
	assert.Equal(t, renderer.SpacingLG, listResponse.ListPage.Layout.Gap)

	w = executeRequest(engine, http.MethodGet, "/api/config?scope=config", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var configResponse module.ConfigResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &configResponse))
	var entry *module.ConfigNavigationEntry
	for i := range configResponse.Navigation {
		if configResponse.Navigation[i].Path == "/admin/dynamic-renderer-items" {
			entry = &configResponse.Navigation[i]
			break
		}
	}
	require.NotNil(t, entry)
	require.NotNil(t, entry.Target.Renderer)
	assert.Equal(t, renderer.PageTypeList, entry.Target.PageType)
}

func TestUniversalRendererMetadata_RenderFuncResultIsValidated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := postgres.IntegerColumn("id")
	table := postgres.NewTable("public", "invalid_dynamic_renderer_items", "", id)

	testModule := &module.BaseModule{
		Name:       "invalid-dynamic-renderer-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
		},
		RenderFunc: func(_ *gin.Context, base renderer.Universal) (renderer.Universal, error) {
			base.List = &renderer.ListPage{ID: "invalid-dynamic-renderer-items"}
			base.ResourceGrid = &renderer.ResourceGridPage{Endpoint: "/invalid-dynamic-renderer-items"}
			return base, nil
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Columns:    []pg.Column{id},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "List",
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()

	w := executeRequest(engine, http.MethodGet, "/admin/invalid-dynamic-renderer-items", nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "renderer.Universal: List and ResourceGrid are mutually exclusive for one list route")
}

func TestUniversalRendererMetadata_RenderFuncDoesNotMutateBaseRender(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := postgres.IntegerColumn("id")
	table := postgres.NewTable("public", "isolated_renderer_items", "", id)

	baseRender := renderer.Universal{
		Form: &renderer.FormPage{
			ID: "isolated-renderer-items-form",
			Context: map[string]interface{}{
				"base":   true,
				"labels": []string{"base"},
				"items": []interface{}{
					map[string]interface{}{"id": "base"},
				},
				"nested": map[string]interface{}{
					"initial": true,
				},
			},
			Sections: []renderer.FormSection{
				{ID: "base", Title: "Base"},
			},
		},
	}

	testModule := &module.BaseModule{
		Name:       "isolated-renderer-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "ID", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
		},
		Render: baseRender,
		RenderFunc: func(c *gin.Context, base renderer.Universal) (renderer.Universal, error) {
			if c.Query("role") == "agency" {
				base.Form.Context["can_manage_models"] = true
				base.Form.Context["nested"].(map[string]interface{})["agency"] = true
				base.Form.Context["items"].([]interface{})[0].(map[string]interface{})["agency"] = true
				labels := base.Form.Context["labels"].([]string)
				labels[0] = "agency"
				base.Form.Context["labels"] = labels
				base.Form.Sections = append(base.Form.Sections, renderer.FormSection{ID: "agency", Title: "Agency"})
			}
			return base, nil
		},
		Actions: []actions.ModuleAction{
			actions.AddModuleAction{
				Columns:    []pg.Column{id},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				Label:      "Add",
			},
		},
	}

	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Run()

	w := executeRequest(engine, http.MethodGet, "/admin/isolated-renderer-items/defrec/?role=agency", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var agencyResponse struct {
		FormPage *renderer.FormPage `json:"form_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &agencyResponse))
	require.NotNil(t, agencyResponse.FormPage)
	assert.Equal(t, true, agencyResponse.FormPage.Context["can_manage_models"])
	assert.Equal(t, true, agencyResponse.FormPage.Context["nested"].(map[string]interface{})["agency"])
	assert.Equal(t, true, agencyResponse.FormPage.Context["items"].([]interface{})[0].(map[string]interface{})["agency"])
	assert.Equal(t, "agency", agencyResponse.FormPage.Context["labels"].([]interface{})[0])
	assert.Len(t, agencyResponse.FormPage.Sections, 2)

	w = executeRequest(engine, http.MethodGet, "/admin/isolated-renderer-items/defrec/?role=client", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var clientResponse struct {
		FormPage *renderer.FormPage `json:"form_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &clientResponse))
	require.NotNil(t, clientResponse.FormPage)
	assert.Equal(t, true, clientResponse.FormPage.Context["base"])
	assert.Equal(t, true, clientResponse.FormPage.Context["nested"].(map[string]interface{})["initial"])
	assert.NotContains(t, clientResponse.FormPage.Context["nested"].(map[string]interface{}), "agency")
	assert.NotContains(t, clientResponse.FormPage.Context["items"].([]interface{})[0].(map[string]interface{}), "agency")
	assert.Equal(t, "base", clientResponse.FormPage.Context["labels"].([]interface{})[0])
	assert.NotContains(t, clientResponse.FormPage.Context, "can_manage_models")
	assert.Len(t, clientResponse.FormPage.Sections, 1)

	assert.NotContains(t, testModule.Render.Form.Context, "can_manage_models")
	assert.NotContains(t, testModule.Render.Form.Context["nested"].(map[string]interface{}), "agency")
	assert.NotContains(t, testModule.Render.Form.Context["items"].([]interface{})[0].(map[string]interface{}), "agency")
	assert.Equal(t, "base", testModule.Render.Form.Context["labels"].([]string)[0])
	assert.Len(t, testModule.Render.Form.Sections, 1)
}
