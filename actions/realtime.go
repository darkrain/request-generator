package actions

// RealtimeEventConfig declares the typed correlation field published by a
// standard action. The field is resolved against the owning module by the
// generator, which also derives and checks its wire type.
type RealtimeEventConfig struct {
	CorrelationField string `json:"correlation_field,omitempty"`
}

// RealtimeEvent returns the configuration declared by a standard action.
func RealtimeEvent(action ModuleAction) *RealtimeEventConfig {
	switch value := action.(type) {
	case AddModuleAction:
		return value.Realtime
	case *AddModuleAction:
		return value.Realtime
	case UpdateModuleAction:
		return value.Realtime
	case *UpdateModuleAction:
		return value.Realtime
	case DeleteModuleAction:
		return value.Realtime
	case *DeleteModuleAction:
		return value.Realtime
	case ListModuleAction:
		return value.Realtime
	case *ListModuleAction:
		return value.Realtime
	case ViewModuleAction:
		return value.Realtime
	case *ViewModuleAction:
		return value.Realtime
	case DefrecModuleAction:
		return value.Realtime
	case *DefrecModuleAction:
		return value.Realtime
	default:
		return nil
	}
}
