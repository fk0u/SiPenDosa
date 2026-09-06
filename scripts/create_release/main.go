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
)

type ReleaseResponse struct {
	ID        int64  `json:"id"`
	TagName   string `json:"tag_name"`
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

	releaseBody := `## 🚀 SiPenDosa v1.0.0 — Unified Multi-Platform Official Release

Selamat datang di rilis perdana **SiPenDosa (Sistem Pengingat Dosen Saatnya)**!

> *"Asisten cerdas yang rela 'berdosa' demi mengingatkan dosen, agar mahasiswa tidak perlu merasa sungkan."*

---

### 📦 Berkas Instalasi Resmi (Siap Pasang Langsung):

| Berkas Distribusi | Arsitektur / Platform | Kegunaan |
| :--- | :--- | :--- |
| **SiPenDosa-Android.apk** | Android 8.0+ (Universal) | Standalone APK dengan Auto Background Service & Terminal View |
| **install-android-termux.sh** | Android (Termux) | Skrip instalasi otomatis Android Termux 24/7 |
| **sipen_linux_arm64** | Linux / Android Termux (ARM64) | Standalone raw binary engine |
| **sipen_linux_amd64** | Linux (x86_64) | Standalone raw binary engine |
| **SiPenDosa-1.0.0.dmg** | macOS (Apple Silicon M1-M4 + Intel) | Apple Disk Image Drag & Drop visual ke Applications |
| **SiPenDosa-1.0.0-Installer.pkg** | macOS (Apple Silicon + Intel) | Apple Installer Package Wizard + auto LaunchAgent daemon |
| **SiPenDosa-Setup.exe** | Windows 10 & 11 (64-bit) | Windows Setup Wizard mandiri (Desktop & Start Menu shortcut) |
| **sipendosa_1.0.0_amd64.deb** | Ubuntu / Debian (x86_64) | Paket DEB dengan auto systemd service |
| **sipendosa_1.0.0_arm64.deb** | Debian / Ubuntu ARM64 | Paket DEB untuk Raspberry Pi & Cloud ARM |
| **sipendosa_linux_amd64.tar.gz** | RedHat / CentOS / Fedora / Arch | Universal standalone tarball bundle |
| **sipendosa_macos_universal.tar.gz** | macOS Universal CLI | Standalone tarball bundle dengan install-macos.sh |

---

### 📱 Cara Instalasi Cepat Android (Termux):
Cukup buka Termux dan jalankan satu baris perintah berikut:
` + "```bash" + `
curl -fsSL https://raw.githubusercontent.com/fk0u/SiPenDosa/master/install-android-termux.sh | bash
` + "```" + `
Gunakan perintah ` + "`sipen-start`" + ` untuk menjalankan server, ` + "`sipen-bg`" + ` untuk mode latar belakang, dan ` + "`sipen-stop`" + ` untuk menghentikan.

---

### 🌟 Fitur Utama v1.0.0:
1. **WhatsApp Multi-Device Engine** dengan simulasi kehadiran manusia (*human presence simulation*) & jitter acak anti-ban.
2. **Smart Scheduler** dengan kepedulian kalender libur nasional dan evaluasi rentang jam kerja.
3. **Template Engine Dinamis** dengan Live Preview interaktif & riwayat versi.
4. **Interactive Terminal Mode** ('/terminal') untuk streaming log WhatsApp realtime & command prompt mirip Termux.
5. **PWA Mobile Ready**: Tampilan 100% responsif, ikon favicon lengkap, dan Web App Manifest.
6. **Zero-CGO Pure Go & SQLite WAL** untuk stabilitas tinggi tanpa dependensi eksternal.
`

	payload := map[string]interface{}{
		"tag_name":         "v1.0.0",
		"target_commitish": "master",
		"name":             "SiPenDosa v1.0.0 — Unified Multi-Platform Release",
		"body":             releaseBody,
		"draft":            false,
		"prerelease":       false,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.github.com/repos/fk0u/SiPenDosa/releases", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Error membuat request release: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
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

	var release ReleaseResponse
	if err := json.Unmarshal(body, &release); err != nil {
		fmt.Printf("Gagal decode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ GitHub Release berhasil dibuat!\n")
	fmt.Printf("  ID: %d\n", release.ID)
	fmt.Printf("  URL: %s\n", release.HTMLURL)

	// File-file yang akan diunggah
	assetsToUpload := []string{
		"dist/SiPenDosa-Android.apk",
		"install-android-termux.sh",
		"dist/bin/sipen_linux_arm64",
		"dist/bin/sipen_linux_amd64",
		"dist/SiPenDosa-1.0.0.dmg",
		"dist/SiPenDosa-1.0.0-Installer.pkg",
		"dist/SiPenDosa-Setup.exe",
		"dist/sipendosa_1.0.0_amd64.deb",
		"dist/sipendosa_1.0.0_arm64.deb",
		"dist/sipendosa_linux_amd64.tar.gz",
		"dist/sipendosa_macos_universal.tar.gz",
	}

	uploadBase := strings.Split(release.UploadURL, "{")[0]

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

		upResp, err := client.Do(upReq)
		file.Close()
		if err != nil {
			fmt.Printf("Gagal kirim: %v\n", err)
			continue
		}

		if upResp.StatusCode == http.StatusCreated || upResp.StatusCode == http.StatusOK {
			fmt.Printf("✓ SELESAI\n")
		} else {
			respBytes, _ := io.ReadAll(upResp.Body)
			fmt.Printf("GAGAL (%d): %s\n", upResp.StatusCode, string(respBytes))
		}
		upResp.Body.Close()
	}

	fmt.Println("\n=======================================================")
	fmt.Printf("🎉 Seluruh paket rilis berhasil dipublikasikan di GitHub!\n")
	fmt.Printf("Kunjungi: %s\n", release.HTMLURL)
	fmt.Println("=======================================================")
}
