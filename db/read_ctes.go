package db

import (
	"fmt"

	pg "github.com/go-jet/jet/v2/postgres"
)

// WithReadCTEs scopes producer-declared read-only CTEs to one executor copy.
// Custom executors must explicitly support them; silently omitting a CTE can
// change the meaning of the producer's filters. Existing modules need no change.
func WithReadCTEs(executor DBExecutor, ctes []pg.CommonTableExpression) (DBExecutor, error) {
	if len(ctes) == 0 {
		return executor, nil
	}
	if scoped, ok := executor.(interface {
		WithReadCTEs([]pg.CommonTableExpression) DBExecutor
	}); ok {
		return scoped.WithReadCTEs(ctes), nil
	}
	return nil, fmt.Errorf("database executor does not support read CTEs")
}

// WithReadCTEs leaves the shared executor unchanged. Producers construct fresh
// Jet expressions for each request and must not mutate them after returning.
func (db *DB) WithReadCTEs(ctes []pg.CommonTableExpression) DBExecutor {
	scoped := *db
	scoped.readCTEs = append([]pg.CommonTableExpression(nil), ctes...)
	return &scoped
}

func (db *DB) readSQL(statement pg.SelectStatement) (string, []interface{}) {
	if len(db.readCTEs) == 0 {
		return statement.Sql()
	}
	return pg.WITH(db.readCTEs...)(statement).Sql()
}
