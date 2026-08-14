package renderer

import (
	"encoding/json"
	"testing"
)

func TestCardActionLayoutValidationAndJSON(t *testing.T) {
	schema := CardSchema{ActionLayout: CardActionLayoutEdgeFill}
	if err := schema.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	encoded, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got, want := string(encoded), `{"action_layout":"edge_fill"}`; got != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}

	invalid := CardSchema{ActionLayout: CardActionLayout("stacked")}
	if got, want := invalid.Validate().Error(), `renderer.CardSchema: unsupported action layout "stacked"`; got != want {
		t.Fatalf("Validate() error = %q, want %q", got, want)
	}
}

func TestCardActionLayoutIsNotChangedByLocalization(t *testing.T) {
	value := true
	source := Universal{List: &ListPage{CardSchema: &CardSchema{
		ActionLayout: CardActionLayoutEdgeFill,
		Actions: []Action{{
			LabelKey:           "action.open",
			ActionPresentation: ActionPresentation{IconOnly: &value},
		}},
	}}}

	localized := Localize(source, func(value, key string) string { return "Open" })
	got := localized.List.CardSchema
	if got.ActionLayout != CardActionLayoutEdgeFill {
		t.Fatalf("ActionLayout = %q, want %q", got.ActionLayout, CardActionLayoutEdgeFill)
	}
	if got.Actions[0].Label != "Open" {
		t.Fatalf("localized action label = %q, want Open", got.Actions[0].Label)
	}
}
