package store

import (
	"database/sql"
	"time"
)

// User represents a system administrator
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // 'superadmin' or 'admin'
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an authenticated user session
type Session struct {
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Settings represents system-wide operational settings
type Settings struct {
	GlobalDryRun       bool   `json:"global_dry_run"`
	RegistrationOpen   bool   `json:"registration_open"`
	SendWindowStart    string `json:"send_window_start"` // e.g. "08:00"
	SendWindowEnd      string `json:"send_window_end"`   // e.g. "16:00"
	Timezone           string `json:"timezone"`          // e.g. "Asia/Makassar"
	RateLimitMinSec    int    `json:"rate_limit_min_sec"`
	RateLimitMaxSec    int    `json:"rate_limit_max_sec"`
	MaxRetries         int    `json:"max_retries"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Contact represents lecturers, class leaders, students, or WA groups
type Contact struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`        // e.g. "628123456789" or "1203630...@g.us"
	ContactType string    `json:"contact_type"` // 'dosen', 'komti', 'mahasiswa', 'group'
	NIM         string    `json:"nim"`
	Email       string    `json:"email"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Template represents a reusable message template
type Template struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateVersion stores edit history of message templates
type TemplateVersion struct {
	ID         int64     `json:"id"`
	TemplateID int64     `json:"template_id"`
	Content    string    `json:"content"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
}

// Schedule represents a course reminder schedule
type Schedule struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Matkul      string         `json:"matkul"`
	DosenID     sql.NullInt64  `json:"dosen_id"`
	RecipientID sql.NullInt64  `json:"recipient_id"` // Target contact (dosen, komti, or group)
	TargetPhone string         `json:"target_phone"` // Fallback if no contact ID linked
	TemplateID  sql.NullInt64  `json:"template_id"`
	DayOfWeek   int            `json:"day_of_week"`  // 0=Minggu, 1=Senin, ..., 6=Sabtu
	StartTime   string         `json:"start_time"`   // "08:00"
	EndTime     string         `json:"end_time"`     // "09:40"
	Location    string         `json:"location"`     // "Ruang 302 / Lab Komputer"
	LinkGroup   string         `json:"link_group"`   // WA group link or Zoom
	Mode        string         `json:"mode"`         // 'H-1' (remind day before) or 'H-0' (same day)
	SendAtTime  string         `json:"send_at_time"` // "08:00"
	IsActive    bool           `json:"is_active"`
	DryRun      bool           `json:"dry_run"`
	LastSentAt  *time.Time     `json:"last_sent_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ScheduleDetail joins schedule with lecturer, recipient, and template names
type ScheduleDetail struct {
	Schedule
	DosenName     string `json:"dosen_name"`
	RecipientName string `json:"recipient_name"`
	RecipientType string `json:"recipient_type"`
	TemplateName  string `json:"template_name"`
}

// Holiday represents non-working / academic holidays
type Holiday struct {
	ID          int64  `json:"id"`
	Date        string `json:"date"` // YYYY-MM-DD
	Description string `json:"description"`
}

// QueueMessage represents a queued WhatsApp message
type QueueMessage struct {
	ID            int64      `json:"id"`
	ScheduleID    sql.NullInt64 `json:"schedule_id"`
	RecipientJID  string     `json:"recipient_jid"`
	RecipientName string     `json:"recipient_name"`
	Message       string     `json:"message"`
	Status        string     `json:"status"` // 'pending', 'processing', 'sent', 'failed', 'cancelled', 'dry_run'
	RetryCount    int        `json:"retry_count"`
	MaxRetries    int        `json:"max_retries"`
	ErrorMessage  string     `json:"error_message"`
	ScheduledFor  time.Time  `json:"scheduled_for"`
	SentAt        *time.Time `json:"sent_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ActivityLog records operations and security events
type ActivityLog struct {
	ID        int64     `json:"id"`
	Category  string    `json:"category"` // 'auth', 'whatsapp', 'scheduler', 'system'
	Message   string    `json:"message"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

// DashboardStats provides overview metrics
type DashboardStats struct {
	TotalSentToday  int     `json:"total_sent_today"`
	TotalSentAll    int     `json:"total_sent_all"`
	TotalFailedAll  int     `json:"total_failed_all"`
	TotalPending    int     `json:"total_pending"`
	TotalSchedules  int     `json:"total_schedules"`
	TotalContacts   int     `json:"total_contacts"`
	SuccessRate     float64 `json:"success_rate"`
}
