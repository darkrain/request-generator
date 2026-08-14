package db

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/actions"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestAtomicExecutorSelectOneReturnsTypedValues(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	id := pg.IntegerColumn("id")
	userID := pg.IntegerColumn("user_id")
	locationIDs := pg.StringColumn("location_ids")
	profiles := pg.NewTable("public", "profiles", "", id, userID, locationIDs)

	mock.ExpectBegin()
	tx, err := sqlDB.Begin()
	require.NoError(t, err)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "location_ids"}).AddRow(17, "{101,102}"))
	mock.ExpectRollback()

	record, err := NewAtomicExecutor(tx).SelectOne(context.Background(), actions.AtomicSelect{
		Table: profiles,
		Fields: []actions.AtomicSelectField{
			{Name: "agency_id", Column: id, Kind: actions.AtomicValueKindInt},
			{Name: "location_ids", Column: locationIDs, Kind: actions.AtomicValueKindInts},
		},
		Where: userID.EQ(pg.Int(7)),
	})
	require.NoError(t, err)
	agency, ok := record.Field("agency_id")
	require.True(t, ok)
	require.Equal(t, int64(17), *agency.Int)
	locations, ok := record.Field("location_ids")
	require.True(t, ok)
	require.Equal(t, []int64{101, 102}, locations.Ints)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicExecutorSelectOneRequiresWhere(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	id := pg.IntegerColumn("id")
	table := pg.NewTable("public", "profiles", "", id)

	mock.ExpectBegin()
	tx, err := sqlDB.Begin()
	require.NoError(t, err)
	mock.ExpectRollback()

	_, err = NewAtomicExecutor(tx).SelectOne(context.Background(), actions.AtomicSelect{
		Table:  table,
		Fields: []actions.AtomicSelectField{{Name: "id", Column: id, Kind: actions.AtomicValueKindInt}},
	})
	require.Error(t, err)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicExecutorUpsertReturnsInsertAndConflictPaths(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		rows     *sqlmock.Rows
		inserted bool
		value    int64
		update   bool
	}{
		{
			name:     "inserted",
			rows:     sqlmock.NewRows([]string{"id", "inserted"}).AddRow(17, true),
			inserted: true,
			value:    17,
		},
		{
			name:     "conflict do nothing",
			rows:     sqlmock.NewRows([]string{"id", "inserted"}),
			inserted: false,
			value:    0,
		},
		{
			name:     "conflict update",
			rows:     sqlmock.NewRows([]string{"id", "inserted"}).AddRow(17, false),
			inserted: false,
			value:    17,
			update:   true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			id := pg.IntegerColumn("id")
			title := pg.StringColumn("title")
			items := pg.NewTable("public", "atomic_items", "", id, title)

			mock.ExpectBegin()
			tx, err := sqlDB.Begin()
			require.NoError(t, err)
			query := `INSERT INTO public."atomic_items" ("title") VALUES ($1) ON CONFLICT ("title") DO NOTHING RETURNING "id", (xmax = 0) AS inserted`
			args := []driver.Value{"unique"}
			if testCase.update {
				query = `INSERT INTO public."atomic_items" ("title") VALUES ($1) ON CONFLICT ("title") DO UPDATE SET "title" = $2 RETURNING "id", (xmax = 0) AS inserted`
				args = append(args, "updated")
			}
			mock.ExpectQuery(regexp.QuoteMeta(query)).
				WithArgs(args...).
				WillReturnRows(testCase.rows)
			mock.ExpectRollback()

			result, err := NewAtomicExecutor(tx).Upsert(context.Background(), actions.AtomicUpsert{
				Insert: actions.AtomicInsert{
					Table:      items,
					PrimaryKey: id,
					Fields:     []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString("unique")}},
				},
				ConflictColumns: []pg.Column{title},
				UpdateFields: func() []actions.AtomicInsertField {
					if !testCase.update {
						return nil
					}
					return []actions.AtomicInsertField{{Column: title, Value: actions.AtomicString("updated")}}
				}(),
			})
			require.NoError(t, err)
			require.Equal(t, testCase.inserted, result.Inserted)
			require.Equal(t, testCase.value, result.Record.Value)
			require.NoError(t, tx.Rollback())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
