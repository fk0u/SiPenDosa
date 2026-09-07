# SiPenDosa Mobile — Official Flutter Dart Multiplatform Client

Client mobile resmi berdesain **Obsidian Dark & Bento Grid** untuk ekosistem SiPenDosa. Dapat dijalankan di **Android, iOS, macOS, Windows, Linux, dan Web**.

## 🎨 Design System
- **Background Utama**: Obsidian `#08090D`
- **Surface & Cards**: Deep Glass `#0D1017` & `#131823`
- **Aksen Primer**: Crimson Scarlet `#E11D48`
- **Aksen Sekunder**: Burnished Gold `#F59E0B`
- **Status Online**: Emerald Green `#10B981`
- **Grid Layout**: Bento-Grid dengan rasio responsif

## 🚀 Fitur Utama
1. **Live Dashboard Metrics**:
   - Tingkat keberhasilan pesan real-time
   - Total terkirim hari ini & sepanjang masa
   - Antrean aktif dan pesan gagal
2. **WhatsApp Connection Dual-Mode**:
   - Pemindai QR Code
   - Permintaan 8-Digit Kode Pairing WhatsApp via Nomor Telepon (tanpa perlu scanner)
3. **Manajemen Kontak & Jadwal**:
   - Integrasi langsung dengan SQLite core SiPenDosa
4. **Auto Discovery Local Host**:
   - Terhubung secara otomatis ke `http://127.0.0.1:8473` atau alamat IP jaringan lokal Wi-Fi.

## 📱 Cara Menjalankan
```bash
cd mobile/flutter_sipendosa
flutter pub get
flutter run
```

## 📦 Cara Membangun APK / Bundle
```bash
flutter build apk --release
flutter build appbundle --release
```
