package web

import (
	"context"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sipen/internal/auth"
	"sipen/internal/store"
	"sipen/internal/tunnel"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// devAssetsEnabled melaporkan apakah aset dari disk dipakai menimpa aset
// embedded — hanya untuk pengembangan. Instalasi produksi selalu memakai
// aset embedded agar frontend ikut terbarui setelah self-update.
func devAssetsEnabled() bool {
	return os.Getenv("SIPEN_DEV_ASSETS") == "1"
}

// SetupRouter initializes Chi router with middlewares and route handlers
func SetupRouter(h *Handlers, staticFS fs.FS, staticDiskDir string) http.Handler {
	h.SetStaticFS(staticFS)
	r := chi.NewRouter()

	// Standard middlewares
	// CATATAN: middleware.RealIP sengaja TIDAK dipakai. RealIP memercayai
	// X-Forwarded-For dari siapapun pengirimnya sehingga memalsukan IP klien
	// (dan melewati guard loopback pair-phone) cukup dengan satu header.
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// OWASP Security Middlewares
	r.Use(SecurityHeadersMiddleware)
	r.Use(MaxBodyBytesMiddleware(2 << 20)) // Max 2MB payload protection
	r.Use(CSRFOriginMiddleware)

	// Static files server: embedded adalah sumber kebenaran (selalu segar
	// sesuai binary). File disk hanya dipakai di mode dev (SIPEN_DEV_ASSETS=1)
	// supaya instalasi lama tidak menimpa aset baru setelah self-update.
	var filesDir http.FileSystem
	if staticDiskDir != "" && devAssetsEnabled() {
		if fi, err := os.Stat(staticDiskDir); err == nil && fi.IsDir() {
			workDir, _ := filepath.Abs(staticDiskDir)
			filesDir = http.Dir(workDir)
		}
	}
	if filesDir == nil && staticFS != nil {
		filesDir = http.FS(staticFS)
	}
	if filesDir != nil {
		FileServer(r, "/static", filesDir)
	}

	// Rate limiter for authentication endpoints (prevent brute-force / credential stuffing)
	authRateLimiter := NewRateLimiter(10, time.Minute)

	// Public routes
	r.Get("/healthz", h.HealthzHandler)
	r.Get("/api/health", h.HealthzHandler) // Alias for Android MainActivity health check

	// Token exchange untuk aplikasi companion mobile (Flutter/iOS/Android)
	r.With(authRateLimiter.Middleware).Post("/api/auth/login", h.APITokenLoginHandler)

	// WebSocket — terproteksi: broadcast QR code / pairing code / status WA
	// tidak boleh bisa diakses tanpa sesi.
	r.Group(func(ws chi.Router) {
		ws.Use(h.authSvc.RequireAuthWS)
		ws.Get("/ws", h.hub.HandleWS)
		ws.Get("/api/ws", h.hub.HandleWS) // Alias for Mobile & Companion WebSocket telemetry
	})

	// Mobile Companion & IoT REST JSON API (BUG-05 Fix) — kini wajib
	// autentikasi (cookie sesi atau Bearer token dari /api/auth/login).
	r.Group(func(api chi.Router) {
		api.Use(h.authSvc.RequireAuth)
		api.Get("/api/stats", h.APIStatsHandler)
		api.Get("/api/whatsapp/status", h.APIWhatsAppStatusHandler)
		api.Get("/api/contacts", h.APIContactsHandler)
		api.Get("/api/schedules", h.APISchedulesHandler)
		api.Get("/api/queue", h.APIQueueHandler)
		api.Post("/api/queue/process", h.APIQueueProcessHandler)
		api.Post("/api/queue/{id}/process", h.APIQueueProcessHandler)
	})

	// Issue #4: Public Academic Schedule Portal & iCalendar .ics download
	r.Get("/jadwal", h.PublicScheduleHandler)
	r.Get("/public/schedule", h.PublicScheduleHandler)
	r.Get("/jadwal/calendar.ics", h.PublicCalendarICSHandler)

	// System Version & Automatic Update API
	r.Get("/api/system/version", h.SystemVersionHandler)
	r.Get("/api/system/update/check", h.CheckUpdateHandler)
	r.Get("/api/system/update/apk", h.DownloadApkHandler)

	// Internal loopback API for local CLI / Android / Termux commands.
	// Saat tunnel publik aktif, trafik internet juga datang dari 127.0.0.1
	// Internal loopback API for local CLI / Android / Termux commands.
	// Saat tunnel publik aktif atau diakses via remote/LAN, trafik mewajibkan sesi login valid.
	// Saat dipanggil dari localhost (loopback CLI/Android lokal), otomatis menggunakan akun utama (User ID 1).
	loopbackPairPhone := func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host == "" {
			host = r.RemoteAddr
		}
		isLoopback := host == "127.0.0.1" || host == "::1" || host == "localhost"
		tunnelActive := h.tunnelMgr.Status().State == tunnel.StateActive

		user := auth.GetUserFromContext(r.Context())
		if user == nil {
			// Periksa apakah ada cookie sesi atau Bearer token
			token := ""
			if cookie, err := r.Cookie(auth.CookieName); err == nil && cookie.Value != "" {
				token = cookie.Value
			} else {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if token != "" {
				if u, err := h.authSvc.ValidateSession(token); err == nil && u != nil {
					user = u
					r = r.WithContext(context.WithValue(r.Context(), auth.UserContextKey, u))
				}
			}
		}

		if !isLoopback || tunnelActive {
			if user == nil {
				http.Error(w, "Forbidden: loopback only", http.StatusForbidden)
				return
			}
		} else if user == nil {
			// Skenario CLI lokal (sipen pair) / Android loopback tanpa sesi
			u, err := h.store.GetUserByID(1)
			if err == nil && u != nil {
				r = r.WithContext(context.WithValue(r.Context(), auth.UserContextKey, u))
			} else {
				fallbackUser := &store.User{ID: 1, Username: "admin", Role: "superadmin"}
				r = r.WithContext(context.WithValue(r.Context(), auth.UserContextKey, fallbackUser))
			}
		}

		h.WhatsAppPairPhoneHandler(w, r)
	}
	r.Post("/api/internal/pair-phone", loopbackPairPhone)
	r.Post("/api/whatsapp/pair-phone", loopbackPairPhone)

	// Authentication routes
	r.Get("/login", h.LoginHandler)
	r.With(authRateLimiter.Middleware).Post("/login", h.LoginPostHandler)
	r.Get("/login/2fa", h.Login2FAHandler)
	r.With(authRateLimiter.Middleware).Post("/login/2fa", h.Login2FAPostHandler)

	r.Get("/register", h.RegisterHandler)
	r.Post("/register", h.RegisterPostHandler)
	r.Get("/logout", h.LogoutHandler)
	r.Post("/logout", h.LogoutHandler)

	// SEO, Discovery & PWA routes
	r.Get("/robots.txt", h.RobotsHandler)
	r.Get("/sitemap.xml", h.SitemapHandler)
	r.Get("/manifest.json", h.ManifestHandler)
	r.Get("/favicon.ico", h.FaviconHandler)

	// Informational & Error routes
	r.Get("/terms", h.TermsHandler)
	r.Get("/tos", h.TermsHandler)
	r.Get("/privacy", h.PrivacyHandler)
	r.Get("/about", h.AboutHandler)
	r.Get("/changelog", h.ChangelogHandler)
	r.Get("/terminal", h.TerminalHandler)
	r.Get("/404", h.NotFoundHandler)
	r.Get("/500", h.InternalServerErrorHandler)
	r.NotFound(h.NotFoundHandler)

	// Protected routes (require session)
	r.Group(func(protected chi.Router) {
		protected.Use(h.authSvc.RequireAuth)

		// Overview
		protected.Get("/", h.OverviewHandler)
		protected.Get("/dashboard", h.OverviewHandler)

		// Workspaces
		protected.Route("/workspaces", func(wr chi.Router) {
			wr.Get("/", h.WorkspacesPageHandler)
			wr.Post("/create", h.CreateWorkspaceHandler)
			wr.Get("/switch/{id}", h.SwitchWorkspaceHandler)
			wr.Post("/switch/{id}", h.SwitchWorkspaceHandler)
			wr.Post("/switch", h.SwitchWorkspaceHandler)
			wr.Post("/{id}/update", h.UpdateWorkspaceHandler)
			wr.Post("/{id}/delete", h.DeleteWorkspaceHandler)
			wr.Post("/{id}/members/add", h.AddWorkspaceMemberHandler)
			wr.Post("/{id}/members/{userId}/remove", h.RemoveWorkspaceMemberHandler)
		})

		// Contacts
		protected.Route("/contacts", func(cr chi.Router) {
			cr.Get("/", h.ContactsHandler)
			cr.Post("/create", h.CreateContactHandler)
			cr.Post("/{id}/update", h.UpdateContactHandler)
			cr.Post("/{id}/delete", h.DeleteContactHandler)
		})

		// Schedules
		protected.Route("/schedules", func(sr chi.Router) {
			sr.Get("/", h.SchedulesHandler)
			sr.Post("/create", h.CreateScheduleHandler)
			sr.Post("/import", h.ImportSchedulesHandler)
			sr.Post("/{id}/update", h.UpdateScheduleHandler)
			sr.Post("/{id}/delete", h.DeleteScheduleHandler)
			sr.Post("/{id}/toggle", h.ToggleScheduleHandler)
			sr.Post("/{id}/trigger", h.TriggerScheduleHandler)
		})

		// Templates
		protected.Route("/templates", func(tr chi.Router) {
			tr.Get("/", h.TemplatesHandler)
			tr.Post("/create", h.CreateTemplateHandler)
			tr.Post("/{id}/update", h.UpdateTemplateHandler)
			tr.Post("/{id}/delete", h.DeleteTemplateHandler)
			tr.Post("/preview", h.PreviewTemplateHandler)
		})

		// History
		protected.Get("/history", h.HistoryHandler)

		// Queue
		protected.Route("/queue", func(qr chi.Router) {
			qr.Get("/", h.QueueHandler)
			qr.Post("/{id}/send-now", h.QueueSendNowHandler)
			qr.Post("/{id}/cancel", h.QueueCancelHandler)
			qr.Post("/{id}/delete", h.QueueDeleteHandler)
		})

		// Settings
		protected.Route("/settings", func(set chi.Router) {
			set.Get("/", h.SettingsHandler)
			set.Post("/update", h.UpdateSettingsHandler)
			set.Post("/holidays/create", h.CreateHolidayHandler)
			set.Post("/holidays/{id}/delete", h.DeleteHolidayHandler)
			set.Post("/tunnel/toggle", h.ToggleTunnelHandler)

			// Admin-restricted user creation (Only admins can create accounts)
			set.Group(func(admin chi.Router) {
				admin.Use(auth.RequireAdmin)
				admin.Post("/users/create", h.CreateUserHandler)
			})

			// SuperAdmin-restricted sensitive endpoints (OWASP BFLA remediation)
			set.Group(func(super chi.Router) {
				super.Use(auth.RequireSuperAdmin)
				super.Post("/registration/toggle", h.ToggleRegistrationHandler)
				super.Get("/backup/download", h.BackupDownloadHandler)
				super.Post("/users/{id}/delete", h.DeleteUserHandler)
			})
		})

		// Logs
		protected.Get("/logs", h.LogsHandler)

		// System Update Apply (SuperAdmin only — unduhan binary adalah risiko RCE)
		protected.Group(func(super chi.Router) {
			super.Use(auth.RequireSuperAdmin)
			super.Post("/api/system/update/apply", h.ApplyUpdateHandler)
		})

		// WhatsApp Actions API (Scoped to User)
		protected.Route("/api/wa", func(war chi.Router) {
			war.Get("/qr", h.WhatsAppQRHandler)
			war.Post("/pair-phone", h.WhatsAppPairPhoneHandler)
			war.Post("/reconnect", h.WhatsAppReconnectHandler)
			war.Post("/disconnect", h.WhatsAppDisconnectHandler)
			war.Post("/test-send", h.WhatsAppTestSendHandler)
			war.Get("/groups", h.WhatsAppGroupsHandler)     // Issue #2
			war.Get("/contacts", h.WhatsAppContactsHandler) // Issue #2
		})
		// Aliases for convenience
		protected.Get("/api/whatsapp/groups", h.WhatsAppGroupsHandler)
		protected.Get("/api/whatsapp/contacts", h.WhatsAppContactsHandler)

		// Two-Factor Authentication API
		protected.Route("/api/2fa", func(twofa chi.Router) {
			twofa.Get("/setup", h.TwoFactorSetupAPIHandler)
			twofa.Post("/enable", h.TwoFactorEnableAPIHandler)
			twofa.Post("/disable", h.TwoFactorDisableAPIHandler)
		})

		// Tunnel status API
		protected.Get("/api/tunnel/status", h.TunnelStatusHandler)
	})

	return r
}

// FileServer conveniently sets up a http.FileServer handler to serve static files
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
