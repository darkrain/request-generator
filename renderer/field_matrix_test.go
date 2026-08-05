package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldMatrixValidate(t *testing.T) {
	tests := []struct {
		name   string
		matrix *FieldMatrix
		valid  bool
	}{
		{
			name: "table",
			matrix: &FieldMatrix{
				Type: FieldMatrixTypeTable,
				Table: &FieldMatrixTable{
					Heads: []string{"matrix.duration", "matrix.incall", "matrix.outcall"},
					Rows:  []FieldMatrixRow{{Label: "matrix.1h", Cells: []FieldMatrixCell{{Field: "incall_1h_price"}, {Field: "outcall_1h_price"}}}},
				},
			},
			valid: true,
		},
		{
			name: "list",
			matrix: &FieldMatrix{
				Type: FieldMatrixTypeList,
				List: &FieldMatrixList{Fields: []string{"first", "second"}, Columns: FieldMatrixColumnsTwo},
			},
			valid: true,
		},
		{
			name: "table cell must select one value source",
			matrix: &FieldMatrix{
				Type: FieldMatrixTypeTable,
				Table: &FieldMatrixTable{
					Heads: []string{"matrix.duration"},
					Rows:  []FieldMatrixRow{{Cells: []FieldMatrixCell{{Field: "price", Text: "matrix.price"}}}},
				},
			},
		},
		{
			name: "list columns are closed enum",
			matrix: &FieldMatrix{
				Type: FieldMatrixTypeList,
				List: &FieldMatrixList{Fields: []string{"first"}, Columns: 5},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.matrix.Validate("rates")
			assert.Equal(t, test.valid, err == nil, err)
		})
	}
}

func TestUniversalValidateFieldMatrixRendererBinding(t *testing.T) {
	tests := []struct {
		name    string
		section FormSection
		valid   bool
	}{
		{
			name:    "matrix renderer without matrix",
			section: FormSection{ID: "rates", Renderer: RendererFieldMatrix},
		},
		{
			name: "matrix with another renderer",
			section: FormSection{ID: "rates", Renderer: RendererUniversalSection, Matrix: &FieldMatrix{
				Type: FieldMatrixTypeList,
				List: &FieldMatrixList{Fields: []string{"price"}, Columns: FieldMatrixColumnsOne},
			}},
		},
		{
			name: "matrix renderer with matrix",
			section: FormSection{ID: "rates", Renderer: RendererFieldMatrix, Matrix: &FieldMatrix{
				Type: FieldMatrixTypeList,
				List: &FieldMatrixList{Fields: []string{"price"}, Columns: FieldMatrixColumnsOne},
			}},
			valid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (Universal{Form: &FormPage{Sections: []FormSection{test.section}}}).Validate()
			assert.Equal(t, test.valid, err == nil, err)
		})
	}
}

func TestFieldMatrixClone(t *testing.T) {
	original := Universal{Form: &FormPage{Sections: []FormSection{{
		ID: "rates",
		Matrix: &FieldMatrix{
			Type:  FieldMatrixTypeTable,
			Table: &FieldMatrixTable{Heads: []string{"matrix.duration"}, Rows: []FieldMatrixRow{{Cells: []FieldMatrixCell{{Field: "price"}}}}},
		},
	}}}}
	form := original.Clone().Form

	form.Sections[0].Matrix.Table.Heads[0] = "changed"
	assert.Equal(t, "matrix.duration", original.Form.Sections[0].Matrix.Table.Heads[0])
}
