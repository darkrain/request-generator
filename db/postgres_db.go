package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type DB struct {
	DBExecutor
	sql   *sql.DB
	Debug bool
}
type Tx struct {
	sql *sql.Tx
}

func NewDB(sql *sql.DB) *DB {
	return &DB{
		sql: sql,
	}
}

func (db *DB) debugLog(log *log.Entry, args ...interface{}) {
	if db.Debug {
		log.Infoln(args...)
	}
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (db *DB) RowExists(query string, args ...interface{}) bool {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	_ = db.sql.QueryRow(query, args...).Scan(&exists)
	return exists
}

// applyJoin applies a single join to the FROM clause
func applyJoin(from pg.ReadableTable, join actions.ModuleActionJoin) pg.ReadableTable {
	switch join.Type {
	case actions.JoinTypeRight:
		return from.RIGHT_JOIN(join.Table, join.OnCondition)
	case actions.JoinTypeInner:
		return from.INNER_JOIN(join.Table, join.OnCondition)
	default:
		return from.LEFT_JOIN(join.Table, join.OnCondition)
	}
}

// buildJoinProjection builds a json_agg(json_build_array(...)) projection for a join's columns
func buildJoinProjection(join actions.ModuleActionJoin) pg.Projection {
	colRefs := make([]string, 0, len(join.Columns))
	for _, col := range join.Columns {
		colRefs = append(colRefs, fmt.Sprintf(`%s."%s"`, join.ResultArrayName, col.Name()))
	}
	rawExpr := fmt.Sprintf("json_agg(json_build_array(%s))", strings.Join(colRefs, ", "))
	return pg.Raw(rawExpr)
}

func (db *DB) List(
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
) (result []interface{}, rowsCount int64, err error) {

	// Build projections: primary key + field columns + join aggregations
	projections := []pg.Projection{primaryKey}
	for _, field := range moduleFields {
		projections = append(projections, field.GetProjection())
	}
	for _, join := range joins {
		if len(join.Columns) > 0 {
			projections = append(projections, buildJoinProjection(join))
		}
	}

	// Build FROM clause with joins
	var from pg.ReadableTable = table
	for _, join := range joins {
		from = applyJoin(from, join)
	}

	// Build WHERE conditions
	tableRef := table.TableName()
	var conditions []pg.BoolExpression
	if where != nil {
		conditions = append(conditions, where)
	}

	// Search
	if len(searchText) > 0 && len(searchColumns) > 0 {
		searchConds := make([]pg.BoolExpression, 0, len(searchColumns))
		for _, col := range searchColumns {
			searchConds = append(searchConds,
				pg.RawBool(
					fmt.Sprintf(`LOWER(%s."%s"::text) LIKE '%%' || #search || '%%'`, tableRef, col.Name()),
					pg.RawArgs{"#search": strings.ToLower(searchText)},
				),
			)
		}
		conditions = append(conditions, pg.OR(searchConds...))
	}

	// Filters
	if len(filter) > 0 {
		for key, value := range filter {
			parts := strings.Split(key, ".")
			if len(parts) > 1 {
				conditions = append(conditions,
					pg.RawBool(
						fmt.Sprintf(`%s."%s" = #val`, parts[0], parts[1]),
						pg.RawArgs{"#val": value},
					),
				)
			} else {
				conditions = append(conditions,
					pg.RawBool(
						fmt.Sprintf(`%s."%s" = #val`, tableRef, key),
						pg.RawArgs{"#val": value},
					),
				)
			}
		}
	}

	// Build SELECT statement
	stmt := pg.SELECT(projections[0], projections[1:]...).FROM(from)
	if len(conditions) > 0 {
		stmt = stmt.WHERE(pg.AND(conditions...))
	}
	stmt = stmt.GROUP_BY(primaryKey).LIMIT(size).OFFSET(size * page)

	// Build COUNT statement
	countStmt := pg.SELECT(pg.COUNT(pg.STAR)).FROM(from)
	if len(conditions) > 0 {
		countStmt = countStmt.WHERE(pg.AND(conditions...))
	}
	if len(joins) > 0 {
		countStmt = countStmt.GROUP_BY(primaryKey)
	}

	query, args := stmt.Sql()
	countQuery, countArgs := countStmt.Sql()

	db.debugLog(log, "[DEBUG] LIST QUERY: ", query)
	db.debugLog(log, "[DEBUG] LIST COUNT QUERY: ", countQuery)

	// Execute main query
	var rows *sql.Rows
	if len(args) > 0 {
		rows, err = db.sql.Query(query, args...)
	} else {
		rows, err = db.sql.Query(query)
	}
	if err != nil {
		log.Errorln("LIST ERR: ", err)
		return nil, 0, err
	}
	defer rows.Close()

	results := make([]interface{}, 0, 10)
	for rows.Next() {
		columnValues := make([]interface{}, 0, 10)
		var primaryValue interface{}
		columnValues = append(columnValues, &primaryValue)

		for i := 0; i < len(moduleFields); i++ {
			columnValues = append(columnValues, moduleFields[i].NewScanValue())
		}
		for _, join := range joins {
			if len(join.Columns) == 0 {
				continue
			}
			var columnValue json.RawMessage
			columnValues = append(columnValues, &columnValue)
		}

		err = rows.Scan(columnValues...)
		if err != nil {
			log.Errorln("[DEBUG] SCAN ERR: ", err)
			continue
		}

		currentResult := make(map[string]interface{})
		offset := 1
		for index, field := range moduleFields {
			value, ok := columnValues[index+offset].(driver.Valuer)
			if ok {
				if field.ResultValueConverter != nil {
					currentResult[field.ColumnName()] = field.ResultValueConverter(value)
				} else {
					currentResult[field.ColumnName()], _ = value.Value()
				}
			} else {
				if field.ResultValueConverter != nil {
					currentResult[field.ColumnName()] = field.ResultValueConverter(value)
				} else {
					currentResult[field.ColumnName()] = value
				}
			}
		}

		if len(moduleFields) > 0 {
			offset = offset + len(moduleFields)
		}

		for index, join := range joins {
			joinValue := columnValues[index+offset]
			converted, ok := joinValue.(*json.RawMessage)
			if !ok {
				continue
			}

			var joinValues [][]interface{}
			err := json.Unmarshal(*converted, &joinValues)
			if err != nil {
				log.Errorln("LIST JOIN ERR: ", err)
				continue
			}

			checkString := ""
			for _, val := range joinValues {
				if val == nil {
					continue
				}
				for _, v := range val {
					if v == nil {
						continue
					}
					checkString = fmt.Sprintf("%v%v", checkString, v)
				}
			}

			joinResults := make([]map[string]interface{}, 0, 10)
			if len(checkString) > 0 {
				for _, joinValue := range joinValues {
					resultMap := make(map[string]interface{})
					for idx, col := range join.Columns {
						resultMap[col.Name()] = joinValue[idx]
					}
					joinResults = append(joinResults, resultMap)
				}
			}

			joinStringsArray := make([]string, 0, 10)
			for _, res := range joinResults {
				jsonRes, err := json.Marshal(res)
				if err != nil {
					continue
				}
				joinStringsArray = append(joinStringsArray, string(jsonRes))
			}
			resultUnique := removeDuplicate(joinStringsArray)

			joinResultUnique := make([]map[string]interface{}, 0, 10)
			for _, res := range resultUnique {
				var mapResult map[string]interface{}
				err := json.Unmarshal([]byte(res), &mapResult)
				if err != nil {
					continue
				}
				joinResultUnique = append(joinResultUnique, mapResult)
			}

			currentResult[join.ResultArrayName] = joinResultUnique
		}

		results = append(results, currentResult)
	}

	result = append(result, results...)

	// Execute count query
	var countResult *sql.Rows
	if len(countArgs) > 0 {
		countResult, err = db.sql.Query(countQuery, countArgs...)
	} else {
		countResult, err = db.sql.Query(countQuery)
	}
	if err != nil {
		log.Errorln("COUNT ERR: ", err)
		return result, 0, nil
	}
	defer countResult.Close()

	var count int64
	if len(joins) > 0 {
		for countResult.Next() {
			count++
		}
	} else {
		for countResult.Next() {
			var currentCount int64
			err = countResult.Scan(&currentCount)
			if err == nil {
				count += currentCount
			}
		}
	}

	return result, count, nil
}

func (db *DB) View(
	log *log.Entry,
	table pg.Table,
	primaryKey pg.Column,
	moduleFields []fields.ModuleField,
	where pg.BoolExpression,
	joins []actions.ModuleActionJoin,
) (interface{}, error) {

	projections := []pg.Projection{primaryKey}
	for _, field := range moduleFields {
		projections = append(projections, field.GetProjection())
	}
	for _, join := range joins {
		if len(join.Columns) > 0 {
			projections = append(projections, buildJoinProjection(join))
		}
	}

	var from pg.ReadableTable = table
	for _, join := range joins {
		from = applyJoin(from, join)
	}

	stmt := pg.SELECT(projections[0], projections[1:]...).FROM(from)
	if where != nil {
		stmt = stmt.WHERE(where)
	}
	stmt = stmt.GROUP_BY(primaryKey).LIMIT(1)

	query, args := stmt.Sql()
	db.debugLog(log, "[DEBUG] VIEW QUERY: ", query)

	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = db.sql.Query(query, args...)
	} else {
		rows, err = db.sql.Query(query)
	}
	if err != nil {
		log.Errorln("VIEW ERR: ", err)
		return nil, err
	}
	defer rows.Close()

	results := make([]interface{}, 0, 10)
	for rows.Next() {
		columnValues := make([]interface{}, 0, 10)
		var primaryValue interface{}
		columnValues = append(columnValues, &primaryValue)

		for i := 0; i < len(moduleFields); i++ {
			columnValues = append(columnValues, moduleFields[i].NewScanValue())
		}
		for _, join := range joins {
			if len(join.Columns) == 0 {
				continue
			}
			var columnValue json.RawMessage
			columnValues = append(columnValues, &columnValue)
		}

		err = rows.Scan(columnValues...)
		if err != nil {
			log.Errorln("[DEBUG] VIEW SCAN ERR: ", err)
			continue
		}

		currentResult := make(map[string]interface{})
		offset := 1
		for index, field := range moduleFields {
			value, ok := columnValues[index+offset].(driver.Valuer)
			if ok {
				currentResult[field.ColumnName()], _ = value.Value()
			} else {
				currentResult[field.ColumnName()] = value
			}
		}

		if len(moduleFields) > 0 {
			offset = offset + len(moduleFields)
		}

		for index, join := range joins {
			joinValue := columnValues[index+offset]
			converted, ok := joinValue.(*json.RawMessage)
			if !ok {
				continue
			}

			var joinValues [][]interface{}
			err := json.Unmarshal(*converted, &joinValues)
			if err != nil {
				log.Errorln("VIEW JOIN ERR: ", err)
				continue
			}

			checkString := ""
			for _, val := range joinValues {
				if val == nil {
					continue
				}
				for _, v := range val {
					if v == nil {
						continue
					}
					checkString = fmt.Sprintf("%v%v", checkString, v)
				}
			}

			joinResults := make([]map[string]interface{}, 0, 10)
			if len(checkString) > 0 {
				for _, joinValue := range joinValues {
					resultMap := make(map[string]interface{})
					for idx, col := range join.Columns {
						resultMap[col.Name()] = joinValue[idx]
					}
					joinResults = append(joinResults, resultMap)
				}
			}

			currentResult[join.ResultArrayName] = joinResults
			offset += 1
		}

		results = append(results, currentResult)
	}

	if len(results) > 0 {
		return results[0], nil
	}

	return nil, errors.New("Record not found")
}

func (db *DB) Add(log *log.Entry, table pg.Table, primaryKey pg.Column, moduleFields []fields.ModuleField, input map[string]interface{}) (interface{}, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	output := struct {
		Value      int64  `json:"value"`
		PrimaryKey string `json:"primary_key"`
	}{}

	keys := make([]string, 0, len(input))
	values := make([]interface{}, 0, len(input))

	for _, field := range moduleFields {
		colName := field.ColumnName()
		value, ok := input[colName]
		if !ok {
			continue
		}
		keys = append(keys, fmt.Sprintf(`"%s"`, colName))
		values = append(values, value)
	}

	valueNumbers := make([]string, 0, len(values))
	for i := range values {
		valueNumbers = append(valueNumbers, fmt.Sprintf(`$%d`, i+1))
	}

	tableName := table.TableName()
	schemaName := table.SchemaName()
	fullTableName := fmt.Sprintf(`"%s"`, tableName)
	if schemaName != "" {
		fullTableName = fmt.Sprintf(`%s."%s"`, schemaName, tableName)
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING "%s"`,
		fullTableName,
		strings.Join(keys, ","),
		strings.Join(valueNumbers, ","),
		primaryKey.Name(),
	)

	db.debugLog(log, "[DEBUG] ADD QUERY: ", query)

	err = tx.QueryRow(query, values...).Scan(&output.Value)
	if err != nil {
		log.Errorln("ADD ERR: ", err)
		return nil, err
	}

	output.PrimaryKey = primaryKey.Name()

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return output, nil
}

func (db *DB) Update(log *log.Entry, table pg.Table, primaryKey pg.Column, moduleFields []fields.ModuleField, input map[string]interface{}, where pg.BoolExpression) (interface{}, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	setClauses := make([]string, 0, len(input))
	values := make([]interface{}, 0, len(input))
	paramIdx := 1

	for _, field := range moduleFields {
		colName := field.ColumnName()
		value, ok := input[colName]
		if !ok {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf(`"%s" = $%d`, colName, paramIdx))
		values = append(values, value)
		paramIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf(`"update_date" = $%d`, paramIdx))
	values = append(values, time.Now())
	paramIdx++

	if where == nil {
		return nil, errors.New("WHERE condition is required for UPDATE")
	}

	// Get WHERE clause SQL from a dummy select
	whereStmt := pg.SELECT(pg.Raw("1")).FROM(table).WHERE(where)
	whereSql, whereArgs := whereStmt.Sql()

	// Extract WHERE part from the full query
	whereIdx := strings.Index(whereSql, "WHERE")
	if whereIdx == -1 {
		return nil, errors.New("could not build WHERE clause")
	}
	whereClause := whereSql[whereIdx:]

	// Re-number placeholders in WHERE clause
	for i, arg := range whereArgs {
		oldPlaceholder := fmt.Sprintf("$%d", i+1)
		newPlaceholder := fmt.Sprintf("$%d", paramIdx)
		whereClause = strings.Replace(whereClause, oldPlaceholder, newPlaceholder, 1)
		values = append(values, arg)
		paramIdx++
	}

	tableName := table.TableName()
	schemaName := table.SchemaName()
	fullTableName := fmt.Sprintf(`"%s"`, tableName)
	if schemaName != "" {
		fullTableName = fmt.Sprintf(`%s."%s"`, schemaName, tableName)
	}

	query := fmt.Sprintf(`UPDATE %s SET %s %s`,
		fullTableName,
		strings.Join(setClauses, ", "),
		whereClause,
	)

	db.debugLog(log, "[DEBUG] UPDATE QUERY: ", query)
	db.debugLog(log, "[DEBUG] UPDATE VALUES: ", values)

	result, err := tx.Exec(query, values...)
	if err != nil {
		return nil, err
	}

	updatedCount, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if updatedCount == 0 {
		return nil, errors.New("record not found")
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return db.View(log, table, primaryKey, moduleFields, where, nil)
}

func (db *DB) Delete(log *log.Entry, table pg.Table, where pg.BoolExpression) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt := table.DELETE().WHERE(where)
	query, args := stmt.Sql()

	db.debugLog(log, "[DEBUG] DELETE QUERY: ", query)

	result, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}

	countOfDeleted, err := result.RowsAffected()
	if err != nil {
		return err
	}

	db.debugLog(log, "[DEBUG] DELETE COUNT OF DELETED: ", countOfDeleted)
	if countOfDeleted == 0 {
		return errors.New("record not found")
	}

	return tx.Commit()
}

func (db *DB) RawRequest(log *log.Entry, query string, params ...interface{}) (*sql.Rows, error) {
	return db.sql.Query(query, params...)
}

func removeDuplicate(sliceList []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
