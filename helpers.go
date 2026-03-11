package module

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

func (generator *Generator) getPagination(page int64, size int64) (int64, int64, int64) {
	var limit int64
	if size <= 0 {
		limit = 10
	} else {
		limit = size
	}
	if page <= 0 {
		page = 0
	}
	offset := page * limit

	return limit, offset, page
}

func (generator *Generator) normalizeFilters(data map[string]string, module *BaseModule, listAction actions.ListModuleAction, lang locale.Lang) map[string]string {
	resultFilterMap := make(map[string]string)

	filters := make(map[string]fields.ModuleField)
	for _, realField := range module.Fields {
		if containsColumn(listAction.Filter, realField.Column) {
			filters[realField.ColumnName()] = realField
		}
	}

parentLoop:
	for _, filter := range filters {
		filterValue, ok := data[filter.ColumnName()]
		if !ok || len(filterValue) == 0 {
			continue
		}

		for _, rule := range filter.Check {
			if err := rule.Validate(filterValue, string(lang)); err != nil {
				continue parentLoop
			}
		}

		resultFilterMap[filter.ColumnName()] = filterValue
	}

	for key, value := range data {
		result := strings.Split(key, ".")
		if len(result) > 1 {
			resultFilterMap[key] = value
		}
	}

	return resultFilterMap
}

func (generator *Generator) checkRequest(
	context *gin.Context,
	data map[string]interface{},
	module *BaseModule,
	action actions.ModuleAction,
	scenario fields.Scenario,
	lang locale.Lang,
) map[string]string {
	errs := make(map[string]string)
	actionColumns := action.GetColumns(context)

	for _, col := range actionColumns {
		colName := col.Name()
		value := data[colName]
		field := module.GetField(colName)
		if field == nil {
			continue
		}

		rules := module.GetRules(context, *field, scenario)

		for _, rule := range rules {
			err := rule.Validate(value, string(lang))
			if err != nil {
				errs[colName] = err.Error()
			}
		}

		if field.Convert != nil && value != nil {
			_, err := field.Convert(value)
			if err != nil {
				errs[colName] = err.Error()
			}
		}
	}

	return errs
}

func (generator *Generator) mapRequestInput(
	data map[string]interface{},
	module *BaseModule,
	actionColumns []pg.Column,
) map[string]interface{} {
	output := make(map[string]interface{})

	for _, field := range module.Fields {
		colName := field.ColumnName()
		value, ok := data[colName]
		if ok && containsColumn(actionColumns, field.Column) {
			if field.Convert != nil {
				convertedValue, err := field.Convert(value)
				if err != nil {
					continue
				}
				output[colName] = convertedValue
			} else {
				output[colName] = value
			}
		}
	}

	return output
}

func queryParam(c *gin.Context, param string) (interface{}, error) {
	result := c.Request.URL.Query().Get(param)
	if len(result) == 0 {
		return nil, fmt.Errorf("param %s incorrect", param)
	}
	return result, nil
}

func int64QueryParam(c *gin.Context, param string, defaultValue int64) int64 {
	resultInterface, err := queryParam(c, param)
	if err != nil {
		return defaultValue
	}

	resultString, ok := resultInterface.(string)
	if !ok {
		return defaultValue
	}

	result, err := strconv.ParseInt(resultString, 0, 10)
	if err != nil {
		_ = err
		return defaultValue
	}

	return result
}

func containsColumn(columns []pg.Column, target pg.Column) bool {
	return fields.ContainsColumn(columns, target)
}
