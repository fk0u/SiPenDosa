# Mistake Registry

Catatan kekeliruan atau kendala yang ditemui untuk perbaikan berkelanjutan.

| Tanggal | Masalah | Penyebab | Solusi & Pencegahan |
|---|---|---|---|
| 2026-09-04 | Inisialisasi awal | Lingkungan Windows tanpa GCC default (`CGO_ENABLED=0`) | Gunakan pure-Go SQLite driver `modernc.org/sqlite` untuk app store & whatsmeow sqlstore agar build CGO-free lancar di Windows maupun Linux VPS. |
