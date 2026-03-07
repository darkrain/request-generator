package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type AddModuleAction struct {
	ModuleAction
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Label        string                           `json:"label"`
	Columns      []pg.Column                      `json:"-"`
	ColumnsFunc  func(c *gin.Context) []pg.Column `json:"-"`
	Permission   []Role      `json:"permission"`
	Auth         bool        `json:"auth"`
}

func (action AddModuleAction) Action() ModuleActionName {
	return ModuleActionNameAdd
}

func (action AddModuleAction) BeforeRequest(c *gin.Context) error {
	if action.BeforeAction == nil {
		return nil
	}

	return action.BeforeAction(c)
}
func (action AddModuleAction) AfterRequest(c *gin.Context) {
	if action.AfterAction == nil {
		return
	}

	action.AfterAction(c)
}

func (action AddModuleAction) GetColumns(c *gin.Context) []pg.Column {
	if action.ColumnsFunc != nil {
		return action.ColumnsFunc(c)
	}
	return action.Columns
}
