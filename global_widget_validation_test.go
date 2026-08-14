package module

import (
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

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
					Detail:    renderer.WorkspaceResource{Module: "detail_records", Action: "list", Bindings: []renderer.WidgetRequestBinding{{Target: renderer.WidgetRequestBindingQuery, Name: "filter[parent_id]", Value: "selection.id"}}},
				},
			},
		},
	}
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `widget "work-area" master resource references unknown module "unknown"`)

	modules = validGlobalWidgetModules()
	modules[2].Render.List.Actions[0].AfterSuccess.Widget.ID = "unknown"
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `module "workspace_entry" references unknown widget "unknown"`)
}

func validGlobalWidgetModules() []*BaseModule {
	id := pg.IntegerColumn("id")
	newModule := func(name string) *BaseModule {
		return &BaseModule{
			Name:    name,
			Path:    "/workspace",
			Fields:  []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView}},
			Render:  renderer.Universal{List: &renderer.ListPage{}},
			Actions: []actions.ModuleAction{actions.ListModuleAction{}},
		}
	}
	master := newModule("master_records")
	detail := newModule("detail_records")
	workspace := newModule("workspace_entry")
	workspace.Render.List.Actions = []renderer.Action{{
		ID:   "open",
		Type: renderer.ActionAPI,
		AfterSuccess: &renderer.ActionResult{Widget: &renderer.WidgetTarget{
			ID:             "work-area",
			State:          renderer.WidgetTargetOpen,
			SelectionField: "id",
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
					Detail:    renderer.WorkspaceResource{Module: "detail_records", Action: "list", Bindings: []renderer.WidgetRequestBinding{{Target: renderer.WidgetRequestBindingQuery, Name: "filter[parent_id]", Value: "selection.id"}}},
				},
			},
		},
	}
	return []*BaseModule{master, detail, workspace}
}
