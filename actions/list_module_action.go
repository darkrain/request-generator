package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type ListModuleAction struct {
	ModuleAction
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Label        string                                       `json:"label"`
	Columns      []pg.Column                                  `json:"-"`
	ColumnsFunc  func(c *gin.Context) []pg.Column             `json:"-"`
	Size         int64                                        `json:"size,omitempty"`
	Maxsize      int64                                        `json:"maxsize"`
	Permission   []Role                                       `json:"permission"`
	Auth         bool                                         `json:"auth"`
	Join         []ModuleActionJoin                           `json:"join"`
	Where        func(c *gin.Context) pg.BoolExpression       `json:"-"`
	Extra        interface{}                                  `json:"extra"`
	Search       []pg.Column                                  `json:"-"`
	Filter       []pg.Column                                  `json:"-"`
	Fields       []RoleContext                                `json:"-"`
}

func (action ListModuleAction) Action() ModuleActionName {
	return ModuleActionNameList
}

func (action ListModuleAction) BeforeRequest(c *gin.Context) error {
	if action.BeforeAction == nil {
		return nil
	}

	return action.BeforeAction(c)
}
func (action ListModuleAction) AfterRequest(c *gin.Context) {
	if action.AfterAction == nil {
		return
	}

	action.AfterAction(c)
}

func (action ListModuleAction) GetColumns(c *gin.Context) []pg.Column {
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
