package module

import (
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type BaseModule struct {
	Name           string                     `json:"name"`
	Label          string                     `json:"label"`
	Labels         map[string]string          `json:"-"`
	Table          pg.Table                   `json:"-"`
	PrimaryKey     pg.Column                  `json:"-"`
	Path           string                     `json:"path"`
	Fields         []fields.ModuleField       `json:"fields"`
	Defrec         actions.DefrecModuleAction `json:"defrec"`
	Actions        []actions.ModuleAction     `json:"actions"`
	RoleWhere      []actions.RoleWhere        `json:"-"`
	RoleJoin       []actions.RoleJoin         `json:"-"`
	RoleBeforeHook []actions.RoleHook         `json:"-"`
	RoleAfterHook  []actions.RoleAfterHook    `json:"-"`
}

func (module BaseModule) GetField(columnName string) *fields.ModuleField {
	for _, field := range module.Fields {
		if field.ColumnName() == columnName {
			return &field
		}
	}
	return nil
}

func (module BaseModule) GetFieldByColumn(col pg.Column) *fields.ModuleField {
	return module.GetField(col.Name())
}

func (module BaseModule) GetRules(context *gin.Context, field fields.ModuleField, scenario fields.Scenario) []fields.CheckRules {
	checkRules := make([]fields.CheckRules, 0, 10)
	if field.Check != nil {
		for _, rule := range field.Check {
			for _, checkScenario := range rule.GetScenarios() {
				if checkScenario == scenario {
					checkRules = append(checkRules, rule)
				}
			}
		}
	}
	if field.CheckFunc != nil {
		for _, rule := range field.CheckFunc(context) {
			for _, checkScenario := range rule.GetScenarios() {
				if checkScenario == scenario {
					checkRules = append(checkRules, rule)
				}
			}
		}
	}
	if len(field.RoleCheck) > 0 {
		role := string(actions.GetRoleFromContext(context))
		for _, rc := range field.RoleCheck {
			if rc.Role == role || rc.Role == string(actions.RoleAll) {
				for _, rule := range rc.Rules {
					for _, checkScenario := range rule.GetScenarios() {
						if checkScenario == scenario {
							checkRules = append(checkRules, rule)
						}
					}
				}
				break
			}
		}
	}
	return checkRules
}
