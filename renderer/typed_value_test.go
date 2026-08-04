package renderer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedValueMarshalJSONPreservesTypedZero(t *testing.T) {
	payload, err := json.Marshal(TypedValue{Type: TypedValueNumber, Number: 0})
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"number","number":0}`, string(payload))
}

func TestTypedValueMarshalJSONPreservesFalse(t *testing.T) {
	value := false
	payload, err := json.Marshal(TypedValue{Type: TypedValueBool, Bool: &value})
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"bool","bool":false}`, string(payload))
}
