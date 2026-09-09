package queue

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"sipen/internal/realtime"
	"sipen/internal/store"
	"sipen/internal/whatsapp"
)

// Manager manages message queue execution, rate limiting, and auto-retry
type Manager struct {
	store     *store.Store
	waManager *whatsapp.Manager
	hub       *realtime.Hub
	stopChan  chan struct{}
	trigger   chan struct{}
	mu        sync.Mutex
	running   bool
}

// NewManager initializes the message queue manager
func NewManager(s *store.Store, waMgr *whatsapp.Manager, hub *realtime.Hub) *Manager {
	return &Manager{
		store:     s,
		waManager: waMgr,
		hub:       hub,
		stopChan:  make(chan struct{}),
		trigger:   make(chan struct{}, 1),
	}
}

// Start begins the background queue worker
func (m *Manager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	go m.workerLoop()
}

// Stop stops the queue worker
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	m.running = false
	close(m.stopChan)
}

// TriggerProcessing signals the queue to check for pending messages immediately
func (m *Manager) TriggerProcessing() {
	select {
	case m.trigger <- struct{}{}:
	default:
	}
}

// Enqueue adds a message to the database queue and triggers worker
func (m *Manager) Enqueue(qm *store.QueueMessage) (*store.QueueMessage, error) {
	settings, err := m.store.GetSettings()
	if err == nil {
		qm.MaxRetries = settings.MaxRetries
	} else {
		qm.MaxRetries = 3
	}

	msg, err := m.store.CreateQueueMessage(qm)
	if err != nil {
		return nil, err
	}

	m.hub.Broadcast("queue_update", map[string]interface{}{
		"action": "enqueued",
		"id":     msg.ID,
		"status": msg.Status,
	})

	m.TriggerProcessing()
	return msg, nil
}

func (m *Manager) workerLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.processPending()
		case <-m.trigger:
			m.processPending()
		}
	}
}

func (m *Manager) processPending() {
	settings, err := m.store.GetSettings()
	if err != nil {
		slog.Error("Failed to fetch settings for queue processing", "err", err)
		return
	}

	now := time.Now()
	// Fetch up to 10 pending messages due now
	messages, err := m.store.GetPendingQueueMessages(now, 10)
	if err != nil {
		slog.Error("Failed to get pending queue messages", "err", err)
		return
	}

	for _, msg := range messages {
		m.processSingleMessage(msg, settings)
	}
}

func (m *Manager) processSingleMessage(msg store.QueueMessage, settings *store.Settings) {
	// Mark processing
	_ = m.store.UpdateQueueStatus(msg.ID, "processing", "", nil)
	m.hub.Broadcast("queue_update", map[string]interface{}{
		"action": "processing",
		"id":     msg.ID,
		"status": "processing",
	})

	// Check Dry Run: Global or Per-Schedule
	isDryRun := settings.GlobalDryRun
	if !isDryRun && msg.ScheduleID.Valid {
		sc, err := m.store.GetScheduleByID(msg.ScheduleID.Int64)
		if err == nil && sc.DryRun {
			isDryRun = true
		}
	}

	if isDryRun {
		now := time.Now()
		_ = m.store.UpdateQueueStatus(msg.ID, "dry_run", "[DRY RUN] Pesan disimulasikan tanpa pengiriman fisik", &now)
		m.store.AddActivityLog("queue", "Pesan Disimulasikan (Dry Run)",
			fmt.Sprintf("Tujuan: %s (%s)", msg.RecipientName, msg.RecipientJID))

		m.hub.Broadcast("queue_update", map[string]interface{}{
			"action": "completed",
			"id":     msg.ID,
			"status": "dry_run",
		})
		return
	}

	// Real WhatsApp Send
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	waClient, err := m.waManager.GetClient(msg.UserID)
	if err != nil {
		slog.Warn("Failed to obtain WhatsApp client for user", "user_id", msg.UserID, "err", err)
	} else {
		err = waClient.SendMessage(ctx, msg.RecipientJID, msg.Message)
	}
	if err != nil {
		slog.Warn("Failed to send queued WhatsApp message", "id", msg.ID, "err", err)

		if msg.RetryCount < msg.MaxRetries {
			// Exponential backoff retry: 1 min, 4 min, 9 min...
			backoffMinutes := (msg.RetryCount + 1) * (msg.RetryCount + 1)
			nextTime := time.Now().Add(time.Duration(backoffMinutes) * time.Minute)

			_ = m.store.IncrementRetry(msg.ID, nextTime, err.Error())
			m.store.AddActivityLog("queue", "Gagal Kirim - Dijadwalkan Ulang",
				fmt.Sprintf("ID: %d, Percobaan: %d, Next: %s, Error: %s",
					msg.ID, msg.RetryCount+1, nextTime.Format("15:04:05"), err.Error()))

			m.hub.Broadcast("queue_update", map[string]interface{}{
				"action": "retry_scheduled",
				"id":     msg.ID,
				"status": "pending",
			})
		} else {
			// Max retries reached
			_ = m.store.UpdateQueueStatus(msg.ID, "failed", err.Error(), nil)
			m.store.AddActivityLog("queue", "Pengiriman Gagal Permanen",
				fmt.Sprintf("ID: %d, Tujuan: %s, Error: %s", msg.ID, msg.RecipientName, err.Error()))

			m.hub.Broadcast("queue_update", map[string]interface{}{
				"action": "failed",
				"id":     msg.ID,
				"status": "failed",
			})
			m.hub.BroadcastToast("error", fmt.Sprintf("Gagal mengirim pesan ke %s", msg.RecipientName))
		}
		return
	}

	// Sent successfully
	sentAt := time.Now()
	_ = m.store.UpdateQueueStatus(msg.ID, "sent", "", &sentAt)
	m.store.AddActivityLog("queue", "Pesan Berhasil Terkirim",
		fmt.Sprintf("Tujuan: %s (%s)", msg.RecipientName, msg.RecipientJID))

	m.hub.Broadcast("queue_update", map[string]interface{}{
		"action": "sent",
		"id":     msg.ID,
		"status": "sent",
	})
	m.hub.BroadcastToast("success", fmt.Sprintf("Pesan berhasil terkirim ke %s", msg.RecipientName))

	// Anti-ban rate limiting jitter delay before next message
	minDelay := settings.RateLimitMinSec
	maxDelay := settings.RateLimitMaxSec
	if maxDelay <= minDelay {
		maxDelay = minDelay + 5
	}
	delaySec := minDelay + rand.Intn(maxDelay-minDelay+1)
	slog.Debug("Anti-ban rate limit delay", "sec", delaySec)
	time.Sleep(time.Duration(delaySec) * time.Second)
}

// ProcessImmediate forces sending of a specific queue message immediately
func (m *Manager) ProcessImmediate(id int64) error {
	msg, err := m.store.GetQueueMessageByID(id)
	if err != nil {
		return err
	}

	settings, err := m.store.GetSettings()
	if err != nil {
		return err
	}

	go m.processSingleMessage(*msg, settings)
	return nil
}
