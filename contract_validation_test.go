package module

import (
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestValidateContractAcceptsCompleteAPIDrivenModule(t *testing.T) {
	id := pg.IntegerColumn("id")
	name := pg.StringColumn("name")
	table := pg.NewTable("public", "items", "", id, name)
	contract := &BaseModule{
		Name: "items", Label: "items.title", Table: table, PrimaryKey: id, Path: "/api",
		Fields: []fields.ModuleField{
			{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeHidden},
			{Column: name, Title: "items.fields.name", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Columns: []pg.Column{id, name}},
			actions.ViewModuleAction{Columns: []pg.Column{id, name}, By: []pg.Column{id}, PageType: renderer.PageTypeRecord},
			actions.AddModuleAction{Columns: []pg.Column{name}},
			actions.UpdateModuleAction{Columns: []pg.Column{name}, By: []pg.Column{id}},
		},
		Navigation: []NavigationEntry{{
			ActionName: "list", Path: "/items", Title: "items.title", Show: true,
			Target: NavigationTarget{Type: "page", PageType: renderer.PageTypeList},
		}},
		Render: renderer.Universal{
			List: &renderer.ListPage{Title: "items.title", Grid: &renderer.Grid{Enabled: true, Mode: renderer.GridModeTable}},
			Form: &renderer.FormPage{ID: "items-form", Fields: []string{"name"}, Sections: []renderer.FormSection{{ID: "main", Fields: []string{"name"}}}},
			Record: &renderer.RecordPage{ID: "items-record", Sections: []renderer.RecordSection{{
				ID: "details", Components: []renderer.DisplayComponent{{ID: "fields", Type: renderer.DisplayDataList, Fields: []string{"id", "name"}}},
			}}},
		},
	}

	report := contract.ValidateContract()
	require.True(t, report.Valid, report.Error())
	require.Empty(t, report.Issues)
}

func TestValidateContractReportsIncompleteSurfacesAndBrokenReferences(t *testing.T) {
	id := pg.IntegerColumn("id")
	name := pg.StringColumn("name")
	table := pg.NewTable("public", "items", "", id, name)
	contract := &BaseModule{
		Name: "items", Label: "items.title", Table: table, PrimaryKey: id, Path: "/api",
		Fields: []fields.ModuleField{
			{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeHidden},
			{Column: name, Title: "items.fields.name", Type: fields.ModuleFieldTypeString, FormType: fields.ModuleFieldFormTypeText},
		},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{},
			actions.ViewModuleAction{Columns: []pg.Column{id, name}, By: []pg.Column{id}, PageType: renderer.PageTypeRecord},
			actions.AddModuleAction{Columns: []pg.Column{name}},
		},
		Render: renderer.Universal{
			List:   &renderer.ListPage{Title: "items.title"},
			Form:   &renderer.FormPage{ID: "items-form", Fields: []string{"missing"}},
			Record: &renderer.RecordPage{ID: "items-record"},
		},
	}

	report := contract.ValidateContract()
	require.False(t, report.Valid)
	requireIssueCodes(t, report,
		"LIST_HAS_NO_COLUMNS",
		"RENDERER_FIELD_UNKNOWN",
		"FORM_WRITE_FIELD_MISSING",
		"RECORD_HAS_NO_VISIBLE_CONTENT",
	)
}

func TestValidateContractRequiresRendererForActions(t *testing.T) {
	id := pg.IntegerColumn("id")
	table := pg.NewTable("public", "items", "", id)
	contract := &BaseModule{
		Name: "items", Label: "items.title", Table: table, PrimaryKey: id,
		Fields: []fields.ModuleField{{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeHidden}},
		Actions: []actions.ModuleAction{
			actions.ListModuleAction{Columns: []pg.Column{id}},
			actions.ViewModuleAction{Columns: []pg.Column{id}, By: []pg.Column{id}, PageType: renderer.PageTypeRecord},
		},
	}

	report := contract.ValidateContract()
	requireIssueCodes(t, report, "LIST_RENDERER_REQUIRED", "RECORD_RENDERER_REQUIRED")
}

func TestValidateContractSupportsPointerActionsAndChecksReadColumns(t *testing.T) {
	id := pg.IntegerColumn("id")
	missing := pg.StringColumn("missing")
	table := pg.NewTable("public", "items", "", id)
	contract := &BaseModule{
		Name: "items", Label: "items.title", Table: table, PrimaryKey: id,
		Fields:  []fields.ModuleField{{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeHidden}},
		Actions: []actions.ModuleAction{&actions.ListModuleAction{Columns: []pg.Column{id, missing}}},
		Render:  renderer.Universal{List: &renderer.ListPage{Title: "items.title", Grid: &renderer.Grid{Enabled: true, Mode: renderer.GridModeTable}}},
	}
	report := contract.ValidateContract()
	requireIssueCodes(t, report, "ACTION_FIELD_UNKNOWN")
}

func TestValidateContractChecksResourceGridNavigationRenderer(t *testing.T) {
	id := pg.IntegerColumn("id")
	table := pg.NewTable("public", "items", "", id)
	contract := &BaseModule{
		Name: "items", Label: "items.title", Table: table, PrimaryKey: id,
		Fields: []fields.ModuleField{{Column: id, Title: "items.fields.id", Type: fields.ModuleFieldTypeInt, FormType: fields.ModuleFieldFormTypeHidden}},
		Navigation: []NavigationEntry{{
			ActionName: "defrec", Target: NavigationTarget{PageType: renderer.PageTypeResourceGrid},
		}},
	}

	report := contract.ValidateContract()
	requireIssueCodes(t, report, "NAVIGATION_RENDERER_MISMATCH")
}

func requireIssueCodes(t *testing.T, report ContractReport, expected ...string) {
	t.Helper()
	actual := make(map[string]struct{}, len(report.Issues))
	for _, issue := range report.Issues {
		actual[issue.Code] = struct{}{}
	}
	for _, code := range expected {
		if _, exists := actual[code]; !exists {
			t.Fatalf("ожидалась ошибка %s, получено: %#v", code, report.Issues)
		}
	}
}
