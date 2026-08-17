package fields

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

func TestResolveProjectionsUsesRequestScopedExpressionWithoutMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("scope", "caller")

	field := ModuleField{
		Column: pg.StringColumn("delivery_mode"),
		SelectExpressionFunc: func(ctx *gin.Context) pg.Projection {
			if ctx.Value("scope") == "caller" {
				return pg.String("private").AS("delivery_mode")
			}
			return pg.String("original").AS("delivery_mode")
		},
	}

	resolved := ResolveProjections(c, []ModuleField{field})
	if got := resolved[0].GetProjection(); got == field.Column {
		t.Fatal("resolved projection must use the request-scoped expression")
	}
	if field.SelectExpression != nil {
		t.Fatal("shared module field must not be mutated")
	}
}
