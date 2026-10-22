package module

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestListFilterRegistrySkipsUnusedOptionsAndKeepsValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/items", nil)
	column := pg.StringColumn("category")
	dynamicCalls := 0
	generator := &Generator{}
	module := &BaseModule{Fields: []fields.ModuleField{{
		Column: column, Title: "Category", Type: fields.ModuleFieldTypeArray,
		FormType: fields.ModuleFieldFormTypeMultiselect,
		OptionsFunc: func(*gin.Context) []fields.ModuleFieldOptions {
			dynamicCalls++
			return []fields.ModuleFieldOptions{{Value: "book", Label: "Book"}}
		},
	}}}
	action := actions.ListModuleAction{Filter: []pg.Column{column}}
	compact := generator.listFilterRegistry(c, module, action, locale.EN, false)
	require.Zero(t, dynamicCalls)
	require.Empty(t, compact["category"].Options)
	require.Equal(t, "{book}", generator.normalizeFilters(map[string]string{"category": "book"}, compact, locale.EN)["category"])
	complete := generator.effectiveListFilters(c, module, action, locale.EN)
	require.Equal(t, 1, dynamicCalls)
	require.Equal(t, "Category", complete["category"].Title)
	require.Equal(t, "book", complete["category"].Options[0].Value)
	require.Equal(t, complete["category"].Type, compact["category"].Type)
	require.Equal(t, complete["category"].FormType, compact["category"].FormType)
	require.Equal(t, complete["category"].Column, compact["category"].Column)
	// Building a metadata-free registry must not mutate either producer or
	// previously returned response metadata.
	generator.listFilterRegistry(c, module, action, locale.RU, false)
	require.Equal(t, 1, dynamicCalls)
	require.Equal(t, "Book", complete["category"].Options[0].Label)
}

func BenchmarkListFilterRegistry(b *testing.B) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/items", nil)
	module := &BaseModule{}
	action := actions.ListModuleAction{}
	for i := 0; i < 40; i++ {
		column := pg.StringColumn(fmt.Sprintf("column_%d", i))
		field := fields.ModuleField{Column: column, Type: fields.ModuleFieldTypeString}
		for j := 0; j < 8; j++ {
			field.Options = append(field.Options, fields.ModuleFieldOptions{Value: fmt.Sprint(j), Label: "Label", Description: "Description"})
		}
		module.Fields = append(module.Fields, field)
		action.Filter = append(action.Filter, column)
	}
	generator := &Generator{}
	for _, metadata := range []bool{true, false} {
		b.Run(fmt.Sprintf("metadata=%t", metadata), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				generator.listFilterRegistry(c, module, action, locale.EN, metadata)
			}
		})
	}
}

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
}
