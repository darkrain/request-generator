package module

import (
	"testing"

	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsInvalidFieldOptionsSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := postgres.IntegerColumn("id")
	status := postgres.StringColumn("status")
	table := postgres.NewTable("public", "option_items", "", id, status)
	base := &BaseModule{
		Name:       "option-items",
		Table:      table,
		PrimaryKey: id,
		Fields: []fields.ModuleField{{
			Column:        status,
			OptionsSource: &fields.FieldOptionsSource{Mode: fields.FieldOptionsSourceModeTree},
		}},
	}
	engine := gin.New()
	group := engine.Group("")
	generator := NewGenerator(nil, *group, []*BaseModule{base}, nil, nil)

	require.PanicsWithValue(t,
		`invalid field options source in module option-items: field "status": field options source endpoint is required`,
		func() { generator.Run() },
	)
}
