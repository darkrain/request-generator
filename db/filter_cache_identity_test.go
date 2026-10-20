package db

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/db/sqlcache"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestListEquivalentFiltersShareSQLResultCache(t *testing.T) {
	backend, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer backend.Close()
	cache := sqlcache.New(sqlcache.Options{TTL: time.Minute, MaxEntries: 16, MaxBytes: 1 << 20, MaxResultBytes: 1 << 16})
	pool := cache.WrapDB(backend)
	defer pool.Close()
	id, category, region := pg.IntegerColumn("id"), pg.StringColumn("category"), pg.StringColumn("region")
	table := pg.NewTable("", "records", "", id, category, region)
	registry := map[string]fields.ModuleFilterField{
		"category": {Column: category, Type: fields.ModuleFieldTypeString},
		"region":   {Column: region, Type: fields.ModuleFieldTypeString},
	}
	// Only two physical reads are allowed for arbitrarily ordered repetitions:
	// the list and its count. A different value must still cause two new reads.
	expect := func(region string, result int64) {
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs("book", region, int64(10), int64(0)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(result))
		mock.ExpectPrepare("SELECT").ExpectQuery().WithArgs("book", region).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	}
	expect("north", 11)
	executor := NewDB(pool)
	logger := log.NewEntry(log.New())
	for i := 0; i < 100; i++ {
		filter := map[string]string{}
		if i%2 == 0 {
			filter["category"], filter["region"] = "book", "north"
		} else {
			filter["region"], filter["category"] = "north", "book"
		}
		rows, count, err := executor.List(logger, table, id, nil, registry, 0, 10, nil, "", filter, nil, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.EqualValues(t, 1, count)
	}
	require.EqualValues(t, 2, cache.Stats().Misses)
	require.EqualValues(t, 198, cache.Stats().Hits)
	// The SQL statements are retained, but the result key includes the values.
	mock.ExpectQuery("SELECT").WithArgs("book", "south", int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(22))
	mock.ExpectQuery("SELECT").WithArgs("book", "south").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	_, count, err := executor.List(logger, table, id, nil, registry, 0, 10, nil, "", map[string]string{"category": "book", "region": "south"}, nil, nil, nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.EqualValues(t, 4, cache.Stats().Misses)
	require.NoError(t, mock.ExpectationsWereMet())
}
