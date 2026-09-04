package web

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRouter initializes Chi router with middlewares and route handlers
func SetupRouter(h *Handlers, staticFS fs.FS, staticDiskDir string) http.Handler {
	r := chi.NewRouter()

	// Standard middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Static files server with disk and embedded fallback
	var filesDir http.FileSystem
	if staticDiskDir != "" {
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

	// Public routes
	r.Get("/healthz", h.HealthzHandler)
	r.Get("/ws", h.hub.HandleWS)

	r.Get("/login", h.LoginHandler)
	r.Post("/login", h.LoginPostHandler)
	r.Get("/register", h.RegisterHandler)
	r.Post("/register", h.RegisterPostHandler)
	r.Get("/logout", h.LogoutHandler)
	r.Post("/logout", h.LogoutHandler)

	// Informational & Error routes
	r.Get("/terms", h.TermsHandler)
	r.Get("/tos", h.TermsHandler)
	r.Get("/privacy", h.PrivacyHandler)
	r.Get("/about", h.AboutHandler)
	r.Get("/404", h.NotFoundHandler)
	r.Get("/500", h.InternalServerErrorHandler)
	r.NotFound(h.NotFoundHandler)

	// Protected routes (require session)
	r.Group(func(protected chi.Router) {
		protected.Use(h.authSvc.RequireAuth)

		// Overview
		protected.Get("/", h.OverviewHandler)
		protected.Get("/dashboard", h.OverviewHandler)

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
			set.Post("/registration/toggle", h.ToggleRegistrationHandler)
			set.Get("/backup/download", h.BackupDownloadHandler)
			set.Post("/users/{id}/delete", h.DeleteUserHandler)
		})

		// Logs
		protected.Get("/logs", h.LogsHandler)

		// WhatsApp Actions API
		protected.Route("/api/wa", func(war chi.Router) {
			war.Get("/qr", h.WhatsAppQRHandler)
			war.Post("/reconnect", h.WhatsAppReconnectHandler)
			war.Post("/disconnect", h.WhatsAppDisconnectHandler)
			war.Post("/test-send", h.WhatsAppTestSendHandler)
		})
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
