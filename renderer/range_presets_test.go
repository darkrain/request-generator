package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniversalValidateRejectsInvalidFilterRangePresets(t *testing.T) {
	tests := []struct {
		name    string
		filters *Filters
		err     string
	}{
		{
			name:    "empty field",
			filters: &Filters{Primary: []string{"rating"}, RangePresets: []FilterRangePresets{{Presets: []FilterRangePreset{{Min: 0, Max: 5}}}}},
			err:     "renderer.Universal: list page range presets field is required",
		},
		{
			name:    "undeclared field",
			filters: &Filters{Primary: []string{"rating"}, RangePresets: []FilterRangePresets{{Field: "price", Presets: []FilterRangePreset{{Min: 0, Max: 5}}}}},
			err:     `renderer.Universal: list page range presets field "price" is not declared in filters`,
		},
		{
			name:    "duplicate field",
			filters: &Filters{Primary: []string{"rating"}, RangePresets: []FilterRangePresets{{Field: "rating", Presets: []FilterRangePreset{{Min: 0, Max: 5}}}, {Field: "rating", Presets: []FilterRangePreset{{Min: 5, Max: 10}}}}},
			err:     `renderer.Universal: list page range presets field "rating" is duplicated`,
		},
		{
			name:    "empty presets",
			filters: &Filters{Primary: []string{"rating"}, RangePresets: []FilterRangePresets{{Field: "rating"}}},
			err:     `renderer.Universal: list page range presets field "rating" must have at least one preset`,
		},
		{
			name:    "reversed bounds",
			filters: &Filters{Primary: []string{"rating"}, RangePresets: []FilterRangePresets{{Field: "rating", Presets: []FilterRangePreset{{Min: 5, Max: 0}}}}},
			err:     `renderer.Universal: list page range preset for field "rating" has min greater than max`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Universal{List: &ListPage{Filters: test.filters}}).Validate()
			require.EqualError(t, err, test.err)
		})
	}

	t.Run("field declared in group", func(t *testing.T) {
		filters := &Filters{Groups: []FilterGroup{{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"rating"}}}, RangePresets: []FilterRangePresets{{Field: "rating", Presets: []FilterRangePreset{{Min: 0, Max: 5}}}}}
		require.NoError(t, (Universal{List: &ListPage{Filters: filters}}).Validate())
	})
}
