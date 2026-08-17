package module

import (
	"testing"

	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/locale"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestLocalizeResultValueUsesRequestTranslatorWithoutMutatingRecord(t *testing.T) {
	generator := &Generator{translations: map[locale.Lang]map[string]string{
		locale.RU: {"notification.preview.title": "Новое уведомление"},
	}}
	preview := pg.StringColumn("preview_payload")
	fields := []fields.ModuleField{{
		Column: preview,
		ResultValueLocalizer: func(value interface{}, translate func(string, string) string) interface{} {
			payload := value.(map[string]interface{})
			return map[string]interface{}{"title": translate("notification.preview.title", payload["title"].(string))}
		},
	}}
	original := map[string]interface{}{"preview_payload": map[string]interface{}{"title": "New notification"}}

	localized := generator.localizeResultValue(locale.RU, fields, original).(map[string]interface{})
	require.Equal(t, "Новое уведомление", localized["preview_payload"].(map[string]interface{})["title"])
	require.Equal(t, "New notification", original["preview_payload"].(map[string]interface{})["title"])
}

func TestLocalizeResultListLeavesFieldsWithoutLocalizerUntouched(t *testing.T) {
	generator := &Generator{}
	name := pg.StringColumn("name")
	rows := []interface{}{map[string]interface{}{"name": "Anna"}}

	localized := generator.localizeResultList(locale.EN, []fields.ModuleField{{Column: name}}, rows)
	require.Equal(t, rows, localized)
}
