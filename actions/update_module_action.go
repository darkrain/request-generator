package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type UpdateModuleAction struct {
	ModuleAction
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Label        string                           `json:"label"`
	Columns      []pg.Column                      `json:"-"`
	ColumnsFunc  func(c *gin.Context) []pg.Column `json:"-"`
	Permission   []Role                           `json:"permission"`
	Auth         bool        `json:"auth"`
	By           []pg.Column `json:"-"`
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
	if action.ColumnsFunc != nil {
		return action.ColumnsFunc(c)
	}
	return action.Columns
}
