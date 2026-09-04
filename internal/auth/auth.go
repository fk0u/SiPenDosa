package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"sipen/internal/store"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	UserContextKey contextKey = "auth_user"
	CookieName                = "sipen_session"
)

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrRegistrationClosed = errors.New("pendaftaran pengguna baru sedang ditutup")
	ErrUserExists       = errors.New("username sudah digunakan")
	ErrInvalidCreds     = errors.New("username atau password salah")
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

	if role != "superadmin" && role != "admin" {
		role = "admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(username, string(hash), role)
	if err != nil {
		return nil, ErrUserExists
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

// SetSessionCookie sets the secure HTTP cookie
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
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
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireAuth middleware protects routes from unauthenticated access
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := s.ValidateSession(cookie.Value)
		if err != nil {
			ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
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
