package hub

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

// Errors returned by Hub operations after close or on stale generations.
var (
	ErrHubClosed       = errors.New("hub is closed")
	ErrAgentOffline    = errors.New("agent is offline")
	ErrStaleConnection = errors.New("agent connection is stale (replaced by a newer generation)")
)

// agentSocket is the write side of an agent WebSocket connection.
type agentSocket interface {
	WriteJSON(v any) error
	Close() error
}

// writeDeadlineSetter is implemented by real WebSocket connections so command
// writes never block connection replacement indefinitely.
type writeDeadlineSetter interface {
	SetWriteDeadline(t time.Time) error
}

// HandshakeGracePeriod is how long the server waits for the stream/watermark
// handshake before treating the connection as deliverable anyway. Agents that
// implement the handshake protocol send it immediately on connect (well under
// a second), so the grace period only expires for connections that will never
// handshake: legacy agents running a protocol older than the handshake,
// unknown/rolled-back builds, or future protocol mismatches. Failing open
// after the grace period keeps agent update commands flowing to those agents
// instead of leaving them stuck in pending forever; every agent version either
// executes "agent_command" messages (legacy agents) or ignores unknown message
// types, so delivery is safe in both directions. Exported as a var so tests
// can shorten it.
var HandshakeGracePeriod = 30 * time.Second

type agentConn struct {
	nodeID            string
	generation        uint64
	conn              agentSocket
	writeMu           sync.Mutex
	fenced            bool
	handshakeDone     bool
	handshakeDeadline time.Time
}

type unregisterRequest struct {
	nodeID     string
	conn       agentSocket
	generation uint64
}

// Hub manages all agent WebSocket connections and dashboard subscribers.
//
// Concurrency model (per docs/concurrency-remediation-plan.md §6.1):
//   - every registration allocates a monotonically increasing generation;
//   - replacing a connection and fencing the old one is a single identity
//     switch under h.mu;
//   - SendToAgent serializes against that switch: it holds h.mu (read) and the
//     entry's writeMu while performing the network write, so the outcome is
//     either "committed to the current writer at the linearization point" or
//     an explicit error. A fenced/stale entry never receives commands.
type Hub struct {
	store       *store.Store
	agents      map[string]*agentConn
	subscribers []chan models.WSMessage
	mu          sync.RWMutex
	register    chan *agentConn
	unregister  chan unregisterRequest
	broadcast   chan models.WSMessage
	closeCh     chan struct{}
	closeOnce   sync.Once
	nextGen     uint64
}

// New creates a Hub. Call Run in a goroutine, and Close when shutting down.
func New(s *store.Store) *Hub {
	return &Hub{
		store:      s,
		agents:     make(map[string]*agentConn),
		register:   make(chan *agentConn, 16),
		unregister: make(chan unregisterRequest, 16),
		broadcast:  make(chan models.WSMessage, 64),
		closeCh:    make(chan struct{}),
	}
}

// Run is the hub's main event loop. Call in a goroutine. It exits after Close.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			shouldBroadcast := false
			h.mu.Lock()
			if h.isClosedLocked() {
				h.mu.Unlock()
				if conn != nil && conn.conn != nil {
					_ = conn.conn.Close()
				}
				continue
			}
			current := h.agents[conn.nodeID]
			if current == nil || current.generation <= conn.generation {
				h.agents[conn.nodeID] = conn
				shouldBroadcast = true
			}
			h.mu.Unlock()
			if !shouldBroadcast {
				continue
			}
			h.Broadcast(models.WSMessage{
				Type: "node_status",
				Payload: map[string]any{
					"node_id":    conn.nodeID,
					"online":     true,
					"generation": conn.generation,
				},
			})

		case req := <-h.unregister:
			shouldBroadcast := false
			h.mu.Lock()
			if current := h.agents[req.nodeID]; current != nil &&
				current.generation == req.generation &&
				(req.conn == nil || current.conn == req.conn) {
				delete(h.agents, req.nodeID)
				shouldBroadcast = true
			}
			h.mu.Unlock()
			if shouldBroadcast {
				h.Broadcast(models.WSMessage{
					Type: "node_status",
					Payload: map[string]any{
						"node_id":    req.nodeID,
						"online":     false,
						"generation": req.generation,
					},
				})
			}

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

		case <-h.closeCh:
			h.mu.Lock()
			for _, agent := range h.agents {
				agent.fenced = true
				if agent.conn != nil {
					_ = agent.conn.Close()
				}
			}
			h.agents = make(map[string]*agentConn)
			h.subscribers = nil
			h.mu.Unlock()
			return
		}
	}
}

// Register adds an agent connection to the hub and returns its generation.
// Replacing an existing connection fences the previous generation: its socket
// is closed and its writes and unregister are ignored. Registering on a closed
// hub closes the connection and returns generation 0.
func (h *Hub) Register(nodeID string, conn agentSocket) uint64 {
	h.mu.Lock()
	if h.isClosedLocked() {
		h.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		return 0
	}
	h.nextGen++
	entry := &agentConn{
		nodeID:            nodeID,
		generation:        h.nextGen,
		conn:              conn,
		handshakeDeadline: time.Now().Add(HandshakeGracePeriod),
	}
	old := h.agents[nodeID]
	if old != nil {
		// Fence the previous generation before swapping so a concurrent
		// SendToAgent either finishes on the then-current writer or fails.
		old.fenced = true
	}
	h.agents[nodeID] = entry
	h.mu.Unlock()

	if old != nil && old.conn != nil {
		// Closing the stale socket is done outside the lock; it is fast and
		// must not stall the hub's event loop or new registrations.
		_ = old.conn.Close()
	}
	select {
	case h.register <- entry:
	case <-h.closeCh:
		// The hub closed between registration and enqueue: undo the map
		// update and close the connection so nothing leaks.
		h.mu.Lock()
		if current := h.agents[nodeID]; current == entry {
			delete(h.agents, nodeID)
		}
		h.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		return 0
	}
	return entry.generation
}

// Unregister removes an agent connection only if it is still the current
// generation for its node.
func (h *Hub) Unregister(nodeID string, conn agentSocket) {
	select {
	case h.unregister <- unregisterRequest{nodeID: nodeID, conn: conn, generation: currentGeneration(h, nodeID, conn)}:
	case <-h.closeCh:
	}
}

func currentGeneration(h *Hub, nodeID string, conn agentSocket) uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if current := h.agents[nodeID]; current != nil && current.conn == conn {
		return current.generation
	}
	return 0
}

// UnregisterGeneration removes an agent connection by generation identity.
func (h *Hub) UnregisterGeneration(nodeID string, generation uint64) {
	if generation == 0 {
		return
	}
	select {
	case h.unregister <- unregisterRequest{nodeID: nodeID, generation: generation}:
	case <-h.closeCh:
	}
}

// SendToAgent writes a typed JSON message to the current connection of an
// online agent. The write is linearized against connection replacement: it
// either commits to the current writer or returns an explicit error. It never
// silently delivers to a stale generation.
func (h *Hub) SendToAgent(nodeID string, msg models.WSMessage) error {
	h.mu.RLock()
	if h.isClosedLocked() {
		h.mu.RUnlock()
		return ErrHubClosed
	}
	agent := h.agents[nodeID]
	if agent == nil || agent.conn == nil {
		h.mu.RUnlock()
		return fmt.Errorf("%w: %s", ErrAgentOffline, nodeID)
	}
	agent.writeMu.Lock()
	if agent.fenced {
		agent.writeMu.Unlock()
		h.mu.RUnlock()
		return fmt.Errorf("%w: %s", ErrStaleConnection, nodeID)
	}
	if deadlineSetter, ok := agent.conn.(writeDeadlineSetter); ok {
		_ = deadlineSetter.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}
	err := agent.conn.WriteJSON(msg)
	agent.writeMu.Unlock()
	h.mu.RUnlock()
	return err
}

// CompleteHandshake marks the current connection of a node as handshake-ready
// (stream/watermark verified). Returns false when the given generation is no
// longer current.
func (h *Hub) CompleteHandshake(nodeID string, generation uint64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.isClosedLocked() {
		return false
	}
	current := h.agents[nodeID]
	if current == nil || current.generation != generation || current.fenced {
		return false
	}
	current.handshakeDone = true
	return true
}

// HandshakeReady reports whether the current connection of a node has
// completed the stream/watermark handshake, or whether the handshake grace
// period has expired. Control commands must only be sent through SendToAgent
// when this is true (and the store control state is OK).
//
// The grace-period fallback is deliberate: agents running a protocol older
// than the handshake never send one, and without this fallback the server
// would withhold their agent-update commands forever (they can only be
// upgraded by receiving exactly that command). New-protocol agents handshake
// immediately on connect, so they are unaffected; handshake gating still
// applies for the full grace period.
func (h *Hub) HandshakeReady(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.isClosedLocked() {
		return false
	}
	current := h.agents[nodeID]
	if current == nil || current.fenced {
		return false
	}
	if current.handshakeDone {
		return true
	}
	return time.Now().After(current.handshakeDeadline)
}

// CurrentGeneration returns the generation of the current node connection, or
// zero when the node is offline or the hub is closed.
func (h *Hub) CurrentGeneration(nodeID string) uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.isClosedLocked() {
		return 0
	}
	current := h.agents[nodeID]
	if current == nil || current.fenced {
		return 0
	}
	return current.generation
}

// IsCurrent reports whether the given generation is still the current
// connection of the node. Only current generations may submit status, metrics,
// heartbeats or online/offline events.
func (h *Hub) IsCurrent(nodeID string, generation uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.isClosedLocked() {
		return false
	}
	current := h.agents[nodeID]
	return current != nil && !current.fenced && current.generation == generation
}

// IsOnline reports whether a node is currently connected.
func (h *Hub) IsOnline(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return !h.isClosedLocked() && h.agents[nodeID] != nil
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

// Broadcast sends a message to all dashboard subscribers. After Close it is a
// no-op; it never blocks the caller.
func (h *Hub) Broadcast(msg models.WSMessage) {
	select {
	case h.broadcast <- msg:
	default:
		// broadcast queue full; drop rather than block
	}
}

// Subscribe adds a channel to receive broadcasted messages.
func (h *Hub) Subscribe(ch chan models.WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.isClosedLocked() {
		return
	}
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

// Done returns a channel closed when the hub begins shutdown. WebSocket
// handlers use it to terminate hijacked connections that HTTP Shutdown cannot
// close itself.
func (h *Hub) Done() <-chan struct{} {
	return h.closeCh
}

// Close fences and closes all agent connections, stops the event loop and
// makes every entry point return quickly. Safe to call multiple times.
func (h *Hub) Close() {
	h.closeOnce.Do(func() {
		h.mu.Lock()
		stale := make([]agentSocket, 0, len(h.agents))
		for _, agent := range h.agents {
			agent.fenced = true
			if agent.conn != nil {
				stale = append(stale, agent.conn)
			}
		}
		h.agents = make(map[string]*agentConn)
		h.subscribers = nil
		h.mu.Unlock()
		for _, conn := range stale {
			_ = conn.Close()
		}
		close(h.closeCh)
	})
}

func (h *Hub) isClosedLocked() bool {
	select {
	case <-h.closeCh:
		return true
	default:
		return false
	}
}
