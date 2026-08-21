package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCardLeadingAccentContract(t *testing.T) {
	source := Universal{List: &ListPage{CardSchema: &CardSchema{
		LeadingAccent: &CardEdgeAccent{Tone: ToneToken("pink")},
	}}}

	require.NoError(t, source.Validate())

	encoded, err := json.Marshal(source)
	require.NoError(t, err)
	require.JSONEq(t, `{"list_page":{"card_schema":{"leading_accent":{"tone":"pink"}}}}`, string(encoded))

	cloned := source.Clone()
	cloned.List.CardSchema.LeadingAccent.Tone = ToneToken("cyan")
	require.Equal(t, ToneToken("pink"), source.List.CardSchema.LeadingAccent.Tone)
}

func TestCardLeadingAccentRequiresTone(t *testing.T) {
	schema := CardSchema{LeadingAccent: &CardEdgeAccent{}}
	require.EqualError(t, schema.Validate(), "renderer.CardSchema: leading_accent tone is required")
}
