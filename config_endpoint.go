package module

import (
	"net/http"
	"sort"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/response"
	"github.com/gin-gonic/gin"
)

// ConfigResponse структурырует ответ эндпоинта /api/config
type ConfigResponse struct {
	LeftMenu []LeftMenuBlock            `json:"left_menu"`
	Routes   map[string]RouteConfig     `json:"routes"`
	Role     string                     `json:"role"`
}

// LeftMenuBlock представляет блок левого меню
type LeftMenuBlock struct {
	BlockTitle string   `json:"blockTitle"`
	Elements   []string `json:"elements"`
}

// RouteConfig конфигурирует маршрут
type RouteConfig struct {
	Title        string                 `json:"title"`
	MenuTitle    string                 `json:"menuTitle,omitempty"`
	Query        *RouteQuery            `json:"query,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Children     map[string]RouteConfig `json:"children,omitempty"`
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

		// Получаем пользователя из контекста (AuthMiddleware уже проверил токен)
		user, ok := icontext.GetUser(ctx)
		if !ok {
			response.ErrorResponse(l, c, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		role := user.Role

		// Собираем доступные модули
		availableModules := make(map[string]*BaseModule)
		moduleActions := make(map[string][]actions.ModuleAction)

		for _, module := range generator.Modules {
			moduleAvailable := false
			var accessibleActions []actions.ModuleAction

			for _, menuEntry := range module.MenuEntries {
				// Находим соответствующее действие по ActionName
				for _, action := range module.Actions {
					if string(action.Action()) == menuEntry.ActionName {
						// Проверяем permission
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

		// Формируем left_menu
		leftMenu := generator.buildLeftMenu(availableModules, moduleActions, role)

		// Формируем routes
		routes := generator.buildRoutes(availableModules, moduleActions, role)

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
	// Получаем permission из action
	var permissions []actions.Role

	switch a := action.(type) {
	case *actions.ListModuleAction:
		permissions = a.Permission
	case *actions.AddModuleAction:
		permissions = a.Permission
	case *actions.ViewModuleAction:
		permissions = a.Permission
	case *actions.UpdateModuleAction:
		permissions = a.Permission
	case *actions.DeleteModuleAction:
		permissions = a.Permission
	default:
		return false
	}

	// Если permissions пустой, доступ открыт всем
	if len(permissions) == 0 {
		return true
	}

	// Проверяем наличие роли
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
) []LeftMenuBlock {
	// Группируем MenuEntries по Group
	type menuEntryWithModule struct {
		entry  MenuEntry
		module *BaseModule
	}

	grouped := make(map[string][]menuEntryWithModule)
	groupOrder := make(map[string]int)

	for _, module := range availableModules {
		actionList := moduleActions[module.Name]
		actionMap := make(map[string]bool)
		for _, action := range actionList {
			actionMap[string(action.Action())] = true
		}

		for _, entry := range module.MenuEntries {
			// Проверяем, что действие доступно
			if !actionMap[entry.ActionName] {
				continue
			}

			group := entry.Group
			if group == "" {
				group = "default"
			}

			grouped[group] = append(grouped[group], menuEntryWithModule{
				entry:  entry,
				module: module,
			})

			// Запоминаем минимальный Order для группы
			if _, exists := groupOrder[group]; !exists {
				groupOrder[group] = entry.Order
			} else if entry.Order < groupOrder[group] {
				groupOrder[group] = entry.Order
			}
		}
	}

	// Сортируем группы по Order
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

	// Формируем LeftMenuBlock
	var result []LeftMenuBlock
	for _, group := range groups {
		entries := grouped[group.name]

		// Сортируем entries по Order
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].entry.Order < entries[j].entry.Order
		})

		// Получаем название группы из GroupTitles
		blockTitle := group.name
		if title, exists := generator.GroupTitles[group.name]; exists {
			blockTitle = title // Это i18n ключ
		}

		// Формируем элементы (ссылки)
		var elements []string
		for _, item := range entries {
			entry := item.entry
			if entry.CustomLink != "" {
				elements = append(elements, entry.CustomLink)
			} else {
				// Генерируем ссылку на основе модуля
				elements = append(elements, item.module.Path+"/"+item.module.Name)
			}
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
	availableModules map[string]*BaseModule,
	moduleActions map[string][]actions.ModuleAction,
	role string,
) map[string]RouteConfig {
	routes := make(map[string]RouteConfig)

	for _, module := range availableModules {
		actionList := moduleActions[module.Name]

		for _, action := range actionList {
			switch a := action.(type) {
			case *actions.ListModuleAction:
				// Генерируем route для списка
				routePath := module.Path + "/" + module.Name
				routeTitle := a.Label // i18n ключ

				viewAdapter := "list_table"
				if adapter, exists := generator.ViewAdapters["list"]; exists {
					viewAdapter = adapter
				}

				route := RouteConfig{
					Title:     routeTitle,
					MenuTitle: routeTitle,
					Query: &RouteQuery{
						Url:    "/api" + routePath,
						Method: "GET",
					},
					Data: map[string]interface{}{
						"view_adapter": viewAdapter,
					},
				}

				// Добавляем actions для маршрута
				route.Data["actions"] = generator.buildRouteActions(module, role)

				// Добавляем children (view, edit, add)
				route.Children = generator.buildRouteChildren(module, role)

				routes[routePath] = route

			case *actions.ViewModuleAction:
				// View обычно является children, но если отдельное действие
				routePath := module.Path + "/" + module.Name + "/:id"

				viewAdapter := "view"
				if adapter, exists := generator.ViewAdapters["view"]; exists {
					viewAdapter = adapter
				}

				route := RouteConfig{
					Title: a.Label,
					Query: &RouteQuery{
						Url:    "/api" + module.Path + "/" + module.Name + "/view/:bykey/:value",
						Method: "GET",
					},
					Data: map[string]interface{}{
						"view_adapter": viewAdapter,
					},
				}

				routes[routePath] = route

			case *actions.AddModuleAction:
				routePath := module.Path + "/" + module.Name + "/add"

				viewAdapter := "add"
				if adapter, exists := generator.ViewAdapters["add"]; exists {
					viewAdapter = adapter
				}

				route := RouteConfig{
					Title: a.Label,
					Query: &RouteQuery{
						Url:    "/api" + module.Path + "/" + module.Name,
						Method: "PUT",
					},
					Data: map[string]interface{}{
						"view_adapter": viewAdapter,
					},
				}

				routes[routePath] = route
			}
		}
	}

	return routes
}

// buildRouteActions формирует actions для маршрута
func (generator *Generator) buildRouteActions(module *BaseModule, role string) []map[string]interface{} {
	var result []map[string]interface{}

	for _, entry := range module.MenuEntries {
		// Находим соответствующее действие
		for _, action := range module.Actions {
			if string(action.Action()) != entry.ActionName {
				continue
			}
			if !hasPermission(action, role) {
				continue
			}

			actionMap := map[string]interface{}{
				"title": entry.Title, // i18n ключ
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
func (generator *Generator) buildRouteChildren(module *BaseModule, role string) map[string]RouteConfig {
	children := make(map[string]RouteConfig)

	for _, action := range module.Actions {
		switch a := action.(type) {
		case *actions.ViewModuleAction:
			if !hasPermission(a, role) {
				continue
			}

			viewAdapter := "view"
			if adapter, exists := generator.ViewAdapters["view"]; exists {
				viewAdapter = adapter
			}

			children[":id"] = RouteConfig{
				Title: a.Label,
				Query: &RouteQuery{
					Url:    "/api" + module.Path + "/" + module.Name + "/view/:bykey/:value",
					Method: "GET",
				},
				Data: map[string]interface{}{
					"view_adapter": viewAdapter,
				},
			}

		case *actions.UpdateModuleAction:
			if !hasPermission(a, role) {
				continue
			}

			editAdapter := "edit"
			if adapter, exists := generator.ViewAdapters["edit"]; exists {
				editAdapter = adapter
			}

			children[":id/edit"] = RouteConfig{
				Title: a.Label,
				Query: &RouteQuery{
					Url:    "/api" + module.Path + "/" + module.Name + "/:bykey/:value",
					Method: "POST",
				},
				Data: map[string]interface{}{
					"view_adapter": editAdapter,
				},
			}

		case *actions.AddModuleAction:
			if !hasPermission(a, role) {
				continue
			}

			addAdapter := "add"
			if adapter, exists := generator.ViewAdapters["add"]; exists {
				addAdapter = adapter
			}

			children["add"] = RouteConfig{
				Title: a.Label,
				Query: &RouteQuery{
					Url:    "/api" + module.Path + "/" + module.Name,
					Method: "PUT",
				},
				Data: map[string]interface{}{
					"view_adapter": addAdapter,
				},
			}
		}
	}

	return children
}
