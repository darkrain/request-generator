package module

import (
	"errors"
	"testing"

	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestValidationErrorMessageUsesLocalizedFieldTitle(t *testing.T) {
	generator := &Generator{translations: map[locale.Lang]map[string]string{
		locale.RU: {"orders.fields.required_models": "Количество моделей"},
	}}
	field := fields.ModuleField{
		Column: pg.IntegerColumn("required_models"),
		Title:  "orders.fields.required_models",
	}

	message := generator.validationErrorMessage(field, locale.RU, errors.New("required_models - не может быть пустым"))
	require.Equal(t, "Количество моделей - не может быть пустым", message)
}
