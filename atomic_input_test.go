package module

import (
	"testing"

	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestAtomicValueFromFieldEncodesJSONStorageArray(t *testing.T) {
	field := fields.ModuleField{
		Column:       pg.StringColumn("media"),
		Type:         fields.ModuleFieldTypeArray,
		ArrayStorage: fields.ModuleFieldArrayStorageJSON,
	}

	value, err := atomicValueFromField(field, []interface{}{
		map[string]interface{}{"cid": "bafy-test", "kind": "image"},
	})
	require.NoError(t, err)
	require.JSONEq(t, `[{"cid":"bafy-test","kind":"image"}]`, string(value.JSON))
	require.Nil(t, value.Strings)
	require.Nil(t, value.Ints)
}

func TestAtomicValueFromFieldKeepsPostgresArrayStorage(t *testing.T) {
	field := fields.ModuleField{Column: pg.StringColumn("tags"), Type: fields.ModuleFieldTypeArray}

	value, err := atomicValueFromField(field, []interface{}{"vip", "verified"})
	require.NoError(t, err)
	require.Equal(t, []string{"vip", "verified"}, value.Strings)
	require.Nil(t, value.JSON)
}

func TestAtomicValueFromFieldRejectsNonArrayJSONStorage(t *testing.T) {
	field := fields.ModuleField{
		Column:       pg.StringColumn("media"),
		Type:         fields.ModuleFieldTypeArray,
		ArrayStorage: fields.ModuleFieldArrayStorageJSON,
	}

	_, err := atomicValueFromField(field, map[string]interface{}{"cid": "bafy-test"})
	require.EqualError(t, err, "expected JSON array")
}

func TestAtomicValueFromFieldBool(t *testing.T) {
	field := fields.ModuleField{Column: pg.BoolColumn("enabled"), Type: fields.ModuleFieldTypeBool}

	value, err := atomicValueFromField(field, true)
	require.NoError(t, err)
	require.NotNil(t, value.Bool)
	require.True(t, *value.Bool)

	_, err = atomicValueFromField(field, "true")
	require.EqualError(t, err, "expected boolean")
}
