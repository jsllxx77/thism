package hub

import (
	"fmt"
	"sync"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

type agentSocket interface {
	WriteJSON(v any) error
	Close() error
}

type agentConn struct {
	nodeID  string
	conn    agentSocket
	writeMu sync.Mutex
}

// Hub manages all agent WebSocket connections and dashboard subscribers.
type Hub struct {
	store       *store.Store
	agents      map[string]*agentConn
	subscribers []chan models.WSMessage
	mu          sync.RWMutex
	register    chan *agentConn
	unregister  chan string
	broadcast   chan models.WSMessage
}

func New(s *store.Store) *Hub {
	return &Hub{
		store:      s,
		agents:     make(map[string]*agentConn),
		register:   make(chan *agentConn, 16),
		unregister: make(chan string, 16),
		broadcast:  make(chan models.WSMessage, 64),
	}
}

// Run is the hub's main event loop. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.agents[conn.nodeID] = conn
			h.mu.Unlock()
			h.Broadcast(models.WSMessage{
				Type: "node_status",
				Payload: map[string]any{
					"node_id": conn.nodeID,
					"online":  true,
				},
			})

		case nodeID := <-h.unregister:
			h.mu.Lock()
			delete(h.agents, nodeID)
			h.mu.Unlock()
			h.Broadcast(models.WSMessage{
				Type: "node_status",
				Payload: map[string]any{
					"node_id": nodeID,
					"online":  false,
				},
			})

		case msg := <-h.broadcast:
			h.mu.RLock()
			subs := make([]chan models.WSMessage, len(h.subscribers))
			copy(subs, h.subscribers)
			h.mu.RUnlock()
			for _, sub := range subs {
				select {
				case sub <- msg:
				default:
					// subscriber is slow; skip rather than block
				}
			}
		}
	}
}

// Register adds an agent connection to the hub.
func (h *Hub) Register(nodeID string, conn agentSocket) {
	h.register <- &agentConn{nodeID: nodeID, conn: conn}
}

// Unregister removes an agent connection from the hub.
func (h *Hub) Unregister(nodeID string) {
	h.unregister <- nodeID
}

// SendToAgent writes a typed JSON message to a specific online agent.
func (h *Hub) SendToAgent(nodeID string, msg models.WSMessage) error {
	h.mu.RLock()
	agent := h.agents[nodeID]
	h.mu.RUnlock()
	if agent == nil || agent.conn == nil {
		return fmt.Errorf("agent %s is offline", nodeID)
	}

	agent.writeMu.Lock()
	defer agent.writeMu.Unlock()
	return agent.conn.WriteJSON(msg)
}

// IsOnline reports whether a node is currently connected.
func (h *Hub) IsOnline(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[nodeID]
	return ok
}

// OnlineNodeIDs returns the IDs of all currently connected nodes.
func (h *Hub) OnlineNodeIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}

// Broadcast sends a message to all dashboard subscribers.
func (h *Hub) Broadcast(msg models.WSMessage) {
	h.broadcast <- msg
}

// Subscribe adds a channel to receive broadcasted messages.
func (h *Hub) Subscribe(ch chan models.WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers = append(h.subscribers, ch)
}

// Unsubscribe removes a channel from the subscriber list.
func (h *Hub) Unsubscribe(ch chan models.WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, sub := range h.subscribers {
		if sub == ch {
			h.subscribers = append(h.subscribers[:i], h.subscribers[i+1:]...)
			return
		}
	}
}
