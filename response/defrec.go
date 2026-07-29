package response

import (
	f "github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
)

type DefrecResponse struct {
	Renderer *renderer.Identity     `json:"renderer,omitempty"`
	FormPage *renderer.FormPage     `json:"form_page,omitempty"`
	Extra    interface{}            `json:"extra,omitempty"`
	Fields   map[string]interface{} `json:"fields"`
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
		if field.Presentation != nil {
			item["presentation"] = field.Presentation
		}
		if field.Media != nil {
			item["media"] = field.Media
		}
		if field.Section != "" {
			item["section"] = field.Section
		}
		if len(field.Roles) > 0 {
			item["roles"] = field.Roles
		}
		for _, rule := range field.Check {
			if intr, ok := rule.(f.CheckRuleIntrospectable); ok {
				info := intr.RuleInfo()
				if info.Type == "required" {
					for _, s := range info.Scenarios {
						if s == f.ScenarioAdd {
							item["required"] = true
							break
						}
					}
				}
			}
			if _, set := item["required"]; set {
				break
			}
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

func (r *DefrecResponse) AttachRender(render renderer.Universal) {
	r.Renderer = render.FormIdentity()
	r.FormPage = render.Form
}
