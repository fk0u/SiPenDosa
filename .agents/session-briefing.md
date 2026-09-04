# Session Briefing - SiPenDosa

## Status Snapshot
- **Current Phase:** Full UI Redesign (Agency Grade Double-Bezel) & Comprehensive Error/Legal Page System Complete
- **Status:** Seluruh halaman web **SiPenDosa** telah dirombak secara visual mengikuti standar estetika tinggi (*Awwwards / Agency-grade*), bug template `can't evaluate field NamaDosen in type web.PageData` telah diperbaiki secara tuntas, seluruh badge/tulisan "OverPower" telah dihapus, dan sistem halaman kustom 404, 500, Ketentuan Layanan (ToS), Kebijakan Privasi, serta Filosofi SiPenDosa telah diintegrasikan.

## Makna & Filosofi Nama
1. **Kepanjangan Resmi:**
   - **SiPenDosa** = **Sistem Pengingat Dosen Saatnya**
   - Versi bernyawa: **Si Pen Dosa** — Asisten yang rela “berdosa” demi mengingatkan dosen.
2. **Penjelasan Makna:**
   - **Versi Formal (SRS / Dokumentasi):** Sistem pengingat dosen yang bertugas mengirimkan pesan pengingat jadwal kuliah secara otomatis kepada dosen pengampu mata kuliah.
   - **Versi Asli / Karakter:** Lahir dari kegelisahan mahasiswa yang merasa "berdosa" (sungkan/takut dinilai tidak sopan) jika berkali-kali mengingatkan dosen. SiPenDosa menanggung beban "dosa" tersebut secara otomatis dan sopan.

## Hasil Perbaikan & Fitur Terkini
1. **Perbaikan Template Engine Bug:**
   - Masalah: Tag Go template `{{.NamaDosen}}` di dalam atribut HTML atau `<script>` dibaca sebagai evaluasi field pada struct `web.PageData`.
   - Solusi: Konstruksi tag dipindahkan ke JavaScript dengan pengkodean karakter hex aman (`\x7B\x7B` & `\x7D\x7D`) sehingga parser Go tidak lagi mengevaluasi token dummy.
   - Unit test terverifikasi: `internal/web/renderer_test.go` memvalidasi parsing seluruh halaman template dengan hasil 100% PASS.
2. **Penghapusan Tulisan "OverPower":**
   - Dihapus dari Header Utama (`base.html`), Halaman Login (`login.html`), Halaman Filosofi (`about.html`), dan Telemetri Banner (`banner.go`).
3. **Pembaruan Visual & UI Redesign (Semua Halaman):**
   - **Palet Warna:** Deep OLED space `#07090e`, glassmorphism dengan *hardware double-bezel* (`.bezel-outer` + `.bezel-card`).
   - **Tipografi:** Menggunakan `Plus Jakarta Sans` (editorial headings) dan `JetBrains Mono` (telemetri/JID/jam/kode).
   - **Halaman yang dirombak:** Dashboard Overview, Kontak & Dosen, Jadwal Kuliah, Template Editor & Live Sandbox Preview, Riwayat Audit Pesan, Antrian Pesan (Live Queue), Pengaturan Sistem & Backup Database, Log Telemetri, Halaman Login, dan Halaman Registrasi SuperAdmin.
4. **Sistem Halaman Error & Legalitas:**
   - **404 Not Found (`/404`):** Desain visual kartu kaca dengan teks karakter SiPenDosa yang edukatif dan tombol navigasi kembali.
   - **500 Internal Server Error (`/500`):** Desain terminal diagnosis teknis dengan opsi muat ulang.
   - **Ketentuan Layanan (`/terms`):** 4 pasal ketentuan akademis, kepatuhan anti-spam WhatsApp, dan batasan tanggung jawab.
   - **Kebijakan Privasi (`/privacy`):** Prinsip kedaulatan data lokal (*local-first zero-telemetry*), enkripsi SQLite, dan proteksi sesi bcrypt.
   - **Tentang & Filosofi (`/about`):** Penjelasan dualitas makna (SRS Formal vs Realita Mahasiswa Sungkan) dan spesifikasi arsitektur sistem.
5. **Autentikasi & Verifikasi Browser:**
   - Akun SuperAdmin `kou_admin` terverifikasi dengan password `#Admin1234`.
   - Seluruh rute telah diaudit dan diverifikasi di browser subagent dengan tangkapan layar lengkap.
