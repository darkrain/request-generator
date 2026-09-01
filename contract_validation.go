package module

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	pg "github.com/go-jet/jet/v2/postgres"
)

// ContractSeverity определяет, блокирует ли проблема генерацию контракта.
type ContractSeverity string

const (
	ContractSeverityError   ContractSeverity = "error"
	ContractSeverityWarning ContractSeverity = "warning"
)

// ContractIssue описывает стабильное машиночитаемое нарушение семантики контракта.
type ContractIssue struct {
	Code     string           `json:"code"`
	Severity ContractSeverity `json:"severity"`
	Path     string           `json:"path"`
	Message  string           `json:"message"`
}

// ContractReport содержит все семантические проблемы исполняемого модуля.
type ContractReport struct {
	Module string          `json:"module"`
	Valid  bool            `json:"valid"`
	Issues []ContractIssue `json:"issues"`
}

func (report ContractReport) Error() string {
	if report.Valid {
		return ""
	}
	parts := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", issue.Code, issue.Path, issue.Message))
	}
	return strings.Join(parts, "; ")
}

// ValidateContract проверяет, достаточно ли полного BaseModule для универсального
// API-driven клиента. Динамические callbacks producer-а метод не выполняет.
func (module *BaseModule) ValidateContract() ContractReport {
	validator := contractValidator{
		module:      module,
		fieldNames:  make(map[string]fields.ModuleField),
		actionNames: make(map[actions.ModuleActionName]struct{}),
	}
	validator.validate()
	sort.SliceStable(validator.issues, func(i, j int) bool {
		if validator.issues[i].Path == validator.issues[j].Path {
			return validator.issues[i].Code < validator.issues[j].Code
		}
		return validator.issues[i].Path < validator.issues[j].Path
	})
	valid := true
	for _, issue := range validator.issues {
		if issue.Severity == ContractSeverityError {
			valid = false
			break
		}
	}
	report := ContractReport{Issues: validator.issues, Valid: valid}
	if module != nil {
		report.Module = module.Name
	}
	return report
}

type contractValidator struct {
	module      *BaseModule
	issues      []ContractIssue
	fieldNames  map[string]fields.ModuleField
	actionNames map[actions.ModuleActionName]struct{}
}

func (validator *contractValidator) validate() {
	if validator.module == nil {
		validator.add("MODULE_REQUIRED", "/", "модуль не задан")
		return
	}
	validator.validateModuleIdentity()
	validator.indexFields()
	validator.validateActions()
	validator.validateRenderer()
	validator.validateNavigation()
}

func (validator *contractValidator) validateModuleIdentity() {
	if strings.TrimSpace(validator.module.Name) == "" {
		validator.add("MODULE_NAME_REQUIRED", "/name", "имя модуля обязательно")
	}
	if strings.TrimSpace(validator.module.Label) == "" {
		validator.add("MODULE_LABEL_REQUIRED", "/label", "серверный ключ заголовка модуля обязателен")
	}
	if validator.module.Table == nil {
		validator.add("MODULE_TABLE_REQUIRED", "/table", "таблица модуля обязательна")
	}
	if validator.module.PrimaryKey == nil {
		validator.add("MODULE_PRIMARY_KEY_REQUIRED", "/primary_key", "первичный ключ модуля обязателен")
	}
	if len(validator.module.Fields) == 0 {
		validator.add("MODULE_FIELDS_EMPTY", "/fields", "модуль должен объявлять хотя бы одно поле")
	}
}

func (validator *contractValidator) indexFields() {
	primaryFound := validator.module.PrimaryKey == nil
	for index, field := range validator.module.Fields {
		name := field.Name()
		path := fmt.Sprintf("/fields/%d", index)
		if name == "" {
			validator.add("FIELD_NAME_REQUIRED", path, "поле должно иметь имя колонки или логическое имя")
			continue
		}
		if _, exists := validator.fieldNames[name]; exists {
			validator.add("FIELD_NAME_DUPLICATED", path, fmt.Sprintf("поле %q объявлено несколько раз", name))
			continue
		}
		validator.fieldNames[name] = field
		if columnName := field.ColumnName(); columnName != "" {
			if previous, exists := validator.fieldNames[columnName]; exists && previous.Name() != name {
				validator.add("FIELD_REFERENCE_DUPLICATED", path, fmt.Sprintf("имя колонки %q конфликтует с полем %q", columnName, previous.Name()))
			}
			validator.fieldNames[columnName] = field
		}
		if strings.TrimSpace(field.Title) == "" {
			validator.add("FIELD_TITLE_REQUIRED", path+"/title", fmt.Sprintf("поле %q не имеет серверного заголовка", name))
		}
		if validator.module.PrimaryKey != nil && field.Column != nil && field.Column.Name() == validator.module.PrimaryKey.Name() {
			primaryFound = true
		}
	}
	if !primaryFound {
		validator.add("PRIMARY_KEY_FIELD_MISSING", "/primary_key", "первичный ключ не объявлен среди полей модуля")
	}
}

func (validator *contractValidator) validateActions() {
	for index, action := range validator.module.Actions {
		path := fmt.Sprintf("/actions/%d", index)
		if action == nil || (reflect.ValueOf(action).Kind() == reflect.Ptr && reflect.ValueOf(action).IsNil()) {
			validator.add("ACTION_REQUIRED", path, "action не может быть nil")
			continue
		}
		name := action.Action()
		if _, exists := validator.actionNames[name]; exists {
			validator.add("ACTION_DUPLICATED", path, fmt.Sprintf("action %q объявлен несколько раз", name))
			continue
		}
		validator.actionNames[name] = struct{}{}
		switch value := action.(type) {
		case actions.ListModuleAction:
			validator.validateListAction(path, value)
		case *actions.ListModuleAction:
			if value != nil {
				validator.validateListAction(path, *value)
			}
		case actions.ViewModuleAction:
			validator.validateViewAction(path, value)
		case *actions.ViewModuleAction:
			if value != nil {
				validator.validateViewAction(path, *value)
			}
		case actions.AddModuleAction:
			validator.validateWriteColumns(path, value.Columns, value.ColumnsFunc != nil || len(value.Fields) > 0)
		case *actions.AddModuleAction:
			if value != nil {
				validator.validateWriteColumns(path, value.Columns, value.ColumnsFunc != nil || len(value.Fields) > 0)
			}
		case actions.UpdateModuleAction:
			validator.validateUpdateAction(path, value)
		case *actions.UpdateModuleAction:
			if value != nil {
				validator.validateUpdateAction(path, *value)
			}
		}
	}
}

func (validator *contractValidator) validateListAction(path string, value actions.ListModuleAction) {
	if len(value.Columns) == 0 && value.ColumnsFunc == nil && len(value.Fields) == 0 {
		validator.add("LIST_HAS_NO_COLUMNS", path+"/columns", "список не возвращает ни одного поля")
	}
	validator.validateReadColumns(path, value.Columns)
}

func (validator *contractValidator) validateViewAction(path string, value actions.ViewModuleAction) {
	validator.validateReadColumns(path, value.Columns)
	if len(value.By) == 0 && validator.module.PrimaryKey == nil {
		validator.add("VIEW_LOOKUP_REQUIRED", path+"/by", "просмотр должен иметь поле поиска или первичный ключ")
	}
}

func (validator *contractValidator) validateUpdateAction(path string, value actions.UpdateModuleAction) {
	validator.validateWriteColumns(path, value.Columns, value.ColumnsFunc != nil || len(value.Fields) > 0)
	if len(value.By) == 0 && validator.module.PrimaryKey == nil {
		validator.add("UPDATE_LOOKUP_REQUIRED", path+"/by", "обновление должно иметь поле поиска или первичный ключ")
	}
}

func (validator *contractValidator) validateReadColumns(path string, columns []pg.Column) {
	for index, column := range columns {
		if column == nil {
			validator.add("ACTION_COLUMN_REQUIRED", fmt.Sprintf("%s/columns/%d", path, index), "колонка не может быть nil")
			continue
		}
		if _, exists := validator.fieldNames[column.Name()]; !exists {
			validator.add("ACTION_FIELD_UNKNOWN", fmt.Sprintf("%s/columns/%d", path, index), fmt.Sprintf("поле %q отсутствует в модуле", column.Name()))
		}
	}
}

func (validator *contractValidator) validateWriteColumns(path string, columns []pg.Column, dynamic bool) {
	if len(columns) == 0 && !dynamic {
		validator.add("WRITE_HAS_NO_COLUMNS", path+"/columns", "записывающее действие не содержит изменяемых полей")
		return
	}
	for index, column := range columns {
		if column == nil {
			validator.add("ACTION_COLUMN_REQUIRED", fmt.Sprintf("%s/columns/%d", path, index), "колонка не может быть nil")
			continue
		}
		if _, exists := validator.fieldNames[column.Name()]; !exists {
			validator.add("ACTION_FIELD_UNKNOWN", fmt.Sprintf("%s/columns/%d", path, index), fmt.Sprintf("поле %q отсутствует в модуле", column.Name()))
		}
	}
}

func (validator *contractValidator) validateRenderer() {
	if err := validator.module.Render.Validate(); err != nil {
		validator.add("RENDERER_INVALID", "/render", err.Error())
	}
	validator.validateListRenderer()
	validator.validateFormRenderer()
	validator.validateRecordRenderer()
}

func (validator *contractValidator) validateListRenderer() {
	if _, exists := validator.actionNames[actions.ModuleActionNameList]; !exists {
		return
	}
	if validator.module.Render.List == nil && validator.module.Render.ResourceGrid == nil {
		validator.add("LIST_RENDERER_REQUIRED", "/render/list_page", "для list action требуется list_page или resource_grid_page")
		return
	}
	page := validator.module.Render.List
	if page == nil {
		return
	}
	if page.GroupBy != nil {
		validator.requireField(page.GroupBy.Field, "/render/list_page/group_by/field")
	}
	if page.Selection != nil {
		validator.requireField(page.Selection.KeyField, "/render/list_page/selection/key_field")
		validator.requireField(page.Selection.ValuesField, "/render/list_page/selection/values_field")
	}
	validator.validateCardSchema(page.CardSchema, "/render/list_page/card_schema")
}

func (validator *contractValidator) validateFormRenderer() {
	_, hasAdd := validator.actionNames[actions.ModuleActionNameAdd]
	_, hasUpdate := validator.actionNames[actions.ModuleActionNameUpdate]
	if !hasAdd && !hasUpdate {
		return
	}
	page := validator.module.Render.Form
	if page == nil {
		validator.add("FORM_RENDERER_REQUIRED", "/render/form_page", "для add/update action требуется form_page")
		return
	}
	declared := make(map[string]struct{})
	for index, field := range page.Fields {
		path := fmt.Sprintf("/render/form_page/fields/%d", index)
		validator.requireField(field, path)
		declared[field] = struct{}{}
	}
	for sectionIndex, section := range page.Sections {
		for fieldIndex, field := range section.Fields {
			path := fmt.Sprintf("/render/form_page/sections/%d/fields/%d", sectionIndex, fieldIndex)
			validator.requireField(field, path)
			declared[field] = struct{}{}
		}
	}
	if len(declared) == 0 {
		validator.add("FORM_HAS_NO_FIELDS", "/render/form_page/fields", "форма не содержит ни одного поля")
		return
	}
	validator.requireWriteFieldsInForm(declared)
}

func (validator *contractValidator) requireWriteFieldsInForm(declared map[string]struct{}) {
	for index, action := range validator.module.Actions {
		var columns []pg.Column
		switch value := action.(type) {
		case actions.AddModuleAction:
			columns = value.Columns
		case *actions.AddModuleAction:
			if value != nil {
				columns = value.Columns
			}
		case actions.UpdateModuleAction:
			columns = value.Columns
		case *actions.UpdateModuleAction:
			if value != nil {
				columns = value.Columns
			}
		default:
			continue
		}
		for _, column := range columns {
			if column == nil {
				continue
			}
			field, exists := validator.fieldNames[column.Name()]
			if !exists || field.FormType == fields.ModuleFieldFormTypeHidden || field.FormType == fields.ModuleFieldFormTypeOnlyView {
				continue
			}
			if _, exists := declared[column.Name()]; !exists {
				validator.add("FORM_WRITE_FIELD_MISSING", fmt.Sprintf("/actions/%d/columns", index), fmt.Sprintf("изменяемое поле %q отсутствует в форме", column.Name()))
			}
		}
	}
}

func (validator *contractValidator) validateRecordRenderer() {
	needsRecord := false
	for _, action := range validator.module.Actions {
		var view actions.ViewModuleAction
		switch value := action.(type) {
		case actions.ViewModuleAction:
			view = value
		case *actions.ViewModuleAction:
			if value == nil {
				continue
			}
			view = *value
		default:
			continue
		}
		if view.PageType == "" || view.PageType == renderer.PageTypeRecord || view.PageTypeFunc != nil {
			needsRecord = true
		}
		if view.PageType == renderer.PageTypeForm && validator.module.Render.Form == nil {
			validator.add("FORM_RENDERER_REQUIRED", "/render/form_page", "view action с page_type=form требует form_page")
		}
	}
	if !needsRecord {
		return
	}
	page := validator.module.Render.Record
	if page == nil {
		validator.add("RECORD_RENDERER_REQUIRED", "/render/record_page", "для record view требуется record_page")
		return
	}
	hasContent := false
	for sectionIndex, section := range page.Sections {
		for componentIndex, component := range section.Components {
			path := fmt.Sprintf("/render/record_page/sections/%d/components/%d", sectionIndex, componentIndex)
			for fieldIndex, field := range component.Fields {
				validator.requireField(field, fmt.Sprintf("%s/fields/%d", path, fieldIndex))
			}
			for itemIndex, item := range component.Items {
				validator.requireField(item.Field, fmt.Sprintf("%s/items/%d/field", path, itemIndex))
			}
			if component.CollectionGroups != nil {
				validator.requireField(component.CollectionGroups.SourceField, path+"/collection_groups/source_field")
			}
			if componentHasPotentialContent(component, page) {
				hasContent = true
			}
		}
	}
	if !hasContent {
		validator.add("RECORD_HAS_NO_VISIBLE_CONTENT", "/render/record_page/sections", "карточка записи не содержит отображаемого контента")
	}
}

func componentHasPotentialContent(component renderer.DisplayComponent, page *renderer.RecordPage) bool {
	if component.Visible != nil && !*component.Visible {
		return false
	}
	if len(component.Fields) > 0 || len(component.Items) > 0 || len(component.MediaItems) > 0 || component.Value != nil || component.Default != nil || component.CollectionGroups != nil {
		return true
	}
	return component.Type == renderer.DisplayActions && len(page.Actions) > 0
}

func (validator *contractValidator) validateCardSchema(schema *renderer.CardSchema, path string) {
	if schema == nil {
		return
	}
	for suffix, binding := range map[string]*renderer.TextBinding{
		"title": schema.Title, "subtitle": schema.Subtitle, "meta": schema.Meta, "description": schema.Description,
	} {
		if binding != nil && binding.Field != "" {
			validator.requireField(binding.Field, path+"/"+suffix+"/field")
		}
	}
	if schema.Media != nil {
		validator.requireOptionalField(schema.Media.Field, path+"/media/field")
		validator.requireOptionalField(schema.Media.GlowField, path+"/media/glow_field")
		validator.requireOptionalField(schema.Media.StatusField, path+"/media/status_field")
	}
	if schema.Status != nil {
		validator.requireOptionalField(schema.Status.Field, path+"/status/field")
	}
	if schema.Icon != nil {
		validator.requireOptionalField(schema.Icon.Field, path+"/icon/field")
		validator.requireOptionalField(schema.Icon.IconField, path+"/icon/icon_field")
		validator.requireOptionalField(schema.Icon.ToneField, path+"/icon/tone_field")
	}
	for index, badge := range append(append([]renderer.Badge{}, schema.Badges...), schema.Stats...) {
		base := fmt.Sprintf("%s/badges/%d", path, index)
		validator.requireOptionalField(badge.Field, base+"/field")
		validator.requireOptionalField(badge.IfField, base+"/if_field")
		if badge.Value != nil {
			validator.requireOptionalField(badge.Value.Field, base+"/value/field")
		}
	}
}

func (validator *contractValidator) validateNavigation() {
	for index, entry := range validator.module.Navigation {
		path := fmt.Sprintf("/navigation/%d", index)
		name := actions.ModuleActionName(entry.ActionName)
		if _, exists := validator.actionNames[name]; !exists && name != actions.ModuleActionNameDefrec {
			validator.add("NAVIGATION_ACTION_UNKNOWN", path+"/action", fmt.Sprintf("action %q отсутствует в модуле", entry.ActionName))
		}
		validator.validatePageType(entry.Target.PageType, path+"/target/page_type")
	}
	for index, route := range validator.module.Routes {
		path := fmt.Sprintf("/routes/%d", index)
		name := actions.ModuleActionName(route.ActionName)
		if _, exists := validator.actionNames[name]; !exists && name != actions.ModuleActionNameDefrec {
			validator.add("ROUTE_ACTION_UNKNOWN", path+"/action", fmt.Sprintf("action %q отсутствует в модуле", route.ActionName))
		}
		validator.validatePageType(route.Target.PageType, path+"/target/page_type")
	}
}

func (validator *contractValidator) validatePageType(pageType renderer.PageType, path string) {
	switch pageType {
	case "":
		return
	case renderer.PageTypeList:
		if validator.module.Render.List == nil && validator.module.Render.ResourceGrid == nil {
			validator.add("NAVIGATION_RENDERER_MISMATCH", path, "list page не имеет list renderer")
		}
	case renderer.PageTypeResourceGrid:
		if validator.module.Render.ResourceGrid == nil {
			validator.add("NAVIGATION_RENDERER_MISMATCH", path, "resource_grid page не имеет resource grid renderer")
		}
	case renderer.PageTypeForm:
		if validator.module.Render.Form == nil {
			validator.add("NAVIGATION_RENDERER_MISMATCH", path, "form page не имеет form renderer")
		}
	case renderer.PageTypeRecord:
		if validator.module.Render.Record == nil {
			validator.add("NAVIGATION_RENDERER_MISMATCH", path, "record page не имеет record renderer")
		}
	}
}

func (validator *contractValidator) requireOptionalField(field, path string) {
	if field != "" {
		validator.requireField(field, path)
	}
}

func (validator *contractValidator) requireField(field, path string) {
	if strings.TrimSpace(field) == "" {
		validator.add("RENDERER_FIELD_REQUIRED", path, "ссылка на поле обязательна")
		return
	}
	if _, exists := validator.fieldNames[field]; !exists {
		validator.add("RENDERER_FIELD_UNKNOWN", path, fmt.Sprintf("поле %q отсутствует в модуле", field))
	}
}

func (validator *contractValidator) add(code, path, message string) {
	validator.issues = append(validator.issues, ContractIssue{
		Code: code, Severity: ContractSeverityError, Path: path, Message: message,
	})
}
