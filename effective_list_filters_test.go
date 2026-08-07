package module

import (
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestEffectiveListFilters_VirtualDefinitionOverridesModuleField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ethnicity := pg.StringColumn("ethnicity")
	request := httptest.NewRequest("GET", "/items", nil)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	generator := &Generator{}
	registry := generator.effectiveListFilters(context, &BaseModule{Fields: []fields.ModuleField{{
		Column: ethnicity, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeSelect,
	}}}, actions.ListModuleAction{
		Filter: []pg.Column{ethnicity},
		VirtualFilters: []fields.ModuleFilterField{{
			FieldName: "ethnicity", Column: ethnicity, Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeMultiselect,
		}},
	}, locale.EN)

	require.Equal(t, fields.ModuleFieldFormTypeMultiselect, registry["ethnicity"].FormType)
	normalized := generator.normalizeFilters(map[string]string{"ethnicity": "{asian,latin}"}, registry, locale.EN)
	require.Equal(t, "{asian,latin}", normalized["ethnicity"])
	effective := effectiveListFilterFields(registry)
	require.Len(t, effective, 1)
	require.Equal(t, fields.ModuleFieldFormTypeMultiselect, effective[0].FormType)
}
