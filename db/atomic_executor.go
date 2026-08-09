package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/darkrain/request-generator/actions"
	"github.com/lib/pq"
)

type atomicExecutor struct {
	tx *sql.Tx
}

// NewAtomicExecutor adapts a generator-owned SQL transaction to the public
// atomic action contract without exposing the transaction to modules.
func NewAtomicExecutor(tx *sql.Tx) actions.AtomicExecutor {
	return atomicExecutor{tx: tx}
}

func (executor atomicExecutor) Insert(ctx context.Context, insert actions.AtomicInsert) (actions.AtomicRecord, error) {
	if insert.Table == nil || insert.PrimaryKey == nil {
		return actions.AtomicRecord{}, fmt.Errorf("atomic insert requires table and primary key")
	}
	if len(insert.Fields) == 0 {
		return actions.AtomicRecord{}, fmt.Errorf("atomic insert requires fields")
	}

	keys := make([]string, 0, len(insert.Fields))
	placeholders := make([]string, 0, len(insert.Fields))
	values := make([]interface{}, 0, len(insert.Fields))
	for index, field := range insert.Fields {
		if field.Column == nil {
			return actions.AtomicRecord{}, fmt.Errorf("atomic insert field %d has no column", index)
		}
		keys = append(keys, fmt.Sprintf(`"%s"`, field.Column.Name()))
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
		values = append(values, atomicDBValue(field.Value))
	}

	tableName := fmt.Sprintf(`"%s"`, insert.Table.TableName())
	if schema := insert.Table.SchemaName(); schema != "" {
		tableName = fmt.Sprintf(`%s."%s"`, schema, insert.Table.TableName())
	}
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING "%s"`, tableName, strings.Join(keys, ","), strings.Join(placeholders, ","), insert.PrimaryKey.Name())

	var value int64
	if err := executor.tx.QueryRowContext(ctx, query, values...).Scan(&value); err != nil {
		return actions.AtomicRecord{}, err
	}
	return actions.AtomicRecord{Value: value, PrimaryKey: insert.PrimaryKey.Name()}, nil
}

func atomicDBValue(value actions.AtomicValue) interface{} {
	if value.Strings != nil {
		return pq.Array(value.Strings)
	}
	if value.Ints != nil {
		return pq.Array(value.Ints)
	}
	if value.JSON != nil {
		return string(value.JSON)
	}
	return value.Interface()
}
