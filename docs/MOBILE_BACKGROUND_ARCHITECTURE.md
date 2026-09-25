# 📱 Arsitektur Eksekusi Latar Belakang & Realtime Mobile (Android & iOS)

Dokumen ini menjelaskan rancangan rekayasa teknis sistematis latar belakang (*background execution*) dan pengiriman pesan secara *realtime* untuk platform **Android** dan **Apple iOS** pada proyek open source **SiPenDosa**.

---

## 🎯 Tantangan Eksekusi Latar Belakang pada Perangkat Mobile

Aplikasi otomasi pengingat perkuliahan seperti SiPenDosa menghadapi tantangan ketat pada sistem operasi seluler modern:
1. **Android Doze Mode & OEM Battery Savers** (MIUI, ColorOS, OneUI, EMUI): Mematikan proses latar belakang, memutus socket TCP/WebSocket WhatsApp saat layar mati, dan menunda eksekusi alarm.
2. **iOS App Sandbox & Background Execution Limit**: iOS menangguhkan (*suspend*) proses pengguna normal dalam waktu 30 detik setelah aplikasi diminimalkan atau layar dikunci, serta melarang eksekusi biner sembarangan di luar App Sandbox.

SiPenDosa mengatasi kedua kendala ini melalui rancangan arsitektur native khusus berikut.

---

## 🤖 1. Arsitektur Latar Belakang Android

```
                     ┌─────────────────────────────────────────┐
                     │          Sistem Operasi Android         │
                     │  (Android 5.0 API 21 s.d. Android 15+)  │
                     └────────────────────┬────────────────────┘
                                          │
            ┌─────────────────────────────┼─────────────────────────────┐
            ▼                             ▼                             ▼
┌───────────────────────┐     ┌───────────────────────┐     ┌───────────────────────┐
│ BootCompletedReceiver │     │ Battery Exemption     │     │ SiPenDosaServerService│
│ • ACTION_BOOT_COMPLETED│    │ • ACTION_REQUEST_     │     │ • Foreground Service  │
│ • MY_PACKAGE_REPLACED │     │   IGNORE_BATTERY_     │     │ • CPU WakeLock        │
│ Auto-start saat HP    │     │   OPTIMIZATIONS       │     │ • High-Perf WifiLock  │
│ dinyalakan/reboot     │     │ Bebas dari Doze Mode  │     │ • Auto-Respawn Watch  │
└───────────┬───────────┘     └───────────────────────┘     └───────────┬───────────┘
            │                                                           │
            └─────────────────────────────┬─────────────────────────────┘
                                          ▼
                      ┌───────────────────────────────────────┐
                      │    Child Process: libsipen.so         │
                      │  (Go Core Engine + whatsmeow engine)  │
                      │  • Port: 8473 (HTTP/WS localhost)     │
                      │  • SQLite WAL Session Storage         │
                      │  • Realtime WebSocket Heartbeat       │
                      │  • Smart Scheduler & Queue Worker     │
                      └───────────────────────────────────────┘
```

### Komponen Kunci Android:

#### A. Layanan Latar Belakang Mandiri (`SiPenDosaServerService`)
* **Foreground Service dengan Persistent Notification**: Berjalan dengan channel prioritas dan tipe `dataSync`, memastikan sistem Android tidak mematikan service karena keterbatasan RAM.
* **Dual Lock Protection (WakeLock + WifiLock)**:
  - `PowerManager.PARTIAL_WAKE_LOCK`: Menjaga CPU tetap terjaga saat mengevaluasi jadwal pada jam-jam pengingat perkuliahan.
  - `WifiManager.WIFI_MODE_FULL_HIGH_PERF`: Mencegah modul Wi-Fi memasuki mode hemat daya ekstrim (*power-saving packet drop*), sehingga koneksi WebSocket WhatsApp ke server Meta tetap tersambung secara *real-time*.
* **Auto-Respawn Watchdog Loop**: Jika binary Go (`libsipen.so`) tiba-tiba keluar atau dihentikan akibat kondisi memori rendah ekstrim, loop runner internal akan mendeteksi dan secara otomatis menghidupkan kembali proses server dalam 3 detik.

#### B. Pelindung Baterai & Doze Mode (`MainActivity`)
* Aplikasi memeriksa izin `PowerManager.isIgnoringBatteryOptimizations()`.
* Jika perangkat membatasi konsumsi baterai, pengguna akan diarahkan ke dialog satu-klik untuk memasukkan SiPenDosa ke dalam daftar putih (*whitelist*), membebaskannya dari penutupan agresif OEM.

#### C. Boot & Update Persistence (`BootCompletedReceiver`)
* Menerima siaran `android.intent.action.BOOT_COMPLETED` dan `android.intent.action.MY_PACKAGE_REPLACED`.
* Begitu perangkat selesai dinyalakan ulang (*reboot*), background server langsung hidup tanpa mengharuskan pengguna membuka aplikasi terlebih dahulu.

---

## 🍏 2. Arsitektur Latar Belakang iOS

Pada ekosistem Apple iOS, SiPenDosa menggunakan kombinasi teknologi **BackgroundTasks (`BGTaskScheduler`)**, **Extended Background Task Execution**, **Local Push Notifications (`UNUserNotificationCenter`)**, dan **Realtime WebSocket Telemetry**:

```
                       ┌────────────────────────────────────────┐
                       │             Apple iOS Engine           │
                       │          (iOS 15.0 s.d. iOS 18+)       │
                       └───────────────────┬────────────────────┘
                                           │
            ┌──────────────────────────────┼──────────────────────────────┐
            ▼                              ▼                              ▼
┌────────────────────────┐     ┌────────────────────────┐     ┌────────────────────────┐
│ Extended Background    │     │ BGTaskScheduler        │     │ UNUserNotification     │
│ Execution              │     │ • BGAppRefreshTask     │     │ • Notifikasi Lokal     │
│ • beginBackgroundTask  │     │   (com.sipendosa.ios.  │     │ • Terpicu Realtime     │
│ Memberikan jendela     │     │    refresh)            │     │   saat antrian terkirim│
│ waktu transmisi saat   │     │ • BGProcessingTask     │     │ • Suara & Ringkasan    │
│ aplikasi diminimalkan  │     │   (Periodic 15m Sync)  │     │   Pengingat Dosen      │
└───────────┬────────────┘     └───────────┬────────────┘     └───────────┬────────────┘
            │                              │                              │
            └──────────────────────────────┼──────────────────────────────┘
                                           ▼
                       ┌────────────────────────────────────────┐
                       │        BackgroundSyncManager           │
                       │  • URLSessionWebSocketTask (Realtime)  │
                       │  • Auto-reconnect & Ping Heartbeat 20s │
                       │  • Live Telemetry Header (ms / state)  │
                       │  • Background Queue Dispatcher         │
                       └───────────────────┬────────────────────┘
                                           │ (HTTP / WebSocket)
                                           ▼
                       ┌────────────────────────────────────────┐
                       │         Server Inti SiPenDosa          │
                       │   (Localhost / Laptop LAN / Cloud)     │
                       └────────────────────────────────────────┘
```

### Komponen Kunci iOS:

#### A. Pengatur Tugas Latar Belakang (`BGTaskScheduler`)
* Terdaftar dalam `Info.plist` melalui kunci `UIBackgroundModes` (`fetch`, `processing`, `remote-notification`) dan `BGTaskSchedulerPermittedIdentifiers`:
  - `com.sipendosa.ios.refresh`: Tugas berkala (*earliest begin date* 15 menit) untuk memeriksa status antrian dan jadwal yang jatuh tempo.
  - `com.sipendosa.ios.sync`: Sinkronisasi batch pemrosesan data saat perangkat terhubung ke pengisi daya.

#### B. Eksekusi Diperpanjang Saat Minimized (`Extended Background Task`)
* Menggunakan `UIApplication.shared.beginBackgroundTask(withName:expirationHandler:)`.
* Saat pengguna menutup atau mengunci layar iPhone, aplikasi tidak langsung mati melainkan memperoleh jendela waktu penyelesaian transmisi data penting ke server WhatsApp.

#### C. Live WebSocket Hub & Telemetri Real-Time
* Menggunakan `URLSessionWebSocketTask` dengan heartbeat ping setiap 20 detik.
* Mendengarkan event server:
  - `wa_status`: Status otentikasi WhatsApp.
  - `queue_update`: Pengiriman pesan real-time. Jika pesan berhasil terkirim (`action == "sent"`), sistem secara otomatis memunculkan notifikasi lokal `UNUserNotificationCenter` berbunyi untuk menginformasikan mahasiswa.
* Header telemetri visual di bagian atas aplikasi menampilkan latency (*ping* ms), status engine online, dan jumlah antrian aktif.

---

## 🛡️ Kebijakan Kepatuhan Open Source

Rancangan di atas sepenuhnya mematuhi prinsip open source:
1. **Transparan & Tanpa Pelacakan**: Tidak ada analitik tersembunyi atau pengiriman data ke pihak ketiga.
2. **Hemat Baterai**: Menggunakan interval heartbeat adaptif dan pelepasan wake lock saat aplikasi tidak dalam jendela antrian aktif.
3. **Privasi Penuh**: Seluruh basis data sesi (`whatsapp.db` dan `sipen.db`) tersimpan secara lokal dan terenkripsi pada sandbox masing-masing perangkat.
