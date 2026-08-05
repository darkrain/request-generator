package integration

import (
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
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestTypedOptionRenderersPreserveIconsAndLocalizedLabels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := pg.IntegerColumn("id")
	status := pg.StringColumn("status")
	categories := pg.StringColumn("categories")
	table := pg.NewTable("public", "typed_option_items", "", id, status, categories)
	testModule := &module.BaseModule{
		Name:       "typed-option-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{
				Column:       status,
				Title:        "items.fields.status",
				Type:         fields.ModuleFieldTypeString,
				FormType:     fields.ModuleFieldFormTypeSelect,
				Presentation: &renderer.FieldPresentation{Renderer: renderer.RendererPrimaryRadio},
				Options: []fields.ModuleFieldOptions{
					{Value: "active", Label: "items.options.active", Icon: "check"},
				},
			},
			{
				Column:       categories,
				Title:        "items.fields.categories",
				Type:         fields.ModuleFieldTypeArray,
				FormType:     fields.ModuleFieldFormTypeMultiselect,
				Presentation: &renderer.FieldPresentation{Renderer: renderer.RendererChipSelect},
				Options: []fields.ModuleFieldOptions{
					{Value: "example", Label: "items.options.example", Icon: "tag"},
				},
			},
		},
		Render: renderer.Universal{
			List:   &renderer.ListPage{ID: "typed-option-items"},
			Form:   &renderer.FormPage{ID: "typed-option-items-form"},
			Record: &renderer.RecordPage{ID: "typed-option-items-record"},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Columns: []pg.Column{id, status, categories}, Filter: []pg.Column{status}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
			actions.AddModuleAction{Columns: []pg.Column{status, categories}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
			actions.ViewModuleAction{Columns: []pg.Column{id, status, categories}, By: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
		},
	}

	translationsPath := filepath.Join(t.TempDir(), "en.json")
	require.NoError(t, os.WriteFile(translationsPath, []byte(`{"items":{"fields":{"id":"ID","status":"Status","categories":"Categories"},"options":{"active":"Active","example":"Example"}}}`), 0o600))

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
	generator.Locales = []locale.Lang{locale.EN}
	require.NoError(t, generator.LoadTranslationsFile(locale.EN, translationsPath))
	generator.Run()

	assertOptions := func(t *testing.T, options []fields.ModuleFieldOptions, label, icon string) {
		t.Helper()
		require.Len(t, options, 1)
		require.Equal(t, label, options[0].Label)
		require.Equal(t, icon, options[0].Icon)
	}

	w := executeRequest(engine, http.MethodGet, "/admin/typed-option-items?addFilters=true&lang=en", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var listResponse struct {
		Filters map[string]struct {
			Options []fields.ModuleFieldOptions `json:"options"`
		} `json:"filters"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResponse))
	assertOptions(t, listResponse.Filters["status"].Options, "Active", "check")

	w = executeRequest(engine, http.MethodGet, "/admin/typed-option-items/defrec/?lang=en", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var defrecResponse struct {
		Fields map[string]struct {
			Presentation *renderer.FieldPresentation `json:"presentation"`
			Options      []fields.ModuleFieldOptions `json:"options"`
		} `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &defrecResponse))
	require.Equal(t, renderer.RendererPrimaryRadio, defrecResponse.Fields["status"].Presentation.Renderer)
	assertOptions(t, defrecResponse.Fields["status"].Options, "Active", "check")
	require.Equal(t, renderer.RendererChipSelect, defrecResponse.Fields["categories"].Presentation.Renderer)
	assertOptions(t, defrecResponse.Fields["categories"].Options, "Example", "tag")

	w = executeRequest(engine, http.MethodGet, "/admin/typed-option-items/view/id/1?lang=en", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var viewResponse struct {
		Item map[string]struct {
			Presentation *renderer.FieldPresentation `json:"presentation"`
			Options      []fields.ModuleFieldOptions `json:"options"`
		} `json:"item"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &viewResponse))
	require.Equal(t, renderer.RendererPrimaryRadio, viewResponse.Item["status"].Presentation.Renderer)
	assertOptions(t, viewResponse.Item["status"].Options, "Active", "check")
	require.Equal(t, renderer.RendererChipSelect, viewResponse.Item["categories"].Presentation.Renderer)
	assertOptions(t, viewResponse.Item["categories"].Options, "Example", "tag")
}
