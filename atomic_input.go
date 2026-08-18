package module

import (
	"fmt"
	"math"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
)

func atomicInputFromFields(moduleFields []fields.ModuleField, input map[string]interface{}) (actions.AtomicInput, error) {
	result := actions.AtomicInput{Fields: make([]actions.AtomicField, 0, len(input))}
	for _, field := range moduleFields {
		name := field.Name()
		if !field.Translatable {
			name = field.ColumnName()
		}
		value, ok := input[name]
		if !ok {
			continue
		}
		atomicValue, err := atomicValueFromField(field, value)
		if err != nil {
			return actions.AtomicInput{}, fmt.Errorf("atomic input field %q: %w", name, err)
		}
		result.Fields = append(result.Fields, actions.AtomicField{Name: name, Value: atomicValue})
	}
	return result, nil
}

func atomicValueFromField(field fields.ModuleField, value interface{}) (actions.AtomicValue, error) {
	if value == nil {
		return actions.AtomicValue{}, nil
	}
	switch field.Type {
	case fields.ModuleFieldTypeString:
		value, ok := value.(string)
		if !ok {
			return actions.AtomicValue{}, fmt.Errorf("expected string")
		}
		return actions.AtomicString(value), nil
	case fields.ModuleFieldTypeInt:
		switch value := value.(type) {
		case int:
			return actions.AtomicInt(int64(value)), nil
		case int64:
			return actions.AtomicInt(value), nil
		case float64:
			if math.Trunc(value) != value {
				return actions.AtomicValue{}, fmt.Errorf("expected integer")
			}
			return actions.AtomicInt(int64(value)), nil
		default:
			return actions.AtomicValue{}, fmt.Errorf("expected integer")
		}
	case fields.ModuleFieldTypeFloat:
		value, ok := value.(float64)
		if !ok {
			return actions.AtomicValue{}, fmt.Errorf("expected number")
		}
		return actions.AtomicFloat(value), nil
	case fields.ModuleFieldTypeBool:
		value, ok := value.(bool)
		if !ok {
			return actions.AtomicValue{}, fmt.Errorf("expected boolean")
		}
		return actions.AtomicBool(value), nil
	case fields.ModuleFieldTypeArray:
		if field.ArrayStorage.Normalize() == fields.ModuleFieldArrayStorageJSON {
			encoded, err := fields.MarshalJSONArray(value)
			if err != nil {
				return actions.AtomicValue{}, err
			}
			return actions.AtomicValue{JSON: encoded}, nil
		}
		switch value := value.(type) {
		case []string:
			return actions.AtomicValue{Strings: value}, nil
		case []int64:
			return actions.AtomicValue{Ints: value}, nil
		case []interface{}:
			strings := make([]string, 0, len(value))
			ints := make([]int64, 0, len(value))
			allStrings := true
			allInts := true
			for _, item := range value {
				stringValue, isString := item.(string)
				if !isString {
					allStrings = false
				} else {
					strings = append(strings, stringValue)
				}
				floatValue, isFloat := item.(float64)
				if !isFloat || math.Trunc(floatValue) != floatValue {
					allInts = false
				} else {
					ints = append(ints, int64(floatValue))
				}
			}
			if allStrings {
				return actions.AtomicValue{Strings: strings}, nil
			}
			if allInts {
				return actions.AtomicValue{Ints: ints}, nil
			}
			return actions.AtomicValue{}, fmt.Errorf("expected an array of strings or integers")
		default:
			return actions.AtomicValue{}, fmt.Errorf("expected array")
		}
	case fields.ModuleFieldTypeObject:
		encoded, err := fields.MarshalJSONObject(value)
		if err != nil {
			return actions.AtomicValue{}, err
		}
		return actions.AtomicValue{JSON: encoded}, nil
	default:
		return actions.AtomicValue{}, fmt.Errorf("unsupported type %q", field.Type)
	}
}
