package realtime

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all local / reverse-proxied origins
	},
}

// Event represents a message broadcast to WebSocket clients
type Event struct {
	Type    string      `json:"type"`    // 'wa_status', 'wa_qr', 'countdown', 'queue_update', 'toast'
	Payload interface{} `json:"payload"` // Detailed data
}

// Hub manages WebSocket clients and message broadcasting
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Event
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
	lastEvents map[string]Event // Cache last status & countdown for instant delivery on connect
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan Event, 64),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		lastEvents: make(map[string]Event),
	}
}

// Run starts the WebSocket event routing loop
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

			// Send cached state immediately upon connection
			h.mu.RLock()
			for _, ev := range h.lastEvents {
				_ = conn.WriteJSON(ev)
			}
			h.mu.RUnlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				_ = conn.Close()
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			// Cache persistent state
			if event.Type == "wa_status" || event.Type == "wa_qr" || event.Type == "countdown" {
				h.mu.Lock()
				h.lastEvents[event.Type] = event
				h.mu.Unlock()
			}

			h.mu.RLock()
			for conn := range h.clients {
				err := conn.WriteJSON(event)
				if err != nil {
					slog.Debug("Error writing to websocket client", "err", err)
					go func(c *websocket.Conn) {
						h.unregister <- c
					}(conn)
				}
			}
			h.mu.RUnlock()

		case <-ticker.C:
			// Heartbeat ping
			h.mu.RLock()
			for conn := range h.clients {
				_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					go func(c *websocket.Conn) {
						h.unregister <- c
					}(conn)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends an event to all connected dashboard clients
func (h *Hub) Broadcast(eventType string, payload interface{}) {
	ev := Event{
		Type:    eventType,
		Payload: payload,
	}
	select {
	case h.broadcast <- ev:
	default:
		slog.Warn("WebSocket broadcast buffer full, event dropped", "type", eventType)
	}
}

// BroadcastToast sends a toast popup event
func (h *Hub) BroadcastToast(level, message string) {
	h.Broadcast("toast", map[string]string{
		"level":   level, // 'success', 'info', 'warning', 'error'
		"message": message,
	})
}

// HandleWS upgrades HTTP connection to WebSocket
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade websocket", "err", err)
		return
	}

	h.register <- conn

	// Keep connection open and read messages (discard or handle ping/pong)
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
