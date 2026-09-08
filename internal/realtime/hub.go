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

// ClientConn wraps websocket.Conn with thread-safe write synchronization
type ClientConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *ClientConn) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteJSON(v)
}

func (c *ClientConn) WritePing() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

func (c *ClientConn) WritePong(appData []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteMessage(websocket.PongMessage, appData)
}

func (c *ClientConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Close()
}

// Hub manages WebSocket clients and message broadcasting
type Hub struct {
	clients    map[*ClientConn]bool
	broadcast  chan Event
	register   chan *ClientConn
	unregister chan *ClientConn
	mu         sync.RWMutex
	lastEvents map[string]Event // Cache last status & countdown for instant delivery on connect
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*ClientConn]bool),
		broadcast:  make(chan Event, 64),
		register:   make(chan *ClientConn),
		unregister: make(chan *ClientConn),
		lastEvents: make(map[string]Event),
	}
}

// Run starts the WebSocket event routing loop
func (h *Hub) Run() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			// Send cached state immediately upon connection
			h.mu.RLock()
			for _, ev := range h.lastEvents {
				_ = client.WriteJSON(ev)
			}
			h.mu.RUnlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				_ = client.Close()
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
			for client := range h.clients {
				err := client.WriteJSON(event)
				if err != nil {
					slog.Debug("Error writing to websocket client", "err", err)
					go func(c *ClientConn) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()

		case <-ticker.C:
			// Heartbeat ping
			h.mu.RLock()
			for client := range h.clients {
				if err := client.WritePing(); err != nil {
					go func(c *ClientConn) {
						h.unregister <- c
					}(client)
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

	client := &ClientConn{conn: conn}
	conn.SetPingHandler(func(appData string) error {
		return client.WritePong([]byte(appData))
	})

	h.register <- client

	// Keep connection open and read messages (discard or handle ping/pong)
	go func() {
		defer func() {
			h.unregister <- client
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
