package integration

import (
	"net/http"
	"testing"

	module "github.com/darkrain/request-generator"
	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDefrecAppliesAddPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("")
	generator := module.NewGenerator(
		nil,
		*group,
		[]*module.BaseModule{{
			Name:   "restricted-items",
			Path:   "/admin",
			Render: renderer.Universal{Form: &renderer.FormPage{ID: "restricted-items"}},
			Actions: []actions.ModuleAction{actions.AddModuleAction{
				Permission: []actions.Role{"admin"},
			}},
		}},
		func(_ actions.ModuleAction, permissions []actions.Role) gin.HandlerFunc {
			return func(c *gin.Context) {
				if permissions[0] != "admin" {
					t.Fatalf("permission middleware received %v, want admin", permissions)
				}
				c.AbortWithStatus(http.StatusForbidden)
			}
		},
		nil,
	)
	generator.Run()

	w := executeRequest(engine, http.MethodGet, "/admin/restricted-items/defrec/", nil)
	require.Equal(t, http.StatusForbidden, w.Code)
}
