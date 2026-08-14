package module

import (
	"fmt"
	"strings"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/renderer"
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
			if err := validateWidgetPathBindings(module, action, widget.Bindings); err != nil {
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
			if target.SelectionField == "" {
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
			if target.SelectionField != widget.Renderer.Workspace.Selection.Field {
				return fmt.Errorf("module %q action target %q selection field %q does not match widget selection %q", module.Name, target.ID, target.SelectionField, widget.Renderer.Workspace.Selection.Field)
			}
			if module.GetField(target.SelectionField) == nil {
				return fmt.Errorf("module %q action target %q references unknown selection field %q", module.Name, target.ID, target.SelectionField)
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
	if masterModule.GetField(workspace.Selection.Field) == nil {
		return fmt.Errorf("widget %q selection field %q is not defined by master module %q", id, workspace.Selection.Field, masterModule.Name)
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
			if _, ok := findModuleAction(module, name); !ok {
				return fmt.Errorf("widget %q subscription %q references unknown action %q", id, subscription.Module, name)
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
	if err := validateWidgetPathBindings(module, action, resource.Bindings); err != nil {
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

func validateWidgetPathBindings(module *BaseModule, action actions.ModuleAction, bindings []renderer.WidgetRequestBinding) error {
	pathParams := routePathParameters(widgetActionPath(module, action))
	for _, binding := range bindings {
		if binding.Target != renderer.WidgetRequestBindingPath {
			continue
		}
		if _, exists := pathParams[binding.Name]; !exists {
			return fmt.Errorf("path binding %q is not declared by action %q", binding.Name, action.Action())
		}
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

func routePathParameters(path string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			result[segment[1:]] = struct{}{}
		}
	}
	return result
}
