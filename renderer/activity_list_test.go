package renderer

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestActivityListPresentationIsTypedAndLocalized(t *testing.T) {
	markerVisible := true
	source := Universal{List: &ListPage{
		Summary: &Summary{Resource: &Resource{
			ActionResource: ActionResource{Module: "activity_summary", Action: "view"},
			Bindings: []RequestBinding{{
				Target: RequestBindingPathValue,
				Source: ValueSource{Runtime: &RuntimeValue{Scope: RuntimeValueSourceCurrentUser, Field: "id"}},
			}},
		}},
		Filters: &Filters{PillRows: [][]FilterPill{{
			{Label: "filter.all", CountField: "all_count"},
			{Label: "filter.messages", Key: "category", Val: "messages", CountField: "messages_count"},
		}}},
		GroupBy: &ListGroupBy{
			Field:          "created_at",
			Type:           ListGroupByDate,
			TodayLabel:     "group.today",
			YesterdayLabel: "group.yesterday",
		},
		Grid: &Grid{Mode: GridModeList},
		CardSchema: &CardSchema{
			Variant: CardVariantActivity,
			Icon: &IconBinding{
				Field:   "kind",
				IconMap: map[string]string{"message": "chat"},
				ToneMap: map[string]string{"message": "success"},
				Marker:  &IconMarker{VisibleIf: &Condition{Path: "record.read", Falsy: &markerVisible}, Tone: "cyan"},
			},
			Meta: &TextBinding{Field: "created_at", Format: TextFormatRelativeTime},
			Badges: []Badge{{
				Field:     "priority",
				LabelMap:  map[string]string{"high": "priority.high"},
				Variant:   "priority",
				VisibleIf: &Condition{Path: "record.priority", In: []interface{}{"critical", "high"}},
			}},
		},
	}}

	if err := source.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	localized := Localize(source, func(value, _ string) string { return "localized:" + value })
	if got := localized.List.GroupBy.TodayLabel; got != "localized:group.today" {
		t.Fatalf("TodayLabel = %q", got)
	}
	if got := localized.List.CardSchema.Badges[0].LabelMap["high"]; got != "localized:priority.high" {
		t.Fatalf("LabelMap = %q", got)
	}

	cloned := source.Clone()
	cloned.List.CardSchema.Icon.IconMap["message"] = "bell"
	if source.List.CardSchema.Icon.IconMap["message"] != "chat" {
		t.Fatal("Clone() shares icon map")
	}
	cloned.List.CardSchema.Icon.Marker.Tone = "pink"
	if source.List.CardSchema.Icon.Marker.Tone != "cyan" {
		t.Fatal("Clone() shares icon marker")
	}
	cloned.List.CardSchema.Badges[0].VisibleIf.Path = "record.other"
	if source.List.CardSchema.Badges[0].VisibleIf.Path != "record.priority" {
		t.Fatal("Clone() shares badge visibility condition")
	}
	cloned.List.Summary.Resource.Bindings[0].Source.Runtime.Field = "other"
	if source.List.Summary.Resource.Bindings[0].Source.Runtime.Field != "id" {
		t.Fatal("Clone() shares summary resource bindings")
	}

	payload, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(payload) == "" {
		t.Fatal("Marshal() returned an empty payload")
	}
	if !strings.Contains(string(payload), `"count_field":"messages_count"`) {
		t.Fatalf("Marshal() omitted pill count field: %s", payload)
	}
	if strings.Contains(string(payload), `"activity_summary"`) {
		t.Fatalf("Marshal() exposed producer summary resource: %s", payload)
	}
}

func TestActivityListPresentationValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		value Universal
		want  string
	}{
		{
			name:  "missing group field",
			value: Universal{List: &ListPage{GroupBy: &ListGroupBy{Type: ListGroupByDate}}},
			want:  "renderer.Universal: list page: renderer.ListGroupBy: field is required",
		},
		{
			name:  "unsupported group type",
			value: Universal{List: &ListPage{GroupBy: &ListGroupBy{Field: "created_at", Type: ListGroupByType("value")}}},
			want:  "renderer.Universal: list page: renderer.ListGroupBy: unsupported type \"value\"",
		},
		{
			name:  "icon without field",
			value: Universal{List: &ListPage{CardSchema: &CardSchema{Icon: &IconBinding{Fallback: "info"}}}},
			want:  "renderer.CardSchema: icon field is required",
		},
		{
			name:  "unsupported text format",
			value: Universal{List: &ListPage{CardSchema: &CardSchema{Meta: &TextBinding{Field: "created_at", Format: TextFormat("date")}}}},
			want:  "renderer.TextBinding: unsupported format \"date\"",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.value.Validate(); err == nil || err.Error() != test.want {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}
