package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"sipen/internal/store"
)

// Regresi: endpoint mobile API & WebSocket tidak boleh bisa diakses tanpa sesi.
func TestMobileAPIAndWSRequireAuth(t *testing.T) {
	router, s, _, _ := setupTestWeb(t)
	defer s.Close()

	paths := []string{
		"/api/stats",
		"/api/whatsapp/status",
		"/api/contacts",
		"/api/schedules",
		"/api/queue",
		"/ws",
		"/api/ws",
	}
	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s tanpa auth: expected 401, got %d", path, w.Code)
		}
	}
}

// Regresi: /api/auth/login menukar kredensial menjadi bearer token yang valid.
func TestTokenLoginGrantsMobileAPI(t *testing.T) {
	router, s, _, adminUser := setupTestWeb(t)
	defer s.Close()

	body := `{"username":"superadmin","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/auth/login: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Token == "" {
		t.Fatalf("Token tidak ditemukan di respons: %s", w.Body.String())
	}

	req2 := httptest.NewRequest("GET", "/api/stats", nil)
	req2.Header.Set("Authorization", "Bearer "+resp.Token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("GET /api/stats dengan bearer token: expected 200, got %d", w2.Code)
	}
	_ = adminUser
}

// Regresi IDOR: akun kelas B tidak boleh menghapus kontak milik akun kelas A.
func TestContactDeletionIsScopedToOwner(t *testing.T) {
	router, s, authSvc, _ := setupTestWeb(t)
	defer s.Close()

	userA, err := authSvc.RegisterUser("kelas_a", "password123", "kelas", true)
	if err != nil {
		t.Fatalf("Gagal membuat akun kelas A: %v", err)
	}
	userB, err := authSvc.RegisterUser("kelas_b", "password123", "kelas", true)
	if err != nil {
		t.Fatalf("Gagal membuat akun kelas B: %v", err)
	}

	contact, err := s.CreateContact(&store.Contact{
		UserID:      userA.ID,
		Name:        "Dosen Milik A",
		Phone:       "081234567890",
		ContactType: "dosen",
	})
	if err != nil {
		t.Fatalf("Gagal membuat kontak milik A: %v", err)
	}

	tokenB, err := authSvc.CreateSession(userB.ID)
	if err != nil {
		t.Fatalf("Gagal membuat sesi B: %v", err)
	}

	req := httptest.NewRequest("POST", "/contacts/"+itoa(contact.ID)+"/delete", nil)
	req.AddCookie(&http.Cookie{Name: "sipen_session", Value: tokenB})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got, err := s.GetContactByID(contact.ID); err != nil || got.ID != contact.ID {
		t.Errorf("Kontak milik A terhapus oleh B (IDOR): err=%v", err)
	}

	// Pemilik sah tetap bisa menghapus
	tokenA, _ := authSvc.CreateSession(userA.ID)
	req2 := httptest.NewRequest("POST", "/contacts/"+itoa(contact.ID)+"/delete", nil)
	req2.AddCookie(&http.Cookie{Name: "sipen_session", Value: tokenA})
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if _, err := s.GetContactByID(contact.ID); err == nil {
		t.Errorf("Kontak milik A seharusnya terhapus oleh pemiliknya")
	}
}

// Regresi: healthz publik tidak boleh membocorkan nomor WhatsApp.
func TestHealthzDoesNotLeakPhone(t *testing.T) {
	router, s, _, _ := setupTestWeb(t)
	defer s.Close()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 from /healthz, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "whatsapp_phone") || strings.Contains(w.Body.String(), "phone") {
		t.Errorf("/healthz membocorkan info telepon: %s", w.Body.String())
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
