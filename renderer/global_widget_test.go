package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func widgetSelectionSource(field string) WidgetValueSource {
	return WidgetValueSource{Runtime: &WidgetRuntimeValue{
		Scope: WidgetRuntimeValueSourceSelection,
		Field: field,
	}}
}

func validGlobalWorkspace() GlobalWidget {
	iconOnly := true
	return GlobalWidget{
		Surface: WidgetSurface{
			Kind:       WidgetSurfaceDrawer,
			Placement:  WidgetPlacementShellEnd,
			LoadPolicy: WidgetLoadOnOpen,
		},
		Workspace: &WorkspaceWidget{
			Selection: WorkspaceSelection{Field: "id"},
			Summary:   &WorkspaceResource{ActionResource: ActionResource{Module: "summary_records", Action: "list"}},
			Master:    WorkspaceResource{ActionResource: ActionResource{Module: "master_records", Action: "list"}},
			Detail: WorkspaceResource{
				ActionResource: ActionResource{Module: "detail_records", Action: "list"},
				Bindings: []WidgetRequestBinding{{
					Target: WidgetRequestBindingFilter,
					Field:  "parent_id",
					Source: widgetSelectionSource("id"),
				}},
			},
			ComposerActions: []Action{{
				ID: "open_related", Type: ActionRoute, LabelKey: "workspace.action.open_related",
				ActionPresentation: ActionPresentation{Icon: "ref-order", IconOnly: &iconOnly, Variant: ActionVariantPrimary, Appearance: ActionAppearanceOutline},
				Route:              RouteAction{Path: "/related", Query: map[string]interface{}{"id": "record.id"}},
			}},
			Commands: []WorkspaceCommand{{
				ID:    "set_status",
				Label: "workspace.command.set_status",
				Presentation: &ActionPresentation{
					Icon:       "ref-status",
					IconOnly:   &iconOnly,
					Variant:    ActionVariantSuccess,
					Appearance: ActionAppearanceOutline,
					Active:     "is_active",
					VisibleIf:  &Condition{Path: "enabled", Equals: true},
				},
				WorkspaceResource: WorkspaceResource{ActionResource: ActionResource{Module: "state_records", Action: "update"}, Bindings: []WidgetRequestBinding{
					{Target: WidgetRequestBindingPathByKey, Source: WidgetValueSource{Literal: &TypedValue{Type: TypedValueString, String: "id"}}},
					{Target: WidgetRequestBindingPathValue, Source: widgetSelectionSource("participant_id")},
					{Target: WidgetRequestBindingBody, Field: "status", Source: WidgetValueSource{Literal: &TypedValue{Type: TypedValueString, String: "active"}}},
				}},
				Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshMaster, WorkspaceRefreshDetail},
			}},
			Subscriptions: []WorkspaceSubscription{{
				Module:      "detail_records",
				Actions:     []string{"add", "update"},
				Correlation: WorkspaceCorrelationBinding{EventField: "parent_id"},
				Refresh:     []WorkspaceRefreshTarget{WorkspaceRefreshMaster, WorkspaceRefreshDetail},
			}},
		},
	}
}

func TestGlobalWidgetValidateAndSerialize(t *testing.T) {
	widget := validGlobalWorkspace()
	require.NoError(t, widget.Validate())
	require.Equal(t, UniversalIdentity(), widget.Identity())

	encoded, err := json.Marshal(widget)
	require.NoError(t, err)
	require.JSONEq(t, `{
  "surface": {"kind":"drawer","placement":"shell_end","load_policy":"on_open"},
  "workspace": {
    "selection":{"field":"id"},
    "summary":{"module":"summary_records","action":"list"},
    "master":{"module":"master_records","action":"list"},
    "detail":{"module":"detail_records","action":"list","bindings":[{"target":"filter","field":"parent_id","source":{"runtime":{"scope":"selection","field":"id"}}}]},
    "composer_actions":[{"icon":"ref-order","icon_only":true,"variant":"primary","appearance":"outline","id":"open_related","type":"route","label_key":"workspace.action.open_related","route":{"path":"/related","query":{"id":"record.id"}}}],
    "commands":[{"id":"set_status","label":"workspace.command.set_status","presentation":{"icon":"ref-status","icon_only":true,"variant":"success","appearance":"outline","active":"is_active","visible_if":{"path":"enabled","equals":true}},"module":"state_records","action":"update","bindings":[{"target":"path_by_key","source":{"literal":{"type":"string","string":"id"}}},{"target":"path_value","source":{"runtime":{"scope":"selection","field":"participant_id"}}},{"target":"body","field":"status","source":{"literal":{"type":"string","string":"active"}}}],"refresh":["master","detail"]}],
    "subscriptions":[{"module":"detail_records","actions":["add","update"],"correlation":{"event_field":"parent_id"},"refresh":["master","detail"]}]
  }
}`, string(encoded))
}

func TestGlobalWidgetValidateRejectsInvalidContract(t *testing.T) {
	tests := []struct {
		name string
		edit func(*GlobalWidget)
		err  string
	}{
		{
			name: "selection is not bound by detail",
			edit: func(widget *GlobalWidget) {
				widget.Workspace.Detail.Bindings[0].Source = WidgetValueSource{Literal: &TypedValue{Type: TypedValueNumber, Number: 1}}
			},
			err: "renderer.GlobalWidget: workspace: detail must bind selection field \"id\"",
		},
		{
			name: "binding source must be a union",
			edit: func(widget *GlobalWidget) {
				widget.Workspace.Detail.Bindings[0].Source = WidgetValueSource{}
			},
			err: "renderer.GlobalWidget: workspace: binding 0 source: must contain exactly one of literal or runtime",
		},
		{
			name: "duplicate refresh target",
			edit: func(widget *GlobalWidget) {
				widget.Workspace.Subscriptions[0].Refresh = []WorkspaceRefreshTarget{WorkspaceRefreshDetail, WorkspaceRefreshDetail}
			},
			err: `renderer.GlobalWidget: workspace: subscription 0: refresh target "detail" is duplicated`,
		},
		{
			name: "drawer cannot be centered",
			edit: func(widget *GlobalWidget) {
				widget.Surface.Placement = WidgetPlacementCenter
			},
			err: `renderer.GlobalWidget: surface: drawer does not support placement "center"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			widget := validGlobalWorkspace()
			test.edit(&widget)
			require.EqualError(t, widget.Validate(), test.err)
		})
	}
}

func TestGlobalWidgetCloneDoesNotShareWorkspaceState(t *testing.T) {
	source := validGlobalWorkspace()
	cloned := LocalizeGlobalWidget(source, func(value string, key string) string { return value + key })
	cloned.Workspace.Detail.Bindings[0].Source.Runtime.Field = "changed"
	cloned.Workspace.Summary.Action = "view"
	cloned.Workspace.Commands[0].Label = "changed"
	cloned.Workspace.ComposerActions[0].Route.(RouteAction).Query["id"] = "changed"
	cloned.Workspace.Commands[0].Presentation.VisibleIf.Path = "changed"
	cloned.Workspace.Commands[0].Bindings[2].Source.Literal.String = "disabled"
	cloned.Workspace.Commands[0].Refresh[0] = WorkspaceRefreshDetail
	cloned.Workspace.Subscriptions[0].Actions[0] = "delete"
	cloned.Workspace.Subscriptions[0].Refresh[0] = WorkspaceRefreshDetail

	require.Equal(t, "id", source.Workspace.Detail.Bindings[0].Source.Runtime.Field)
	require.Equal(t, "list", source.Workspace.Summary.Action)
	require.Equal(t, "workspace.command.set_status", source.Workspace.Commands[0].Label)
	require.Equal(t, "record.id", source.Workspace.ComposerActions[0].Route.(RouteAction).Query["id"])
	require.Equal(t, "enabled", source.Workspace.Commands[0].Presentation.VisibleIf.Path)
	require.Equal(t, "active", source.Workspace.Commands[0].Bindings[2].Source.Literal.String)
	require.Equal(t, WorkspaceRefreshMaster, source.Workspace.Commands[0].Refresh[0])
	require.Equal(t, "add", source.Workspace.Subscriptions[0].Actions[0])
	require.Equal(t, WorkspaceRefreshMaster, source.Workspace.Subscriptions[0].Refresh[0])
}

func TestWorkspaceCommandInputValidation(t *testing.T) {
	inputSource := func(field string) WidgetValueSource {
		return WidgetValueSource{Runtime: &WidgetRuntimeValue{Scope: WidgetRuntimeValueSourceInput, Field: field}}
	}
	base := WorkspaceCommand{
		ID:    "create-entry",
		Label: "workspace.command.create_entry",
		Input: &WorkspaceCommandInput{Fields: []string{"text"}},
		WorkspaceResource: WorkspaceResource{ActionResource: ActionResource{Module: "entries", Action: "add"}, Bindings: []WidgetRequestBinding{{
			Target: WidgetRequestBindingBody,
			Field:  "text",
			Source: inputSource("text"),
		}}},
		Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshDetail},
	}
	require.NoError(t, base.Validate())

	tests := []struct {
		name string
		edit func(*WorkspaceCommand)
		err  string
	}{
		{
			name: "input source without declaration",
			edit: func(command *WorkspaceCommand) { command.Input = nil },
			err:  "input: runtime input source requires input declaration",
		},
		{
			name: "undeclared input field",
			edit: func(command *WorkspaceCommand) { command.Bindings[0].Source = inputSource("title") },
			err:  `input: runtime input field "title" is not declared`,
		},
		{
			name: "input field maps to another body field",
			edit: func(command *WorkspaceCommand) { command.Bindings[0].Field = "body" },
			err:  `input: runtime input field "text" must bind body field "text"`,
		},
		{
			name: "duplicate input field",
			edit: func(command *WorkspaceCommand) { command.Input.Fields = []string{"text", "text"} },
			err:  `input: field "text" is duplicated`,
		},
		{
			name: "input field without binding",
			edit: func(command *WorkspaceCommand) { command.Input.Fields = []string{"text", "attachments"} },
			err:  `input: field "attachments" has no input binding`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := base
			command.Input = &WorkspaceCommandInput{Fields: cloneSlice(base.Input.Fields)}
			command.Bindings = cloneWidgetRequestBindings(base.Bindings)
			test.edit(&command)
			require.EqualError(t, command.Validate(), test.err)
		})
	}
}

func TestWorkspaceCommandTriggerValidation(t *testing.T) {
	command := validGlobalWorkspace().Workspace.Commands[0]
	command.Trigger = WorkspaceCommandTriggerSelectionOpen
	require.NoError(t, command.Validate())
	encoded, err := json.Marshal(command)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"trigger":"selection_open"`)

	command.Input = &WorkspaceCommandInput{Fields: []string{"text"}}
	require.EqualError(t, command.Validate(), "triggered command must not declare input")

	command.Input = nil
	command.Trigger = WorkspaceCommandTrigger("on_open")
	require.EqualError(t, command.Validate(), `trigger: unsupported value "on_open"`)
}

func TestLocalizeGlobalWidgetLocalizesCommandAndComposerActionLabels(t *testing.T) {
	source := validGlobalWorkspace()
	localized := LocalizeGlobalWidget(source, func(value string, key string) string {
		if key != "" {
			return "translated:" + key
		}
		return "translated:" + value
	})

	require.Equal(t, "translated:workspace.command.set_status", localized.Workspace.Commands[0].Label)
	require.Equal(t, "translated:workspace.action.open_related", localized.Workspace.ComposerActions[0].Label)
	require.Empty(t, localized.Workspace.ComposerActions[0].LabelKey)
	require.Equal(t, "workspace.command.set_status", source.Workspace.Commands[0].Label)
}

func TestWorkspaceRejectsDuplicateComposerActions(t *testing.T) {
	widget := validGlobalWorkspace()
	widget.Workspace.ComposerActions = append(widget.Workspace.ComposerActions, widget.Workspace.ComposerActions[0])
	require.EqualError(t, widget.Validate(), `renderer.GlobalWidget: workspace: composer action "open_related" is duplicated`)
}

func TestGlobalWorkspaceCommandValidation(t *testing.T) {
	widget := validGlobalWorkspace()
	widget.Workspace.Commands = append(widget.Workspace.Commands, widget.Workspace.Commands[0])
	require.EqualError(t, widget.Validate(), `renderer.GlobalWidget: workspace: command "set_status" is duplicated`)

	widget = validGlobalWorkspace()
	widget.Workspace.Commands[0].Refresh = nil
	require.EqualError(t, widget.Validate(), `renderer.GlobalWidget: workspace: command 0: refresh targets are required`)
}

func TestWidgetLoadCloneDoesNotShareCommandInputState(t *testing.T) {
	load := WidgetLoad{Commands: []WorkspaceCommandLoad{{
		ID: "create-entry",
		Input: &WorkspaceCommandInputLoad{Definition: WidgetResourceLoad{
			Request: APIAction{Method: "GET", Endpoint: "/api/entries/defrec/"},
		}},
	}}}

	cloned := load.Clone()
	cloned.Commands[0].Input.Definition.Request.Endpoint = "/changed"

	require.Equal(t, "/api/entries/defrec/", load.Commands[0].Input.Definition.Request.Endpoint)
}

func TestGlobalWorkspaceAllowsOmittedSummary(t *testing.T) {
	widget := validGlobalWorkspace()
	widget.Workspace.Summary = nil

	require.NoError(t, widget.Validate())
	encoded, err := json.Marshal(widget)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), `"summary"`)
}

func TestActionResultWidgetTargetUsesResultField(t *testing.T) {
	render := Universal{Record: &RecordPage{Actions: []Action{{
		ID:   "open",
		Type: ActionAPI,
		AfterSuccess: &ActionResult{Widget: &WidgetTarget{
			ID:    "work-area",
			State: WidgetTargetOpen,
			Selection: &WidgetSelectionResultBinding{
				Source: ActionResultSource{
					Resource: ActionResource{Module: "records", Action: "add"},
					Field:    ActionResultFieldValue,
				},
			},
			Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshDetail},
		}},
	}}}}
	require.NoError(t, render.Validate())

	cloned := render.Clone()
	cloned.Record.Actions[0].AfterSuccess.Widget.Selection.Source.Field = "changed"
	cloned.Record.Actions[0].AfterSuccess.Widget.Refresh[0] = WorkspaceRefreshMaster
	require.Equal(t, ActionResultFieldValue, render.Record.Actions[0].AfterSuccess.Widget.Selection.Source.Field)
	require.Equal(t, WorkspaceRefreshDetail, render.Record.Actions[0].AfterSuccess.Widget.Refresh[0])

	actions := render.Actions()
	require.Len(t, actions, 1)
	require.Equal(t, "work-area", actions[0].AfterSuccess.Widget.ID)
	encoded, err := json.Marshal(render)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"selection":{"source":{"resource":{"module":"records","action":"add"},"field":"value"}}`)

	invalid := render.Clone()
	invalid.Record.Actions[0].AfterError = &ActionResult{Widget: &WidgetTarget{
		ID: "work-area",
		Selection: &WidgetSelectionResultBinding{Source: ActionResultSource{
			Resource: ActionResource{Module: "records", Action: "add"}, Field: ActionResultFieldValue,
		}},
	}}
	require.EqualError(t, invalid.Validate(), `renderer.Universal: record page action "open": after error: widget selection is only allowed after success`)

	invalid = render.Clone()
	invalid.Record.Actions[0].AfterSuccess.Widget.State = WidgetTargetClose
	require.EqualError(t, invalid.Validate(), `renderer.Universal: record page action "open": after success: widget: closed widget cannot set selection`)

	invalid = render.Clone()
	invalid.Record.Actions[0].AfterSuccess.Widget.Refresh = []WorkspaceRefreshTarget{WorkspaceRefreshDetail, WorkspaceRefreshDetail}
	require.EqualError(t, invalid.Validate(), `renderer.Universal: record page action "open": after success: widget: refresh target "detail" is duplicated`)
}
