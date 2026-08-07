package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniversalValidateFilterGroups(t *testing.T) {
	validationTests := []struct {
		name    string
		group   FilterGroup
		errText string
	}{
		{name: "missing id", group: FilterGroup{Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}}, errText: "renderer.Universal: list page filter group id is required"},
		{name: "missing label", group: FilterGroup{ID: "price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}}, errText: `renderer.Universal: list page filter group "price" label is required`},
		{name: "invalid placement", group: FilterGroup{ID: "price", Label: "Price", Placement: "topbar", Fields: []string{"price"}}, errText: `renderer.Universal: list page filter group "price" has invalid placement "topbar"`},
		{name: "empty fields", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary}, errText: `renderer.Universal: list page filter group "price" must contain at least one field`},
		{name: "duplicate field", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price", "price"}}, errText: `renderer.Universal: list page filter field "price" is declared in both group "price" and group "price"`},
	}

	for _, test := range validationTests {
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

	t.Run("field repeated in flat placement and group", func(t *testing.T) {
		filters := &Filters{
			Primary: []string{"price"},
			Groups:  []FilterGroup{{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}}},
		}
		err := (Universal{List: &ListPage{Filters: filters}}).Validate()
		require.EqualError(t, err, `renderer.Universal: list page filter field "price" is declared in both primary and group "price"`)
	})

	t.Run("field repeated in two groups", func(t *testing.T) {
		filters := &Filters{Groups: []FilterGroup{
			{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}},
			{ID: "options", Label: "Options", Placement: FilterGroupPlacementPrimary, Fields: []string{"price"}},
		}}
		err := (Universal{List: &ListPage{Filters: filters}}).Validate()
		require.EqualError(t, err, `renderer.Universal: list page filter field "price" is declared in both group "price" and group "options"`)
	})

	presentationTests := []struct {
		name    string
		group   FilterGroup
		errText string
	}{
		{name: "unknown presentation", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Presentation: "accordion", Sections: []FilterGroupSection{{ID: "incall", Label: "Incall", Fields: []string{"price"}}}}, errText: `renderer.Universal: list page filter group "price" has invalid presentation "accordion"`},
		{name: "sections without presentation", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Sections: []FilterGroupSection{{ID: "incall", Label: "Incall", Fields: []string{"price"}}}}, errText: `renderer.Universal: list page filter group "price" sections require a presentation`},
		{name: "tabs with fields", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Presentation: FilterGroupPresentationTabs, Fields: []string{"price"}, Sections: []FilterGroupSection{{ID: "incall", Label: "Incall", Fields: []string{"other_price"}}}}, errText: `renderer.Universal: list page filter group "price" with presentation "tabs" must use sections instead of fields`},
		{name: "tabs without sections", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Presentation: FilterGroupPresentationTabs}, errText: `renderer.Universal: list page filter group "price" with presentation "tabs" must contain at least one section`},
		{name: "section without label", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Presentation: FilterGroupPresentationTabs, Sections: []FilterGroupSection{{ID: "incall", Fields: []string{"price"}}}}, errText: `renderer.Universal: list page filter group "price" section "incall" label is required`},
		{name: "field repeated in sections", group: FilterGroup{ID: "price", Label: "Price", Placement: FilterGroupPlacementPrimary, Presentation: FilterGroupPresentationTabs, Sections: []FilterGroupSection{{ID: "incall", Label: "Incall", Fields: []string{"price"}}, {ID: "outcall", Label: "Outcall", Fields: []string{"price"}}}}, errText: `renderer.Universal: list page filter field "price" is declared in both group "price" section "incall" and group "price" section "outcall"`},
	}
	for _, test := range presentationTests {
		t.Run(test.name, func(t *testing.T) {
			err := (Universal{List: &ListPage{Filters: &Filters{Groups: []FilterGroup{test.group}}}}).Validate()
			require.EqualError(t, err, test.errText)
		})
	}
}
