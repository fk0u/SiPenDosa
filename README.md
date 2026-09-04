# SiPen — Advanced WhatsApp Assistant Bot (OverPower Edition)

**SiPen** adalah sistem otomasi dan asisten WhatsApp cerdas tingkat *production-grade* yang dibangun khusus menggunakan bahasa **Golang**. Sistem ini dirancang untuk beroperasi secara mandiri 24/7 di VPS sebagai daemon tangguh dengan Web Dashboard modern (DaisyUI Dark Mode + Tailwind CSS + HTMX).

---

## 🌟 Fitur Utama (OverPower)

1. **Unofficial WhatsApp Engine Terpercaya (`go.mau.fi/whatsmeow`)**:
   - Pemindaian QR Code di Terminal maupun langsung melalui Web Dashboard via WebSocket secara *real-time*.
   - Penyimpanan sesi terisolasi di folder database SQLite terpisah (`session/whatsapp.db`).
   - Deteksi pemutusan koneksi otomatis dengan *exponential backoff auto-reconnect*.
   - **Anti-Ban Architecture**:
     - Normalisasi format nomor otomatis (`08...` &rarr; `628...@s.whatsapp.net` & ID grup `@g.us`).
     - Simulasi kehadiran manusia (*human presence simulation*): Menandai status *Online*, simulasi *Composing / Sedang mengetik...* dengan jeda acak (*jitter*) sebelum pengiriman pesan fisik.

2. **Smart Scheduler (H-1 & Timezone Aware)**:
   - Pengingat perkuliahan mode **H-1** (satu hari sebelumnya) atau **H-0** (hari H).
   - Zona waktu default **Asia/Makassar (WITA)** (dapat disesuaikan ke WIB atau WIT).
   - Pengecekan daftar **Hari Libur**: Jika jadwal jatuh pada tanggal libur yang terdaftar, pesan pengingat dilewati secara otomatis.
   - Pengecekan rentang jam kirim operasional (default `08:00` s.d. `16:00`).
   - **Hitung Mundur (*Countdown*) Real-time** menuju pengiriman berikutnya di dashboard.
   - Fitur **Manual Trigger**: Eksekusi instan jadwal sewaktu-waktu.
   - **Dry-Run Mode**: Mode simulasi global maupun per-jadwal (pesan dicatat tanpa pengiriman ke WhatsApp).

3. **Template Engine Dinamis**:
   - Go Template Engine lengkap dengan fungsi bantuan string dan tanggal.
   - Variabel bawaan:
     - `{{.NamaDosen}}`, `{{.NamaMahasiswa}}`, `{{.NIM}}`
     - `{{.Matkul}}`, `{{.Hari}}`, `{{.Tanggal}}`, `{{.JamMulai}}`, `{{.JamSelesai}}`
     - `{{.Lokasi}}`, `{{.LinkGroup}}`, `{{.WaktuSekarang}}`, `{{.HariDalamBahasa}}`
   - Editor dengan tombol sisip variabel cepat dan **Live Preview** yang langsung ter-render saat mengetik.
   - **Audit Trail & Versioning**: Setiap modifikasi template tersimpan riwayat versinya.

4. **Message Queue & Auto-Retry**:
   - Antrian pesan terjadwal dengan pemrosesan mandiri di latar belakang.
   - Mekanisme *retry* otomatis hingga batas percobaan (*default* 3x) jika terjadi gangguan jaringan.
   - Kemampuan pembatalan (*cancel*) atau paksa kirim segera (*send now*).

5. **Web Dashboard Modern**:
   - DaisyUI Theme `night` (Dark Mode elegan) + Tailwind CSS + HTMX (responsif tanpa beban berat JS).
   - WebSocket real-time untuk status koneksi WhatsApp, QR Code, dan notifikasi Toast pop-up.
   - Manajemen Dosen, Kontak, Jadwal Kuliah, Template, Antrian, dan Pengaturan.
   - Pengunduhan cadangan database SQLite (`.db`) langsung dari dashboard.
   - Viewer log aktivitas sistem dan audit trail.

6. **Zero Hardcoded Data & Zero CGO**:
   - Tidak ada data pribadi yang di-hardcode di kode program.
   - Menggunakan driver SQLite pure-Go (`modernc.org/sqlite`), sehingga dapat di-compile statis tanpa membutuhkan GCC (`CGO_ENABLED=0`).

---

## 🛠️ Persyaratan Sistem

- **Sistem Operasi**: Linux (Ubuntu 20.04 / 22.04 / 24.04 LTS direkomendasikan) atau Windows / macOS.
- **Go**: Versi 1.22 atau lebih baru (jika melakukan build dari source).
- **RAM**: Minimal 512 MB (Sangat ringan).
- **Port Default**: `8473`.

---

## 🚀 Panduan Instalasi di Ubuntu VPS

### Langkah 1: Persiapan Server & Clone Repositori

```bash
# Update paket sistem
sudo apt update && sudo apt upgrade -y
sudo apt install -y git make curl

# Buat direktori aplikasi
sudo mkdir -p /opt/sipen
sudo chown -R $USER:$USER /opt/sipen

# Clone repositori
cd /opt/sipen
git clone <URL_REPOSITORI_ANDA> .
```

### Langkah 2: Konfigurasi Environment (`.env`)

Salin file contoh konfigurasi dan sesuaikan nilai rahasia:

```bash
cp .env.example .env
nano .env
```

Isi konfigurasi `.env`:
```env
PORT=8473
HOST=0.0.0.0
APP_ENV=production
SESSION_SECRET=ganti-dengan-string-acak-rahasia-minimal-32-karakter

DB_PATH=data/sipen.db
WA_SESSION_PATH=session/whatsapp.db

DEFAULT_TIMEZONE=Asia/Makassar
SEND_WINDOW_START=08:00
SEND_WINDOW_END=16:00

RATE_LIMIT_MIN_SEC=5
RATE_LIMIT_MAX_SEC=15
MAX_RETRIES=3

GLOBAL_DRY_RUN=false
```

### Langkah 3: Kompilasi Binary Aplikasi

Kompilasi binary statis yang efisien:

```bash
make build
# Binary akan tercipta di ./sipen
```

### Langkah 4: Setup Service Systemd (Daemon 24/7)

Pasang unit service systemd agar aplikasi berjalan otomatis saat VPS menyala dan auto-restart jika terjadi kendala:

```bash
# Buat user sistem jika diperlukan, atau gunakan www-data
sudo chown -R www-data:www-data /opt/sipen

# Pasang service menggunakan Makefile
sudo make install-service

# Periksa status service
sudo systemctl status sipen
```

Untuk melihat log jalannya daemon:
```bash
sudo journalctl -u sipen -f
```

---

## 🌐 Konfigurasi Reverse Proxy & SSL (Domain)

Agar dashboard dapat diakses menggunakan domain aman (HTTPS) dan WebSocket berjalan lancar, Anda dapat menggunakan **Nginx** atau **Caddy**.

### Opsi A: Menggunakan Nginx + Certbot

Buat file konfigurasi `/etc/nginx/sites-available/sipen`:

```nginx
server {
    server_name sipen.domainanda.com;

    location / {
        proxy_pass http://127.0.0.1:8473;
        proxy_http_version 1.1;
        
        # Dukungan WebSocket
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Aktifkan dan pasang SSL gratis via Let's Encrypt:
```bash
sudo ln -s /etc/nginx/sites-available/sipen /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d sipen.domainanda.com
```

### Opsi B: Menggunakan Caddy (Otomatis SSL)

Cukup tambahkan blok berikut ke `/etc/caddy/Caddyfile`:

```caddy
sipen.domainanda.com {
    reverse_proxy localhost:8473
}
```

Lalu reload:
```bash
sudo systemctl reload caddy
```

---

## 📱 Alur Penggunaan Pertama Kali

1. **Buka Dashboard**: Akses `http://IP_VPS:8473` atau `https://sipen.domainanda.com`.
2. **Inisialisasi Akun Pertama (SuperAdmin)**:
   - Sistem secara otomatis mengarahkan Anda ke halaman pendaftaran SuperAdmin.
   - Buat username dan kata sandi Anda.
   - Setelah selesai, pendaftaran akan otomatis dikunci demi keamanan.
3. **Tautkan WhatsApp**:
   - Di terminal (saat pertama dijalankan) atau klik tombol **"Scan QR Code"** di navbar dashboard.
   - Buka WhatsApp di smartphone Anda &gt; **Perangkat Tertaut** &gt; **Tautkan Perangkat** &gt; Scan QR.
   - Status di navbar akan langsung berubah menjadi hijau (*Terhubung*).
4. **Input Data**:
   - Masukkan daftar dosen dan kontak di menu **Dosen & Kontak**.
   - Buat jadwal perkuliahan di menu **Jadwal Kuliah**.
   - Sesuaikan format pesan di menu **Template Pesan**.
5. **Uji Coba**:
   - Gunakan fitur **Picu Jadwal** dengan mencentang *Dry-Run* atau tombol **Uji Kirim Cepat** di Dashboard Overview.

---

## 🔒 Pemeliharaan & Cadangan Data

- **Backup Database**: Unduh database langsung dari menu **Pengaturan &gt; Unduh Cadangan Database (.db)**, atau salin file `/opt/sipen/data/sipen.db`.
- **Sesi WhatsApp**: Terletak di `/opt/sipen/session/whatsapp.db`. Selama file ini ada, koneksi WhatsApp Anda akan tetap aktif meskipun server di-restart.
