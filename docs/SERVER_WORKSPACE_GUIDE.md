# Panduan SiPenDosa Server Workspace Edition

> **SiPenDosa Server Edition** adalah varian arsitektur server kolaboratif multi-tenant yang memungkinkan banyak mahasiswa/pengurus kelas (Sipen) menggunakan satu instalasi server secara bersamaan, dengan efisiensi penggunaan sumber daya (RAM & CPU) yang sangat hemat serta format pesan WhatsApp modern interaktif layaknya WhatsApp Official API Bot.

---

## 1. Keunggulan & Arsitektur Utama

### A. Pendaftaran Publik Terbuka & Smart Rate Limiting (Non-IP Based)
Di lingkungan kampus, asrama, atau kantor, ratusan pengguna terhubung ke satu jaringan WiFi yang sama dengan satu alamat IP publik keluar (**NAT / CGNAT**).
* **Masalah pada Rate Limiting IP Tradisional:** Ketika 3–5 mahasiswa mendaftar di jam yang sama dari WiFi kampus, mahasiswa berikutnya langsung terkena blokir `HTTP 429 Too Many Requests`.
* **Solusi SmartLimiter SiPenDosa:** Sistem menggunakan **Device Fingerprint** berbasis `SHA-256(User-Agent + Accept-Language + Username)` dikombinasikan dengan global token sliding window dan entropy timing.
* **Hasil:** Mahasiswa dalam satu kampus dapat mendaftar bebas tanpa saling memblokir satu sama lain, sementara server tetap kebal dari serangan bot spam massal.

---

### B. Fleksibilitas Nomor WhatsApp: Pribadi vs. Bot Bersama
Setiap Ruang Kerja (Workspace) dapat dikonfigurasi secara mandiri sesuai preferensi organisasi:
1. **Mode Nomor Pribadi (Personal Gateway):**
   - Setiap sipen menautkan nomor WhatsApp pribadinya masing-masing via scan QR atau nomor HP.
   - Pesan pengingat dosen dikirim langsung atas nama nomor pribadi sipen tersebut.
   - Dosen mengenal nomor sipen secara langsung.
2. **Mode Bot Bersama (Shared Gateway):**
   - Workspace dikonfigurasi dengan satu nomor WhatsApp bot resmi bersama (misalnya nomor HP admin angkatan atau bot resmi himpunan).
   - Seluruh anggota kelas tidak perlu melakukan scan QR satu per satu.
   - Ketika jadwal kuliah tiba, pesan terkirim otomatis melalui bot resmi workspace.

---

### C. Pesan WhatsApp Interaktif (NativeFlow Buttons & Smart Bot)
SiPenDosa mengintegrasikan protokol mutakhir WhatsApp `waE2E.InteractiveMessage` dengan format tombol interaktif:

```
┌─────────────────────────────────────────────────────────────┐
│ 📚 PENGINGAT JADWAL KULIAH                                   │
│ Yth. Bpk/Ibu Dosen Dr. Ir. Fulan, M.T.                      │
│ Jadwal Pemrograman Web akan dimulai 30 menit lagi (08:00).  │
│                                                             │
│ Mohon konfirmasi kesediaan kehadiran Bapak/Ibu:             │
├─────────────────────────────────────────────────────────────┤
│ [📍 Konfirmasi Hadir Offline]                               │
│ [💻 Konfirmasi Hadir Online]                                │
│ [🔄 Mohon Jadwal Ulang]                                     │
└─────────────────────────────────────────────────────────────┘
```

#### Cara Kerja Tombol & Chatbot Konfirmasi Otomatis:
1. **Dosen Menekan Tombol:** Klien WhatsApp dosen mengirimkan event interaktif ke nomor pengirim.
2. **Chatbot Mendeteksi Jawaban:**
   - Menekan `[Hadir Offline]` atau membalas `1`: Bot otomatis membalas santun terima kasih dan mencatat konfirmasi offline.
   - Menekan `[Hadir Online]` atau membalas `2`: Bot otomatis membalas dan menyematkan catatan bahwa dosen bersedia hadir daring.
   - Menekan `[Mohon Jadwal Ulang]` atau membalas `3`: Bot otomatis meminta maaf dan meneruskan status reschedule ke sistem.
3. **Pembaruan Live Dashboard:** WebSocket server langsung memicu toast notifikasi di browser sipen dan memperbarui catatan antrian pesan secara realtime.
4. **Fallback Teks Terstruktur:** Jika dosen menggunakan WhatsApp versi sangat lama yang belum mendukung rendering tombol dinamis, bot otomatis menampilkan opsi bernomor (`Balas 1 untuk Offline, 2 untuk Online, 3 untuk Reschedule`).

---

### D. Efisiensi Sumber Daya Ekstrem (VPS Hemat Friendly: RAM < 35MB)
Agar biaya operasional server dapat ditekan hingga ke tier VPS termurah (512MB RAM seharga $2–$3/bulan):
1. **WhatsApp Socket Hibernation (Tidur Otomatis):**
   - Setiap koneksi WebSocket WhatsApp yang tidak memiliki aktivitas pengiriman selama 15 menit otomatis dilepas (hibernasi).
   - Ketika scheduler menemukan antrian pesan yang harus dikirim, socket dibangunkan seketika dalam < 1 detik.
   - Menghemat ratusan megabyte RAM dan mengeliminasi overhead TCP keep-alive yang tidak perlu.
2. **SQLite Write-Ahead Logging (WAL) & In-Memory Pragma:**
   - `PRAGMA journal_mode = WAL;`
   - `PRAGMA synchronous = NORMAL;`
   - `PRAGMA temp_store = MEMORY;`
   - `PRAGMA cache_size = -4000;` (4MB memory cache)
   - Konkurensi tinggi tanpa risiko database locked.
3. **Auto-Pruner Sisa Log:**
   - Log audit dan antrian pesan yang telah selesai dibersihkan otomatis sesuai konfigurasi `PRUNE_RETENTION_DAYS` (default: 30 hari).

---

## 2. Cara Deploy Cepat Menggunakan Docker

### Prasyarat
- Docker Engine 20.10+ & Docker Compose v2+
- Port `8473` terbuka di firewall server Anda.

### Langkah-Langkah:
1. **Clone Repositori:**
   ```bash
   git clone https://github.com/fk0u/SiPenDosa.git
   cd SiPenDosa
   ```

2. **Jalankan dengan Docker Compose:**
   ```bash
   docker compose -f docker-compose.server.yml up -d --build
   ```

3. **Periksa Status Layanan:**
   ```bash
   docker compose -f docker-compose.server.yml ps
   docker compose -f docker-compose.server.yml logs -f
   ```

4. **Buka Web Dashboard:**
   Buka browser di: `http://IP_SERVER_ANDA:8473`
   - Akun pertama yang Anda daftarkan otomatis menjadi **SuperAdmin**.
   - Pengguna berikutnya yang mendaftar langsung dialokasikan ke workspace awal ("Workspace Utama").

---

## 3. Cara Deploy Mandiri di Linux (Systemd Service)

Jika ingin menjalankan tanpa Docker di Linux VPS (Ubuntu/Debian/Arch/Alpine):

1. **Unduh Binary SiPenDosa:**
   ```bash
   curl -sL https://raw.githubusercontent.com/fk0u/SiPenDosa/master/install-linux.sh | bash
   ```

2. **Aktifkan Server Mode di `.env`:**
   ```bash
   echo "SIPEN_SERVER_MODE=true" >> /etc/sipendosa/.env
   echo "SIPEN_UI=web" >> /etc/sipendosa/.env
   echo "PORT=8473" >> /etc/sipendosa/.env
   ```

3. **Mulai Layanan Systemd:**
   ```bash
   sudo systemctl enable --now sipen
   sudo systemctl status sipen
   ```

---

## 4. Panduan Penggunaan Ruang Kerja (Workspaces)

### A. Membuat Ruang Kerja Baru
1. Masuk ke menu **Ruang Kerja** di sidebar navigasi kiri (`/workspaces`).
2. Klik tombol **Buat Ruang Kerja Baru**.
3. Masukkan nama ruang kerja (misalnya: `Sistem Informasi 2024 Kelas A`).
4. Pilih Mode Gateway:
   - **Nomor Pribadi:** Jika masing-masing sipen ingin memakai nomor WhatsApp-nya sendiri.
   - **Bot Bersama:** Jika ingin memakai 1 nomor WhatsApp resmi milik admin kelas.
5. Klik **Simpan**.

### B. Mengundang Mahasiswa/Sipen Lain
1. Di halaman detail Ruang Kerja, klik **Tambah Anggota ke Workspace**.
2. Masukkan `username` mahasiswa yang sudah membuat akun.
3. Tentukan peran (`member` atau `admin`).
4. Klik **Tambahkan Anggota**.

### C. Beralih Antar Ruang Kerja
- Klik **Pill Ruang Kerja** di pojok kanan atas bilah navigasi (Topbar).
- Pilih ruang kerja yang ingin Anda kelola.
- Seluruh jadwal, kontak dosen, dan antrian pesan akan otomatis tersaring sesuai ruang kerja aktif tersebut.

---

## 5. Konfigurasi Lingkungan (`.env`)

| Variabel | Bawaan | Deskripsi |
|---|---|---|
| `SIPEN_SERVER_MODE` | `true` | Mengaktifkan arsitektur multi-tenant dan public register |
| `PORT` | `8473` | Port HTTP dashboard web |
| `DEFAULT_TIMEZONE` | `Asia/Makassar` | Zona waktu scheduler (mis. `Asia/Jakarta`, `Asia/Jayapura`) |
| `COOKIE_SECURE` | `false` | Set `true` jika menggunakan domain HTTPS atau Cloudflare Tunnel |
| `PRUNE_RETENTION_DAYS` | `30` | Jumlah hari retensi log sebelum dibersihkan otomatis |
| `RATE_LIMIT_MIN_SEC` | `5` | Jeda minimum acak antar pesan (Anti-Ban WhatsApp) |
| `RATE_LIMIT_MAX_SEC` | `15` | Jeda maksimum acak antar pesan (Anti-Ban WhatsApp) |

---

## 6. Integrasi Domain & HTTPS (Cloudflare Tunnel)

SiPenDosa telah dilengkapi dengan Cloudflare Tunnel terintegrasi:
1. Buka menu **Pengaturan** > **Public Tunnel**.
2. Masukkan **Cloudflare Tunnel Token** Anda.
3. Klik **Aktifkan Tunnel**.
4. Dashboard dan Portal Jadwal Publik dapat diakses langsung dari domain HTTPS publik Anda tanpa perlu membuka port NAT router secara manual.
