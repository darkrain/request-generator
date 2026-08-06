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
				Column:   status,
				Title:    "items.fields.status",
				Type:     fields.ModuleFieldTypeString,
				FormType: fields.ModuleFieldFormTypeSelect,
				Presentation: &renderer.FieldPresentation{
					Renderer:    renderer.RendererPrimaryRadio,
					Prefix:      "items.status.prefix",
					Suffix:      "items.status.suffix",
					Hint:        "items.status.hint",
					Description: "items.status.description",
					VisibleIf:   &renderer.Condition{Path: "enabled", Equals: true},
					ToneByValue: []renderer.FieldValueTone{{Value: renderer.TypedValue{Type: renderer.TypedValueString, String: "active"}, Tone: "success"}},
				},
				OptionsSource: &fields.FieldOptionsSource{
					Endpoint:    "/admin/status-options",
					Query:       []fields.FieldOptionsQueryParam{{Key: "scope", Value: "profiles"}},
					SearchParam: "search",
					Mode:        fields.FieldOptionsSourceModeTree,
				},
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
			List: &renderer.ListPage{
				ID: "typed-option-items",
				Filters: &renderer.Filters{
					Primary: []string{"status", "managed_services", "rating"},
					RangePresets: []renderer.FilterRangePresets{{
						Field:   "rating",
						Presets: []renderer.FilterRangePreset{{Label: "items.rating.any", Min: 0, Max: 5}},
					}},
				},
			},
			Form:   &renderer.FormPage{ID: "typed-option-items-form"},
			Record: &renderer.RecordPage{ID: "typed-option-items-record"},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Columns:    []pg.Column{id, status, categories},
				Filter:     []pg.Column{status},
				Permission: []actions.Role{actions.RoleAll},
				Auth:       true,
				VirtualFilters: []fields.ModuleFilterField{{
					FieldName: "managed_services",
					Title:     "items.fields.managed_services",
					Type:      fields.ModuleFieldTypeArray,
					FormType:  fields.ModuleFieldFormTypeMultiselect,
					OptionsSource: &fields.FieldOptionsSource{
						Endpoint: "/admin/services",
					},
				}, {
					FieldName: "rating",
					Title:     "items.fields.rating",
					Type:      fields.ModuleFieldTypeFloat,
					FormType:  fields.ModuleFieldFormTypeNumber,
				}},
			},
			actions.AddModuleAction{Columns: []pg.Column{status, categories}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
			actions.ViewModuleAction{Columns: []pg.Column{id, status, categories}, By: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}, Auth: true},
		},
	}

	translationsPath := filepath.Join(t.TempDir(), "en.json")
	require.NoError(t, os.WriteFile(translationsPath, []byte(`{"items":{"fields":{"id":"ID","status":"Status","categories":"Categories","managed_services":"Managed services","rating":"Rating"},"options":{"active":"Active","example":"Example"},"status":{"prefix":"Current:","suffix":"state","hint":"Choose current state","description":"Controls item visibility"},"rating":{"any":"Any rating"}}}`), 0o600))

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
	require.NotContains(t, w.Body.String(), `"extra"`)
	var listResponse struct {
		Filters map[string]struct {
			Options       []fields.ModuleFieldOptions `json:"options"`
			OptionsSource *fields.FieldOptionsSource  `json:"options_source"`
		} `json:"filters"`
		ListPage *renderer.ListPage `json:"list_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listResponse))
	assertOptions(t, listResponse.Filters["status"].Options, "Active", "check")
	require.Equal(t, "/admin/status-options", listResponse.Filters["status"].OptionsSource.Endpoint)
	require.Equal(t, fields.FieldOptionsSourceModeTree, listResponse.Filters["status"].OptionsSource.Mode)
	require.Equal(t, "/admin/services", listResponse.Filters["managed_services"].OptionsSource.Endpoint)
	require.Equal(t, "Any rating", listResponse.ListPage.Filters.RangePresets[0].Presets[0].Label)

	w = executeRequest(engine, http.MethodGet, "/admin/typed-option-items/defrec/?lang=en", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), `"extra"`)
	var defrecResponse struct {
		Fields map[string]struct {
			Presentation  *renderer.FieldPresentation `json:"presentation"`
			Options       []fields.ModuleFieldOptions `json:"options"`
			OptionsSource *fields.FieldOptionsSource  `json:"options_source"`
		} `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &defrecResponse))
	require.Equal(t, renderer.RendererPrimaryRadio, defrecResponse.Fields["status"].Presentation.Renderer)
	require.Equal(t, "Current:", defrecResponse.Fields["status"].Presentation.Prefix)
	require.Equal(t, "state", defrecResponse.Fields["status"].Presentation.Suffix)
	require.Equal(t, "Choose current state", defrecResponse.Fields["status"].Presentation.Hint)
	require.Equal(t, "Controls item visibility", defrecResponse.Fields["status"].Presentation.Description)
	require.Equal(t, "/admin/status-options", defrecResponse.Fields["status"].OptionsSource.Endpoint)
	assertOptions(t, defrecResponse.Fields["status"].Options, "Active", "check")
	require.Equal(t, renderer.RendererChipSelect, defrecResponse.Fields["categories"].Presentation.Renderer)
	assertOptions(t, defrecResponse.Fields["categories"].Options, "Example", "tag")

	w = executeRequest(engine, http.MethodGet, "/admin/typed-option-items/view/id/1?lang=en", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), `"extra"`)
	var viewResponse struct {
		Item map[string]struct {
			Presentation  *renderer.FieldPresentation `json:"presentation"`
			Options       []fields.ModuleFieldOptions `json:"options"`
			OptionsSource *fields.FieldOptionsSource  `json:"options_source"`
		} `json:"item"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &viewResponse))
	require.Equal(t, renderer.RendererPrimaryRadio, viewResponse.Item["status"].Presentation.Renderer)
	require.Equal(t, "/admin/status-options", viewResponse.Item["status"].OptionsSource.Endpoint)
	assertOptions(t, viewResponse.Item["status"].Options, "Active", "check")
	require.Equal(t, renderer.RendererChipSelect, viewResponse.Item["categories"].Presentation.Renderer)
	assertOptions(t, viewResponse.Item["categories"].Options, "Example", "tag")
}

func TestListFilterContractRejectsUnavailableFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		primary        string
		virtualFilters []fields.ModuleFilterField
	}{
		{
			name:    "missing regular filter",
			primary: "missing",
		},
		{
			name:    "hidden virtual filter",
			primary: "managed_services",
			virtualFilters: []fields.ModuleFilterField{{
				FieldName:       "managed_services",
				Title:           "items.fields.managed_services",
				Type:            fields.ModuleFieldTypeArray,
				FormType:        fields.ModuleFieldFormTypeMultiselect,
				FilterCondition: func(_ *gin.Context) bool { return false },
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			id := pg.IntegerColumn("id")
			status := pg.StringColumn("status")
			table := pg.NewTable("public", "filter_contract_items", "", id, status)
			testModule := &module.BaseModule{
				Name:       "filter-contract-items",
				Path:       "/admin",
				Table:      table,
				PrimaryKey: id,
				Fields: []fields.ModuleField{
					{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
					{Column: status, Title: "items.fields.status", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeSelect},
				},
				Render: renderer.Universal{List: &renderer.ListPage{Filters: &renderer.Filters{Primary: []string{test.primary}}}},
				Actions: []actions.ModuleAction{actions.ListModuleAction{
					Columns:        []pg.Column{id, status},
					Filter:         []pg.Column{status},
					VirtualFilters: test.virtualFilters,
					Permission:     []actions.Role{actions.RoleAll},
					Auth:           true,
				}},
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

			w := executeRequest(engine, http.MethodGet, "/admin/filter-contract-items?addFilters=true", nil)
			require.Equal(t, http.StatusBadRequest, w.Code)
			var response struct {
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, `renderer filter "`+test.primary+`" is not available for the current request`, response.Message)
		})
	}
}
