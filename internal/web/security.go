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

			// If both are missing, check if it's an internal API/CLI request or loopback
			host := r.Host
			if origin != "" {
				// Strip protocol
				originHost := origin
				if idx := strings.Index(originHost, "://"); idx != -1 {
					originHost = originHost[idx+3:]
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
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedHost(clientHost, serverHost string) bool {
	// Allow loopback alias matches (localhost, 127.0.0.1)
	cBase, _, _ := net.SplitHostPort(clientHost)
	if cBase == "" {
		cBase = clientHost
	}
	sBase, _, _ := net.SplitHostPort(serverHost)
	if sBase == "" {
		sBase = serverHost
	}

	if (cBase == "localhost" || cBase == "127.0.0.1") && (sBase == "localhost" || sBase == "127.0.0.1") {
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
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return strings.TrimSpace(ip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
