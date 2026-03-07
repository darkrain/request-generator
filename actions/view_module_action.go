package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type ViewModuleAction struct {
	ModuleAction
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Label        string                           `json:"label"`
	Columns      []pg.Column                      `json:"-"`
	ColumnsFunc  func(c *gin.Context) []pg.Column `json:"-"`
	Permission   []Role             `json:"permission"`
	Auth         bool               `json:"auth"`
	Join         []ModuleActionJoin `json:"join"`
	By           []pg.Column        `json:"-"`
	Extra        interface{}        `json:"extra"`
	Fields       []RoleContext      `json:"-"`
}

func (action ViewModuleAction) Action() ModuleActionName {
	return ModuleActionNameView
}

func (action ViewModuleAction) BeforeRequest(c *gin.Context) error {
	if action.BeforeAction == nil {
		return nil
	}

	return action.BeforeAction(c)
}
func (action ViewModuleAction) AfterRequest(c *gin.Context) {
	if action.AfterAction == nil {
		return
	}

	action.AfterAction(c)
}

func (action ViewModuleAction) GetColumns(c *gin.Context) []pg.Column {
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
