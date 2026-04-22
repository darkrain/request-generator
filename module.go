package module

import (
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type MenuEntry struct {
	ActionName  string                 `json:"action"`
	Title       string                 `json:"title"`
	Icon        string                 `json:"icon,omitempty"`
	Show        bool                   `json:"show"`
	Order       int                    `json:"order"`
	Group       string                 `json:"group"`
	CustomLink  string                 `json:"custom_link,omitempty"`
	CustomQuery map[string]interface{} `json:"custom_query,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

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
	EntityName     string                     `json:"-"`
	MenuEntries    []MenuEntry                `json:"menu_entries,omitempty"`
}

func (module BaseModule) GetEntityName() string {
	if module.EntityName != "" {
		return module.EntityName
	}
	return module.Table.TableName()
}

func (module BaseModule) TranslatableFields() []fields.ModuleField {
	var result []fields.ModuleField
	for _, f := range module.Fields {
		if f.Translatable {
			result = append(result, f)
		}
	}
	return result
}

func (module BaseModule) GetField(columnName string) *fields.ModuleField {
	for i, field := range module.Fields {
		if field.ColumnName() == columnName {
			return &module.Fields[i]
		}
		if field.Translatable && field.FieldName == columnName {
			return &module.Fields[i]
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
