package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type UpdateModuleAction struct {
	ModuleAction
	BeforeAction    func(c *gin.Context) error
	AfterAction     func(c *gin.Context)
	Label           string                           `json:"label"`
	Labels          map[string]string                `json:"-"`
	Columns         []pg.Column                      `json:"-"`
	ColumnsFunc     func(c *gin.Context) []pg.Column `json:"-"`
	Permission      []Role                           `json:"permission"`
	Auth            bool                             `json:"auth"`
	By              []pg.Column                      `json:"-"`
	Fields          []RoleContext                    `json:"-"`
	ViewAfterUpdate *bool                            `json:"-"` // default true; if true and ViewAction exists, return view response after update
}

func (action UpdateModuleAction) Action() ModuleActionName {
	return ModuleActionNameUpdate
}

func (action UpdateModuleAction) BeforeRequest(c *gin.Context) error {
	if action.BeforeAction == nil {
		return nil
	}

	return action.BeforeAction(c)
}
func (action UpdateModuleAction) AfterRequest(c *gin.Context) {
	if action.AfterAction == nil {
		return
	}

	action.AfterAction(c)
}

func (action UpdateModuleAction) GetColumns(c *gin.Context) []pg.Column {
	if len(action.Fields) > 0 {
		role := GetRoleFromContext(c)
		if cols := ResolveRoleColumns(action.Fields, role); cols != nil {
			return cols
		}
	}
	if action.ColumnsFunc != nil {
		return action.ColumnsFunc(c)
	}
	return action.Columns
}
