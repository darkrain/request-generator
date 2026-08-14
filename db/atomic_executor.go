package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/darkrain/request-generator/actions"
	pg "github.com/go-jet/jet/v2/postgres"
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
		if err := field.Value.Validate(); err != nil {
			return actions.AtomicRecord{}, fmt.Errorf("atomic insert field %q: %w", field.Column.Name(), err)
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

func (executor atomicExecutor) Upsert(ctx context.Context, upsert actions.AtomicUpsert) (actions.AtomicUpsertResult, error) {
	insert := upsert.Insert
	if insert.Table == nil || insert.PrimaryKey == nil {
		return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert requires table and primary key")
	}
	if len(insert.Fields) == 0 {
		return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert requires insert fields")
	}
	if len(upsert.ConflictColumns) == 0 {
		return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert requires conflict columns")
	}

	keys := make([]string, 0, len(insert.Fields))
	placeholders := make([]string, 0, len(insert.Fields))
	values := make([]interface{}, 0, len(insert.Fields)+len(upsert.UpdateFields))
	for index, field := range insert.Fields {
		if field.Column == nil {
			return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert insert field %d has no column", index)
		}
		if err := field.Value.Validate(); err != nil {
			return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert insert field %q: %w", field.Column.Name(), err)
		}
		keys = append(keys, fmt.Sprintf(`"%s"`, field.Column.Name()))
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
		values = append(values, atomicDBValue(field.Value))
	}

	conflicts := make([]string, 0, len(upsert.ConflictColumns))
	for index, column := range upsert.ConflictColumns {
		if column == nil {
			return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert conflict column %d is nil", index)
		}
		conflicts = append(conflicts, fmt.Sprintf(`"%s"`, column.Name()))
	}

	resolution := "DO NOTHING"
	if len(upsert.UpdateFields) > 0 {
		assignments := make([]string, 0, len(upsert.UpdateFields))
		for index, field := range upsert.UpdateFields {
			if field.Column == nil {
				return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert update field %d has no column", index)
			}
			if err := field.Value.Validate(); err != nil {
				return actions.AtomicUpsertResult{}, fmt.Errorf("atomic upsert update field %q: %w", field.Column.Name(), err)
			}
			values = append(values, atomicDBValue(field.Value))
			assignments = append(assignments, fmt.Sprintf(`"%s" = $%d`, field.Column.Name(), len(values)))
		}
		resolution = "DO UPDATE SET " + strings.Join(assignments, ",")
	}

	tableName := fmt.Sprintf(`"%s"`, insert.Table.TableName())
	if schema := insert.Table.SchemaName(); schema != "" {
		tableName = fmt.Sprintf(`%s."%s"`, schema, insert.Table.TableName())
	}
	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) %s RETURNING "%s", (xmax = 0) AS inserted`,
		tableName,
		strings.Join(keys, ","),
		strings.Join(placeholders, ","),
		strings.Join(conflicts, ","),
		resolution,
		insert.PrimaryKey.Name(),
	)

	var value int64
	var inserted bool
	err := executor.tx.QueryRowContext(ctx, query, values...).Scan(&value, &inserted)
	if err == sql.ErrNoRows {
		return actions.AtomicUpsertResult{Inserted: false}, nil
	}
	if err != nil {
		return actions.AtomicUpsertResult{}, err
	}
	return actions.AtomicUpsertResult{
		Record:   actions.AtomicRecord{Value: value, PrimaryKey: insert.PrimaryKey.Name()},
		Inserted: inserted,
	}, nil
}

func (executor atomicExecutor) SelectOne(ctx context.Context, selectRequest actions.AtomicSelect) (actions.AtomicRecord, error) {
	if selectRequest.Table == nil || len(selectRequest.Fields) == 0 || selectRequest.Where == nil {
		return actions.AtomicRecord{}, fmt.Errorf("atomic select requires table, fields, and where")
	}
	projections := make([]pg.Projection, 0, len(selectRequest.Fields))
	scans := make([]interface{}, 0, len(selectRequest.Fields))
	values := make([]func() (actions.AtomicValue, error), 0, len(selectRequest.Fields))
	for _, field := range selectRequest.Fields {
		if field.Name == "" || field.Column == nil {
			return actions.AtomicRecord{}, fmt.Errorf("atomic select field requires name and column")
		}
		scan, value, err := atomicSelectScan(field.Kind)
		if err != nil {
			return actions.AtomicRecord{}, fmt.Errorf("atomic select field %q: %w", field.Name, err)
		}
		projections = append(projections, field.Column)
		scans = append(scans, scan)
		values = append(values, value)
	}
	query := pg.SELECT(projections[0], projections[1:]...).FROM(selectRequest.Table).WHERE(selectRequest.Where).LIMIT(1)
	statement, args := query.Sql()
	if err := executor.tx.QueryRowContext(ctx, statement, args...).Scan(scans...); err != nil {
		return actions.AtomicRecord{}, err
	}
	record := actions.AtomicRecord{Fields: make([]actions.AtomicField, 0, len(selectRequest.Fields))}
	for index, field := range selectRequest.Fields {
		value, err := values[index]()
		if err != nil {
			return actions.AtomicRecord{}, fmt.Errorf("atomic select field %q: %w", field.Name, err)
		}
		record.Fields = append(record.Fields, actions.AtomicField{Name: field.Name, Value: value})
	}
	return record, nil
}

func atomicSelectScan(kind actions.AtomicValueKind) (interface{}, func() (actions.AtomicValue, error), error) {
	switch kind {
	case actions.AtomicValueKindString:
		value := &sql.NullString{}
		return value, func() (actions.AtomicValue, error) {
			if !value.Valid {
				return actions.AtomicValue{}, fmt.Errorf("value is null")
			}
			return actions.AtomicString(value.String), nil
		}, nil
	case actions.AtomicValueKindInt:
		value := &sql.NullInt64{}
		return value, func() (actions.AtomicValue, error) {
			if !value.Valid {
				return actions.AtomicValue{}, fmt.Errorf("value is null")
			}
			return actions.AtomicInt(value.Int64), nil
		}, nil
	case actions.AtomicValueKindFloat:
		value := &sql.NullFloat64{}
		return value, func() (actions.AtomicValue, error) {
			if !value.Valid {
				return actions.AtomicValue{}, fmt.Errorf("value is null")
			}
			return actions.AtomicFloat(value.Float64), nil
		}, nil
	case actions.AtomicValueKindBool:
		value := &sql.NullBool{}
		return value, func() (actions.AtomicValue, error) {
			if !value.Valid {
				return actions.AtomicValue{}, fmt.Errorf("value is null")
			}
			return actions.AtomicBool(value.Bool), nil
		}, nil
	case actions.AtomicValueKindInts:
		value := &pq.Int64Array{}
		return value, func() (actions.AtomicValue, error) { return actions.AtomicValue{Ints: []int64(*value)}, nil }, nil
	case actions.AtomicValueKindStrings:
		value := &pq.StringArray{}
		return value, func() (actions.AtomicValue, error) { return actions.AtomicValue{Strings: []string(*value)}, nil }, nil
	default:
		return nil, nil, fmt.Errorf("unsupported kind %q", kind)
	}
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
