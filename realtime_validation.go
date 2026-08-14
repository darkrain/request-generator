package module

import (
	"fmt"

	"github.com/darkrain/request-generator/actions"
)

func (generator *Generator) validateRealtimeEvents() error {
	for _, module := range generator.Modules {
		for _, action := range module.Actions {
			if err := validateRealtimeEventConfig(module, action); err != nil {
				return fmt.Errorf("module %q action %q: %w", module.Name, action.Action(), err)
			}
		}
	}
	return nil
}

func validateRealtimeEventConfig(module *BaseModule, action actions.ModuleAction) error {
	event := actions.RealtimeEvent(action)
	if event == nil {
		return nil
	}
	switch action.Action() {
	case actions.ModuleActionNameAdd, actions.ModuleActionNameUpdate, actions.ModuleActionNameDelete:
	default:
		return fmt.Errorf("realtime event is only supported by add, update, or delete actions")
	}
	if event.CorrelationField == "" {
		return fmt.Errorf("realtime correlation field is required")
	}
	field := module.GetField(event.CorrelationField)
	if field == nil {
		return fmt.Errorf("realtime correlation field %q is not declared", event.CorrelationField)
	}
	if _, err := runtimeTypedValueType(*field); err != nil {
		return fmt.Errorf("realtime correlation field %q: %w", event.CorrelationField, err)
	}
	return nil
}
