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

func widgetInputSource(field string) renderer.WidgetValueSource {
	return renderer.WidgetValueSource{Runtime: &renderer.WidgetRuntimeValue{
		Scope: renderer.WidgetRuntimeValueSourceInput,
		Field: field,
	}}
}

func TestConfigEndpointSerializesTypedGlobalWidgets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("")
	id := pg.IntegerColumn("id")
	parentID := pg.IntegerColumn("parent_id")
	participantID := pg.IntegerColumn("participant_id")
	status := pg.StringColumn("status")
	enabled := pg.StringColumn("enabled")
	text := pg.StringColumn("text")

	master := &module.BaseModule{
		Name: "master_records",
		Path: "/workspace",
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: participantID, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: enabled, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeOnlyView},
		},
		Render: renderer.Universal{List: &renderer.ListPage{}},
		Actions: []actions.ModuleAction{actions.ListModuleAction{
			Label:      "Master records",
			Permission: []actions.Role{"member"},
			Auth:       true,
			Columns:    []pg.Column{id, participantID, enabled},
		}},
	}
	summary := &module.BaseModule{
		Name:   "summary_records",
		Path:   "/workspace",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView}},
		Render: renderer.Universal{List: &renderer.ListPage{}},
		Actions: []actions.ModuleAction{actions.ListModuleAction{
			Label:      "Summary records",
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
			{Column: text, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeTextArea},
		},
		Render: renderer.Universal{List: &renderer.ListPage{}},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Label: "Detail records", Permission: []actions.Role{"member"}, Auth: true, Filter: []pg.Column{parentID}},
			actions.AddModuleAction{Columns: []pg.Column{parentID, text}, Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
			actions.UpdateModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
		},
	}
	state := &module.BaseModule{
		Name: "state_records",
		Path: "/workspace",
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: status, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
		},
		Actions: []actions.ModuleAction{actions.UpdateModuleAction{
			Label:      "Set status",
			Permission: []actions.Role{"member"},
			Auth:       true,
			By:         []pg.Column{id},
			Columns:    []pg.Column{status},
		}, actions.DeleteModuleAction{
			Label:      "Delete status",
			Permission: []actions.Role{"admin"},
			Auth:       true,
			By:         []pg.Column{id},
		}},
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
					Source: renderer.ActionResultSource{
						Resource: renderer.ActionResource{Module: "workspace_entry", Action: "add"},
						Field:    renderer.ActionResultFieldValue,
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
							Summary:   &renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "summary_records", Action: "list"}},
							Master:    renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "master_records", Action: "list"}},
							Detail: renderer.WorkspaceResource{
								ActionResource: renderer.ActionResource{Module: "detail_records", Action: "list"},
								Bindings: []renderer.WidgetRequestBinding{{
									Target: renderer.WidgetRequestBindingFilter,
									Field:  "parent_id",
									Source: widgetSelectionSource("id"),
								}},
							},
							Commands: []renderer.WorkspaceCommand{{
								ID:    "set_status",
								Label: "workspace.command.set_status",
								Presentation: &renderer.ActionPresentation{
									Icon:      "ref-status",
									VisibleIf: &renderer.Condition{Path: "enabled", Equals: "yes"},
								},
								WorkspaceResource: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "state_records", Action: "update"}, Bindings: []renderer.WidgetRequestBinding{
									{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
									{Target: renderer.WidgetRequestBindingPathValue, Source: widgetSelectionSource("participant_id")},
									{Target: renderer.WidgetRequestBindingBody, Field: "status", Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "active"}}},
								}},
								Refresh: []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshMaster, renderer.WorkspaceRefreshDetail},
							}, {
								ID:    "create_detail",
								Label: "workspace.command.create_detail",
								Input: &renderer.WorkspaceCommandInput{Fields: []string{"text"}},
								WorkspaceResource: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "detail_records", Action: "add"}, Bindings: []renderer.WidgetRequestBinding{
									{Target: renderer.WidgetRequestBindingBody, Field: "parent_id", Source: widgetSelectionSource("id")},
									{Target: renderer.WidgetRequestBindingBody, Field: "text", Source: widgetInputSource("text")},
								}},
								Refresh: []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshMaster, renderer.WorkspaceRefreshDetail},
							}, {
								ID:    "delete_status",
								Label: "workspace.command.delete_status",
								WorkspaceResource: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "state_records", Action: "delete"}, Bindings: []renderer.WidgetRequestBinding{
									{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
									{Target: renderer.WidgetRequestBindingPathValue, Source: widgetSelectionSource("participant_id")},
								}},
								Refresh: []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshMaster},
							}},
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

	generator := module.NewGenerator(nil, *group, []*module.BaseModule{master, summary, detail, state, workspace, resource}, func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
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
	require.NotNil(t, workArea.Load.Summary)
	require.Equal(t, "/api/workspace/summary_records", workArea.Load.Summary.Request.Endpoint)
	require.Equal(t, "/api/workspace/master_records", workArea.Load.Master.Request.Endpoint)
	require.Equal(t, "/api/workspace/detail_records", workArea.Load.Detail.Request.Endpoint)
	require.Equal(t, renderer.WidgetRequestBindingFilter, workArea.Load.Detail.Bindings[0].Target)
	require.Equal(t, "parent_id", workArea.Load.Detail.Bindings[0].Field)
	require.Equal(t, "id", workArea.Load.Detail.Bindings[0].Source.Runtime.Field)
	require.Len(t, workArea.Widget.Workspace.Commands, 2)
	require.Equal(t, "workspace.command.set_status", workArea.Widget.Workspace.Commands[0].Label)
	require.Equal(t, "ref-status", workArea.Widget.Workspace.Commands[0].Presentation.Icon)
	require.Equal(t, "enabled", workArea.Widget.Workspace.Commands[0].Presentation.VisibleIf.Path)
	require.Len(t, workArea.Load.Commands, 2)
	require.Equal(t, "set_status", workArea.Load.Commands[0].ID)
	require.Equal(t, http.MethodPost, workArea.Load.Commands[0].Request.Method)
	require.Equal(t, "/api/workspace/state_records/:bykey/:value", workArea.Load.Commands[0].Request.Endpoint)
	require.Equal(t, renderer.WidgetRequestBindingPathValue, workArea.Load.Commands[0].Bindings[1].Target)
	require.Equal(t, "participant_id", workArea.Load.Commands[0].Bindings[1].Source.Runtime.Field)
	require.Equal(t, renderer.WidgetRequestBindingBody, workArea.Load.Commands[0].Bindings[2].Target)
	require.Equal(t, "status", workArea.Load.Commands[0].Bindings[2].Field)
	require.Equal(t, "create_detail", workArea.Load.Commands[1].ID)
	require.Equal(t, http.MethodPut, workArea.Load.Commands[1].Request.Method)
	require.Equal(t, "/api/workspace/detail_records", workArea.Load.Commands[1].Request.Endpoint)
	require.NotNil(t, workArea.Load.Commands[1].Input)
	require.Equal(t, http.MethodGet, workArea.Load.Commands[1].Input.Definition.Request.Method)
	require.Equal(t, "/api/workspace/detail_records/defrec/", workArea.Load.Commands[1].Input.Definition.Request.Endpoint)

	require.NotNil(t, resourceMenu)
	require.Nil(t, resourceMenu.Widget.Workspace)
	require.NotNil(t, resourceMenu.Load.Resource)
	require.Equal(t, "/api/shell/resource_entry/view/:bykey/:value", resourceMenu.Load.Resource.Request.Endpoint)
	require.Equal(t, renderer.WidgetRequestBindingPathValue, resourceMenu.Load.Resource.Bindings[1].Target)
	require.Equal(t, renderer.WidgetRuntimeValueSourceCurrentUser, resourceMenu.Load.Resource.Bindings[1].Source.Runtime.Scope)
	require.NotContains(t, response.Body.String(), `"config"`)
	require.NotContains(t, response.Body.String(), `"params"`)
}
