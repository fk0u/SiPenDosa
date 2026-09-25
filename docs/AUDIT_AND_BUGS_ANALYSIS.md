# 🔍 Laporan Audit Mendalam & Analisis Bug Sistem SiPenDosa

Dokumen ini memuat perincian analisis teknis, temuan kelemahan sistem (*bugs & edge cases*), serta rencana perbaikan menyeluruh pada seluruh layer aplikasi **SiPenDosa** (Core Backend Go, Web & Security, macOS Native Companion, Mobile Android, dan Mobile iOS).

---

## 📋 Ringkasan Temuan Audit

| ID | Komponen | Tingkat Keparahan | Deskripsi Masalah |
| :--- | :--- | :---: | :--- |
| **BUG-01** | `internal/queue` & `internal/store` | **Tinggi (Critical)** | **Race Condition & Duplikasi Pengiriman Pesan:** `ProcessImmediate()` berjalan bersamaan dengan `workerLoop` tanpa atomic claim lock. Status `processing` tidak diverifikasi secara atomik, berpotensi mengirim pesan ganda ke WhatsApp dosen. |
| **BUG-02** | `internal/queue` | **Sedang (Medium)** | **Graceful Shutdown Terblokir:** Jeda jitter anti-ban menggunakan `time.Sleep()` yang mengabaikan `stopChan`, menahan shutdown server hingga 15 detik. |
| **BUG-03** | `internal/whatsapp` | **Tinggi (High)** | **Goroutine Leak pada Auto-Reconnect:** Pemanggilan `c.Start()` atau `c.Reconnect()` memicu `go c.keepAliveLoop()` tanpa batasan, menyebabkan tumpukan goroutine loop duplikat di latar belakang. |
| **BUG-04** | `internal/whatsapp` | **Sedang (Medium)** | **Potensi Panic pada Double-Close:** Pemanggilan `c.Close()` menutup `stopChan` tanpa perlindungan `sync.Once`, memicu panic runtime jika ditutup lebih dari sekali. |
| **BUG-05** | `internal/web` (Router & Handlers) | **Tinggi (High)** | **Ketiadaan Endpoint JSON API Mobile:** Mobile Flutter dan iOS Background Worker memanggil `/api/stats`, `/api/whatsapp/status`, `/api/contacts`, `/api/schedules`, `/api/queue`, dan `/api/ws`, namun endpoint ini belum terdaftar di router (hanya rute HTML web). |
| **BUG-06** | `internal/web` (Security) | **Tinggi (High)** | **Penolakan CSRF pada Reverse Proxy / Tunnel:** `CSRFOriginMiddleware` mengabaikan `X-Forwarded-Host` dan `isAllowedHost` hanya mengizinkan `localhost`/`127.0.0.1`. Mengakibatkan HTTP 403 Forbidden pada akses via Nginx, Caddy, atau Cloudflare Tunnel. |
| **BUG-07** | `internal/auth` | **Sedang (Medium)** | **Respon Redirection 303 untuk API Request:** `RequireAuth` selalu melakukan redirect HTML ke `/login` dan tidak mendukung header `Authorization: Bearer <token>`, mematahkan klien mobile REST. |
| **BUG-08** | `native/macos` | **Rendah (Low)** | **Hardcoded Port pada Companion App macOS:** `SiPenDosaApp.swift` mematok port `8473` secara statis, menyebabkan status offline jika pengguna mengubah port di `.env`. |
| **BUG-09** | `mobile/android` | **Sedang (Medium)** | **IP LAN Mengembalikan 0.0.0.0 pada Android 11+:** Metode lama `getConnectionInfo().getIpAddress()` tidak lagi mengembalikan alamat IP aktual pada Android modern tanpa izin lokasi. |
| **BUG-10** | `internal/scheduler` | **Sedang (Medium)** | **Kalkulasi Countdown Mengabaikan Hari Libur:** `calculateNextFireTime` tidak memvalidasi kalender libur, sehingga countdown dapat menunjuk pada tanggal yang nantinya akan dilewati (*skipped*). |
| **BUG-11** | `internal/whatsapp` | **Rendah (Low)** | **Backoff Sleep Terblokir pada Auto-Reconnect:** `keepAliveLoop` menggunakan `time.Sleep(backoff)` saat reconnect gagal, menahan goroutine exit jika client di-close. |
| **BUG-12** | `internal/queue` | **Sedang (Medium)** | **Validasi Status pada ProcessImmediate:** Endpoint pemicu antrian langsung tidak memverifikasi apakah status pesan masih `pending`, berisiko mengeksekusi ulang pesan yang telah terkirim. |

---

## 🛠️ Rincian Analisis & Solusi Perbaikan

### 1. BUG-01: Atomic Claiming pada Message Queue
* **Analisis Masalah:** Di `internal/queue/queue.go`, saat fungsi `processSingleMessage` dipanggil (baik dari `workerLoop` maupun via tombol *Kirim Sekarang* `ProcessImmediate`), status pesan diubah via `UpdateQueueStatus(id, "processing", ...)`. Jika kedua proses berjalan serentak, keduanya dapat meloloskan pesan yang sama dan memanggil `waClient.SendMessage` dua kali.
* **Solusi Perbaikan:**
  Tambahkan query atomik di `Store`:
  ```sql
  UPDATE queue_messages SET status = 'processing', updated_at = ? WHERE id = ? AND status = 'pending'
  ```
  Jika baris yang terpengaruh (*rows affected*) bernilai 0, eksekusi langsung dibatalkan karena pesan telah diklaim oleh worker lain.

---

### 2. BUG-02: Graceful Shutdown Non-Blocking pada Queue Jitter
* **Analisis Masalah:** Pada akhir `processSingleMessage`, antrian menunggu jeda acak:
  `time.Sleep(time.Duration(delaySec) * time.Second)`
  Jika server menerima sinyal SIGINT/SIGTERM saat jeda ini aktif, thread tertahan dan proses shutdown tertunda.
* **Solusi Perbaikan:**
  Ganti `time.Sleep` dengan `select`:
  ```go
  select {
  case <-m.stopChan:
      return
  case <-time.After(time.Duration(delaySec) * time.Second):
  }
  ```

---

### 3. BUG-03 & BUG-04: Goroutine Leak & Channel Panic pada WhatsApp Client
* **Analisis Masalah:**
  - Di `client.go`, baris `go c.keepAliveLoop()` dieksekusi setiap kali `Start()` dipanggil.
  - `Close()` langsung memanggil `close(c.stopChan)` tanpa pengaman idempoten.
* **Solusi Perbaikan:**
  - Tambahkan flag `keepAliveRunning bool` yang dilindungi mutex untuk memastikan `keepAliveLoop` hanya berjalan satu instance per client.
  - Gunakan `sync.Once` untuk penutupan `stopChan`.

---

### 4. BUG-05: Implementasi REST JSON API Handlers Lengkap
* **Analisis Masalah:** Klien mobile (Flutter `api_service.dart` dan iOS `BackgroundSyncManager.swift`) membutuhkan data berformat JSON murni untuk pemantauan latar belakang dan sinkronisasi antrian.
* **Solusi Perbaikan:**
  Implementasikan dan daftarkan endpoint berikut di `internal/web/router.go`:
  - `GET /api/ws` (WebSocket live telemetry alias)
  - `GET /api/stats` (JSON statistik: total sent, failed, pending, countdown)
  - `GET /api/whatsapp/status` (JSON status: state, phone, pushName, qr)
  - `GET /api/contacts` (JSON daftar kontak terintegrasi)
  - `GET /api/schedules` (JSON daftar jadwal perkuliahan)
  - `GET /api/queue` (JSON antrian pesan dan status kirim)
  - `POST /api/queue/process` (Pemicu pengiriman antrian instan)

---

### 5. BUG-06: Dukungan Reverse Proxy (`X-Forwarded-Host`) & Base Host Matching
* **Analisis Masalah:** Pada arsitektur reverse proxy (Nginx, Traefik, Caddy, Cloudflare Tunnels), header `Origin` berasal dari domain publik (`https://kampus.ac.id`), sedangkan `r.Host` lokal berisi `127.0.0.1:8473`. Pengecekan CSRF saat ini langsung menolak dengan 403 Forbidden.
* **Solusi Perbaikan:**
  - Ekstrak host dari `X-Forwarded-Host` jika tersedia.
  - Pada `isAllowedHost`, bandingkan nama host dasar (*base host*) terlepas dari port (`net.SplitHostPort`).

---

### 6. BUG-07: Respon JSON 401 & Header Authorization untuk API
* **Analisis Masalah:** Middleware `RequireAuth` selalu melakukan redirect 303 ke `/login` jika cookie tidak ada. Ini membingungkan aplikasi mobile yang mengharapkan status HTTP 401 Unauthorized dalam format JSON.
* **Solusi Perbaikan:**
  Periksa apakah request menuju `/api/...` atau memiliki header `Accept: application/json`. Jika tidak terotentikasi, kembalikan JSON HTTP 401. Dukung pula header `Authorization: Bearer <token>`.

---

### 7. BUG-08: Parsing Port Dinamis di macOS Menu Bar Companion
* **Analisis Masalah:** Jika pengguna mengubah `PORT` di `~/.local/share/sipendosa/.env`, aplikasi companion Swift tetap mengecek port default 8473.
* **Solusi Perbaikan:**
  Baca file `.env` di direktori data pada saat inisialisasi Swift dan ekstrak variabel `PORT=`, kemudian gunakan port tersebut untuk socket bridge dan pembukaan dashboard.

---

### 8. BUG-09: Robust LAN IP Detection pada Android
* **Analisis Masalah:** `WifiManager.getConnectionInfo().getIpAddress()` mengembalikan `0.0.0.0` pada Android 11+ jika izin lokasi (*fine location*) tidak diaktifkan.
* **Solusi Perbaikan:**
  Gunakan iterasi `NetworkInterface.getNetworkInterfaces()` untuk menemukan IP IPv4 non-loopback yang aktif (`wlan0`, `eth0`, `ap0`).

---

### 9. BUG-10: Pengecekan Hari Libur pada Perhitungan Countdown
* **Analisis Masalah:** Widget countdown pada dashboard dapat menghitung mundur ke tanggal perkuliahan yang jatuh pada hari libur nasional atau cuti bersama.
* **Solusi Perbaikan:**
  Pada fungsi `calculateNextFireTime`, lakukan iterasi pengecekan hari libur hingga beberapa minggu ke depan via `store.IsHoliday()`. Jika hari perkuliahan bertepatan dengan libur, tanggal fireTime digeser ke pekan berikutnya.

---

### 10. BUG-11: Backoff Non-Blocking pada WhatsApp Auto-Reconnect
* **Analisis Masalah:** Di `keepAliveLoop`, jika reconnect gagal, thread mengeksekusi `time.Sleep(backoff)` yang tidak mendengarkan `stopChan`. Jika sistem melakukan shutdown, thread ini tertahan hingga durasi backoff berakhir (maksimal 60 detik).
* **Solusi Perbaikan:**
  Ganti pemanggilan `time.Sleep` dengan konstruksi `select`:
  ```go
  select {
  case <-c.stopChan:
      return
  case <-time.After(backoff):
  }
  ```

---

### 11. BUG-12: Validasi Status Pesan pada ProcessImmediate
* **Analisis Masalah:** Fungsi `ProcessImmediate(id int64)` dapat dipanggil berulang kali untuk pesan yang sama tanpa memeriksa apakah statusnya masih `pending`. Jika dipanggil saat status pesan sudah `processing` atau `sent`, pesan tersebut dapat dieksekusi ulang secara ganda.
* **Solusi Perbaikan:**
  Validasi status pesan: pastikan `msg.Status == "pending"` sebelum meluncurkan goroutine `processSingleMessage`. Jika bukan `pending`, kembalikan error deskriptif.

