package renderer

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfirmCancelLabelJSONCloneAndLocalization(t *testing.T) {
	source := Universal{List: &ListPage{Actions: []Action{{
		ID: "delete",
		Confirm: &Confirm{
			Title:        "Delete record",
			Message:      "This cannot be undone",
			CancelLabel:  "Cancel",
			ConfirmLabel: "Delete",
		},
	}}}}

	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if got, want := string(encoded), `"cancel_label":"Cancel"`; !strings.Contains(got, want) {
		t.Fatalf("Marshal() = %s, want field %s", got, want)
	}

	localized := Localize(source, func(value, key string) string { return "localized: " + value })
	confirm := localized.List.Actions[0].Confirm
	if got, want := confirm.CancelLabel, "localized: Cancel"; got != want {
		t.Fatalf("localized cancel label = %q, want %q", got, want)
	}
	if got, want := source.List.Actions[0].Confirm.CancelLabel, "Cancel"; got != want {
		t.Fatalf("source cancel label = %q, want %q", got, want)
	}
}
