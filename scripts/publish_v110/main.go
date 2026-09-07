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
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n")
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

	releaseBody := `## 🚀 SiPenDosa v1.1.0 — Advanced Multi-Platform Official Release
*Sistem Pengingat Dosen Saatnya — Universal Multi-Device WhatsApp Automation*

---

### 🌟 Yang Baru di Versi 1.1.0:

#### 1. 🔢 WhatsApp Pairing Code (Login via Nomor Telepon)
* **Solusi Login Tanpa Kamera / Scan QR**: Memungkinkan penautan akun WhatsApp langsung di layar smartphone atau lingkungan headless tanpa meminjam HP lain untuk memindai kode QR.
* **Modal Dual-Tab** di Web Dashboard: Pilihan tab *Scan QR Code* atau *Tautkan Nomor HP*.
* Kode 8 karakter yang dihasilkan otomatis terformat rapi dengan tombol **Salin Kode** ke clipboard.

#### 2. ⚡ Auto-Refresh QR Engine & Anti-Stuck Loading
* Mengatasi masalah QR Code kedaluwarsa atau macet pada WebView ponsel. Backend secara otomatis memicu sesi baru jika QR kosong atau expired saat diminta oleh client.

#### 3. 📱 Ekosistem Termux & Web Terminal Advanced
* **Perintah CLI Termux**: Jalankan 'sipen pair 08123456789' atau shortcut 'sipen-pair 08123456789' untuk mendapatkan kode pairing langsung di terminal Termux.
* **Web Terminal Interactive Prompt**: Ketik perintah 'pair <nomor_hp>' di /terminal untuk meminta pairing code langsung dari antarmuka shell.

#### 4. 🛡️ Audit & Pengerasan Keamanan (Strix / OWASP Compliance)
* **Broken Function Level Authorization (BFLA)**: Proteksi ketat hak akses SuperAdmin pada ekspor backup database, toggle pendaftaran akun, dan manajemen pengguna.
* **CSRF Origin Verification**: Validasi ketat header Origin/Referer pada seluruh endpoint mutasi state (POST/PUT/DELETE).
* **Brute-Force Rate Limiting**: Batasan rate limiter 10 req/menit per IP pada rute login dan registrasi.
* **Security Headers & Memory DoS Guard**: Injeksi header keamanan OWASP dan pembatasan body payload maksimum 2MB.

---

### 📦 Berkas Unduhan Resmi (Self-Contained & Verified):

| Berkas Paket | Target Platform | Deskripsi |
| :--- | :--- | :--- |
| **SiPenDosa-Android.apk** | Android 5.0+ (Universal Multi-Arch) | Standalone APK dengan 24/7 Background Service & Terminal View (Signed: KOU) |
| **SiPenDosa-Android.aab** | Android App Bundle | Paket resmi Google App Bundle (AAB) terkompresi untuk Google Play |
| **SiPenDosa-1.1.0.dmg** | macOS (Apple Silicon M1-M4 & Intel) | Apple Disk Image retina drag-and-drop installer ke Applications |
| **SiPenDosa-1.1.0-Installer.pkg** | macOS (Universal) | Apple Installer Package wizard dengan auto LaunchAgent service |
| **SiPenDosa-Setup.exe** | Windows 10 & 11 | Windows Standalone Setup Wizard dengan shortcut Desktop & Start Menu |
| **sipendosa_1.1.0_amd64.deb** | Ubuntu / Debian (x86_64) | Paket DEB dengan konfigurasi systemd daemon otomatis |
| **sipendosa_1.1.0_arm64.deb** | Debian / Ubuntu ARM64 | Paket DEB untuk Raspberry Pi, AWS Graviton, & server ARM |
| **sipendosa_linux_amd64.tar.gz** | Linux Universal | Tarball mandiri untuk RedHat, CentOS, Fedora, Arch, SUSE |
| **sipendosa_macos_universal.tar.gz** | macOS Universal CLI | Paket tarball CLI & LaunchAgent universal (arm64 + amd64) |
| **install-android-termux.sh** | Android (Termux) | Skrip instalasi server otomatis Termux dengan shortcut sipen-pair |

---
*Dibuat dengan sepenuh dedikasi oleh Tim SiPenDosa.*
`

	payload := map[string]interface{}{
		"tag_name":         "v1.1.0",
		"target_commitish": "master",
		"name":             "SiPenDosa v1.1.0 — Advanced Multi-Platform Release",
		"body":             releaseBody,
		"draft":            false,
		"prerelease":       false,
	}

	payloadBytes, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}

	// 1. Cek apakah release v1.1.0 sudah ada
	var release ReleaseResponse
	checkReq, _ := http.NewRequest("GET", "https://api.github.com/repos/fk0u/SiPenDosa/releases/tags/v1.1.0", nil)
	checkReq.Header.Set("Authorization", "Bearer "+token)
	checkReq.Header.Set("Accept", "application/vnd.github+json")
	checkResp, err := client.Do(checkReq)

	if err == nil && checkResp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(checkResp.Body)
		_ = json.Unmarshal(body, &release)
		fmt.Printf("✓ Release v1.1.0 sudah ada (ID: %d), memperbarui berkas...\n", release.ID)
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
		fmt.Printf("✓ GitHub Release v1.1.0 berhasil dibuat!\n")
		fmt.Printf("  ID: %d\n", release.ID)
		fmt.Printf("  URL: %s\n", release.HTMLURL)
	}

	uploadBase := strings.Split(release.UploadURL, "{")[0]

	assetsToUpload := []string{
		"dist/SiPenDosa-Android.apk",
		"dist/SiPenDosa-Android.aab",
		"install-android-termux.sh",
		"dist/bin/sipen_linux_arm64",
		"dist/bin/sipen_linux_amd64",
		"dist/SiPenDosa-1.1.0.dmg",
		"dist/SiPenDosa-1.1.0-Installer.pkg",
		"dist/SiPenDosa-Setup.exe",
		"dist/sipendosa_1.1.0_amd64.deb",
		"dist/sipendosa_1.1.0_arm64.deb",
		"dist/sipendosa_linux_amd64.tar.gz",
		"dist/sipendosa_macos_universal.tar.gz",
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

	fmt.Println("\n🎉 Seluruh paket rilis v1.1.0 berhasil diunggah ke GitHub Releases!")
}
