# 📐 Dokumentasi Desain Sistem, Diagram Alur & DFD Lengkap SiPenDosa (v1.1.0)
*Sistem Pengingat Dosen Saatnya — Universal Multi-Platform & Multi-Device WhatsApp Automation*

---

## 📑 Daftar Isi
1. [Ikhtisar Arsitektur Sistem (High-Level Topology)](#1-ikhtisar-arsitektur-sistem-high-level-topology)
2. [Data Flow Diagram (DFD) Seluruh Level](#2-data-flow-diagram-dfd-seluruh-level)
   - [2.1 DFD Level 0 (Context Diagram)](#21-dfd-level-0-context-diagram)
   - [2.2 DFD Level 1 (Macro Subsystems Decomposition)](#22-dfd-level-1-macro-subsystems-decomposition)
   - [2.3 DFD Level 2 — Proses 2.0: WhatsApp Connection Engine (QR & Pairing)](#23-dfd-level-2--proses-20-whatsapp-connection-engine-qr--pairing)
   - [2.4 DFD Level 2 — Proses 4.0 & 6.0: Smart Scheduler & Anti-Ban Dispatch Queue](#24-dfd-level-2--proses-40--60-smart-scheduler--anti-ban-dispatch-queue)
   - [2.5 DFD Level 2 — Proses 1.0: Autentikasi, RBAC & OWASP Security Pipeline](#25-dfd-level-2--proses-10-autentikasi-rbac--owasp-security-pipeline)
3. [Diagram Mesin Status (Flow State Diagrams)](#3-diagram-mesin-status-flow-state-diagrams)
   - [3.1 WhatsApp Connection State Machine](#31-whatsapp-connection-state-machine)
   - [3.2 Message Queue Lifecycle State Machine](#32-message-queue-lifecycle-state-machine)
   - [3.3 Schedule Job Lifecycle State Machine](#33-schedule-job-lifecycle-state-machine)
   - [3.4 Android Daemon & Wakelock Service Lifecycle](#34-android-daemon--wakelock-service-lifecycle)
4. [User Flow & Alur Pengguna Interaktif](#4-user-flow--alur-pengguna-interaktif)
   - [4.1 Alur Onboarding & Registrasi SuperAdmin Pertama](#41-alur-onboarding--registrasi-superadmin-pertama)
   - [4.2 Alur Taut Perangkat WhatsApp (QR vs Pairing Code Nomor HP)](#42-alur-taut-perangkat-whatsapp-qr-vs-pairing-code-nomor-hp)
   - [4.3 Alur Pengelolaan Jadwal Kuliah & Template Dinamis](#43-alur-pengelolaan-jadwal-kuliah--template-dinamis)
   - [4.4 Alur Eksekusi Headless via Android Termux CLI](#44-alur-eksekusi-headless-via-android-termux-cli)
5. [Diagram Urutan Interaksi (Sequence Diagrams)](#5-diagram-urutan-interaksi-sequence-diagrams)
   - [5.1 Sequence: Login via Phone Pairing Code](#51-sequence-login-via-phone-pairing-code)
   - [5.2 Sequence: Smart Scheduler -> Evaluasi Hari Libur -> Anti-Ban Dispatch](#52-sequence-smart-scheduler---evaluasi-hari-libur---anti-ban-dispatch)
   - [5.3 Sequence: OWASP Security Interceptor Pipeline](#53-sequence-owasp-security-interceptor-pipeline)
6. [Diagram Relasi Entitas (Entity-Relationship Diagram / ERD)](#6-diagram-relasi-entitas-entity-relationship-diagram--erd)

---

## 1. Ikhtisar Arsitektur Sistem (High-Level Topology)

SiPenDosa mengusung arsitektur **Zero-CGO Pure Go Monolith** dengan antarmuka native adaptif untuk macOS (Swift + C++17 Socket Bridge), Android (Java Foreground Service + Wakelock), Linux (Systemd Service), dan Termux (Headless Shell CLI).

```mermaid
graph TB
    subgraph ClientLayers["Lapisan Antarmuka Pengguna (Client Layer)"]
        WebUI["Web Dashboard (HTML5 / Tailwind / HTMX)"]
        MobileApp["Android App (WebView + Native Service)"]
        TermuxCLI["Termux / Terminal Shell CLI ('sipen pair')"]
        MacApp["macOS Status Bar App (Swift AppKit)"]
    end

    subgraph SecurityGate["Lapisan Pertahanan Keamanan (OWASP Shield)"]
        SecHeaders["Security Headers Middleware (CSP, HSTS, Sniff-Guard)"]
        RateLimiter["Sliding-Window IP Rate Limiter (10 req/min)"]
        CSRFGuard["CSRF Origin / Referer Matcher"]
        RBAC["RBAC Enforcement (SuperAdmin vs Admin)"]
    end

    subgraph CoreEngine["SiPenDosa Engine (Pure Go Core Runtime)"]
        Router["Chi HTTP Router (:8473)"]
        AuthSvc["Authentication & Session Manager"]
        Hub["Realtime WebSocket Hub (Pub/Sub)"]
        Sch["Smart Scheduler (H-1 & Timezone Aware)"]
        TmplEng["Dynamic Template Interpolator Engine"]
        QM["Anti-Ban Queue Manager (Jitter & Presence)"]
        WACli["WhatsApp Multi-Device Client (whatsmeow)"]
    end

    subgraph StorageLayer["Lapisan Persistensi Data (Storage Layer)"]
        AppDB[("SQLite WAL App Database (data/sipen.db)")]
        WADB[("SQLite WAL Session Database (session/whatsapp.db)")]
    end

    subgraph ExternalEntities["Entitas Eksternal"]
        WAServer["WhatsApp Cloud WebSocket Gateway (Noise Protocol)"]
        DosenPhone["Ponsel Dosen / Mahasiswa Penerima"]
    end

    WebUI --> SecHeaders
    MobileApp --> SecHeaders
    MacApp --> SecHeaders
    TermuxCLI -.->|Internal Loopback| Router

    SecHeaders --> RateLimiter
    RateLimiter --> CSRFGuard
    CSRFGuard --> RBAC
    RBAC --> Router

    Router --> AuthSvc
    Router --> Hub
    Router --> QM
    Router --> Sch

    Sch --> AppDB
    Sch --> TmplEng
    Sch --> QM

    QM --> AppDB
    QM --> WACli

    WACli --> WADB
    WACli -->|TLS / Noise Encrypted| WAServer
    WAServer -->|Pesan WhatsApp Terkirim| DosenPhone

    Hub -.->|Realtime Push Event| WebUI
    Hub -.->|Realtime Log Push| MobileApp
```

---

## 2. Data Flow Diagram (DFD) Seluruh Level

### 2.1 DFD Level 0 (Context Diagram)

Context diagram memetakan batas sistem (*system boundary*) antara SiPenDosa dengan 4 entitas eksternal: **Pengguna / Mahasiswa (PJ Matkul)**, **SuperAdmin Sistem**, **WhatsApp Gateway Server**, dan **Dosen Tujuan**.

```mermaid
flowchart TD
    User["PJ Matkul / Admin Mahasiswa"]
    SuperAdmin["SuperAdmin Sistem"]
    Dosen["Dosen Pengampu Kuliah"]
    WAGateway["WhatsApp Server Cloud (Meta)"]
    
    System(("SISTEM SIPENDOSA (v1.1.0)"))

    User -->|"Input Kredensial Login"| System
    User -->|"Kelola Kontak Dosen & Jadwal Kuliah"| System
    User -->|"Pilih Template & Kustomisasi Kata Pesan"| System
    User -->|"Permintaan Pairing Code / Scan QR"| System

    System -->|"Tampilan Dashboard, Status Antrian & Log Realtime"| User
    System -->|"Kode Pairing (8 Digit) / Data QR Code"| User

    SuperAdmin -->|"Toggle Pendaftaran Pengguna & Manajemen User"| System
    SuperAdmin -->|"Ekspor / Unduh Cadangan Database (.db)"| System
    System -->|"Akses Konfigurasi Penuh & Log Audit Sistem"| SuperAdmin

    System -->|"Handshake Noise Protocol, Status Presence & Pesan Teks"| WAGateway
    WAGateway -->|"Feedback Pengiriman (ACK, Read Receipt, Disconnect)"| System

    WAGateway -->|"Pesan Pengingat Kuliah Santun Terformat"| Dosen
```

---

### 2.2 DFD Level 1 (Macro Subsystems Decomposition)

Mendekomposisi sistem menjadi 7 proses subsistem utama dan 2 data store SQLite independen.

```mermaid
flowchart TD
    User["Pengguna (Mahasiswa / PJ)"]
    SuperAdmin["SuperAdmin"]
    WAGateway["WhatsApp Server Cloud"]
    Dosen["Dosen Tujuan"]

    DS1[("D1: App Database (sipen.db)")]
    DS2[("D2: WA Session Store (whatsapp.db)")]

    P1["1.0 Autentikasi & Otorisasi RBAC"]
    P2["2.0 WhatsApp Device Synchronization Engine"]
    P3["3.0 Pengelolaan Data Dosen & Kontak"]
    P4["4.0 Smart Scheduling & Evaluasi Hari Libur"]
    P5["5.0 Dynamic Template Interpolation"]
    P6["6.0 Anti-Ban Rate-Limited Dispatch Queue"]
    P7["7.0 Telemetri, Audit Log & Realtime Hub"]

    %% Process 1
    User -->|"Kredensial Login"| P1
    P1 <-->|"Verifikasi Hash & Session Token"| DS1
    P1 -->|"Sesi Terverifikasi (Cookie)"| P3
    P1 -->|"Sesi Terverifikasi"| P4
    P1 -->|"Sesi Terverifikasi"| P5
    SuperAdmin -->|"Toggle Registrasi & Backup"| P1

    %% Process 2
    User -->|"Request Pairing Nomor HP / QR"| P2
    P2 <-->|"Simpan / Muat Kunci Sesi E2EE"| DS2
    P2 <-->|"Handshake Noise Socket"| WAGateway
    P2 -->|"Status Koneksi & Pairing Code"| P7

    %% Process 3
    User -->|"Data Dosen (Nama, Gelar, No HP)"| P3
    P3 <-->|"CRUD Kontak"| DS1

    %% Process 4
    User -->|"Atur Jadwal (Hari, Jam, Matkul, H-1)"| P4
    P4 <-->|"Baca Jadwal & Daftar Hari Libur"| DS1
    P4 -->|"Trigger Jadwal Aktif"| P5

    %% Process 5
    P5 <-->|"Ambil Template Pesan"| DS1
    P5 -->|"Pesan Terkomposisi Lengkap"| P6

    %% Process 6
    P6 <-->|"Kelola Status Antrian (Pending -> Sent)"| DS1
    P6 -->|"Kirim Pesan + Human Jitter"| P2
    P2 -->|"Pesan WhatsApp Fisik"| WAGateway
    WAGateway -->|"Pesan Diterima"| Dosen

    %% Process 7
    P6 -->|"Event Broadcast Antrian"| P7
    P2 -->|"Event Status WhatsApp"| P7
    P7 -->|"Streaming Log & Toast Realtime via WebSocket"| User
```

---

### 2.3 DFD Level 2 — Proses 2.0: WhatsApp Connection Engine (QR & Pairing)

Menampilkan detail proses penautan perangkat baru melalui pemindaian QR Code visual vs penautan kode telepon 8 karakter (Pairing Code) tanpa kamera.

```mermaid
flowchart TD
    User["Aplikasi Web / Android / Termux"]
    WAGateway["WhatsApp Server Cloud"]
    DS2[("D2: WhatsApp Session Store")]
    Hub["Realtime WebSocket Hub"]

    P2_1["2.1 Deteksi Status Sesi Aktif"]
    P2_2["2.2 Inisialisasi QR Channel Listener"]
    P2_3["2.3 Normalisasi Nomor HP ('08' -> '628')"]
    P2_4["2.4 Permintaan Pairing Code (PairPhone API)"]
    P2_5["2.5 Formatter Kode 8 Karakter (XXXX-XXXX)"]
    P2_6["2.6 Auto-Reconnect & Timeout Handler"]

    User -->|"Klik Scan QR / Buka Modal"| P2_1
    P2_1 <-->|"Cek Sesi ID"| DS2
    
    P2_1 -->|"Sesi Kosong (Mode QR)"| P2_2
    P2_2 <-->|"Minta QR Code Channel"| WAGateway
    P2_2 -->|"Data URL QR PNG"| Hub
    P2_2 -->|"Timeout (>60s)"| P2_6
    P2_6 -->|"Picu Ulang Reconnect"| P2_1

    User -->|"Input Nomor HP (Misal: 08123456789)"| P2_3
    P2_3 -->|"Nomor Bersih Standar Internasional"| P2_4
    P2_4 <-->|"PairPhone Request ('Chrome (Linux)')"| WAGateway
    WAGateway -->|"Raw 8-Char Pairing Code"| P2_4
    P2_4 --> P2_5
    P2_5 -->|"Tampilkan Kode & Kirim WS"| User
    P2_5 --> Hub

    WAGateway -->|"Konfirmasi Otorisasi Pairing dari HP Dosen"| P2_1
    P2_1 -->|"Simpan Kunci Sesi Baru"| DS2
    P2_1 -->|"Broadcast Status 'connected'"| Hub
```

---

### 2.4 DFD Level 2 — Proses 4.0 & 6.0: Smart Scheduler & Anti-Ban Dispatch Queue

Alur evaluasi jadwal cerdas (mengecek hari libur dan rentang waktu kerja) hingga pengiriman pesan dengan jeda kehadiran acak (*human jitter*).

```mermaid
flowchart TD
    Clock["Timer / Chrono Ticker (Tiap Menit)"]
    DS1[("D1: App Database")]
    WACli["WhatsApp Engine"]
    Hub["WebSocket Hub"]

    P4_1["4.1 Pengecekan Waktu & Konversi Zona (WITA/WIB/WIT)"]
    P4_2["4.2 Filter Hari Libur Nasional & Akhir Pekan"]
    P4_3["4.3 Evaluasi Jendela Waktu Kirim (08:00 - 16:00)"]
    P4_4["4.4 Filter Trigger Duplikat Hari Ini"]
    P5_1["5.1 Template Variable Binding & Politeness Greeting"]
    P6_1["6.1 Enqueue Pesan ke Database (Status: Pending)"]
    P6_2["6.2 Dispatcher Worker Loop"]
    P6_3["6.3 Simulasi Kehadiran Manusia (Composing Jitter 1.5s - 2.5s)"]
    P6_4["6.4 Eksekusi Pengiriman Pesan Fisik"]
    P6_5["6.5 Retry Handler & Exponential Backoff"]

    Clock --> P4_1
    P4_1 <-->|"Ambil Jadwal Aktif"| DS1
    P4_1 --> P4_2
    P4_2 <-->|"Cek Tabel holidays"| DS1
    P4_2 -->|"Bukan Libur"| P4_3
    P4_2 -.->|"Hari Libur (Suppressed)"| Hub
    P4_3 -->|"Dalam Jendela Kirim"| P4_4
    P4_4 <-->|"Cek Log Kirim Hari Ini"| DS1
    P4_4 -->|"Belum Terkirim"| P5_1

    P5_1 <-->|"Ambil Variabel: {dosen}, {matkul}, {jam}, dll"| DS1
    P5_1 -->|"Teks Pesan Final Siap"| P6_1
    P6_1 -->|"Insert Queue Record"| DS1
    P6_1 --> Hub

    P6_2 <-->|"Ambil Pesan Status 'pending' Urut Prioritas"| DS1
    P6_2 --> P6_3
    P6_3 -->|"Kirim Status 'ChatPresenceComposing'"| WACli
    P6_3 -->|"Delay Jitter Acak (Anti-Ban)"| P6_4
    P6_4 -->|"Panggil SendMessage"| WACli
    
    WACli -->|"Sukses Dikirim"| P6_2
    P6_2 -->|"Update Status: 'sent', sent_at: NOW"| DS1
    
    WACli -.->|"Gagal / Jaringan Putus"| P6_5
    P6_5 -->|"Retries < 3"| P6_2
    P6_5 -->|"Retries >= 3 -> Update Status: 'failed'"| DS1
```

---

### 2.5 DFD Level 2 — Proses 1.0: Autentikasi, RBAC & OWASP Security Pipeline

Alur verifikasi keamanan berlapis untuk setiap request HTTP yang masuk ke server.

```mermaid
flowchart TD
    Req["Request HTTP Masuk"]
    DS1[("D1: App Database")]
    Handlers["Target Route Handlers"]

    P1_1["1.1 Injeksi OWASP Security Headers (nosniff, SAMEORIGIN, CSP)"]
    P1_2["1.2 Max Body Size Limiter (Max 2MB)"]
    P1_3["1.3 IP Sliding-Window Rate Limiter (Max 10 req/menit)"]
    P1_4["1.4 CSRF Origin / Referer Matcher (Untuk POST/PUT/DELETE)"]
    P1_5["1.5 Session Cookie Validator (sipen_session)"]
    P1_6["1.6 Role-Based Access Control (RBAC) Guard"]

    Req --> P1_1
    P1_1 --> P1_2
    P1_2 -->|"Ukuran > 2MB"| Err413["HTTP 413 Payload Too Large"]
    P1_2 -->|"Ukuran Valid"| P1_3
    P1_3 -->|"Frekuensi Melebihi Batas"| Err429["HTTP 429 Too Many Requests"]
    P1_3 -->|"Frekuensi Aman"| P1_4
    P1_4 -->|"Origin / Referer Berbeda Domain"| Err403["HTTP 403 Forbidden (CSRF Attack)"]
    P1_4 -->|"Origin Sah / Loopback"| P1_5

    P1_5 <-->|"Cek Token di Tabel sessions"| DS1
    P1_5 -->|"Token Tidak Sah / Expired"| RedirectLogin["Redirect ke /login"]
    P1_5 -->|"Token Valid (User Ditemukan)"| P1_6

    P1_6 -->|"Akses Route Biasa (Admin/SuperAdmin)"| Handlers
    P1_6 -->|"Akses Endpoint Sensitif (/backup, /registration)"| CheckSuper{"Role == superadmin?"}
    CheckSuper -->|"Ya"| Handlers
    CheckSuper -->|"Bukan (Admin Biasa)"| ErrForbidden["HTTP 403 Forbidden (BFLA Blocked)"]
```

---

## 3. Diagram Mesin Status (Flow State Diagrams)

### 3.1 WhatsApp Connection State Machine

Status koneksi WhatsApp client dikelola oleh event-driven state machine:

```mermaid
stateDiagram-v2
    [*] --> Disconnected: Server Boot / Inisialisasi
    
    Disconnected --> Connecting: Memulai Koneksi (c.Start)
    
    Connecting --> NeedQR: Sesi Baru (Belum ada kredensial di Store.ID)
    Connecting --> Connected: Sesi Lama Ditemukan & Kunci Valid
    Connecting --> Disconnected: Gagal Terhubung ke WebSocket WA
    
    state NeedQR {
        [*] --> MenungguPilihan
        MenungguPilihan --> ModeQRCode: Pengguna Membuka Tab Scan QR
        MenungguPilihan --> ModePairingCode: Pengguna Meminta Pairing No HP
        
        ModeQRCode --> QRDisplay: Generate DataURL QR PNG
        QRDisplay --> QRTimeout: Waktu Habis (>60s)
        QRTimeout --> ModeQRCode: Auto-Reconnect / Tombol Minta QR Baru
        
        ModePairingCode --> PairingDisplay: whatsmeow.PairPhone (Chrome Linux)
        PairingDisplay --> PairingTimeout: Kode Kedaluwarsa (~160s)
        PairingTimeout --> ModePairingCode: Permintaan Kode Baru
    }
    
    NeedQR --> Connected: Otorisasi Berhasil dari Aplikasi WhatsApp Ponsel
    
    Connected --> Connecting: Koneksi Terputus Sementara (Auto-Reconnect Loop)
    Connected --> LoggedOut: Pengguna Logout dari Ponsel / Sesi Dicabut
    Connected --> Disconnected: Pengguna Klik 'Putuskan Sesi' di Web UI
    
    LoggedOut --> Disconnected: Bersihkan Session Database
```

---

### 3.2 Message Queue Lifecycle State Machine

Setiap pesan pengingat dosen yang dijadwalkan melewati siklus hidup berikut:

```mermaid
stateDiagram-v2
    [*] --> Pending: Scheduler Memicu Pengiriman / Trigger Manual
    
    Pending --> Processing: Queue Worker Mengambil Job Antrian Teratas
    Pending --> Cancelled: Dibatalkan Manual oleh Pengguna via Web UI
    
    state Processing {
        [*] --> SetPresenceComposing: Broadcast 'Sedang Mengetik...' ke WA
        SetPresenceComposing --> WaitJitter: Jeda Acak Manusia (1.5s - 2.5s)
        WaitJitter --> DispatchMessage: whatsmeow.SendMessage
    }
    
    Processing --> Sent: Pengiriman Berhasil Diterima Server WA
    Processing --> RetryPending: Jaringan Error / WA Disconnected (Retries < 3)
    
    RetryPending --> Processing: Exponential Backoff Retry (5s, 10s, 20s)
    RetryPending --> Failed: Batas Retries Tercapai (3 Kali Gagal)
    
    Sent --> [*]
    Failed --> [*]
    Cancelled --> [*]
```

---

### 3.3 Schedule Job Lifecycle State Machine

Status jadwal mata kuliah yang dievaluasi berkala oleh scheduler:

```mermaid
stateDiagram-v2
    [*] --> Draft: Jadwal Baru Dibuat
    Draft --> Active: Diaktifkan (is_active = true)
    
    Active --> Paused: Dinonaktifkan Sementara (is_active = false)
    Paused --> Active: Diaktifkan Kembali
    
    state Active {
        [*] --> IdleWaiting: Menunggu Waktu Kirim (H-1 Jam Kuliah)
        IdleWaiting --> HolidayCheck: Waktu Kirim Tiba
        
        HolidayCheck --> SuppressedHoliday: Hari Termasuk Libur Nasional / Cuti
        SuppressedHoliday --> IdleWaiting: Jadwal Dilewati, Tunggu Pekan Depan
        
        HolidayCheck --> WindowCheck: Bukan Hari Libur
        WindowCheck --> SuppressedOffHours: Di Luar Jam Kerja (Contoh: >16:00)
        SuppressedOffHours --> IdleWaiting: Ditunda Hingga Jam Buka Jendela
        
        WindowCheck --> ReadyToEnqueue: Dalam Jam Kerja Sah
        ReadyToEnqueue --> Enqueued: Pesan Masuk ke Antrian (Tabel queue)
        Enqueued --> IdleWaiting: Catat Trigger Hari Ini, Tunggu Siklus Berikutnya
    }
```

---

### 3.4 Android Daemon & Wakelock Service Lifecycle

Siklus hidup aplikasi Android APK saat beroperasi sebagai server mandiri di smartphone:

```mermaid
stateDiagram-v2
    [*] --> AppLaunch: Pengguna Membuka Aplikasi SiPenDosa di Android
    
    AppLaunch --> ServiceStarting: MainActivity Memanggil startForegroundService
    
    state SiPenDosaServerService {
        [*] --> AcquireWakelock: PowerManager.PARTIAL_WAKE_LOCK
        AcquireWakelock --> StartForegroundNotification: Tampilkan Notifikasi Sticky (Port 8473)
        StartForegroundNotification --> SpawnNativeDaemon: ProcessBuilder eksekusi libsipen.so / sipen
        
        state SpawnNativeDaemon {
            [*] --> LoadEnvironment: Set PORT=8473, DB_PATH, WA_SESSION_PATH
            LoadEnvironment --> StartGoRuntime: SQLite Engine & Chi HTTP Server Aktif
            StartGoRuntime --> PipeLogsToBuffer: Tangkap stdout/stderr ke Ring Buffer
        }
    }
    
    ServiceStarting --> HealthCheckPolling: Thread UI Polling http://127.0.0.1:8473/healthz
    HealthCheckPolling --> ServerOnline: HTTP 200 OK Diterima
    
    state ServerOnline {
        [*] --> LoadWebView: webView.loadUrl('http://127.0.0.1:8473')
        LoadWebView --> ReadyState: Antarmuka Terminal / Dashboard Tampil
    }
    
    ReadyState --> AppBackgrounded: Pengguna Menekan Tombol Home / Layar Terkunci
    AppBackgrounded --> ReadyState: Server Tetap 100% Aktif Karena Partial Wakelock
    
    ReadyState --> Terminated: Pengguna Memaksa Berhenti (Force Stop)
    Terminated --> ReleaseWakelock: wakeLock.release()
    ReleaseWakelock --> [*]
```

---

## 4. User Flow & Alur Pengguna Interaktif

### 4.1 Alur Onboarding & Registrasi SuperAdmin Pertama

```mermaid
flowchart TD
    Start([Pengguna Membuka http://localhost:8473]) --> CheckDB{Apakah Sudah Ada User di Database?}
    
    CheckDB -->|Belum Ada Sama Sekali| RegisterFirst[Tampilkan Form 'Inisialisasi SuperAdmin']
    RegisterFirst --> InputCreds[Input Username & Password SuperAdmin]
    InputCreds --> SubmitFirst[Submit Pendaftaran Pertama]
    SubmitFirst --> CreateSuper[Simpan Hash Bcrypt & Beri Role 'superadmin']
    CreateSuper --> AutoCloseReg[Otomatis Kunci Pendaftaran Baru: registration_open = false]
    AutoCloseReg --> CreateSess[Generate Session Token 32-Byte & Set Cookie HttpOnly]
    CreateSess --> OverviewRedirect[Redirect ke Dashboard Utama /]

    CheckDB -->|Sudah Ada Pengguna| CheckCookie{Apakah Cookie Sesi Tersedia & Valid?}
    CheckCookie -->|Valid| OverviewRedirect
    CheckCookie -->|Tidak Ada / Kadaluarsa| ShowLogin[Tampilkan Halaman Login]
    
    ShowLogin --> SubmitLogin[Input Username & Password]
    SubmitLogin --> RateCheck{Lolos Rate Limiting 10 req/min?}
    RateCheck -->|Melebihi| Block429[Tampilkan Error HTTP 429 Terlalu Banyak Percobaan]
    RateCheck -->|Lolos| VerifyBcrypt{Kecocokan Hash Password?}
    VerifyBcrypt -->|Salah| ShowErrLogin[Peringatan: Username atau Password Salah]
    VerifyBcrypt -->|Benar| CreateSess
```

---

### 4.2 Alur Taut Perangkat WhatsApp (QR vs Pairing Code Nomor HP)

```mermaid
flowchart TD
    Start([Pengguna Ingin Menautkan WhatsApp]) --> OpenModal[Klik Tombol 'Tautkan WA' di Dashboard]
    OpenModal --> DisplayModal[Modal Muncul dengan Tab 'Scan QR' & 'Tautkan Nomor HP']
    
    DisplayModal --> ChooseMethod{Pilih Metode Tautan}
    
    %% Cabang QR Code
    ChooseMethod -->|Metode 1: Kamera Laptop / HP Lain| TabQR[Pilih Tab 'Scan QR Code']
    TabQR --> FetchQR[Frontend Otomatis Tarik /api/wa/qr via AJAX]
    FetchQR --> QRValid{Apakah QR Tersedia & Belum Timeout?}
    QRValid -->|Tersedia| RenderQR[Tampilkan Gambar QR Code]
    QRValid -->|Kosong / Expired| AutoRecon[Backend Otomatis Picu Sesi Baru] --> RenderQR
    RenderQR --> ScanCamera[Pengguna Buka WA di Ponsel > Perangkat Tertaut > Scan QR]
    ScanCamera --> WAAccepted[Handshake Sukses]

    %% Cabang Pairing Code (Nomor HP)
    ChooseMethod -->|Metode 2: Ponsel Ini / Termux / Tanpa Kamera| TabPhone[Pilih Tab 'Tautkan Nomor HP']
    TabPhone --> InputPhone[Ketik Nomor WhatsApp Pengirim: Misal 08123456789]
    InputPhone --> ClickPair[Klik 'Dapatkan Kode Pairing']
    ClickPair --> CallAPI[POST /api/wa/pair-phone]
    CallAPI --> CleanNum[Normalisasi Awalan '08' -> '628']
    CleanNum --> GenPairCode[WhatsApp Cloud Kembalikan 8 Karakter: ABCD-1234]
    GenPairCode --> ShowCodeCard[Tampilkan Kode dalam Box Monospace Emas]
    ShowCodeCard --> CopyBtn[Klik 'Salin Kode']
    CopyBtn --> OpenWAPhone[Buka WhatsApp di Ponsel > Perangkat Tertaut > Tautkan Perangkat]
    OpenWAPhone --> TapPhoneLink[Ketuk 'Tautkan dengan nomor telepon saja' di Bawah Kamera]
    TapPhoneLink --> PasteCode[Ketikkan 8 Karakter Kode]
    PasteCode --> WAAccepted

    %% Final Connected State
    WAAccepted --> BroadcastSuccess[WebSocket Hub Broadcast Event 'wa_status: connected']
    BroadcastSuccess --> UpdateUI[Badge Hijau Menyala: 'WhatsApp Terhubung' + Nomor HP Tampil]
    UpdateUI --> CloseModal[Modal Menutup Otomatis & Sistem Siap Digunakan]
```

---

### 4.3 Alur Pengelolaan Jadwal Kuliah & Template Dinamis

```mermaid
flowchart TD
    Start([Pengguna Mengakses Menu Jadwal /schedules]) --> ViewList[Tinjau Daftar Jadwal Aktif & Filter Hari]
    
    ViewList --> ActionChoice{Aksi Pengguna}
    
    ActionChoice -->|Tambah Jadwal Baru| OpenForm[Klik 'Tambah Jadwal Kuliah']
    OpenForm --> FillData[Isi: Nama Matkul, Kode Kelas, Ruang, Hari, Jam Mulai, Dosen Pengampu]
    FillData --> SelectTmpl[Pilih Template Pesan Pengingat]
    SelectTmpl --> SetHMin[Tentukan Waktu Kirim: H-1 Jam Kerja atau Hari-H]
    SetHMin --> SaveSchedule[Simpan ke SQLite Database]
    SaveSchedule --> ToastSuccess[Notifikasi Toast: 'Jadwal Kuliah Berhasil Disimpan']

    ActionChoice -->|Uji Coba Pesan Instan| TestTrigger[Klik Tombol 'Trigger Sekarang']
    TestTrigger --> QueueImmediate[Enqueue Pesan ke Antrian dengan Prioritas Tinggi]
    QueueImmediate --> DispatchNow[Eksekusi via WhatsApp Engine Seketika]

    ActionChoice -->|Kustomisasi Template| OpenTmplMenu[Buka Menu /templates]
    OpenTmplMenu --> EditTemplate[Pilih Template & Ubah Kalimat Salam / Bahasa]
    EditTemplate --> InsertVariables[Sisipkan Tag: {dosen}, {matkul}, {jam}, {ruang}, {hari}]
    InsertVariables --> LivePreview[Gunakan Live Preview Interaktif untuk Uji Tampilan Pesan]
    LivePreview --> SaveTmpl[Simpan Template]
```

---

### 4.4 Alur Eksekusi Headless via Android Termux CLI

```mermaid
flowchart TD
    Start([Buka Aplikasi Termux di Smartphone Android]) --> InstallCheck{SiPenDosa Sudah Terpasang?}
    
    InstallCheck -->|Belum| RunInstaller[Jalankan: curl -fsSL .../install-android-termux.sh | bash]
    RunInstaller --> SetupFiles[Installer Memasang Binary, Config .env & Shortcut PATH]
    SetupFiles --> ReadyCLI[Shortcut Siap: sipen, sipen-start, sipen-pair, sipen-stop]
    
    InstallCheck -->|Sudah| StartChoice{Mode Menjalankan Server}
    
    StartChoice -->|Mode Latar Belakang (24/7)| RunBg[Ketik: sipen-bg]
    RunBg --> AcquireLock[Aktifkan Android Wakelock via termux-wake-lock]
    AcquireLock --> RunNohup[nohup sipen > sipen.log 2>&1 &]
    RunNohup --> ServerRunning[Server Aktif di Latar Belakang (Port 8473)]

    StartChoice -->|Mode Interaktif| RunFg[Ketik: sipen-start]
    RunFg --> ServerRunning

    ServerRunning --> NeedPairing{Apakah Perlu Login WhatsApp?}
    NeedPairing -->|Ya (Tanpa Buka Browser)| RunPairCmd[Ketik: sipen pair 08123456789]
    RunPairCmd --> LoopbackReq[Kirim Permintaan ke http://127.0.0.1:8473/api/internal/pair-phone]
    LoopbackReq --> OutputTerm[Konsol Termux Menampilkan Banner Kuning & Kode: ABCD-1234]
    OutputTerm --> EnterInWA[Buka WhatsApp Ponsel > Perangkat Tertaut > Tautkan No Telepon]
    EnterInWA --> ConnectedTerm[WhatsApp Terhubung Secara Headless]

    NeedPairing -->|Tidak (Sesi Sudah Tersimpan)| CheckStatus[Ketik: curl http://localhost:8473/healthz]
```

---

## 5. Diagram Urutan Interaksi (Sequence Diagrams)

### 5.1 Sequence: Login via Phone Pairing Code

Diagram urutan langkah sinkronisasi kode pairing antara Browser/Klien, Server SiPenDosa, Engine WhatsMeow, WebSocket Hub, dan Cloud Server WhatsApp.

```mermaid
sequenceDiagram
    autonumber
    actor User as Pengguna / Mahasiswa
    participant UI as Web Browser / Mobile UI
    participant Handler as Web Handler (/api/wa/pair-phone)
    participant WAClient as WhatsApp Client (whatsmeow)
    participant WAGateway as WhatsApp Cloud Gateway (Noise Protocol)
    participant Hub as Realtime WebSocket Hub

    User->>UI: Buka Tab 'Tautkan Nomor HP' & Ketik Nomor (08123456789)
    User->>UI: Klik 'Dapatkan Kode Pairing (8 Digit)'
    UI->>Handler: POST /api/wa/pair-phone { phone: "08123456789" }
    
    Handler->>Handler: Normalisasi Nomor: '08' -> '628123456789'
    Handler->>WAClient: PairPhone(ctx, "628123456789")
    
    WAClient->>WAGateway: Noise Encrypted Frame: Request Pairing Code for 628123456789 (Client: Chrome Linux)
    WAGateway-->>WAClient: Pairing Code: "A1B2C3D4"
    
    WAClient->>Hub: Broadcast Event ("wa_pairing_code", { code: "A1B2C3D4" })
    WAClient-->>Handler: Return "A1B2C3D4"
    Handler-->>UI: HTTP 200 JSON { success: true, code: "A1B2C3D4" }
    
    UI->>UI: Render Box Emas: "A1B2 - C3D4" + Tampilkan Panduan WA
    User->>User: Buka WhatsApp di HP > Perangkat Tertaut > Tautkan via No Telepon
    User->>User: Masukkan Kode "A1B2C3D4"
    
    WAGateway->>WAClient: Otorisasi Berhasil! Kirim Identity Keys & Push Name
    WAClient->>WAClient: Simpan Sesi Kredensial ke session/whatsapp.db
    WAClient->>Hub: Broadcast Event ("wa_status", { state: "connected", phone: "628123456789" })
    Hub-->>UI: Push WebSocket Event: Status Berubah ke Connected
    UI->>UI: Update Tampilan: Badge Hijau "WhatsApp Terhubung" & Modal Tutup
```

---

### 5.2 Sequence: Smart Scheduler -> Evaluasi Hari Libur -> Anti-Ban Dispatch

Menunjukkan bagaimana scheduler mengevaluasi hari libur nasional sebelum merilis pesan ke antrian dan menyimulasikan jeda pengetikan manusia (*typing presence*).

```mermaid
sequenceDiagram
    autonumber
    participant Ticker as Chrono Scheduler (1 Menit Loop)
    participant DB as SQLite App Database (sipen.db)
    participant TmplEngine as Template Interpolator
    participant Queue as Queue Manager
    participant WA as WhatsMeow Engine
    participant WhatsAppCloud as WhatsApp Server Cloud

    Ticker->>DB: Query Jadwal Aktif pada Jam & Hari Ini (Timezone Aware)
    DB-->>Ticker: Daftar Jadwal Terpilih (Contoh: Kuliah Besok 08:00 WITA)
    
    Ticker->>DB: Cek Kalender Hari Libur (Tabel holidays) untuk Tanggal Terkait
    alt Tanggal Masuk Hari Libur Nasional / Cuti
        DB-->>Ticker: Data Libur Ditemukan (Misal: 'Idul Fitri' / 'Tahun Baru')
        Ticker->>Ticker: Suppress Jadwal (Batalkan Pengiriman Sesuai Kebijakan)
        Ticker->>DB: Catat Log Audit: 'Jadwal Ditunda Karena Hari Libur'
    else Bukan Hari Libur
        DB-->>Ticker: Hari Kuliah Aktif Sah
        Ticker->>DB: Ambil Template Pesan & Data Dosen Pengampu
        DB-->>Ticker: Template & Info Kontak Dosen
        
        Ticker->>TmplEngine: Interpolasi: Render({dosen}, {matkul}, {jam}, {ruang})
        TmplEngine-->>Ticker: Teks Pesan Santun Siap Kirim
        
        Ticker->>Queue: Enqueue(Recipient, TeksPesan, Priority)
        Queue->>DB: Insert ke Tabel queue (Status: 'pending')
        
        Queue->>Queue: Worker Mengambil Job Antrian Teratas
        Queue->>DB: Update Status queue: 'processing'
        
        Note over Queue,WA: Simulasi Kehadiran Manusia (Anti-Ban Shield)
        Queue->>WA: SendChatPresence(RecipientJID, ChatPresenceComposing)
        WA->>WhatsAppCloud: Status "Sedang Mengetik..." Terlihat di HP Dosen
        
        Queue->>Queue: Jeda Jitter Acak (Tidur 1500ms - 2500ms)
        
        Queue->>WA: SendMessage(RecipientJID, MessageText)
        WA->>WhatsAppCloud: Enkripsi E2EE & Kirim Pesan Teks
        WhatsAppCloud-->>WA: Message ACK (Terkirim ke Server)
        
        WA->>Queue: Status Sukses Terkirim
        Queue->>DB: Update Tabel queue (Status: 'sent', sent_at: NOW)
        Queue->>DB: Insert Activity Log ('Pesan Pengingat Berhasil Terkirim ke Dosen')
    end
```

---

### 5.3 Sequence: OWASP Security Interceptor Pipeline

Menjelaskan lapisan pencegahan terhadap serangan BFLA, CSRF, Brute Force, dan DoS pada setiap request API.

```mermaid
sequenceDiagram
    autonumber
    actor Attacker as Klien / Penyerang Potensial
    participant SecHeaders as Security Headers Middleware
    participant MaxBody as Body Size Limiter (2MB)
    participant RateLimiter as Sliding Window Rate Limiter
    participant CSRF as CSRF Origin Matcher
    participant AuthGuard as Auth & RBAC Middleware
    participant DB as SQLite Database
    participant CoreHandler as Handler Inti (/settings/backup/download)

    Attacker->>SecHeaders: POST /settings/backup/download
    SecHeaders->>SecHeaders: Injeksi Header: X-Content-Type-Options, SAMEORIGIN, CSP
    
    SecHeaders->>MaxBody: Teruskan Request
    alt Ukuran Payload Body > 2MB
        MaxBody-->>Attacker: HTTP 413 Payload Too Large (Mencegah Memory DoS)
    else Ukuran Payload Aman
        MaxBody->>RateLimiter: Teruskan Request
    end

    alt Frekuensi Request IP > 10 Kali Per Menit
        RateLimiter-->>Attacker: HTTP 429 Too Many Requests (Anti Brute-Force)
    else Frekuensi Wajar
        RateLimiter->>CSRF: Teruskan Request
    end

    alt Header Origin Mismatch / Cross-Origin Domain
        CSRF-->>Attacker: HTTP 403 Forbidden: CSRF Origin Validation Failed
    else Origin Sah (Domain Sendiri / Loopback)
        CSRF->>AuthGuard: Teruskan Request
    end

    AuthGuard->>DB: Validasi Session Token dari Cookie 'sipen_session'
    alt Token Tidak Ditemukan / Kadaluarsa
        DB-->>AuthGuard: Session Invalid
        AuthGuard-->>Attacker: Redirect HTTP 303 ke /login
    else Sesi Sah Ditemukan (User: Role 'admin')
        DB-->>AuthGuard: User Record (Role: 'admin')
        AuthGuard->>AuthGuard: Evaluasi RBAC: Endpoint Sensitif Memerlukan 'superadmin'
        alt Role Bukan 'superadmin' (Percobaan Privilege Escalation / BFLA)
            AuthGuard-->>Attacker: HTTP 403 Forbidden: 'Hanya SuperAdmin yang Diizinkan'
        else Role Adalah 'superadmin'
            AuthGuard->>CoreHandler: Izinkan Eksekusi Handler
            CoreHandler->>DB: Baca Berkas sipen.db
            CoreHandler-->>Attacker: HTTP 200 Streaming File Backup
        end
    end
```

---

## 6. Diagram Relasi Entitas (Entity-Relationship Diagram / ERD)

Struktur tabel database SQLite murni (`sipen.db`) yang telah dinormalisasi dengan mode transaksi WAL (*Write-Ahead Logging*):

```mermaid
erDiagram
    USERS ||--o{ SESSIONS : "memiliki"
    USERS ||--o{ ACTIVITY_LOGS : "mencatat"
    CONTACTS ||--o{ SCHEDULES : "dijadwalkan_untuk"
    TEMPLATES ||--o{ SCHEDULES : "digunakan_oleh"
    SCHEDULES ||--o{ QUEUE : "menghasilkan"
    CONTACTS ||--o{ QUEUE : "menerima"

    USERS {
        INTEGER id PK "Auto Increment"
        TEXT username UK "Username Pengguna Unik"
        TEXT password_hash "Bcrypt DefaultCost Hash"
        TEXT role "Role: superadmin | admin"
        DATETIME created_at "Waktu Pendaftaran Akun"
    }

    SESSIONS {
        TEXT token PK "Random Hex 32-Byte"
        INTEGER user_id FK "Relasi ke users.id"
        DATETIME expires_at "Masa Aktif Sesi (7 Hari)"
        DATETIME created_at "Waktu Login"
    }

    CONTACTS {
        INTEGER id PK "Auto Increment"
        TEXT name "Nama Dosen Lengkap"
        TEXT title "Gelar Akademik (Dr., M.Kom, dll)"
        TEXT phone "Nomor WhatsApp Terformat JID"
        TEXT email "Email Dosen (Opsional)"
        TEXT notes "Catatan Dosen / Ruang Kantor"
        DATETIME created_at "Waktu Ditambahkan"
    }

    SCHEDULES {
        INTEGER id PK "Auto Increment"
        INTEGER contact_id FK "Relasi ke contacts.id"
        INTEGER template_id FK "Relasi ke templates.id"
        TEXT course_name "Nama Mata Kuliah"
        TEXT class_name "Kelas / Ruangan"
        INTEGER day_of_week "0=Minggu s/d 6=Sabtu"
        TEXT start_time "Waktu Mulai Kuliah (HH:MM)"
        TEXT end_time "Waktu Selesai Kuliah (HH:MM)"
        TEXT send_time "Waktu Kirim Pengingat (HH:MM)"
        INTEGER is_active "1=Aktif, 0=Nonaktif"
        INTEGER remind_h_minus_1 "1=Kirim H-1, 0=Hari H"
        DATETIME last_sent_at "Waktu Terakhir Berhasil Terkirim"
    }

    TEMPLATES {
        INTEGER id PK "Auto Increment"
        TEXT name "Nama Template Pengingat"
        TEXT content "Format Teks dengan Variabel Dinamis"
        INTEGER is_default "1=Template Baku Default"
        DATETIME updated_at "Waktu Terakhir Diedit"
    }

    QUEUE {
        INTEGER id PK "Auto Increment"
        INTEGER schedule_id FK "Relasi ke schedules.id"
        INTEGER contact_id FK "Relasi ke contacts.id"
        TEXT recipient_phone "Nomor WA Target Pengiriman"
        TEXT recipient_name "Nama Penerima"
        TEXT message "Teks Pesan Final yang Dikirim"
        TEXT status "Status: pending | processing | sent | failed | cancelled"
        INTEGER retries "Jumlah Percobaan Gagal"
        DATETIME scheduled_for "Target Waktu Kirim"
        DATETIME sent_at "Waktu Realisasi Pengiriman"
    }

    HOLIDAYS {
        INTEGER id PK "Auto Increment"
        TEXT holiday_date UK "Format YYYY-MM-DD"
        TEXT name "Nama Hari Libur Nasional / Cuti"
        TEXT description "Deskripsi Keterangan Libur"
    }

    SETTINGS {
        TEXT key PK "Kunci Pengaturan Unik"
        TEXT value "Nilai Pengaturan String"
        DATETIME updated_at "Waktu Diubah"
    }

    ACTIVITY_LOGS {
        INTEGER id PK "Auto Increment"
        TEXT category "Kategori: auth, whatsapp, schedule, queue"
        TEXT action "Aksi yang Dilakukan"
        TEXT details "Rincian Parameter & Output"
        DATETIME created_at "Waktu Pencatatan Log"
    }
```

---

## 7. Rangkuman Spesifikasi Teknis & Kepatuhan Keamanan

| Komponen | Spesifikasi & Standar yang Diimplementasikan |
| :--- | :--- |
| **Bahasa & Compiler** | Go 1.27 (Zero-CGO Pure Go Monolith), C++17 Socket Bridge, Swift 6.3 AppKit |
| **Engine WhatsApp** | `go.mau.fi/whatsmeow` Multi-Device E2EE (Noise Protocol) |
| **Protokol Login WA** | Dual-Method: (1) Visual QR Code PNG + (2) 8-Digit Pairing Code (`whatsmeow.PairPhone`) |
| **Database Storage** | SQLite 3 dengan mode `WAL (Write-Ahead Logging)` & Foreign Keys Enforced |
| **Arsitektur Android** | Standalone Universal APK (API 21+) + AAB Google Play + Termux Headless CLI |
| **Keamanan OWASP** | BFLA Protection, CSRF Origin Verification, Sliding-Window Rate Limiter, Security Headers, Memory DoS Cap (2MB) |
| **Realtime Transport** | Gorilla WebSocket Hub dengan event multiplexing (`wa_status`, `wa_qr`, `wa_pairing_code`, `countdown`, `toast`, `queue_update`) |

---
*Dokumentasi ini resmi disusun untuk SiPenDosa Unified Edition versi 1.1.0.*
