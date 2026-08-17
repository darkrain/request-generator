package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalizeUsesResolverWithoutMutatingSource(t *testing.T) {
	iconOnly := true
	source := Universal{Form: &FormPage{
		Title: "form.title",
		Actions: []Action{{
			ID:                 "save",
			Label:              "Save",
			LabelKey:           "actions.save",
			AriaLabel:          "Save",
			AriaLabelKey:       "actions.save",
			Title:              "Save",
			TitleKey:           "actions.save",
			ActionPresentation: ActionPresentation{IconOnly: &iconOnly},
		}},
		Sections: []FormSection{{
			ID: "rates",
			Matrix: &FieldMatrix{Type: FieldMatrixTypeTable, Table: &FieldMatrixTable{
				Heads: []string{"rates.duration", "Incall"},
				Rows:  []FieldMatrixRow{{Label: "rates.1h", Description: "rates.description", Cells: []FieldMatrixCell{{Label: "rates.channel", Text: "rates.none"}}}},
			}},
		}},
	}}
	translations := map[string]string{
		"form.title":        "Settings",
		"actions.save":      "Save changes",
		"rates.duration":    "Duration",
		"rates.1h":          "1 hour",
		"rates.none":        "Not available",
		"rates.description": "Price for one hour",
		"rates.channel":     "Channel",
	}

	localized := Localize(source, func(value, key string) string {
		if key != "" {
			if text, ok := translations[key]; ok {
				return text
			}
			return value
		}
		if text, ok := translations[value]; ok {
			return text
		}
		return value
	})

	require.Equal(t, "Settings", localized.Form.Title)
	require.Equal(t, "Save changes", localized.Form.Actions[0].Label)
	require.Empty(t, localized.Form.Actions[0].LabelKey)
	require.Equal(t, "Save changes", localized.Form.Actions[0].AriaLabel)
	require.Equal(t, "Save changes", localized.Form.Actions[0].Title)
	require.NotNil(t, localized.Form.Actions[0].IconOnly)
	require.True(t, *localized.Form.Actions[0].IconOnly)
	encoded, err := json.Marshal(localized.Form.Actions[0])
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"save","label":"Save changes","aria_label":"Save changes","title":"Save changes","icon_only":true}`, string(encoded))
	require.Equal(t, "Duration", localized.Form.Sections[0].Matrix.Table.Heads[0])
	require.Equal(t, "Not available", localized.Form.Sections[0].Matrix.Table.Rows[0].Cells[0].Text)
	require.Equal(t, "Channel", localized.Form.Sections[0].Matrix.Table.Rows[0].Cells[0].Label)
	require.Equal(t, "Price for one hour", localized.Form.Sections[0].Matrix.Table.Rows[0].Description)
	require.Equal(t, "form.title", source.Form.Title)
	require.Equal(t, "actions.save", source.Form.Actions[0].LabelKey)
}
