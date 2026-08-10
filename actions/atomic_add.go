package actions

import (
	"context"
	"encoding/json"
	"fmt"

	pg "github.com/go-jet/jet/v2/postgres"
)

// AddMode selects the persistence path for an add action.
type AddMode string

const (
	AddModeStandard AddMode = "standard"
	AddModeAtomic   AddMode = "atomic"
)

// AtomicValue is the explicit value union passed to an atomic operation.
// It avoids exposing the request's untyped JSON map to domain code.
type AtomicValue struct {
	String  *string  `json:"string,omitempty"`
	Int     *int64   `json:"int,omitempty"`
	Float   *float64 `json:"float,omitempty"`
	Bool    *bool    `json:"bool,omitempty"`
	Strings []string `json:"strings,omitempty"`
	Ints    []int64  `json:"ints,omitempty"`
	JSON    []byte   `json:"json,omitempty"`
}

func AtomicString(value string) AtomicValue { return AtomicValue{String: &value} }
func AtomicInt(value int64) AtomicValue     { return AtomicValue{Int: &value} }
func AtomicFloat(value float64) AtomicValue { return AtomicValue{Float: &value} }
func AtomicBool(value bool) AtomicValue     { return AtomicValue{Bool: &value} }

func (value AtomicValue) Validate() error {
	variants := 0
	if value.String != nil {
		variants++
	}
	if value.Int != nil {
		variants++
	}
	if value.Float != nil {
		variants++
	}
	if value.Bool != nil {
		variants++
	}
	if value.Strings != nil {
		variants++
	}
	if value.Ints != nil {
		variants++
	}
	if value.JSON != nil {
		variants++
		if !json.Valid(value.JSON) {
			return fmt.Errorf("atomic JSON value is invalid")
		}
	}
	if variants != 1 {
		return fmt.Errorf("atomic value must contain exactly one variant")
	}
	return nil
}

func (value AtomicValue) Interface() interface{} {
	switch {
	case value.String != nil:
		return *value.String
	case value.Int != nil:
		return *value.Int
	case value.Float != nil:
		return *value.Float
	case value.Bool != nil:
		return *value.Bool
	case value.Strings != nil:
		return value.Strings
	case value.Ints != nil:
		return value.Ints
	case value.JSON != nil:
		return value.JSON
	default:
		return nil
	}
}

type AtomicField struct {
	Name  string      `json:"name"`
	Value AtomicValue `json:"value"`
}

// AtomicInput contains normalized and validated add fields.
type AtomicInput struct {
	Fields []AtomicField `json:"fields"`
}

func (input AtomicInput) Field(name string) (AtomicValue, bool) {
	for _, field := range input.Fields {
		if field.Name == name {
			return field.Value, true
		}
	}
	return AtomicValue{}, false
}

func (input AtomicInput) String(name string) (string, bool) {
	value, ok := input.Field(name)
	if !ok || value.String == nil {
		return "", false
	}
	return *value.String, true
}

func (input AtomicInput) Int(name string) (int64, bool) {
	value, ok := input.Field(name)
	if !ok || value.Int == nil {
		return 0, false
	}
	return *value.Int, true
}

func (input AtomicInput) RequireString(name string) (string, error) {
	value, ok := input.String(name)
	if !ok {
		return "", fmt.Errorf("atomic input field %q is not a string", name)
	}
	return value, nil
}

// AtomicInsert is a single INSERT statement executed within the generator-owned
// transaction. The operation never receives the underlying transaction type.
type AtomicInsert struct {
	Table      pg.Table            `json:"-"`
	PrimaryKey pg.Column           `json:"-"`
	Fields     []AtomicInsertField `json:"-"`
}

type AtomicInsertField struct {
	Column pg.Column   `json:"-"`
	Value  AtomicValue `json:"value"`
}

// AtomicValueKind declares the SQL result shape requested by AtomicSelect.
// Keeping it explicit avoids leaking driver scan values into domain operations.
type AtomicValueKind string

const (
	AtomicValueKindString  AtomicValueKind = "string"
	AtomicValueKindInt     AtomicValueKind = "int"
	AtomicValueKindFloat   AtomicValueKind = "float"
	AtomicValueKindBool    AtomicValueKind = "bool"
	AtomicValueKindStrings AtomicValueKind = "strings"
	AtomicValueKindInts    AtomicValueKind = "ints"
)

type AtomicSelectField struct {
	Name   string          `json:"name"`
	Column pg.Column       `json:"-"`
	Kind   AtomicValueKind `json:"kind"`
}

// AtomicSelect describes a typed read executed inside the same transaction as
// the following atomic writes.
type AtomicSelect struct {
	Table  pg.Table            `json:"-"`
	Fields []AtomicSelectField `json:"-"`
	Where  pg.BoolExpression   `json:"-"`
}

// AtomicRecord is both the atomic add response and the source for route
// interpolation. Fields are serialized at the top level of the response, so a
// renderer can resolve routes such as /profiles/{nick} from the HTTP result.
type AtomicRecord struct {
	Value      int64         `json:"value"`
	PrimaryKey string        `json:"primary_key"`
	Fields     []AtomicField `json:"fields,omitempty"`
}

func (record AtomicRecord) Validate() error {
	seen := map[string]struct{}{"value": {}, "primary_key": {}}
	for _, field := range record.Fields {
		if field.Name == "" {
			return fmt.Errorf("atomic record field name is required")
		}
		if _, exists := seen[field.Name]; exists {
			return fmt.Errorf("atomic record field %q is duplicated or reserved", field.Name)
		}
		seen[field.Name] = struct{}{}
		if err := field.Value.Validate(); err != nil {
			return fmt.Errorf("atomic record field %q: %w", field.Name, err)
		}
	}
	return nil
}

func (record AtomicRecord) MarshalJSON() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"value":       record.Value,
		"primary_key": record.PrimaryKey,
	}
	for _, field := range record.Fields {
		result[field.Name] = atomicResponseValue(field.Value)
	}
	return json.Marshal(result)
}

func atomicResponseValue(value AtomicValue) interface{} {
	if value.JSON != nil {
		return json.RawMessage(value.JSON)
	}
	return value.Interface()
}

func (record AtomicRecord) Field(name string) (AtomicValue, bool) {
	for _, field := range record.Fields {
		if field.Name == name {
			return field.Value, true
		}
	}
	return AtomicValue{}, false
}

func (record AtomicRecord) String(name string) (string, bool) {
	value, ok := record.Field(name)
	if !ok || value.String == nil {
		return "", false
	}
	return *value.String, true
}

// AtomicExecutor deliberately exposes only the operations needed by domain
// creation logic, not a driver transaction or connection.
type AtomicExecutor interface {
	Insert(context.Context, AtomicInsert) (AtomicRecord, error)
	SelectOne(context.Context, AtomicSelect) (AtomicRecord, error)
}

type AtomicAddOperation func(context.Context, AtomicExecutor, AtomicInput) (AtomicRecord, error)

type AtomicAddConfig struct {
	Operation AtomicAddOperation `json:"-"`
}
