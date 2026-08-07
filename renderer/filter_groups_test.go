package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniversalValidateFilterGroups(t *testing.T) {
	tests := []struct {
		name    string
		group   FilterGroup
		errText string
	}{
		{name: "missing id", group: FilterGroup{Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}}, errText: "renderer.Universal: list page filter group id is required"},
		{name: "missing label", group: FilterGroup{ID: "price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}}, errText: `renderer.Universal: list page filter group "price" label is required`},
		{name: "invalid placement", group: FilterGroup{ID: "price", Label: "Price", Placement: "topbar", Fields: []string{"price"}}, errText: `renderer.Universal: list page filter group "price" has invalid placement "topbar"`},
		{name: "empty fields", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary}, errText: `renderer.Universal: list page filter group "price" must contain at least one field`},
		{name: "duplicate field", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price", "price"}}, errText: `renderer.Universal: list page filter group "price" contains duplicate field "price"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Universal{List: &ListPage{Filters: &Filters{Groups: []FilterGroup{test.group}}}}).Validate()
			require.EqualError(t, err, test.errText)
		})
	}

	t.Run("duplicate ids", func(t *testing.T) {
		filters := &Filters{Groups: []FilterGroup{
			{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}},
			{ID: "price", Label: "Options", Placement: FilterGroupPlacementPrimary, Fields: []string{"option"}},
		}}
		err := (Universal{List: &ListPage{Filters: filters}}).Validate()
		require.EqualError(t, err, `renderer.Universal: list page filter group "price" is duplicated`)
	})
}
