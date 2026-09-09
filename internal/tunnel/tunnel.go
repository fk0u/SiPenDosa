package tunnel

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
)

// State represents the status of the public tunnel
type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateActive   State = "active"
	StateError    State = "error"
)

// Status represents the public tunnel runtime information
type Status struct {
	State     State  `json:"state"`
	URL       string `json:"url"`
	QRCode    string `json:"qr_code"`
	Error     string `json:"error"`
	StartedAt string `json:"started_at"`
}

// Manager controls the Cloudflare quick tunnel lifecycle
type Manager struct {
	binDir     string
	port       string
	state      State
	url        string
	qrCode     string
	errMessage string
	startedAt  time.Time
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

var urlRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

// NewManager creates a new tunnel manager
func NewManager(binDir, port string) *Manager {
	_ = os.MkdirAll(binDir, 0755)
	return &Manager{
		binDir: binDir,
		port:   port,
		state:  StateStopped,
	}
}

// Status returns current tunnel status
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	startedStr := ""
	if !m.startedAt.IsZero() {
		startedStr = m.startedAt.Format(time.RFC3339)
	}

	return Status{
		State:     m.state,
		URL:       m.url,
		QRCode:    m.qrCode,
		Error:     m.errMessage,
		StartedAt: startedStr,
	}
}

// Start launches the cloudflared quick tunnel daemon
func (m *Manager) Start(localPort string) error {
	m.mu.Lock()
	if m.state == StateActive || m.state == StateStarting {
		m.mu.Unlock()
		return nil
	}
	if localPort != "" {
		m.port = localPort
	}
	m.state = StateStarting
	m.url = ""
	m.qrCode = ""
	m.errMessage = ""
	m.mu.Unlock()

	binaryPath, err := m.ensureCloudflared()
	if err != nil {
		m.mu.Lock()
		m.state = StateError
		m.errMessage = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("cloudflared binary not available: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binaryPath, "tunnel", "--url", fmt.Sprintf("http://127.0.0.1:%s", m.port), "--no-autoupdate")

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		m.mu.Lock()
		m.state = StateError
		m.errMessage = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("failed to open stderr pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		m.mu.Lock()
		m.state = StateError
		m.errMessage = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("failed to open stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		m.mu.Lock()
		m.state = StateError
		m.errMessage = err.Error()
		m.mu.Unlock()
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}

	m.mu.Lock()
	m.cmd = cmd
	m.cancel = cancel
	m.startedAt = time.Now()
	m.mu.Unlock()

	go m.monitorOutput(io.MultiReader(stdoutPipe, stderrPipe))

	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.state == StateActive || m.state == StateStarting {
			m.state = StateStopped
			m.url = ""
			m.qrCode = ""
		}
		m.mu.Unlock()
	}()

	return nil
}

// Stop terminates the running tunnel
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		m.cmd = nil
	}

	m.state = StateStopped
	m.url = ""
	m.qrCode = ""
	m.errMessage = ""
	return nil
}

func (m *Manager) monitorOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		slog.Debug("cloudflared log", "line", line)

		if m.url == "" {
			match := urlRegex.FindString(line)
			if match != "" {
				qrBytes, err := qrcode.Encode(match, qrcode.Medium, 256)
				var qrDataURL string
				if err == nil {
					qrDataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrBytes)
				}

				m.mu.Lock()
				m.url = match
				m.qrCode = qrDataURL
				m.state = StateActive
				m.mu.Unlock()

				slog.Info("Cloudflare Quick Tunnel established!", "url", match)
			}
		}
	}
}

func (m *Manager) ensureCloudflared() (string, error) {
	// 1. Look in system PATH
	if p, err := exec.LookPath("cloudflared"); err == nil {
		return p, nil
	}

	// 2. Look in local binDir
	localBinName := "cloudflared"
	if runtime.GOOS == "windows" {
		localBinName = "cloudflared.exe"
	}
	localPath := filepath.Join(m.binDir, localBinName)
	if fi, err := os.Stat(localPath); err == nil && !fi.IsDir() {
		_ = os.Chmod(localPath, 0755)
		return localPath, nil
	}

	// 3. Download standalone binary
	downloadURL, err := getCloudflaredDownloadURL()
	if err != nil {
		return "", err
	}

	slog.Info("Downloading cloudflared standalone binary...", "url", downloadURL, "target", localPath)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected download status %d", resp.StatusCode)
	}

	outFile, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed writing cloudflared binary: %w", err)
	}

	_ = os.Chmod(localPath, 0755)
	slog.Info("cloudflared binary ready", "path", localPath)
	return localPath, nil
}

func getCloudflaredDownloadURL() (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	switch osName {
	case "darwin":
		if arch == "arm64" {
			return "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-arm64.tgz", nil
		}
		return "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-amd64.tgz", nil
	case "linux":
		if arch == "arm64" {
			return "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64", nil
		}
		return "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64", nil
	case "windows":
		return "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe", nil
	default:
		return "", errors.New("unsupported OS for automated cloudflared download")
	}
}
