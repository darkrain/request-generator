package module

import (
	"context"
	"testing"

	"github.com/darkrain/request-generator/actions"
	"github.com/stretchr/testify/require"
)

func TestExternalRealtimeDeliveryDoesNotEchoOrRepublish(t *testing.T) {
	broker := &realtimeBrokerStub{}
	hub := newRealtimeHub()
	subscriber := &sseConnection{send: make(chan RealtimeEvent, 2), topics: map[string]struct{}{"actor:7": {}}}
	other := &sseConnection{send: make(chan RealtimeEvent, 2), topics: map[string]struct{}{"actor:8": {}}}
	hub.addSSE(subscriber)
	hub.addSSE(other)
	g := &Generator{Modules: []*BaseModule{{Name: "records", Actions: []actions.ModuleAction{actions.AddModuleAction{}}}},
		Realtime: RealtimeConfig{Enabled: true, Broker: broker, ExternalDelivery: true}, realtimeHub: hub}
	event, err := g.PublishCommittedRealtime(context.Background(), "records", actions.ModuleActionNameAdd, RealtimePublish{Topics: []string{"actor:7"}})
	require.NoError(t, err)
	require.Empty(t, subscriber.send)
	require.NoError(t, g.DeliverRealtime(event))
	require.Equal(t, event.EventID, (<-subscriber.send).EventID)
	require.Empty(t, other.send)
	require.Equal(t, 1, broker.publishCount)
	event.Module = "unknown"
	require.Error(t, g.DeliverRealtime(event))
	require.Empty(t, subscriber.send)
}
