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

func inputSource(field string) renderer.WidgetValueSource {
	return renderer.WidgetValueSource{Runtime: &renderer.WidgetRuntimeValue{
		Scope: renderer.WidgetRuntimeValueSourceInput,
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
					Master:    renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "unknown", Action: "list"}},
					Detail: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "detail_records", Action: "list"}, Bindings: []renderer.WidgetRequestBinding{{
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

	modules = validGlobalWidgetModules()
	modules[3].Render.Form = &renderer.FormPage{}
	modules[3].Actions = []actions.ModuleAction{actions.DefrecModuleAction{}}
	modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Summary.Action = "defrec"
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `widget "work-area" summary action must be list or view`)

	modules = validGlobalWidgetModules()
	modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Commands[0].Module = "master_records"
	modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Commands[0].Action = "list"
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `widget "work-area" command "set_status" action must be add, update or delete`)
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

func TestValidateWorkspaceCommandPresentation(t *testing.T) {
	enabled := pg.StringColumn("enabled")
	modules := validGlobalWidgetModules()
	modules[0].Fields = append(modules[0].Fields, fields.ModuleField{Column: enabled, Type: fields.ModuleFieldTypeString})
	command := &modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace.Commands[0]
	command.Presentation = &renderer.ActionPresentation{
		Active:    "enabled",
		VisibleIf: &renderer.Condition{Path: "enabled", Equals: "yes"},
	}
	require.NoError(t, (&Generator{Modules: modules}).validateGlobalWidgets())

	command.Presentation.Active = "missing"
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `widget "work-area" command "set_status" presentation: active field "missing" is not declared by master resource`)

	command.Presentation.Active = ""
	command.Presentation.VisibleIf.Path = "missing"
	require.EqualError(t, (&Generator{Modules: modules}).validateGlobalWidgets(), `widget "work-area" command "set_status" presentation: condition field "missing" is not declared by master resource`)
}

func TestValidateWorkspaceCommandInput(t *testing.T) {
	id := pg.IntegerColumn("id")
	text := pg.StringColumn("text")
	note := pg.StringColumn("note")
	hidden := pg.StringColumn("hidden")
	module := &BaseModule{
		Name: "entries",
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: text, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeTextArea},
			{Column: note, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
			{Column: hidden, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeHidden},
		},
		Actions: []actions.ModuleAction{
			actions.AddModuleAction{Columns: []pg.Column{text, hidden}},
			actions.UpdateModuleAction{By: []pg.Column{id}, Columns: []pg.Column{hidden}},
		},
	}
	command := renderer.WorkspaceCommand{
		ID:    "create",
		Label: "workspace.command.create",
		Input: &renderer.WorkspaceCommandInput{Fields: []string{"text"}},
		WorkspaceResource: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "entries", Action: "add"}, Bindings: []renderer.WidgetRequestBinding{{
			Target: renderer.WidgetRequestBindingBody,
			Field:  "text",
			Source: inputSource("text"),
		}}},
		Refresh: []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshDetail},
	}
	selection := widgetSelectionScope{Fields: map[string]renderer.TypedValueType{"id": renderer.TypedValueNumber}}
	generator := &Generator{Modules: []*BaseModule{module}}
	require.NoError(t, command.Validate())
	require.NoError(t, generator.validateWorkspaceCommand("workspace", command, selection))

	command.Input = &renderer.WorkspaceCommandInput{Fields: []string{"note"}}
	command.Bindings[0].Field = "note"
	command.Bindings[0].Source = inputSource("note")
	require.EqualError(t, generator.validateWorkspaceCommand("workspace", command, selection), `widget "workspace" command "create" input: field "note" is not declared by add action`)

	command.Input = &renderer.WorkspaceCommandInput{Fields: []string{"hidden"}}
	command.Bindings[0].Field = "hidden"
	command.Bindings[0].Source = inputSource("hidden")
	require.EqualError(t, generator.validateWorkspaceCommand("workspace", command, selection), `widget "workspace" command "create" input: field "hidden" is not editable`)

	command.Module = "entries"
	command.Action = "update"
	require.EqualError(t, generator.validateWorkspaceCommand("workspace", command, selection), `widget "workspace" command "create" input: is only supported by add action`)
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

func TestValidateWidgetRequestBindingShapeForUpdateCommand(t *testing.T) {
	id := pg.IntegerColumn("id")
	status := pg.StringColumn("status")
	module := &BaseModule{
		Name: "state_records",
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt},
			{Column: status, Type: fields.ModuleFieldTypeString},
		},
	}
	action := actions.UpdateModuleAction{By: []pg.Column{id}, Columns: []pg.Column{status}}
	err := validateWidgetRequestBindingShape(module, action, []renderer.WidgetRequestBinding{
		{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
		{Target: renderer.WidgetRequestBindingPathValue, Source: selectionSource("participant_id")},
		{Target: renderer.WidgetRequestBindingBody, Field: "status", Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "active"}}},
	})
	require.NoError(t, err)

	err = validateWidgetRequestBindingShape(module, action, []renderer.WidgetRequestBinding{
		{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
		{Target: renderer.WidgetRequestBindingPathValue, Source: selectionSource("participant_id")},
	})
	require.EqualError(t, err, "update action requires body bindings")

	err = validateWidgetRequestBindingShape(module, action, []renderer.WidgetRequestBinding{
		{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
		{Target: renderer.WidgetRequestBindingPathValue, Source: selectionSource("participant_id")},
		{Target: renderer.WidgetRequestBindingBody, Field: "status", Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 1}}},
	})
	require.EqualError(t, err, `body field "status" literal type "number" does not match expected type "string"`)

	context, _ := ginTestContext()
	available, err := (&Generator{}).validateWidgetRequestBindingAvailability(context, module, action, []renderer.WidgetRequestBinding{
		{Target: renderer.WidgetRequestBindingBody, Field: "status", Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "active"}}},
		{Target: renderer.WidgetRequestBindingPathValue, Source: selectionSource("participant_id")},
		{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
	}, &widgetSelectionScope{Fields: map[string]renderer.TypedValueType{"participant_id": renderer.TypedValueNumber}}, nil)
	require.NoError(t, err)
	require.True(t, available)
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
	}, nil, nil)
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
	available, err := (&Generator{}).validateWidgetRequestBindingAvailability(context, module, actions.ListModuleAction{Filter: []pg.Column{parentID}}, []renderer.WidgetRequestBinding{binding}, nil, nil)
	require.NoError(t, err)
	require.False(t, available)

	module.Fields[1].FilterCondition = nil
	available, err = (&Generator{}).validateWidgetRequestBindingAvailability(context, module, actions.ListModuleAction{FilterFunc: func(*gin.Context) []pg.Column { return []pg.Column{parentID} }}, []renderer.WidgetRequestBinding{binding}, nil, nil)
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

func TestWorkspaceSelectionScopeRequiresReturnedMasterFields(t *testing.T) {
	modules := validGlobalWidgetModules()
	workspace := modules[2].Actions[0].(actions.ListModuleAction).Widget.Renderer.Workspace
	context, _ := ginTestContext()
	generator := &Generator{Modules: modules}

	_, err := generator.workspaceSelectionScopeForContext(context, "work-area", *workspace)
	require.EqualError(t, err, `widget "work-area" selection field "id" is not returned by master action`)

	masterAction := modules[0].Actions[0].(actions.ListModuleAction)
	masterAction.Columns = []pg.Column{modules[0].Fields[0].Column, modules[0].Fields[1].Column}
	modules[0].Actions[0] = masterAction
	scope, err := generator.workspaceSelectionScopeForContext(context, "work-area", *workspace)
	require.NoError(t, err)
	require.Equal(t, renderer.TypedValueNumber, scope.Fields["participant_id"])
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
				Master:    renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "master", Action: "list"}},
				Detail: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "detail", Action: "list"}, Bindings: []renderer.WidgetRequestBinding{{
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
	participantID := pg.IntegerColumn("participant_id")
	status := pg.StringColumn("status")
	newModule := func(name string, moduleFields []fields.ModuleField, moduleActions []actions.ModuleAction) *BaseModule {
		return &BaseModule{
			Name:    name,
			Path:    "/workspace",
			Fields:  moduleFields,
			Render:  renderer.Universal{List: &renderer.ListPage{}},
			Actions: moduleActions,
		}
	}
	master := newModule("master_records", []fields.ModuleField{
		{Column: id, Type: fields.ModuleFieldTypeInt},
		{Column: participantID, Type: fields.ModuleFieldTypeInt},
	}, []actions.ModuleAction{actions.ListModuleAction{}})
	summary := newModule("summary_records", []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}}, []actions.ModuleAction{actions.ListModuleAction{}})
	detail := newModule("detail_records", []fields.ModuleField{
		{Column: id, Type: fields.ModuleFieldTypeInt},
		{Column: parentID, Type: fields.ModuleFieldTypeInt},
	}, []actions.ModuleAction{
		actions.ListModuleAction{Filter: []pg.Column{parentID}},
		actions.AddModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
		actions.UpdateModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"}},
	})
	workspace := newModule("workspace_entry", []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}}, []actions.ModuleAction{actions.ListModuleAction{}, actions.AddModuleAction{}})
	state := newModule("state_records", []fields.ModuleField{
		{Column: id, Type: fields.ModuleFieldTypeInt},
		{Column: status, Type: fields.ModuleFieldTypeString},
	}, []actions.ModuleAction{actions.UpdateModuleAction{By: []pg.Column{id}, Columns: []pg.Column{status}}})
	workspace.Render.List.Actions = []renderer.Action{{
		ID:   "open",
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
		}},
	}}
	workspace.Actions[0] = actions.ListModuleAction{
		Widget: &actions.WidgetConfig{
			ID: "work-area",
			Renderer: renderer.GlobalWidget{
				Surface: renderer.WidgetSurface{Kind: renderer.WidgetSurfaceDrawer, Placement: renderer.WidgetPlacementShellEnd, LoadPolicy: renderer.WidgetLoadOnOpen},
				Workspace: &renderer.WorkspaceWidget{
					Selection: renderer.WorkspaceSelection{Field: "id"},
					Summary:   &renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "summary_records", Action: "list"}},
					Master:    renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "master_records", Action: "list"}},
					Detail: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "detail_records", Action: "list"}, Bindings: []renderer.WidgetRequestBinding{{
						Target: renderer.WidgetRequestBindingFilter,
						Field:  "parent_id",
						Source: selectionSource("id"),
					}}},
					Commands: []renderer.WorkspaceCommand{{
						ID:    "set_status",
						Label: "workspace.command.set_status",
						WorkspaceResource: renderer.WorkspaceResource{ActionResource: renderer.ActionResource{Module: "state_records", Action: "update"}, Bindings: []renderer.WidgetRequestBinding{
							{Target: renderer.WidgetRequestBindingPathByKey, Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "id"}}},
							{Target: renderer.WidgetRequestBindingPathValue, Source: selectionSource("participant_id")},
							{Target: renderer.WidgetRequestBindingBody, Field: "status", Source: renderer.WidgetValueSource{Literal: &renderer.TypedValue{Type: renderer.TypedValueString, String: "active"}}},
						}},
						Refresh: []renderer.WorkspaceRefreshTarget{renderer.WorkspaceRefreshMaster, renderer.WorkspaceRefreshDetail},
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
	}
	return []*BaseModule{master, detail, workspace, summary, state}
}
