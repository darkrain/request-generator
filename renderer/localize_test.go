package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalizeUsesResolverWithoutMutatingSource(t *testing.T) {
	source := Universal{Form: &FormPage{
		Title:   "form.title",
		Actions: []Action{{ID: "save", Label: "Save", LabelKey: "actions.save"}},
		Sections: []FormSection{{
			ID: "rates",
			Matrix: &FieldMatrix{Type: FieldMatrixTypeTable, Table: &FieldMatrixTable{
				Heads: []string{"rates.duration", "Incall"},
				Rows:  []FieldMatrixRow{{Label: "rates.1h", Cells: []FieldMatrixCell{{Text: "rates.none"}}}},
			}},
		}},
	}}
	translations := map[string]string{
		"form.title":     "Settings",
		"actions.save":   "Save changes",
		"rates.duration": "Duration",
		"rates.1h":       "1 hour",
		"rates.none":     "Not available",
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
	require.Equal(t, "Duration", localized.Form.Sections[0].Matrix.Table.Heads[0])
	require.Equal(t, "Not available", localized.Form.Sections[0].Matrix.Table.Rows[0].Cells[0].Text)
	require.Equal(t, "form.title", source.Form.Title)
	require.Equal(t, "actions.save", source.Form.Actions[0].LabelKey)
}
