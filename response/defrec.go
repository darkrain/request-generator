package response

import (
	f "github.com/darkrain/request-generator/fields"
)

type DefrecResponse struct {
	Extra  interface{}                `json:"extra,omitempty"`
	Fields map[string]interface{} `json:"fields"`
}

func NewDefrecResponse(extra interface{}, fields []f.ModuleField) DefrecResponse {
	fieldsMap := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		item := map[string]interface{}{
			"title":     field.Title,
			"type":      string(field.Type),
			"form_type": string(field.FormType),
		}
		if field.Example != "" {
			item["example"] = field.Example
		}
		if len(field.Options) > 0 {
			item["options"] = field.Options
		}
		if field.Extra != nil && field.Extra.Defrec != nil {
			item["extra"] = field.Extra.Defrec
		}
		key := field.ColumnName()
		if field.Translatable {
			key = field.Name()
			item["translatable"] = true
		}
		fieldsMap[key] = item
	}

	return DefrecResponse{
		Extra:  extra,
		Fields: fieldsMap,
	}
}
