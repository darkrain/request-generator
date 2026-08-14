package module

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/darkrain/request-generator/renderer"
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
			Key:   "parent_id",
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
	event.Correlation.Key = "changed"
	replayed, resync, err := broker.Replay(context.Background(), "0", 1)
	if err != nil || resync || len(replayed) != 1 {
		t.Fatalf("replay typed correlation: events=%#v resync=%v err=%v", replayed, resync, err)
	}
	if replayed[0].Correlation.Key != "parent_id" {
		t.Fatalf("broker stored a mutable correlation: %#v", replayed[0].Correlation)
	}
	if _, err := broker.Publish(context.Background(), RealtimeEvent{
		Correlation: &RealtimeCorrelation{Key: "parent_id", Value: renderer.TypedValue{}},
	}); err == nil {
		t.Fatal("expected invalid correlation to be rejected")
	}
}
