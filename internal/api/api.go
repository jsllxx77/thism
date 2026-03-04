package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/thism-dev/thism/internal/hub"
	"github.com/thism-dev/thism/internal/models"
	"github.com/thism-dev/thism/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewRouter builds and returns the HTTP router.
// If frontendHandler is non-nil it is used as a fallback for unmatched routes;
// otherwise unmatched routes return 404.
func NewRouter(s *store.Store, h *hub.Hub, adminToken string, frontendHandler http.Handler) http.Handler {
	r := chi.NewRouter()

	// ---------------------------------------------------------------
	// WebSocket endpoints (auth handled inside each handler)
	// ---------------------------------------------------------------

	// Agent WebSocket: authenticates via node token (?token=)
	r.Get("/ws/agent", func(w http.ResponseWriter, req *http.Request) {
		handleAgentWS(w, req, s, h)
	})

	// Dashboard WebSocket: requires admin token
	r.Get("/ws/dashboard", func(w http.ResponseWriter, req *http.Request) {
		if !checkToken(req, adminToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handleDashboardWS(w, req, h)
	})

	// ---------------------------------------------------------------
	// REST API (all require admin token)
	// ---------------------------------------------------------------
	r.Group(func(r chi.Router) {
		r.Use(adminAuth(adminToken))

		r.Get("/api/nodes", func(w http.ResponseWriter, req *http.Request) {
			handleListNodes(w, req, s, h)
		})

		r.Post("/api/nodes/register", func(w http.ResponseWriter, req *http.Request) {
			handleRegisterNode(w, req, s)
		})

		r.Get("/api/nodes/{id}/metrics", func(w http.ResponseWriter, req *http.Request) {
			handleGetMetrics(w, req, s)
		})

		r.Get("/api/nodes/{id}/processes", func(w http.ResponseWriter, req *http.Request) {
			handleGetProcesses(w, req, s)
		})

		r.Get("/api/nodes/{id}/services", func(w http.ResponseWriter, req *http.Request) {
			handleGetServices(w, req, s)
		})
	})

	// ---------------------------------------------------------------
	// Fallback
	// ---------------------------------------------------------------
	if frontendHandler != nil {
		r.Handle("/*", frontendHandler)
	} else {
		r.Handle("/*", http.NotFoundHandler())
	}

	return r
}

// -----------------------------------------------------------------------
// Auth helpers
// -----------------------------------------------------------------------

// checkToken validates the bearer token from the Authorization header or
// the ?token= query parameter.
func checkToken(r *http.Request, expected string) bool {
	// Check Authorization header first.
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ") == expected
		}
	}
	// Fall back to query parameter.
	return r.URL.Query().Get("token") == expected
}

// adminAuth returns a middleware that enforces the admin token.
func adminAuth(adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !checkToken(r, adminToken) {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// -----------------------------------------------------------------------
// Token / ID generation
// -----------------------------------------------------------------------

// generateHex returns a random 32-character hex string (16 random bytes).
func generateHex() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// -----------------------------------------------------------------------
// JSON helpers
// -----------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// -----------------------------------------------------------------------
// REST handlers
// -----------------------------------------------------------------------

func handleListNodes(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	nodes, err := s.ListNodes()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Build a set of online node IDs for O(1) lookup.
	onlineSet := make(map[string]struct{})
	for _, id := range h.OnlineNodeIDs() {
		onlineSet[id] = struct{}{}
	}

	// Ensure we never serialise null for an empty list.
	result := make([]*models.Node, 0, len(nodes))
	for _, n := range nodes {
		_, n.Online = onlineSet[n.ID]
		result = append(result, n)
	}

	writeJSON(w, http.StatusOK, map[string]any{"nodes": result})
}

func handleRegisterNode(w http.ResponseWriter, r *http.Request, s *store.Store) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	id, err := generateHex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		return
	}
	token, err := generateHex()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
		return
	}

	node := &models.Node{
		ID:        id,
		Name:      req.Name,
		Token:     token,
		CreatedAt: time.Now().Unix(),
	}
	if err := s.UpsertNode(node); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": id, "token": token})
}

func handleGetMetrics(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	now := time.Now().Unix()
	from := now - 3600 // default: last hour
	to := now

	if v := r.URL.Query().Get("from"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			from = parsed
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			to = parsed
		}
	}

	rows, err := s.QueryMetrics(nodeID, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rows == nil {
		rows = []*store.MetricsRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"metrics": rows})
}

func handleGetProcesses(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	data, err := s.GetProcesses(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// data is already a JSON string; write it raw.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data))
}

func handleGetServices(w http.ResponseWriter, r *http.Request, s *store.Store) {
	nodeID := chi.URLParam(r, "id")
	checks, err := s.GetServiceChecks(nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if checks == nil {
		checks = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": checks})
}

// -----------------------------------------------------------------------
// WebSocket handlers
// -----------------------------------------------------------------------

func handleAgentWS(w http.ResponseWriter, r *http.Request, s *store.Store, h *hub.Hub) {
	token := r.URL.Query().Get("token")
	if token == "" {
		// Also accept bearer token in header for agent connections.
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	node, err := s.GetNodeByToken(token)
	if err != nil || node == nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.Register(node.ID, conn)
	defer func() {
		conn.Close()
		h.Unregister(node.ID)
	}()

	// Read loop: parse incoming MetricsPayload messages and persist them.
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var payload models.MetricsPayload
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		// Persist metrics.
		_ = s.InsertMetrics(node.ID, &payload)
		_ = s.UpdateLastSeen(node.ID)

		// Persist processes as a JSON string.
		if len(payload.Processes) > 0 {
			procJSON, err := json.Marshal(payload.Processes)
			if err == nil {
				_ = s.UpsertProcesses(node.ID, payload.TS, string(procJSON))
			}
		}

		// Persist service checks.
		for _, svc := range payload.Services {
			_ = s.UpsertServiceCheck(node.ID, svc.Name, svc.Status)
		}

		// Broadcast the raw metrics to dashboard subscribers.
		h.Broadcast(models.WSMessage{
			Type:    "metrics",
			Payload: payload,
		})
	}
}

func handleDashboardWS(w http.ResponseWriter, r *http.Request, h *hub.Hub) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan models.WSMessage, 32)
	h.Subscribe(ch)
	defer h.Unsubscribe(ch)

	for msg := range ch {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			break
		}
	}
}
