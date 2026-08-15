package module

import (
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestRecordChildRouteBindsStandardSelector(t *testing.T) {
	id := pg.IntegerColumn("id")
	mod := &BaseModule{Name: "records", Path: "/api", PrimaryKey: id}
	query := standardRecordActionRouteQuery(mod, actions.ViewModuleAction{By: []pg.Column{pg.StringColumn("nick"), id}}, []pg.Column{pg.StringColumn("nick"), id})

	require.NotNil(t, query)
	require.Equal(t, "/api/records/view/:bykey/:value", query.Url)
	require.Equal(t, "id", query.Params["bykey"])
	require.Equal(t, "{id}", query.Params["value"])
}

func TestRecordChildRouteUsesFirstAllowedSelector(t *testing.T) {
	mod := &BaseModule{Name: "records", Path: "/api", PrimaryKey: pg.IntegerColumn("id")}
	by := []pg.Column{pg.StringColumn("nick")}
	query := standardRecordActionRouteQuery(mod, actions.UpdateModuleAction{By: by}, by)

	require.NotNil(t, query)
	require.Equal(t, "/api/records/:bykey/:value", query.Url)
	require.Equal(t, "nick", query.Params["bykey"])
	require.Equal(t, "{id}", query.Params["value"])
}

func TestFieldMatrixSourceResolvesStandardListAndDynamicUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := pg.IntegerColumn("id")
	code := pg.StringColumn("group_code")
	enabled := pg.BoolColumn("email_enabled")
	available := pg.BoolColumn("email_available")
	table := pg.NewTable("public", "preferences", "", id, code, enabled, available)
	target := &BaseModule{
		Name: "preferences", Path: "/api", Table: table, PrimaryKey: id,
		Fields: []fields.ModuleField{
			{Column: id, Type: fields.ModuleFieldTypeInt},
			{Column: code, Type: fields.ModuleFieldTypeString},
			{Column: enabled, Type: fields.ModuleFieldTypeBool},
			{Column: available, Type: fields.ModuleFieldTypeBool},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Permission: []actions.Role{actions.Role("member")}},
			actions.UpdateModuleAction{By: []pg.Column{id}, Columns: []pg.Column{enabled}, Permission: []actions.Role{actions.Role("member")}},
		},
	}
	section := renderer.FormSection{ID: "delivery", Renderer: renderer.RendererFieldMatrix, Matrix: &renderer.FieldMatrix{
		Type: renderer.FieldMatrixTypeTable,
		Table: &renderer.FieldMatrixTable{
			Heads: []string{"Type", "Email"},
			Rows:  []renderer.FieldMatrixRow{{ID: "chat", Label: "Chat", Cells: []renderer.FieldMatrixCell{{Field: "email_enabled", AvailableField: "email_available"}}}},
			Source: &renderer.FieldMatrixDataSource{
				IDField: "id", KeyField: "group_code",
				List:   renderer.ActionResource{Module: "preferences", Action: "list"},
				Update: renderer.ActionResource{Module: "preferences", Action: "update"},
			},
		},
	}}
	engine := gin.New()
	group := engine.Group("")
	generator := NewGenerator(nil, *group, []*BaseModule{target}, nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request = c.Request.WithContext(icontext.SetUser(c.Request.Context(), &icontext.UserInfo{ID: 1, Role: "member"}))

	require.NoError(t, generator.resolveFieldMatrixSource(c, &section, actions.Role("member")))
	require.NotNil(t, section.Matrix.Table.Source.Load)
	require.Equal(t, "GET", section.Matrix.Table.Source.Load.List.Request.Method)
	require.Equal(t, "/api/preferences", section.Matrix.Table.Source.Load.List.Request.Endpoint)
	require.Equal(t, "POST", section.Matrix.Table.Source.Load.Update.Request.Method)
	require.Equal(t, "/api/preferences/:bykey/:value", section.Matrix.Table.Source.Load.Update.Request.Endpoint)
	require.Empty(t, section.Matrix.Table.Source.Load.Update.Bindings)
}
