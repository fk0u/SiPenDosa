package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"sipen/internal/store"
	"sipen/internal/totp"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	UserContextKey contextKey = "auth_user"
	CookieName                = "sipen_session"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrRegistrationClosed = errors.New("pendaftaran pengguna baru hanya dapat dibuat oleh Administrator")
	ErrUserExists         = errors.New("username sudah digunakan")
	ErrInvalidCreds       = errors.New("username atau password salah")
	ErrInvalid2FA         = errors.New("kode verifikasi 2FA tidak valid atau telah kedaluwarsa")
)

// Service handles authentication and user authorization
type Service struct {
	store *store.Store
}

// NewService creates a new authentication service
func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

// RegisterFirstUser registers the very first user as superadmin
func (s *Service) RegisterFirstUser(username, password string) (*store.User, error) {
	count, err := s.store.CountUsers()
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("superadmin sudah terdaftar")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(username, string(hash), "superadmin")
	if err != nil {
		return nil, err
	}

	// Close registration automatically once superadmin is registered
	_ = s.store.SetRegistrationOpen(false)
	s.store.AddActivityLog("auth", "Inisialisasi SuperAdmin Pertama", "Username: "+username)

	return user, nil
}

// RegisterUser registers a new user if registration is open or requested by superadmin
func (s *Service) RegisterUser(username, password, role string, allowOverride bool) (*store.User, error) {
	if !allowOverride {
		settings, err := s.store.GetSettings()
		if err != nil || !settings.RegistrationOpen {
			return nil, ErrRegistrationClosed
		}
	}

	if role != "superadmin" && role != "admin" && role != "user" && role != "kelas" {
		role = "kelas"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(username, string(hash), role)
	if err != nil {
		return nil, ErrUserExists
	}

	// Auto-assign new user to default workspace if available
	ws, err := s.store.GetWorkspaceByID(1)
	if err == nil && ws != nil {
		_ = s.store.AddWorkspaceMember(ws.ID, user.ID, "member")
		_ = s.store.SetActiveWorkspace(user.ID, ws.ID)
	}

	s.store.AddActivityLog("auth", "Pendaftaran Pengguna Baru", "User: "+username+" Role: "+role)
	return user, nil
}

// Authenticate verifies credentials and returns the User
func (s *Service) Authenticate(username, password string) (*store.User, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	return user, nil
}

// CreateSession generates a new session token and stores it in the database
func (s *Service) CreateSession(userID int64) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	if err := s.store.CreateSession(token, userID, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

// ValidateSession verifies token and returns user
func (s *Service) ValidateSession(token string) (*store.User, error) {
	sess, err := s.store.GetSession(token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := s.store.GetUserByID(sess.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

// RevokeSession removes the session token
func (s *Service) RevokeSession(token string) error {
	return s.store.DeleteSession(token)
}

// SetSessionCookie sets the secure HTTP cookie. secure harus true jika
// aplikasi disajikan lewat HTTPS (mis. di belakang tunnel Cloudflare).
func SetSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the cookie
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireAuth middleware protects routes from unauthenticated access
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if cookie, err := r.Cookie(CookieName); err == nil && cookie.Value != "" {
			token = cookie.Value
		} else {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		isAPIRequest := strings.HasPrefix(r.URL.Path, "/api/") || strings.Contains(r.Header.Get("Accept"), "application/json")

		if token == "" {
			if isAPIRequest {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Autentikasi diperlukan. Silakan login atau sertakan token."}`))
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := s.ValidateSession(token)
		if err != nil {
			ClearSessionCookie(w)
			if isAPIRequest {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized","message":"Sesi autentikasi telah kedaluwarsa atau tidak valid."}`))
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthWS protects WebSocket upgrade requests. Accepts session cookie,
// ?token= query parameter (iOS URLSession cannot set custom headers), or Bearer header.
// Unlike RequireAuth it never redirects — failed handshakes get a plain 401.
func (s *Service) RequireAuthWS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			if cookie, err := r.Cookie(CookieName); err == nil {
				token = cookie.Value
			}
		}
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		user, err := s.ValidateSession(token)
		if err != nil || user == nil {
			http.Error(w, "Unauthorized: WebSocket memerlukan sesi yang valid", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Create2FAChallenge creates a short-lived token (5 minutes) for verifying 2FA code
func (s *Service) Create2FAChallenge(userID int64) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	expiresAt := time.Now().Add(5 * time.Minute)

	if err := s.store.Create2FAChallenge(token, userID, expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

// Verify2FA validates the 6-digit TOTP code against the challenge and user's 2FA secret
func (s *Service) Verify2FA(challengeToken, passcode string) (*store.User, error) {
	challenge, err := s.store.Get2FAChallenge(challengeToken)
	if err != nil {
		return nil, ErrInvalid2FA
	}

	user, err := s.store.GetUserByID(challenge.UserID)
	if err != nil {
		return nil, ErrInvalid2FA
	}

	if !totp.ValidatePasscode(user.TwoFactorSecret, passcode, 1) {
		return nil, ErrInvalid2FA
	}

	// Delete challenge after successful consumption
	_ = s.store.Delete2FAChallenge(challengeToken)
	return user, nil
}

// RequireAdmin middleware restricts access to Admins or SuperAdmins
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil || (user.Role != "superadmin" && user.Role != "admin") {
			http.Error(w, "Akses ditolak: Hanya Administrator yang memiliki izin tindakan ini", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSuperAdmin middleware restricts access to SuperAdmins only
func RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil || user.Role != "superadmin" {
			http.Error(w, "Akses ditolak: Hanya SuperAdmin yang memiliki izin tindakan ini", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the authenticated user from the context
func GetUserFromContext(ctx context.Context) *store.User {
	if user, ok := ctx.Value(UserContextKey).(*store.User); ok {
		return user
	}
	return nil
}
