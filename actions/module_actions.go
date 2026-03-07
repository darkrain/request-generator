package actions

import (
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
