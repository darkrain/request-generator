package actions

import (
	"github.com/gin-gonic/gin"
)

type DefrecModuleAction struct {
	ModuleAction
	Label        string            `json:"label"`
	Labels       map[string]string `json:"-"`
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
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
