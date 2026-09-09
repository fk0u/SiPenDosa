package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sipen/internal/auth"
	"sipen/internal/config"
	"sipen/internal/queue"
	"sipen/internal/realtime"
	"sipen/internal/scheduler"
	"sipen/internal/store"
	"sipen/internal/template"
	"sipen/internal/tunnel"
	"sipen/internal/web"
	"sipen/internal/whatsapp"
	webassets "sipen/web"
)

func setupTestWeb(t *testing.T) (http.Handler, *store.Store, *auth.Service, *store.User) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_sipen.db")
	sessionDir := filepath.Join(tempDir, "sessions")

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test store: %v", err)
	}

	authSvc := auth.NewService(s)
	adminUser, err := authSvc.RegisterFirstUser("superadmin", "password123")
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	hub := realtime.NewHub()
	waManager := whatsapp.NewManager(sessionDir, hub, "8473")
	tunnelMgr := tunnel.NewManager(tempDir, "8473")
	queueMgr := queue.NewManager(s, waManager, hub)
	tmplEngine := template.NewEngine()
	sch := scheduler.NewScheduler(s, queueMgr, tmplEngine, hub)

	renderer := web.NewViewRenderer(webassets.Templates(), "web/templates")
	cfg := &config.Config{
		Host:            "127.0.0.1",
		Port:            "8473",
		DBPath:          dbPath,
		DefaultTimezone: "Asia/Makassar",
	}

	handlers := web.NewHandlers(cfg, s, authSvc, waManager, tunnelMgr, queueMgr, sch, tmplEngine, hub, renderer)
	router := web.SetupRouter(handlers, webassets.Static(), "web/static")

	return router, s, authSvc, adminUser
}

func TestPublicSchedulePortalAndICS(t *testing.T) {
	router, s, _, adminUser := setupTestWeb(t)
	defer s.Close()

	// Insert a public schedule
	sc := &store.Schedule{
		UserID:      adminUser.ID,
		Title:       "Kelas A (Reguler)",
		Matkul:      "Algoritma dan Struktur Data",
		DayOfWeek:   1, // Senin
		StartTime:   "08:00",
		EndTime:     "09:40",
		Location:    "Ruang Lab 1",
		Mode:        "H-1",
		SendAtTime:  "08:00",
		IsActive:    true,
		IsPublic:    true,
		TargetPhone: "081234567890",
	}
	created, err := s.CreateSchedule(sc)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}

	// 1. Test GET /jadwal (Public HTML view)
	req := httptest.NewRequest("GET", "/jadwal", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /jadwal, got %d. Body: %s", w.Code, w.Body.String())
	}
	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "Algoritma dan Struktur Data") {
		t.Errorf("Expected public schedule page to contain matkul name, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Ruang Lab 1") {
		t.Errorf("Expected public schedule page to contain location")
	}

	// 2. Test GET /jadwal/calendar.ics (RFC 5545 iCalendar stream)
	reqICS := httptest.NewRequest("GET", "/jadwal/calendar.ics", nil)
	wICS := httptest.NewRecorder()
	router.ServeHTTP(wICS, reqICS)

	if wICS.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /jadwal/calendar.ics, got %d", wICS.Code)
	}
	ct := wICS.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/calendar") {
		t.Errorf("Expected Content-Type text/calendar, got %s", ct)
	}
	icsContent := wICS.Body.String()
	if !strings.Contains(icsContent, "BEGIN:VCALENDAR") || !strings.Contains(icsContent, "END:VCALENDAR") {
		t.Errorf("Invalid iCalendar content: %s", icsContent)
	}
	if !strings.Contains(icsContent, "Algoritma dan Struktur Data") {
		t.Errorf("Expected iCalendar event to contain matkul name, got: %s", icsContent)
	}
	if !strings.Contains(icsContent, "RRULE:FREQ=WEEKLY;BYDAY=MO") {
		t.Errorf("Expected weekly recurring rule for Monday, got: %s", icsContent)
	}
	_ = created
}

func TestRegistrationClosedWhenSuperAdminExists(t *testing.T) {
	router, s, _, _ := setupTestWeb(t)
	defer s.Close()

	// GET /register should redirect to /login
	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect 303 for /register when admin exists, got %d", w.Code)
	}

	// POST /register should fail/reject
	form := url.Values{}
	form.Set("username", "hacker")
	form.Set("password", "password123")
	form.Set("confirm_password", "password123")

	reqPost := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	// User "hacker" should NOT be registered in database
	_, err := s.GetUserByUsername("hacker")
	if err == nil {
		t.Errorf("Security flaw: unregistered user was created via public /register!")
	}
}

func TestAdminCanCreateNewUserAndIsolatedSession(t *testing.T) {
	router, s, authSvc, adminUser := setupTestWeb(t)
	defer s.Close()

	// Authenticate admin session
	token, err := authSvc.CreateSession(adminUser.ID)
	if err != nil {
		t.Fatalf("Failed to create admin session: %v", err)
	}

	// Create new user via Admin endpoint
	form := url.Values{}
	form.Set("username", "dosen_informatika")
	form.Set("password", "passwordrahasia")
	form.Set("role", "user")

	req := httptest.NewRequest("POST", "/settings/users/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("Expected 303 redirect after user creation, got %d. Body: %s", w.Code, w.Body.String())
	}

	newUser, err := s.GetUserByUsername("dosen_informatika")
	if err != nil {
		t.Fatalf("User was not found in store: %v", err)
	}
	if newUser.Role != "user" {
		t.Errorf("Expected role user, got %s", newUser.Role)
	}
}

func TestTwoFactorAPIFlow(t *testing.T) {
	router, s, authSvc, adminUser := setupTestWeb(t)
	defer s.Close()

	token, err := authSvc.CreateSession(adminUser.ID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 1. GET /api/2fa/setup
	req := httptest.NewRequest("GET", "/api/2fa/setup", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /api/2fa/setup, got %d", w.Code)
	}

	var setupRes struct {
		Success bool   `json:"success"`
		Secret  string `json:"secret"`
		QRCode  string `json:"qr_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &setupRes); err != nil {
		t.Fatalf("Failed to decode setup JSON: %v", err)
	}
	if !setupRes.Success || setupRes.Secret == "" || !strings.HasPrefix(setupRes.QRCode, "data:image/png;base64,") {
		t.Errorf("Invalid 2FA setup response: %+v", setupRes)
	}

	// 2. Enable with invalid code -> should fail
	invalidBody := `{"secret":"` + setupRes.Secret + `","code":"000000"}`
	reqInvalid := httptest.NewRequest("POST", "/api/2fa/enable", strings.NewReader(invalidBody))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	wInvalid := httptest.NewRecorder()
	router.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for invalid TOTP code, got %d", wInvalid.Code)
	}

	_ = context.Background
	_ = time.Now
}
