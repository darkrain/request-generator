package response

import (
	f "github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
)

type DefrecResponse struct {
	Extra   interface{}                            `json:"extra,omitempty"`
	Locale  string                                 `json:"locale"`
	Locales []string                               `json:"locales"`
	Fields  map[string]f.ModuleField               `json:"fields"`
	I18n    map[string]map[string]locale.FieldI18n `json:"i18n"`
}

func NewDefrecResponse(extra interface{}, fields []f.ModuleField, lang string, locales []string, i18n map[string]map[string]locale.FieldI18n) DefrecResponse {
	fieldsMap := make(map[string]f.ModuleField)
	for _, field := range fields {
		fieldsMap[field.ColumnName()] = field
	}

	return DefrecResponse{
		Extra:   extra,
		Locale:  lang,
		Locales: locales,
		Fields:  fieldsMap,
		I18n:    i18n,
	}
}
