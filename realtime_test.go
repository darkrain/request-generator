package module

import (
	"context"
	"testing"
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
