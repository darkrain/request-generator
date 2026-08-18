package db

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const jsonArrayStorageValue = `[{"cid":"bafy-test","kind":"image"}]`
const jsonObjectStorageValue = `{"source":"profile_settings","upload":true}`

func TestDBAddStoresJSONArray(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	media := pg.StringColumn("media")
	messages := pg.NewTable("public", "messages", "", id, media)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."messages" ("media") VALUES ($1) RETURNING "id"`)).
		WithArgs(jsonArrayStorageValue).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(17))
	mock.ExpectCommit()

	result, err := NewDB(sqlDB).Add(log.NewEntry(log.New()), messages, id, []fields.ModuleField{{
		Column:       media,
		Type:         fields.ModuleFieldTypeArray,
		ArrayStorage: fields.ModuleFieldArrayStorageJSON,
	}}, map[string]interface{}{
		"media": []interface{}{map[string]interface{}{"cid": "bafy-test", "kind": "image"}},
	}, nil)

	require.NoError(t, err)
	stored, ok := result.(struct {
		Value      int64  `json:"value"`
		PrimaryKey string `json:"primary_key"`
	})
	require.True(t, ok)
	require.Equal(t, int64(17), stored.Value)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBAddStoresJSONObject(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	metadata := pg.StringColumn("metadata")
	assets := pg.NewTable("public", "assets", "", id, metadata)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."assets" ("metadata") VALUES ($1) RETURNING "id"`)).
		WithArgs(jsonObjectStorageValue).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(18))
	mock.ExpectCommit()

	_, err = NewDB(sqlDB).Add(log.NewEntry(log.New()), assets, id, []fields.ModuleField{{
		Column: metadata,
		Type:   fields.ModuleFieldTypeObject,
	}}, map[string]interface{}{
		"metadata": map[string]interface{}{"source": "profile_settings", "upload": true},
	}, nil)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDBUpdateStoresJSONObject(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	metadata := pg.StringColumn("metadata")
	assets := pg.NewTable("public", "assets", "", id, metadata)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public."assets" SET "metadata" = $1 WHERE assets.id = $2`)).
		WithArgs(jsonObjectStorageValue, 18).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT assets.id AS "assets.id", assets.metadata AS "assets.metadata" FROM public.assets WHERE assets.id = $1 GROUP BY assets.id LIMIT $2;`)).
		WithArgs(18, 1).
		WillReturnRows(sqlmock.NewRows([]string{"assets.id", "assets.metadata"}).AddRow(18, jsonObjectStorageValue))

	_, err = NewDB(sqlDB).Update(log.NewEntry(log.New()), assets, id, []fields.ModuleField{{
		Column: metadata,
		Type:   fields.ModuleFieldTypeObject,
	}}, map[string]interface{}{
		"metadata": map[string]interface{}{"source": "profile_settings", "upload": true},
	}, id.EQ(pg.Int(18)), nil)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAtomicInsertStoresJSONArray(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	id := pg.IntegerColumn("id")
	media := pg.StringColumn("media")
	messages := pg.NewTable("public", "messages", "", id, media)
	mock.ExpectBegin()
	tx, err := sqlDB.Begin()
	require.NoError(t, err)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO public."messages" ("media") VALUES ($1) RETURNING "id"`)).
		WithArgs(jsonArrayStorageValue).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	mock.ExpectRollback()

	record, err := NewAtomicExecutor(tx).Insert(context.Background(), actions.AtomicInsert{
		Table:      messages,
		PrimaryKey: id,
		Fields: []actions.AtomicInsertField{{
			Column: media,
			Value:  actions.AtomicValue{JSON: []byte(jsonArrayStorageValue)},
		}},
	})

	require.NoError(t, err)
	require.Equal(t, int64(21), record.Value)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
