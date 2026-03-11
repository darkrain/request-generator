package db

import (
	"database/sql"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
)

type DBExecutor interface {
	List(
		log *log.Entry,
		table pg.Table,
		primaryKey pg.Column,
		moduleFields []fields.ModuleField,
		page int64,
		size int64,
		searchColumns []pg.Column,
		searchText string,
		filter map[string]string,
		where pg.BoolExpression,
		joins []actions.ModuleActionJoin,
	) (result []interface{}, rowsCount int64, err error)
	View(
		log *log.Entry,
		table pg.Table,
		primaryKey pg.Column,
		moduleFields []fields.ModuleField,
		where pg.BoolExpression,
		joins []actions.ModuleActionJoin,
	) (interface{}, error)
	Add(log *log.Entry, table pg.Table, primaryKey pg.Column, moduleFields []fields.ModuleField, input map[string]interface{}) (interface{}, error)
	Update(log *log.Entry, table pg.Table, primaryKey pg.Column, moduleFields []fields.ModuleField, input map[string]interface{}, where pg.BoolExpression) (interface{}, error)
	Delete(log *log.Entry, table pg.Table, where pg.BoolExpression) error
	RawRequest(log *log.Entry, query string, params ...interface{}) (*sql.Rows, error)
}
