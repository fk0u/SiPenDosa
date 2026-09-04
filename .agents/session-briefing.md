# Session Briefing - SiPenDosa

## Status Snapshot
- **Current Phase:** Production-Ready Rebranded to SiPenDosa with Full Aesthetic Integration
- **Goal:** Aplikasi telah resmi dibranding menjadi **SiPenDosa** (**Sistem Pengingat Dosen Saatnya** / *"Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak perlu merasa sungkan"*). Seluruh antarmuka web, daemon ASCII cyberpunk banner, telemetri sistem, template default, dokumentasi SRS/README, systemd unit, dan handlers telah diperbarui. Lulus uji kompilasi statis CGO-free dan 100% unit tests PASS.

## Makna & Filosofi Nama
1. **Kepanjangan Resmi:**
   - **SiPenDosa** = **Sistem Pengingat Dosen Saatnya**
   - Versi bernyawa: **Si Pen Dosa** — Asisten yang rela “berdosa” demi mengingatkan dosen.
2. **Penjelasan Makna:**
   - **Versi Formal (SRS / Dokumentasi):** Sistem pengingat dosen yang bertugas mengirimkan pesan pengingat jadwal kuliah secara otomatis kepada dosen pengampu mata kuliah.
   - **Versi Asli / Karakter:** Lahir dari kegelisahan mahasiswa yang merasa "berdosa" (sungkan/takut dianggap tidak sopan) jika berkali-kali mengingatkan dosen. SiPenDosa hadir menanggung beban "dosa" tersebut dengan pesan sopan dan terjadwal.

## Komponen Utama
1. `internal/banner`: Modul banner ASCII art bergradasi SIPENDOSA, kotak telemetri sistem, frame panduan scan QR code, dan step loggers bergaya terminal futuristik.
2. `internal/config`: Konfigurasi `.env` dengan defaults port 8473, timezone Asia/Makassar (WITA), dan anti-ban delay.
3. `internal/store`: SQLite pure-Go (`modernc.org/sqlite`) untuk users, sessions, settings, contacts, schedules, templates (default template SiPenDosa seeded), template_versions, holidays, queue_messages, dan activity_logs.
4. `internal/auth`: Sistem login, registrasi SuperAdmin bootstrap, password hashing (bcrypt), secure cookie sessions, dan role-based middleware.
5. `internal/realtime`: WebSocket Hub (`/ws`) untuk live WhatsApp status, QR code stream, countdown timer, dan toast alerts.
6. `internal/whatsapp`: Integrasi `go.mau.fi/whatsmeow` dengan `sqlstore`, generator QR terminal berbingkai + PNG web, auto-reconnect, dan simulasi kehadiran manusia (anti-ban typing presence).
7. `internal/queue`: Message queue worker dengan anti-ban randomized delay, auto-retry (max 3x) dengan exponential backoff, dan mode dry-run.
8. `internal/scheduler`: Smart Scheduler H-1 dengan pengecekan kalender hari libur, rentang jam kirim operasional (08:00 - 16:00 WITA), kalkulasi countdown live, dan pemicu manual.
9. `internal/template`: Go template engine dengan variabel lengkap, template formal bahasa Indonesia default, dan live preview via HTMX.
10. `internal/web`: Chi router dan controller untuk Overview, Contacts, Schedules, Templates, History, Queue, Settings, dan Logs.
11. `web/templates`: Seluruh template HTML DaisyUI theme `night` + Tailwind CSS + HTMX responsif dan modern dengan identitas visual SiPenDosa.
12. `Deployment`: `Makefile`, `sipen.service` (systemd unit untuk VPS Ubuntu), `.env.example`, `.env`, dan `README.md` panduan deployment komprehensif.
