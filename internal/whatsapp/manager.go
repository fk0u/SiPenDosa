package whatsapp

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sipen/internal/realtime"
)

// Manager coordinates WhatsApp clients for multiple user accounts
type Manager struct {
	sessionsDir string
	hub         *realtime.Hub
	port        string
	clients     map[int64]*Client
	mu          sync.RWMutex
}

// NewManager creates a WhatsApp multi-account manager
func NewManager(sessionsDir string, hub *realtime.Hub, port string) *Manager {
	baseDir := sessionsDir
	if strings.HasSuffix(strings.ToLower(sessionsDir), ".db") {
		baseDir = filepath.Dir(sessionsDir)
	}
	if baseDir == "" {
		baseDir = "session"
	}
	_ = os.MkdirAll(baseDir, 0755)
	_ = os.MkdirAll(filepath.Join(baseDir, "users"), 0755)

	return &Manager{
		sessionsDir: baseDir,
		hub:         hub,
		port:        port,
		clients:     make(map[int64]*Client),
	}
}

// GetClient retrieves or initializes the WhatsApp client for a specific user ID
func (m *Manager) GetClient(userID int64) (*Client, error) {
	if userID <= 0 {
		userID = 1
	}

	m.mu.RLock()
	client, exists := m.clients[userID]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := m.clients[userID]; exists {
		return client, nil
	}

	// Resolve database path for this user
	var dbPath string
	if userID == 1 {
		// User 1 uses legacy session/whatsapp.db if exists, otherwise session/users/user_1.db
		legacyPath := filepath.Join(m.sessionsDir, "whatsapp.db")
		userPath := filepath.Join(m.sessionsDir, "users", "user_1.db")
		if _, err := os.Stat(legacyPath); err == nil {
			dbPath = legacyPath
		} else {
			dbPath = userPath
		}
	} else {
		dbPath = filepath.Join(m.sessionsDir, "users", fmt.Sprintf("user_%d.db", userID))
	}

	slog.Info("Initializing WhatsApp client for user", "user_id", userID, "db_path", dbPath)
	c, err := NewClient(dbPath, m.hub, m.port, userID)
	if err != nil {
		return nil, fmt.Errorf("failed creating whatsapp client for user %d: %w", userID, err)
	}

	m.clients[userID] = c

	// Auto-start WhatsApp connection in background
	go func() {
		if err := c.Start(); err != nil {
			slog.Warn("WhatsApp initial connect failed for user", "user_id", userID, "error", err)
		}
	}()

	return c, nil
}

// CloseClient disconnects and closes the client for a given user
func (m *Manager) CloseClient(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.clients[userID]; ok {
		c.Close()
		delete(m.clients, userID)
	}
}

// CloseAll cleanly terminates all active WhatsApp connections
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for uid, c := range m.clients {
		slog.Info("Closing WhatsApp client for user", "user_id", uid)
		c.Close()
	}
	m.clients = make(map[int64]*Client)
}
