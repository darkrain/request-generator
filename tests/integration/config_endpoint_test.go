package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/renderer"
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
		Render: renderer.Universal{
			List: &renderer.ListPage{
				ID: "users",
			},
		},
		Navigation: []module.NavigationEntry{
			{
				ActionName: "list",
				Title:      "Пользователи",
				Group:      "Управление",
				Order:      1,
				Icon:       "user",
				Show:       true,
				Path:       "/admin/users",
			},
			{
				ActionName: "add",
				Title:      "Добавить",
				Group:      "Управление",
				Order:      2,
				Icon:       "plus",
				Show:       true,
				Path:       "/admin/users/add",
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

	assert.Contains(t, response, "navigation")
	assert.Contains(t, response, "widgets")
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
		Navigation: []module.NavigationEntry{
			{
				ActionName: "list",
				Title:      "Restricted Module",
				Group:      "Admin",
				Order:      1,
				Show:       true,
				Path:       "/admin/restricted",
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
	assert.NotEmpty(t, adminResponse.Navigation, "Admin should see navigation")

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
	assert.Empty(t, managerResponse.Navigation, "Manager should not see admin-only navigation")
}

// TestConfigEndpoint_NavigationStructure — проверка структуры navigation
func TestConfigEndpoint_NavigationStructure(t *testing.T) {
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

	require.NotEmpty(t, response.Navigation, "Navigation should not be empty")

	for _, element := range response.Navigation {
		assert.Equal(t, "page", element.Target.Type, "Navigation item should default to page target")
		assert.NotEmpty(t, element.Path, "Page navigation item path should not be empty")
		assert.NotEmpty(t, element.Target.Query.Url, "Page navigation target query URL should not be empty")
		assert.NotEmpty(t, element.Target.Query.Method, "Page navigation target query method should not be empty")
		assert.NotNil(t, element.Target.Data, "Page navigation target data should not be nil")
	}
}

// TestConfigEndpoint_PageTargetStructure — проверка структуры page target
func TestConfigEndpoint_PageTargetStructure(t *testing.T) {
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

	require.NotEmpty(t, response.Navigation, "Navigation should not be empty")

	var usersEntry *module.ConfigNavigationEntry
	for i := range response.Navigation {
		if response.Navigation[i].Path == "/admin/users" {
			usersEntry = &response.Navigation[i]
			break
		}
	}
	require.NotNil(t, usersEntry, "Should have navigation entry for /admin/users")

	require.NotNil(t, usersEntry.Target.Renderer, "Users page target should expose renderer discovery")
	assert.Equal(t, renderer.Name, usersEntry.Target.Renderer.Name)
	assert.Equal(t, renderer.Version, usersEntry.Target.Renderer.Version)
	assert.Equal(t, renderer.PageTypeList, usersEntry.Target.PageType)
	assert.Equal(t, "/api/admin/users", usersEntry.Target.Query.Url)
	assert.Equal(t, "GET", usersEntry.Target.Query.Method)
	assert.NotNil(t, usersEntry.Target.Data, "Users page target data should not be nil")
}
