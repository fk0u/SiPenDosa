package scheduler_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"sipen/internal/auth"
	"sipen/internal/queue"
	"sipen/internal/realtime"
	"sipen/internal/scheduler"
	"sipen/internal/store"
	"sipen/internal/template"
	"sipen/internal/whatsapp"
)

func setupTestScheduler(t *testing.T) (*store.Store, *scheduler.Scheduler, *store.User) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_scheduler.db")
	sessionDir := filepath.Join(tempDir, "sessions")

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test store: %v", err)
	}

	authSvc := auth.NewService(s)
	user, err := authSvc.RegisterFirstUser("admin", "password123")
	if err != nil {
		t.Fatalf("Failed to register admin: %v", err)
	}

	hub := realtime.NewHub()
	waMgr := whatsapp.NewManager(sessionDir, hub, "8473")
	qMgr := queue.NewManager(s, waMgr, hub)
	tmplEngine := template.NewEngine()
	sch := scheduler.NewScheduler(s, qMgr, tmplEngine, hub)

	return s, sch, user
}

func TestManualTriggerUsesTodayForSameDayLecture(t *testing.T) {
	s, sch, user := setupTestScheduler(t)
	defer s.Close()

	loc := time.FixedZone("WITA", 8*3600)
	now := time.Now().In(loc)
	todayDayOfWeek := int(now.Weekday())

	// Create contact
	contact, err := s.CreateContact(&store.Contact{
		UserID:      user.ID,
		Name:        "Dr. Test Lecturer",
		Phone:       "081234567890",
		ContactType: "dosen",
	})
	if err != nil {
		t.Fatalf("Failed to create contact: %v", err)
	}

	// Create a schedule for today
	sc, err := s.CreateSchedule(&store.Schedule{
		UserID:      user.ID,
		Title:       "Test Kuliah Hari Ini",
		Matkul:      "Algoritma Pemrograman",
		DosenID:     sql.NullInt64{Int64: contact.ID, Valid: true},
		DayOfWeek:   todayDayOfWeek,
		StartTime:   "08:00",
		EndTime:     "10:00",
		Location:    "Lab 1",
		Mode:        "H-0",
		SendAtTime:  "07:00",
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}

	qm, err := sch.TriggerScheduleManually(sc.ID, true)
	if err != nil {
		t.Fatalf("TriggerScheduleManually failed: %v", err)
	}

	// Message should contain today's date formatted
	expectedDate := template.FormatIndonesianDate(now)
	if !containsSubstring(qm.Message, expectedDate) {
		t.Errorf("Expected message to contain today's lecture date '%s', but got: %s", expectedDate, qm.Message)
	}
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
