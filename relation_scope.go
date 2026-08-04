package module

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

type resolvedRelationScope struct {
	relation ModuleRelation
	scope    RelationScope
}

func (generator *Generator) resolveRelationScope(c *gin.Context, module *BaseModule) (*resolvedRelationScope, int, error) {
	query := c.QueryMap("scope")
	relationName := strings.TrimSpace(query["relation"])
	scopeID := strings.TrimSpace(query["id"])
	if relationName == "" && scopeID == "" {
		return nil, 0, nil
	}
	if relationName == "" || scopeID == "" {
		return nil, http.StatusBadRequest, errors.New("scope[relation] and scope[id] are required together")
	}

	relation, ok := findModuleRelation(module, relationName)
	if !ok {
		return nil, http.StatusBadRequest, fmt.Errorf("unknown relation scope %q", relationName)
	}
	if relation.ScopeCheck == nil {
		return nil, http.StatusBadRequest, fmt.Errorf("relation scope %q is not enabled", relationName)
	}
	canonicalID, err := generator.canonicalRelationScopeID(c, relation, scopeID)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	scope := RelationScope{
		Relation: relationName,
		ID:       canonicalID,
	}
	if err = relation.ScopeCheck(c, scope); err != nil {
		return nil, http.StatusForbidden, err
	}

	return &resolvedRelationScope{
		relation: relation,
		scope:    scope,
	}, 0, nil
}

func findModuleRelation(module *BaseModule, name string) (ModuleRelation, bool) {
	for _, relation := range module.Relations {
		if relation.Name == name {
			return relation, true
		}
	}
	return ModuleRelation{}, false
}

func (generator *Generator) canonicalRelationScopeID(c *gin.Context, relation ModuleRelation, raw string) (interface{}, error) {
	targetModule := generator.findModule(relation.TargetModule)
	if targetModule == nil {
		return nil, fmt.Errorf("relation scope target module %q is not registered", relation.TargetModule)
	}
	targetField := targetModule.GetFieldByColumn(relation.TargetField)
	if targetField == nil {
		return nil, fmt.Errorf("relation target field %q is not declared in module fields", relation.TargetField.Name())
	}
	if targetField.Convert != nil {
		return targetField.Convert(c, raw)
	}

	switch targetField.Type {
	case fields.ModuleFieldTypeInt:
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid relation scope id %q for int field %q", raw, targetField.ColumnName())
		}
		return parsed, nil
	case fields.ModuleFieldTypeFloat:
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid relation scope id %q for float field %q", raw, targetField.ColumnName())
		}
		return parsed, nil
	case fields.ModuleFieldTypeString:
		return raw, nil
	default:
		return nil, fmt.Errorf("relation scope id field %q has unsupported type %q", targetField.ColumnName(), targetField.Type)
	}
}

func (generator *Generator) findModule(name string) *BaseModule {
	for _, candidate := range generator.Modules {
		if candidate.Name == name {
			return candidate
		}
	}
	return nil
}

func relationScopeWhere(module *BaseModule, scope *resolvedRelationScope) pg.BoolExpression {
	if scope == nil {
		return nil
	}
	tableRef := scope.relation.SourceField.TableName()
	if tableRef == "" {
		tableRef = module.Table.Alias()
	}
	if tableRef == "" {
		tableRef = module.Table.TableName()
	}
	return pg.RawBool(
		fmt.Sprintf(`%s."%s" = #scope_id`, tableRef, scope.relation.SourceField.Name()),
		pg.RawArgs{"#scope_id": scope.scope.ID},
	)
}

func rejectScopedSourceField(input map[string]interface{}, scope *resolvedRelationScope) error {
	if scope == nil {
		return nil
	}
	if _, ok := input[scope.relation.SourceField.Name()]; ok {
		return fmt.Errorf("field %q is controlled by relation scope", scope.relation.SourceField.Name())
	}
	return nil
}

func appendRelationScopeWhere(module *BaseModule, where pg.BoolExpression, scope *resolvedRelationScope) pg.BoolExpression {
	scopeWhere := relationScopeWhere(module, scope)
	if scopeWhere == nil {
		return where
	}
	if where == nil {
		return scopeWhere
	}
	return pg.AND(where, scopeWhere)
}

func injectRelationScopeInput(
	c *gin.Context,
	input map[string]interface{},
	mapInput map[string]interface{},
	realFields []fields.ModuleField,
	module *BaseModule,
	scope *resolvedRelationScope,
) ([]fields.ModuleField, error) {
	if scope == nil {
		return realFields, nil
	}

	field := module.GetFieldByColumn(scope.relation.SourceField)
	if field == nil {
		return nil, fmt.Errorf("relation source field %q is not declared in module fields", scope.relation.SourceField.Name())
	}

	// scope.ID has already been converted by the target field before ScopeCheck.
	// A source-field converter accepts a transport value and is not required to
	// be idempotent, so applying it again would corrupt typed relation IDs.
	value := scope.scope.ID

	input[field.ColumnName()] = value
	if mapInput != nil {
		mapInput[field.ColumnName()] = value
	}
	if !containsModuleField(realFields, field.Column) {
		realFields = append(realFields, *field)
	}
	return realFields, nil
}

func containsModuleField(items []fields.ModuleField, target pg.Column) bool {
	for _, item := range items {
		if fields.ContainsColumn([]pg.Column{item.Column}, target) {
			return true
		}
	}
	return false
}
