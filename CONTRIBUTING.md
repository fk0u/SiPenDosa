# 🤝 Panduan Kontribusi Open Source (Contributing Guidelines)

Terima kasih atas minat Anda untuk berkontribusi pada proyek **SiPenDosa** (*Sistem Pengingat Dosen Saatnya*)! Proyek ini dikembangkan secara terbuka untuk mempermudah kehidupan perkuliahan mahasiswa di seluruh Indonesia dengan etika dan kesopanan digital tingkat tinggi.

---

## 🧭 Prinsip Utama Proyek
1. **Etika & Kesopanan:** Seluruh fitur pengingat harus mengedepankan kesantunan bahasa akademik dan waktu pengiriman yang pantas.
2. **Zero-CGO & Portabilitas:** Inti aplikasi ditulis dalam Go murni (*pure Go*) dan SQLite WAL tanpa dependensi CGO yang memberatkan kompilasi silang.
3. **Privasi & Keamanan:** Seluruh data kontak dosen, jadwal, dan sesi WhatsApp tersimpan secara lokal dan terisolasi di perangkat pengguna.

---

## 🛠️ Prasyarat Lingkungan Pengembangan

Untuk berkontribusi pada repositori ini, siapkan perangkat lunak berikut:
* **Go 1.22+** (Direkomendasikan Go 1.26+)
* **Clang++ (C++17)** — untuk kompilasi C++ socket bridge macOS
* **Swift Compiler (`swiftc`)** — untuk macOS AppKit Menu Bar companion app
* **Make** — untuk menjalankan pipeline Makefile multi-platform

---

## 🚀 Perintah Dasar Pengembangan

### 1. Menjalankan Unit Tests
```bash
make test
# atau
go test -v ./internal/...
```

### 2. Menjalankan Server Lokal (Development)
```bash
make run
```
Akses dashboard di peramban: `http://localhost:8473`

### 3. Kompilasi Multi-Platform
```bash
make build-all         # Kompilasi binary Linux, Windows, dan macOS (Universal)
make package-macos     # Membangun bundle SiPenDosa.app, PKG installer, DMG, dan tarball
make package-deb       # Membangun paket Debian/Ubuntu (.deb) amd64 & arm64
make installer-windows # Membangun installer standalone Windows (.exe)
```

---

## 📐 Alur Pengajuan Kode (Pull Request Workflow)

1. **Fork** repositori ini ke akun GitHub Anda.
2. Buat branch baru dari `main` dengan nama fitur yang deskriptif:
   ```bash
   git checkout -b fitur/pengingat-harian-cerdas
   ```
3. Lakukan perubahan kode dan pastikan seluruh unit test lolos:
   ```bash
   go test ./internal/...
   ```
4. Commit perubahan dengan pesan commit yang jelas dan terstruktur:
   ```bash
   git commit -m "fix(scheduler): cegah jadwal terlewat saat sistem terbangun dari sleep"
   ```
5. Push branch ke fork Anda:
   ```bash
   git push origin fitur/pengingat-harian-cerdas
   ```
6. Buka **Pull Request (PR)** baru dengan menyertakan deskripsi perubahan, alasan perubahan, dan bukti pengujian.

---

## 📜 Lisensi
Dengan berkontribusi pada repositori SiPenDosa, Anda menyetujui bahwa seluruh kontribusi kode Anda dilisensikan di bawah lisensi [MIT License](LICENSE).
