package db

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
)

func TestGroupByPrimaryAndJoinedFieldsKeepsJoinedColumns(t *testing.T) {
	preferenceID := pg.IntegerColumn("id")
	base := pg.NewTable("public", "preferences", "", preferenceID)
	available := pg.BoolColumn("email_available")
	position := pg.IntegerColumn("position")
	groups := pg.NewTable("public", "notification_groups", "", available, position)
	fields := []fields.ModuleField{{Column: preferenceID}, {Column: available}}
	clauses := groupByPrimaryAndJoinedFields(preferenceID, fields, base.TableName(), position)
	statement := pg.SELECT(preferenceID, available).FROM(base.INNER_JOIN(groups, pg.Bool(true))).GROUP_BY(clauses...)
	query, _ := statement.Sql()

	if !strings.Contains(query, "GROUP BY preferences.id, notification_groups.email_available, notification_groups.position") {
		t.Fatalf("joined fields must remain in GROUP BY, got %s", query)
	}
}

func TestModuleFieldResultValueParsesJSONArray(t *testing.T) {
	field := fields.ModuleField{Type: fields.ModuleFieldTypeArray}

	value := moduleFieldResultValue(field, &sql.NullString{
		String: `[{"id":1,"type":"bondage"},{"id":2,"type":"gfe"}]`,
		Valid:  true,
	})

	items, ok := value.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", value)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first item map, got %T", items[0])
	}
	if first["type"] != "bondage" {
		t.Fatalf("expected first type bondage, got %v", first["type"])
	}
}

func TestModuleFieldResultValueParsesJSONObject(t *testing.T) {
	field := fields.ModuleField{Type: fields.ModuleFieldTypeObject}

	value := moduleFieldResultValue(field, &sql.NullString{
		String: `{"enabled":true,"count":3}`,
		Valid:  true,
	})

	object, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", value)
	}
	if object["enabled"] != true {
		t.Fatalf("expected enabled true, got %v", object["enabled"])
	}
}

func TestModuleFieldResultValueKeepsPostgresTextArray(t *testing.T) {
	field := fields.ModuleField{Type: fields.ModuleFieldTypeArray}

	value := moduleFieldResultValue(field, &sql.NullString{
		String: `{cash,cards}`,
		Valid:  true,
	})

	if value != "{cash,cards}" {
		t.Fatalf("expected postgres text array string to stay unchanged, got %#v", value)
	}
}

func TestModuleFieldResultValueUsesExplicitConverter(t *testing.T) {
	field := fields.ModuleField{
		Type: fields.ModuleFieldTypeArray,
		ResultValueConverter: func(value interface{}) interface{} {
			return "converted"
		},
	}

	value := moduleFieldResultValue(field, &sql.NullString{
		String: `[{"id":1}]`,
		Valid:  true,
	})

	if value != "converted" {
		t.Fatalf("expected explicit converter result, got %#v", value)
	}
}

func TestDBValueEncodesJSONArrayStorage(t *testing.T) {
	field := fields.ModuleField{Type: fields.ModuleFieldTypeArray, ArrayStorage: fields.ModuleFieldArrayStorageJSON}

	value := dbValue(field, []interface{}{
		map[string]interface{}{"cid": "bafy-test", "kind": "image"},
	})

	encoded, ok := value.(string)
	if !ok {
		t.Fatalf("expected JSON string, got %T", value)
	}
	if encoded != `[{"cid":"bafy-test","kind":"image"}]` {
		t.Fatalf("unexpected JSON encoding: %s", encoded)
	}
}
