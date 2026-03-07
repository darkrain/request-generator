package response

import (
	"fmt"

	f "github.com/portalenergy/pe-request-generator/fields"
	"github.com/portalenergy/pe-request-generator/locale"
)

type DefrecResponse struct {
	Extra   interface{}                                `json:"extra,omitempty"`
	Locale  string                                     `json:"locale"`
	Locales []string                                   `json:"locales"`
	Fields  map[string]f.ModuleField                   `json:"fields"`
	I18n    map[string]map[string]locale.FieldI18n     `json:"i18n"`
}

func NewDefrecResponse(extra interface{}, fields []f.ModuleField, lang locale.Lang, locales []locale.Lang) DefrecResponse {
	fieldsMap := make(map[string]f.ModuleField)
	for _, field := range fields {
		fieldsMap[field.ColumnName()] = field
	}

	i18n := buildFieldsI18n(fields, locales)

	localeStrings := make([]string, len(locales))
	for i, l := range locales {
		localeStrings[i] = string(l)
	}

	return DefrecResponse{
		Extra:   extra,
		Locale:  string(lang),
		Locales: localeStrings,
		Fields:  fieldsMap,
		I18n:    i18n,
	}
}

func buildFieldsI18n(fields []f.ModuleField, locales []locale.Lang) map[string]map[string]locale.FieldI18n {
	result := make(map[string]map[string]locale.FieldI18n, len(locales))
	for _, lang := range locales {
		langStr := string(lang)
		langMap := make(map[string]locale.FieldI18n, len(fields))
		for _, field := range fields {
			fi := locale.FieldI18n{
				Title: locale.Resolve(field.Titles, lang, field.Title),
			}
			if len(field.Options) > 0 {
				fi.Options = make(map[string]string, len(field.Options))
				for _, opt := range field.Options {
					key := fmt.Sprintf("%v", opt.Value)
					fi.Options[key] = locale.Resolve(opt.Labels, lang, opt.Label)
				}
			}
			langMap[field.ColumnName()] = fi
		}
		result[langStr] = langMap
	}
	return result
}
