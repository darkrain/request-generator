package module

import (
	"fmt"
	"time"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/renderer"
	"github.com/gin-gonic/gin"
)

func validateAtomicAddConfig(module *BaseModule, action actions.AddModuleAction) error {
	config := action.Atomic
	if config == nil {
		return fmt.Errorf("atomic add action requires an operation")
	}
	return validateAtomicResultConfig(module, action.Realtime, config.ResultFields, config.Publish)
}

func validateAtomicUpdateConfig(module *BaseModule, action actions.UpdateModuleAction) error {
	config := action.Atomic
	if config == nil {
		return fmt.Errorf("atomic update action requires an operation")
	}
	return validateAtomicResultConfig(module, action.Realtime, config.ResultFields, config.Publish)
}

func validateAtomicResultConfig(module *BaseModule, realtime *actions.RealtimeEventConfig, resultFieldList []actions.AtomicResultField, configuredPublishes []actions.AtomicRealtimePublishConfig) error {
	resultFields := make(map[string]actions.AtomicValueKind, len(resultFieldList))
	for _, field := range resultFieldList {
		if err := field.Validate(); err != nil {
			return err
		}
		if _, exists := resultFields[field.Name]; exists {
			return fmt.Errorf("atomic result field %q is duplicated", field.Name)
		}
		resultFields[field.Name] = field.Kind
	}
	if len(configuredPublishes) == 0 {
		return nil
	}
	if realtime == nil || realtime.CorrelationField == "" {
		return fmt.Errorf("atomic realtime publish requires declared realtime correlation")
	}
	for index, publish := range configuredPublishes {
		if len(publish.Recipients) == 0 {
			return fmt.Errorf("atomic realtime publish %d requires recipients", index)
		}
		for recipientIndex, recipient := range publish.Recipients {
			if err := recipient.UserID.Validate(); err != nil {
				return fmt.Errorf("atomic realtime publish %d recipient %d: %w", index, recipientIndex, err)
			}
			if recipient.UserID.Scope != actions.AtomicValueSourceResult {
				return fmt.Errorf("atomic realtime publish %d recipient %d must use a result field", index, recipientIndex)
			}
			kind, ok := resultFields[recipient.UserID.Field]
			if !ok {
				return fmt.Errorf("atomic realtime publish %d recipient %d references undeclared result field %q", index, recipientIndex, recipient.UserID.Field)
			}
			if kind != actions.AtomicValueKindInt && kind != actions.AtomicValueKindInts {
				return fmt.Errorf("atomic realtime publish %d recipient %d field %q must be int or ints", index, recipientIndex, recipient.UserID.Field)
			}
		}
		if publish.Correlation == nil {
			return fmt.Errorf("atomic realtime publish %d requires correlation", index)
		}
		if publish.Correlation.Field != realtime.CorrelationField {
			return fmt.Errorf("atomic realtime publish %d correlation field %q does not match declared field %q", index, publish.Correlation.Field, realtime.CorrelationField)
		}
		if err := publish.Correlation.Source.Validate(); err != nil {
			return fmt.Errorf("atomic realtime publish %d correlation: %w", index, err)
		}
		kind, err := atomicSourceKind(module, resultFields, publish.Correlation.Source)
		if err != nil {
			return fmt.Errorf("atomic realtime publish %d correlation: %w", index, err)
		}
		field := module.GetField(publish.Correlation.Field)
		if field == nil {
			return fmt.Errorf("atomic realtime publish %d correlation field %q is not declared", index, publish.Correlation.Field)
		}
		expected, err := runtimeTypedValueType(*field)
		if err != nil {
			return fmt.Errorf("atomic realtime publish %d correlation field %q: %w", index, publish.Correlation.Field, err)
		}
		actual, ok := atomicKindTypedValueType(kind)
		if !ok || actual != expected {
			return fmt.Errorf("atomic realtime publish %d correlation field %q type does not match declared field", index, publish.Correlation.Field)
		}
		payloadKeys := make(map[string]struct{}, len(publish.Payload))
		for payloadIndex, payload := range publish.Payload {
			if payload.Key == "" {
				return fmt.Errorf("atomic realtime publish %d payload %d key is required", index, payloadIndex)
			}
			if _, exists := payloadKeys[payload.Key]; exists {
				return fmt.Errorf("atomic realtime publish %d payload key %q is duplicated", index, payload.Key)
			}
			payloadKeys[payload.Key] = struct{}{}
			if err := payload.Source.Validate(); err != nil {
				return fmt.Errorf("atomic realtime publish %d payload %q: %w", index, payload.Key, err)
			}
			if payload.Source.Scope != actions.AtomicValueSourceResult {
				return fmt.Errorf("atomic realtime publish %d payload %q must use a result field", index, payload.Key)
			}
			if _, ok := resultFields[payload.Source.Field]; !ok {
				return fmt.Errorf("atomic realtime publish %d payload %q references undeclared result field %q", index, payload.Key, payload.Source.Field)
			}
		}
	}
	return nil
}

func atomicSourceKind(module *BaseModule, resultFields map[string]actions.AtomicValueKind, source actions.AtomicValueSource) (actions.AtomicValueKind, error) {
	switch source.Scope {
	case actions.AtomicValueSourceResult:
		kind, ok := resultFields[source.Field]
		if !ok {
			return "", fmt.Errorf("references undeclared result field %q", source.Field)
		}
		return kind, nil
	case actions.AtomicValueSourceInput:
		field := module.GetField(source.Field)
		if field == nil {
			return "", fmt.Errorf("references unknown input field %q", source.Field)
		}
		switch field.Type {
		case fields.ModuleFieldTypeString:
			return actions.AtomicValueKindString, nil
		case fields.ModuleFieldTypeInt:
			return actions.AtomicValueKindInt, nil
		case fields.ModuleFieldTypeFloat:
			return actions.AtomicValueKindFloat, nil
		case fields.ModuleFieldTypeBool:
			return actions.AtomicValueKindBool, nil
		default:
			return "", fmt.Errorf("input field %q type %q cannot be used by atomic realtime", source.Field, field.Type)
		}
	default:
		return "", fmt.Errorf("source scope %q is unsupported", source.Scope)
	}
}

func atomicKindTypedValueType(kind actions.AtomicValueKind) (renderer.TypedValueType, bool) {
	switch kind {
	case actions.AtomicValueKindString:
		return renderer.TypedValueString, true
	case actions.AtomicValueKindInt, actions.AtomicValueKindFloat:
		return renderer.TypedValueNumber, true
	case actions.AtomicValueKindBool:
		return renderer.TypedValueBool, true
	default:
		return "", false
	}
}

func validateAtomicResult(config *actions.AtomicAddConfig, record actions.AtomicRecord) error {
	if config == nil {
		return nil
	}
	return validateAtomicResultFields(config.ResultFields, record)
}

func validateAtomicUpdateResult(config *actions.AtomicUpdateConfig, record actions.AtomicRecord) error {
	if config == nil {
		return nil
	}
	return validateAtomicResultFields(config.ResultFields, record)
}

func validateAtomicResultFields(resultFields []actions.AtomicResultField, record actions.AtomicRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	if len(resultFields) == 0 {
		return nil
	}
	actual := make(map[string]actions.AtomicValueKind, len(record.Fields))
	for _, field := range record.Fields {
		actual[field.Name] = atomicValueKind(field.Value)
	}
	for _, declared := range resultFields {
		kind, exists := actual[declared.Name]
		if !exists {
			return fmt.Errorf("atomic result field %q is missing", declared.Name)
		}
		if kind != declared.Kind {
			return fmt.Errorf("atomic result field %q has kind %q, expected %q", declared.Name, kind, declared.Kind)
		}
	}
	return nil
}

func atomicValueKind(value actions.AtomicValue) actions.AtomicValueKind {
	switch {
	case value.String != nil:
		return actions.AtomicValueKindString
	case value.Int != nil:
		return actions.AtomicValueKindInt
	case value.Float != nil:
		return actions.AtomicValueKindFloat
	case value.Bool != nil:
		return actions.AtomicValueKindBool
	case value.Time != nil:
		return actions.AtomicValueKindTime
	case value.Strings != nil:
		return actions.AtomicValueKindStrings
	case value.Ints != nil:
		return actions.AtomicValueKindInts
	default:
		return "json"
	}
}

func atomicRealtimePublishes(config *actions.AtomicAddConfig, input actions.AtomicInput, record actions.AtomicRecord) ([]RealtimePublish, error) {
	if config == nil {
		return nil, nil
	}
	return atomicRealtimePublishesFor(config.Publish, input, record)
}

func atomicUpdateRealtimePublishes(config *actions.AtomicUpdateConfig, input actions.AtomicInput, record actions.AtomicRecord) ([]RealtimePublish, error) {
	if config == nil {
		return nil, nil
	}
	return atomicRealtimePublishesFor(config.Publish, input, record)
}

func atomicRealtimePublishesFor(configuredPublishes []actions.AtomicRealtimePublishConfig, input actions.AtomicInput, record actions.AtomicRecord) ([]RealtimePublish, error) {
	if len(configuredPublishes) == 0 {
		return nil, nil
	}
	result := make(map[string]actions.AtomicValue, len(record.Fields))
	for _, field := range record.Fields {
		result[field.Name] = field.Value
	}
	publishes := make([]RealtimePublish, 0, len(configuredPublishes))
	for index, configured := range configuredPublishes {
		if configured.Correlation == nil {
			return nil, fmt.Errorf("atomic realtime publish %d requires correlation", index)
		}
		topics := make([]string, 0, len(configured.Recipients))
		seenTopics := make(map[string]struct{})
		for recipientIndex, recipient := range configured.Recipients {
			value, err := resolveAtomicValueSource(input, result, recipient.UserID)
			if err != nil {
				return nil, fmt.Errorf("atomic realtime publish %d recipient %d: %w", index, recipientIndex, err)
			}
			for _, userID := range atomicUserIDs(value) {
				topic := fmt.Sprintf("user:%d", userID)
				if _, exists := seenTopics[topic]; exists {
					continue
				}
				seenTopics[topic] = struct{}{}
				topics = append(topics, topic)
			}
		}
		if len(topics) == 0 {
			if configured.SkipEmptyRecipients {
				continue
			}
			return nil, fmt.Errorf("atomic realtime publish %d has no recipients", index)
		}
		correlationValue, err := resolveAtomicValueSource(input, result, configured.Correlation.Source)
		if err != nil {
			return nil, fmt.Errorf("atomic realtime publish %d correlation: %w", index, err)
		}
		typedValue, err := atomicTypedValue(correlationValue)
		if err != nil {
			return nil, fmt.Errorf("atomic realtime publish %d correlation: %w", index, err)
		}
		payload := make(map[string]interface{}, len(configured.Payload))
		for payloadIndex, configuredField := range configured.Payload {
			value, err := resolveAtomicValueSource(input, result, configuredField.Source)
			if err != nil {
				return nil, fmt.Errorf("atomic realtime publish %d payload %d: %w", index, payloadIndex, err)
			}
			payload[configuredField.Key] = atomicRealtimePayloadValue(value)
		}
		publishes = append(publishes, RealtimePublish{
			Topics: topics,
			Correlation: &RealtimeCorrelation{
				Field: configured.Correlation.Field,
				Value: typedValue,
			},
			Payload: payload,
		})
	}
	return publishes, nil
}

func atomicRealtimePayloadValue(value actions.AtomicValue) interface{} {
	switch {
	case value.Time != nil:
		return value.Time.UTC().Format(time.RFC3339Nano)
	case value.Strings != nil:
		return append([]string(nil), value.Strings...)
	case value.Ints != nil:
		return append([]int64(nil), value.Ints...)
	default:
		return value.Interface()
	}
}

func resolveAtomicValueSource(input actions.AtomicInput, result map[string]actions.AtomicValue, source actions.AtomicValueSource) (actions.AtomicValue, error) {
	switch source.Scope {
	case actions.AtomicValueSourceInput:
		value, ok := input.Field(source.Field)
		if !ok {
			return actions.AtomicValue{}, fmt.Errorf("input field %q is missing", source.Field)
		}
		return value, nil
	case actions.AtomicValueSourceResult:
		value, ok := result[source.Field]
		if !ok {
			return actions.AtomicValue{}, fmt.Errorf("result field %q is missing", source.Field)
		}
		return value, nil
	default:
		return actions.AtomicValue{}, fmt.Errorf("source scope %q is unsupported", source.Scope)
	}
}

func atomicUserIDs(value actions.AtomicValue) []int64 {
	if value.Int != nil {
		return []int64{*value.Int}
	}
	if value.Ints != nil {
		return value.Ints
	}
	return nil
}

func atomicTypedValue(value actions.AtomicValue) (renderer.TypedValue, error) {
	switch {
	case value.String != nil:
		return renderer.TypedValue{Type: renderer.TypedValueString, String: *value.String}, nil
	case value.Int != nil:
		return renderer.TypedValue{Type: renderer.TypedValueNumber, Number: float64(*value.Int)}, nil
	case value.Float != nil:
		return renderer.TypedValue{Type: renderer.TypedValueNumber, Number: *value.Float}, nil
	case value.Bool != nil:
		return renderer.TypedValue{Type: renderer.TypedValueBool, Bool: value.Bool}, nil
	default:
		return renderer.TypedValue{}, fmt.Errorf("value must be scalar")
	}
}

func (generator *Generator) publishAtomicRealtime(c *gin.Context, module *BaseModule, action actions.ModuleActionName, record actions.AtomicRecord, publishes []RealtimePublish) {
	for _, publish := range publishes {
		_, _ = generator.publishRealtimeEvent(c.Request.Context(), module, action, record, publish)
	}
}
