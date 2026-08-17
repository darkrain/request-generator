package module

import (
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
)

// localizeResultValue applies request-scoped field localizers to one result
// record. Database conversion remains in the DB package; this layer only
// resolves presentation text after the request locale is known.
func (generator *Generator) localizeResultValue(lang locale.Lang, moduleFields []fields.ModuleField, value interface{}) interface{} {
	record, ok := value.(map[string]interface{})
	if !ok {
		return value
	}

	localized := make(map[string]interface{}, len(record))
	for key, item := range record {
		localized[key] = item
	}
	translate := func(key string, fallback string) string {
		return generator.TranslateWithFallback(lang, key, fallback)
	}

	for _, field := range moduleFields {
		if field.ResultValueLocalizer == nil {
			continue
		}
		key := field.Name()
		item, ok := localized[key]
		if !ok {
			continue
		}
		localized[key] = field.ResultValueLocalizer(item, translate)
	}
	return localized
}

func (generator *Generator) localizeResultList(lang locale.Lang, moduleFields []fields.ModuleField, values []interface{}) []interface{} {
	if len(values) == 0 {
		return values
	}

	localized := make([]interface{}, len(values))
	for index, value := range values {
		localized[index] = generator.localizeResultValue(lang, moduleFields, value)
	}
	return localized
}
