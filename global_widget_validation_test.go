package module

import (
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func selectionSource(field string) renderer.WidgetValueSource {
	return renderer.WidgetValueSource{Runtime: &renderer.WidgetRuntimeValue{
		Scope: renderer.WidgetRuntimeValueSourceSelection,
		Field: field,
	}}
}

func TestValidateGlobalWidgets(t *testing.T) {
	modules := validGlobalWidgetModules()
	generator := &Generator{Modules: modules}
	require.NoError(t, generator.validateGlobalWidgets())

	modules = validGlobalWidgetModules()
	modules[2].Actions[0] = actions.ListModuleAction{
		Widget: &actions.WidgetConfig{
			ID: "work-area",
			Renderer: renderer.GlobalWidget{
				Surface: renderer.WidgetSurface{Kind: renderer.WidgetSurfaceDrawer, Placement: renderer.WidgetPlacementShellEnd, LoadPolicy: renderer.WidgetLoadOnOpen},
				Workspace: &renderer.WorkspaceWidget{
					Selection: renderer.WorkspaceSelection{Field: "id"},
					Master:    renderer.WorkspaceResource{Module: "unknown", Action: "list"},
					Detail: renderer.WorkspaceResource{Module: "detail_records", Action: "list", Bindings: []renderer.WidgetRequestBinding{{
						Target: renderer.WidgetRequestBindingFilter,
						Field:  "parent_id",
						Source: selectionSource("id"),
					}}},
				},
			},
		},
	}
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `widget "work-area" master resource references unknown module "unknown"`)

	modules = validGlobalWidgetModules()
	modules[2].Render.List.Actions[0].AfterSuccess.Widget.ID = "unknown"
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `module "workspace_entry" references unknown widget "unknown"`)
}

func TestValidateGlobalWidgetsRejectsUnresolvedSourcesAndBindings(t *testing.T) {
	tests := []struct {
		name string
		edit func([]*BaseModule)
		err  string
	}{
		{
			name: "selection source typo",
			edit: func(modules []*BaseModule) {
				modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Detail.Bindings[0].Source.Runtime.Field = "selecion_id"
			},
			err: `module "workspace_entry" action "list": renderer.GlobalWidget: workspace: detail must bind selection field "id"`,
		},
		{
			name: "event correlation typo",
			edit: func(modules []*BaseModule) {
				modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Subscriptions[0].Correlation.EventField = "unknown"
			},
			err: `widget "work-area" subscription "detail_records" action "add" correlation field "unknown" does not match declared field "parent_id"`,
		},
		{
			name: "action result source typo",
			edit: func(modules []*BaseModule) {
				modules[2].Render.List.Actions[0].AfterSuccess.Widget.Selection.Source.Field = "unknown"
			},
			err: `module "workspace_entry" action target "work-area" selection source resource "workspace_entry" action "add": response field "unknown" is not declared`,
		},
		{
			name: "action request does not match source resource",
			edit: func(modules []*BaseModule) {
				modules[2].Render.List.Actions[0].API.Endpoint = "/api/wrong"
			},
			err: `module "workspace_entry" action target "work-area" selection source resource "workspace_entry" action "add" does not match action request PUT /api/workspace/workspace_entry`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modules := validGlobalWidgetModules()
			test.edit(modules)
			require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), test.err)
		})
	}
}

func TestValidateWidgetRequestBindingShapeRequiresCompleteViewPath(t *testing.T) {
	id := pg.IntegerColumn("id")
	module := &BaseModule{
		Name:   "records",
		Path:   "/records",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
	}
	action := actions.ViewModuleAction{By: []pg.Column{id}}
	err := validateWidgetRequestBindingShape(module, action, []renderer.WidgetRequestBinding{{
		Target: renderer.WidgetRequestBindingPathByKey,
		Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{
			Type:   renderer.TypedValueString,
			String: "id",
		}},
	}})
	require.EqualError(t, err, "view action requires path_by_key and path_value bindings")
}

func TestValidateWidgetRequestBindingAvailabilityRejectsUnknownRuntimeSourceField(t *testing.T) {
	id := pg.IntegerColumn("id")
	module := &BaseModule{
		Name:   "records",
		Path:   "/records",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
	}
	action := actions.ViewModuleAction{By: []pg.Column{id}}
	context, _ := ginTestContext()
	available, err := (&Generator{}).validateWidgetRequestBindingAvailability(context, module, action, []renderer.WidgetRequestBinding{
		{
			Target: renderer.WidgetRequestBindingPathByKey,
			Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{
				Type:   renderer.TypedValueString,
				String: "id",
			}},
		},
		{
			Target: renderer.WidgetRequestBindingPathValue,
			Source: renderer.WidgetValueSource{Runtime: &renderer.WidgetRuntimeValue{
				Scope: renderer.WidgetRuntimeValueSourceCurrentUser,
				Field: "subject",
			}},
		},
	}, nil)
	require.False(t, available)
	require.EqualError(t, err, `current_user field "subject" is not declared`)
}

func TestWidgetRequestBindingAvailabilityUsesEffectiveListFilters(t *testing.T) {
	id := pg.IntegerColumn("id")
	parentID := pg.IntegerColumn("parent_id")
	module := &BaseModule{
		Name: "records",
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt},
			{Column: parentID, Type: fields.ModuleFieldTypeInt, FilterCondition: func(*gin.Context) bool { return false }},
		},
	}
	binding := renderer.WidgetRequestBinding{
		Target: renderer.WidgetRequestBindingFilter,
		Field:  "parent_id",
		Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 1}},
	}
	context, _ := ginTestContext()
	available, err := (&Generator{}).validateWidgetRequestBindingAvailability(context, module, actions.ListModuleAction{Filter: []pg.Column{parentID}}, []renderer.WidgetRequestBinding{binding}, nil)
	require.NoError(t, err)
	require.False(t, available)

	module.Fields[1].FilterCondition = nil
	available, err = (&Generator{}).validateWidgetRequestBindingAvailability(context, module, actions.ListModuleAction{FilterFunc: func(*gin.Context) []pg.Column { return []pg.Column{parentID} }}, []renderer.WidgetRequestBinding{binding}, nil)
	require.NoError(t, err)
	require.True(t, available)

	module.Path = "/records"
	module.Render = renderer.Universal{List: &renderer.ListPage{}}
	module.Actions = []actions.ModuleAction{actions.ListModuleAction{
		FilterFunc: func(*gin.Context) []pg.Column { return []pg.Column{parentID} },
		Widget: &actions.WidgetConfig{
			ID:       "records-filter",
			Renderer: renderer.GlobalWidget{Surface: renderer.WidgetSurface{Kind: renderer.WidgetSurfaceDrawer, Placement: renderer.WidgetPlacementShellStart, LoadPolicy: renderer.WidgetLoadOnOpen}},
			Bindings: []renderer.WidgetRequestBinding{binding},
		},
	}}
	require.NoError(t, (&Generator{Modules: []*BaseModule{module}}).validateGlobalWidgets())

	module.Fields[1].FilterCondition = func(*gin.Context) bool { return false }
	load, available, err := (&Generator{}).buildWidgetLoad(context, module, module.Actions[0], *actionWidget(module.Actions[0]), "")
	require.NoError(t, err)
	require.False(t, available)
	require.Equal(t, renderer.WidgetLoad{}, load)
}

func ginTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest("GET", "/", nil)
	return context, response
}

func TestWidgetConfigRejectsIgnoredWorkspaceBindings(t *testing.T) {
	config := actions.WidgetConfig{
		ID: "work-area",
		Renderer: renderer.GlobalWidget{
			Surface: renderer.WidgetSurface{Kind: renderer.WidgetSurfaceDrawer, Placement: renderer.WidgetPlacementShellEnd, LoadPolicy: renderer.WidgetLoadOnOpen},
			Workspace: &renderer.WorkspaceWidget{
				Selection: renderer.WorkspaceSelection{Field: "id"},
				Master:    renderer.WorkspaceResource{Module: "master", Action: "list"},
				Detail: renderer.WorkspaceResource{Module: "detail", Action: "list", Bindings: []renderer.WidgetRequestBinding{{
					Target: renderer.WidgetRequestBindingFilter,
					Field:  "parent_id",
					Source: selectionSource("id"),
				}}},
			},
		},
		Bindings: []renderer.WidgetRequestBinding{{
			Target: renderer.WidgetRequestBindingFilter,
			Field:  "ignored",
			Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{
				Type:   renderer.TypedValueString,
				String: "value",
			}},
		}},
	}
	require.EqualError(t, config.Validate(), "workspace widget bindings must be declared by a workspace resource")
}

func validGlobalWidgetModules() []*BaseModule {
	id := pg.IntegerColumn("id")
	parentID := pg.IntegerColumn("parent_id")
	newModule := func(name string, moduleFields []fields.ModuleField, moduleActions []actions.ModuleAction) *BaseModule {
		return &BaseModule{
			Name:    name,
			Path:    "/workspace",
			Fields:  moduleFields,
			Render:  renderer.Universal{List: &renderer.ListPage{}},
			Actions: moduleActions,
		}
	}
	master := newModule("master_records", []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}}, []actions.ModuleAction{actions.ListModuleAction{}})
	detail := newModule("detail_records", []fields.ModuleField{
		{Column: id, Type: fields.ModuleFieldTypeInt},
		{Column: parentID, Type: fields.ModuleFieldTypeInt},
	}, []actions.ModuleAction{
		actions.ListModuleAction{Filter: []pg.Column{parentID}},
		actions.AddModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
		actions.UpdateModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
	})
	workspace := newModule("workspace_entry", []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}}, []actions.ModuleAction{actions.ListModuleAction{}, actions.AddModuleAction{}})
	workspace.Render.List.Actions = []renderer.Action{{
		ID:   "open",
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
		}},
	}}
	workspace.Actions[0] = actions.ListModuleAction{
		Widget: &actions.WidgetConfig{
			ID: "work-area",
			Renderer: renderer.GlobalWidget{
				Surface: renderer.WidgetSurface{Kind: renderer.WidgetSurfaceDrawer, Placement: renderer.WidgetPlacementShellEnd, LoadPolicy: renderer.WidgetLoadOnOpen},
				Workspace: &renderer.WorkspaceWidget{
					Selection: renderer.WorkspaceSelection{Field: "id"},
					Master:    renderer.WorkspaceResource{Module: "master_records", Action: "list"},
					Detail: renderer.WorkspaceResource{Module: "detail_records", Action: "list", Bindings: []renderer.WidgetRequestBinding{{
						Target: renderer.WidgetRequestBindingFilter,
						Field:  "parent_id",
						Source: selectionSource("id"),
					}}},
					Subscriptions: []renderer.WorkspaceSubscription{{
						Module:      "detail_records",
						Actions:     []string{"add", "update"},
						Correlation: renderer.WorkspaceCorrelationBinding{EventField: "parent_id"},
						Refresh:     []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshMaster, renderer.WorkspaceRefreshDetail},
					}},
				},
			},
		},
	}
	return []*BaseModule{master, detail, workspace}
}
