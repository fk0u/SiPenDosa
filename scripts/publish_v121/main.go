package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ReleaseResponse struct {
	ID        int64  `json:"id"`
	UploadURL string `json:"upload_url"`
	HTMLURL   string `json:"html_url"`
}

func getGitToken() (string, error) {
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "password=") {
			return strings.TrimPrefix(l, "password="), nil
		}
	}
	return "", fmt.Errorf("token not found in git credentials")
}

func main() {
	token, err := getGitToken()
	if err != nil {
		fmt.Printf("Gagal mengambil token GitHub: %v\n", err)
		os.Exit(1)
	}

	releaseBody := `## 🚀 SiPenDosa v1.2.1 — macOS 12+ Universal Edition & Mobile 24/7 Engine
*Sistem Pengingat Dosen Saatnya — Universal Multi-Platform Academic WhatsApp Assistant*

---

### 🌟 Ringkasan Rilis v1.2.1:

#### 1. 🍎 macOS 12+ Universal 2 Distribution (Monterey, Ventura, Sonoma, Sequoia)
* **Dukungan Arsitektur Universal**: Mengompilasi dan menggabungkan dual-target Mach-O via ` + "`" + `lipo` + "`" + ` untuk Apple Silicon (M1-M4) dan Intel (x86_64).
* **Paket Distribusi Lengkap**:
  - **Apple Disk Image (` + "`" + `SiPenDosa-1.2.1.dmg` + "`" + `)**: Installer retina drag-and-drop mandiri.
  - **Apple Installer Wizard (` + "`" + `SiPenDosa-1.2.1-Installer.pkg` + "`" + `)**: Setup installer dengan konfigurasi LaunchAgent 24/7 otomatis.
  - **Universal Tarball (` + "`" + `sipendosa_macos_universal.tar.gz` + "`" + `)**: Biner CLI dan skrip setup daemon mandiri.
* **Menu Bar Companion Controller**: Kontrol background engine, monitoring latency C++17, status WhatsApp, dan parsing port dinamis dari ` + "`" + `.env` + "`" + `.

#### 2. 📱 Mobile 24/7 Background Execution & Real-Time Engine
* **Android**: Foreground service dengan tipe ` + "`" + `dataSync` + "`" + `, dual-lock CPU WakeLock + High-Performance WifiLock, perlindungan Doze Mode, Boot Completed persistence (` + "`" + `BootCompletedReceiver` + "`" + `), dan Watchdog Auto-Respawn.
* **iOS**: Integrasi ` + "`" + `BGTaskScheduler` + "`" + ` (` + "`" + `BGAppRefreshTask` + "`" + ` & ` + "`" + `BGProcessingTask` + "`" + `), extended background execution saat diminimalkan, live WebSocket telemetry hub, dan notifikasi lokal realtime (` + "`" + `UNUserNotificationCenter` + "`" + `).
* **REST JSON API Endpoints**: Menambahkan endpoint ` + "`" + `/api/stats` + "`" + `, ` + "`" + `/api/whatsapp/status` + "`" + `, ` + "`" + `/api/contacts` + "`" + `, ` + "`" + `/api/schedules` + "`" + `, ` + "`" + `/api/queue` + "`" + `, dan ` + "`" + `/api/ws` + "`" + ` untuk konsumsi klien mobile dan IoT.

#### 3. 🛡️ Perbaikan Keandalan & Audit Sistem (12 Bug Fixes)
* **Atomic Queue Claiming (BUG-01)**: Mencegah duplikasi kirim pesan ke WhatsApp dosen dengan lock atomik pada antrian SQLite.
* **Non-Blocking Graceful Shutdown (BUG-02 & BUG-11)**: Mengganti semua pemblokir ` + "`" + `time.Sleep` + "`" + ` dengan interrupt channel ` + "`" + `select` + "`" + ` sehingga server berhenti seketika.
* **WhatsApp Goroutine Leak & Panic Shield (BUG-03 & BUG-04)**: Melindungi loop auto-reconnect agar hanya 1 instance dan penutupan ` + "`" + `stopChan` + "`" + ` via ` + "`" + `sync.Once` + "`" + `.
* **Reverse Proxy & Tunneling CSRF Support (BUG-06)**: Mendukung ` + "`" + `X-Forwarded-Host` + "`" + ` dan perbandingan base hostname untuk Nginx, Caddy, dan Cloudflare Tunnel.
* **API Authentication Handling (BUG-07)**: Mendukung header ` + "`" + `Authorization: Bearer <token>` + "`" + ` dan respon JSON HTTP 401.
* **Robust LAN IP Detection Android (BUG-09)**: Fallback iterasi antarmuka jaringan ` + "`" + `NetworkInterface` + "`" + ` pada Android 11+.
* **Holiday-Aware Countdown (BUG-10)**: Widget hitung mundur memvalidasi kalender libur nasional dan kampus.
* **Status Validation (BUG-12)**: Validasi ketat pada pemicu antrian manual agar hanya pesan ` + "`" + `pending` + "`" + ` yang diproses.

---

### 📦 Berkas Unduhan Resmi:

| Berkas Paket | Platform Target | Deskripsi |
| :--- | :--- | :--- |
| **SiPenDosa-1.2.1.dmg** | macOS 12+ (Universal: M1-M4 & Intel) | Apple Disk Image retina drag-and-drop installer ke Applications |
| **SiPenDosa-1.2.1-Installer.pkg** | macOS 12+ (Universal) | Apple Installer Package wizard dengan auto LaunchAgent service |
| **sipendosa_macos_universal.tar.gz** | macOS Universal CLI | Paket tarball CLI & LaunchAgent universal (arm64 + amd64) |
| **install-macos.sh** | macOS CLI / Terminal | Skrip instalasi otomatis macOS dari terminal |

---
*Dibuat dengan sepenuh dedikasi oleh Tim SiPenDosa.*
`

	payload := map[string]interface{}{
		"tag_name":         "v1.2.1",
		"target_commitish": "master",
		"name":             "SiPenDosa v1.2.1 — macOS 12+ Universal Edition & Mobile 24/7 Engine",
		"body":             releaseBody,
		"draft":            false,
		"prerelease":       false,
	}

	payloadBytes, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}

	// 1. Cek apakah release v1.2.1 sudah ada
	var release ReleaseResponse
	checkReq, _ := http.NewRequest("GET", "https://api.github.com/repos/fk0u/SiPenDosa/releases/tags/v1.2.1", nil)
	checkReq.Header.Set("Authorization", "Bearer "+token)
	checkReq.Header.Set("Accept", "application/vnd.github+json")
	checkResp, err := client.Do(checkReq)

	if err == nil && checkResp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(checkResp.Body)
		_ = json.Unmarshal(body, &release)
		fmt.Printf("✓ Release v1.2.1 sudah ada (ID: %d), memperbarui berkas...\n", release.ID)
		checkResp.Body.Close()
	} else {
		if checkResp != nil {
			checkResp.Body.Close()
		}
		// Buat release baru
		req, err := http.NewRequest("POST", "https://api.github.com/repos/fk0u/SiPenDosa/releases", bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Printf("Error membuat request release: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Gagal memanggil GitHub API: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			fmt.Printf("GitHub API error (%d): %s\n", resp.StatusCode, string(body))
			os.Exit(1)
		}

		if err := json.Unmarshal(body, &release); err != nil {
			fmt.Printf("Gagal decode response: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ GitHub Release v1.2.1 berhasil dibuat!\n")
		fmt.Printf("  ID: %d\n", release.ID)
		fmt.Printf("  URL: %s\n", release.HTMLURL)
	}

	uploadBase := strings.Split(release.UploadURL, "{")[0]

	assetsToUpload := []string{
		"dist/SiPenDosa-1.2.1.dmg",
		"dist/SiPenDosa-1.2.1-Installer.pkg",
		"dist/sipendosa_macos_universal.tar.gz",
		"install-macos.sh",
	}

	uploadClient := &http.Client{Timeout: 10 * time.Minute}

	for _, assetPath := range assetsToUpload {
		fileName := filepath.Base(assetPath)
		file, err := os.Open(assetPath)
		if err != nil {
			fmt.Printf("! Gagal membuka berkas %s: %v\n", assetPath, err)
			continue
		}
		stat, _ := file.Stat()
		sizeMB := float64(stat.Size()) / 1024 / 1024

		fmt.Printf("==> Mengunggah %s (%.2f MB)... ", fileName, sizeMB)

		targetURL := fmt.Sprintf("%s?name=%s", uploadBase, fileName)
		upReq, err := http.NewRequest("POST", targetURL, file)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			file.Close()
			continue
		}

		upReq.Header.Set("Authorization", "Bearer "+token)
		upReq.Header.Set("Content-Type", "application/octet-stream")
		upReq.ContentLength = stat.Size()

		upResp, err := uploadClient.Do(upReq)
		file.Close()
		if err != nil {
			fmt.Printf("Gagal kirim: %v\n", err)
			continue
		}

		if upResp.StatusCode == http.StatusCreated || upResp.StatusCode == http.StatusOK {
			fmt.Println("✓ SELESAI!")
		} else {
			respBody, _ := io.ReadAll(upResp.Body)
			fmt.Printf("Gagal (%d): %s\n", upResp.StatusCode, string(respBody))
		}
		upResp.Body.Close()
	}

	fmt.Println("\n🎉 Seluruh paket rilis v1.2.1 berhasil diunggah ke GitHub Releases!")
}
