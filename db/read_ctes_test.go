package db

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestReadCTEsScopeParametersAndCoverListCountView(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	base := NewDB(pool)
	id, label := pg.IntegerColumn("id"), pg.StringColumn("label")
	table := pg.NewTable("public", "items", "", id, label)
	field := []fields.ModuleField{{Column: label, Type: fields.ModuleFieldTypeString}}
	where := pg.RawBool("items.id IN (SELECT id FROM visible_items)")
	entry := log.NewEntry(log.New())
	for _, tenant := range []int64{11, 22, 11} {
		ctes := []pg.CommonTableExpression{pg.CTE("visible_items").AS(pg.RawStatement(
			"SELECT id FROM items WHERE tenant_id = #tenant", pg.RawArgs{"#tenant": tenant}))}
		executor, err := WithReadCTEs(WithRequestContext(base, context.Background()), ctes)
		require.NoError(t, err)
		// The same tenant scope must precede both the paginated read and count.
		mock.ExpectQuery(`WITH visible_items AS`).WithArgs(tenant, int64(10), int64(0)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "label"}).AddRow(1, "visible"))
		mock.ExpectQuery(`WITH visible_items AS`).WithArgs(tenant).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows, count, err := executor.List(entry, table, id, field, nil, 0, 10, nil, "", nil, where, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.EqualValues(t, 1, count)
		mock.ExpectQuery(`WITH visible_items AS`).WithArgs(tenant, int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "label"}).AddRow(1, "visible"))
		_, err = executor.View(entry, table, id, field, where, nil, nil)
		require.NoError(t, err)
	}
	// Request-local definitions never leak into the shared executor.
	q, args := base.readSQL(pg.SELECT(id).FROM(table))
	require.False(t, strings.Contains(q, "WITH"))
	require.Empty(t, args)
	require.NoError(t, mock.ExpectationsWereMet())
}

// An existing custom executor remains usable without CTEs. A producer opting
// in must not have its declared SQL scope silently ignored.
type executorWithoutCTEs struct{ DBExecutor }

func TestReadCTEsRejectUnsupportedExecutor(t *testing.T) {
	custom := &executorWithoutCTEs{}
	unchanged, err := WithReadCTEs(custom, nil)
	require.NoError(t, err)
	require.Same(t, custom, unchanged)
	_, err = WithReadCTEs(custom, []pg.CommonTableExpression{
		pg.CTE("visible_items").AS(pg.SELECT(pg.Int(1))),
	})
	require.ErrorContains(t, err, "does not support read CTEs")
}
