# 🗺️ Roadmap & Rencana Pengembangan Masa Depan (SiPenDosa)

Dokumen ini merangkum rencana pengembangan fitur masa depan untuk **SiPenDosa** (*Sistem Pengingat Dosen Saatnya*). Semua ide dan masukan dapat dipantau langsung pada [GitHub Issues](https://github.com/fk0u/SiPenDosa/issues).

---

## 📌 Fitur Mendatang (Upcoming Features)

### 1. 👥 Dynamic WhatsApp Group & Contact Picker Dropdown
* **Status**: Direncanakan (Roadmap v1.2.0)
* **GitHub Issue**: [#2](https://github.com/fk0u/SiPenDosa/issues/2)
* **Masalah**: Pengguna kesulitan mencari dan menyalin **WhatsApp Group JID** (`... @g.us`) secara manual saat ingin mengirimkan pengingat ke grup kelas/praktikum.
* **Solusi & Rencana**:
  - Integrasi WhatsMeow API: memanfaatkan `client.GetJoinedGroups()` untuk mengambil seluruh grup yang diikuti bot dan `client.Store.Contacts.GetAllContacts()` untuk kontak.
  - Penambahan endpoint backend: `/api/whatsapp/groups` dan `/api/whatsapp/contacts`.
  - UI Searchable Dropdown / Combobox pada form Tambah/Ubah Jadwal dengan pilihan:
    - 👥 **Grup WhatsApp** (dilengkapi nama grup dan jumlah anggota).
    - 👤 **Kontak Tersimpan** (nama dosen/mahasiswa dan nomor telepon).
    - ✍️ **Input Manual / Nomor Baru**.
  - Pengguna cukup memilih nama grup, dan sistem otomatis mengisi target JID grup di balik layar.

---

### 2. 🌐 Built-in Zero-Config Public Tunneling (Akses Global Otomatis)
* **Status**: Direncanakan (Roadmap v1.2.0)
* **GitHub Issue**: [#3](https://github.com/fk0u/SiPenDosa/issues/3)
* **Masalah**: SiPenDosa berjalan di localhost / LAN lokal. Pengguna yang menjalankan SiPenDosa di laptop/server lokal kesulitan mengaksesnya dari jaringan luar (internet publik) tanpa port forwarding atau IP publik statis.
* **Solusi & Rencana**:
  - Integrasi modul tunneling otomatis di backend Go (misalnya *Cloudflare Quick Tunnel* via `cloudflared` / `trycloudflare.com` atau *Bore/Ngrok*).
  - 1-Klik Toggle di menu **Pengaturan**: *"Aktifkan Akses Global (Tunneling)"*.
  - Menghasilkan URL HTTPS publik instan secara otomatis (contoh: `https://sipendosa-xyz123.trycloudflare.com`).
  - Menampilkan **QR Code** di dashboard agar pengguna dapat langsung memindai dan membuka SiPenDosa di smartphone dari jaringan seluler mana pun.

---

### 3. 📅 Public Academic Schedule Board / Portal (`/jadwal/public`)
* **Status**: Direncanakan (Roadmap v1.2.0)
* **GitHub Issue**: [#4](https://github.com/fk0u/SiPenDosa/issues/4)
* **Masalah**: Teman sekelas atau mahasiswa sering membutuhkan informasi jadwal kuliah, lokasi ruangan/gedung, dan dosen pengampu secara transparan tanpa perlu hak akses admin.
* **Solusi & Rencana**:
  - Halaman publik khusus `/jadwal` atau `/public/schedule` yang dapat diakses siapa saja tanpa perlu login.
  - Tampilan papan jadwal responsif, elegan, dan *mobile-friendly*:
    - 📚 Mata Kuliah & Kode Kelas (contoh: *IF3102 - Rekayasa Perangkat Lunak*).
    - 🏛️ Lokasi Perkuliahan (contoh: *Gedung Fasilkom Ruang Lab 3 / R.204*).
    - 👨‍🏫 Dosen Pengampu (contoh: *Prof. Dr. Ir. Budi Santoso, M.Kom*).
    - ⏰ Hari, Jam Mulai - Selesai, & SKS.
    - ⏳ Widget Countdown Real-Time (*"Kuliah berikutnya dimulai dalam 30 menit"*).
  - Filter interaktif per hari, pencarian dosen/matkul, dan tombol download kalender (`.ics`).
  - Opsi admin: Toggle *"Tampilkan di Portal Publik"* pada setiap jadwal perkuliahan.

---

*Tertarik berkontribusi atau memberi masukan? Silakan buka [Issue](https://github.com/fk0u/SiPenDosa/issues) atau diskusikan di repositori resmi.*
