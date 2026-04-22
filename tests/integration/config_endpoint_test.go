package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/icontext"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter создаёт тестовый роутер с мокированным AuthMiddleware
func setupTestRouter(authMiddleware func(actions.ModuleAction) gin.HandlerFunc) (*module.Generator, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	// Создаём простой тестовый модуль
	testModule := &module.BaseModule{
		Name: "users",
		Path: "/admin",
		MenuEntries: []module.MenuEntry{
			{
				ActionName: "list",
				Title:      "Пользователи",
				Group:      "Управление",
				Order:      1,
				Icon:       "user",
				Show:       true,
			},
			{
				ActionName: "add",
				Title:      "Добавить",
				Group:      "Управление",
				Order:      2,
				Icon:       "plus",
				Show:       true,
			},
		},
		Actions: []actions.ModuleAction{
			&actions.ListModuleAction{
				Permission: []actions.Role{"admin", "manager"},
				Label:      "Список пользователей",
			},
			&actions.AddModuleAction{
				Permission: []actions.Role{"admin"},
				Label:      "Добавить пользователя",
			},
			&actions.ViewModuleAction{
				Permission: []actions.Role{"admin", "manager", actions.RoleAll},
				Label:      "Просмотр пользователя",
			},
			&actions.UpdateModuleAction{
				Permission: []actions.Role{"admin"},
				Label:      "Обновить пользователя",
			},
		},
	}

	modules := []*module.BaseModule{testModule}

	// Мок PermissionMiddleware
	permissionMiddleware := func(action actions.ModuleAction, permissions []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	g := engine.Group("") // корневой роутер, т.к. Run() сам создаёт группу /api
	generator := module.NewGenerator(
		nil, // db функция не нужна для теста конфиг эндпоинта
		*g,
		modules,
		permissionMiddleware,
		authMiddleware,
	)

	// Регистрируем маршруты (это создаст конфиг эндпоинт)
	generator.Run()

	return generator, engine
}

// createMockAuthMiddleware создаёт мок AuthMiddleware, который устанавливает пользователя в контекст
func createMockAuthMiddleware(user *icontext.UserInfo) func(actions.ModuleAction) gin.HandlerFunc {
	return func(action actions.ModuleAction) gin.HandlerFunc {
		return func(c *gin.Context) {
			if user == nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			ctx := icontext.SetUser(c.Request.Context(), user)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}
	}
}

func executeRequest(engine *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// TestConfigEndpoint_ValidToken — успешный запрос с валидным токеном
func TestConfigEndpoint_ValidToken(t *testing.T) {
	// Мокированный пользователь с ролью admin
	mockUser := &icontext.UserInfo{
		ID:   1,
		Role: "admin",
	}

	_, engine := setupTestRouter(createMockAuthMiddleware(mockUser))

	w := executeRequest(engine, "GET", "/api/config", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что ответ валидный JSON с правильной структурой
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "left_menu")
	assert.Contains(t, response, "routes")
	assert.Contains(t, response, "role")
	assert.Equal(t, "admin", response["role"])
}

// TestConfigEndpoint_InvalidToken — 401 при невалидном токене
func TestConfigEndpoint_InvalidToken(t *testing.T) {
	// Мокированный middleware, который всегда возвращает 401
	invalidAuthMiddleware := func(action actions.ModuleAction) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}

	_, engine := setupTestRouter(invalidAuthMiddleware)

	w := executeRequest(engine, "GET", "/api/config", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestConfigEndpoint_MissingToken — 401 при отсутствии токена
func TestConfigEndpoint_MissingToken(t *testing.T) {
	// Мокированный middleware, который проверяет наличие токена
	noTokenMiddleware := func(action actions.ModuleAction) gin.HandlerFunc {
		return func(c *gin.Context) {
			// Имитируем отсутствие токена
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}

	_, engine := setupTestRouter(noTokenMiddleware)

	w := executeRequest(engine, "GET", "/api/config", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestConfigEndpoint_RoleFiltering — проверка что модули фильтруются по роли
func TestConfigEndpoint_RoleFiltering(t *testing.T) {
	// Создаём отдельный роутер для теста с модулем
	gin.SetMode(gin.TestMode)

	testModule := &module.BaseModule{
		Name: "restricted",
		Path: "/admin",
		MenuEntries: []module.MenuEntry{
			{
				ActionName: "list",
				Title:      "Restricted Module",
				Group:      "Admin",
				Order:      1,
			},
		},
		Actions: []actions.ModuleAction{
			&actions.ListModuleAction{
				Permission: []actions.Role{"admin"}, // Только админ
				Label:      "Restricted List",
			},
		},
	}

	modules := []*module.BaseModule{testModule}

	permissionMiddleware := func(action actions.ModuleAction, permissions []actions.Role) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	mockAdmin := &icontext.UserInfo{ID: 1, Role: "admin"}
	mockManager := &icontext.UserInfo{ID: 2, Role: "manager"}

	// Тестируем доступ для админа (должен видеть модуль)
	adminEngine := gin.New()
	authMiddlewareAdmin := createMockAuthMiddleware(mockAdmin)
	ag := adminEngine.Group("")
	adminGenerator := module.NewGenerator(nil, *ag, modules, permissionMiddleware, authMiddlewareAdmin)
	adminGenerator.Run()

	w := executeRequest(adminEngine, "GET", "/api/config", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var adminResponse module.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &adminResponse)
	require.NoError(t, err)
	assert.NotEmpty(t, adminResponse.Routes, "Admin should see routes")

	// Тестируем доступ для менеджера (НЕ должен видеть модуль)
	managerEngine := gin.New()
	authMiddlewareManager := createMockAuthMiddleware(mockManager)
	mg := managerEngine.Group("")
	managerGenerator := module.NewGenerator(nil, *mg, modules, permissionMiddleware, authMiddlewareManager)
	managerGenerator.Run()

	w = executeRequest(managerEngine, "GET", "/api/config", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var managerResponse module.ConfigResponse
	err = json.Unmarshal(w.Body.Bytes(), &managerResponse)
	require.NoError(t, err)
	assert.Empty(t, managerResponse.Routes, "Manager should not see admin-only routes")
}

// TestConfigEndpoint_LeftMenuStructure — проверка структуры left_menu
func TestConfigEndpoint_LeftMenuStructure(t *testing.T) {
	mockUser := &icontext.UserInfo{
		ID:   1,
		Role: "admin",
	}

	_, engine := setupTestRouter(createMockAuthMiddleware(mockUser))

	w := executeRequest(engine, "GET", "/api/config", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response module.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	require.NotEmpty(t, response.LeftMenu, "Left menu should not be empty")

	// Проверяем структуру первого блока меню
	leftMenuBlock := response.LeftMenu[0]
	assert.NotEmpty(t, leftMenuBlock.BlockTitle, "Block title should not be empty")
	assert.NotEmpty(t, leftMenuBlock.Elements, "Block should have elements")

	// Проверяем, что элементы являются строками (ссылками)
	for _, element := range leftMenuBlock.Elements {
		assert.NotEmpty(t, element, "Menu element should not be empty")
	}
}

// TestConfigEndpoint_RoutesStructure — проверка структуры routes
func TestConfigEndpoint_RoutesStructure(t *testing.T) {
	mockUser := &icontext.UserInfo{
		ID:   1,
		Role: "admin",
	}

	_, engine := setupTestRouter(createMockAuthMiddleware(mockUser))

	w := executeRequest(engine, "GET", "/api/config", nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var response module.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	require.NotEmpty(t, response.Routes, "Routes should not be empty")

	// Проверяем структуру маршрутов
	for routePath, routeConfig := range response.Routes {
		assert.NotEmpty(t, routePath, "Route path should not be empty")
		assert.NotEmpty(t, routeConfig.Title, "Route title should not be empty")

		// Проверяем наличие query для маршрута
		if routeConfig.Query != nil {
			assert.NotEmpty(t, routeConfig.Query.Url, "Query URL should not be empty")
			assert.NotEmpty(t, routeConfig.Query.Method, "Query method should not be empty")
		}

		// Проверяем наличие data
		assert.NotNil(t, routeConfig.Data, "Route data should not be nil")
	}

	// Проверяем, что есть хотя бы один маршрут для нашего тестового модуля
	foundUsersRoute := false
	for routePath := range response.Routes {
		if routePath == "/admin/users" {
			foundUsersRoute = true
			break
		}
	}
	assert.True(t, foundUsersRoute, "Should have route for /admin/users")
}
