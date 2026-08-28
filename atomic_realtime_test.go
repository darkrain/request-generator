package module

import (
	"context"
	"testing"
	"time"

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
			Payload: []actions.AtomicRealtimePayloadField{{
				Key:    "message",
				Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "message"},
			}},
		}},
	}
	record := actions.AtomicRecord{Fields: []actions.AtomicField{
		{Name: "recipient_user_ids", Value: actions.AtomicValue{Ints: []int64{4, 4, 9}}},
		{Name: "chat_id", Value: actions.AtomicInt(42)},
		{Name: "message", Value: actions.AtomicString("Hello")},
	}}

	publishes, err := atomicRealtimePublishes(config, actions.AtomicInput{}, record)
	require.NoError(t, err)
	require.Len(t, publishes, 1)
	require.Equal(t, []string{"user:4", "user:9"}, publishes[0].Topics)
	require.Equal(t, "chat_id", publishes[0].Correlation.Field)
	require.Equal(t, float64(42), publishes[0].Correlation.Value.Number)
	require.Equal(t, map[string]interface{}{"message": "Hello"}, publishes[0].Payload)
}

func TestValidateAtomicRealtimePayloadRejectsInputAndUnknownResult(t *testing.T) {
	chatID := pg.IntegerColumn("chat_id")
	text := pg.StringColumn("text")
	mod := &BaseModule{Fields: []fields.ModuleField{{Column: chatID, Type: fields.ModuleFieldTypeInt}, {Column: text, Type: fields.ModuleFieldTypeString}}}
	config := &actions.AtomicAddConfig{
		Operation: func(context.Context, actions.AtomicExecutor, actions.AtomicInput) (actions.AtomicRecord, error) {
			return actions.AtomicRecord{}, nil
		},
		ResultFields: []actions.AtomicResultField{
			{Name: "recipient_user_id", Kind: actions.AtomicValueKindInt},
			{Name: "chat_id", Kind: actions.AtomicValueKindInt},
		},
		Publish: []actions.AtomicRealtimePublishConfig{{
			Recipients:  []actions.AtomicRealtimeRecipient{{UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "recipient_user_id"}}},
			Correlation: &actions.AtomicRealtimeCorrelation{Field: "chat_id", Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"}},
			Payload:     []actions.AtomicRealtimePayloadField{{Key: "message", Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceInput, Field: "text"}}},
		}},
	}
	require.EqualError(t, validateAtomicAddConfig(mod, actions.AddModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "chat_id"}, Atomic: config}), `atomic realtime publish 0 payload "message" must use a result field`)

	config.Publish[0].Payload[0].Source = actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "unknown"}
	require.EqualError(t, validateAtomicAddConfig(mod, actions.AddModuleAction{Realtime: &actions.RealtimeEventConfig{CorrelationField: "chat_id"}, Atomic: config}), `atomic realtime publish 0 payload "message" references undeclared result field "unknown"`)
}

func TestAtomicRealtimeOptionalPublishSkipsEmptyRecipients(t *testing.T) {
	config := &actions.AtomicAddConfig{Publish: []actions.AtomicRealtimePublishConfig{{
		Recipients:          []actions.AtomicRealtimeRecipient{{UserID: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "recipient_user_ids"}}},
		Correlation:         &actions.AtomicRealtimeCorrelation{Field: "chat_id", Source: actions.AtomicValueSource{Scope: actions.AtomicValueSourceResult, Field: "chat_id"}},
		SkipEmptyRecipients: true,
	}}}
	publishes, err := atomicRealtimePublishes(config, actions.AtomicInput{}, actions.AtomicRecord{Fields: []actions.AtomicField{
		{Name: "recipient_user_ids", Value: actions.AtomicValue{Ints: []int64{}}},
		{Name: "chat_id", Value: actions.AtomicInt(42)},
	}})
	require.NoError(t, err)
	require.Empty(t, publishes)
}

func TestValidateAtomicResultRequiresDeclaredOutput(t *testing.T) {
	config := &actions.AtomicAddConfig{ResultFields: []actions.AtomicResultField{{Name: "chat_id", Kind: actions.AtomicValueKindInt}}}
	err := validateAtomicResult(config, actions.AtomicRecord{Value: 1, PrimaryKey: "id"})
	require.EqualError(t, err, `atomic result field "chat_id" is missing`)
}

func TestValidateAtomicResultAcceptsDeclaredTime(t *testing.T) {
	config := &actions.AtomicAddConfig{ResultFields: []actions.AtomicResultField{{Name: "deadline", Kind: actions.AtomicValueKindTime}}}
	record := actions.AtomicRecord{Fields: []actions.AtomicField{{Name: "deadline", Value: actions.AtomicTime(time.Now().UTC())}}}
	require.NoError(t, validateAtomicResult(config, record))
}
