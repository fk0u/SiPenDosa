# Session Briefing - SiPenDosa

## Status Snapshot
- **Current Phase:** Standalone Installer Architecture Complete & Production-Ready (Zero-Code Distribution)
- **Goal:** Aplikasi **SiPenDosa** (**Sistem Pengingat Dosen Saatnya**) telah dilengkapi dengan sistem **Standalone Installer** (`SiPenDosa-Setup.exe`), asset embedding (`go:embed`) untuk seluruh template dan static files (100% self-contained), skrip instalasi satu klik (`install.bat` / `install.ps1`), dan skrip Inno Setup (`SiPenDosa.iss`). Pengguna akhir dapat langsung menjalankan aplikasi tanpa perlu menginstal Go, Git, GCC, atau menyalin source code.

## Makna & Filosofi Nama
1. **Kepanjangan Resmi:**
   - **SiPenDosa** = **Sistem Pengingat Dosen Saatnya**
   - Versi bernyawa: **Si Pen Dosa** — Asisten yang rela “berdosa” demi mengingatkan dosen.
2. **Penjelasan Makna:**
   - **Versi Formal (SRS / Dokumentasi):** Sistem pengingat dosen yang bertugas mengirimkan pesan pengingat jadwal kuliah secara otomatis kepada dosen pengampu mata kuliah.
   - **Versi Asli / Karakter:** Lahir dari kegelisahan mahasiswa yang merasa "berdosa" (sungkan/takut dianggap tidak sopan) jika berkali-kali mengingatkan dosen. SiPenDosa hadir menanggung beban "dosa" tersebut dengan pesan sopan dan terjadwal.

## Komponen Distribusi & Installer
1. `web/assets.go`: Menyematkan seluruh HTML templates dan static web assets ke binary via `go:embed`.
2. `cmd/installer/main.go`: Program installer mandiri berbasis Go yang membundel `sipen.exe`, meng-generate `.env` produksi unik, membuat Desktop & Start Menu shortcuts, background launcher (`run-background.vbs`), uninstaller (`uninstall.bat`), dan mendaftarkan program ke Windows Add/Remove Programs.
3. `cmd/installer/manifest.xml`: Manifest Windows dengan privilege `asInvoker` sehingga installer dapat diinstal tanpa memerlukan hak akses Administrator.
4. `install.ps1` & `install.bat`: Skrip instalasi portabel satu klik untuk PowerShell dan Command Prompt.
5. `SiPenDosa.iss`: Skrip Inno Setup untuk kemudahan membangun wizard installer tradisional (Next > Next > Finish).
6. `Makefile`: Mendukung target `make build` dan `make installer`.
