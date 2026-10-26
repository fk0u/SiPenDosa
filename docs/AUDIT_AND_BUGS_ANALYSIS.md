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
| **BUG-13** | `cmd/installer` | **Tinggi (High)** | **Kompilasi Gagal pada `//go:embed`:** `main.go` meng-embed `sipen.exe` tanpa build tag, menyebabkan `go test ./...` dan build di luar pipeline rilis Windows gagal total (`pattern sipen.exe: no matching files found`). |
| **BUG-14** | `internal/web` (API Auth) | **Kritis (Critical)** | **Bypass 2FA via `/api/auth/login`:** Endpoint `APITokenLoginHandler` tidak memvalidasi passcode 2FA untuk akun yang mengaktifkan 2FA, memungkinkan bypass otentikasi dua faktor penuh. |
| **BUG-15** | `internal/web` (Router) | **Tinggi (High)** | **Kegagalan Inisialisasi WhatsApp Client pada CLI & Android:** `/api/internal/pair-phone` dan `/api/whatsapp/pair-phone` memanggil `getActiveWAClient(r)` tanpa user context, selalu merespon 500 (`WhatsApp client not available`) pada CLI `sipen pair` dan menolak remote request dengan 403. |
| **BUG-16** | `internal/auth` | **Sedang (Medium)** | **Kegagalan Logout Cookie pada Koneksi HTTP / Localhost:** `ClearSessionCookie` menetapkan `Secure: true` secara statis, menyebabkan browser menolak menghapus cookie saat diakses via HTTP (`localhost:8473` atau IP LAN). |
| **BUG-17** | `internal/scheduler` | **Sedang (Medium)** | **Distorsi Tanggal Kuliah pada Pemicu Manual & Countdown:** `calculateNextLectureDate` menambahkan offset 7 hari saat pemicu dijalankan pada hari kuliah itu sendiri (H-0), dan countdown preview menampilkan tanggal pekan depan. |
| **BUG-18** | `internal/scheduler` | **Sedang (Medium)** | **Pengiriman Pengingat Kelas Kedaluwarsa pada H-0:** `evaluateTick` tetap mengirimkan notifikasi untuk mata kuliah yang jam mulainya telah berlalu jika daemon dinyalakan siang/sore hari. |
| **BUG-19** | `internal/tunnel` | **Tinggi (High)** | **Kegagalan Unduh & Ekstraksi Cloudflared:** Derivasi URL checksum menghasilkan 404 (`checksums.txtlinux-amd64`), Cloudflare tidak menyediakan aset `checksums.txt`, dan arsip `.tgz` macOS disimpan mentah tanpa dekompresi tar.gz. |
| **BUG-20** | `internal/updater` | **Sedang (Medium)** | **Kegagalan Self-Update Lintas Filesystem (EXDEV):** `os.Rename` gagal saat fallback direktori sementara berada di mount point berbeda (`/tmp` tmpfs vs target binary). |
| **BUG-21** | `native/bridge` (C++) | **Sedang (Medium)** | **Deteksi Liveness Proses Keliru & Buffer Overflow Risk:** `sipen_is_process_alive` mengabaikan `errno == EPERM`, dan `sipen_check_socket_latency_us` tidak memvalidasi batas `sock >= FD_SETSIZE` sebelum `select()`. |
| **BUG-22** | `internal/web` (Renderer) | **Rendah (Low)** | **Kerusakan Karakter Multibyte (UTF-8) pada Truncate:** Fungsi pembantu template `truncate` memotong byte alih-alih rune, berisiko merusak encoding emoji / UTF-8. |

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

---

### 12. BUG-13: Build Constraint pada `cmd/installer`
* **Analisis Masalah:** `cmd/installer/main.go` menggunakan direktif `//go:embed sipen.exe` tanpa build tag. Karena `sipen.exe` di-ignore oleh `.gitignore`, perintah `go test ./...` atau build lintas platform di mesin pengembang/CI langsung gagal (`pattern sipen.exe: no matching files found`).
* **Solusi Perbaikan:**
  Tambahkan build tag `//go:build windows && installer` pada `cmd/installer/main.go` dan perbarui target `installer-windows` di `Makefile` dengan flag `-tags installer`.

---

### 13. BUG-14: Penegakan 2FA (TOTP) pada API Login (`/api/auth/login`)
* **Analisis Masalah:** Handler `APITokenLoginHandler` langsung menerbitkan token sesi hanya berdasarkan verifikasi username & password tanpa mengecek flag `user.TwoFactorEnabled`, memungkinkan bypass 2FA melalui permintaan API.
* **Solusi Perbaikan:**
  Tambahkan pengecekan `user.TwoFactorEnabled`. Jika aktif, validasi field `passcode` atau `code` via `totp.ValidatePasscode()`. Jika tidak disertakan atau salah, tolak dengan HTTP 401 dan flag `2fa_required`.

---

### 14. BUG-15: Autentikasi & Scoping pada WhatsApp Pair Phone Loopback
* **Analisis Masalah:** Route `/api/internal/pair-phone` dan `/api/whatsapp/pair-phone` memanggil `getActiveWAClient(r)` yang mengharuskan user terikat pada context. Namun, route ini berada di luar middleware `RequireAuth`, sehingga pemanggilan CLI `sipen pair` lokal selalu merespon HTTP 500 (`WhatsApp client not available`) dan pemanggilan remote ber-token tertolak HTTP 403.
* **Solusi Perbaikan:**
  Ekstrak dan validasi cookie/Bearer token pada `loopbackPairPhone`. Jika valid, inject user ke context. Jika request berasal dari loopback lokal tanpa token (CLI/Android lokal), otomatis pasangkan ke akun utama (User ID 1). Jika bukan loopback dan tidak ada sesi, tolak dengan HTTP 403 Forbidden.

---

### 15. BUG-16: Flag Secure Cookie pada Penghapusan Sesi HTTP
* **Analisis Masalah:** `ClearSessionCookie` menetapkan `Secure: true`. Pada koneksi HTTP murni (`http://localhost:8473` atau IP jaringan lokal), browser modern menolak header `Set-Cookie` dengan flag `Secure` sesuai RFC 6265bis, sehingga sesi gagal di-logout di browser.
* **Solusi Perbaikan:**
  Ubah nilai `Secure: false` pada `ClearSessionCookie` agar pembersihan cookie berhasil di koneksi HTTP lokal maupun HTTPS.

---

### 16. BUG-17: Kalkulasi Tanggal Kuliah pada Pemicu Manual & Countdown
* **Analisis Masalah:** `calculateNextLectureDate` selalu menambahkan 7 hari jika `daysUntil == 0`. Akibatnya, pemicu manual untuk jadwal perkuliahan hari ini (H-0) menggeser tanggal perkuliahan di pesan WhatsApp ke pekan depan. Selain itu, countdown preview pesan dihitung dari `now` alih-alih `nearestTime`.
* **Solusi Perbaikan:**
  Pada `TriggerScheduleManually`, gunakan `now` jika hari ini adalah hari kuliah (`now.Weekday() == sc.DayOfWeek`). Pada `updateCountdown`, turunkan tanggal perkuliahan langsung dari `nearestTime` (+1 hari untuk mode H-1).

---

### 17. BUG-18: Pencegahan Pengiriman Jadwal Kuliah Kedaluwarsa
* **Analisis Masalah:** Pada evaluasi per menit (`evaluateTick`), jadwal mode H-0 tetap dikirimkan jika jam saat ini `>= sc.SendAtTime` tanpa mengecek apakah jam mulai kuliah (`sc.StartTime`) telah berlalu. Jika server baru dinyalakan sore hari, bot akan mengirimkan pengingat untuk kuliah pagi yang sudah selesai.
* **Solusi Perbaikan:**
  Tambahkan pengecekan: jika `sc.Mode != "H-1" && sc.StartTime != "" && currentTimeStr > sc.StartTime`, lewati jadwal hari ini dengan log diagnostik.

---

### 18. BUG-19: Unduh, Ekstraksi, dan Verifikasi Checksum Cloudflared
* **Analisis Masalah:**
  1. `strings.Replace(downloadURL, "/cloudflared-", "/checksums.txt", 1)` menghasilkan URL invalid (`.../checksums.txtlinux-amd64` 404).
  2. Cloudflare tidak merilis file `checksums.txt`, melainkan menyertakan daftar hash SHA256 di body rilis GitHub.
  3. Pada macOS, file unduhan berformat `.tgz` (tar.gz), tetapi langsung ditulis sebagai file biner mentah tanpa dekompresi.
* **Solusi Perbaikan:**
  Ambil hash SHA256 dari release body GitHub API, dan tambahkan rutin dekompresi gzip + tar untuk mengekstrak biner `cloudflared` pada macOS.

---

### 19. BUG-20: Penanganan EXDEV (Cross-Device Link) pada Self-Update
* **Analisis Masalah:** Pada `ApplyBinarySelfUpdate`, jika file sementara dibuat di `/tmp` (tmpfs) sedangkan binary terpasang di filesystem disk biasa, `os.Rename` menghasilkan error POSIX `EXDEV: cross-device link`.
* **Solusi Perbaikan:**
  Tangani kegagalan `os.Rename` dengan fallback penyalinan byte (`copyFile`) dan pembaruan permission `0755`.

---

### 20. BUG-21: Deteksi Liveness POSIX & Socket FD Bounds Check (C++ Bridge)
* **Analisis Masalah:**
  - `sipen_is_process_alive(pid)` mengembalikan `kill(pid, 0) == 0`. Jika proses ada tetapi dimiliki pengguna lain/root, `kill` mengembalikan -1 dengan `errno == EPERM`, yang salah dianggap tidak aktif (`false`).
  - `sipen_check_socket_latency_us` memanggil `FD_SET(sock, &write_fds)` tanpa memastikan `sock < FD_SETSIZE`, berisiko memory corruption jika socket descriptor `>= 1024`.
* **Solusi Perbaikan:**
  Periksa `(kill(pid, 0) == 0 || errno == EPERM)`. Tambahkan validasi `sock >= FD_SETSIZE` dan `timeout_ms <= 0` sebelum `select()`.

---

### 21. BUG-22: Truncate Template Berbasis Karakter Rune (UTF-8)
* **Analisis Masalah:** Fungsi pembantu template `truncate` melakukan pemotongan string via byte slicing `s[:maxLen]`. Karakter multibyte (seperti emoji dan karakter aksen) berisiko terpotong di tengah byte encoding UTF-8, menghasilkan string rusak.
* **Solusi Perbaikan:**
  Konversikan string ke `[]rune` sebelum melakukan slicing `runes[:maxLen]`.

---

### 22. BUG-23: Android Missing `PowerManager` Import pada `MainActivity.java`
* **Analisis Masalah:** Pada `MainActivity.java`, inisialisasi pengecualian optimasi baterai (`Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS`) memanggil `(PowerManager) getSystemService(Context.POWER_SERVICE)` tanpa mengimpor `android.os.PowerManager`, menyebabkan kompilasi SDK `javac` gagal pada pipeline rilis Android.
* **Solusi Perbaikan:**
  Tambahkan `import android.os.PowerManager;` pada daftar impor kelas Android.

---

### 23. BUG-24: Kegagalan Pembaruan Versi dari 1.2.0 ke 1.4.1 (Zombie Daemon, Port Hijack, & Missing Version Check)
* **Analisis Masalah:**
  1. Pada sistem yang sebelumnya terpasang v1.2.0, daemon `com.sipendosa.daemon` berjalan di latar belakang (KeepAlive) memegang port `8473`.
  2. Saat pengguna memasang v1.4.1 (PKG, DMG, atau installer script), skrip instalasi lama tidak menghentikan proses `sipen` yang aktif di RAM dan tidak memuat ulang (*reload*) job launchd di sesi pengguna aktif.
  3. Aplikasi native controller macOS (`SiPenDosaApp.swift`) hanya menguji apakah port `8473` terbuka (`latency >= 0`). Karena port masih dipegang proses v1.2.0, aplikasi menganggap engine sudah aktif dan tidak meluncurkan biner v1.4.1 yang tertanam. Dashboard yang diakses pun tetap menampilkan backend dan antarmuka usang v1.2.0.
  4. Pengenalan mode akses terminal secara default (`SIPEN_UI=terminal`) pada v1.4.1 menyebabkan proses biner langsung keluar jika diluncurkan tanpa argumen CLI `web`.
* **Solusi Perbaikan:**
  - Pada `SiPenDosaApp.swift`: Tambahkan pengecekan versi otomatis ke `/api/system/version`. Jika proses yang mendengarkan di port `8473` versinya tidak sesuai atau usang, hentikan paksa proses lama (`kill -9`), sinkronkan biner baru ke disk, dan luncurkan biner v1.4.1 dengan argumen `web`.
  - Pada `scripts/build-pkg.sh` (postinstall): Hentikan semua proses lama, unload LaunchAgent di domain pengguna (`launchctl asuser`), sinkronkan biner baru ke seluruh lokasi instalasi (`.local/share/sipendosa` dan `~/Applications/SiPenDosa`), pastikan `.env` memiliki `SIPEN_UI=web`, pasang plist dengan argumen `web`, dan kickstart daemon baru.
  - Pada `install-macos.sh` & `cmd/installer/main.go`: Hentikan proses lama sebelum menulis biner baru dan pastikan mode `web` diaktifkan.




