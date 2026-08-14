package module

import (
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestValidateModuleFieldArrayStorageRejectsInvalidConfiguration(t *testing.T) {
	media := pg.StringColumn("media")
	table := pg.NewTable("public", "messages", "", media)
	mod := &BaseModule{Table: table, Fields: []fields.ModuleField{{
		Column:       media,
		Type:         fields.ModuleFieldTypeString,
		ArrayStorage: fields.ModuleFieldArrayStorageJSON,
	}}}

	require.EqualError(t, validateModuleFieldArrayStorage(mod), `field "media" configures array storage but is not an array`)
}

func TestValidateModuleFieldArrayStorageRejectsJSONFilter(t *testing.T) {
	media := pg.StringColumn("media")
	table := pg.NewTable("public", "messages", "", media)
	mod := &BaseModule{
		Table: table,
		Fields: []fields.ModuleField{{
			Column:       media,
			Type:         fields.ModuleFieldTypeArray,
			ArrayStorage: fields.ModuleFieldArrayStorageJSON,
		}},
		Actions: []actions.ModuleAction{actions.ListModuleAction{Filter: []pg.Column{media}}},
	}

	require.EqualError(t, validateModuleFieldArrayStorage(mod), `field "media" uses JSON array storage and cannot use the PostgreSQL array filter`)
}

func TestValidateModuleFieldArrayStorageRejectsJSONFilterPointerAction(t *testing.T) {
	media := pg.StringColumn("media")
	table := pg.NewTable("public", "messages", "", media)
	mod := &BaseModule{
		Table: table,
		Fields: []fields.ModuleField{{
			Column:       media,
			Type:         fields.ModuleFieldTypeArray,
			ArrayStorage: fields.ModuleFieldArrayStorageJSON,
		}},
		Actions: []actions.ModuleAction{&actions.ListModuleAction{Filter: []pg.Column{media}}},
	}

	require.EqualError(t, validateModuleFieldArrayStorage(mod), `field "media" uses JSON array storage and cannot use the PostgreSQL array filter`)
}
