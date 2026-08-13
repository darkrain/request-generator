package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActionBehaviorJSONAndClone(t *testing.T) {
	render := Universal{Form: &FormPage{Actions: []Action{
		{ID: "save", Type: ActionAPI, Behavior: ActionBehaviorSubmit, API: &APIAction{Method: "POST", Endpoint: "/records"}},
		{ID: "discard", Type: ActionEmit, Behavior: ActionBehaviorReset},
		{ID: "help", Type: ActionModal},
	}}}

	encoded, err := json.Marshal(render)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"behavior":"submit"`)
	require.Contains(t, string(encoded), `"behavior":"reset"`)
	require.NotContains(t, string(encoded), `"id":"help","type":"modal","behavior"`)

	cloned := render.Clone()
	require.Equal(t, ActionBehaviorSubmit, cloned.Form.Actions[0].Behavior)
	require.Equal(t, ActionBehaviorReset, cloned.Form.Actions[1].Behavior)
	require.Empty(t, cloned.Form.Actions[2].Behavior)
}

func TestActionAppearanceIsOpenToken(t *testing.T) {
	render := Universal{Form: &FormPage{Actions: []Action{{
		ID:               "custom-style",
		Appearance:       "service",
		ActiveAppearance: "service-active",
	}}}}

	require.NoError(t, render.Validate())
	encoded, err := json.Marshal(render)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"appearance":"service"`)
	require.Contains(t, string(encoded), `"active_appearance":"service-active"`)

	cloned := render.Clone()
	require.Equal(t, ActionAppearance("service"), cloned.Form.Actions[0].Appearance)
	require.Equal(t, ActionAppearance("service-active"), cloned.Form.Actions[0].ActiveAppearance)
}

func TestUniversalValidateActionBehavior(t *testing.T) {
	for _, behavior := range []ActionBehavior{"", ActionBehaviorSubmit, ActionBehaviorReset} {
		render := Universal{Form: &FormPage{Actions: []Action{{ID: "action", Type: ActionAPI, Behavior: behavior}}}}
		require.NoError(t, render.Validate())
	}

	invalid := Action{ID: "invalid", Behavior: ActionBehavior("unsupported")}
	tests := []struct {
		name   string
		render Universal
		err    string
	}{
		{"list action", Universal{List: &ListPage{Actions: []Action{invalid}}}, `renderer.Universal: list page action "invalid": unsupported behavior "unsupported"`},
		{"list card action", Universal{List: &ListPage{CardSchema: &CardSchema{Actions: []Action{invalid}}}}, `renderer.Universal: list page card schema action "invalid": unsupported behavior "unsupported"`},
		{"nested list card action", Universal{Form: &FormPage{Sections: []FormSection{{ID: "nested", ListPage: &ListPage{CardSchema: &CardSchema{Actions: []Action{invalid}}}}}}}, `renderer.Universal: form section list page card schema action "invalid": unsupported behavior "unsupported"`},
		{"form action", Universal{Form: &FormPage{Actions: []Action{invalid}}}, `renderer.Universal: form page action "invalid": unsupported behavior "unsupported"`},
		{"collection action", Universal{Form: &FormPage{Sections: []FormSection{{ID: "collection", Collection: &CollectionConfig{Module: "items", Actions: []Action{invalid}}}}}}, `renderer.Universal: collection action "invalid": unsupported behavior "unsupported"`},
		{"collection bucket action", Universal{Form: &FormPage{Sections: []FormSection{{ID: "collection", Collection: &CollectionConfig{Module: "items", Buckets: []CollectionBucket{{ID: "bucket", Actions: []Action{invalid}}}}}}}}, `renderer.Universal: collection bucket action "invalid": unsupported behavior "unsupported"`},
		{"media action", Universal{Form: &FormPage{Sections: []FormSection{{ID: "media", MediaActions: &MediaGalleryActions{Upload: &invalid}}}}}, `renderer.Universal: media upload action "invalid": unsupported behavior "unsupported"`},
		{"record action", Universal{Record: &RecordPage{Actions: []Action{invalid}}}, `renderer.Universal: record page action "invalid": unsupported behavior "unsupported"`},
		{"resource grid action", Universal{ResourceGrid: &ResourceGridPage{Create: &invalid}}, `renderer.Universal: resource grid create action "invalid": unsupported behavior "unsupported"`},
		{"resource grid card action", Universal{ResourceGrid: &ResourceGridPage{Card: &CardSchema{Actions: []Action{invalid}}}}, `renderer.Universal: resource grid card action "invalid": unsupported behavior "unsupported"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.EqualError(t, test.render.Validate(), test.err)
		})
	}
}
