# SiPenDosa — Advanced WhatsApp Assistant Bot (OverPower Edition)

> ⚡ **"Sistem Pengingat Dosen Saatnya"**  
> *"Asisten yang rela 'berdosa' demi mengingatkan dosen, agar mahasiswa tidak perlu merasa sungkan."*

---

## 📖 Makna & Filosofi Nama: SiPenDosa

### 1. Kepanjangan Resmi
* **SiPenDosa** = **Sistem Pengingat Dosen Saatnya**  
* *Atau versi yang lebih “bernyawa”:*  
  **Si Pen Dosa** — Asisten yang rela “berdosa” demi mengingatkan dosen.

### 2. Penjelasan Makna

#### 🏛️ Versi Formal (Dokumentasi / SRS / Proposal)
> **SiPenDosa** adalah Sistem Pengingat Dosen yang bertugas mengirimkan pesan pengingat jadwal perkuliahan secara otomatis dan terjadwal kepada dosen pengampu mata kuliah. Nama ini diambil dari akronim resmi: **"Sistem Pengingat Dosen Saatnya"**.

#### ☕ Versi Asli / Karakter (Jiwa & Realita Mahasiswa)
> Nama **SiPenDosa** lahir dari realita dan kegelisahan mahasiswa:
>
> *Mengingatkan dosen berkali-kali tentang jadwal kuliah, perkuliahan pengganti, atau kepastian kelas kadang terasa seperti **“berdosa”** — takut dianggap mengganggu waktu istirahat dosen, takut dinilai kurang sopan, atau takut dicap tidak mandiri.*
>
> *Tapi di sisi lain, mahasiswa juga butuh kepastian agar perkuliahan tetap berjalan tertib.*
>
> Maka lahirlah **SiPenDosa**:  
> **Sebuah asisten otomasi cerdas yang rela menanggung beban "dosa" tersebut, bertutur kata santun dan terjadwal, sehingga mahasiswa tidak perlu lagi merasa sungkan!**

---

## 🌟 Fitur Utama (OverPower Engine)

1. **Unofficial WhatsApp Engine Terpercaya (`go.mau.fi/whatsmeow`)**:
   - Pemindaian QR Code interaktif di Terminal (dengan ASCII art cyberpunk) maupun langsung melalui Web Dashboard via WebSocket secara *real-time*.
   - Penyimpanan sesi terisolasi di database SQLite terpisah (`session/whatsapp.db`).
   - Deteksi pemutusan koneksi otomatis dengan *exponential backoff auto-reconnect*.
   - **Anti-Ban Shield**:
     - Normalisasi format nomor otomatis (`08...` &rarr; `628...@s.whatsapp.net` & ID grup `@g.us`).
     - Simulasi kehadiran manusia (*human presence simulation*): Menandai status *Online*, simulasi *Composing / Sedang mengetik...* dengan jeda acak (*jitter* 1.5–2.5 detik) sebelum pengiriman pesan fisik.

2. **Smart Scheduler (H-1 & Timezone Aware)**:
   - Pengingat perkuliahan mode **H-1** (satu hari sebelumnya) atau **H-0** (hari H).
   - Zona waktu bawaan **Asia/Makassar (WITA, UTC+8)** (dapat disesuaikan ke WIB atau WIT melalui dashboard atau `.env`).
   - Pengecekan daftar **Hari Libur**: Jika jadwal jatuh pada tanggal libur yang terdaftar, pesan pengingat dilewati secara otomatis tanpa mengganggu dosen.
   - Pengecekan rentang jam kirim operasional (default `08:00` s.d. `16:00`).
   - **Hitung Mundur (*Countdown*) Real-time** menuju pengiriman jadwal berikutnya di dashboard.
   - Fitur **Manual Trigger**: Eksekusi instan jadwal sewaktu-waktu.
   - **Dry-Run Mode**: Mode simulasi global maupun per-jadwal (pesan dicatat dalam antrian dan riwayat tanpa pengiriman fisik ke WhatsApp).

3. **Template Engine Dinamis**:
   - Go Template Engine lengkap dengan fungsi bantuan string, tanggal, dan kapitalisasi.
   - Variabel bawaan:
     - `{{.NamaDosen}}`, `{{.NamaMahasiswa}}`, `{{.NIM}}`
     - `{{.Matkul}}`, `{{.Hari}}`, `{{.Tanggal}}`, `{{.JamMulai}}`, `{{.JamSelesai}}`
     - `{{.Lokasi}}`, `{{.LinkGroup}}`, `{{.WaktuSekarang}}`, `{{.HariDalamBahasa}}`
   - Editor dengan tombol sisip variabel cepat dan **Live Preview** interaktif yang langsung ter-render saat mengetik.
   - **Audit Trail & Versioning**: Setiap modifikasi template tersimpan riwayat versinya untuk kemudahan audit.

4. **Message Queue & Anti-Ban Rate Limiter**:
   - Antrian pesan terjadwal dengan pemrosesan mandiri di latar belakang (*background worker*).
   - Rate limiter cerdas dengan *jitter* acak 5–15 detik antar pesan.
   - Mekanisme *auto-retry* otomatis hingga 3x percobaan dengan jeda bertingkat (*exponential backoff*) jika terjadi gangguan jaringan.
   - Kemampuan pembatalan (*cancel*) atau paksa kirim segera (*send now*) melalui antarmuka web.

5. **Web Dashboard Modern & Responsif**:
   - DaisyUI Theme `night` (Dark Mode elegan) + Tailwind CSS + HTMX (interaktivitas tinggi tanpa beban komputasi JS berlebih).
   - WebSocket real-time untuk status koneksi WhatsApp, QR Code, hitung mundur, dan notifikasi Toast pop-up.
   - Manajemen Dosen, Kontak, Jadwal Kuliah, Template, Antrian, dan Pengaturan.
   - Fitur pengunduhan cadangan database SQLite (`.db`) langsung dari menu pengaturan.
   - Log viewer aktivitas sistem dan audit trail yang rapi.

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

Buat file konfigurasi `/etc/nginx/sites-available/sipendosa`:

```nginx
server {
    server_name sipendosa.domainanda.com;

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
sudo ln -s /etc/nginx/sites-available/sipendosa /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d sipendosa.domainanda.com
```

### Opsi B: Menggunakan Caddy (Otomatis SSL)

Cukup tambahkan blok berikut ke `/etc/caddy/Caddyfile`:

```caddy
sipendosa.domainanda.com {
    reverse_proxy localhost:8473
}
```

Lalu reload:
```bash
sudo systemctl reload caddy
```

---

## 📱 Alur Penggunaan Pertama Kali

1. **Buka Dashboard**: Akses `http://IP_VPS:8473` atau `https://sipendosa.domainanda.com`.
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
