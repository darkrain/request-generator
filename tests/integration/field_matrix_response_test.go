package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	dbpkg "github.com/darkrain/request-generator/db"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/locale"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestFieldMatrixViewResponseLocalizesPublicContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := pg.IntegerColumn("id")
	price := pg.IntegerColumn("price")
	table := pg.NewTable("public", "matrix_items", "", id, price)
	testModule := &module.BaseModule{
		Name:       "matrix-items",
		Path:       "/admin",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Title: "matrix.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeOnlyView},
			{Column: price, Title: "matrix.price", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeNumber},
		},
		Render: renderer.Universal{Form: &renderer.FormPage{Sections: []renderer.FormSection{{
			ID:       "rates",
			Renderer: renderer.RendererFieldMatrix,
			Matrix: &renderer.FieldMatrix{
				Type: renderer.FieldMatrixTypeTable,
				Table: &renderer.FieldMatrixTable{
					Heads: []string{"matrix.duration", "matrix.price", "matrix.note"},
					Rows: []renderer.FieldMatrixRow{{
						Label: "matrix.1h",
						Cells: []renderer.FieldMatrixCell{{Field: "price"}, {Text: "matrix.none"}},
					}},
				},
			},
		}}}},
		Actions: []actions.ModuleAction{actions.ViewModuleAction{
			Columns:    []pg.Column{id, price},
			By:         []pg.Column{id},
			Permission: []actions.Role{actions.RoleAll},
			Auth:       true,
			PageType:   renderer.PageTypeForm,
		}},
	}

	translationsPath := filepath.Join(t.TempDir(), "en.json")
	require.NoError(t, os.WriteFile(translationsPath, []byte(`{"matrix":{"id":"ID","price":"Price","duration":"Duration","note":"Note","1h":"1 hour","none":"Not available"}}`), 0o600))
	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		func(_ *module.BaseModule) dbpkg.DBExecutor { return fakeRendererDB{} },
		*group,
		[]*module.BaseModule{testModule},
		func(_ actions.ModuleAction, _ []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) { c.Next() }
		},
		createMockAuthMiddleware(&icontext.UserInfo{ID: 1, Role: "admin"}),
	)
	generator.Locales = []locale.Lang{locale.EN}
	require.NoError(t, generator.LoadTranslationsFile(locale.EN, translationsPath))
	generator.Run()

	w := executeRequest(engine, http.MethodGet, "/admin/matrix-items/view/id/1?lang=en", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var response struct {
		FormPage *renderer.FormPage `json:"form_page"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.NotNil(t, response.FormPage)
	require.Len(t, response.FormPage.Sections, 1)
	matrix := response.FormPage.Sections[0].Matrix
	require.NotNil(t, matrix)
	require.Equal(t, []string{"Duration", "Price", "Note"}, matrix.Table.Heads)
	require.Equal(t, "1 hour", matrix.Table.Rows[0].Label)
	require.Equal(t, "price", matrix.Table.Rows[0].Cells[0].Field)
	require.Equal(t, "Not available", matrix.Table.Rows[0].Cells[1].Text)
}
