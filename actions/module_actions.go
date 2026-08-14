package actions

import (
	"fmt"

	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type ModuleActionName string

const (
	ModuleActionNameList   ModuleActionName = "list"
	ModuleActionNameAdd    ModuleActionName = "add"
	ModuleActionNameDefrec ModuleActionName = "defrec"
	ModuleActionNameView   ModuleActionName = "view"
	ModuleActionNameUpdate ModuleActionName = "update"
	ModuleActionNameDelete ModuleActionName = "delete"
)

type ModuleAction interface {
	GetModuleName() string
	Action() ModuleActionName
	BeforeRequest(c *gin.Context) error
	AfterRequest(c *gin.Context)
	GetColumns(c *gin.Context) []pg.Column
}

type WidgetConfig struct {
	ID       string                          `json:"id"`
	Order    int                             `json:"order,omitempty"`
	Renderer renderer.GlobalWidget           `json:"renderer"`
	Bindings []renderer.WidgetRequestBinding `json:"bindings,omitempty"`
}

func (config WidgetConfig) Validate() error {
	if config.ID == "" {
		return fmt.Errorf("widget id is required")
	}
	if err := config.Renderer.Validate(); err != nil {
		return err
	}
	if config.Renderer.Workspace != nil && len(config.Bindings) != 0 {
		return fmt.Errorf("workspace widget bindings must be declared by a workspace resource")
	}
	return renderer.ValidateWidgetRequestBindings(config.Bindings)
}

type JoinType string

const (
	JoinTypeLeft       JoinType = "LEFT"
	JoinTypeLeftOuter  JoinType = "LEFT OUTER"
	JoinTypeRight      JoinType = "RIGHT"
	JoinTypeRightOuter JoinType = "RIGHT OUTER"
	JoinTypeInner      JoinType = "INNER"
)

type ModuleActionJoin struct {
	Table           pg.ReadableTable  `json:"-"`
	Type            JoinType          `json:"type"`
	OnCondition     pg.BoolExpression `json:"-"`
	Columns         []pg.Column       `json:"-"`
	ResultArrayName string            `json:"result_array_name"`
}

func NewJoin(table pg.ReadableTable, joinType JoinType, onCondition pg.BoolExpression, columns []pg.Column, resultArrayName string) ModuleActionJoin {
	return ModuleActionJoin{
		Table:           table,
		Type:            joinType,
		OnCondition:     onCondition,
		Columns:         columns,
		ResultArrayName: resultArrayName,
	}
}
