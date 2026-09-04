package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store encapsulates the database connection and operations
type Store struct {
	db *sql.DB
}

// New initializes the SQLite database connection, applies WAL mode, and migrates tables
func New(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite connection pool configuration
	db.SetMaxOpenConns(1) // Single writer for SQLite to avoid lock contention
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return s, nil
}

// DB returns the underlying sql.DB instance
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'admin',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			global_dry_run INTEGER NOT NULL DEFAULT 0,
			registration_open INTEGER NOT NULL DEFAULT 0,
			send_window_start TEXT NOT NULL DEFAULT '08:00',
			send_window_end TEXT NOT NULL DEFAULT '16:00',
			timezone TEXT NOT NULL DEFAULT 'Asia/Makassar',
			rate_limit_min_sec INTEGER NOT NULL DEFAULT 5,
			rate_limit_max_sec INTEGER NOT NULL DEFAULT 15,
			max_retries INTEGER NOT NULL DEFAULT 3,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS contacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			phone TEXT NOT NULL,
			contact_type TEXT NOT NULL DEFAULT 'dosen',
			nim TEXT DEFAULT '',
			email TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			content TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS template_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			version INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			matkul TEXT NOT NULL,
			dosen_id INTEGER REFERENCES contacts(id) ON DELETE SET NULL,
			recipient_id INTEGER REFERENCES contacts(id) ON DELETE SET NULL,
			target_phone TEXT DEFAULT '',
			template_id INTEGER REFERENCES templates(id) ON DELETE SET NULL,
			day_of_week INTEGER NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			location TEXT DEFAULT '',
			link_group TEXT DEFAULT '',
			mode TEXT NOT NULL DEFAULT 'H-1',
			send_at_time TEXT NOT NULL DEFAULT '08:00',
			is_active INTEGER NOT NULL DEFAULT 1,
			dry_run INTEGER NOT NULL DEFAULT 0,
			last_sent_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS holidays (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS queue_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			schedule_id INTEGER REFERENCES schedules(id) ON DELETE SET NULL,
			recipient_jid TEXT NOT NULL,
			recipient_name TEXT NOT NULL,
			message TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			error_message TEXT DEFAULT '',
			scheduled_for DATETIME NOT NULL,
			sent_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS activity_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT NOT NULL,
			message TEXT NOT NULL,
			details TEXT DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);`,
		`CREATE INDEX IF NOT EXISTS idx_queue_status_time ON queue_messages(status, scheduled_for);`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_active ON schedules(is_active, day_of_week);`,
		`CREATE INDEX IF NOT EXISTS idx_activity_created ON activity_logs(created_at DESC);`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed executing migration statement: %w\nQuery: %s", err, query)
		}
	}

	// Seed default settings row if not exists
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM settings WHERE id = 1").Scan(&count); err == nil && count == 0 {
		_, _ = s.db.Exec(`INSERT INTO settings 
			(id, global_dry_run, registration_open, send_window_start, send_window_end, timezone, rate_limit_min_sec, rate_limit_max_sec, max_retries, updated_at) 
			VALUES (1, 0, 0, '08:00', '16:00', 'Asia/Makassar', 5, 15, 3, CURRENT_TIMESTAMP)`)
	}

	// Seed default formal template if no templates exist
	if err := s.db.QueryRow("SELECT COUNT(*) FROM templates").Scan(&count); err == nil && count == 0 {
		defaultContent := `*PENGINGAT PERKULIAHAN (H-1)*

Assalamu'alaikum Warahmatullahi Wabarakatuh / Selamat Siang,
Yth. Bapak/Ibu {{.NamaDosen}},

Mohon izin mengingatkan jadwal perkuliahan untuk esok hari:
📚 *Mata Kuliah:* {{.Matkul}}
🗓️ *Hari/Tanggal:* {{.Hari}}, {{.Tanggal}}
⏰ *Waktu:* {{.JamMulai}} - {{.JamSelesai}} WITA
📍 *Ruang/Lokasi:* {{.Lokasi}}
{{if .LinkGroup}}🔗 *Tautan Kelas/Grup:* {{.LinkGroup}}{{end}}

Pemberitahuan ini dikirim otomatis oleh Asisten Perkuliahan (SiPen).
Demikian informasi ini disampaikan. Terima kasih atas perhatian dan kerja sama Bapak/Ibu.

Hormat kami,
{{if .NamaMahasiswa}}Ketua Tingkat: {{.NamaMahasiswa}}{{if .NIM}} ({{.NIM}}){{end}}{{else}}Mahasiswa SiPen Bot{{end}}`

		res, err := s.db.Exec(`INSERT INTO templates (name, content, is_default, created_at, updated_at) VALUES (?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			"Pengingat Perkuliahan Formal (H-1)", defaultContent)
		if err == nil {
			templateID, _ := res.LastInsertId()
			_, _ = s.db.Exec(`INSERT INTO template_versions (template_id, content, version, created_at) VALUES (?, ?, 1, CURRENT_TIMESTAMP)`,
				templateID, defaultContent)
		}
	}

	return nil
}
