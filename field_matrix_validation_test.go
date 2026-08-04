package module

import (
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestRenderForRejectsUnknownFieldMatrixReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := postgres.IntegerColumn("id")
	table := postgres.NewTable("public", "matrix_items", "", id)
	base := &BaseModule{
		Name:       "matrix-items",
		Table:      table,
		PrimaryKey: id,
		Fields:     []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
		Render: renderer.Universal{Form: &renderer.FormPage{Sections: []renderer.FormSection{{
			ID:       "rates",
			Renderer: renderer.RendererFieldMatrix,
			Matrix: &renderer.FieldMatrix{
				Type: renderer.FieldMatrixTypeList,
				List: &renderer.FieldMatrixList{Fields: []string{"missing"}, Columns: renderer.FieldMatrixColumnsOne},
			},
		}}}},
		RenderFunc: func(_ *gin.Context, render renderer.Universal) (renderer.Universal, error) { return render, nil },
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	_, err := base.RenderFor(c)
	require.EqualError(t, err, `renderer.Universal: matrix section "rates" references unknown field "missing"`)
}

func TestRunRejectsUnknownStaticFieldMatrixReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := postgres.IntegerColumn("id")
	table := postgres.NewTable("public", "matrix_items", "", id)
	base := &BaseModule{
		Name:       "matrix-items",
		Table:      table,
		PrimaryKey: id,
		Fields:     []fields.ModuleField{{Column: id, Type: fields.ModuleFieldTypeInt}},
		Render: renderer.Universal{Form: &renderer.FormPage{Sections: []renderer.FormSection{{
			ID:       "rates",
			Renderer: renderer.RendererFieldMatrix,
			Matrix: &renderer.FieldMatrix{
				Type: renderer.FieldMatrixTypeList,
				List: &renderer.FieldMatrixList{Fields: []string{"missing"}, Columns: renderer.FieldMatrixColumnsOne},
			},
		}}}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := NewGenerator(nil, *group, []*BaseModule{base}, nil, nil)

	require.PanicsWithValue(t,
		`invalid field matrix config in module matrix-items: renderer.Universal: matrix section "rates" references unknown field "missing"`,
		func() { generator.Run() },
	)
}

func TestLocalizeRendererFieldMatrix(t *testing.T) {
	generator := &Generator{translations: map[locale.Lang]map[string]string{
		locale.EN: {
			"matrix.duration": "Duration",
			"matrix.incall":   "In-call",
			"matrix.1h":       "1 hour",
			"matrix.none":     "Not available",
		},
	}}
	render := generator.localizeRenderer(locale.EN, renderer.Universal{Form: &renderer.FormPage{Sections: []renderer.FormSection{{
		ID:       "rates",
		Renderer: renderer.RendererFieldMatrix,
		Matrix: &renderer.FieldMatrix{
			Type: renderer.FieldMatrixTypeTable,
			Table: &renderer.FieldMatrixTable{
				Heads: []string{"matrix.duration", "matrix.incall", "matrix.outcall"},
				Rows:  []renderer.FieldMatrixRow{{Label: "matrix.1h", Cells: []renderer.FieldMatrixCell{{Text: "matrix.none"}, {Field: "incall_1h_price"}}}},
			},
		},
	}}}})

	matrix := render.Form.Sections[0].Matrix.Table
	require.Equal(t, "Duration", matrix.Heads[0])
	require.Equal(t, "In-call", matrix.Heads[1])
	require.Equal(t, "1 hour", matrix.Rows[0].Label)
	require.Equal(t, "Not available", matrix.Rows[0].Cells[0].Text)
	require.Equal(t, "incall_1h_price", matrix.Rows[0].Cells[1].Field)
}
