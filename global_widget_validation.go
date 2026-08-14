package module

import (
	"fmt"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
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
			if err := validateWidgetRequestBindingShape(module, action, widget.Bindings); err != nil {
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
		for _, declaration := range module.Render.WidgetTargetActions() {
			target := declaration.Target
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
			if err := generator.validateWidgetTargetSelection(module, declaration, widget); err != nil {
				return err
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
	masterModule, masterAction, err := generator.validateWorkspaceResource(id, "master", workspace.Master)
	if err != nil {
		return err
	}
	if masterAction.Action() != actions.ModuleActionNameList {
		return fmt.Errorf("widget %q master action must be list", id)
	}
	selection, err := generator.workspaceSelectionScope(id, workspace)
	if err != nil {
		return err
	}
	if _, _, err := generator.validateWorkspaceResource(id, "detail", workspace.Detail); err != nil {
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
			if err := validateRuntimeValueTypeCompatibility(*eventField, *masterModule.GetField(selection.Field)); err != nil {
				return fmt.Errorf("widget %q subscription %q action %q correlation field %q: %w", id, subscription.Module, name, event.CorrelationField, err)
			}
		}
	}
	return nil
}

func (generator *Generator) validateWorkspaceResource(widgetID, name string, resource renderer.WorkspaceResource) (*BaseModule, actions.ModuleAction, error) {
	module, ok := generator.moduleByName(resource.Module)
	if !ok {
		return nil, nil, fmt.Errorf("widget %q %s resource references unknown module %q", widgetID, name, resource.Module)
	}
	action, ok := findModuleAction(module, resource.Action)
	if !ok {
		return nil, nil, fmt.Errorf("widget %q %s resource %q references unknown action %q", widgetID, name, resource.Module, resource.Action)
	}
	if err := validateWidgetRequestBindingShape(module, action, resource.Bindings); err != nil {
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

func validateWidgetRequestBindingShape(module *BaseModule, action actions.ModuleAction, bindings []renderer.WidgetRequestBinding) error {
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
		_, hasValue := byTarget[renderer.WidgetRequestBindingPathValue]
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
		return nil
	case actions.ModuleActionNameList:
		for _, binding := range bindings {
			if binding.Target != renderer.WidgetRequestBindingFilter {
				return fmt.Errorf("list action only supports filter bindings")
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

// validateWidgetRequestBindingAvailability resolves the one source of truth
// for filter availability in the current request context. Static validation
// intentionally only checks binding shape because FilterFunc and
// FilterCondition are context-dependent.
func (generator *Generator) validateWidgetRequestBindingAvailability(c *gin.Context, module *BaseModule, action actions.ModuleAction, bindings []renderer.WidgetRequestBinding, selection *widgetSelectionScope) (bool, error) {
	if err := validateWidgetRequestBindingShape(module, action, bindings); err != nil {
		return false, err
	}

	switch action.Action() {
	case actions.ModuleActionNameView:
		for _, binding := range bindings {
			if binding.Target != renderer.WidgetRequestBindingPathValue {
				continue
			}
			byKey := bindings[0]
			if byKey.Target != renderer.WidgetRequestBindingPathByKey {
				byKey = bindings[1]
			}
			field := widgetViewByField(module, action, byKey.Source.Literal.String)
			fieldType, err := runtimeTypedValueType(*field)
			if err != nil {
				return false, fmt.Errorf("path_by_key %q: %w", byKey.Source.Literal.String, err)
			}
			if err := validateWidgetRequestValueSource(binding.Source, fieldType, selection); err != nil {
				return false, err
			}
		}
	case actions.ModuleActionNameList:
		list, ok := widgetListAction(action)
		if !ok {
			return false, fmt.Errorf("list action has unsupported type %T", action)
		}
		filters := generator.effectiveListFilters(c, module, list, generator.getLang(c))
		for _, binding := range bindings {
			field, exists := filters[binding.Field]
			if !exists {
				return false, nil
			}
			fieldType, ok := runtimeModuleFieldType(field.Type)
			if !ok {
				return false, fmt.Errorf("filter field %q type %q cannot be used as a runtime value", binding.Field, field.Type)
			}
			if err := validateWidgetRequestValueSource(binding.Source, fieldType, selection); err != nil {
				return false, fmt.Errorf("filter field %q: %w", binding.Field, err)
			}
		}
	}
	return true, nil
}

func widgetListAction(action actions.ModuleAction) (actions.ListModuleAction, bool) {
	switch value := action.(type) {
	case actions.ListModuleAction:
		return value, true
	case *actions.ListModuleAction:
		return *value, true
	default:
		return actions.ListModuleAction{}, false
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

func runtimeTypedValueType(field fields.ModuleField) (renderer.TypedValueType, error) {
	valueType, ok := runtimeModuleFieldType(field.Type)
	if !ok {
		return "", fmt.Errorf("field type %q cannot be used as a runtime value", field.Type)
	}
	return valueType, nil
}

func (generator *Generator) workspaceSelectionScope(widgetID string, workspace renderer.WorkspaceWidget) (widgetSelectionScope, error) {
	masterModule, ok := generator.moduleByName(workspace.Master.Module)
	if !ok {
		return widgetSelectionScope{}, fmt.Errorf("widget %q master resource references unknown module %q", widgetID, workspace.Master.Module)
	}
	selectionField := masterModule.GetField(workspace.Selection.Field)
	if selectionField == nil {
		return widgetSelectionScope{}, fmt.Errorf("widget %q selection field %q is not defined by master module %q", widgetID, workspace.Selection.Field, masterModule.Name)
	}
	selectionType, err := runtimeTypedValueType(*selectionField)
	if err != nil {
		return widgetSelectionScope{}, fmt.Errorf("widget %q selection field %q: %w", widgetID, workspace.Selection.Field, err)
	}
	return widgetSelectionScope{Field: workspace.Selection.Field, Type: selectionType}, nil
}

func (generator *Generator) validateWidgetTargetSelection(owner *BaseModule, declaration renderer.WidgetTargetAction, widget actions.WidgetConfig) error {
	target := declaration.Target
	if !declaration.AfterSuccess {
		return fmt.Errorf("module %q action target %q selection is only allowed after success", owner.Name, target.ID)
	}
	if declaration.ActionType != renderer.ActionAPI || declaration.Request == nil {
		return fmt.Errorf("module %q action target %q selection requires an api action request", owner.Name, target.ID)
	}

	source := target.Selection.Source
	sourceModule, ok := generator.moduleByName(source.Resource.Module)
	if !ok {
		return fmt.Errorf("module %q action target %q selection source resource references unknown module %q", owner.Name, target.ID, source.Resource.Module)
	}
	sourceAction, ok := findModuleAction(sourceModule, source.Resource.Action)
	if !ok {
		return fmt.Errorf("module %q action target %q selection source resource %q references unknown action %q", owner.Name, target.ID, source.Resource.Module, source.Resource.Action)
	}
	expected, ok := standardActionRequest(sourceModule, sourceAction)
	if !ok {
		return fmt.Errorf("module %q action target %q selection source resource %q action %q has no standard request", owner.Name, target.ID, source.Resource.Module, source.Resource.Action)
	}
	if declaration.Request.Method != expected.Method || declaration.Request.Endpoint != expected.Endpoint {
		return fmt.Errorf("module %q action target %q selection source resource %q action %q does not match action request %s %s", owner.Name, target.ID, source.Resource.Module, source.Resource.Action, expected.Method, expected.Endpoint)
	}
	sourceType, err := standardActionResponseFieldType(sourceAction, source.Field)
	if err != nil {
		return fmt.Errorf("module %q action target %q selection source resource %q action %q: %w", owner.Name, target.ID, source.Resource.Module, source.Resource.Action, err)
	}
	selection, err := generator.workspaceSelectionScope(target.ID, *widget.Renderer.Workspace)
	if err != nil {
		return err
	}
	if sourceType != selection.Type {
		return fmt.Errorf("module %q action target %q selection source field %q has type %q, target field %q has type %q", owner.Name, target.ID, source.Field, sourceType, selection.Field, selection.Type)
	}
	return nil
}

func standardActionRequest(module *BaseModule, action actions.ModuleAction) (renderer.APIAction, bool) {
	base := apiQueryURL(module.Path + "/" + module.Name)
	switch action.Action() {
	case actions.ModuleActionNameList:
		return renderer.APIAction{Method: "GET", Endpoint: base}, true
	case actions.ModuleActionNameAdd:
		return renderer.APIAction{Method: "PUT", Endpoint: base}, true
	case actions.ModuleActionNameDefrec:
		return renderer.APIAction{Method: "GET", Endpoint: base + "/defrec/"}, true
	case actions.ModuleActionNameView:
		return renderer.APIAction{Method: "GET", Endpoint: base + "/view/:bykey/:value"}, true
	case actions.ModuleActionNameUpdate:
		return renderer.APIAction{Method: "POST", Endpoint: base + "/:bykey/:value"}, true
	case actions.ModuleActionNameDelete:
		return renderer.APIAction{Method: "DELETE", Endpoint: base + "/delete/:bykey/:value"}, true
	default:
		return renderer.APIAction{}, false
	}
}

func standardActionResponseFieldType(action actions.ModuleAction, field string) (renderer.TypedValueType, error) {
	if action.Action() != actions.ModuleActionNameAdd {
		return "", fmt.Errorf("action does not expose a typed scalar response")
	}
	switch field {
	case "value":
		return renderer.TypedValueNumber, nil
	case "primary_key":
		return renderer.TypedValueString, nil
	default:
		return "", fmt.Errorf("response field %q is not declared", field)
	}
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
