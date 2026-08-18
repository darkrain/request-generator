package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummaryTrendUsesTheExistingSummaryResource(t *testing.T) {
	value := Universal{List: &ListPage{Summary: &Summary{
		Presentation: SummaryPresentationDashboard,
		Items: []SummaryItem{{
			ID: "total", Label: "summary.total", ValueField: "total_display",
			ChangeField: "total_change", DirectionField: "total_direction", Icon: "chart", Tone: "cyan",
		}},
		Trend: &SummaryTrend{
			PointsField: "points", PeriodField: "period", AriaLabelKey: "summary.chart",
			EmptyLabelKey: "summary.empty", LoadingLabelKey: "summary.loading", Tone: "pink",
		},
	}}}

	require.NoError(t, value.Validate())
	localized := Localize(value, func(value, key string) string {
		if key != "" {
			return "localized:" + key
		}
		return "localized:" + value
	})
	require.Equal(t, "localized:summary.total", localized.List.Summary.Items[0].Label)
	require.Equal(t, "localized:summary.chart", localized.List.Summary.Trend.AriaLabel)
	require.Equal(t, "localized:summary.empty", localized.List.Summary.Trend.EmptyLabel)
	require.Equal(t, "localized:summary.loading", localized.List.Summary.Trend.LoadingLabel)
	require.Empty(t, localized.List.Summary.Trend.AriaLabelKey)

	clone := value.Clone()
	clone.List.Summary.Trend.PointsField = "other_points"
	require.Equal(t, "points", value.List.Summary.Trend.PointsField)

	payload, err := json.Marshal(value)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"points_field":"points"`)
}

func TestSummaryTrendRequiresPointsField(t *testing.T) {
	err := (Universal{List: &ListPage{Summary: &Summary{Trend: &SummaryTrend{Tone: "cyan"}}}}).Validate()
	require.EqualError(t, err, "renderer.Universal: list page: renderer.Summary: trend points field is required")
}

func TestSummaryRejectsUnknownPresentation(t *testing.T) {
	err := (Universal{List: &ListPage{Summary: &Summary{Presentation: "hero"}}}).Validate()
	require.EqualError(t, err, `renderer.Universal: list page: renderer.Summary: unsupported presentation "hero"`)
}

func TestDateRangeSectionBindsDeclaredOrdinaryFields(t *testing.T) {
	value := Universal{Form: &FormPage{
		Fields: []string{"starts_on", "ends_on", "title"},
		Sections: []FormSection{{
			ID: "dates", Renderer: RendererDateRange, Fields: []string{"starts_on", "ends_on"},
			DateRange: &DateRangeConfig{
				StartField: "starts_on", EndField: "ends_on", Min: "2026-01-01", Max: "2027-01-01",
				DisabledDates: []string{"2026-12-31"}, Placeholder: "dates.placeholder", ApplyLabel: "ui.apply",
				CancelLabel: "ui.cancel", StartLabel: "dates.start", EndLabel: "dates.end", EmptyLabel: "ui.empty",
				Months:   []string{"month.1", "month.2", "month.3", "month.4", "month.5", "month.6", "month.7", "month.8", "month.9", "month.10", "month.11", "month.12"},
				Weekdays: []string{"weekday.1", "weekday.2", "weekday.3", "weekday.4", "weekday.5", "weekday.6", "weekday.7"},
			},
		}},
	}}

	require.NoError(t, value.Validate())
	localized := Localize(value, func(value, _ string) string { return "localized:" + value })
	require.Equal(t, "localized:dates.placeholder", localized.Form.Sections[0].DateRange.Placeholder)
	require.Equal(t, "localized:month.1", localized.Form.Sections[0].DateRange.Months[0])
	require.Equal(t, "localized:weekday.7", localized.Form.Sections[0].DateRange.Weekdays[6])

	clone := value.Clone()
	clone.Form.Sections[0].DateRange.Months[0] = "changed"
	require.Equal(t, "month.1", value.Form.Sections[0].DateRange.Months[0])
}

func TestDateRangeSectionRejectsInvalidContracts(t *testing.T) {
	base := func() Universal {
		return Universal{Form: &FormPage{Fields: []string{"from", "to"}, Sections: []FormSection{{
			ID: "dates", Renderer: RendererDateRange, Fields: []string{"from", "to"},
			DateRange: &DateRangeConfig{StartField: "from", EndField: "to"},
		}}}}
	}

	tests := []struct {
		name string
		edit func(*DateRangeConfig)
		want string
	}{
		{name: "same field", edit: func(config *DateRangeConfig) { config.EndField = "from" }, want: `renderer.Universal: date range section "dates" must define distinct start and end fields`},
		{name: "unknown field", edit: func(config *DateRangeConfig) { config.EndField = "unknown" }, want: `renderer.Universal: date range section "dates" field "unknown" is not declared by the form`},
		{name: "months", edit: func(config *DateRangeConfig) { config.Months = []string{"one"} }, want: `renderer.Universal: date range section "dates" months must contain 12 values`},
		{name: "weekdays", edit: func(config *DateRangeConfig) { config.Weekdays = []string{"one"} }, want: `renderer.Universal: date range section "dates" weekdays must contain 7 values`},
		{name: "date", edit: func(config *DateRangeConfig) { config.Min = "2026-99-99" }, want: `renderer.Universal: date range section "dates" date "2026-99-99" must use YYYY-MM-DD`},
	}
	for _, current := range tests {
		t.Run(current.name, func(t *testing.T) {
			value := base()
			current.edit(value.Form.Sections[0].DateRange)
			require.EqualError(t, value.Validate(), current.want)
		})
	}
}

func TestStatusTimelineUsesOneDeclaredField(t *testing.T) {
	require.NoError(t, (DisplayComponent{Type: DisplayStatusTimeline, Fields: []string{"history"}}).Validate())
	require.EqualError(t, (DisplayComponent{Type: DisplayStatusTimeline}).Validate(), "status timeline requires exactly one field")
	require.EqualError(t, (DisplayComponent{Type: DisplayStatusTimeline, Fields: []string{"one", "two"}}).Validate(), "status timeline requires exactly one field")
}
