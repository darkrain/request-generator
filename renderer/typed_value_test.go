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

func TestTypedValueValidate(t *testing.T) {
	value := false
	require.NoError(t, (TypedValue{Type: TypedValueString}).Validate())
	require.NoError(t, (TypedValue{Type: TypedValueNumber, Number: 0}).Validate())
	require.NoError(t, (TypedValue{Type: TypedValueBool, Bool: &value}).Validate())
	require.EqualError(t, (TypedValue{Type: TypedValueBool}).Validate(), "boolean typed value requires bool")
	require.EqualError(t, (TypedValue{}).Validate(), `unsupported typed value type ""`)
}
