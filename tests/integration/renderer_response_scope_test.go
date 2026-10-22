package integration

import (
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
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestResponseResolvesOnlyEmittedPageResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := pg.IntegerColumn("id")
	table := pg.NewTable("public", "entries", "", id)
	calls := map[string]int{}
	resource := func(name string) *renderer.Resource {
		r := &renderer.Resource{ActionResource: renderer.ActionResource{Module: name, Action: "list"}}
		if name == "form-entries" {
			r.Action = "view"
			r.Bindings = []renderer.RequestBinding{
				{Target: renderer.RequestBindingPathByKey, Source: renderer.ValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
				{Target: renderer.RequestBindingPathValue, Source: renderer.ValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 1}}},
			}
		}
		return r
	}
	parent := &module.BaseModule{Name: "parent", Path: "/api", Table: table, PrimaryKey: id,
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Columns: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}},
			actions.AddModuleAction{Columns: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}},
			actions.ViewModuleAction{Columns: []pg.Column{id}, By: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}, PageTypeFunc: func(c *gin.Context) renderer.PageType {
				if c.Query("form") == "1" {
					return renderer.PageTypeForm
				}
				return renderer.PageTypeRecord
			}},
		},
		Defrec: actions.DefrecModuleAction{Permission: []actions.Role{actions.RoleAll}},
		Render: renderer.Universal{
			List:   &renderer.ListPage{ID: "parent-list"},
			Form:   &renderer.FormPage{ID: "parent-form", Sections: []renderer.FormSection{{ID: "form-child", Renderer: renderer.RendererUniversalSection, Resource: resource("form-entries")}}},
			Record: &renderer.RecordPage{ID: "parent-record", Sections: []renderer.RecordSection{{ID: "record-child", Renderer: renderer.RendererUniversalSection, Resource: resource("record-entries")}}},
		},
	}
	modules := []*module.BaseModule{parent}
	for _, name := range []string{"form-entries", "record-entries"} {
		name := name
		modules = append(modules, &module.BaseModule{Name: name, Path: "/api", Table: table, PrimaryKey: id,
			Fields:  []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
			Actions: []actions.ModuleAction{actions.ListModuleAction{Columns: []pg.Column{id}, Permission: []actions.Role{actions.RoleAll}}, actions.ViewModuleAction{Columns: []pg.Column{id}, By: []pg.Column{id}, PageType: renderer.PageTypeForm, Permission: []actions.Role{actions.RoleAll}}},
			Render:  renderer.Universal{List: &renderer.ListPage{ID: name}, Form: &renderer.FormPage{ID: name}},
			RenderFunc: func(_ *gin.Context, base renderer.Universal) (renderer.Universal, error) {
				calls[name]++
				return base, nil
			},
		})
	}
	engine := gin.New()
	g := module.NewGenerator(func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} }, *engine.Group(""), modules,
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}))
	g.Run()
	for _, tc := range []struct{ path, page, child string }{
		{"/api/parent", "list_page", ""},
		{"/api/parent/defrec/", "form_page", "form-entries"},
		{"/api/parent/view/id/1?form=1", "form_page", "form-entries"},
		{"/api/parent/view/id/1", "record_page", "record-entries"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			clear(calls)
			w := executeRequest(engine, http.MethodGet, tc.path, nil)
			require.Equal(t, 200, w.Code, w.Body.String())
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Contains(t, body, tc.page)
			for _, page := range []string{"list_page", "form_page", "record_page"} {
				if page != tc.page {
					require.NotContains(t, body, page)
				}
			}
			for _, name := range []string{"form-entries", "record-entries"} {
				if name == tc.child {
					require.Positive(t, calls[name])
					require.Contains(t, w.Body.String(), "/api/"+name)
				} else {
					require.Zero(t, calls[name])
				}
			}
		})
	}
	require.Nil(t, parent.Render.Form.Sections[0].Load)
	require.Nil(t, parent.Render.Record.Sections[0].Load)
	// Dropped metadata must still be validated before response projection.
	parent.RenderFunc = func(_ *gin.Context, base renderer.Universal) (renderer.Universal, error) {
		base.Record.Sections[0].Resource.Action = "delete"
		return base, nil
	}
	w := executeRequest(engine, http.MethodGet, "/api/parent", nil)
	require.Equal(t, 400, w.Code)
}
