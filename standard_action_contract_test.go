package module

import (
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/renderer"
	"github.com/stretchr/testify/require"
)

func TestAtomicUpdateContractDeclaresTypedResult(t *testing.T) {
	contract, ok := resolveStandardActionContract(&BaseModule{Path: "/api", Name: "entries"}, actions.UpdateModuleAction{Mode: actions.UpdateModeAtomic})
	require.True(t, ok)
	require.Equal(t, "POST", contract.Request.Method)
	require.Equal(t, "/api/entries/:bykey/:value", contract.Request.Endpoint)
	require.Equal(t, renderer.TypedValueNumber, mustActionResultType(t, contract, renderer.ActionResultFieldValue))
	require.Equal(t, renderer.TypedValueString, mustActionResultType(t, contract, renderer.ActionResultFieldPrimaryKey))
}

func TestStandardUpdateContractDoesNotDeclareAtomicResult(t *testing.T) {
	contract, ok := resolveStandardActionContract(&BaseModule{Path: "/api", Name: "entries"}, actions.UpdateModuleAction{})
	require.True(t, ok)
	_, exists := contract.resultFieldType(renderer.ActionResultFieldValue)
	require.False(t, exists)
}

func mustActionResultType(t *testing.T, contract standardActionContract, field renderer.ActionResultField) renderer.TypedValueType {
	t.Helper()
	value, ok := contract.resultFieldType(field)
	require.True(t, ok)
	return value
}
