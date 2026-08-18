package module

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/fields"
	"github.com/darkrain/request-generator/icontext"
	"github.com/darkrain/request-generator/renderer"
	pg "github.com/go-jet/jet/v2/postgres"
	"github.com/stretchr/testify/require"
)

func TestMemoryBrokerReplay(t *testing.T) {
	broker := NewMemoryBroker(MemoryBrokerOptions{MaxEvents: 10})

	first, err := broker.Publish(context.Background(), RealtimeEvent{Module: "chat_messages", Action: "add"})
	if err != nil {
		t.Fatalf("publish first: %v", err)
	}
	second, err := broker.Publish(context.Background(), RealtimeEvent{Module: "chat_messages", Action: "add"})
	if err != nil {
		t.Fatalf("publish second: %v", err)
	}

	events, resync, err := broker.Replay(context.Background(), first.EventID, 10)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if resync {
		t.Fatal("replay should not require resync")
	}
	if len(events) != 1 || events[0].EventID != second.EventID {
		t.Fatalf("unexpected replay events: %#v", events)
	}
}

func TestRealtimeTopicsForUserRestrictsRoleTopics(t *testing.T) {
	topics := realtimeTopicsForUser(&icontext.UserInfo{ID: 42, Role: "model"})
	require.Contains(t, topics, "user:42")
	require.Contains(t, topics, "role:model")
	require.Contains(t, topics, "role:all")
	require.NotContains(t, topics, "role:agency")
}

func TestMemoryBrokerReplayOverflowRequiresResync(t *testing.T) {
	broker := NewMemoryBroker(MemoryBrokerOptions{MaxEvents: 2})

	first, err := broker.Publish(context.Background(), RealtimeEvent{Module: "chat_messages", Action: "add"})
	if err != nil {
		t.Fatalf("publish first: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := broker.Publish(context.Background(), RealtimeEvent{Module: "chat_messages", Action: "add"}); err != nil {
			t.Fatalf("publish overflow event %d: %v", i, err)
		}
	}

	_, resync, err := broker.Replay(context.Background(), first.EventID, 10)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !resync {
		t.Fatal("expected resync after ring buffer overflow")
	}
}

func TestRealtimeCorrelationIsTypedAndValidated(t *testing.T) {
	broker := NewMemoryBroker(MemoryBrokerOptions{})
	event, err := broker.Publish(context.Background(), RealtimeEvent{
		Module: "records",
		Action: "update",
		Correlation: &RealtimeCorrelation{
			Field: "parent_id",
			Value: renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 0},
		},
	})
	if err != nil {
		t.Fatalf("publish typed correlation: %v", err)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal typed correlation: %v", err)
	}
	if string(encoded) == "" {
		t.Fatal("expected serialized event")
	}
	event.Correlation.Field = "changed"
	replayed, resync, err := broker.Replay(context.Background(), "0", 1)
	if err != nil || resync || len(replayed) != 1 {
		t.Fatalf("replay typed correlation: events=%#v resync=%v err=%v", replayed, resync, err)
	}
	if replayed[0].Correlation.Field != "parent_id" {
		t.Fatalf("broker stored a mutable correlation: %#v", replayed[0].Correlation)
	}
	if _, err := broker.Publish(context.Background(), RealtimeEvent{
		Correlation: &RealtimeCorrelation{Field: "parent_id", Value: renderer.TypedValue{}},
	}); err == nil {
		t.Fatal("expected invalid correlation to be rejected")
	}
}

func TestValidateRealtimePublishUsesDeclaredActionCorrelation(t *testing.T) {
	parentID := pg.IntegerColumn("parent_id")
	module := &BaseModule{
		Name:   "records",
		Fields: []fields.ModuleField{{Column: parentID, Type: fields.ModuleFieldTypeInt}},
		Actions: []actions.ModuleAction{actions.AddModuleAction{
			Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"},
		}},
	}
	generator := &Generator{Modules: []*BaseModule{module}}

	valid := RealtimePublish{Correlation: &RealtimeCorrelation{
		Field: "parent_id",
		Value: renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 42},
	}}
	require.NoError(t, generator.validateRealtimePublish(module, actions.ModuleActionNameAdd, valid))

	wrongField := valid
	wrongField.Correlation = &RealtimeCorrelation{
		Field: "unknown",
		Value: renderer.TypedValue{Type: renderer.TypedValueNumber, Number: 42},
	}
	require.EqualError(t, generator.validateRealtimePublish(module, actions.ModuleActionNameAdd, wrongField), `realtime correlation field "unknown" does not match declared field "parent_id"`)

	wrongType := valid
	wrongType.Correlation = &RealtimeCorrelation{
		Field: "parent_id",
		Value: renderer.TypedValue{Type: renderer.TypedValueString, String: "42"},
	}
	require.EqualError(t, generator.validateRealtimePublish(module, actions.ModuleActionNameAdd, wrongType), `realtime correlation field "parent_id" has type "string", expected "number"`)
}

func TestValidateRealtimeEventsRejectsNonPublishingAction(t *testing.T) {
	parentID := pg.IntegerColumn("parent_id")
	generator := &Generator{Modules: []*BaseModule{{
		Name:   "records",
		Fields: []fields.ModuleField{{Column: parentID, Type: fields.ModuleFieldTypeInt}},
		Actions: []actions.ModuleAction{actions.ListModuleAction{
			Realtime: &actions.RealtimeEventConfig{CorrelationField: "parent_id"},
		}},
	}}}
	require.EqualError(t, generator.validateRealtimeEvents(), `module "records" action "list": realtime event is only supported by add, update, or delete actions`)
}
