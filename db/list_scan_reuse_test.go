package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestListScanDestinationsKeepRowsIndependent(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	id := pg.IntegerColumn("id")
	table := pg.NewTable("", "records", "", id)
	fieldList := []fields.ModuleField{
		{Column: pg.StringColumn("title")},
		{Column: pg.IntegerColumn("number")},
		{Column: pg.BoolColumn("enabled")},
		{Column: pg.FloatColumn("fraction")},
		{Column: pg.TimestampColumn("moment")},
		{Column: pg.StringColumn("object"), Type: fields.ModuleFieldTypeObject},
		{Column: pg.StringColumn("array"), Type: fields.ModuleFieldTypeArray},
		// Retaining the pointer is a legal producer callback. Later rows
		// must never overwrite the earlier callback's result.
		{Column: pg.StringColumn("custom"), ResultValueConverter: func(v interface{}) interface{} { return v }},
	}
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "title", "number", "enabled", "fraction", "moment", "object", "array", "custom"}).
		AddRow(1, "first", 12, true, 1.25, stamp, `{"name":"first"}`, `[1,2]`, "custom-first").
		AddRow(2, nil, nil, nil, nil, nil, nil, nil, nil).
		AddRow(3, "last", 34, false, 2.5, stamp.Add(time.Hour), `{"name":"last"}`, `[3]`, "custom-last")
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	result, count, err := NewDB(pool).List(log.NewEntry(log.New()), table, id, fieldList, nil, 0, 10, nil, "", nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
	require.Equal(t, map[string]interface{}{
		"title": "first", "number": int64(12), "enabled": true, "fraction": 1.25, "moment": stamp,
		"object": map[string]interface{}{"name": "first"}, "array": []interface{}{float64(1), float64(2)},
		"custom": &sql.NullString{String: "custom-first", Valid: true},
	}, result[0])
	require.Equal(t, map[string]interface{}{
		"title": nil, "number": nil, "enabled": nil, "fraction": nil, "moment": nil, "object": nil, "array": nil, "custom": &sql.NullString{},
	}, result[1])
	require.Equal(t, map[string]interface{}{
		"title": "last", "number": int64(34), "enabled": false, "fraction": 2.5, "moment": stamp.Add(time.Hour),
		"object": map[string]interface{}{"name": "last"}, "array": []interface{}{float64(3)},
		"custom": &sql.NullString{String: "custom-last", Valid: true},
	}, result[2])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListScanReuseKeepsTranslationsAndJoinsIndependent(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	id, title := pg.IntegerColumn("id"), pg.StringColumn("title")
	table := pg.NewTable("", "records", "", id, title)
	label := pg.StringColumn("label")
	other := pg.NewTable("", "labels", "", label)
	joins := []actions.ModuleActionJoin{{Table: other, OnCondition: pg.Bool(true), Columns: []pg.Column{label}, ResultArrayName: "labels"}}
	rows := sqlmock.NewRows([]string{"id", "title", "labels"}).
		AddRow(1, `{"en":"first"}`, []byte(`[["one"],["one"]]`)).
		AddRow(2, nil, []byte(`[[null]]`)).
		AddRow(3, `{"en":"last"}`, []byte(`[["three"]]`))
	mock.ExpectQuery("SELECT").WillReturnRows(rows).RowsWillBeClosed()
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	result, _, err := NewDB(pool).List(log.NewEntry(log.New()), table, id,
		[]fields.ModuleField{{Column: title, Translatable: true}}, nil, 0, 10, nil, "", nil, nil, joins, nil, &TranslationContext{EntityName: "records"})
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"title": map[string]string{"en": "first"}, "labels": []map[string]interface{}{{"label": "one"}}}, result[0])
	require.Equal(t, map[string]interface{}{"title": map[string]string{}, "labels": []map[string]interface{}{}}, result[1])
	require.Equal(t, map[string]interface{}{"title": map[string]string{"en": "last"}, "labels": []map[string]interface{}{{"label": "three"}}}, result[2])
	require.NoError(t, mock.ExpectationsWereMet())
}
