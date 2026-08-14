package actions

import (
	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type DefrecModuleAction struct {
	ModuleAction
	Label        string            `json:"label"`
	Labels       map[string]string `json:"-"`
	Permission   []Role            `json:"permission"`
	Auth         bool              `json:"auth"`
	Fields       []RoleContext     `json:"-"`
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Widget       *WidgetConfig        `json:"widget,omitempty"`
	Realtime     *RealtimeEventConfig `json:"-"`
}

func (action DefrecModuleAction) Action() ModuleActionName {
	return ModuleActionNameDefrec
}

func (action DefrecModuleAction) BeforeRequest(c *gin.Context) error {
	if action.BeforeAction == nil {
		return nil
	}

	return action.BeforeAction(c)
}

func (action DefrecModuleAction) AfterRequest(c *gin.Context) {
	if action.AfterAction == nil {
		return
	}

	action.AfterAction(c)
}

func (action DefrecModuleAction) GetColumns(c *gin.Context) []pg.Column {
	if len(action.Fields) > 0 {
		role := GetRoleFromContext(c)
		if cols := ResolveRoleColumns(action.Fields, role); cols != nil {
			return cols
		}
	}
	return nil
}

func (action DefrecModuleAction) GetFields(c *gin.Context, moduleFields []fields.ModuleField) []fields.ModuleField {
	if len(action.Fields) == 0 {
		return moduleFields
	}
	role := GetRoleFromContext(c)
	cols := ResolveRoleColumns(action.Fields, role)
	if cols == nil {
		return moduleFields
	}

	allowed := make(map[string]struct{}, len(cols))
	for _, col := range cols {
		allowed[col.Name()] = struct{}{}
	}
	result := make([]fields.ModuleField, 0, len(moduleFields))
	for _, field := range moduleFields {
		if _, ok := allowed[field.ColumnName()]; ok {
			result = append(result, field)
		}
	}
	return result
}
