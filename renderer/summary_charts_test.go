package renderer

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRecordSummaryCharts(t *testing.T) {
	for _, kind := range []SummaryChartType{SummaryChartLine, SummaryChartBar, SummaryChartDonut, SummaryChartGauge} {
		collapsed := false
		summary := &Summary{Collapsible: &collapsed, Columns: 2, EmptyLabel: "missing", Items: []SummaryItem{{ID: "total", LabelKey: "total", ValueField: "metrics.total", PointsField: "metrics.history"}}, Trend: &SummaryTrend{Type: kind, Series: []SummaryTrendSeries{{ID: "a", LabelKey: "a", PointsField: "points"}}}}
		value := Universal{Record: &RecordPage{Sections: []RecordSection{{ID: "overview", Summary: summary}}}}
		require.NoError(t, value.Validate())
		clone := value.Clone()
		*clone.Record.Sections[0].Summary.Collapsible = true
		clone.Record.Sections[0].Summary.Trend.Series[0].PointsField = "changed"
		require.False(t, *value.Record.Sections[0].Summary.Collapsible)
		require.Equal(t, "points", value.Record.Sections[0].Summary.Trend.Series[0].PointsField)
		localized := Localize(value, func(value, key string) string {
			if key != "" {
				return "tr:" + key
			}
			return value
		})
		require.Equal(t, "tr:total", localized.Record.Sections[0].Summary.Items[0].Label)
		data, err := json.Marshal(localized)
		require.NoError(t, err)
		require.Contains(t, string(data), `"type":"`+string(kind)+`"`)
	}
}

func TestSummaryChartInvalidContracts(t *testing.T) {
	for _, summary := range []*Summary{
		{Columns: 5}, {RangeActionID: "range"},
		{Trend: &SummaryTrend{Type: "pie"}},
		{Trend: &SummaryTrend{Type: SummaryChartDonut, Horizontal: true}},
		{Trend: &SummaryTrend{Type: SummaryChartGauge}},
	} {
		require.Error(t, summary.Validate())
	}
	value := Universal{Record: &RecordPage{Sections: []RecordSection{{Summary: &Summary{DateRange: &DateRangeToolbar{Field: "range"}, RangeActionID: "missing"}}}}}
	require.Error(t, value.Validate())
}

func TestSummarySelectedRangeBindingSurvivesCloneAndLocalization(t *testing.T) {
	value := Universal{Record: &RecordPage{Sections: []RecordSection{{Summary: &Summary{DateRange: &DateRangeToolbar{Field: "range", ValueField: "metrics.range"}}}}}}
	require.NoError(t, value.Validate())
	clone := value.Clone()
	clone.Record.Sections[0].Summary.DateRange.ValueField = "other.range"
	require.Equal(t, "metrics.range", value.Record.Sections[0].Summary.DateRange.ValueField)
	localized := Localize(value, func(value, key string) string { return value })
	data, err := json.Marshal(localized)
	require.NoError(t, err)
	require.Contains(t, string(data), `"value_field":"metrics.range"`)
}
