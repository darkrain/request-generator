package db

import (
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/actions"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestShortFirstPageCountBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		enabled    bool
		page, size int64
		rows       int
		count      bool
	}{
		{"opt in partial", true, 0, 10, 2, false},
		{"opt in empty", true, 0, 10, 0, false},
		{"default unchanged", false, 0, 10, 2, true},
		{"full first page", true, 0, 2, 2, true},
		{"later partial page", true, 1, 10, 2, true},
		{"later empty page", true, 1, 10, 0, true},
		{"zero limit", true, 0, 0, 0, true},
		{"negative limit", true, 0, -1, 2, true},
		{"negative page", true, -1, 10, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer pool.Close()
			base := NewDB(pool)
			db := base
			if tc.enabled {
				db = base.WithShortFirstPageCount()
			}
			require.False(t, base.shortFirstPageCount)
			id := pg.IntegerColumn("id")
			table := pg.NewTable("", "records", "", id)
			rows := sqlmock.NewRows([]string{"id"})
			for i := 0; i < tc.rows; i++ {
				rows.AddRow(i + 1)
			}
			args := []driver.Value{int64(0)}
			if tc.size >= 0 {
				args = append(args, tc.size)
			}
			args = append(args, tc.size*tc.page)
			mock.ExpectQuery("(?s)SELECT.*WHERE").WithArgs(args...).WillReturnRows(rows).RowsWillBeClosed()
			want := int64(tc.rows)
			if tc.count {
				want = 37
				mock.ExpectQuery("(?s)SELECT COUNT.*WHERE").WithArgs(int64(0)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(want))
			}
			result, count, err := db.List(log.NewEntry(log.New()), table, id, nil, nil, tc.page, tc.size, nil, "", nil, id.GT(pg.Int(0)), nil, nil, nil)
			require.NoError(t, err)
			require.Len(t, result, tc.rows)
			require.Equal(t, want, count)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestShortFirstPageCountRetainsJoinCount(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	id := pg.IntegerColumn("id")
	table := pg.NewTable("", "records", "", id)
	// Even a declaration with no selected join columns must disable inference.
	otherID := pg.IntegerColumn("record_id")
	other := pg.NewTable("", "related_records", "", otherID)
	joins := []actions.ModuleActionJoin{{Table: other, OnCondition: id.EQ(otherID)}}
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(9))
	_, count, err := NewDB(pool).WithShortFirstPageCount().List(log.NewEntry(log.New()), table, id, nil, nil, 0, 10, nil, "", nil, nil, joins, nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 9, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestShortFirstPageCountDoesNotHideReadFailure(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	id := pg.IntegerColumn("id")
	table := pg.NewTable("", "records", "", id)
	failure := errors.New("row stream failed")
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).RowError(0, failure))
	_, _, err = NewDB(pool).WithShortFirstPageCount().List(log.NewEntry(log.New()), table, id, nil, nil, 0, 10, nil, "", nil, nil, nil, nil, nil)
	require.ErrorIs(t, err, failure)
	require.NoError(t, mock.ExpectationsWereMet())
}
