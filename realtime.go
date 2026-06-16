package module

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/darkrain/request-generator/actions"
	"github.com/darkrain/request-generator/icontext"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	realtimeContextKey = "_rg_realtime_event"
)

type RealtimeConfig struct {
	Enabled bool
	Broker  RealtimeBroker
	Path    string
}

type RealtimeBroker interface {
	Publish(ctx context.Context, event RealtimeEvent) (RealtimeEvent, error)
	Replay(ctx context.Context, afterID string, limit int) ([]RealtimeEvent, bool, error)
}

type RealtimeEvent struct {
	EventID   string                 `json:"event_id"`
	Type      string                 `json:"type"`
	Module    string                 `json:"module"`
	Action    string                 `json:"action"`
	RecordID  interface{}            `json:"record_id,omitempty"`
	Topics    []string               `json:"topics,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

type RealtimePublish struct {
	Module   string
	Action   string
	Topics   []string
	RecordID interface{}
	Payload  map[string]interface{}
}

type MemoryBrokerOptions struct {
	MaxEvents int
}

type MemoryBroker struct {
	mu     sync.RWMutex
	seq    int64
	events []RealtimeEvent
	max    int
}

func NewMemoryBroker(options MemoryBrokerOptions) *MemoryBroker {
	max := options.MaxEvents
	if max <= 0 {
		max = 1000
	}
	return &MemoryBroker{max: max}
}

func (b *MemoryBroker) Publish(ctx context.Context, event RealtimeEvent) (RealtimeEvent, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	event.EventID = strconv.FormatInt(b.seq, 10)
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	b.events = append(b.events, event)
	if len(b.events) > b.max {
		b.events = append([]RealtimeEvent(nil), b.events[len(b.events)-b.max:]...)
	}
	return event, nil
}

func (b *MemoryBroker) Replay(ctx context.Context, afterID string, limit int) ([]RealtimeEvent, bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if afterID == "" {
		return nil, false, nil
	}
	after, err := strconv.ParseInt(afterID, 10, 64)
	if err != nil {
		return nil, true, nil
	}
	if len(b.events) == 0 {
		return nil, true, nil
	}
	first, _ := strconv.ParseInt(b.events[0].EventID, 10, 64)
	if after < first-1 {
		return nil, true, nil
	}
	if limit <= 0 {
		limit = 500
	}
	out := make([]RealtimeEvent, 0)
	for _, event := range b.events {
		id, _ := strconv.ParseInt(event.EventID, 10, 64)
		if id > after {
			out = append(out, event)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, false, nil
}

type realtimeHub struct {
	mu          sync.RWMutex
	connections map[*realtimeConnection]struct{}
}

type realtimeConnection struct {
	hub    *realtimeHub
	conn   *websocket.Conn
	write  sync.Mutex
	send   chan RealtimeEvent
	topics map[string]struct{}
	userID int64
	role   string
}

func newRealtimeHub() *realtimeHub {
	return &realtimeHub{connections: make(map[*realtimeConnection]struct{})}
}

func (h *realtimeHub) add(c *realtimeConnection) {
	h.mu.Lock()
	h.connections[c] = struct{}{}
	h.mu.Unlock()
}

func (h *realtimeHub) remove(c *realtimeConnection) {
	h.mu.Lock()
	delete(h.connections, c)
	h.mu.Unlock()
	close(c.send)
}

func (h *realtimeHub) publish(event RealtimeEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.connections {
		if !c.matches(event.Topics) {
			continue
		}
		select {
		case c.send <- event:
		default:
		}
	}
}

func (c *realtimeConnection) matches(topics []string) bool {
	if len(c.topics) == 0 {
		return false
	}
	for _, topic := range topics {
		if _, ok := c.topics[topic]; ok {
			return true
		}
	}
	return false
}

func (generator *Generator) initRealtime() {
	if !generator.Realtime.Enabled {
		return
	}
	if generator.Realtime.Broker == nil {
		panic("request-generator realtime enabled without broker")
	}
	if generator.realtimeHub == nil {
		generator.realtimeHub = newRealtimeHub()
	}
	path := generator.Realtime.Path
	if path == "" {
		path = "/api/ws"
	}
	group := generator.group.Group(path)
	dummyAction := actions.ListModuleAction{Auth: true, Permission: []actions.Role{}}
	if generator.AuthMiddleware != nil {
		group.Use(func(c *gin.Context) {
			if token := c.Query("token"); token != "" && c.GetHeader("Authorization") == "" {
				c.Request.Header.Set("Authorization", "Bearer "+token)
			}
			generator.AuthMiddleware(dummyAction)(c)
		})
	}
	group.GET("", generator.handleRealtimeWebSocket())
}

func (generator *Generator) handleRealtimeWebSocket() gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return func(c *gin.Context) {
		user, ok := icontext.GetUser(c.Request.Context())
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		rc := &realtimeConnection{
			hub:    generator.realtimeHub,
			conn:   conn,
			send:   make(chan RealtimeEvent, 64),
			topics: map[string]struct{}{fmt.Sprintf("user:%d", user.ID): {}},
			userID: user.ID,
			role:   user.Role,
		}
		generator.realtimeHub.add(rc)
		defer func() {
			generator.realtimeHub.remove(rc)
			conn.Close()
		}()

		go rc.writeLoop()
		rc.readLoop(generator)
	}
}

type realtimeClientMessage struct {
	Type        string   `json:"type"`
	RequestID   string   `json:"request_id,omitempty"`
	LastEventID string   `json:"last_event_id,omitempty"`
	Topics      []string `json:"topics,omitempty"`
}

func (c *realtimeConnection) readLoop(generator *Generator) {
	for {
		var msg realtimeClientMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		switch msg.Type {
		case "hello":
			c.handleHello(generator, msg)
		case "subscribe":
			c.handleSubscribe(msg)
		case "unsubscribe":
			c.handleUnsubscribe(msg)
		case "ping":
			_ = c.writeJSON(gin.H{"type": "pong", "request_id": msg.RequestID})
		}
	}
}

func (c *realtimeConnection) writeLoop() {
	for event := range c.send {
		if err := c.writeJSON(event); err != nil {
			return
		}
	}
}

func (c *realtimeConnection) writeJSON(v interface{}) error {
	c.write.Lock()
	defer c.write.Unlock()
	return c.conn.WriteJSON(v)
}

func (c *realtimeConnection) handleHello(generator *Generator, msg realtimeClientMessage) {
	replay := "ok"
	if msg.LastEventID != "" {
		events, resync, err := generator.Realtime.Broker.Replay(context.Background(), msg.LastEventID, 500)
		if err != nil || resync {
			replay = "resync_required"
		} else {
			for _, event := range events {
				if c.matches(event.Topics) {
					c.send <- event
				}
			}
		}
	}
	_ = c.writeJSON(gin.H{"type": "ready", "request_id": msg.RequestID, "replay": replay})
}

func (c *realtimeConnection) handleSubscribe(msg realtimeClientMessage) {
	accepted := []string{}
	for _, topic := range msg.Topics {
		if c.canSubscribe(topic) {
			c.topics[topic] = struct{}{}
			accepted = append(accepted, topic)
		}
	}
	_ = c.writeJSON(gin.H{"type": "subscribed", "request_id": msg.RequestID, "topics": accepted})
}

func (c *realtimeConnection) handleUnsubscribe(msg realtimeClientMessage) {
	for _, topic := range msg.Topics {
		if topic == fmt.Sprintf("user:%d", c.userID) {
			continue
		}
		delete(c.topics, topic)
	}
	_ = c.writeJSON(gin.H{"type": "unsubscribed", "request_id": msg.RequestID})
}

func (c *realtimeConnection) canSubscribe(topic string) bool {
	own := fmt.Sprintf("user:%d", c.userID)
	if topic == own {
		return true
	}
	return false
}

func SetRealtimePublish(c *gin.Context, event RealtimePublish) {
	if len(event.Topics) == 0 {
		return
	}
	raw, ok := c.Get(realtimeContextKey)
	if !ok {
		c.Set(realtimeContextKey, []RealtimePublish{event})
		return
	}
	switch existing := raw.(type) {
	case []RealtimePublish:
		c.Set(realtimeContextKey, append(existing, event))
	case RealtimePublish:
		c.Set(realtimeContextKey, []RealtimePublish{existing, event})
	default:
		c.Set(realtimeContextKey, []RealtimePublish{event})
	}
}

func (generator *Generator) publishRealtime(c *gin.Context, module *BaseModule, action actions.ModuleActionName, output interface{}) {
	if !generator.Realtime.Enabled || generator.Realtime.Broker == nil || generator.realtimeHub == nil {
		return
	}
	raw, ok := c.Get(realtimeContextKey)
	if !ok {
		return
	}
	pubs := []RealtimePublish{}
	switch value := raw.(type) {
	case RealtimePublish:
		pubs = append(pubs, value)
	case []RealtimePublish:
		pubs = value
	default:
		return
	}
	for _, pub := range pubs {
		generator.publishRealtimeEvent(c, module, action, output, pub)
	}
}

func (generator *Generator) publishRealtimeEvent(c *gin.Context, module *BaseModule, action actions.ModuleActionName, output interface{}, pub RealtimePublish) {
	if len(pub.Topics) == 0 {
		return
	}
	event := RealtimeEvent{
		Type:      "event",
		Module:    module.Name,
		Action:    string(action),
		RecordID:  pub.RecordID,
		Topics:    pub.Topics,
		CreatedAt: time.Now().UTC(),
		Payload:   pub.Payload,
	}
	if pub.Module != "" {
		event.Module = pub.Module
	}
	if pub.Action != "" {
		event.Action = pub.Action
	}
	if event.Payload == nil {
		event.Payload = map[string]interface{}{}
	}
	if pub.RecordID == nil {
		if mapped := outputRecordID(output); mapped != nil {
			event.RecordID = mapped
		}
	}
	published, err := generator.Realtime.Broker.Publish(c.Request.Context(), event)
	if err != nil {
		return
	}
	generator.realtimeHub.publish(published)
}

func outputRecordID(output interface{}) interface{} {
	if output == nil {
		return nil
	}
	body, err := json.Marshal(output)
	if err != nil {
		return nil
	}
	var mapped map[string]interface{}
	if err := json.Unmarshal(body, &mapped); err != nil {
		return nil
	}
	if v, ok := mapped["value"]; ok {
		return v
	}
	if v, ok := mapped["id"]; ok {
		return v
	}
	return nil
}
