package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptListJSONCloneAndLocalization(t *testing.T) {
	pushKey := "public-key"
	render := Universal{
		Form: &FormPage{Sections: []FormSection{{
			ID: "channels",
			Prompts: &PromptList{Variant: "compact", Items: []Prompt{{
				ID:   "push",
				Kind: "push",
				Icon: "push",
				Tone: "info",
				Text: "notifications.prompts.push.text",
				Action: &Action{
					ID:    "connect_push",
					Type:  ActionEmit,
					Label: "notifications.prompts.push.connect",
					Client: &ClientAction{Name: "browser_push.connect", Arguments: []ClientActionArgument{{
						Name:  "application_server_key",
						Value: TypedValue{Type: TypedValueString, String: pushKey},
					}}},
				},
				VisibleIf: &Condition{Path: "record.push_connected", Falsy: boolPtr(true)},
			}}},
		}}},
	}

	require.NoError(t, render.Validate())
	encoded, err := json.Marshal(render)
	require.NoError(t, err)
	require.JSONEq(t, `{
  "form_page": {
    "sections": [{
      "id": "channels",
      "prompts": {
        "variant": "compact",
        "items": [{
          "id": "push",
          "kind": "push",
          "tone": "info",
          "icon": "push",
          "text": "notifications.prompts.push.text",
          "action": {
            "id": "connect_push",
            "type": "emit",
            "label": "notifications.prompts.push.connect",
            "client": {
              "name": "browser_push.connect",
              "arguments": [{
                "name": "application_server_key",
                "value": {"type":"string","string":"public-key"}
              }]
            }
          },
          "visible_if": {"path":"record.push_connected","falsy":true}
        }]
      }
    }]
  }
}`, string(encoded))

	cloned := render.Clone()
	cloned.Form.Sections[0].Prompts.Items[0].Action.Client.Arguments[0].Value.String = "other-key"
	require.Equal(t, pushKey, render.Form.Sections[0].Prompts.Items[0].Action.Client.Arguments[0].Value.String)

	localized := Localize(render, func(value, _ string) string {
		return "ru:" + value
	})
	prompt := localized.Form.Sections[0].Prompts.Items[0]
	require.Equal(t, "ru:notifications.prompts.push.text", prompt.Text)
	require.Equal(t, "ru:notifications.prompts.push.connect", prompt.Action.Label)
}

func TestPromptListValidation(t *testing.T) {
	tests := []struct {
		name   string
		prompt Prompt
	}{
		{name: "empty id", prompt: Prompt{Text: "text"}},
		{name: "empty content", prompt: Prompt{ID: "prompt"}},
		{name: "invalid action", prompt: Prompt{ID: "prompt", Text: "text", Action: &Action{Type: ActionEmit, Client: &ClientAction{}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			render := Universal{Form: &FormPage{Sections: []FormSection{{ID: "section", Prompts: &PromptList{Items: []Prompt{test.prompt}}}}}}
			require.Error(t, render.Validate())
		})
	}
}

func boolPtr(value bool) *bool { return &value }
