package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func validGlobalWorkspace() GlobalWidget {
	return GlobalWidget{
		Surface: WidgetSurface{
			Kind:       WidgetSurfaceDrawer,
			Placement:  WidgetPlacementShellEnd,
			LoadPolicy: WidgetLoadOnOpen,
		},
		Workspace: &WorkspaceWidget{
			Selection: WorkspaceSelection{Field: "id"},
			Master:    WorkspaceResource{Module: "master_records", Action: "list"},
			Detail: WorkspaceResource{
				Module: "detail_records",
				Action: "list",
				Bindings: []WidgetRequestBinding{{
					Target: WidgetRequestBindingQuery,
					Name:   "filter[parent_id]",
					Value:  "selection.id",
				}},
			},
			Subscriptions: []WorkspaceSubscription{{
				Module:         "detail_records",
				Actions:        []string{"add", "update"},
				CorrelationKey: "parent_id",
				Refresh:        []WorkspaceRefreshTarget{WorkspaceRefreshMaster, WorkspaceRefreshDetail},
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
    "master":{"module":"master_records","action":"list"},
    "detail":{"module":"detail_records","action":"list","bindings":[{"target":"query","name":"filter[parent_id]","value":"selection.id"}]},
    "subscriptions":[{"module":"detail_records","actions":["add","update"],"correlation_key":"parent_id","refresh":["master","detail"]}]
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
				widget.Workspace.Detail.Bindings[0].Value = "selection.other_id"
			},
			err: "renderer.GlobalWidget: workspace: detail must bind selection.id",
		},
		{
			name: "duplicate refresh target",
			edit: func(widget *GlobalWidget) {
				widget.Workspace.Subscriptions[0].Refresh = []WorkspaceRefreshTarget{WorkspaceRefreshDetail, WorkspaceRefreshDetail}
			},
			err: "renderer.GlobalWidget: workspace: subscription 0: refresh target \"detail\" is duplicated",
		},
		{
			name: "drawer cannot be centered",
			edit: func(widget *GlobalWidget) {
				widget.Surface.Placement = WidgetPlacementCenter
			},
			err: "renderer.GlobalWidget: surface: drawer does not support placement \"center\"",
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
	cloned.Workspace.Detail.Bindings[0].Value = "selection.changed"
	cloned.Workspace.Subscriptions[0].Actions[0] = "delete"
	cloned.Workspace.Subscriptions[0].Refresh[0] = WorkspaceRefreshDetail

	require.Equal(t, "selection.id", source.Workspace.Detail.Bindings[0].Value)
	require.Equal(t, "add", source.Workspace.Subscriptions[0].Actions[0])
	require.Equal(t, WorkspaceRefreshMaster, source.Workspace.Subscriptions[0].Refresh[0])
}

func TestActionResultWidgetTargetIsTypedAndCloned(t *testing.T) {
	render := Universal{Record: &RecordPage{Actions: []Action{{
		ID:   "open",
		Type: ActionAPI,
		AfterSuccess: &ActionResult{Widget: &WidgetTarget{
			ID:             "work-area",
			State:          WidgetTargetOpen,
			SelectionField: "id",
			Refresh:        []WorkspaceRefreshTarget{WorkspaceRefreshDetail},
		}},
	}}}}
	require.NoError(t, render.Validate())

	cloned := render.Clone()
	cloned.Record.Actions[0].AfterSuccess.Widget.Refresh[0] = WorkspaceRefreshMaster
	require.Equal(t, WorkspaceRefreshDetail, render.Record.Actions[0].AfterSuccess.Widget.Refresh[0])

	targets := render.WidgetTargets()
	require.Len(t, targets, 1)
	require.Equal(t, "work-area", targets[0].ID)

	invalid := render.Clone()
	invalid.Record.Actions[0].AfterSuccess.Widget.State = WidgetTargetClose
	require.EqualError(t, invalid.Validate(), `renderer.Universal: record page action "open": after success: widget: closed widget cannot set selection`)

	invalid = render.Clone()
	invalid.Record.Actions[0].AfterSuccess.Widget.Refresh = []WorkspaceRefreshTarget{WorkspaceRefreshDetail, WorkspaceRefreshDetail}
	require.EqualError(t, invalid.Validate(), `renderer.Universal: record page action "open": after success: widget: refresh target "detail" is duplicated`)
}
