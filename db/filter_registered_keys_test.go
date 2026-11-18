package db

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// A list that joins child rows must not let a caller test a column of those
// rows by naming the join alias in a filter key. Only registered keys filter,
// and their predicate is qualified with the declared column's table.
func TestListIgnoresUnregisteredDottedFilterOnJoinAlias(t *testing.T) {
	var statements []string
	matcher := sqlmock.QueryMatcherFunc(func(_, actual string) error {
		statements = append(statements, actual)
		return nil
	})
	backend, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	defer backend.Close()

	id, status := pg.IntegerColumn("id"), pg.StringColumn("status")
	table := pg.NewTable("", "orders", "", id, status)
	registry := map[string]fields.ModuleFilterField{
		"status": {Column: status, Type: fields.ModuleFieldTypeString},
	}
	mock.ExpectQuery("SELECT").WithArgs("open", int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT").WithArgs("open").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	filter := map[string]string{
		"status":              "open",
		"child_alias.status":  "declined",
		"child_alias.id":      "7",
		"orders.status":       "closed",
		"other_alias.missing": "x",
	}
	_, _, err = NewDB(backend).List(log.NewEntry(log.New()), table, id, nil, registry, 0, 10, nil, "", filter, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.NotEmpty(t, statements)
	for _, statement := range statements {
		require.NotContains(t, statement, "child_alias")
		require.NotContains(t, statement, "other_alias")
		require.Equal(t, 1, strings.Count(statement, `orders."status" =`), statement)
	}
}

// A registered key whose column lives on a joined table is qualified with
// that table, whatever prefix the caller wrote.
func TestListQualifiesRegisteredFilterWithDeclaredColumnTable(t *testing.T) {
	var statements []string
	matcher := sqlmock.QueryMatcherFunc(func(_, actual string) error {
		statements = append(statements, actual)
		return nil
	})
	backend, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	defer backend.Close()

	id := pg.IntegerColumn("id")
	table := pg.NewTable("", "orders", "", id)
	ownStatus := pg.StringColumn("status")
	pg.NewTable("", "own_invitation", "", ownStatus)
	registry := map[string]fields.ModuleFilterField{
		"invitation_status": {Column: ownStatus, Type: fields.ModuleFieldTypeString},
	}
	mock.ExpectQuery("SELECT").WithArgs("accepted", int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT").WithArgs("accepted").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	filter := map[string]string{"invitation_status": "accepted", "child_alias.invitation_status": "declined"}
	_, _, err = NewDB(backend).List(log.NewEntry(log.New()), table, id, nil, registry, 0, 10, nil, "", filter, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	for _, statement := range statements {
		require.Contains(t, statement, `own_invitation."status" =`)
		require.NotContains(t, statement, "child_alias")
	}
}
