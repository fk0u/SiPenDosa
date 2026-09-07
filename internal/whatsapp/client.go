package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"sipen/internal/banner"
	"sipen/internal/realtime"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	_ "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// State represents the current status of WhatsApp connection
type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateNeedQR       State = "need_qr"
	StateConnected    State = "connected"
	StateLoggedOut    State = "logged_out"
)

// Client wraps whatsmeow with connection tracking, auto-reconnect, and anti-ban presence simulation
type Client struct {
	client     *whatsmeow.Client
	container  *sqlstore.Container
	hub        *realtime.Hub
	dbPath     string
	port       string
	state       State
	currentQR   string
	pairingCode string
	phoneJID    string
	pushName    string
	mu          sync.RWMutex
	cancelFunc  context.CancelFunc
	ctx         context.Context
	stopChan    chan struct{}
}

// NewClient initializes the whatsmeow storage container and client wrapper
func NewClient(dbPath string, hub *realtime.Hub, port string) (*Client, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_foreign_keys=on", dbPath)
	container, err := sqlstore.New(context.Background(), "sqlite", dsn, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize whatsmeow sqlstore: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get first device: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		container:  container,
		hub:        hub,
		dbPath:     dbPath,
		port:       port,
		state:      StateDisconnected,
		ctx:        ctx,
		cancelFunc: cancel,
		stopChan:   make(chan struct{}),
	}

	waClient := whatsmeow.NewClient(deviceStore, waLog.Noop)
	waClient.AddEventHandler(c.handleEvent)
	c.client = waClient

	return c, nil
}

// Start connects to WhatsApp or initiates the QR code login flow
func (c *Client) Start() error {
	c.mu.Lock()
	c.setState(StateConnecting)
	c.mu.Unlock()

	if c.client.Store.ID == nil {
		// No existing session, QR code login required
		qrChan, err := c.client.GetQRChannel(context.Background())
		if err != nil {
			c.setState(StateDisconnected)
			return fmt.Errorf("failed to get QR channel: %w", err)
		}

		err = c.client.Connect()
		if err != nil {
			c.setState(StateDisconnected)
			return fmt.Errorf("failed to connect: %w", err)
		}

		go c.listenQR(qrChan)
	} else {
		// Existing session found, connect directly
		err := c.client.Connect()
		if err != nil {
			c.setState(StateDisconnected)
			slog.Warn("Failed initial connect, auto-reconnector will retry", "err", err)
		}
	}

	go c.keepAliveLoop()
	return nil
}

func (c *Client) listenQR(qrChan <-chan whatsmeow.QRChannelItem) {
	for item := range qrChan {
		switch item.Event {
		case "code":
			banner.PrintQRHeader()
			// Print ASCII QR to terminal
			qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, os.Stdout)
			banner.PrintQRFooter(c.port)

			// Generate PNG Data URL for Web Dashboard
			pngBytes, err := qrcode.Encode(item.Code, qrcode.Medium, 256)
			if err == nil {
				dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
				c.mu.Lock()
				c.currentQR = dataURL
				c.setState(StateNeedQR)
				c.mu.Unlock()

				c.hub.Broadcast("wa_qr", dataURL)
			}

		case "timeout":
			slog.Warn("WhatsApp QR Code timed out")
			c.mu.Lock()
			c.currentQR = ""
			c.setState(StateDisconnected)
			c.mu.Unlock()

		case "success":
			slog.Info("WhatsApp QR login successful!")
			c.mu.Lock()
			c.currentQR = ""
			c.mu.Unlock()
		}
	}
}

func (c *Client) handleEvent(evt interface{}) {
	switch evt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.phoneJID = c.client.Store.ID.String()
		c.pushName = c.client.Store.PushName
		c.currentQR = ""
		c.setState(StateConnected)
		c.mu.Unlock()

		banner.PrintConnectedBox(c.phoneJID, c.pushName)
		c.hub.BroadcastToast("success", "WhatsApp berhasil terhubung!")

	case *events.Disconnected:
		slog.Warn("WhatsApp client disconnected")
		c.mu.Lock()
		c.setState(StateDisconnected)
		c.mu.Unlock()

		c.hub.BroadcastToast("warning", "WhatsApp terputus, mencoba menyambung ulang...")

	case *events.LoggedOut:
		slog.Warn("WhatsApp client logged out from phone")
		c.mu.Lock()
		c.phoneJID = ""
		c.currentQR = ""
		c.setState(StateLoggedOut)
		c.mu.Unlock()

		c.hub.BroadcastToast("error", "Sesi WhatsApp telah dikeluarkan dari perangkat.")
	}
}

func (c *Client) keepAliveLoop() {
	backoff := 5 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-c.stopChan:
			return
		case <-time.After(15 * time.Second):
			c.mu.RLock()
			isConnected := c.client != nil && c.client.IsConnected()
			hasSession := c.client != nil && c.client.Store.ID != nil
			c.mu.RUnlock()

			if hasSession && !isConnected {
				slog.Info("Auto-reconnect: trying to reconnect to WhatsApp...", "backoff", backoff)
				err := c.client.Connect()
				if err != nil {
					slog.Warn("Auto-reconnect failed", "err", err)
					time.Sleep(backoff)
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				} else {
					slog.Info("Auto-reconnect succeeded")
					backoff = 5 * time.Second
				}
			}
		}
	}
}

// SendMessage sends a text message with anti-ban presence simulation
func (c *Client) SendMessage(ctx context.Context, to string, message string) error {
	c.mu.RLock()
	client := c.client
	isConnected := client != nil && client.IsConnected()
	c.mu.RUnlock()

	if !isConnected {
		return errors.New("whatsapp client is not connected")
	}

	jid, err := FormatJID(to)
	if err != nil {
		return fmt.Errorf("invalid recipient JID (%s): %w", to, err)
	}

	// 1. Emulate user presence: mark online
	_ = client.SendPresence(ctx, types.PresenceAvailable)

	// 2. Emulate user typing (composing)
	_ = client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	// Random jitter typing delay between 1500ms and 2500ms
	jitterMs := 1500 + rand.Intn(1000)
	select {
	case <-time.After(time.Duration(jitterMs) * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	// 3. Send message
	msg := &waE2E.Message{
		Conversation: proto.String(message),
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	// 4. Mark typing paused
	_ = client.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)

	if err != nil {
		return fmt.Errorf("failed to send message via whatsmeow: %w", err)
	}

	slog.Info("WhatsApp message delivered", "to", jid.String(), "msg_id", resp.ID)
	return nil
}

// Disconnect disconnects the client without logging out
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect()
		c.setState(StateDisconnected)
	}
}

// Reconnect forces reconnection or requests new QR
func (c *Client) Reconnect() error {
	c.Disconnect()
	return c.Start()
}

// Logout disconnects and clears credentials
func (c *Client) Logout() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		if c.client.IsConnected() {
			_ = c.client.Logout(context.Background())
		}
		c.client.Disconnect()
		c.phoneJID = ""
		c.currentQR = ""
		c.setState(StateLoggedOut)
	}
	return nil
}

// Close gracefully terminates client
func (c *Client) Close() {
	close(c.stopChan)
	c.cancelFunc()
	c.Disconnect()
	_ = c.container.Close()
}

func (c *Client) setState(s State) {
	c.state = s
	c.hub.Broadcast("wa_status", map[string]interface{}{
		"state":     string(s),
		"phone":     c.phoneJID,
		"push_name": c.pushName,
	})
}

// Status returns current WhatsApp connection info
func (c *Client) Status() (State, string, string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state, c.phoneJID, c.pushName, c.currentQR
}

// PairPhone requests an 8-digit WhatsApp pairing code for the given phone number.
func (c *Client) PairPhone(ctx context.Context, phone string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return "", errors.New("whatsapp client belum diinisialisasi")
	}

	if c.client.IsConnected() && c.client.Store.ID != nil {
		return "", errors.New("whatsapp sudah terhubung dengan sesi aktif")
	}

	// Normalisasi nomor HP: hilangkan +, -, spasi, dll.
	re := regexp.MustCompile(`[^\d]`)
	cleaned := re.ReplaceAllString(phone, "")
	if strings.HasPrefix(cleaned, "08") {
		cleaned = "628" + cleaned[2:]
	} else if strings.HasPrefix(cleaned, "8") {
		cleaned = "62" + cleaned
	}

	if len(cleaned) < 9 {
		return "", fmt.Errorf("nomor telepon terlalu pendek: %s", phone)
	}

	if !c.client.IsConnected() {
		if err := c.client.Connect(); err != nil {
			return "", fmt.Errorf("gagal terhubung ke server WhatsApp: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	code, err := c.client.PairPhone(ctx, cleaned, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("gagal meminta kode pairing: %w", err)
	}

	c.pairingCode = code
	c.currentQR = ""
	c.setState(StateNeedQR)

	c.hub.Broadcast("wa_pairing_code", map[string]string{
		"code":  code,
		"phone": cleaned,
	})

	slog.Info("WhatsApp pairing code generated", "phone", cleaned, "code", code)
	return code, nil
}

// GetPairingCode returns the active pairing code if any
func (c *Client) GetPairingCode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pairingCode
}

// FormatJID normalizes raw phone numbers or group links into WhatsApp JID
func FormatJID(target string) (types.JID, error) {
	target = strings.TrimSpace(target)

	// If already in JID format
	if strings.Contains(target, "@") {
		return types.ParseJID(target)
	}

	// Remove spaces, hyphens, and parentheses
	re := regexp.MustCompile(`[^\d]`)
	cleaned := re.ReplaceAllString(target, "")

	// Handle Indonesian local prefix '08' -> '628'
	if strings.HasPrefix(cleaned, "08") {
		cleaned = "62" + cleaned[1:]
	}

	if len(cleaned) < 8 {
		return types.EmptyJID, fmt.Errorf("nomor telepon terlalu pendek: %s", target)
	}

	return types.NewJID(cleaned, types.DefaultUserServer), nil
}
