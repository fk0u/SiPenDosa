# Roadmap Proyek: SiPenDosa (Sistem Pengingat Dosen Saatnya)

## Status Saat Ini (v1.0.0-prod)
- [x] Inisialisasi arsitektur bersih Golang (CGO-free dengan `modernc.org/sqlite`).
- [x] Integrasi WhatsApp Web Multi-Device (`go.mau.fi/whatsmeow`) dengan anti-ban typing simulation.
- [x] Web Dashboard modern bertema dark mode (DaisyUI Theme `night` + Tailwind CSS).
- [x] WebSocket realtime untuk status WhatsApp, scan QR code, countdown pengiriman, dan toast alerts.
- [x] Message Queue dengan penanganan auto-retry (exponential backoff) dan jitter rate limiter.
- [x] Smart Scheduler dengan dukungan mode H-1, evaluasi rentang jam operasional (08:00 - 16:00), dan kalender hari libur.
- [x] Template Engine Go Template dengan variabel lengkap dan Live Preview interaktif via HTMX.
- [x] Manajemen autentikasi sesi aman (HttpOnly, SameSite) dan hak akses (SuperAdmin / Admin).
- [x] Deployment assets: `Makefile`, unit systemd `sipen.service`, dan panduan lengkap Ubuntu VPS di `README.md`.

## Rencana Pengembangan Selanjutnya (Future Roadmap)
- [ ] Pengiriman attachment media (PDF materi kuliah, silabus, atau gambar pengumuman).
- [ ] Integrasi webhook penerimaan pesan balasan (misal konfirmasi kehadiran dari dosen / mahasiswa).
- [ ] Backup otomatis database ke cloud storage (Google Drive / S3 / Telegram Bot).
