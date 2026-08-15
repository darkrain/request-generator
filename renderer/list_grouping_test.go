package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListGroupingValidation(t *testing.T) {
	t.Run("accepts a row label field", func(t *testing.T) {
		err := (Universal{List: &ListPage{GroupBy: &ListGrouping{Field: "day_label"}}}).Validate()
		require.NoError(t, err)
	})

	t.Run("requires a row label field", func(t *testing.T) {
		err := (Universal{List: &ListPage{GroupBy: &ListGrouping{}}}).Validate()
		require.EqualError(t, err, "renderer.ListPage: group_by.field is required")
	})
}
