package renderer

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecorationContract(t *testing.T) {
	r := Universal{Record: &RecordPage{Sections: []RecordSection{{ID: "summary", Renderer: RecordRendererDisplay,
		Block:      &Block{Decoration: &Decoration{BackgroundSource: "asset://sample/pattern.png"}},
		Components: []DisplayComponent{{ID: "metrics", Type: DisplayDataList, Size: SizeLG, Fields: []string{"count"}, Decoration: &Decoration{Source: "asset://sample/object.png"}}},
	}}}}
	require.NoError(t, r.Validate())
	encoded, err := json.Marshal(r)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"decoration":{"source":"asset://sample/object.png"}`)
	clone := r.Clone()
	clone.Record.Sections[0].Block.Decoration.BackgroundSource = "changed"
	clone.Record.Sections[0].Components[0].Decoration.Source = "changed"
	require.Equal(t, "asset://sample/pattern.png", r.Record.Sections[0].Block.Decoration.BackgroundSource)
	require.Equal(t, "asset://sample/object.png", r.Record.Sections[0].Components[0].Decoration.Source)
	localized := Localize(r, func(value, key string) string { return "translated" })
	require.Equal(t, r.Record.Sections[0].Block.Decoration, localized.Record.Sections[0].Block.Decoration)
	require.Equal(t, r.Record.Sections[0].Components[0].Decoration, localized.Record.Sections[0].Components[0].Decoration)
	legacy, err := json.Marshal(Block{})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(legacy))
}

func TestDecorationInvalid(t *testing.T) {
	for _, source := range []string{"javascript:alert(1)", "data:image/png;base64,x", "file:///private", "//external/path", "asset://a b", "asset://a%GG", "asset://a\n.png"} {
		require.Error(t, (&Decoration{Source: source}).Validate(), source)
	}
	require.Error(t, (&Decoration{}).Validate())
	require.Error(t, (DisplayComponent{Type: DisplayText, Decoration: &Decoration{Source: "/object.png"}}).Validate())
	require.Error(t, (&Block{Decoration: &Decoration{}}).Validate())
}
