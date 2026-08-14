package module

import (
	"fmt"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	pg "github.com/go-jet/jet/v2/postgres"
)

func (generator *Generator) validateGlobalWidgets() error {
	widgets := make(map[string]actions.WidgetConfig)
	for _, module := range generator.Modules {
		for _, action := range moduleActions(module) {
			widget := actionWidget(action)
			if widget == nil {
				continue
			}
			if err := widget.Validate(); err != nil {
				return fmt.Errorf("module %q action %q: %w", module.Name, action.Action(), err)
			}
			if _, exists := widgets[widget.ID]; exists {
				return fmt.Errorf("widget id %q is duplicated", widget.ID)
			}
			widgets[widget.ID] = *widget
			if err := validateWidgetRequestBindings(module, action, widget.Bindings, nil); err != nil {
				return fmt.Errorf("widget %q: %w", widget.ID, err)
			}
			if widget.Renderer.Workspace != nil {
				if err := generator.validateWorkspaceWidget(widget.ID, *widget.Renderer.Workspace); err != nil {
					return err
				}
			} else if err := validateWidgetResourcePresentation(module, action); err != nil {
				return fmt.Errorf("widget %q: %w", widget.ID, err)
			}
		}
	}
	for _, module := range generator.Modules {
		for _, target := range module.Render.WidgetTargets() {
			widget, exists := widgets[target.ID]
			if !exists {
				return fmt.Errorf("module %q references unknown widget %q", module.Name, target.ID)
			}
			if target.Selection == nil {
				if len(target.Refresh) == 0 {
					continue
				}
				if widget.Renderer.Workspace == nil {
					return fmt.Errorf("module %q action target %q refreshes a non-workspace widget", module.Name, target.ID)
				}
				continue
			}
			if widget.Renderer.Workspace == nil {
				return fmt.Errorf("module %q action target %q sets selection on a non-workspace widget", module.Name, target.ID)
			}
			sourceField := module.GetField(target.Selection.SourceField)
			if sourceField == nil {
				return fmt.Errorf("module %q action target %q selection source field %q is not returned by the action resource", module.Name, target.ID, target.Selection.SourceField)
			}
			masterModule, ok := generator.moduleByName(widget.Renderer.Workspace.Master.Module)
			if !ok {
				return fmt.Errorf("widget %q master resource references unknown module %q", target.ID, widget.Renderer.Workspace.Master.Module)
			}
			selectionField := masterModule.GetField(widget.Renderer.Workspace.Selection.Field)
			if selectionField == nil {
				return fmt.Errorf("widget %q selection field %q is not defined by master module %q", target.ID, widget.Renderer.Workspace.Selection.Field, masterModule.Name)
			}
			if err := validateRuntimeValueTypeCompatibility(*sourceField, *selectionField); err != nil {
				return fmt.Errorf("module %q action target %q selection source field %q: %w", module.Name, target.ID, target.Selection.SourceField, err)
			}
		}
	}
	return nil
}

func moduleActions(module *BaseModule) []actions.ModuleAction {
	result := make([]actions.ModuleAction, 0, len(module.Actions)+1)
	result = append(result, module.Actions...)
	if module.Defrec.Widget != nil {
		result = append(result, module.Defrec)
	}
	return result
}

func (generator *Generator) validateWorkspaceWidget(id string, workspace renderer.WorkspaceWidget) error {
	masterModule, masterAction, err := generator.validateWorkspaceResource(id, "master", workspace.Master, nil)
	if err != nil {
		return err
	}
	if masterAction.Action() != actions.ModuleActionNameList {
		return fmt.Errorf("widget %q master action must be list", id)
	}
	selectionField := masterModule.GetField(workspace.Selection.Field)
	if selectionField == nil {
		return fmt.Errorf("widget %q selection field %q is not defined by master module %q", id, workspace.Selection.Field, masterModule.Name)
	}
	selectionType, err := runtimeTypedValueType(*selectionField)
	if err != nil {
		return fmt.Errorf("widget %q selection field %q: %w", id, workspace.Selection.Field, err)
	}
	selection := widgetSelectionScope{Field: workspace.Selection.Field, Type: selectionType}
	if _, _, err := generator.validateWorkspaceResource(id, "detail", workspace.Detail, &selection); err != nil {
		return err
	}
	for _, subscription := range workspace.Subscriptions {
		module, ok := generator.moduleByName(subscription.Module)
		if !ok {
			return fmt.Errorf("widget %q subscription references unknown module %q", id, subscription.Module)
		}
		for _, name := range subscription.Actions {
			action, ok := findModuleAction(module, name)
			if !ok {
				return fmt.Errorf("widget %q subscription %q references unknown action %q", id, subscription.Module, name)
			}
			event := actions.RealtimeEvent(action)
			if event == nil || event.CorrelationField == "" {
				return fmt.Errorf("widget %q subscription %q action %q does not declare realtime correlation", id, subscription.Module, name)
			}
			if event.CorrelationField != subscription.Correlation.EventField {
				return fmt.Errorf("widget %q subscription %q action %q correlation field %q does not match declared field %q", id, subscription.Module, name, subscription.Correlation.EventField, event.CorrelationField)
			}
			eventField := module.GetField(event.CorrelationField)
			if eventField == nil {
				return fmt.Errorf("widget %q subscription %q action %q declares unknown correlation field %q", id, subscription.Module, name, event.CorrelationField)
			}
			if err := validateRuntimeValueTypeCompatibility(*eventField, *selectionField); err != nil {
				return fmt.Errorf("widget %q subscription %q action %q correlation field %q: %w", id, subscription.Module, name, event.CorrelationField, err)
			}
		}
	}
	return nil
}

func (generator *Generator) validateWorkspaceResource(widgetID, name string, resource renderer.WorkspaceResource, selection *widgetSelectionScope) (*BaseModule, actions.ModuleAction, error) {
	module, ok := generator.moduleByName(resource.Module)
	if !ok {
		return nil, nil, fmt.Errorf("widget %q %s resource references unknown module %q", widgetID, name, resource.Module)
	}
	action, ok := findModuleAction(module, resource.Action)
	if !ok {
		return nil, nil, fmt.Errorf("widget %q %s resource %q references unknown action %q", widgetID, name, resource.Module, resource.Action)
	}
	if err := validateWidgetRequestBindings(module, action, resource.Bindings, selection); err != nil {
		return nil, nil, fmt.Errorf("widget %q %s resource: %w", widgetID, name, err)
	}
	if err := validateWidgetResourcePresentation(module, action); err != nil {
		return nil, nil, fmt.Errorf("widget %q %s resource: %w", widgetID, name, err)
	}
	return module, action, nil
}

func validateWidgetResourcePresentation(module *BaseModule, action actions.ModuleAction) error {
	switch action.Action() {
	case actions.ModuleActionNameList:
		if module.Render.List == nil {
			return fmt.Errorf("list action requires ListPage")
		}
	case actions.ModuleActionNameView:
		pageType := renderer.PageTypeRecord
		switch view := action.(type) {
		case actions.ViewModuleAction:
			pageType = viewActionPageType(view)
		case *actions.ViewModuleAction:
			pageType = viewActionPageType(*view)
		}
		switch pageType {
		case renderer.PageTypeRecord:
			if module.Render.Record == nil {
				return fmt.Errorf("view action requires RecordPage")
			}
		case renderer.PageTypeForm:
			if module.Render.Form == nil {
				return fmt.Errorf("view action requires FormPage")
			}
		default:
			return fmt.Errorf("view action has unsupported page type %q", pageType)
		}
	case actions.ModuleActionNameDefrec:
		if module.Render.Form == nil {
			return fmt.Errorf("defrec action requires FormPage")
		}
	default:
		return fmt.Errorf("action %q does not expose a readable renderer", action.Action())
	}
	return nil
}

type widgetSelectionScope struct {
	Field string
	Type  renderer.TypedValueType
}

func validateWidgetRequestBindings(module *BaseModule, action actions.ModuleAction, bindings []renderer.WidgetRequestBinding, selection *widgetSelectionScope) error {
	if err := renderer.ValidateWidgetRequestBindings(bindings); err != nil {
		return err
	}
	byTarget := make(map[renderer.WidgetRequestBindingTarget]renderer.WidgetRequestBinding, len(bindings))
	for _, binding := range bindings {
		byTarget[binding.Target] = binding
	}

	switch action.Action() {
	case actions.ModuleActionNameView:
		byKey, hasByKey := byTarget[renderer.WidgetRequestBindingPathByKey]
		value, hasValue := byTarget[renderer.WidgetRequestBindingPathValue]
		if !hasByKey || !hasValue {
			return fmt.Errorf("view action requires path_by_key and path_value bindings")
		}
		if len(bindings) != 2 {
			return fmt.Errorf("view action only supports path_by_key and path_value bindings")
		}
		if byKey.Source.Literal == nil || byKey.Source.Literal.Type != renderer.TypedValueString {
			return fmt.Errorf("path_by_key requires a string literal")
		}
		field := widgetViewByField(module, action, byKey.Source.Literal.String)
		if field == nil {
			return fmt.Errorf("path_by_key %q is not declared by view action", byKey.Source.Literal.String)
		}
		fieldType, err := runtimeTypedValueType(*field)
		if err != nil {
			return fmt.Errorf("path_by_key %q: %w", byKey.Source.Literal.String, err)
		}
		return validateWidgetRequestValueSource(value.Source, fieldType, selection)
	case actions.ModuleActionNameList:
		for _, binding := range bindings {
			if binding.Target != renderer.WidgetRequestBindingFilter {
				return fmt.Errorf("list action only supports filter bindings")
			}
			fieldType, ok := widgetListFilterType(module, action, binding.Field)
			if !ok {
				return fmt.Errorf("filter field %q is not declared by list action", binding.Field)
			}
			if err := validateWidgetRequestValueSource(binding.Source, fieldType, selection); err != nil {
				return fmt.Errorf("filter field %q: %w", binding.Field, err)
			}
		}
		return nil
	case actions.ModuleActionNameDefrec:
		if len(bindings) != 0 {
			return fmt.Errorf("defrec action does not accept bindings")
		}
		return nil
	default:
		return fmt.Errorf("action %q does not support widget request bindings", action.Action())
	}
}

func validateWidgetRequestValueSource(source renderer.WidgetValueSource, expected renderer.TypedValueType, selection *widgetSelectionScope) error {
	if source.Literal != nil {
		if source.Literal.Type != expected {
			return fmt.Errorf("literal type %q does not match expected type %q", source.Literal.Type, expected)
		}
		return nil
	}
	if source.Runtime == nil {
		return fmt.Errorf("source is required")
	}
	switch source.Runtime.Scope {
	case renderer.WidgetRuntimeValueSourceCurrentUser:
		if source.Runtime.Field != "id" {
			return fmt.Errorf("current_user field %q is not declared", source.Runtime.Field)
		}
		if expected != renderer.TypedValueNumber {
			return fmt.Errorf("current_user.id has type %q, expected %q", renderer.TypedValueNumber, expected)
		}
	case renderer.WidgetRuntimeValueSourceSelection:
		if selection == nil {
			return fmt.Errorf("selection source is unavailable")
		}
		if source.Runtime.Field != selection.Field {
			return fmt.Errorf("selection field %q does not match declared field %q", source.Runtime.Field, selection.Field)
		}
		if selection.Type != expected {
			return fmt.Errorf("selection field %q has type %q, expected %q", selection.Field, selection.Type, expected)
		}
	default:
		return fmt.Errorf("runtime scope %q is unsupported", source.Runtime.Scope)
	}
	return nil
}

func widgetViewByField(module *BaseModule, action actions.ModuleAction, name string) *fields.ModuleField {
	var by []pg.Column
	switch value := action.(type) {
	case actions.ViewModuleAction:
		by = value.By
	case *actions.ViewModuleAction:
		by = value.By
	default:
		return nil
	}
	for _, column := range by {
		if column.Name() == name {
			return module.GetField(name)
		}
	}
	return nil
}

func widgetListFilterType(module *BaseModule, action actions.ModuleAction, name string) (renderer.TypedValueType, bool) {
	var list actions.ListModuleAction
	switch value := action.(type) {
	case actions.ListModuleAction:
		list = value
	case *actions.ListModuleAction:
		list = *value
	default:
		return "", false
	}
	for _, column := range list.Filter {
		if column.Name() == name {
			field := module.GetField(name)
			if field == nil {
				return "", false
			}
			return runtimeModuleFieldType(field.Type)
		}
	}
	for _, field := range list.VirtualFilters {
		fieldName := field.FieldName
		if fieldName == "" && field.Column != nil {
			fieldName = field.Column.Name()
		}
		if fieldName == name {
			return runtimeModuleFieldType(field.Type)
		}
	}
	return "", false
}

func runtimeTypedValueType(field fields.ModuleField) (renderer.TypedValueType, error) {
	valueType, ok := runtimeModuleFieldType(field.Type)
	if !ok {
		return "", fmt.Errorf("field type %q cannot be used as a runtime value", field.Type)
	}
	return valueType, nil
}

func runtimeModuleFieldType(fieldType fields.ModuleFieldType) (renderer.TypedValueType, bool) {
	switch fieldType {
	case fields.ModuleFieldTypeString:
		return renderer.TypedValueString, true
	case fields.ModuleFieldTypeInt, fields.ModuleFieldTypeFloat:
		return renderer.TypedValueNumber, true
	default:
		return "", false
	}
}

func validateRuntimeValueTypeCompatibility(source, target fields.ModuleField) error {
	sourceType, err := runtimeTypedValueType(source)
	if err != nil {
		return err
	}
	targetType, err := runtimeTypedValueType(target)
	if err != nil {
		return err
	}
	if sourceType != targetType {
		return fmt.Errorf("type %q does not match target type %q", sourceType, targetType)
	}
	return nil
}

func widgetActionPath(module *BaseModule, action actions.ModuleAction) string {
	base := apiQueryURL(module.Path + "/" + module.Name)
	switch action.Action() {
	case actions.ModuleActionNameView:
		return base + "/view/:bykey/:value"
	case actions.ModuleActionNameDefrec:
		return base + "/defrec/"
	default:
		return base
	}
}
