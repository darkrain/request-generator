package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGridColumnsSerializeResponsiveLayouts(t *testing.T) {
	tests := []struct {
		name    string
		columns GridColumns
		want    string
	}{
		{
			name:    "wide cards",
			columns: GridColumns{Desktop: GridColumnsTwo, Tablet: GridColumnsTwo, Mobile: GridColumnsOne},
			want:    `{"list_page":{"grid":{"enabled":true,"mode":"cards","columns":{"desktop":2,"tablet":2,"mobile":1}}}}`,
		},
		{
			name:    "dense cards",
			columns: GridColumns{Desktop: GridColumnsSix, Tablet: GridColumnsThree, Mobile: GridColumnsTwo},
			want:    `{"list_page":{"grid":{"enabled":true,"mode":"cards","columns":{"desktop":6,"tablet":3,"mobile":2}}}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := Universal{List: &ListPage{Grid: &Grid{Enabled: true, Mode: GridModeCards, Columns: &test.columns}}}
			require.NoError(t, value.Validate())
			payload, err := json.Marshal(value)
			require.NoError(t, err)
			require.JSONEq(t, test.want, string(payload))
		})
	}
}

func TestGridValidation(t *testing.T) {
	tests := []struct {
		name string
		grid Grid
		want string
	}{
		{
			name: "unsupported mode",
			grid: Grid{Mode: GridMode("masonry")},
			want: `renderer.Universal: list page: renderer.Grid: unsupported mode "masonry"`,
		},
		{
			name: "zero value",
			grid: Grid{Mode: GridModeCards, Columns: &GridColumns{Desktop: 0, Tablet: GridColumnsTwo, Mobile: GridColumnsOne}},
			want: "renderer.Universal: list page: renderer.Grid: columns.desktop must be between 1 and 6",
		},
		{
			name: "out of range",
			grid: Grid{Mode: GridModeCards, Columns: &GridColumns{Desktop: 7, Tablet: GridColumnsTwo, Mobile: GridColumnsOne}},
			want: "renderer.Universal: list page: renderer.Grid: columns.desktop must be between 1 and 6",
		},
		{
			name: "incomplete",
			grid: Grid{Mode: GridModeCards, Columns: &GridColumns{Desktop: GridColumnsTwo, Mobile: GridColumnsOne}},
			want: "renderer.Universal: list page: renderer.Grid: columns.tablet must be between 1 and 6",
		},
		{
			name: "non-monotonic",
			grid: Grid{Mode: GridModeCards, Columns: &GridColumns{Desktop: GridColumnsTwo, Tablet: GridColumnsThree, Mobile: GridColumnsOne}},
			want: "renderer.Universal: list page: renderer.Grid: columns must satisfy mobile <= tablet <= desktop",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Universal{List: &ListPage{Grid: &test.grid}}).Validate()
			require.EqualError(t, err, test.want)
		})
	}
}

func TestGridColumnsCloneIsIndependent(t *testing.T) {
	source := Universal{List: &ListPage{Grid: &Grid{
		Mode:    GridModeCards,
		Columns: &GridColumns{Desktop: GridColumnsSix, Tablet: GridColumnsThree, Mobile: GridColumnsTwo},
	}}}

	cloned := source.Clone()
	cloned.List.Grid.Columns.Desktop = GridColumnsTwo

	require.Equal(t, GridColumnsSix, source.List.Grid.Columns.Desktop)
}
