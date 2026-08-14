package module

import (
	"context"
	"fmt"
	"strconv"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/gin-gonic/gin"
	pg "github.com/go-jet/jet/v2/postgres"
)

func atomicUpdateSelectorFromRoute(c *gin.Context, module *BaseModule, by []pg.Column, byKey, rawValue string) (actions.AtomicSelector, error) {
	var column pg.Column
	for _, candidate := range by {
		if candidate.Name() == byKey {
			column = candidate
			break
		}
	}
	if column == nil {
		return actions.AtomicSelector{}, fmt.Errorf("atomic update selector field %q is not declared", byKey)
	}
	field := module.GetFieldByColumn(column)
	if field == nil {
		return actions.AtomicSelector{}, fmt.Errorf("atomic update selector field %q is not declared in module fields", byKey)
	}

	value, err := atomicRouteSelectorValue(c, *field, rawValue)
	if err != nil {
		return actions.AtomicSelector{}, fmt.Errorf("atomic update selector %q: %w", byKey, err)
	}
	atomicValue, err := atomicValueFromField(*field, value)
	if err != nil {
		return actions.AtomicSelector{}, fmt.Errorf("atomic update selector %q: %w", byKey, err)
	}
	selector := actions.AtomicSelector{ByKey: byKey, Value: atomicValue}
	if err := selector.Validate(); err != nil {
		return actions.AtomicSelector{}, err
	}
	return selector, nil
}

func atomicRouteSelectorValue(c *gin.Context, field fields.ModuleField, rawValue string) (interface{}, error) {
	if field.Convert != nil {
		return field.Convert(c, rawValue)
	}
	switch field.Type {
	case fields.ModuleFieldTypeString:
		return rawValue, nil
	case fields.ModuleFieldTypeInt:
		value, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected integer")
		}
		return value, nil
	case fields.ModuleFieldTypeFloat:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return nil, fmt.Errorf("expected number")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("field type %q is not supported", field.Type)
	}
}

func atomicPrimaryKeyKind(module *BaseModule) (actions.AtomicValueKind, error) {
	field := module.GetFieldByColumn(module.PrimaryKey)
	if field == nil {
		return "", fmt.Errorf("atomic update primary key %q is not declared in module fields", module.PrimaryKey.Name())
	}
	if field.Type != fields.ModuleFieldTypeInt {
		return "", fmt.Errorf("atomic update primary key %q must have int type", module.PrimaryKey.Name())
	}
	return actions.AtomicValueKindInt, nil
}

func atomicUpdateSubject(ctx context.Context, executor actions.AtomicExecutor, module *BaseModule, where pg.BoolExpression) (int64, error) {
	kind, err := atomicPrimaryKeyKind(module)
	if err != nil {
		return 0, err
	}
	record, err := executor.SelectOne(ctx, actions.AtomicSelect{
		Table:  module.Table,
		Fields: []actions.AtomicSelectField{{Name: "record_id", Column: module.PrimaryKey, Kind: kind}},
		Where:  where,
	})
	if err != nil {
		return 0, err
	}
	recordID, ok := record.Int("record_id")
	if !ok || recordID <= 0 {
		return 0, fmt.Errorf("atomic update target is unavailable")
	}
	return recordID, nil
}
