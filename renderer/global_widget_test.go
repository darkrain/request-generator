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
				Correlation: &WorkspaceCorrelationBinding{EventField: "parent_id"},
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
	cloned.Workspace.Commands[0].Presentation.VisibleIf.Path = "changed"
	cloned.Workspace.Commands[0].Bindings[2].Source.Literal.String = "disabled"
	cloned.Workspace.Commands[0].Refresh[0] = WorkspaceRefreshDetail
	cloned.Workspace.Subscriptions[0].Actions[0] = "delete"
	cloned.Workspace.Subscriptions[0].Refresh[0] = WorkspaceRefreshDetail

	require.Equal(t, "id", source.Workspace.Detail.Bindings[0].Source.Runtime.Field)
	require.Equal(t, "list", source.Workspace.Summary.Action)
	require.Equal(t, "workspace.command.set_status", source.Workspace.Commands[0].Label)
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

func TestLocalizeGlobalWidgetLocalizesCommandLabels(t *testing.T) {
	source := validGlobalWorkspace()
	localized := LocalizeGlobalWidget(source, func(value string, _ string) string {
		return "translated:" + value
	})

	require.Equal(t, "translated:workspace.command.set_status", localized.Workspace.Commands[0].Label)
	require.Equal(t, "workspace.command.set_status", source.Workspace.Commands[0].Label)
}

func TestWidgetTriggerIsLocalizedValidatedAndCloned(t *testing.T) {
	widget := validGlobalWorkspace()
	widget.Surface.Trigger = &WidgetTrigger{
		Label: "workspace.notifications",
		Icon:  "bell",
		Badge: &Badge{Field: "unread_count", IfField: "unread_count", Tone: "pink"},
	}
	require.NoError(t, widget.Validate())

	localized := LocalizeGlobalWidget(widget, func(value string, _ string) string {
		return "translated:" + value
	})
	require.Equal(t, "translated:workspace.notifications", localized.Surface.Trigger.Label)
	localized.Surface.Trigger.Badge.Field = "changed"
	require.Equal(t, "unread_count", widget.Surface.Trigger.Badge.Field)

	widget.Surface.Trigger = &WidgetTrigger{Label: "workspace.notifications"}
	require.EqualError(t, widget.Validate(), "renderer.GlobalWidget: surface: trigger: icon is required")

	widget.Surface.Trigger = &WidgetTrigger{Label: "workspace.notifications", Icon: "bell", Badge: &Badge{}}
	require.EqualError(t, widget.Validate(), "renderer.GlobalWidget: surface: trigger: badge field or value is required")
}

func TestGlobalWorkspaceCommandValidation(t *testing.T) {
	widget := validGlobalWorkspace()
	widget.Workspace.Commands = append(widget.Workspace.Commands, widget.Workspace.Commands[0])
	require.EqualError(t, widget.Validate(), `renderer.GlobalWidget: workspace: command "set_status" is duplicated`)

	widget = validGlobalWorkspace()
	widget.Workspace.Commands[0].Refresh = nil
	require.EqualError(t, widget.Validate(), `renderer.GlobalWidget: workspace: command 0: refresh targets are required`)
}

func TestWorkspaceCommandWithoutSelectionRejectsSelectionBinding(t *testing.T) {
	requireSelection := false
	command := WorkspaceCommand{
		ID:               "mark-all",
		Label:            "workspace.command.mark_all",
		RequireSelection: &requireSelection,
		WorkspaceResource: WorkspaceResource{ActionResource: ActionResource{
			Module: "entries",
			Action: "update",
		}, Bindings: []WidgetRequestBinding{{
			Target: WidgetRequestBindingPathValue,
			Source: widgetSelectionSource("id"),
		}}},
		Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshMaster},
	}
	require.EqualError(t, command.Validate(), "does not require selection but binding reads selection")
}

func TestWorkspaceCommandRejectsSelectionTriggerWithoutSelection(t *testing.T) {
	requireSelection := false
	command := WorkspaceCommand{
		ID:               "mark-all",
		Label:            "workspace.command.mark_all",
		Trigger:          WorkspaceCommandTriggerSelectionOpen,
		RequireSelection: &requireSelection,
		WorkspaceResource: WorkspaceResource{ActionResource: ActionResource{
			Module: "entries",
			Action: "update",
		}},
		Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshMaster},
	}
	require.EqualError(t, command.Validate(), "selection_open trigger requires selection")
}

func TestWorkspaceSubscriptionWithoutCorrelationIsValid(t *testing.T) {
	subscription := WorkspaceSubscription{
		Module:  "entries",
		Actions: []string{"add"},
		Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshSummary, WorkspaceRefreshMaster},
	}
	require.NoError(t, subscription.Validate())
}

func TestWorkspaceSubscriptionToastIsValidatedAndCloned(t *testing.T) {
	subscription := WorkspaceSubscription{
		Module:  "events",
		Actions: []string{"add"},
		Refresh: []WorkspaceRefreshTarget{WorkspaceRefreshMaster},
		Toast: &WorkspaceSubscriptionToast{
			Title:   &TextBinding{Field: "title"},
			Message: &TextBinding{Field: "message"},
			Tone:    "info",
		},
	}
	require.NoError(t, subscription.Validate())

	cloned := cloneWorkspaceSubscriptions([]WorkspaceSubscription{subscription})
	require.Len(t, cloned, 1)
	require.NotSame(t, subscription.Toast, cloned[0].Toast)
	cloned[0].Toast.Title.Field = "changed"
	require.Equal(t, "title", subscription.Toast.Title.Field)

	subscription.Toast = &WorkspaceSubscriptionToast{}
	require.EqualError(t, subscription.Validate(), "toast: title or message is required")
}

func TestWidgetLoadCloneDoesNotShareCommandInputState(t *testing.T) {
	load := WidgetLoad{Commands: []WorkspaceCommandLoad{{
		ID: "create-entry",
		Input: &WorkspaceCommandInputLoad{Definition: WidgetResourceLoad{
			Request: APIAction{Method: "GET", Endpoint: "/api/entries/defrec/"},
		}},
		AfterSuccess: &ActionResult{Widget: &WidgetTarget{ID: "other", State: WidgetTargetOpen}},
	}}}

	cloned := load.Clone()
	cloned.Commands[0].Input.Definition.Request.Endpoint = "/changed"
	cloned.Commands[0].AfterSuccess.Widget.ID = "changed-widget"

	require.Equal(t, "/api/entries/defrec/", load.Commands[0].Input.Definition.Request.Endpoint)
	require.Equal(t, "other", load.Commands[0].AfterSuccess.Widget.ID)
}

func TestWorkspaceCommandAllowsTypedAfterSuccess(t *testing.T) {
	command := WorkspaceCommand{
		ID:    "open",
		Label: "workspace.command.open",
		AfterSuccess: &ActionResult{Widget: &WidgetTarget{
			ID:    "chat-workspace",
			State: WidgetTargetOpen,
		}},
		WorkspaceResource: WorkspaceResource{ActionResource: ActionResource{Module: "entries", Action: "update"}},
		Refresh:           []WorkspaceRefreshTarget{WorkspaceRefreshMaster},
	}
	require.NoError(t, command.Validate())

	command.AfterSuccess.Widget.State = WidgetTargetClose
	command.AfterSuccess.Widget.Selection = &WidgetSelectionResultBinding{Source: ActionResultSource{Resource: ActionResource{Module: "entries", Action: "update"}, Field: ActionResultFieldValue}}
	require.EqualError(t, command.Validate(), "after success: widget: closed widget cannot set selection")
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
