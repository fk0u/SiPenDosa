package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware(dummy)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8473/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("expected X-Frame-Options: SAMEORIGIN, got %s", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy: strict-origin-when-cross-origin, got %s", rec.Header().Get("Referrer-Policy"))
	}
}

func TestCSRFOriginMiddleware(t *testing.T) {
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CSRFOriginMiddleware(dummy)

	// Valid origin
	reqValid := httptest.NewRequest(http.MethodPost, "http://localhost:8473/api/test", nil)
	reqValid.Header.Set("Origin", "http://localhost:8473")
	recValid := httptest.NewRecorder()
	handler.ServeHTTP(recValid, reqValid)
	if recValid.Code != http.StatusOK {
		t.Errorf("expected 200 OK for matching origin, got %d", recValid.Code)
	}

	// Malicious cross-site origin
	reqBad := httptest.NewRequest(http.MethodPost, "http://localhost:8473/api/test", nil)
	reqBad.Header.Set("Origin", "http://evil-attacker.com")
	recBad := httptest.NewRecorder()
	handler.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for cross-site origin, got %d", recBad.Code)
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	dummy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := NewRateLimiter(2, 500*time.Millisecond)
	handler := rl.Middleware(dummy)

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8473/login", nil)
	req.RemoteAddr = "192.168.1.50:12345"

	// 1st request -> OK
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Errorf("1st req: expected 200, got %d", rec1.Code)
	}

	// 2nd request -> OK
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Errorf("2nd req: expected 200, got %d", rec2.Code)
	}

	// 3rd request -> Rate limited (429)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("3rd req: expected 429 Too Many Requests, got %d", rec3.Code)
	}
}
