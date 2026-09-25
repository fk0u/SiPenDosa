# 📜 Catatan Perubahan (CHANGELOG)

Semua perubahan penting pada proyek **SiPenDosa** didokumentasikan di sini dengan format standar [Keep a Changelog](https://keepachangelog.com/id/1.0.0/) dan mengikuti [Semantic Versioning](https://semver.org/lang/id/).

---

## [1.2.1] — 2026-09-25

### 🚀 Peningkatan Utama & Fitur Baru
* **macOS 12+ Universal 2 Distribution:**
  * Dukungan native penuh untuk macOS 12 Monterey, 13 Ventura, 14 Sonoma, 15 Sequoia (Apple Silicon M1-M4 & Intel x64).
  * Kompilasi dual-architecture `lipo` untuk Go Core Server, C++17 Socket Bridge, dan Swift Companion AppKit.
  * Opsi rilis mandiri: Apple Disk Image (`SiPenDosa-1.2.0.dmg`), Apple Installer Package (`SiPenDosa-1.2.0-Installer.pkg`), dan Tarball Universal.
* **Mobile 24/7 Background Execution & Real-Time Engine:**
  * **Android:** Foreground Service `dataSync`, CPU WakeLock + High-Performance WifiLock, perlindungan Doze Mode, Boot Completed persistence (`BootCompletedReceiver`), dan Watchdog Auto-Respawn.
  * **iOS:** Integrasi `BGTaskScheduler` (`BGAppRefreshTask` & `BGProcessingTask`), extended background task execution saat diminimalkan, live WebSocket telemetry hub, dan notifikasi lokal realtime (`UNUserNotificationCenter`).
* **REST JSON API Endpoints:**
  * Penambahan rute API terstruktur: `/api/stats`, `/api/whatsapp/status`, `/api/contacts`, `/api/schedules`, `/api/queue`, dan `/api/ws` untuk konsumsi klien mobile dan IoT.

### 🐛 Perbaikan Bug & Keandalan (Bug Fixes)
* **Atomic Queue Claiming (BUG-01):** Mencegah duplikasi pengiriman pesan WhatsApp ke dosen saat pengiriman terjadwal dan manual *Kirim Sekarang* berjalan bersamaan.
* **Non-Blocking Jitter Delay (BUG-02):** Mengganti `time.Sleep` pada rate limiter worker dengan `select` berbasis channel `stopChan` agar shutdown berlangsung seketika.
* **Goroutine Leak Auto-Reconnect (BUG-03):** Membatasi inisialisasi `keepAliveLoop` WhatsApp client agar hanya berjalan satu instance per sesi.
* **Safe Channel Close (BUG-04):** Mencegah panic runtime saat `client.Close()` dipanggil berulang kali dengan proteksi `sync.Once`.
* **Reverse Proxy & Tunneling CSRF Support (BUG-06):** Mendukung header `X-Forwarded-Host` dan perbandingan base hostname agar form submission via Nginx/Cloudflare Tunnel tidak tertolak dengan 403 Forbidden.
* **API Authentication Handling (BUG-07):** Memberikan respon HTTP 401 Unauthorized dalam format JSON untuk permintaan API `/api/...` dan mendukung header `Authorization: Bearer <token>`.
* **Dynamic Port Parsing macOS (BUG-08):** Menu bar controller macOS membaca variabel `PORT` langsung dari file `.env` lokal.
* **Android IP Discovery (BUG-09):** Menambahkan deteksi IP antarmuka jaringan `NetworkInterface` aktif sebagai fallback jika `WifiManager` mengembalikan `0.0.0.0`.
* **Holiday-Aware Countdown (BUG-10):** Mengintegrasikan pengecekan kalender libur nasional/kampus pada widget hitung mundur jadwal kuliah.
* **Auto-Reconnect Graceful Shutdown (BUG-11):** Mengganti `time.Sleep` pada backoff WhatsApp auto-reconnector dengan channel interrupt.
* **Queue Process Status Validation (BUG-12):** Memvalidasi status pesan antrian pada `ProcessImmediate` agar hanya pesan berstatus `pending` yang diproses.
* **Manual Test Send Safety:** Pesan simulasi manual ditandai dengan `[MANUAL_DRY_RUN]` sehingga 100% aman tanpa pengiriman fisik ke WhatsApp dosen.
* **Indonesian Phone Normalization:** Menambahkan normalisasi otomatis untuk nomor berawalan `8...` menjadi format internasional `628...`.

### 📚 Dokumentasi
* Menambahkan [Panduan Paket macOS (MACOS_PACKAGE_GUIDE.md)](docs/MACOS_PACKAGE_GUIDE.md).
* Menambahkan [Arsitektur Latar Belakang Mobile (MOBILE_BACKGROUND_ARCHITECTURE.md)](docs/MOBILE_BACKGROUND_ARCHITECTURE.md).
* Menambahkan [Laporan Audit & Analisis Bug (AUDIT_AND_BUGS_ANALYSIS.md)](docs/AUDIT_AND_BUGS_ANALYSIS.md).
* Menambahkan [Panduan Kontribusi Open Source (CONTRIBUTING.md)](CONTRIBUTING.md).

---

## [1.1.2] — 2026-09-20
* Peningkatan performa database SQLite WAL dan isolasi sesi multi-user.
* Pembaruan aset visual banner dashboard bertema Crimson Scarlet & Burnished Gold.

## [1.1.1] — 2026-09-15
* Penambahan template audit trail dan history pesan WhatsApp.
* Integrasi awal menu bar macOS companion controller.
