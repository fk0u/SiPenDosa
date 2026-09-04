# Keputusan Arsitektur: SiPen (OverPower Edition)

**Tanggal:** 4 September 2026  
**Status:** Diterima & Diimplementasikan  

## Konteks
Aplikasi asisten WhatsApp SiPen membutuhkan keandalan tingkat tinggi untuk beroperasi 24/7 di VPS Linux serta kemudahan kompilasi di lingkungan pengembangan Windows lokal yang tidak memiliki toolchain GCC/CGO bawaan.

## Keputusan Teknis
1. **Pilihan Driver SQLite (Pure-Go / CGO-Free):**
   - Menggunakan `modernc.org/sqlite` untuk basis data relasional aplikasi (`data/sipen.db`) maupun sqlstore whatsmeow (`session/whatsapp.db`).
   - Hasil: Kompilasi sepenuhnya statis dengan `CGO_ENABLED=0`, menghilangkan dependensi terhadap libc/gcc eksternal.
2. **Arsitektur Pengiriman Pesan (Queue & Anti-Ban):**
   - Menggantikan pengiriman langsung dengan pola antrian persisten (`queue_messages`).
   - Setiap pengiriman pesan mensimulasikan kehadiran manusia (*Presence Available* & *Chat Presence Composing*) dengan jeda waktu acak (*jitter*) 1.5 - 2.5 detik sebelum pengiriman pesan teks.
   - Jeda antar pesan dalam antrian dikonfigurasi dinamis (5 - 15 detik) untuk mencegah flag deteksi bot WhatsApp.
3. **Frontend Interaktif Tanpa SPA Framework:**
   - Memadukan DaisyUI Dark Mode (`night`), Tailwind CSS CDN, dan HTMX.
   - Menggunakan WebSocket untuk event server-to-client (status WA, QR stream, countdown) sehingga tidak memerlukan SPA React/Vue yang kompleks dan berat.
4. **Smart Scheduler H-1 & Timezone:**
   - Scheduler mengevaluasi hari kuliah dan mengirimkan pesan tepat 1 hari sebelumnya (H-1) pada jam yang ditentukan, dengan memperhatikan kalender libur nasional/kampus serta rentang jam operasional (08:00 - 16:00 WITA).
