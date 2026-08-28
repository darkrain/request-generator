package response

import (
	"testing"

	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestNewDefrecResponseExposesActiveLengthAndInputMode(t *testing.T) {
	code := pg.StringColumn("code")
	response := NewDefrecResponse([]fields.ModuleField{{
		Column:       code,
		Title:        "Code",
		Type:         fields.ModuleFieldTypeString,
		FormType:     fields.ModuleFieldFormTypeText,
		Presentation: &renderer.FieldPresentation{InputMode: renderer.FieldInputModeNumeric},
		Check: []fields.CheckRules{
			fields.RequiredRule(code, []fields.Scenario{fields.ScenarioAdd}),
			fields.LenRule(code, 4, 4, []fields.Scenario{fields.ScenarioAdd}),
			fields.LenRule(code, 8, 8, []fields.Scenario{fields.ScenarioUpdate}),
		},
	}})

	item := response.Fields["code"].(map[string]interface{})
	require.Equal(t, true, item["required"])
	require.Equal(t, 4, item["min_length"])
	require.Equal(t, 4, item["max_length"])
	require.Equal(t, renderer.FieldInputModeNumeric, item["presentation"].(*renderer.FieldPresentation).InputMode)
}
