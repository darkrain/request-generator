package module

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/locale"
	"github.com/darkrain/request-generator/renderer"
	"github.com/darkrain/request-generator/response"
	"github.com/gin-gonic/gin"
)

// ConfigResponse структурирует ответ эндпоинта /api/config
type ConfigResponse struct {
	Navigation []ConfigNavigationEntry `json:"navigation"`
	Routes     []ConfigRouteEntry      `json:"routes"`
	Widgets    []ConfigWidget          `json:"widgets"`
	Role       string                  `json:"role"`
}

type ConfigRouteEntry struct {
	Path   string               `json:"path"`
	Target NavigationPageTarget `json:"target"`
}

type ConfigNavigationEntry struct {
	ID         string                 `json:"id,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Target     NavigationPageTarget   `json:"target"`
	Title      string                 `json:"title"`
	Icon       string                 `json:"icon,omitempty"`
	Order      int                    `json:"order,omitempty"`
	Group      string                 `json:"group,omitempty"`
	GroupTitle string                 `json:"group_title,omitempty"`
	Query      map[string]interface{} `json:"query,omitempty"`
}

type NavigationPageTarget struct {
	Type     string                 `json:"type"`
	Name     string                 `json:"name,omitempty"`
	Params   map[string]interface{} `json:"params,omitempty"`
	Renderer *renderer.Identity     `json:"renderer,omitempty"`
	PageType renderer.PageType      `json:"page_type,omitempty"`
	Query    *RouteQuery            `json:"query,omitempty"`
	Children map[string]RouteConfig `json:"children,omitempty"`
}

type ConfigWidget struct {
	ID       string                `json:"id"`
	Order    int                   `json:"order,omitempty"`
	Renderer renderer.Identity     `json:"renderer"`
	Widget   renderer.GlobalWidget `json:"widget"`
	Load     renderer.WidgetLoad   `json:"load"`
}

// RouteConfig конфигурирует маршрут
type RouteConfig struct {
	Title     string                 `json:"title"`
	MenuTitle string                 `json:"menuTitle,omitempty"`
	Renderer  *renderer.Identity     `json:"renderer,omitempty"`
	PageType  renderer.PageType      `json:"page_type,omitempty"`
	Query     *RouteQuery            `json:"query,omitempty"`
	Children  map[string]RouteConfig `json:"children,omitempty"`
}

// RouteQuery описывает параметры запроса для маршрута
type RouteQuery struct {
	Url    string                 `json:"url"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// actionConfigEndpoint генерирует конфиг для webapp
func (generator *Generator) actionConfigEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		l, _ := icontext.GetLogger(ctx)

		user, ok := icontext.GetUser(ctx)
		if !ok {
			response.ErrorResponse(l, c, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		role := user.Role
		lang := generator.getLang(c)
		generator.setTranslationContext(c, lang)

		navigation, err := generator.buildNavigation(c, role, lang)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		routes, err := generator.buildRouteRegistry(c, role)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		widgets, err := generator.buildWidgets(c, role)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		config := ConfigResponse{
			Navigation: navigation,
			Routes:     routes,
			Widgets:    widgets,
			Role:       role,
		}

		response.Response(l, c, config)
	}
}

// hasPermission проверяет, есть ли у роли доступ к действию
func hasPermission(action actions.ModuleAction, role string) bool {
	var permissions []actions.Role

	switch a := action.(type) {
	case actions.ListModuleAction:
		permissions = a.Permission
	case *actions.ListModuleAction:
		permissions = a.Permission
	case actions.AddModuleAction:
		permissions = a.Permission
	case *actions.AddModuleAction:
		permissions = a.Permission
	case actions.ViewModuleAction:
		permissions = a.Permission
	case *actions.ViewModuleAction:
		permissions = a.Permission
	case actions.UpdateModuleAction:
		permissions = a.Permission
	case *actions.UpdateModuleAction:
		permissions = a.Permission
	case actions.DeleteModuleAction:
		permissions = a.Permission
	case *actions.DeleteModuleAction:
		permissions = a.Permission
	case actions.DefrecModuleAction:
		permissions = a.Permission
	case *actions.DefrecModuleAction:
		permissions = a.Permission
	default:
		return false
	}

	if len(permissions) == 0 {
		return true
	}

	for _, perm := range permissions {
		if string(perm) == role || perm == actions.RoleAll {
			return true
		}
	}

	return false
}

func navigationID(module *BaseModule, entry NavigationEntry) string {
	if entry.ID != "" {
		return entry.ID
	}
	if entry.Group != "" || entry.ActionName != "" {
		return entry.Group + "." + module.Name + "." + entry.ActionName
	}
	return module.Name
}

func navigationPath(module *BaseModule, entry NavigationEntry, targetType string) string {
	if entry.Path != "" {
		return entry.Path
	}
	if targetType != "page" {
		return ""
	}
	return module.Path + "/" + module.Name
}

func navigationTarget(entry NavigationEntry) NavigationPageTarget {
	target := NavigationPageTarget{
		Type:   entry.Target.Type,
		Name:   entry.Target.Name,
		Params: entry.Target.Params,
	}
	if target.Type == "" {
		target.Type = "page"
	}
	return target
}

func (generator *Generator) buildNavigation(c *gin.Context, role string, lang locale.Lang) ([]ConfigNavigationEntry, error) {
	var result []ConfigNavigationEntry

	for _, module := range generator.Modules {
		for _, entry := range module.Navigation {
			if !entry.Show {
				continue
			}
			action, ok := findModuleAction(module, entry.ActionName)
			if !ok || !hasPermission(action, role) || !navigationRoleAllowed(entry, role) {
				continue
			}

			target, err := generator.buildPageTarget(c, module, action, entry.Target, entry.Query, role)
			if err != nil {
				return nil, err
			}

			configEntry := ConfigNavigationEntry{
				ID:     navigationID(module, entry),
				Path:   navigationPath(module, entry, target.Type),
				Title:  generator.Translate(lang, entry.Title),
				Icon:   entry.Icon,
				Group:  entry.Group,
				Order:  entry.Order,
				Target: target,
				Query:  entry.Query,
			}
			if titleKey, ok := generator.GroupTitles[entry.Group]; ok {
				configEntry.GroupTitle = generator.Translate(lang, titleKey)
			}
			result = append(result, configEntry)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Group == result[j].Group {
			return result[i].Order < result[j].Order
		}
		return result[i].Group < result[j].Group
	})

	return result, nil
}

func (generator *Generator) buildRouteRegistry(c *gin.Context, role string) ([]ConfigRouteEntry, error) {
	result := make([]ConfigRouteEntry, 0)
	seen := make(map[string]struct{})
	appendRoute := func(module *BaseModule, path string, action actions.ModuleAction, targetConfig NavigationTarget, queryParams map[string]interface{}) error {
		if path == "" || targetConfig.Type != "" && targetConfig.Type != "page" {
			return nil
		}
		target, err := generator.buildPageTarget(c, module, action, targetConfig, queryParams, role)
		if err != nil {
			return err
		}
		if _, exists := seen[path]; exists {
			return fmt.Errorf("config route path %q is declared more than once", path)
		}
		seen[path] = struct{}{}
		result = append(result, ConfigRouteEntry{Path: path, Target: target})
		return nil
	}

	for _, module := range generator.Modules {
		for _, entry := range module.Navigation {
			if !entry.Show || !navigationRoleAllowed(entry, role) {
				continue
			}
			action, ok := findModuleAction(module, entry.ActionName)
			if !ok || !hasPermission(action, role) {
				continue
			}
			if err := appendRoute(module, navigationPath(module, entry, "page"), action, entry.Target, entry.Query); err != nil {
				return nil, err
			}
		}
		for _, page := range module.Routes {
			action, ok := findModuleAction(module, page.ActionName)
			if !ok || !hasPermission(action, role) || !routeRolesAllowed(page, role) {
				continue
			}
			if err := appendRoute(module, page.Path, action, page.Target, nil); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (generator *Generator) buildPageTarget(c *gin.Context, module *BaseModule, action actions.ModuleAction, targetConfig NavigationTarget, queryParams map[string]interface{}, role string) (NavigationPageTarget, error) {
	target := navigationTarget(NavigationEntry{Target: targetConfig})
	if target.Type != "page" {
		return target, nil
	}
	render, err := module.RenderFor(c)
	if err != nil {
		return NavigationPageTarget{}, err
	}
	route := generator.buildRouteForAction(module, render, action, role)
	if targetConfig.PageType != "" {
		route.Renderer = viewRouteIdentity(render, targetConfig.PageType)
		route.PageType = viewRoutePageType(render, targetConfig.PageType)
	}
	target.Renderer = route.Renderer
	target.PageType = route.PageType
	target.Query = route.Query
	target.Children = route.Children
	if target.Query != nil && queryParams != nil {
		target.Query.Params = queryParams
	}
	return target, nil
}

func navigationRoleAllowed(entry NavigationEntry, role string) bool {
	if len(entry.Roles) == 0 {
		return true
	}
	for _, r := range entry.Roles {
		if string(r) == role || r == actions.RoleAll {
			return true
		}
	}
	return false
}

func routeRolesAllowed(route RoutablePage, role string) bool {
	if len(route.Roles) == 0 {
		return true
	}
	for _, allowed := range route.Roles {
		if string(allowed) == role || allowed == actions.RoleAll {
			return true
		}
	}
	return false
}

func findModuleAction(module *BaseModule, actionName string) (actions.ModuleAction, bool) {
	if actionName == string(actions.ModuleActionNameDefrec) {
		return module.Defrec, true
	}
	for _, action := range module.Actions {
		if string(action.Action()) == actionName {
			return action, true
		}
	}
	return nil, false
}

func (generator *Generator) buildRouteForAction(module *BaseModule, render renderer.Universal, action actions.ModuleAction, role string) RouteConfig {
	switch a := action.(type) {
	case actions.ListModuleAction:
		return generator.buildListRoute(module, render, a, role)
	case *actions.ListModuleAction:
		return generator.buildListRoute(module, render, *a, role)
	case actions.ViewModuleAction:
		return generator.buildViewRoute(module, render, a)
	case *actions.ViewModuleAction:
		return generator.buildViewRoute(module, render, *a)
	case actions.AddModuleAction:
		return generator.buildAddRoute(module, render, a)
	case *actions.AddModuleAction:
		return generator.buildAddRoute(module, render, *a)
	case actions.DefrecModuleAction:
		return generator.buildDefrecRoute(module, render, a)
	case *actions.DefrecModuleAction:
		return generator.buildDefrecRoute(module, render, *a)
	default:
		return RouteConfig{}
	}
}

func (generator *Generator) buildWidgets(c *gin.Context, role string) ([]ConfigWidget, error) {
	var result []ConfigWidget

	for _, module := range generator.Modules {
		for _, action := range module.Actions {
			widget := actionWidget(action)
			if widget == nil || !hasPermission(action, role) {
				continue
			}
			config, available, err := generator.buildWidget(c, module, action, *widget, role)
			if err != nil {
				return nil, err
			}
			if available {
				result = append(result, config)
			}
		}

		if module.Defrec.Widget != nil && hasPermission(module.Defrec, role) {
			config, available, err := generator.buildWidget(c, module, module.Defrec, *module.Defrec.Widget, role)
			if err != nil {
				return nil, err
			}
			if available {
				result = append(result, config)
			}
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Widget.Surface.Placement == result[j].Widget.Surface.Placement {
			return result[i].Order < result[j].Order
		}
		return result[i].Widget.Surface.Placement < result[j].Widget.Surface.Placement
	})

	return result, nil
}

func (generator *Generator) buildWidget(c *gin.Context, owner *BaseModule, action actions.ModuleAction, widget actions.WidgetConfig, role string) (ConfigWidget, bool, error) {
	load, available, err := generator.buildWidgetLoad(c, owner, action, widget, role)
	if err != nil || !available {
		return ConfigWidget{}, available, err
	}
	localized := renderer.LocalizeGlobalWidget(widget.Renderer, generator.rendererTextResolver(generator.getLang(c)))
	filterWorkspaceCommands(&localized, load.Commands)
	return ConfigWidget{
		ID:       widget.ID,
		Order:    widget.Order,
		Renderer: localized.Identity(),
		Widget:   localized,
		Load:     load,
	}, true, nil
}

func filterWorkspaceCommands(widget *renderer.GlobalWidget, loads []renderer.WorkspaceCommandLoad) {
	if widget == nil || widget.Workspace == nil || len(widget.Workspace.Commands) == 0 {
		return
	}
	available := make(map[string]struct{}, len(loads))
	for _, load := range loads {
		available[load.ID] = struct{}{}
	}
	commands := make([]renderer.WorkspaceCommand, 0, len(loads))
	for _, command := range widget.Workspace.Commands {
		if _, exists := available[command.ID]; exists {
			commands = append(commands, command)
		}
	}
	widget.Workspace.Commands = commands
}

func (generator *Generator) buildWidgetLoad(c *gin.Context, owner *BaseModule, action actions.ModuleAction, widget actions.WidgetConfig, role string) (renderer.WidgetLoad, bool, error) {
	if widget.Renderer.Workspace == nil {
		resource, available, err := generator.buildWidgetResourceLoad(c, owner, action, widget.Bindings, nil)
		if err != nil || !available {
			return renderer.WidgetLoad{}, available, err
		}
		return renderer.WidgetLoad{Resource: &resource}, true, nil
	}

	workspace := widget.Renderer.Workspace
	selection, err := generator.workspaceSelectionScopeForContext(c, widget.ID, *workspace)
	if err != nil {
		return renderer.WidgetLoad{}, false, err
	}
	var summary *renderer.WidgetResourceLoad
	if workspace.Summary != nil {
		resource, available, err := generator.buildReferencedWidgetResourceLoad(c, *workspace.Summary, role, &selection)
		if err != nil || !available {
			return renderer.WidgetLoad{}, available, err
		}
		summary = &resource
	}
	master, available, err := generator.buildReferencedWidgetResourceLoad(c, workspace.Master, role, nil)
	if err != nil || !available {
		return renderer.WidgetLoad{}, available, err
	}
	detail, available, err := generator.buildReferencedWidgetResourceLoad(c, workspace.Detail, role, &selection)
	if err != nil || !available {
		return renderer.WidgetLoad{}, available, err
	}
	commands, err := generator.buildWorkspaceCommandLoads(c, workspace.Commands, role, &selection)
	if err != nil {
		return renderer.WidgetLoad{}, false, err
	}
	return renderer.WidgetLoad{Summary: summary, Master: &master, Detail: &detail, Commands: commands}, true, nil
}

func (generator *Generator) buildWorkspaceCommandLoads(c *gin.Context, commands []renderer.WorkspaceCommand, role string, selection *widgetSelectionScope) ([]renderer.WorkspaceCommandLoad, error) {
	loads := make([]renderer.WorkspaceCommandLoad, 0, len(commands))
	for _, command := range commands {
		load, available, err := generator.buildWorkspaceCommandLoad(c, command, role, selection)
		if err != nil {
			return nil, fmt.Errorf("workspace command %q: %w", command.ID, err)
		}
		if !available {
			continue
		}
		loads = append(loads, load)
	}
	return loads, nil
}

func (generator *Generator) buildReferencedWidgetResourceLoad(c *gin.Context, resource renderer.WorkspaceResource, role string, selection *widgetSelectionScope) (renderer.WidgetResourceLoad, bool, error) {
	module, ok := generator.moduleByName(resource.Module)
	if !ok {
		return renderer.WidgetResourceLoad{}, false, fmt.Errorf("widget resource references unknown module %q", resource.Module)
	}
	action, ok := findModuleAction(module, resource.Action)
	if !ok {
		return renderer.WidgetResourceLoad{}, false, fmt.Errorf("widget resource %q references unknown action %q", resource.Module, resource.Action)
	}
	if !hasPermission(action, role) {
		return renderer.WidgetResourceLoad{}, false, nil
	}
	load, available, err := generator.buildWidgetResourceLoad(c, module, action, resource.Bindings, selection)
	if err != nil || !available {
		return renderer.WidgetResourceLoad{}, available, err
	}
	return load, true, nil
}

func (generator *Generator) buildWorkspaceCommandLoad(c *gin.Context, command renderer.WorkspaceCommand, role string, selection *widgetSelectionScope) (renderer.WorkspaceCommandLoad, bool, error) {
	module, ok := generator.moduleByName(command.Module)
	if !ok {
		return renderer.WorkspaceCommandLoad{}, false, fmt.Errorf("references unknown module %q", command.Module)
	}
	action, ok := findModuleAction(module, command.Action)
	if !ok {
		return renderer.WorkspaceCommandLoad{}, false, fmt.Errorf("resource %q references unknown action %q", command.Module, command.Action)
	}
	if !hasPermission(action, role) {
		return renderer.WorkspaceCommandLoad{}, false, nil
	}
	if !workspaceCommandPresentationAvailable(command, *selection) {
		return renderer.WorkspaceCommandLoad{}, false, nil
	}
	inputScope, inputLoad, available, err := workspaceCommandInputScopeForContext(c, module, action, command.Input)
	if err != nil || !available {
		return renderer.WorkspaceCommandLoad{}, available, err
	}
	resource, available, err := generator.buildWorkspaceCommandResourceLoad(c, module, action, command.Bindings, selection, inputScope)
	if err != nil || !available {
		return renderer.WorkspaceCommandLoad{}, available, err
	}
	return renderer.WorkspaceCommandLoad{
		ID:       command.ID,
		Request:  resource.Request,
		Bindings: resource.Bindings,
		Input:    inputLoad,
	}, true, nil
}

func (generator *Generator) buildWidgetResourceLoad(c *gin.Context, module *BaseModule, action actions.ModuleAction, bindings []renderer.WidgetRequestBinding, selection *widgetSelectionScope) (renderer.WidgetResourceLoad, bool, error) {
	available, err := generator.validateWidgetRequestBindingAvailability(c, module, action, bindings, selection, nil)
	if err != nil || !available {
		return renderer.WidgetResourceLoad{}, available, err
	}
	if _, err := module.RenderFor(c); err != nil {
		return renderer.WidgetResourceLoad{}, false, err
	}
	contract, ok := resolveStandardActionContract(module, action)
	if !ok {
		return renderer.WidgetResourceLoad{}, false, fmt.Errorf("widget resource %q action %q has no request", module.Name, action.Action())
	}
	return renderer.WidgetResourceLoad{
		Request:  contract.Request,
		Bindings: append([]renderer.WidgetRequestBinding(nil), bindings...),
	}, true, nil
}

func (generator *Generator) buildWorkspaceCommandResourceLoad(c *gin.Context, module *BaseModule, action actions.ModuleAction, bindings []renderer.WidgetRequestBinding, selection *widgetSelectionScope, input *workspaceCommandInputScope) (renderer.WidgetResourceLoad, bool, error) {
	available, err := generator.validateWidgetRequestBindingAvailability(c, module, action, bindings, selection, input)
	if err != nil || !available {
		return renderer.WidgetResourceLoad{}, available, err
	}
	contract, ok := resolveStandardActionContract(module, action)
	if !ok {
		return renderer.WidgetResourceLoad{}, false, fmt.Errorf("workspace command resource %q action %q has no request", module.Name, action.Action())
	}
	return renderer.WidgetResourceLoad{
		Request:  contract.Request,
		Bindings: append([]renderer.WidgetRequestBinding(nil), bindings...),
	}, true, nil
}

func (generator *Generator) moduleByName(name string) (*BaseModule, bool) {
	for _, candidate := range generator.Modules {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return nil, false
}

func actionWidget(action actions.ModuleAction) *actions.WidgetConfig {
	switch a := action.(type) {
	case actions.ListModuleAction:
		return a.Widget
	case *actions.ListModuleAction:
		return a.Widget
	case actions.ViewModuleAction:
		return a.Widget
	case *actions.ViewModuleAction:
		return a.Widget
	case actions.AddModuleAction:
		return a.Widget
	case *actions.AddModuleAction:
		return a.Widget
	case actions.UpdateModuleAction:
		return a.Widget
	case *actions.UpdateModuleAction:
		return a.Widget
	case actions.DeleteModuleAction:
		return a.Widget
	case *actions.DeleteModuleAction:
		return a.Widget
	case actions.DefrecModuleAction:
		return a.Widget
	case *actions.DefrecModuleAction:
		return a.Widget
	default:
		return nil
	}
}

func (generator *Generator) buildListRoute(module *BaseModule, render renderer.Universal, action actions.ListModuleAction, role string) RouteConfig {
	route := RouteConfig{
		Title:     action.Label,
		MenuTitle: action.Label,
		Renderer:  render.ListIdentity(),
		PageType:  render.ListRoutePageType(),
		Query:     standardActionRouteQuery(module, action),
	}

	route.Children = generator.buildRouteChildren(module, render, role)

	return route
}

func (generator *Generator) buildViewRoute(module *BaseModule, render renderer.Universal, action actions.ViewModuleAction) RouteConfig {
	pageType := viewActionPageType(action)

	return RouteConfig{
		Title:    action.Label,
		Renderer: viewRouteIdentity(render, pageType),
		PageType: viewRoutePageType(render, pageType),
		Query:    standardActionRouteQuery(module, action),
	}
}

func viewActionPageType(action actions.ViewModuleAction) renderer.PageType {
	if action.PageType != "" {
		return action.PageType
	}
	return renderer.PageTypeRecord
}

func viewActionPageTypeForContext(action actions.ViewModuleAction, c *gin.Context) renderer.PageType {
	if action.PageTypeFunc != nil {
		if pageType := action.PageTypeFunc(c); pageType != "" {
			return pageType
		}
	}
	return viewActionPageType(action)
}

func viewRouteIdentity(render renderer.Universal, pageType renderer.PageType) *renderer.Identity {
	switch pageType {
	case renderer.PageTypeForm:
		return render.FormIdentity()
	case renderer.PageTypeRecord:
		return render.RecordIdentity()
	default:
		return nil
	}
}

func viewRoutePageType(render renderer.Universal, pageType renderer.PageType) renderer.PageType {
	switch pageType {
	case renderer.PageTypeForm:
		return render.FormRoutePageType()
	case renderer.PageTypeRecord:
		return render.RecordRoutePageType()
	default:
		return ""
	}
}

func (generator *Generator) buildAddRoute(module *BaseModule, render renderer.Universal, action actions.AddModuleAction) RouteConfig {
	return RouteConfig{
		Title:    action.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query:    standardActionRouteQuery(module, action),
	}
}

func (generator *Generator) buildDefrecRoute(module *BaseModule, render renderer.Universal, action actions.DefrecModuleAction) RouteConfig {
	return RouteConfig{
		Title:    action.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query:    standardActionRouteQuery(module, action),
	}
}

func apiQueryURL(path string) string {
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return path
	}
	return "/api" + path
}

// buildRouteChildren формирует children маршруты (view, edit, add)
func (generator *Generator) buildRouteChildren(module *BaseModule, render renderer.Universal, role string) map[string]RouteConfig {
	children := make(map[string]RouteConfig)

	for _, action := range module.Actions {
		switch a := action.(type) {
		case actions.ViewModuleAction:
			if child, ok := generator.buildViewChild(module, render, a, role); ok {
				children[":id"] = child
			}
		case *actions.ViewModuleAction:
			if child, ok := generator.buildViewChild(module, render, *a, role); ok {
				children[":id"] = child
			}
		case actions.UpdateModuleAction:
			if child, ok := generator.buildUpdateChild(module, render, a, role); ok {
				children[":id/edit"] = child
			}
		case *actions.UpdateModuleAction:
			if child, ok := generator.buildUpdateChild(module, render, *a, role); ok {
				children[":id/edit"] = child
			}
		case actions.AddModuleAction:
			if child, ok := generator.buildAddChild(module, render, a, role); ok {
				children["add"] = child
			}
		case *actions.AddModuleAction:
			if child, ok := generator.buildAddChild(module, render, *a, role); ok {
				children["add"] = child
			}
		}
	}

	return children
}

func (generator *Generator) buildViewChild(module *BaseModule, render renderer.Universal, a actions.ViewModuleAction, role string) (RouteConfig, bool) {
	if !hasPermission(a, role) {
		return RouteConfig{}, false
	}

	return RouteConfig{
		Title:    a.Label,
		Renderer: viewRouteIdentity(render, viewActionPageType(a)),
		PageType: viewRoutePageType(render, viewActionPageType(a)),
		Query:    standardActionRouteQuery(module, a),
	}, true
}

func (generator *Generator) buildUpdateChild(module *BaseModule, render renderer.Universal, a actions.UpdateModuleAction, role string) (RouteConfig, bool) {
	if !hasPermission(a, role) {
		return RouteConfig{}, false
	}

	return RouteConfig{
		Title:    a.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query:    standardActionRouteQuery(module, a),
	}, true
}

func (generator *Generator) buildAddChild(module *BaseModule, render renderer.Universal, a actions.AddModuleAction, role string) (RouteConfig, bool) {
	if !hasPermission(a, role) {
		return RouteConfig{}, false
	}

	return RouteConfig{
		Title:    a.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query:    standardActionRouteQuery(module, a),
	}, true
}
