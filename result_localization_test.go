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

func TestLocalizeResultListKeepsRequestLanguagesIndependent(t *testing.T) {
	generator := &Generator{translations: map[locale.Lang]map[string]string{
		locale.RU: {"label": "Название"},
		locale.EN: {"label": "Title"},
	}}
	columns := []fields.ModuleField{{Column: pg.StringColumn("title"), ResultValueLocalizer: func(value interface{}, translate func(string, string) string) interface{} {
		return translate("label", value.(string))
	}}}
	rows := []interface{}{map[string]interface{}{"title": "original"}, map[string]interface{}{"other": "unchanged"}, nil}
	ru := generator.localizeResultList(locale.RU, columns, rows)
	en := generator.localizeResultList(locale.EN, columns, rows)
	require.Equal(t, "Название", ru[0].(map[string]interface{})["title"])
	require.Equal(t, "Title", en[0].(map[string]interface{})["title"])
	require.Equal(t, "original", rows[0].(map[string]interface{})["title"])
	require.Equal(t, rows[1:], ru[1:])
}

func BenchmarkResultListWithoutLocalizers(b *testing.B) {
	generator := &Generator{}
	columns := []fields.ModuleField{{Column: pg.StringColumn("name")}, {Column: pg.IntegerColumn("id")}}
	rows := make([]interface{}, 100)
	for i := range rows {
		rows[i] = map[string]interface{}{"name": "name", "id": i}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if len(generator.localizeResultList(locale.EN, columns, rows)) != len(rows) {
			b.Fatal("lost rows")
		}
	}
}
