package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func widgetSelectionSource(field string) renderer.WidgetValueSource {
	return renderer.WidgetValueSource{Runtime: &renderer.WidgetRuntimeValue{
		Scope: renderer.WidgetRuntimeValueSourceSelection,
		Field: field,
	}}
}

func TestConfigEndpointSerializesTypedGlobalWidgets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("")
	id := pg.IntegerColumn("id")
	parentID := pg.IntegerColumn("parent_id")

	master := &module.BaseModule{
		Name:   "master_records",
		Path:   "/workspace",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView}},
		Render: renderer.Universal{List: &renderer.ListPage{}},
		Actions: []actions.ModuleAction{actions.ListModuleAction{
			Label:      "Master records",
			Permission: []actions.Role{"member"},
			Auth:       true,
		}},
	}
	detail := &module.BaseModule{
		Name: "detail_records",
		Path: "/workspace",
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: parentID, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
		},
		Render: renderer.Universal{List: &renderer.ListPage{}},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Label: "Detail records", Permission: []actions.Role{"member"}, Auth: true, Filter: []pg.Column{parentID}},
			actions.AddModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
			actions.UpdateModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
		},
	}
	workspace := &module.BaseModule{
		Name:   "workspace_entry",
		Path:   "/workspace",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView}},
		Render: renderer.Universal{List: &renderer.ListPage{Actions: []renderer.Action{{
			ID:   "open_workspace",
			Type: renderer.ActionAPI,
			API:  &renderer.APIAction{Method: "PUT", Endpoint: "/api/workspace/workspace_entry"},
			AfterSuccess: &renderer.ActionResult{Widget: &renderer.WidgetTarget{
				ID:    "work-area",
				State: renderer.WidgetTargetOpen,
				Selection: &renderer.WidgetSelectionResultBinding{
					Source: renderer.WidgetActionResultSource{
						Resource: renderer.WidgetActionResultResource{Module: "workspace_entry", Action: "add"},
						Field:    "value",
					},
				},
				Refresh: []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshDetail},
			}},
		}}}},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{
				Label:      "Workspace entry",
				Permission: []actions.Role{"member"},
				Auth:       true,
				Widget: &actions.WidgetConfig{
					ID: "work-area",
					Renderer: renderer.GlobalWidget{
						Surface: renderer.WidgetSurface{
							Kind:       renderer.WidgetSurfaceDrawer,
							Placement:  renderer.WidgetPlacementShellEnd,
							LoadPolicy: renderer.WidgetLoadOnOpen,
						},
						Workspace: &renderer.WorkspaceWidget{
							Selection: renderer.WorkspaceSelection{Field: "id"},
							Master:    renderer.WorkspaceResource{Module: "master_records", Action: "list"},
							Detail: renderer.WorkspaceResource{
								Module: "detail_records",
								Action: "list",
								Bindings: []renderer.WidgetRequestBinding{{
									Target: renderer.WidgetRequestBindingFilter,
									Field:  "parent_id",
									Source: widgetSelectionSource("id"),
								}},
							},
							Subscriptions: []renderer.WorkspaceSubscription{{
								Module:      "detail_records",
								Actions:     []string{"add", "update"},
								Correlation: renderer.WorkspaceCorrelationBinding{EventField: "parent_id"},
								Refresh:     []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshMaster, renderer.WorkspaceRefreshDetail},
							}},
						},
					},
				},
			},
			actions.AddModuleAction{Label: "Add workspace entry", Permission: []actions.Role{"member"}, Auth: true},
		},
	}
	resource := &module.BaseModule{
		Name:   "resource_entry",
		Path:   "/shell",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView}},
		Render: renderer.Universal{Record: &renderer.RecordPage{}},
		Actions: []actions.ModuleAction{actions.ViewModuleAction{
			Label:      "Resource entry",
			Permission: []actions.Role{"member"},
			Auth:       true,
			By:         []pg.Column{id},
			Widget: &actions.WidgetConfig{
				ID: "resource-menu",
				Renderer: renderer.GlobalWidget{Surface: renderer.WidgetSurface{
					Kind:       renderer.WidgetSurfacePopup,
					Placement:  renderer.WidgetPlacementShellEnd,
					LoadPolicy: renderer.WidgetLoadOnOpen,
				}},
				Bindings: []renderer.WidgetRequestBinding{
					{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
					{Target: renderer.WidgetRequestBindingPathValue, Source: renderer.WidgetValueSource{Runtime: &renderer.WidgetRuntimeValue{Scope: renderer.WidgetRuntimeValueSourceCurrentUser, Field: "id"}}},
				},
			},
		}},
	}

	generator := module.NewGenerator(nil, *group, []*module.BaseModule{master, detail, workspace, resource}, func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}, func(_ actions.ModuleAction) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Request = c.Request.WithContext(icontext.SetUser(c.Request.Context(), &icontext.UserInfo{ID: 1, Role: "member"}))
			c.Next()
		}
	})
	generator.Run()

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	type configWidget struct {
		ID       string                `json:"id"`
		Renderer renderer.Identity     `json:"renderer"`
		Widget   renderer.GlobalWidget `json:"widget"`
		Load     renderer.WidgetLoad   `json:"load"`
	}
	var config struct {
		Widgets []configWidget `json:"widgets"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &config))
	require.Len(t, config.Widgets, 2)

	var workArea, resourceMenu *configWidget
	for index := range config.Widgets {
		widget := &config.Widgets[index]
		switch widget.ID {
		case "work-area":
			workArea = widget
		case "resource-menu":
			resourceMenu = widget
		}
	}
	require.NotNil(t, workArea)
	require.Equal(t, renderer.UniversalIdentity(), workArea.Renderer)
	require.NotNil(t, workArea.Widget.Workspace)
	require.Nil(t, workArea.Load.Resource)
	require.Equal(t, "/api/workspace/master_records", workArea.Load.Master.Request.Endpoint)
	require.Equal(t, "/api/workspace/detail_records", workArea.Load.Detail.Request.Endpoint)
	require.Equal(t, renderer.WidgetRequestBindingFilter, workArea.Load.Detail.Bindings[0].Target)
	require.Equal(t, "parent_id", workArea.Load.Detail.Bindings[0].Field)
	require.Equal(t, "id", workArea.Load.Detail.Bindings[0].Source.Runtime.Field)

	require.NotNil(t, resourceMenu)
	require.Nil(t, resourceMenu.Widget.Workspace)
	require.NotNil(t, resourceMenu.Load.Resource)
	require.Equal(t, "/api/shell/resource_entry/view/:bykey/:value", resourceMenu.Load.Resource.Request.Endpoint)
	require.Equal(t, renderer.WidgetRequestBindingPathValue, resourceMenu.Load.Resource.Bindings[1].Target)
	require.Equal(t, renderer.WidgetRuntimeValueSourceCurrentUser, resourceMenu.Load.Resource.Bindings[1].Source.Runtime.Scope)
	require.NotContains(t, response.Body.String(), `"config"`)
	require.NotContains(t, response.Body.String(), `"params"`)
}
