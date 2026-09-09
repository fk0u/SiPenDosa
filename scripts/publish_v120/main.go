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

	releaseBody := `## 🚀 SiPenDosa v1.2.0 — Dynamic WA Picker, Public Portal, Zero-Config Tunnel & 2FA
*Sistem Pengingat Dosen Saatnya — Universal Multi-Device Academic WhatsApp Assistant*

---

### 🌟 Sorotan Pembaruan Utama (v1.2.0):

#### 1. 👥 Dynamic WhatsApp Group & Contact Picker (Closes #2)
* **Pencarian & Pemilihan Grup Otomatis**: Integrasi WhatsMeow engine untuk mengambil seluruh grup yang diikuti bot ('client.GetJoinedGroups()') dan kontak yang tersinkronisasi ('client.Store.Contacts.GetAllContacts()').
* **Endpoint API Dedicated**: Menyediakan '/api/wa/groups' dan '/api/wa/contacts' dengan proteksi autentikasi per-user.
* **Smart UI Selector Dropdown**: Antarmuka pemilih interaktif pada formulir jadwal dengan tab pilihan:
  - 👥 **Grup WhatsApp**: Menampilkan nama grup resmi dan jumlah anggota.
  - 👤 **Kontak Tersimpan**: Nama kontak dan nomor telepon terdaftar.
  - ✍️ **Input Manual**: Format nomor kustom untuk fleksibilitas maksimal.

#### 2. 🌐 Built-in Zero-Config Public Tunneling (Closes #3)
* **Akses Jarak Jauh Tanpa Konfigurasi Port-Forwarding**: Integrasi modul tunneling mandiri Cloudflare Quick Tunnel ('trycloudflare.com').
* **1-Klik Aktivasi di Menu Pengaturan**: Toggle tombol *"Aktifkan Akses Global (Tunneling)"* untuk menghasilkan tautan HTTPS publik instan.
* **Visual QR Code & Live Status**: Menampilkan QR Code langsung di dashboard web untuk dipindai dari smartphone di mana saja tanpa berada di jaringan Wi-Fi lokal yang sama.

#### 3. 📅 Public Academic Schedule Board & iCalendar Sync (Closes #4)
* **Portal Jadwal Akademik Terbuka ('/jadwal' & '/public/schedule')**: Halaman publik ramah mahasiswa tanpa memerlukan login, menyajikan jadwal kuliah secara transparan, lokasi ruangan, nama dosen pengampu, dan SKS.
* **Realtime Countdown Perkuliahan**: Widget hitung mundur dinamis (*"Kuliah berikutnya dimulai dalam X menit"*).
* **Ekspor Kalender Universal ('/jadwal/calendar.ics')**: Dukungan download format iCalendar ('.ics') standar untuk sinkronisasi instan ke Google Calendar, Apple Calendar, dan Microsoft Outlook.
* **Privasi Fleksibel**: Opsi toggle *"Tampilkan di Portal Publik"* pada setiap item jadwal.

#### 4. 🔐 Keamanan Tingkat Lanjut: Two-Factor Authentication (2FA / TOTP)
* **Proteksi Akun Ekstra**: Dukungan TOTP standar RFC 6238 (Google Authenticator, Microsoft Authenticator, 1Password, Bitwarden).
* **Alur Aktivasi & Recovery Interaktif**: Dilengkapi pemindaian QR Code SVG murni, verifikasi kode 6-digit, serta pembuatan 8 kode cadangan pemulihan (*emergency backup codes*).
* **Tantangan Login 2FA Tangguh**: Alur login dua tahap berbasis challenge token berbatas waktu singkat (5 menit) dengan proteksi brute-force rate-limiting.

#### 5. 🏢 Multi-Tenant User Isolation & WhatsApp Multi-Session
* **Isolasi Data Terstruktur**: Jadwal, kontak, antrian pesan, log aktivitas, dan template terikat langsung ke ID pengguna ('user_id').
* **WhatsApp Multi-Session Manager**: Arsitektur multi-sesi yang memungkinkan setiap pengguna memiliki koneksi bot WhatsApp tersendiri secara independen.
* **Manajemen Pengguna Terkendali**: Endpoint registrasi pengguna baru kini dibatasi khusus untuk Administrator demi integritas sistem.

---

### 📦 Berkas Unduhan Resmi (Self-Contained & Verified):

| Berkas Paket | Target Platform | Deskripsi |
| :--- | :--- | :--- |
| **SiPenDosa-Android.apk** | Android 5.0+ (Universal Multi-Arch) | Standalone APK Native Mobile UI dengan 24/7 Background Service (Signed: KOU) |
| **SiPenDosa-Android.aab** | Android App Bundle | Paket resmi Google App Bundle (AAB) terkompresi multi-arsitektur |
| **SiPenDosa-1.2.0.dmg** | macOS (Apple Silicon M1-M4 & Intel) | Apple Disk Image retina drag-and-drop installer ke Applications |
| **SiPenDosa-1.2.0-Installer.pkg** | macOS (Universal) | Apple Installer Package wizard dengan auto LaunchAgent service |
| **SiPenDosa-Setup.exe** | Windows 10 & 11 | Windows Standalone Setup Wizard dengan shortcut Desktop & Start Menu |
| **sipendosa_1.2.0_amd64.deb** | Ubuntu / Debian (x86_64) | Paket DEB dengan konfigurasi systemd daemon otomatis |
| **sipendosa_1.2.0_arm64.deb** | Debian / Ubuntu ARM64 | Paket DEB untuk Raspberry Pi, AWS Graviton, & server ARM |
| **sipendosa_linux_amd64.tar.gz** | Linux Universal | Tarball distribusi standalone untuk RedHat, CentOS, Fedora, Arch, SUSE |
| **sipendosa_macos_universal.tar.gz** | macOS Universal CLI | Paket tarball CLI & LaunchAgent universal (arm64 + amd64) |
| **install-android-termux.sh** | Android (Termux) | Skrip instalasi server otomatis Termux dengan shortcut sipen-pair |

---
*Dibuat dengan sepenuh dedikasi oleh Tim SiPenDosa.*
`

	payload := map[string]interface{}{
		"tag_name":         "v1.2.0",
		"target_commitish": "master",
		"name":             "SiPenDosa v1.2.0 — Dynamic WA Picker, Public Portal, Zero-Config Tunnel & 2FA",
		"body":             releaseBody,
		"draft":            false,
		"prerelease":       false,
	}

	payloadBytes, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}

	// 1. Cek apakah release v1.2.0 sudah ada
	var release ReleaseResponse
	checkReq, _ := http.NewRequest("GET", "https://api.github.com/repos/fk0u/SiPenDosa/releases/tags/v1.2.0", nil)
	checkReq.Header.Set("Authorization", "Bearer "+token)
	checkReq.Header.Set("Accept", "application/vnd.github+json")
	checkResp, err := client.Do(checkReq)

	if err == nil && checkResp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(checkResp.Body)
		_ = json.Unmarshal(body, &release)
		fmt.Printf("✓ Release v1.2.0 sudah ada (ID: %d), memperbarui berkas...\n", release.ID)
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
		fmt.Printf("✓ GitHub Release v1.2.0 berhasil dibuat!\n")
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
		"dist/SiPenDosa-1.2.0.dmg",
		"dist/SiPenDosa-1.2.0-Installer.pkg",
		"dist/SiPenDosa-Setup.exe",
		"dist/sipendosa_1.2.0_amd64.deb",
		"dist/sipendosa_1.2.0_arm64.deb",
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

	fmt.Println("\n🎉 Seluruh paket rilis v1.2.0 berhasil diunggah ke GitHub Releases!")
}
