package module

import (
	"net/http"
	"sort"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/locale"
	"github.com/darkrain/request-generator/renderer"
	"github.com/darkrain/request-generator/response"
	"github.com/gin-gonic/gin"
)

// ConfigResponse структурирует ответ эндпоинта /api/config
type ConfigResponse struct {
	LeftMenu []LeftMenuBlock        `json:"left_menu"`
	Routes   map[string]RouteConfig `json:"routes"`
	Role     string                 `json:"role"`
}

// LeftMenuItem представляет один пункт меню
type LeftMenuItem struct {
	URL   string                 `json:"url"`
	Title string                 `json:"title"`
	Icon  string                 `json:"icon,omitempty"`
	Query map[string]interface{} `json:"query,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// LeftMenuBlock представляет блок левого меню
type LeftMenuBlock struct {
	BlockTitle string         `json:"blockTitle"`
	Elements   []LeftMenuItem `json:"elements"`
}

// RouteConfig конфигурирует маршрут
type RouteConfig struct {
	Title     string                 `json:"title"`
	MenuTitle string                 `json:"menuTitle,omitempty"`
	Renderer  *renderer.Identity     `json:"renderer,omitempty"`
	PageType  renderer.PageType      `json:"page_type,omitempty"`
	Query     *RouteQuery            `json:"query,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Children  map[string]RouteConfig `json:"children,omitempty"`
}

// RouteQuery описывает параметры запроса для маршрута
type RouteQuery struct {
	Url    string `json:"url"`
	Method string `json:"method"`
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

		availableModules := make(map[string]*BaseModule)
		moduleActions := make(map[string][]actions.ModuleAction)

		for _, module := range generator.Modules {
			moduleAvailable := false
			var accessibleActions []actions.ModuleAction

			for _, menuEntry := range module.MenuEntries {
				if !menuEntry.Show {
					continue
				}
				for _, action := range module.Actions {
					if string(action.Action()) == menuEntry.ActionName {
						if hasPermission(action, role) {
							moduleAvailable = true
							accessibleActions = append(accessibleActions, action)
						}
						break
					}
				}
			}

			if moduleAvailable {
				availableModules[module.Name] = module
				moduleActions[module.Name] = accessibleActions
			}
		}

		leftMenu := generator.buildLeftMenu(availableModules, moduleActions, role, lang)
		routes, err := generator.buildRoutes(c, availableModules, moduleActions, role)
		if err != nil {
			response.ErrorResponse(l, c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		config := ConfigResponse{
			LeftMenu: leftMenu,
			Routes:   routes,
			Role:     role,
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

// buildLeftMenu формирует левое меню с группировкой по Group
func (generator *Generator) buildLeftMenu(
	availableModules map[string]*BaseModule,
	moduleActions map[string][]actions.ModuleAction,
	role string,
	lang locale.Lang,
) []LeftMenuBlock {
	type menuEntryWithModule struct {
		entry  MenuEntry
		module *BaseModule
	}

	grouped := make(map[string][]menuEntryWithModule)
	groupOrder := make(map[string]int)

	for _, module := range availableModules {
		actionMap := make(map[string]bool)
		for _, action := range moduleActions[module.Name] {
			actionMap[string(action.Action())] = true
		}

		for _, entry := range module.MenuEntries {
			if !entry.Show {
				continue
			}
			if !actionMap[entry.ActionName] {
				continue
			}

			// Role-specific visibility
			if len(entry.Roles) > 0 {
				allowed := false
				for _, r := range entry.Roles {
					if string(r) == role || r == actions.RoleAll {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}

			group := entry.Group
			if group == "" {
				group = "default"
			}

			grouped[group] = append(grouped[group], menuEntryWithModule{
				entry:  entry,
				module: module,
			})

			if _, exists := groupOrder[group]; !exists {
				groupOrder[group] = entry.Order
			} else if entry.Order < groupOrder[group] {
				groupOrder[group] = entry.Order
			}
		}
	}

	type groupInfo struct {
		name  string
		order int
	}
	var groups []groupInfo
	for name, order := range groupOrder {
		groups = append(groups, groupInfo{name: name, order: order})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].order < groups[j].order
	})

	var result []LeftMenuBlock
	for _, group := range groups {
		entries := grouped[group.name]

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].entry.Order < entries[j].entry.Order
		})

		blockTitle := group.name
		if title, exists := generator.GroupTitles[group.name]; exists {
			blockTitle = title
		}

		var elements []LeftMenuItem
		for _, item := range entries {
			entry := item.entry
			menuItem := LeftMenuItem{
				Title: generator.Translate(lang, entry.Title),
				Icon:  entry.Icon,
			}

			if entry.CustomLink != "" {
				menuItem.URL = entry.CustomLink
			} else {
				menuItem.URL = item.module.Path + "/" + item.module.Name
			}
			if entry.CustomQuery != nil {
				menuItem.Query = entry.CustomQuery
			}
			if entry.CustomData != nil {
				menuItem.Data = entry.CustomData
			}

			elements = append(elements, menuItem)
		}

		if len(elements) > 0 {
			result = append(result, LeftMenuBlock{
				BlockTitle: blockTitle,
				Elements:   elements,
			})
		}
	}

	return result
}

// buildRoutes формирует маршруты с view-адаптерами, actions, children
func (generator *Generator) buildRoutes(
	c *gin.Context,
	availableModules map[string]*BaseModule,
	moduleActions map[string][]actions.ModuleAction,
	role string,
) (map[string]RouteConfig, error) {
	routes := make(map[string]RouteConfig)

	for _, module := range availableModules {
		actionList := moduleActions[module.Name]
		render, err := module.RenderFor(c)
		if err != nil {
			return nil, err
		}

		for _, action := range actionList {
			switch a := action.(type) {
			case actions.ListModuleAction:
				routes[module.Path+"/"+module.Name] = generator.buildListRoute(module, render, a, role)
			case *actions.ListModuleAction:
				routes[module.Path+"/"+module.Name] = generator.buildListRoute(module, render, *a, role)
			case actions.ViewModuleAction:
				routes[module.Path+"/"+module.Name+"/:id"] = generator.buildViewRoute(module, render, a)
			case *actions.ViewModuleAction:
				routes[module.Path+"/"+module.Name+"/:id"] = generator.buildViewRoute(module, render, *a)
			case actions.AddModuleAction:
				routes[module.Path+"/"+module.Name+"/add"] = generator.buildAddRoute(module, render, a)
			case *actions.AddModuleAction:
				routes[module.Path+"/"+module.Name+"/add"] = generator.buildAddRoute(module, render, *a)
			}
		}
	}

	return routes, nil
}

func (generator *Generator) buildListRoute(module *BaseModule, render renderer.Universal, action actions.ListModuleAction, role string) RouteConfig {
	routePath := module.Path + "/" + module.Name

	viewAdapter := "list_table"
	if adapter, exists := generator.ViewAdapters["list"]; exists {
		viewAdapter = adapter
	}

	route := RouteConfig{
		Title:     action.Label,
		MenuTitle: action.Label,
		Renderer:  render.ListIdentity(),
		PageType:  render.ListRoutePageType(),
		Query: &RouteQuery{
			Url:    "/api" + routePath,
			Method: "GET",
		},
		Data: map[string]interface{}{
			"view_adapter": viewAdapter,
		},
	}

	route.Data["actions"] = generator.buildRouteActions(module, role)
	route.Children = generator.buildRouteChildren(module, render, role)

	return route
}

func (generator *Generator) buildViewRoute(module *BaseModule, render renderer.Universal, action actions.ViewModuleAction) RouteConfig {
	viewAdapter := "view"
	if adapter, exists := generator.ViewAdapters["view"]; exists {
		viewAdapter = adapter
	}

	return RouteConfig{
		Title:    action.Label,
		Renderer: render.RecordIdentity(),
		PageType: render.RecordRoutePageType(),
		Query: &RouteQuery{
			Url:    "/api" + module.Path + "/" + module.Name + "/view/:bykey/:value",
			Method: "GET",
		},
		Data: map[string]interface{}{
			"view_adapter": viewAdapter,
		},
	}
}

func (generator *Generator) buildAddRoute(module *BaseModule, render renderer.Universal, action actions.AddModuleAction) RouteConfig {
	viewAdapter := "add"
	if adapter, exists := generator.ViewAdapters["add"]; exists {
		viewAdapter = adapter
	}

	return RouteConfig{
		Title:    action.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query: &RouteQuery{
			Url:    "/api" + module.Path + "/" + module.Name,
			Method: "PUT",
		},
		Data: map[string]interface{}{
			"view_adapter": viewAdapter,
		},
	}
}

// buildRouteActions формирует actions для маршрута
func (generator *Generator) buildRouteActions(module *BaseModule, role string) []map[string]interface{} {
	var result []map[string]interface{}

	for _, entry := range module.MenuEntries {
		for _, action := range module.Actions {
			if string(action.Action()) != entry.ActionName {
				continue
			}
			if !hasPermission(action, role) {
				continue
			}

			actionMap := map[string]interface{}{
				"title": entry.Title,
				"type":  entry.ActionName,
				"icon":  entry.Icon,
				"show":  entry.Show,
			}

			if entry.CustomQuery != nil {
				actionMap["query"] = entry.CustomQuery
			}
			if entry.CustomData != nil {
				actionMap["data"] = entry.CustomData
			}

			result = append(result, actionMap)
			break
		}
	}

	return result
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

	viewAdapter := "view"
	if adapter, exists := generator.ViewAdapters["view"]; exists {
		viewAdapter = adapter
	}

	return RouteConfig{
		Title:    a.Label,
		Renderer: render.RecordIdentity(),
		PageType: render.RecordRoutePageType(),
		Query: &RouteQuery{
			Url:    "/api" + module.Path + "/" + module.Name + "/view/:bykey/:value",
			Method: "GET",
		},
		Data: map[string]interface{}{
			"view_adapter": viewAdapter,
		},
	}, true
}

func (generator *Generator) buildUpdateChild(module *BaseModule, render renderer.Universal, a actions.UpdateModuleAction, role string) (RouteConfig, bool) {
	if !hasPermission(a, role) {
		return RouteConfig{}, false
	}

	editAdapter := "edit"
	if adapter, exists := generator.ViewAdapters["edit"]; exists {
		editAdapter = adapter
	}

	return RouteConfig{
		Title:    a.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query: &RouteQuery{
			Url:    "/api" + module.Path + "/" + module.Name + "/:bykey/:value",
			Method: "POST",
		},
		Data: map[string]interface{}{
			"view_adapter": editAdapter,
		},
	}, true
}

func (generator *Generator) buildAddChild(module *BaseModule, render renderer.Universal, a actions.AddModuleAction, role string) (RouteConfig, bool) {
	if !hasPermission(a, role) {
		return RouteConfig{}, false
	}

	addAdapter := "add"
	if adapter, exists := generator.ViewAdapters["add"]; exists {
		addAdapter = adapter
	}

	return RouteConfig{
		Title:    a.Label,
		Renderer: render.FormIdentity(),
		PageType: render.FormRoutePageType(),
		Query: &RouteQuery{
			Url:    "/api" + module.Path + "/" + module.Name,
			Method: "PUT",
		},
		Data: map[string]interface{}{
			"view_adapter": addAdapter,
		},
	}, true
}
