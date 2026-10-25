package web

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SecurityHeadersMiddleware injects recommended OWASP security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Prevent clickjacking via iframes
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		// XSS protection for older browsers
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Permissions Policy
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		
		next.ServeHTTP(w, r)
	})
}

// MaxBodyBytesMiddleware limits request body size to prevent DoS memory exhaustion
func MaxBodyBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRFOriginMiddleware validates Origin and Referer for state-changing HTTP methods (POST, PUT, DELETE, PATCH)
func CSRFOriginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete || method == http.MethodPatch {
			origin := r.Header.Get("Origin")
			referer := r.Header.Get("Referer")

			// BUG-06 Fix: Support X-Forwarded-Host for reverse proxies (Caddy, Nginx, Cloudflare Tunnels)
			host := r.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = r.Host
			}
			// Handle comma-separated X-Forwarded-Host
			if commaIdx := strings.Index(host, ","); commaIdx != -1 {
				host = strings.TrimSpace(host[:commaIdx])
			}

			if origin != "" {
				// Strip protocol
				originHost := origin
				if idx := strings.Index(originHost, "://"); idx != -1 {
					originHost = originHost[idx+3:]
				}
				// Strip path if any
				if slashIdx := strings.Index(originHost, "/"); slashIdx != -1 {
					originHost = originHost[:slashIdx]
				}
				if !strings.EqualFold(originHost, host) && !isAllowedHost(originHost, host) {
					http.Error(w, "Forbidden: CSRF Origin validation failed", http.StatusForbidden)
					return
				}
			} else if referer != "" {
				refHost := referer
				if idx := strings.Index(refHost, "://"); idx != -1 {
					refHost = refHost[idx+3:]
				}
				if slashIdx := strings.Index(refHost, "/"); slashIdx != -1 {
					refHost = refHost[:slashIdx]
				}
				if !strings.EqualFold(refHost, host) && !isAllowedHost(refHost, host) {
					http.Error(w, "Forbidden: CSRF Referer validation failed", http.StatusForbidden)
					return
				}
			} else if !isCSRFExempt(r) {
				// Fail-closed: permintaan state-changing yang diautentikasi cookie
				// WAJIB membawa Origin/Referer (browser selalu mengirimkannya).
				// Permintaan Bearer token (API mobile) bebas CSRF karena tidak
				// memakai cookie, dan loopback internal dipercaya secara desain.
				http.Error(w, "Forbidden: permintaan state-changing memerlukan header Origin", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// isCSRFExempt melonggarkan validasi untuk klien non-browser yang aman dari CSRF:
// auth Bearer (tanpa cookie) dan koneksi loopback internal.
func isCSRFExempt(r *http.Request) bool {
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		return true
	}
	// Cookie session ada? Jika tidak ada sama sekali (anon/loopback CLI), tidak
	// ada sesi yang bisa diserang via CSRF.
	if _, err := r.Cookie("sipen_session"); err != nil {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host == "127.0.0.1" || host == "::1"
}

func isAllowedHost(clientHost, serverHost string) bool {
	cBase, _, _ := net.SplitHostPort(clientHost)
	if cBase == "" {
		cBase = clientHost
	}
	sBase, _, _ := net.SplitHostPort(serverHost)
	if sBase == "" {
		sBase = serverHost
	}

	// Direct match of hostnames (disregarding port differences from reverse proxy)
	if strings.EqualFold(cBase, sBase) {
		return true
	}

	// Allow loopback alias matches (localhost, 127.0.0.1, ::1)
	isLoopback := func(h string) bool {
		return h == "localhost" || h == "127.0.0.1" || h == "::1" || strings.HasPrefix(h, "127.")
	}
	if isLoopback(cBase) && isLoopback(sBase) {
		return true
	}
	return false
}

// RateLimiter implements a token bucket / sliding window rate limiter per client IP
type RateLimiter struct {
	mu     sync.Mutex
	visits map[string][]time.Time
	limit  int           // max requests
	window time.Duration // time window
}

// NewRateLimiter creates a rate limiter allowing `limit` requests per `window`
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visits: make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}

	// Periodic cleanup of stale IP entries
	go func() {
		ticker := time.NewTicker(2 * window)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.window)
			for ip, times := range rl.visits {
				var valid []time.Time
				for _, t := range times {
					if t.After(cutoff) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(rl.visits, ip)
				} else {
					rl.visits[ip] = valid
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// Middleware returns HTTP handler enforcing the rate limit
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)

		rl.mu.Lock()
		times, exists := rl.visits[ip]
		now := time.Now()
		cutoff := now.Add(-rl.window)

		var valid []time.Time
		if exists {
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
		}

		if len(valid) >= rl.limit {
			rl.mu.Unlock()
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
			http.Error(w, "Terlalu banyak percobaan. Harap tunggu beberapa saat sebelum mencoba lagi.", http.StatusTooManyRequests)
			return
		}

		valid = append(valid, now)
		rl.visits[ip] = valid
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	// Header X-Forwarded-For hanya dipercaya jika koneksi datang dari loopback
	// (di belakang cloudflared / reverse proxy lokal). Data XFF yang di-inject
	// langsung oleh klien dari luar tidak pernah dipakai, dan meskipun lewat
	// proxy kita ambil entri TERAKHIR (yang diisi oleh proxy, bukan klien).
	if host == "127.0.0.1" || host == "::1" {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if candidate := strings.TrimSpace(parts[len(parts)-1]); candidate != "" {
				return candidate
			}
		}
	}
	return host
}
