package renderer

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPageHintOptionalLocalizedAndCloned(t *testing.T) {
	r := Universal{Record: &RecordPage{Hint: &PageHint{Key: "intro-v1", Title: "hint.title", Text: "hint.text", Acknowledge: "hint.ok", Close: "hint.close", Icon: "info"}, Actions: []Action{{ID: "details", Type: ActionRoute, ActionPresentation: ActionPresentation{Placement: ActionPlacementHead}, Route: RouteAction{Path: "/details"}}}}}
	require.NoError(t, r.Validate())
	cp := r.Clone()
	cp.Record.Hint.Text = "changed"
	require.Equal(t, "hint.text", r.Record.Hint.Text)
	loc := Localize(r, func(value, key string) string { return "translated" })
	require.Equal(t, "translated", loc.Record.Hint.Text)
	require.Equal(t, "translated", loc.Record.Hint.Title)
	require.Equal(t, "translated", loc.Record.Hint.Acknowledge)
	require.Equal(t, "translated", loc.Record.Hint.Close)
	require.Equal(t, "intro-v1", loc.Record.Hint.Key)
	require.Equal(t, "info", loc.Record.Hint.Icon)
	b, err := json.Marshal(RecordPage{})
	require.NoError(t, err)
	require.NotContains(t, string(b), "hint")
	require.NoError(t, (&PageHint{}).Validate())
	r.Record.Hint.Key = "bad\nkey"
	require.Error(t, r.Validate())
	require.NoError(t, (&TextBinding{Field: "created_at", Format: TextFormatShortDate}).Validate())
}
