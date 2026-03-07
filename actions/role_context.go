package actions

import (
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/portalenergy/pe-request-generator/icontext"
)

const RoleAll Role = "all"

func GetRoleFromContext(c *gin.Context) Role {
	user, ok := icontext.GetUser(c.Request.Context())
	if !ok {
		return ""
	}
	return Role(user.Role)
}

// --- Fields ---

type RoleContext struct {
	Role    Role
	Columns []pg.Column
}

func ResolveRoleColumns(contexts []RoleContext, role Role) []pg.Column {
	for _, ctx := range contexts {
		if ctx.Role == role {
			return ctx.Columns
		}
	}
	for _, ctx := range contexts {
		if ctx.Role == RoleAll {
			return ctx.Columns
		}
	}
	return nil
}

// --- Where ---

type RoleWhere struct {
	Role  Role
	Where func(c *gin.Context) pg.BoolExpression
}

func ResolveRoleWhere(entries []RoleWhere, role Role) func(*gin.Context) pg.BoolExpression {
	for _, e := range entries {
		if e.Role == role {
			return e.Where
		}
	}
	for _, e := range entries {
		if e.Role == RoleAll {
			return e.Where
		}
	}
	return nil
}

// --- Join ---

type RoleJoin struct {
	Role Role
	Join []ModuleActionJoin
}

func ResolveRoleJoin(entries []RoleJoin, role Role) []ModuleActionJoin {
	for _, e := range entries {
		if e.Role == role {
			return e.Join
		}
	}
	for _, e := range entries {
		if e.Role == RoleAll {
			return e.Join
		}
	}
	return nil
}

// --- Hooks ---

type RoleHook struct {
	Role Role
	Hook func(c *gin.Context) error
}

type RoleAfterHook struct {
	Role Role
	Hook func(c *gin.Context)
}

func ResolveRoleHook(entries []RoleHook, role Role) func(*gin.Context) error {
	for _, e := range entries {
		if e.Role == role {
			return e.Hook
		}
	}
	for _, e := range entries {
		if e.Role == RoleAll {
			return e.Hook
		}
	}
	return nil
}

func ResolveRoleAfterHook(entries []RoleAfterHook, role Role) func(*gin.Context) {
	for _, e := range entries {
		if e.Role == role {
			return e.Hook
		}
	}
	for _, e := range entries {
		if e.Role == RoleAll {
			return e.Hook
		}
	}
	return nil
}
