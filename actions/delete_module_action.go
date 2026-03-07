package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type DeleteModuleAction struct {
	ModuleAction
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Label        string            `json:"label"`
	Labels       map[string]string `json:"-"`
	Permission   []Role      `json:"permission"`
	Auth         bool        `json:"auth"`
	By           []pg.Column `json:"-"`
}

func (action DeleteModuleAction) Action() ModuleActionName {
	return ModuleActionNameDelete
}

func (action DeleteModuleAction) BeforeRequest(c *gin.Context) error {
	if action.BeforeAction == nil {
		return nil
	}

	return action.BeforeAction(c)
}
func (action DeleteModuleAction) AfterRequest(c *gin.Context) {
	if action.AfterAction == nil {
		return
	}

	action.AfterAction(c)
}

func (action DeleteModuleAction) GetColumns(c *gin.Context) []pg.Column {
	return nil
}
