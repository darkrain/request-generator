package actions

import (
	"context"
	"fmt"
	"strings"

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

// AtomicRecord is both the atomic add response and the source for route
// interpolation. Fields such as nick must be explicitly returned by the
// operation when a follow-up route needs them.
type AtomicRecord struct {
	Value      int64         `json:"value"`
	PrimaryKey string        `json:"primary_key"`
	Fields     []AtomicField `json:"fields,omitempty"`
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

// InterpolateRoute resolves a renderer AfterSuccess.Route template from this
// record only. It intentionally has no secondary binding source.
func (record AtomicRecord) InterpolateRoute(template string) (string, error) {
	result := template
	for _, field := range record.Fields {
		if field.Value.String != nil {
			result = strings.ReplaceAll(result, "{"+field.Name+"}", *field.Value.String)
		}
		if field.Value.Int != nil {
			result = strings.ReplaceAll(result, "{"+field.Name+"}", fmt.Sprintf("%d", *field.Value.Int))
		}
	}
	if strings.Contains(result, "{") || strings.Contains(result, "}") {
		return "", fmt.Errorf("route has unresolved record fields")
	}
	return result, nil
}

// AtomicExecutor deliberately exposes only the operations needed by domain
// creation logic, not a driver transaction or connection.
type AtomicExecutor interface {
	Insert(context.Context, AtomicInsert) (AtomicRecord, error)
}

type AtomicAddOperation func(context.Context, AtomicExecutor, AtomicInput) (AtomicRecord, error)

type AtomicAddConfig struct {
	Operation AtomicAddOperation `json:"-"`
}
