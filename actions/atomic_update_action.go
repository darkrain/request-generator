package actions

import (
	"context"
	"fmt"
)

// UpdateMode selects the persistence path for a standard update action.
type UpdateMode string

const (
	UpdateModeStandard UpdateMode = "standard"
	UpdateModeAtomic   UpdateMode = "atomic"
)

// AtomicSelector is the validated and type-normalized selector from the
// standard update route `/:bykey/:value`.
type AtomicSelector struct {
	ByKey string      `json:"by_key"`
	Value AtomicValue `json:"value"`
}

func (selector AtomicSelector) Validate() error {
	if selector.ByKey == "" {
		return fmt.Errorf("atomic update selector by_key is required")
	}
	if err := selector.Value.Validate(); err != nil {
		return fmt.Errorf("atomic update selector value: %w", err)
	}
	return nil
}

// AtomicUpdateInput keeps the validated request body and typed route selector
// separate. Domain operations never receive gin context or an untyped map.
type AtomicUpdateInput struct {
	Input    AtomicInput    `json:"input"`
	Selector AtomicSelector `json:"selector"`
}

// AtomicUpdateActionOperation runs in the generator-owned transaction for a
// standard update route. It is intentionally distinct from AtomicUpdate, the
// closed SQL update declaration accepted by AtomicExecutor.
type AtomicUpdateActionOperation func(context.Context, AtomicExecutor, AtomicUpdateInput) (AtomicRecord, error)

// AtomicUpdateConfig declares the domain operation and generator-owned
// post-commit effects for an atomic standard update action.
type AtomicUpdateConfig struct {
	Operation    AtomicUpdateActionOperation   `json:"-"`
	ResultFields []AtomicResultField           `json:"result_fields,omitempty"`
	Publish      []AtomicRealtimePublishConfig `json:"publish,omitempty"`
}
