package response

import (
	f "github.com/portalenergy/pe-request-generator/fields"
)

type DefrecResponse struct {
	Extra  interface{}              `json:"extra,omitempty"`
	Fields map[string]f.ModuleField `json:"fields"`
}

func NewDefrecResponse(Extra interface{}, fields []f.ModuleField) DefrecResponse {
	fieldsMap := make(map[string]f.ModuleField)
	for _, field := range fields {
		fieldsMap[field.ColumnName()] = field
	}

	return DefrecResponse{
		Extra:  Extra,
		Fields: fieldsMap,
	}
}
