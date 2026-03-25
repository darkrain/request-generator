package module

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/db"
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
		field := module.GetField(colName)
		if field == nil {
			continue
		}

		if field.Translatable {
			// For translatable fields, look up by FieldName and validate each language
			value := data[field.Name()]
			rules := module.GetRules(context, *field, scenario)

			if langMap, ok := value.(map[string]interface{}); ok {
				for langKey, langVal := range langMap {
					for _, rule := range rules {
						err := rule.Validate(langVal, string(lang))
						if err != nil {
							errs[field.Name()+"."+langKey] = err.Error()
						}
					}
				}
			} else {
				for _, rule := range rules {
					err := rule.Validate(value, string(lang))
					if err != nil {
						errs[field.Name()] = err.Error()
					}
				}
			}
			continue
		}

		value := data[colName]
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
		if field.Translatable {
			// For translatable fields, look up by FieldName
			value, ok := data[field.Name()]
			if ok && containsColumn(actionColumns, field.Column) {
				if field.Convert != nil {
					convertedValue, err := field.Convert(value)
					if err != nil {
						continue
					}
					output[field.Name()] = convertedValue
				} else {
					output[field.Name()] = value
				}
			}
			continue
		}

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

func findViewAction(module *BaseModule) *actions.ViewModuleAction {
	for _, a := range module.Actions {
		if a.Action() == actions.ModuleActionNameView {
			if va, ok := a.(actions.ViewModuleAction); ok {
				return &va
			}
		}
	}
	return nil
}

func findUpdateAction(module *BaseModule) *actions.UpdateModuleAction {
	for _, a := range module.Actions {
		if a.Action() == actions.ModuleActionNameUpdate {
			if ua, ok := a.(actions.UpdateModuleAction); ok {
				return &ua
			}
		}
	}
	return nil
}

func (generator *Generator) buildTranslationContext(module *BaseModule) *db.TranslationContext {
	transFields := module.TranslatableFields()
	if len(transFields) == 0 {
		return nil
	}
	langs := make([]string, len(generator.Locales))
	for i, l := range generator.Locales {
		langs[i] = string(l)
	}
	fieldInfos := make([]db.TranslatableFieldInfo, len(transFields))
	for i, f := range transFields {
		fieldInfos[i] = db.TranslatableFieldInfo{FieldName: f.Name()}
	}
	return &db.TranslationContext{
		EntityName: module.GetEntityName(),
		Fields:     fieldInfos,
		Langs:      langs,
	}
}
