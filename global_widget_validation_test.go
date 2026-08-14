package module

import (
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
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
			err: `widget "work-area" detail resource: filter field "parent_id": selection field "selecion_id" does not match declared field "id"`,
		},
		{
			name: "filter field typo",
			edit: func(modules []*BaseModule) {
				modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Detail.Bindings[0].Field = "unknown"
			},
			err: `widget "work-area" detail resource: filter field "unknown" is not declared by list action`,
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
				modules[2].Render.List.Actions[0].AfterSuccess.Widget.Selection.SourceField = "unknown"
			},
			err: `module "workspace_entry" action target "work-area" selection source field "unknown" is not returned by the action resource`,
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

func TestValidateWidgetRequestBindingsRequiresCompleteViewPath(t *testing.T) {
	id := pg.IntegerColumn("id")
	module := &BaseModule{
		Name:   "records",
		Path:   "/records",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
	}
	action := actions.ViewModuleAction{By: []pg.Column{id}}
	err := validateWidgetRequestBindings(module, action, []renderer.WidgetRequestBinding{{
		Target: renderer.WidgetRequestBindingPathByKey,
		Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{
			Type:   renderer.TypedValueString,
			String: "id",
		}},
	}}, nil)
	require.EqualError(t, err, "view action requires path_by_key and path_value bindings")
}

func TestValidateWidgetRequestBindingsRejectsUnknownRuntimeSourceField(t *testing.T) {
	id := pg.IntegerColumn("id")
	module := &BaseModule{
		Name:   "records",
		Path:   "/records",
		Fields: []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
	}
	action := actions.ViewModuleAction{By: []pg.Column{id}}
	err := validateWidgetRequestBindings(module, action, []renderer.WidgetRequestBinding{
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
	require.EqualError(t, err, `current_user field "subject" is not declared`)
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
	relatedID := pg.IntegerColumn("related_id")
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
	workspace := newModule("workspace_entry", []fields.ModuleField{
		{Column: id, Type: fields.ModuleFieldTypeInt},
		{Column: relatedID, Type: fields.ModuleFieldTypeInt},
	}, []actions.ModuleAction{actions.ListModuleAction{}})
	workspace.Render.List.Actions = []renderer.Action{{
		ID:   "open",
		Type: renderer.ActionAPI,
		AfterSuccess: &renderer.ActionResult{Widget: &renderer.WidgetTarget{
			ID:    "work-area",
			State: renderer.WidgetTargetOpen,
			Selection: &renderer.WidgetSelectionResultBinding{
				SourceField: "related_id",
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
