package tunnel

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
	"strings"
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

	assetName := filepath.Base(downloadURL)
	slog.Info("Downloading cloudflared standalone binary...", "url", downloadURL, "target", localPath)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected download status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(m.binDir, "cloudflared_dl_*")
	if err != nil {
		tmpFile, err = os.CreateTemp("", "cloudflared_dl_*")
		if err != nil {
			return "", fmt.Errorf("failed creating temporary download file: %w", err)
		}
	}
	tmpFilePath := tmpFile.Name()
	defer os.Remove(tmpFilePath)

	hasher := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, hasher)
	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed downloading cloudflared: %w", err)
	}
	tmpFile.Close()

	// Verifikasi integritas terhadap SHA256 di catatan rilis resmi GitHub
	if expected, err := fetchExpectedCloudflaredChecksum(assetName); err == nil && expected != "" {
		actual := hex.EncodeToString(hasher.Sum(nil))
		if actual != expected {
			return "", fmt.Errorf("cloudflared sha256 mismatch for %s: got %s, want %s", assetName, actual, expected)
		}
		slog.Info("cloudflared checksum verified successfully", "asset", assetName, "sha256", actual)
	} else if err != nil {
		slog.Warn("Skipping cloudflared checksum validation (release notes query unavailable)", "err", err)
	}

	// Jika arsip tar.gz (.tgz / .tar.gz seperti pada macOS), dekompresi dan ekstrak biner
	if strings.HasSuffix(assetName, ".tgz") || strings.HasSuffix(assetName, ".tar.gz") {
		f, err := os.Open(tmpFilePath)
		if err != nil {
			return "", err
		}
		defer f.Close()

		gzr, err := gzip.NewReader(f)
		if err != nil {
			return "", fmt.Errorf("failed to decompress gzip archive: %w", err)
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		extracted := false
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("error reading tar archive: %w", err)
			}
			if hdr.Typeflag == tar.TypeReg && (hdr.Name == "cloudflared" || strings.HasSuffix(hdr.Name, "/cloudflared")) {
				outFile, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
				if err != nil {
					return "", fmt.Errorf("failed creating binary file: %w", err)
				}
				if _, err := io.Copy(outFile, tr); err != nil {
					outFile.Close()
					return "", fmt.Errorf("failed writing extracted binary: %w", err)
				}
				outFile.Close()
				extracted = true
				break
			}
		}
		if !extracted {
			return "", fmt.Errorf("binary 'cloudflared' not found inside archive %s", assetName)
		}
	} else {
		// Biner langsung (Linux & Windows)
		if err := copyExecutableFile(tmpFilePath, localPath); err != nil {
			return "", fmt.Errorf("failed installing cloudflared binary: %w", err)
		}
	}

	_ = os.Chmod(localPath, 0755)
	slog.Info("cloudflared binary ready", "path", localPath)
	return localPath, nil
}

// fetchExpectedCloudflaredChecksum mengambil SHA256 checksum dari catatan rilis resmi cloudflared di GitHub
func fetchExpectedCloudflaredChecksum(assetName string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/cloudflare/cloudflared/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "SiPenDosa-Tunnel/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var rel struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(rel.Body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && strings.TrimSpace(parts[0]) == assetName {
			return strings.ToLower(strings.TrimSpace(parts[1])), nil
		}
	}

	return "", fmt.Errorf("checksum entry not found in release body for %s", assetName)
}

func copyExecutableFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
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
