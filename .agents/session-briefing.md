# Session Briefing - SiPen

## Status Snapshot
- **Current Phase:** Production-Ready & Cyber Aesthetic Terminal Interface Complete
- **Goal:** Aplikasi **SiPen** (Advanced WhatsApp Assistant Bot - OverPower Edition) telah selesai dibangun secara penuh, dilengkapi dengan antarmuka terminal/daemon bergaya Cyberpunk/OverPower (ASCII art banner bergradasi, tabel telemetri sistem, kotak panduan QR code berbingkai, dan log alur daemon berwarna), lulus uji kompilasi statis CGO-free, dan lulus seluruh unit tests (100% PASS).

## Komponen Utama
1. `internal/banner`: Modul banner ASCII art bergradasi, kotak telemetri sistem, frame panduan scan QR code, dan step loggers bergaya terminal futuristik.
2. `internal/config`: Konfigurasi `.env` dengan defaults port 8473, timezone Asia/Makassar (WITA), dan anti-ban delay.
3. `internal/store`: SQLite pure-Go (`modernc.org/sqlite`) untuk users, sessions, settings, contacts, schedules, templates, template_versions, holidays, queue_messages, dan activity_logs.
4. `internal/auth`: Sistem login, registrasi SuperAdmin bootstrap, password hashing (bcrypt), secure cookie sessions, dan role-based middleware.
5. `internal/realtime`: WebSocket Hub (`/ws`) untuk live WhatsApp status, QR code stream, countdown timer, dan toast alerts.
6. `internal/whatsapp`: Integrasi `go.mau.fi/whatsmeow` dengan `sqlstore`, generator QR terminal berbingkai + PNG web, auto-reconnect, dan simulasi kehadiran manusia (anti-ban typing presence).
7. `internal/queue`: Message queue worker dengan anti-ban randomized delay, auto-retry (max 3x) dengan exponential backoff, dan mode dry-run.
8. `internal/scheduler`: Smart Scheduler H-1 dengan pengecekan kalender hari libur, rentang jam kirim operasional (08:00 - 16:00 WITA), kalkulasi countdown live, dan pemicu manual.
9. `internal/template`: Go template engine dengan variabel lengkap, template formal bahasa Indonesia default, dan live preview via HTMX.
10. `internal/web`: Chi router dan controller untuk Overview, Contacts, Schedules, Templates, History, Queue, Settings, dan Logs.
11. `web/templates`: Seluruh template HTML DaisyUI theme `night` + Tailwind CSS + HTMX responsif dan modern.
12. `Deployment`: `Makefile`, `sipen.service` (systemd unit untuk VPS Ubuntu), `.env.example`, dan `README.md` panduan deployment komprehensif.
