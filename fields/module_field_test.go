package fields

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuleFieldDoesNotSerializeInternalOrdering(t *testing.T) {
	payload, err := json.Marshal(ModuleField{Group: "profile", Order: 10})
	require.NoError(t, err)
	require.NotContains(t, string(payload), "group")
	require.NotContains(t, string(payload), "order")
}

func TestModuleFieldTypeOfRecognizesScalarTypes(t *testing.T) {
	for _, expected := range []ModuleFieldType{ModuleFieldTypeString, ModuleFieldTypeInt, ModuleFieldTypeFloat, ModuleFieldTypeBool} {
		actual, err := ModuleFieldTypeOf(string(expected))
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}
