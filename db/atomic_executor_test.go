package db

import (
	"context"
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
