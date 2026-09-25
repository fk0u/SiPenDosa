# 🍎 Panduan Paket macOS (macOS 12+ Universal Edition)

Dokumen ini menjelaskan spesifikasi teknis, arsitektur, dan cara instalasi paket **SiPenDosa** untuk **macOS 12.0 Monterey, macOS 13 Ventura, macOS 14 Sonoma, macOS 15 Sequoia, dan yang lebih baru**.

---

## ⚡ Arsitektur Universal 2 (Apple Silicon & Intel)

Seluruh distribusi SiPenDosa untuk macOS dikompilasi secara **Dual-Architecture (Universal 2 Mach-O Binary)** menggunakan `lipo`, sehingga berjalan secara native pada:
1. **Apple Silicon**: M1, M2, M3, M4 (arsitektur `arm64`).
2. **Intel 64-bit**: Mac berbasis prosesor Intel (arsitektur `x86_64`).

```
                ┌──────────────────────────────────────────────┐
                │             SiPenDosa.app                    │
                │        (macOS 12+ Universal Bundle)          │
                └──────────────────────┬───────────────────────┘
                                       │
            ┌──────────────────────────┴──────────────────────────┐
            ▼                                                     ▼
┌───────────────────────────────┐             ┌───────────────────────────────┐
│ Contents/MacOS/SiPenDosa      │             │ Contents/Resources/sipen      │
│ (Swift AppKit Menu Bar App    │             │ (Go Core Unified Server       │
│  + C++17 Socket Bridge)       │             │  + WhatsApp Engine whatsmeow) │
│ • arm64 (Apple Silicon)       │             │ • arm64 (Apple Silicon)       │
│ • x86_64 (Intel 64-bit)       │             │ • x86_64 (Intel 64-bit)       │
└───────────────────────────────┘             └───────────────────────────────┘
```

---

## 📦 Pilihan Format Distribusi

SiPenDosa menyediakan 3 format kemasan resmi untuk pengguna macOS:

| Format Paket | Nama File | Penggunaan Terbaik |
| :--- | :--- | :--- |
| **Apple Disk Image** | `SiPenDosa-1.2.0.dmg` | Instalasi drag-and-drop klasik dengan visual retina kustom menuju folder `/Applications`. |
| **Apple Installer Package** | `SiPenDosa-1.2.0-Installer.pkg` | Wizard instalasi otomatis standar macOS yang sekaligus mendaftarkan **macOS LaunchAgent** agar server berjalan 24/7 di latar belakang. |
| **Universal Tarball** | `sipendosa_macos_universal.tar.gz` | Pengguna CLI / Developer yang ingin menjalankan daemon mandiri atau integrasi server lokal. |

---

## 🚀 Panduan Pemasangan

### 1. Menggunakan Apple Installer Wizard (`.pkg`) — *Direkomendasikan*
1. Unduh berkas `SiPenDosa-1.2.0-Installer.pkg`.
2. Klik dua kali untuk membuka wizard penginstal.
3. Klik **Lanjutkan (Continue)** hingga selesai.
4. Paket ini secara otomatis:
   - Memasang `SiPenDosa.app` ke `/Applications`.
   - Menginisialisasi direktori data di `~/.local/share/sipendosa`.
   - Mendaftarkan LaunchAgent di `~/Library/LaunchAgents/com.sipendosa.daemon.plist` sehingga server selalu aktif saat Anda login ke Mac.

### 2. Menggunakan Apple Disk Image (`.dmg`)
1. Unduh dan buka `SiPenDosa-1.2.0.dmg`.
2. Tarik (*drag*) ikon **SiPenDosa** ke pintasan **Applications**.
3. Buka **SiPenDosa** dari Launchpad atau Finder.
4. Menu Bar Mac Anda akan menampilkan ikon status **⚡ SiPenDosa**.

### 3. Menggunakan Universal Tarball CLI
```bash
# Ekstrak arsip
tar -xzf sipendosa_macos_universal.tar.gz
cd sipendosa_macos_universal

# Jalankan skrip instalasi interaktif
./install-macos.sh
```

---

## ⚙️ Layanan Latar Belakang (LaunchAgent)

SiPenDosa dapat berjalan sebagai background daemon yang hidup terus-menerus tanpa perlu membuka terminal atau aplikasi jendela:

* **File Konfigurasi Daemon:** `~/Library/LaunchAgents/com.sipendosa.daemon.plist`
* **Direktori Database & Sesi:** `~/.local/share/sipendosa/data` dan `~/.local/share/sipendosa/session`
* **Berkas Log:** `~/.local/share/sipendosa/sipen.log`

### Perintah Manajemen Layanan via Terminal:
```bash
# Memulai layanan LaunchAgent
launchctl load ~/Library/LaunchAgents/com.sipendosa.daemon.plist

# Menghentikan layanan LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.sipendosa.daemon.plist

# Memantau log realtime
tail -f ~/.local/share/sipendosa/sipen.log
```

---

## 🛠️ Kompilasi dari Sumber (Developers)

Jika Anda ingin mengompilasi seluruh paket macOS secara mandiri dari source code:

```bash
# Pastikan Go, clang++, dan swiftc tersedia di sistem
make build-darwin    # Mengompilasi biner Go arm64 + x86_64 dan lipo universal
make app-macos       # Membangun bundle SiPenDosa.app universal macOS 12+
make pkg             # Membangun installer package .pkg
make dmg             # Membangun disk image .dmg
make package-macos   # Membangun seluruh paket di atas sekaligus ke folder dist/
```
