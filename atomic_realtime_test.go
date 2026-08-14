package module

import (
	"context"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestValidateAtomicAddConfigRejectsClientControlledRecipient(t *testing.T) {
	chatID := pg.IntegerColumn("chat_id")
	recipientUserID := pg.IntegerColumn("recipient_user_id")
	mod := &BaseModule{Fields: []fields.ModuleField{
		{Column: chatID, Type: fields.ModuleFieldTypeInt},
		{Column: recipientUserID, Type: fields.ModuleFieldTypeInt},
	}}
	action := actions.AddModuleAction{
		Realtime: &actions.RealtimeEventConfig{CorrelationField: "chat_id"},
		Atomic: &actions.AtomicAddConfig{
			Operation: func(context.Context, actions.AtomicExecutor, actions.AtomicInput) (actions.AtomicRecord, error) {
				return actions.AtomicRecord{}, nil
			},
			ResultFields: []actions.AtomicResultField{{Name: "chat_id", Kind: actions.AtomicValueKindInt}},
			Publish: []actions.AtomicRealtimePublishConfig{{
				Recipients: []actions.AtomicRealtimeRecipient{{
					UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceInput, Field: "recipient_user_id"},
				}},
				Correlation: &actions.AtomicRealtimeCorrelation{
					Field:  "chat_id",
					Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"},
				},
			}},
		},
	}

	require.EqualError(t, validateAtomicAddConfig(mod, action), `atomic realtime publish 0 recipient 0 must use a result field`)
}

func TestAtomicRealtimePublishesUsesTrustedResults(t *testing.T) {
	config := &actions.AtomicAddConfig{
		Publish: []actions.AtomicRealtimePublishConfig{{
			Recipients: []actions.AtomicRealtimeRecipient{{
				UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "recipient_user_ids"},
			}},
			Correlation: &actions.AtomicRealtimeCorrelation{
				Field:  "chat_id",
				Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"},
			},
		}},
	}
	record := actions.AtomicRecord{Fields: []actions.AtomicField{
		{Name: "recipient_user_ids", Value: actions.AtomicValue{Ints: []int64{4, 4, 9}}},
		{Name: "chat_id", Value: actions.AtomicInt(42)},
	}}

	publishes, err := atomicRealtimePublishes(config, actions.AtomicInput{}, record)
	require.NoError(t, err)
	require.Len(t, publishes, 1)
	require.Equal(t, []string{"user:4", "user:9"}, publishes[0].Topics)
	require.Equal(t, "chat_id", publishes[0].Correlation.Field)
	require.Equal(t, float64(42), publishes[0].Correlation.Value.Number)
}

func TestValidateAtomicResultRequiresDeclaredOutput(t *testing.T) {
	config := &actions.AtomicAddConfig{ResultFields: []actions.AtomicResultField{{Name: "chat_id", Kind: actions.AtomicValueKindInt}}}
	err := validateAtomicResult(config, actions.AtomicRecord{Value: 1, PrimaryKey: "id"})
	require.EqualError(t, err, `atomic result field "chat_id" is missing`)
}
