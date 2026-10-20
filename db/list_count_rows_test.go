package db

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestListCountRowsKeepsPageTotalAndEmptyPageFallback(t *testing.T) {
	for _, tc := range []struct {
		name       string
		page, size int64
		populated  bool
		count      int64
	}{
		{"first", 0, 2, true, 7}, {"later", 2, 2, true, 7}, {"beyond", 4, 2, false, 7}, {"empty", 0, 2, false, 0}, {"zero size", 0, 0, false, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var seen []string
			pool, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(_, actual string) error { seen = append(seen, actual); return nil })))
			require.NoError(t, err)
			defer pool.Close()
			id, label := pg.IntegerColumn("id"), pg.StringColumn("label")
			table := pg.NewTable("public", "records", "", id, label)
			base := NewDB(pool)
			executor := WithListCountRows(base)
			row := sqlmock.NewRows([]string{"id", "label", "total"})
			if tc.populated {
				row.AddRow(12, "first", tc.count).AddRow(13, "second", tc.count)
			}
			mock.ExpectQuery("page").WithArgs("allowed", tc.size, tc.page*tc.size).WillReturnRows(row).RowsWillBeClosed()
			if !tc.populated {
				mock.ExpectQuery("count").WithArgs("allowed").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.count))
			}
			result, count, err := executor.List(log.NewEntry(log.New()), table, id, []fields.ModuleField{{Column: label, Type: fields.ModuleFieldTypeString}}, nil, tc.page, tc.size, nil, "", nil, label.EQ(pg.String("allowed")), nil, &actions.SortOption{Column: id, Direction: actions.SortDESC}, nil)
			require.NoError(t, err)
			require.Equal(t, tc.count, count)
			if tc.populated {
				require.Equal(t, []interface{}{map[string]interface{}{"label": "first"}, map[string]interface{}{"label": "second"}}, result)
			} else {
				require.Nil(t, result)
			}
			if tc.size > 0 {
				require.Contains(t, seen[0], "COUNT(*) OVER ()")
				require.Equal(t, 1, strings.Count(seen[0], "WHERE"))
				require.Contains(t, seen[0], "ORDER BY records.id DESC")
			} else {
				require.NotContains(t, seen[0], "OVER ()")
			}
			if len(seen) > 1 {
				require.NotContains(t, seen[1], "__rg_matched")
			}
			require.False(t, base.listCountWithRows)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestListCountRowsJoinedReadRetainsDistinctCount(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	id, otherID := pg.IntegerColumn("id"), pg.IntegerColumn("id")
	table, other := pg.NewTable("public", "records", "", id), pg.NewTable("public", "related", "", otherID)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`COUNT\(DISTINCT records.id\)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(9))
	_, count, err := WithListCountRows(NewDB(pool)).List(log.NewEntry(log.New()), table, id, nil, nil, 0, 10, nil, "", nil, nil, []actions.ModuleActionJoin{{Table: other, OnCondition: id.EQ(otherID)}}, nil, nil)
	require.NoError(t, err)
	require.EqualValues(t, 9, count)
	require.NoError(t, mock.ExpectationsWereMet())
}
