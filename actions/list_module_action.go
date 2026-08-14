package actions

import (
	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type ListModuleAction struct {
	ModuleAction
	BeforeAction func(c *gin.Context) error
	AfterAction  func(c *gin.Context)
	Label        string                                 `json:"label"`
	Labels       map[string]string                      `json:"-"`
	Columns      []pg.Column                            `json:"-"`
	ColumnsFunc  func(c *gin.Context) []pg.Column       `json:"-"`
	Size         int64                                  `json:"size,omitempty"`
	Maxsize      int64                                  `json:"maxsize"`
	Permission   []Role                                 `json:"permission"`
	Auth         bool                                   `json:"auth"`
	Join         []ModuleActionJoin                     `json:"join"`
	Where        func(c *gin.Context) pg.BoolExpression `json:"-"`
	Search       []pg.Column                            `json:"-"`
	Filter       []pg.Column                            `json:"-"`
	FilterFunc   func(c *gin.Context) []pg.Column       `json:"-"`
	// VirtualFilters declares filters that are not backed by a module field.
	// Their UI metadata is returned together with regular filters.
	VirtualFilters       []fields.ModuleFilterField       `json:"-"`
	Sort                 []pg.Column                      `json:"-"`
	SortDefault          pg.Column                        `json:"-"`
	SortDefaultDirection SortDirection                    `json:"-"`
	SortDefaultFunc      func(c *gin.Context) *SortOption `json:"-"`
	Fields               []RoleContext                    `json:"-"`
	Widget               *WidgetConfig                    `json:"widget,omitempty"`
	Realtime             *RealtimeEventConfig             `json:"-"`
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
