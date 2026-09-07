package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"sipen/internal/version"
)

// GitHubRelease represents the release object returned by GitHub API
type GitHubRelease struct {
	ID          int64         `json:"id"`
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset represents a single downloadable binary/installer asset
type GitHubAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

// CheckResult represents the outcome of an update check
type CheckResult struct {
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	HasUpdate      bool      `json:"has_update"`
	ReleaseTitle   string    `json:"release_title"`
	ReleaseNotes   string    `json:"release_notes"`
	PublishedAt    string    `json:"published_at"`
	HTMLURL        string    `json:"html_url"`
	ApkURL         string    `json:"apk_url"`
	DownloadURL    string    `json:"download_url"`
	AssetName      string    `json:"asset_name"`
	AssetSize      int64     `json:"asset_size"`
	CheckedAt      time.Time `json:"checked_at"`
}

// Manager handles checking and applying updates
type Manager struct {
	mu          sync.Mutex
	cachedCheck *CheckResult
	cacheExpiry time.Time
	httpClient  *http.Client
}

// NewManager creates a new updater manager
func NewManager() *Manager {
	return &Manager{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// CheckUpdate queries GitHub for the latest release and compares with current version
func (m *Manager) CheckUpdate(ctx context.Context, force bool) (*CheckResult, error) {
	m.mu.Lock()
	if !force && m.cachedCheck != nil && time.Now().Before(m.cacheExpiry) {
		res := *m.cachedCheck
		m.mu.Unlock()
		return &res, nil
	}
	m.mu.Unlock()

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", version.GitRepoOwner, version.GitRepoName)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create update check request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "SiPenDosa-Updater/"+version.CurrentVersion)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api returned status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release payload: %w", err)
	}

	latestClean := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	currentClean := strings.TrimPrefix(strings.TrimSpace(version.CurrentVersion), "v")

	hasUpdate := CompareVersions(latestClean, currentClean) > 0

	var apkURL string
	var targetDownloadURL string
	var targetAssetName string
	var targetAssetSize int64

	// Find assets
	for _, asset := range release.Assets {
		nameLower := strings.ToLower(asset.Name)
		if strings.HasSuffix(nameLower, ".apk") {
			apkURL = asset.BrowserDownloadURL
		}

		// Match current platform
		if matchPlatformAsset(nameLower, runtime.GOOS, runtime.GOARCH) {
			targetDownloadURL = asset.BrowserDownloadURL
			targetAssetName = asset.Name
			targetAssetSize = asset.Size
		}
	}

	// Fallback download url if no specific asset matches
	if targetDownloadURL == "" {
		if apkURL != "" {
			targetDownloadURL = apkURL
			targetAssetName = "SiPenDosa-Android.apk"
		} else {
			targetDownloadURL = release.HTMLURL
		}
	}

	res := &CheckResult{
		CurrentVersion: "v" + currentClean,
		LatestVersion:  "v" + latestClean,
		HasUpdate:      hasUpdate,
		ReleaseTitle:   release.Name,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt.Format("02 Jan 2006, 15:04 MST"),
		HTMLURL:        release.HTMLURL,
		ApkURL:         apkURL,
		DownloadURL:    targetDownloadURL,
		AssetName:      targetAssetName,
		AssetSize:      targetAssetSize,
		CheckedAt:      time.Now(),
	}

	m.mu.Lock()
	m.cachedCheck = res
	m.cacheExpiry = time.Now().Add(10 * time.Minute)
	m.mu.Unlock()

	return res, nil
}

// CompareVersions compares two semver version strings (e.g. "1.1.0" vs "1.0.9")
// Returns:
//
//	 1 if vA > vB
//	-1 if vA < vB
//	 0 if vA == vB
func CompareVersions(vA, vB string) int {
	partsA := parseSemver(vA)
	partsB := parseSemver(vB)

	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		if partsA[i] > partsB[i] {
			return 1
		}
		if partsA[i] < partsB[i] {
			return -1
		}
	}

	if len(partsA) > len(partsB) {
		return 1
	}
	if len(partsA) < len(partsB) {
		return -1
	}
	return 0
}

func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		// Strip any build suffix like -rc1
		clean := strings.Split(p, "-")[0]
		num, err := strconv.Atoi(clean)
		if err == nil {
			res = append(res, num)
		} else {
			res = append(res, 0)
		}
	}
	return res
}

func matchPlatformAsset(name, goos, goarch string) bool {
	switch goos {
	case "darwin":
		if strings.HasSuffix(name, ".dmg") || strings.HasSuffix(name, ".pkg") {
			return true
		}
		if strings.Contains(name, "macos") && strings.HasSuffix(name, ".tar.gz") {
			return true
		}
	case "windows":
		if strings.HasSuffix(name, ".exe") || strings.Contains(name, "windows") {
			return true
		}
	case "linux":
		if strings.HasSuffix(name, ".deb") {
			if goarch == "arm64" && strings.Contains(name, "arm64") {
				return true
			}
			if (goarch == "amd64" || goarch == "386") && strings.Contains(name, "amd64") {
				return true
			}
		}
		if strings.Contains(name, "linux") && strings.HasSuffix(name, ".tar.gz") {
			return true
		}
	}
	return false
}

// ApplyBinarySelfUpdate downloads and replaces current binary (for CLI/Server/Termux)
func (m *Manager) ApplyBinarySelfUpdate(ctx context.Context, downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("empty download URL")
	}

	// If Termux environment detected, can run updater script
	if os.Getenv("TERMUX_VERSION") != "" || strings.Contains(os.Getenv("PREFIX"), "com.termux") {
		cmd := exec.CommandContext(ctx, "bash", "-c", "curl -fsSL https://raw.githubusercontent.com/fk0u/SiPenDosa/master/install-android-termux.sh | bash")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("termux updater failed: %w (output: %s)", err, string(out))
		}
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate current executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SiPenDosa-Updater/"+version.CurrentVersion)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update download failed with status %d", resp.StatusCode)
	}

	tmpDir := filepath.Dir(exePath)
	tmpFile, err := os.CreateTemp(tmpDir, "sipen_update_*")
	if err != nil {
		// Fallback to os.TempDir
		tmpFile, err = os.CreateTemp("", "sipen_update_*")
		if err != nil {
			return fmt.Errorf("failed to create temporary update file: %w", err)
		}
	}
	tmpFilePath := tmpFile.Name()
	defer os.Remove(tmpFilePath)

	// If download is a tar.gz archive, extract binary
	if strings.HasSuffix(downloadURL, ".tar.gz") {
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		found := false
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				tmpFile.Close()
				return fmt.Errorf("tar read error: %w", err)
			}
			if hdr.Typeflag == tar.TypeReg && (hdr.Name == "sipen" || strings.HasSuffix(hdr.Name, "/sipen") || strings.HasSuffix(hdr.Name, ".exe")) {
				if _, err := io.Copy(tmpFile, tr); err != nil {
					tmpFile.Close()
					return fmt.Errorf("failed to write binary from tar: %w", err)
				}
				found = true
				break
			}
		}
		tmpFile.Close()
		if !found {
			return fmt.Errorf("binary 'sipen' not found inside downloaded archive")
		}
	} else if strings.HasSuffix(downloadURL, ".zip") {
		// Handle zip
		tmpZip, err := os.CreateTemp("", "sipen_zip_*")
		if err != nil {
			tmpFile.Close()
			return err
		}
		defer os.Remove(tmpZip.Name())
		_, _ = io.Copy(tmpZip, resp.Body)
		tmpZip.Close()

		zr, err := zip.OpenReader(tmpZip.Name())
		if err != nil {
			tmpFile.Close()
			return err
		}
		defer zr.Close()

		found := false
		for _, f := range zr.File {
			if strings.HasSuffix(f.Name, "sipen") || strings.HasSuffix(f.Name, "sipen.exe") {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				_, _ = io.Copy(tmpFile, rc)
				rc.Close()
				found = true
				break
			}
		}
		tmpFile.Close()
		if !found {
			return fmt.Errorf("binary not found in zip archive")
		}
	} else {
		// Direct binary
		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to save binary: %w", err)
		}
		tmpFile.Close()
	}

	// Make executable
	if err := os.Chmod(tmpFilePath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	// Atomic rename/replace
	oldBackup := exePath + ".old"
	_ = os.Remove(oldBackup)
	if err := os.Rename(exePath, oldBackup); err != nil {
		// On windows or permission error, try copying
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := os.Rename(tmpFilePath, exePath); err != nil {
		// Rollback
		_ = os.Rename(oldBackup, exePath)
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	_ = os.Remove(oldBackup)
	return nil
}
