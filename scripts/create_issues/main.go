package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type IssueRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels,omitempty"`
}

type IssueResponse struct {
	ID      int64  `json:"id"`
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
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

	client := &http.Client{Timeout: 30 * time.Second}

	issues := []IssueRequest{
		{
			Title: "[Feature Request] Dynamic WhatsApp Group & Contact Picker Dropdown (Auto-Fetch via WhatsMeow)",
			Labels: []string{"enhancement"},
			Body: `## 💡 Latar Belakang & Masalah
Saat ini, ketika pengguna ingin mengatur pengingat otomatis untuk Grup WhatsApp perkuliahan (misalnya grup kelas atau kelompok praktikum), pengguna kesulitan mendapatkan **WhatsApp Group JID** (` + "`... @g.us`" + `) karena format JID grup WhatsApp tidak ditampilkan di aplikasi WhatsApp biasa.

## 🎯 Solusi yang Diusulkan
Menyediakan fitur **Group & Contact Picker Dropdown** otomatis pada form pembuatan Jadwal / Pengingat di Web UI dan Mobile UI:

1. **Backend Integration (WhatsMeow Engine)**:
   - Membuat endpoint ` + "`/api/whatsapp/groups`" + ` yang memanggil ` + "`client.GetJoinedGroups()`" + ` dari WhatsMeow.
   - Endpoint mengembalikan daftar grup yang diikuti akun:
     - ` + "`jid`" + `: string (contoh: ` + "`120363028394829102@g.us`" + `)
     - ` + "`name`" + `: string (contoh: "Kelas Pemrograman Web A")
     - ` + "`participants_count`" + `: int
     - ` + "`topic`" + `: string
   - Endpoint ` + "`/api/whatsapp/contacts`" + ` untuk sinkronisasi kontak dari ` + "`client.Store.Contacts.GetAllContacts()`" + `.

2. **Frontend UI/UX**:
   - Di modal/form *Tambah Jadwal*, input "Tujuan / Penerima" dilengkapi dengan searchable dropdown / combobox.
   - Opsi pilihan:
     - 👥 **Grup WhatsApp** (menampilkan ikon grup, nama grup, dan jumlah anggota).
     - 👤 **Kontak Tersimpan** (menampilkan nama dan nomor).
     - ✍️ **Input Manual / Nomor Baru** (untuk fleksibilitas input manual nomor telepon/JID).
   - Pengguna cukup memilih nama grup dari dropdown, dan sistem otomatis mengisi target JID grup di balik layar.

## 🚀 Kriteria Selesai (Acceptance Criteria)
- [ ] Pengguna tidak perlu lagi mencari atau mengetik Group JID secara manual.
- [ ] Tersedia tombol "Segarkan Grup & Kontak" untuk sinkronisasi ulang saat ada grup baru.
- [ ] Pesan pengingat berhasil dikirim ke grup WhatsApp terpilih tepat waktu.`,
		},
		{
			Title: "[Feature Request] Built-in Zero-Config Public Tunneling (Global Remote Access)",
			Labels: []string{"enhancement"},
			Body: `## 💡 Latar Belakang & Masalah
Saat ini SiPenDosa berjalan secara lokal (` + "`http://localhost:8080`" + ` atau IP lokal LAN ` + "`192.168.x.x`" + `). Jika pengguna menjalankan SiPenDosa di laptop/PC rumah atau server mini dan ingin mengakses dasbor atau menerima webhook dari jaringan luar (internet publik global) tanpa repot mengatur *port forwarding* router atau membeli IP publik statis.

## 🎯 Solusi yang Diusulkan
Integrasi **Built-in Automatic Tunneling Daemon** langsung di dalam binary engine SiPenDosa:

1. **Pilihan Provider Tunneling**:
   - **Cloudflare Quick Tunnel (` + "`trycloudflare.com`" + `)**: Menggunakan ` + "`cloudflared`" + ` bawaan/terintegrasi tanpa perlu akun untuk mendapatkan URL HTTPS publik instan (contoh: ` + "`https://sipendosa-abc123.trycloudflare.com`" + `).
   - Opsi alternatif: **Bore / Inlets / Localtunnel / Ngrok**.

2. **Dashboard & Mobile Integration**:
   - Menu toggle di Pengaturan: *"Aktifkan Akses Global (Tunneling)"*.
   - Begitu diaktifkan, sistem otomatis membuat secure tunnel dan menampilkan:
     - URL Domain HTTPS publik yang aktif.
     - QR Code yang dapat langsung dipindai dari smartphone untuk membuka SiPenDosa di mana saja.
     - Status koneksi real-time (Latency, Status: Connected).

3. **Keamanan (Security)**:
   - Memastikan proteksi autentikasi (Session Cookie / PIN) aktif ketika dashboard diakses melalui URL tunnel publik.
   - Proteksi CSRF & Host header verification disesuaikan dengan domain tunnel.

## 🚀 Kriteria Selesai (Acceptance Criteria)
- [ ] 1-klik toggle di antarmuka SiPenDosa untuk membuat URL domain publik otomatis.
- [ ] Aplikasi dapat diakses dari jaringan seluler (4G/5G) atau Wi-Fi luar tanpa konfigurasi router.
- [ ] QR Code akses instan untuk smartphone.`,
		},
		{
			Title: "[Feature Request] Public Academic Schedule Board / Portal (/jadwal/public)",
			Labels: []string{"enhancement"},
			Body: `## 💡 Latar Belakang & Masalah
Mahasiswa sekelas atau dosen seringkali membutuhkan informasi jadwal kuliah yang jelas: mata kuliah apa yang sedang/akan berlangsung, lokasi gedung/ruangan di mana, siapa dosen pengampu, dan kode kelasnya. Saat ini dasbor SiPenDosa dilindungi oleh autentikasi login/admin.

## 🎯 Solusi yang Diusulkan
Menambahkan halaman **Portal Papan Jadwal Publik** (Public Schedule Board) yang bisa dibagikan ke teman-teman sekelas:

1. **Halaman Publik ` + "`/jadwal` atau `/public/schedule`" + `**:
   - Dapat diakses siapa saja secara publik tanpa perlu login atau akun admin.
   - Tampilan antarmuka modern, elegan, dan *mobile-first* (Glassmorphism / Obsidian theme).

2. **Informasi Lengkap Jadwal**:
   - 📚 **Mata Kuliah & Kode Kelas** (contoh: *IF3102 - Rekayasa Perangkat Lunak*).
   - 🏛️ **Lokasi Perkuliahan** (contoh: *Gedung Fasilkom Ruang Lab 3 / R.204*).
   - 👨‍🏫 **Dosen Pengampu** (contoh: *Prof. Dr. Ir. Budi Santoso, M.Kom*).
   - ⏰ **Hari, Jam Mulai - Selesai, & SKS**.
   - ⏳ **Widget Countdown Real-Time**: "Kuliah berikutnya dalam 45 menit".

3. **Fitur Interaktif Mahasiswa**:
   - Filter jadwal berdasarkan Hari (*Senin, Selasa, dst.*) atau pencarian mata kuliah/dosen.
   - Tombol "Tautkan Pengingat Kalender" (Download file ` + "`.ics`" + ` / Sync ke Google Calendar).
   - Opsi privasi di Pengaturan Admin: Admin dapat memilih jadwal mana yang ingin di-publish ke publik (*Toggle: Tampilkan di Portal Publik*).

## 🚀 Kriteria Selesai (Acceptance Criteria)
- [ ] URL publik yang dapat dibagikan via grup WhatsApp kelas.
- [ ] Mahasiswa sekelas dapat memantau jadwal, lokasi ruangan, dan dosen secara real-time.
- [ ] Admin memiliki kontrol penuh atas jadwal mana yang ditampilkan ke publik.`,
		},
	}

	for i, issue := range issues {
		payloadBytes, _ := json.Marshal(issue)
		req, err := http.NewRequest("POST", "https://api.github.com/repos/fk0u/SiPenDosa/issues", bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Printf("Error creating issue request %d: %v\n", i+1, err)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error sending issue %d: %v\n", i+1, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			fmt.Printf("Failed to create issue %d (HTTP %d): %s\n", i+1, resp.StatusCode, string(body))
			continue
		}

		var created IssueResponse
		_ = json.Unmarshal(body, &created)
		fmt.Printf("✓ Created Issue #%d: %s\n  URL: %s\n", created.Number, created.Title, created.HTMLURL)
	}
}
