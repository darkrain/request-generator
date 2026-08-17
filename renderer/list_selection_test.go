package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validListSelectionPage() *ListPage {
	return &ListPage{
		CardSchema: &CardSchema{Actions: []Action{{ID: "toggle", Type: ActionAPI}}},
		Selection: &ListSelection{
			KeyField: "id", ToggleAction: "toggle", ValuesField: "selected_ids", Limit: 5,
			SelectedLabel: "ui.selected",
			Source:        &APIAction{Method: "GET", Endpoint: "/api/selections"},
			Clear:         &Action{ID: "clear", Type: ActionAPI, API: &APIAction{Method: "PUT", Endpoint: "/api/selections"}},
			Proceed:       &Action{ID: "proceed", Type: ActionRoute, Route: RouteAction{Path: "/next"}},
		},
	}
}

func TestListSelectionValidation(t *testing.T) {
	require.NoError(t, (Universal{List: validListSelectionPage()}).Validate())

	t.Run("unknown toggle action", func(t *testing.T) {
		page := validListSelectionPage()
		page.Selection.ToggleAction = "missing"
		err := (Universal{List: page}).Validate()
		require.EqualError(t, err, `renderer.ListPage: selection.toggle_action "missing" is not declared in card_schema.actions`)
	})

	t.Run("invalid limit", func(t *testing.T) {
		page := validListSelectionPage()
		page.Selection.Limit = 0
		err := (Universal{List: page}).Validate()
		require.EqualError(t, err, "renderer.ListPage: selection.limit must be greater than zero")
	})
}

func TestListSelectionCloneAndLocalization(t *testing.T) {
	source := Universal{List: validListSelectionPage()}
	localized := Localize(source, func(value, key string) string {
		if key != "" {
			return "translated:" + key
		}
		return "translated:" + value
	})

	require.Equal(t, "translated:ui.selected", localized.List.Selection.SelectedLabel)
	localized.List.Selection.Source.Endpoint = "/changed"
	require.Equal(t, "/api/selections", source.List.Selection.Source.Endpoint)
}
