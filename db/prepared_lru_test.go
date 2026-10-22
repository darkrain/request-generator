package db

import (
	"context"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/db/sqlcache"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestPreparedReadsPromoteLaterHotShapes(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	executor := NewDBWithPreparedReads(pool, 2)
	a := mock.ExpectPrepare("SELECT 1").WillBeClosed()
	a.ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	read := func(query string) {
		t.Helper()
		rows, e := executor.QueryContext(context.Background(), query)
		require.NoError(t, e)
		require.NoError(t, rows.Close())
	}
	read("SELECT 1")
	b := mock.ExpectPrepare("SELECT 2").WillBeClosed()
	b.ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	read("SELECT 2")
	a.ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	read("SELECT 1")
	c := mock.ExpectPrepare("SELECT 3").WillBeClosed()
	c.ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
	read("SELECT 3")
	require.Contains(t, executor.preparedViews.statements, "SELECT 1")
	require.NotContains(t, executor.preparedViews.statements, "SELECT 2")
	c.ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
	read("SELECT 3")
	stats := executor.PreparedReadCacheStats()
	require.Equal(t, 2, stats.Capacity)
	require.Equal(t, 2, stats.Entries)
	require.Zero(t, stats.Pending)
	require.EqualValues(t, 1, stats.Evictions)
	require.EqualValues(t, 2, stats.Hits)
	require.NoError(t, executor.ClosePreparedViews())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPreparedReadsDoNotEvictAcquiredStatement(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	executor := NewDBWithPreparedReads(pool, 1)
	cache := executor.preparedViews
	held := mock.ExpectPrepare("SELECT 1").WillBeClosed()
	entry, err := cache.statement(context.Background(), pool, "SELECT 1")
	require.NoError(t, err)
	// A query arriving after acquisition but before execution must use the
	// ordinary path, not close the statement about to be executed.
	mock.ExpectQuery("SELECT 2").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	rows, err := executor.QueryContext(context.Background(), "SELECT 2")
	require.NoError(t, err)
	require.NoError(t, rows.Close())
	held.ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	rows, err = entry.stmt.QueryContext(context.Background())
	require.NoError(t, err)
	cache.release(entry)
	require.NoError(t, rows.Close())
	mock.ExpectPrepare("SELECT 2").WillBeClosed().ExpectQuery().WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	rows, err = executor.QueryContext(context.Background(), "SELECT 2")
	require.NoError(t, err)
	require.NoError(t, rows.Close())
	require.EqualValues(t, 1, executor.PreparedReadCacheStats().Bypasses)
	require.NoError(t, executor.ClosePreparedViews())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPreparedReadsEvictionKeepsReturnedRowsPostgres(t *testing.T) {
	pool := preparedTestPool(t)
	pool.SetMaxOpenConns(2)
	pool.SetMaxIdleConns(2)
	executor := NewDBWithPreparedReads(pool, 1)
	defer executor.ClosePreparedViews()
	rows, err := executor.QueryContext(context.Background(), "SELECT generate_series(1,100)")
	require.NoError(t, err)
	defer rows.Close()
	next, err := executor.QueryContext(context.Background(), "SELECT 200")
	require.NoError(t, err)
	require.NoError(t, next.Close())
	n := 0
	for rows.Next() {
		var value int
		require.NoError(t, rows.Scan(&value))
		n++
		require.Equal(t, n, value)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 100, n)
	require.GreaterOrEqual(t, executor.PreparedReadCacheStats().Evictions, uint64(1))
}

func TestPreparedReadsConcurrentEvictionPostgres(t *testing.T) {
	pool := preparedTestPool(t)
	pool.SetMaxOpenConns(4)
	pool.SetMaxIdleConns(4)
	executor := NewDBWithPreparedReads(pool, 2)
	defer executor.ClosePreparedViews()
	var wg sync.WaitGroup
	failures := make(chan error, 16)
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				id := worker*1000 + i
				query := fmt.Sprintf("SELECT $1::bigint,$2::text,pg_sleep(0.001) /* shape %d */", (worker+i)%8)
				rows, err := executor.QueryContext(context.Background(), query, id, fmt.Sprint(id))
				if err != nil {
					failures <- err
					return
				}
				var got int
				var text string
				var unused interface{}
				if !rows.Next() {
					err = fmt.Errorf("missing row: %v", rows.Err())
				} else {
					err = rows.Scan(&got, &text, &unused)
				}
				closeErr := rows.Close()
				if err != nil {
					failures <- err
					return
				}
				if closeErr != nil {
					failures <- closeErr
					return
				}
				if got != id || text != fmt.Sprint(id) {
					failures <- fmt.Errorf("mixed arguments: %d/%s expected %d", got, text, id)
					return
				}
				stats := executor.PreparedReadCacheStats()
				if stats.Entries+stats.Pending > 2 {
					failures <- fmt.Errorf("cache exceeded bound: %+v", stats)
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	require.Greater(t, executor.PreparedReadCacheStats().Evictions, uint64(0))
	require.LessOrEqual(t, executor.PreparedReadCacheStats().Entries, 2)
}

func TestPreparedReadsWrappedPoolEvictionPostgres(t *testing.T) {
	backend := preparedTestPool(t)
	backend.SetMaxOpenConns(2)
	backend.SetMaxIdleConns(2)
	cache := sqlcache.New(sqlcache.Options{})
	pool := cache.WrapDB(backend)
	defer pool.Close()
	executor := NewDBWithPreparedReads(pool, 2)
	defer executor.ClosePreparedViews()
	rows, err := executor.QueryContext(context.Background(), "SELECT generate_series(1,100)")
	require.NoError(t, err)
	defer rows.Close()
	// Cross both the executor cap and the wrapped pool's idle statement cap
	// while a previously returned result is still being consumed.
	for i := 0; i < 140; i++ {
		query := fmt.Sprintf("SELECT $1::bigint /* wrapped %d */", i)
		next, e := executor.QueryContext(context.Background(), query, i)
		require.NoError(t, e)
		require.True(t, next.Next())
		var got int
		require.NoError(t, next.Scan(&got))
		require.Equal(t, i, got)
		require.NoError(t, next.Close())
	}
	n := 0
	for rows.Next() {
		var value int
		require.NoError(t, rows.Scan(&value))
		n++
		require.Equal(t, n, value)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 100, n)
	require.LessOrEqual(t, executor.PreparedReadCacheStats().Entries, 2)
}
