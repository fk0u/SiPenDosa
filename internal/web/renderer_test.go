package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"sipen/internal/store"
	webassets "sipen/web"
)

func TestAllTemplatesRender(t *testing.T) {
	// Ensure working directory is workspace root
	err := os.Chdir("../..")
	if err != nil {
		t.Skip("Skipping template test: cannot chdir to root")
	}

	renderer := NewViewRenderer(webassets.Templates(), "web/templates")

	dummyUser := &store.User{
		ID:       1,
		Username: "admin_test",
		Role:     "superadmin",
	}

	dummyPageData := PageData{
		Title:       "Test Page",
		ActivePage:  "overview",
		User:        dummyUser,
		CurrentYear: 2026,
		Data: map[string]interface{}{
			"Stats": &store.DashboardStats{
				TotalSentToday:  5,
				TotalSentAll:    42,
				TotalFailedAll:  0,
				TotalPending:    1,
				TotalSchedules:  8,
				ActiveSchedules: 8,
				TotalContacts:   12,
				SuccessRate:     100.0,
			},
			"WANumber":       "628111222333@s.whatsapp.net",
			"WAConnected":    true,
			"SentToday":      5,
			"TotalSent":      42,
			"FailedCount":    0,
			"PendingCount":   1,
			"SuccessRate":    "100.0%",
			"RecentMessages": []store.QueueMessage{},
			"NextSchedule":   "Tidak ada",
			"Contacts":       []store.Contact{},
			"Schedules":      []store.ScheduleDetail{},
			"Templates": []store.Template{
				{
					ID:        1,
					Name:      "Pengingat Dosen",
					Content:   "Halo {{.NamaDosen}}, besok ada kuliah {{.Matkul}}",
					IsDefault: true,
				},
			},
			"Selected": &store.Template{
				ID:        1,
				Name:      "Pengingat Dosen",
				Content:   "Halo {{.NamaDosen}}, besok ada kuliah {{.Matkul}}",
				IsDefault: true,
			},
			"PreviewText":    "Halo Dr. Budi, besok ada kuliah Sistem Terdistribusi",
			"Versions":       []store.TemplateVersion{},
			"Messages":       []store.QueueMessage{},
			"SelectedStatus": "",
			"Pending":        []store.QueueMessage{},
			"Processing":     []store.QueueMessage{},
			"Settings": &store.Settings{
				Timezone:         "Asia/Makassar",
				SendWindowStart:  "06:00",
				SendWindowEnd:    "21:00",
				RateLimitMinSec:  10,
				RateLimitMaxSec:  30,
				MaxRetries:       3,
				GlobalDryRun:     false,
				RegistrationOpen: true,
			},
			"Holidays":         []store.Holiday{},
			"Users":            []store.User{*dummyUser},
			"Logs":             []store.ActivityLog{},
			"SelectedCategory": "",
			"Timezone":         "Asia/Makassar",
			"ErrorDetails":     "Test internal error stack trace",
			"Now":              time.Now(),
		},
	}

	pagesToTest := []string{
		"pages/overview.html",
		"pages/contacts.html",
		"pages/schedules.html",
		"pages/templates.html",
		"pages/history.html",
		"pages/queue.html",
		"pages/settings.html",
		"pages/logs.html",
		"pages/terms.html",
		"pages/privacy.html",
		"pages/about.html",
		"errors/404.html",
		"errors/500.html",
	}

	for _, page := range pagesToTest {
		t.Run(page, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			renderer.Render(w, req, page, dummyPageData)
			if w.Code != http.StatusOK {
				t.Fatalf("Expected 200 OK for %s, got %d. Body: %s", page, w.Code, w.Body.String())
			}
		})
	}

	// Test Plain Auth Pages
	plainPages := []string{
		"auth/login.html",
		"auth/register.html",
	}

	for _, page := range plainPages {
		t.Run(page, func(t *testing.T) {
			w := httptest.NewRecorder()
			renderer.RenderPlain(w, page, dummyPageData)
			if w.Code != http.StatusOK {
				t.Fatalf("Plain template render error for %s: code %d, body: %s", page, w.Code, w.Body.String())
			}
		})
	}
}

func TestRenderStatusPages(t *testing.T) {
	renderer := NewViewRenderer(webassets.Templates(), "web/templates")

	dummyPageData := PageData{
		Title: "Error Test",
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/404", nil)
	renderer.RenderWithStatus(w, req, "errors/404.html", dummyPageData, http.StatusNotFound)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", w.Code)
	}

	w500 := httptest.NewRecorder()
	req500 := httptest.NewRequest("GET", "/500", nil)
	renderer.RenderWithStatus(w500, req500, "errors/500.html", dummyPageData, http.StatusInternalServerError)
	if w500.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", w500.Code)
	}
}
