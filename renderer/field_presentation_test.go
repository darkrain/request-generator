package renderer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldPresentationInputModeValidationAndClone(t *testing.T) {
	source := &FieldPresentation{InputMode: FieldInputModeNumeric}
	require.NoError(t, source.Validate())
	require.Equal(t, FieldInputModeNumeric, CloneFieldPresentation(source).InputMode)
	require.EqualError(t, (&FieldPresentation{InputMode: "digits"}).Validate(), `renderer.FieldPresentation: unsupported input mode "digits"`)
}
