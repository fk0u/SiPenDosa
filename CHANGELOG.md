# 📜 Catatan Perubahan (CHANGELOG)

Semua perubahan penting pada proyek **SiPenDosa** didokumentasikan di sini dengan format standar [Keep a Changelog](https://keepachangelog.com/id/1.0.0/) dan mengikuti [Semantic Versioning](https://semver.org/lang/id/).

---

## [1.5.0] — 2026-10-26

### 🏢 Server Workspace Edition & Multi-Tenant Support
* **Multi-Tenant Collaboration (`/workspaces`):** Dukungan ruang kerja (workspace) kolaboratif untuk kelas, program studi, atau organisasi mahasiswa. Setiap anggota dapat berpindah antar-workspace melalui Workspace Switcher di navigasi utama.
* **Fleksibilitas Gateway WhatsApp:** Ruang kerja dapat dikonfigurasi menggunakan **Nomor Pribadi (Personal Gateway)** di mana setiap sipen menautkan nomornya masing-masing, atau **Bot Bersama (Shared Gateway)** menggunakan satu nomor bot WhatsApp resmi milik admin workspace.
* **Smart Non-IP Registration Rate Limiting:** Pendaftaran akun publik terbuka kini dilindungi oleh sistem token bucket cerdas berbasis device fingerprint (`SHA-256(User-Agent + Accept-Language + Username)`) dan timing entropy. Seluruh mahasiswa dalam jaringan WiFi/NAT kampus yang sama bebas mendaftar tanpa terkena pemblokiran IP massal (`HTTP 429`).

### 🤖 Pesan WhatsApp Interaktif & Smart Auto-Reply Chatbot
* **WhatsApp NativeFlow Interactive Buttons:** Pesan pengingat ke dosen kini dapat dilengkapi dengan tombol interaktif resmi WhatsApp: `[Konfirmasi Hadir Offline]`, `[Konfirmasi Hadir Online]`, dan `[Mohon Jadwal Ulang]`.
* **Asisten Chatbot Konfirmasi Otomatis:** Sistem mendeteksi ketika dosen menekan tombol atau membalas dengan angka `1`, `2`, atau `3`, secara otomatis membalas secara santun, dan memperbarui status kehadiran dosen secara live di antrian pesan dan WebSocket dashboard.
* **Structured Text Fallback:** Untuk versi aplikasi WhatsApp lama yang belum mendukung rendering tombol dinamis, sistem otomatis mengalirkan menu interaktif berbasis teks berformat rapi.

### ⚡ Efisiensi Sumber Daya Server Ekstrem (Ultra-Low Footprint)
* **WhatsApp Socket Hibernation:** Socket WhatsApp otomatis masuk mode hibernasi setelah 15 menit tanpa aktivitas pengiriman, menjaga pemakaian RAM server tetap di bawah 35MB (sangat aman untuk VPS tier 256MB/512MB RAM). Socket otomatis dibangunkan seketika dalam < 1 detik saat ada antrian pesan.
* **SQLite WAL & Memory Caching Pragma:** Konfigurasi `journal_mode = WAL`, `synchronous = NORMAL`, `temp_store = MEMORY`, dan `cache_size = -4000` untuk performa konkurensi multi-user tanpa database locked.
* **Docker Server Deployment:** Menyediakan `Dockerfile.server` (multi-stage Alpine < 30MB) dan `docker-compose.server.yml` dengan konfigurasi pembatas resource (`mem_limit: 256M`), serta panduan operasional lengkap di `docs/SERVER_WORKSPACE_GUIDE.md`.

---

## [1.4.2] — 2026-10-26

### 📱 Android & Embedded Engine Improvements
* **Otomasi Mode Web pada Android:** Binary engine mendeteksi lingkungan Android (`netutil.IsAndroid()` dan `runtime.GOOS == "android"` / variabel lingkungan Android) dan secara otomatis berjalan dalam mode web (`SIPEN_UI=web`) tanpa meminta input TTY atau wizard.
* **Deteksi Daemon & Pemulihan Service Android:** `SiPenDosaServerService` memverifikasi ketersediaan port 8473 (`/healthz`) sebelum memulai instance baru, mencegah konflik port saat service direstart.
* **Ekstraksi Biner APK Otomatis:** Menambahkan fallback ekstraksi `libsipen.so` langsung dari arsip APK jika sistem Android menolak akses `nativeLibraryDir`.
* **Koneksi WebView Tangguh:** `InternalWebClient` kini otomatis me-reload WebView saat daemon lokal sedang dalam proses booting.

---

## [1.4.1] — 2026-10-25

### 🐛 Perbaikan Bug
* **Dialog update berulang (loop)**: self-update kini **me-restart proses otomatis** setelah binary baru terpasang (re-exec pada Unix, peluncuran proses baru pada Windows) — sebelumnya binary diganti di disk tapi proses lama tetap berjalan sehingga dialog update muncul terus.
* **Tampilan tidak berubah setelah update**: dashboard kini selalu menyajikan **aset embedded** yang tertanam di binary — direktori `web/templates` & `web/static` peninggalan instalasi lama tidak lagi menimpa frontend baru. File disk hanya dipakai di mode pengembangan (`SIPEN_DEV_ASSETS=1`).
* `manifest.json` & `favicon.ico` kini disajikan dari aset embedded (sebelumnya path relatif CWD yang bisa hilang tergantung direktori kerja).
* Toast sukses/gagal update kini muncul melalui WebSocket sebelum proses restart.

### ⚠️ Catatan Migrasi
* Pengguna v1.2.0 atau lebih lama: logika restart baru ada di binary ini. Setelah memasang v1.4.1, **restart SiPenDosa satu kali secara manual** (macOS: menu bar "Mulai Ulang Engine"; lainnya: hentikan dan jalankan ulang `sipen`). Update berikutnya akan me-restart otomatis.

---

## [1.4.0] — 2026-10-25

### 🌟 Fitur Baru
* **Mode Akses First-Class — `SIPEN_UI=terminal|web|desktop`:**
  * **Terminal (default)**: konsol interaktif penuh tanpa server HTTP — status engine live, countdown, daftar jadwal & antrian, trigger pengiriman, pairing WhatsApp, dan daftar akun kelas langsung dari konsol (`internal/termui`).
  * **Web**: daemon + dashboard browser (perilaku lama).
  * **Desktop**: web + jendela aplikasi desktop — di macOS berupa jendela native WKWebView pada companion app; di Windows/Linux jendela app-mode Edge/Chrome.
* **First-Run Wizard**: instalasi baru yang dijalankan interaktif ditanya mode aksesnya (tertulis ke `.env`); subcommand baru `sipen mode [terminal|web|desktop]`, `sipen web`, `sipen terminal`, `sipen desktop`.
* **Installer Bertanya Mode Akses**: `install-macos.sh`, `install-linux.sh`, `install-android-termux.sh`, `install.ps1`, dan Windows exe installer kini menanyakan terminal/web/desktop saat instalasi; PKG macOS menulis `SIPEN_UI=web` (companion app terpasang), instalasi lama non-interaktif dimigrasi otomatis ke mode web.
* **Desktop Companion macOS**: jendela aplikasi native (WKWebView) dengan menu "Buka Aplikasi Desktop" dan auto-terbuka saat engine dimulai oleh aplikasi.

---

## [1.3.0] — 2026-10-25

### 🌟 Fitur Baru
* **Akun Kelas (Multi-Tenant):**
  * Peran baru `kelas` untuk penanggung jawab mata kuliah — kini default akun yang dibuat administrator.
  * Setiap akun kelas memiliki **sesi WhatsApp terpisah** (`session/users/user_<id>.db`): pairing bisa memakai nomor pribadi PJ **atau** nomor WA khusus milik kelas.
  * Data kontak, jadwal, template, dan antrian sepenuhnya terisolasi antar akun.
* **Endpoint Token Login Companion:** `POST /api/auth/login` menukar username/password menjadi bearer token untuk aplikasi mobile.
* **Mobile Companion Authenticating:** Flutter (`ApiService.login` + token persist), iOS (session cookie WebView + WS token), dan Android (cookie sesi untuk pair-phone) mengikuti skema autentikasi baru.

### 🔒 Keamanan (Security Audit Fixes)
* **IDOR Diperbaiki:** seluruh mutasi kontak/jadwal/template/antrian kini di-scope `user_id` pada store layer dan divalidasi kepemilikannya di handler — akun kelas tidak dapat mengubah data kelas lain.
* **WebSocket & Mobile API Wajib Autentikasi:** `/ws`, `/api/ws`, dan seluruh endpoint `/api/*` companion tidak lagi terbuka; QR code dan pairing code tidak lagi broadcast ke klien tanpa sesi. Origin check WebSocket dikunci same-origin.
* **Bypass Pair-Phone Ditambal:** `middleware.RealIP` dihapus (X-Forwarded-For tidak lagi dipercaya dari koneksi langsung), rate limiter hanya menerima XFF dari koneksi loopback (entri terakhir), dan saat tunnel publik aktif endpoint loopback menuntut sesi login.
* **Self-Update Diperkuat:** hanya superadmin, URL unduhan dikunci ke host GitHub resmi (HTTPS), mencegah RCE via `download_url` arbitrer.
* **Info Leak:** `/healthz` tidak lagi membocorkan nomor WhatsApp; `SESSION_SECRET` hardcoded dihapus dari konfigurasi; cookie sesi mendukung flag `Secure` (`COOKIE_SECURE=true`).
* **Regresi Test Keamanan:** test baru untuk auth mobile API, token login, isolasi antar akun, dan kebocoran healthz.

### 🐛 Perbaikan & Keandalan
* Pembersihan otomatis sesi & tantangan 2FA kedaluwarsa setiap 24 jam (sebelumnya menumpuk selamanya).
* Error store pada update/hapus kontak & jadwal tidak lagi disembunyikan sebagai toast "sukses".
* Preview template di-escape HTML (mencegah XSS refleksi).
* Repo hygiene: untrack `.DS_Store`, artefak kompilasi, dan berkas catatan agent yang memuat kredensial.

### ⚠️ Catatan Migrasi
* Aplikasi companion lama yang memanggil `/api/*` tanpa token akan menerima HTTP 401 — perbarui ke versi terbaru dan login melalui `POST /api/auth/login`.
* Set `COOKIE_SECURE=true` pada `.env` bila dashboard disajikan lewat HTTPS/tunnel.

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
