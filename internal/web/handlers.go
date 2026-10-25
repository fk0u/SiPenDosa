package web

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"sipen/internal/auth"
	"sipen/internal/config"
	"sipen/internal/queue"
	"sipen/internal/realtime"
	"sipen/internal/scheduler"
	"sipen/internal/store"
	"sipen/internal/template"
	"sipen/internal/totp"
	"sipen/internal/tunnel"
	"sipen/internal/updater"
	"sipen/internal/version"
	"sipen/internal/whatsapp"

	"github.com/go-chi/chi/v5"
)

// Handlers bundles all web application controller dependencies
type Handlers struct {
	cfg        *config.Config
	store      *store.Store
	authSvc    *auth.Service
	waManager  *whatsapp.Manager
	tunnelMgr  *tunnel.Manager
	queueMgr   *queue.Manager
	scheduler  *scheduler.Scheduler
	tmplEngine *template.Engine
	hub        *realtime.Hub
	renderer   *ViewRenderer
	updater    *updater.Manager
}

// NewHandlers creates an instance of application HTTP handlers
func NewHandlers(
	cfg *config.Config,
	s *store.Store,
	a *auth.Service,
	wam *whatsapp.Manager,
	tun *tunnel.Manager,
	q *queue.Manager,
	sch *scheduler.Scheduler,
	t *template.Engine,
	h *realtime.Hub,
	r *ViewRenderer,
) *Handlers {
	return &Handlers{
		cfg:        cfg,
		store:      s,
		authSvc:    a,
		waManager:  wam,
		tunnelMgr:  tun,
		queueMgr:   q,
		scheduler:  sch,
		tmplEngine: t,
		hub:        h,
		renderer:   r,
		updater:    updater.NewManager(),
	}
}

// Helper to retrieve the isolated WhatsApp client for the authenticated user.
// Returns nil when there is no authenticated user — never falls back to another account.
func (h *Handlers) getActiveWAClient(r *http.Request) *whatsapp.Client {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.ID <= 0 {
		return nil
	}
	client, err := h.waManager.GetClient(user.ID)
	if err != nil {
		slog.Error("Failed to get WA client for user", "user_id", user.ID, "err", err)
	}
	return client
}

// Helper to retrieve the current user ID (0 when unauthenticated)
func (h *Handlers) getActiveUserID(r *http.Request) int64 {
	user := auth.GetUserFromContext(r.Context())
	if user != nil && user.ID > 0 {
		return user.ID
	}
	return 0
}

// currentUser returns the authenticated user (nil-safe shorthand)
func (h *Handlers) currentUser(r *http.Request) *store.User {
	return auth.GetUserFromContext(r.Context())
}

// isPrivileged reports whether the user can manage resources across accounts
func isPrivileged(user *store.User) bool {
	return user != nil && (user.Role == "admin" || user.Role == "superadmin")
}

// ==========================================
// Authentication Handlers
// ==========================================

func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Check if any users exist; if not, redirect to register
	count, _ := h.store.CountUsers()
	if count == 0 {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie(auth.CookieName)
	if err == nil && cookie.Value != "" {
		if _, err := h.authSvc.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	h.renderer.RenderPlain(w, "auth/login.html", PageData{
		Title: "Masuk — SiPenDosa",
	})
}

func (h *Handlers) LoginPostHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, err := h.authSvc.Authenticate(username, password)
	if err != nil {
		h.renderer.RenderPlain(w, "auth/login.html", PageData{
			Title:      "Masuk — SiPenDosa",
			FlashError: "Username atau kata sandi salah.",
		})
		return
	}

	// Check if 2FA is enabled for this account
	if user.TwoFactorEnabled {
		challengeID, err := h.authSvc.Create2FAChallenge(user.ID)
		if err != nil {
			h.renderer.RenderPlain(w, "auth/login.html", PageData{
				Title:      "Masuk — SiPenDosa",
				FlashError: "Gagal membuat sesi 2FA: " + err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/login/2fa?challenge="+challengeID, http.StatusSeeOther)
		return
	}

	token, err := h.authSvc.CreateSession(user.ID)
	if err != nil {
		h.renderer.RenderPlain(w, "auth/login.html", PageData{
			Title:      "Masuk — SiPenDosa",
			FlashError: "Gagal membuat sesi login.",
		})
		return
	}

	auth.SetSessionCookie(w, token, h.cfg.CookieSecure)
	h.store.AddActivityLog("auth", "User Login Berhasil", "Username: "+user.Username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) Login2FAHandler(w http.ResponseWriter, r *http.Request) {
	challenge := strings.TrimSpace(r.URL.Query().Get("challenge"))
	if challenge == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.renderer.RenderPlain(w, "auth/two_factor.html", PageData{
		Title: "Verifikasi Dua Langkah (2FA) — SiPenDosa",
		Data: map[string]interface{}{
			"Challenge": challenge,
		},
	})
}

func (h *Handlers) Login2FAPostHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	challenge := strings.TrimSpace(r.FormValue("challenge"))
	code := strings.TrimSpace(r.FormValue("code"))

	user, err := h.authSvc.Verify2FA(challenge, code)
	if err != nil {
		h.renderer.RenderPlain(w, "auth/two_factor.html", PageData{
			Title:      "Verifikasi Dua Langkah (2FA) — SiPenDosa",
			FlashError: "Kode autentikasi 2FA tidak valid atau telah kadaluarsa.",
			Data: map[string]interface{}{
				"Challenge": challenge,
			},
		})
		return
	}

	token, err := h.authSvc.CreateSession(user.ID)
	if err != nil {
		h.renderer.RenderPlain(w, "auth/two_factor.html", PageData{
			Title:      "Verifikasi Dua Langkah (2FA) — SiPenDosa",
			FlashError: "Gagal membuat sesi login.",
			Data: map[string]interface{}{
				"Challenge": challenge,
			},
		})
		return
	}

	auth.SetSessionCookie(w, token, h.cfg.CookieSecure)
	h.store.AddActivityLog("auth", "User Login 2FA Berhasil", "Username: "+user.Username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	count, _ := h.store.CountUsers()
	isFirstUser := count == 0

	// If superadmin already exists, public registration is completely closed!
	if !isFirstUser {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.renderer.RenderPlain(w, "auth/register.html", PageData{
		Title:          "Inisialisasi Superadmin — SiPenDosa",
		IsRegistration: true,
	})
}

func (h *Handlers) RegisterPostHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	count, _ := h.store.CountUsers()
	isFirstUser := count == 0

	// Disallow public registration if user already exists
	if !isFirstUser {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:      "Pendaftaran Ditutup — SiPenDosa",
			FlashError: "Pendaftaran publik dinonaktifkan. Akun baru hanya dapat dibuat oleh Administrator melalui Pengaturan.",
		})
		return
	}

	if username == "" || len(password) < 6 {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:          "Inisialisasi Superadmin — SiPenDosa",
			IsRegistration: isFirstUser,
			FlashError:     "Username wajib diisi dan kata sandi minimal 6 karakter.",
		})
		return
	}

	if password != confirmPassword {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:          "Inisialisasi Superadmin — SiPenDosa",
			IsRegistration: isFirstUser,
			FlashError:     "Konfirmasi kata sandi tidak cocok.",
		})
		return
	}

	_, err := h.authSvc.RegisterFirstUser(username, password)
	if err != nil {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:          "Inisialisasi Superadmin — SiPenDosa",
			IsRegistration: isFirstUser,
			FlashError:     err.Error(),
		})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.CookieName)
	if err == nil && cookie.Value != "" {
		_ = h.authSvc.RevokeSession(cookie.Value)
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ==========================================
// 2FA Management API (Protected)
// ==========================================

func (h *Handlers) TwoFactorSetupAPIHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	secret, err := totp.GenerateSecret(20)
	if err != nil {
		http.Error(w, "Gagal membuat rahasia 2FA: "+err.Error(), http.StatusInternalServerError)
		return
	}

	qrCode, err := totp.GenerateQRCodeDataURL(user.Username, secret, "SiPenDosa")
	if err != nil {
		http.Error(w, "Gagal membuat QR code 2FA: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"secret":  secret,
		"qr_code": qrCode,
	})
}

func (h *Handlers) TwoFactorEnableAPIHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.Secret = strings.TrimSpace(req.Secret)
	req.Code = strings.TrimSpace(req.Code)

	if !totp.ValidatePasscode(req.Secret, req.Code, 1) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Kode verifikasi 6-digit salah atau kadaluarsa. Pastikan jam pada perangkat Anda akurat.",
		})
		return
	}

	if err := h.store.UpdateUser2FA(user.ID, req.Secret, true); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Gagal menyimpan status 2FA: " + err.Error(),
		})
		return
	}

	h.store.AddActivityLog("auth", "2FA Diaktifkan", fmt.Sprintf("User %s mengaktifkan 2FA", user.Username))
	h.hub.BroadcastToast("success", "Two-Factor Authentication (2FA) berhasil diaktifkan!")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Two-Factor Authentication berhasil diaktifkan.",
	})
}

func (h *Handlers) TwoFactorDisableAPIHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify current password before disabling
	if _, err := h.authSvc.Authenticate(user.Username, req.Password); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Kata sandi saat ini tidak valid.",
		})
		return
	}

	if err := h.store.UpdateUser2FA(user.ID, "", false); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Gagal menonaktifkan 2FA: " + err.Error(),
		})
		return
	}

	h.store.AddActivityLog("auth", "2FA Dinonaktifkan", fmt.Sprintf("User %s menonaktifkan 2FA", user.Username))
	h.hub.BroadcastToast("info", "Two-Factor Authentication telah dinonaktifkan.")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Two-Factor Authentication berhasil dinonaktifkan.",
	})
}

// ==========================================
// Admin-Only User Management Handlers
// ==========================================

func (h *Handlers) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	role := strings.TrimSpace(r.FormValue("role"))

	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil || (currentUser.Role != "admin" && currentUser.Role != "superadmin") {
		http.Error(w, "Akses ditolak: Hanya administrator yang dapat membuat akun", http.StatusForbidden)
		return
	}

	// Peran default untuk akun baru adalah "kelas" (penanggung jawab kelas).
	// Peran admin hanya boleh diberikan oleh superadmin.
	if role != "kelas" && role != "user" && role != "admin" {
		role = "kelas"
	}
	if role == "admin" && currentUser.Role != "superadmin" {
		role = "kelas"
	}

	if username == "" || len(password) < 6 {
		h.hub.BroadcastToast("error", "Username wajib diisi dan kata sandi minimal 6 karakter.")
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	newUser, err := h.authSvc.RegisterUser(username, password, role, true)
	if err != nil {
		h.hub.BroadcastToast("error", "Gagal membuat akun: "+err.Error())
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	h.store.AddActivityLog("user_management", "Akun Baru Dibuat", fmt.Sprintf("Admin %s membuat akun %s (Role: %s)", currentUser.Username, newUser.Username, newUser.Role))
	h.hub.BroadcastToast("success", fmt.Sprintf("Akun %s (%s) berhasil dibuat!", newUser.Username, newUser.Role))
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handlers) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser != nil && currentUser.ID == id {
		h.hub.BroadcastToast("error", "Anda tidak dapat menghapus akun Anda sendiri.")
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	// Close WhatsApp client for this user if running
	h.waManager.CloseClient(id)

	_ = h.store.DeleteUser(id)
	h.store.AddActivityLog("user_management", "Akun Pengguna Dihapus", fmt.Sprintf("User ID %d dihapus", id))
	h.hub.BroadcastToast("info", "Pengguna telah dihapus.")
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// ==========================================
// Overview Dashboard
// ==========================================

func (h *Handlers) OverviewHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	stats, err := h.store.GetDashboardStats(userID)
	if err != nil || stats == nil {
		slog.Error("Failed getting dashboard stats, using safe defaults", "err", err)
		stats = &store.DashboardStats{
			SuccessRate: 100.0,
		}
	}

	waClient := h.getActiveWAClient(r)
	var waState whatsapp.State = whatsapp.StateDisconnected
	var waPhone, waPushName, waQR string
	if waClient != nil {
		waState, waPhone, waPushName, waQR = waClient.Status()
	}

	recentMessages, _ := h.store.ListQueueMessages(userID, "", 6)
	nextInfo := h.scheduler.GetNextDeliveryInfo()
	settings, _ := h.store.GetSettings()
	tunnelStatus := h.tunnelMgr.Status()

	// Salam sesuai waktu setempat untuk header dashboard
	hour := time.Now().In(h.cfg.GetLocation()).Hour()
	greeting := "Selamat malam"
	if hour >= 4 && hour < 11 {
		greeting = "Selamat pagi"
	} else if hour >= 11 && hour < 15 {
		greeting = "Selamat siang"
	} else if hour >= 15 && hour < 18 {
		greeting = "Selamat sore"
	}

	data := map[string]interface{}{
		"Greeting":       greeting,
		"Stats":          stats,
		"WAState":        string(waState),
		"WAPhone":        waPhone,
		"WAPushName":     waPushName,
		"WAQR":           waQR,
		"RecentMessages": recentMessages,
		"NextDelivery":   nextInfo,
		"Settings":       settings,
		"Tunnel":         tunnelStatus,
	}

	h.renderer.Render(w, r, "pages/overview.html", PageData{
		Title:      "Dashboard Overview",
		ActivePage: "overview",
		Data:       data,
	})
}

// ==========================================
// Contacts Handlers (Dosen, Mahasiswa, Komti, Group)
// ==========================================

func (h *Handlers) ContactsHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	filterType := r.URL.Query().Get("type")
	contacts, err := h.store.ListContacts(userID, filterType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Contacts":   contacts,
		"FilterType": filterType,
	}

	h.renderer.Render(w, r, "pages/contacts.html", PageData{
		Title:      "Dosen & Kontak",
		ActivePage: "contacts",
		Data:       data,
	})
}

func (h *Handlers) CreateContactHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	userID := h.getActiveUserID(r)

	c := &store.Contact{
		UserID:      userID,
		Name:        strings.TrimSpace(r.FormValue("name")),
		Phone:       strings.TrimSpace(r.FormValue("phone")),
		ContactType: r.FormValue("contact_type"),
		NIM:         strings.TrimSpace(r.FormValue("nim")),
		Email:       strings.TrimSpace(r.FormValue("email")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
	}

	if c.Name == "" || c.Phone == "" {
		http.Redirect(w, r, "/contacts", http.StatusSeeOther)
		return
	}

	_, err := h.store.CreateContact(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.store.AddActivityLog("contacts", "Kontak Ditambahkan", fmt.Sprintf("Nama: %s, Tipe: %s", c.Name, c.ContactType))
	h.hub.BroadcastToast("success", fmt.Sprintf("Kontak %s berhasil ditambahkan!", c.Name))
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

func (h *Handlers) UpdateContactHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	userID := h.getActiveUserID(r)

	_ = r.ParseForm()
	c := &store.Contact{
		ID:          id,
		UserID:      userID,
		Name:        strings.TrimSpace(r.FormValue("name")),
		Phone:       strings.TrimSpace(r.FormValue("phone")),
		ContactType: r.FormValue("contact_type"),
		NIM:         strings.TrimSpace(r.FormValue("nim")),
		Email:       strings.TrimSpace(r.FormValue("email")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
	}

	if err := h.store.UpdateContact(c); err != nil {
		slog.Error("Gagal memperbarui kontak", "id", id, "err", err)
		h.hub.BroadcastToast("error", "Gagal memperbarui kontak.")
	} else {
		h.store.AddActivityLog("contacts", "Kontak Diperbarui", fmt.Sprintf("ID: %d, Nama: %s", id, c.Name))
		h.hub.BroadcastToast("success", "Kontak berhasil diperbarui!")
	}
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

func (h *Handlers) DeleteContactHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := h.getActiveUserID(r)
	if err := h.store.DeleteContact(id, userID); err != nil {
		h.hub.BroadcastToast("error", "Gagal menghapus kontak.")
	} else {
		h.store.AddActivityLog("contacts", "Kontak Dihapus", fmt.Sprintf("ID: %d", id))
		h.hub.BroadcastToast("info", "Kontak telah dihapus.")
	}
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

// ==========================================
// Schedules Handlers
// ==========================================

func (h *Handlers) SchedulesHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	schedules, _ := h.store.ListSchedules(userID)
	contacts, _ := h.store.ListContacts(userID, "")
	templates, _ := h.store.ListTemplates()

	data := map[string]interface{}{
		"Schedules": schedules,
		"Contacts":  contacts,
		"Templates": templates,
	}

	h.renderer.Render(w, r, "pages/schedules.html", PageData{
		Title:      "Jadwal Mata Kuliah",
		ActivePage: "schedules",
		Data:       data,
	})
}

func (h *Handlers) CreateScheduleHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	userID := h.getActiveUserID(r)

	dayOfWeek, _ := strconv.Atoi(r.FormValue("day_of_week"))
	dosenID, _ := strconv.ParseInt(r.FormValue("dosen_id"), 10, 64)
	recipientID, _ := strconv.ParseInt(r.FormValue("recipient_id"), 10, 64)
	templateID, _ := strconv.ParseInt(r.FormValue("template_id"), 10, 64)

	sc := &store.Schedule{
		UserID:      userID,
		Title:       strings.TrimSpace(r.FormValue("title")),
		Matkul:      strings.TrimSpace(r.FormValue("matkul")),
		TargetPhone: strings.TrimSpace(r.FormValue("target_phone")),
		DayOfWeek:   dayOfWeek,
		StartTime:   r.FormValue("start_time"),
		EndTime:     r.FormValue("end_time"),
		Location:    strings.TrimSpace(r.FormValue("location")),
		LinkGroup:   strings.TrimSpace(r.FormValue("link_group")),
		Mode:        r.FormValue("mode"),
		SendAtTime:  r.FormValue("send_at_time"),
		IsActive:    r.FormValue("is_active") == "on" || r.FormValue("is_active") == "1",
		IsPublic:    r.FormValue("is_public") == "on" || r.FormValue("is_public") == "1",
		DryRun:      r.FormValue("dry_run") == "on" || r.FormValue("dry_run") == "1",
	}

	if dosenID > 0 {
		c, err := h.store.GetContactByID(dosenID)
		if err != nil || c.UserID != userID {
			h.hub.BroadcastToast("error", "Kontak dosen tidak ditemukan pada akun Anda.")
			http.Redirect(w, r, "/schedules", http.StatusSeeOther)
			return
		}
		sc.DosenID = sql.NullInt64{Int64: dosenID, Valid: true}
	}
	if recipientID > 0 {
		c, err := h.store.GetContactByID(recipientID)
		if err != nil || c.UserID != userID {
			h.hub.BroadcastToast("error", "Kontak tujuan tidak ditemukan pada akun Anda.")
			http.Redirect(w, r, "/schedules", http.StatusSeeOther)
			return
		}
		sc.RecipientID = sql.NullInt64{Int64: recipientID, Valid: true}
	}
	if templateID > 0 {
		sc.TemplateID = sql.NullInt64{Int64: templateID, Valid: true}
	}

	_, err := h.store.CreateSchedule(sc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.store.AddActivityLog("scheduler", "Jadwal Baru Ditambahkan", fmt.Sprintf("Matkul: %s, Hari: %d", sc.Matkul, sc.DayOfWeek))
	h.hub.BroadcastToast("success", fmt.Sprintf("Jadwal %s berhasil ditambahkan!", sc.Matkul))
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}

func (h *Handlers) UpdateScheduleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	userID := h.getActiveUserID(r)

	_ = r.ParseForm()
	dayOfWeek, _ := strconv.Atoi(r.FormValue("day_of_week"))
	dosenID, _ := strconv.ParseInt(r.FormValue("dosen_id"), 10, 64)
	recipientID, _ := strconv.ParseInt(r.FormValue("recipient_id"), 10, 64)
	templateID, _ := strconv.ParseInt(r.FormValue("template_id"), 10, 64)

	sc := &store.Schedule{
		ID:          id,
		UserID:      userID,
		Title:       strings.TrimSpace(r.FormValue("title")),
		Matkul:      strings.TrimSpace(r.FormValue("matkul")),
		TargetPhone: strings.TrimSpace(r.FormValue("target_phone")),
		DayOfWeek:   dayOfWeek,
		StartTime:   r.FormValue("start_time"),
		EndTime:     r.FormValue("end_time"),
		Location:    strings.TrimSpace(r.FormValue("location")),
		LinkGroup:   strings.TrimSpace(r.FormValue("link_group")),
		Mode:        r.FormValue("mode"),
		SendAtTime:  r.FormValue("send_at_time"),
		IsActive:    r.FormValue("is_active") == "on" || r.FormValue("is_active") == "1",
		IsPublic:    r.FormValue("is_public") == "on" || r.FormValue("is_public") == "1",
		DryRun:      r.FormValue("dry_run") == "on" || r.FormValue("dry_run") == "1",
	}

	if dosenID > 0 {
		c, err := h.store.GetContactByID(dosenID)
		if err != nil || c.UserID != userID {
			h.hub.BroadcastToast("error", "Kontak dosen tidak ditemukan pada akun Anda.")
			http.Redirect(w, r, "/schedules", http.StatusSeeOther)
			return
		}
		sc.DosenID = sql.NullInt64{Int64: dosenID, Valid: true}
	}
	if recipientID > 0 {
		c, err := h.store.GetContactByID(recipientID)
		if err != nil || c.UserID != userID {
			h.hub.BroadcastToast("error", "Kontak tujuan tidak ditemukan pada akun Anda.")
			http.Redirect(w, r, "/schedules", http.StatusSeeOther)
			return
		}
		sc.RecipientID = sql.NullInt64{Int64: recipientID, Valid: true}
	}
	if templateID > 0 {
		sc.TemplateID = sql.NullInt64{Int64: templateID, Valid: true}
	}

	if err := h.store.UpdateSchedule(sc); err != nil {
		slog.Error("Gagal memperbarui jadwal", "id", id, "err", err)
		h.hub.BroadcastToast("error", "Gagal memperbarui jadwal.")
	} else {
		h.store.AddActivityLog("scheduler", "Jadwal Diperbarui", fmt.Sprintf("ID: %d, Matkul: %s", id, sc.Matkul))
		h.hub.BroadcastToast("success", "Jadwal berhasil diperbarui!")
	}
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}

func (h *Handlers) ToggleScheduleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	sc, err := h.store.GetScheduleByID(id)
	if err == nil && sc.UserID == h.getActiveUserID(r) {
		_ = h.store.ToggleScheduleActive(id, h.getActiveUserID(r), !sc.IsActive)
		statusStr := "diaktifkan"
		if sc.IsActive {
			statusStr = "dinonaktifkan"
		}
		h.hub.BroadcastToast("info", fmt.Sprintf("Jadwal %s %s.", sc.Matkul, statusStr))
	}
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}

func (h *Handlers) TriggerScheduleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_ = r.ParseForm()
	dryRun := r.FormValue("dry_run") == "true" || r.FormValue("dry_run") == "1"

	sc, err := h.store.GetScheduleByID(id)
	if err != nil || sc.UserID != h.getActiveUserID(r) {
		h.hub.BroadcastToast("error", "Akses ditolak: jadwal ini bukan milik akun Anda.")
		http.Redirect(w, r, "/schedules", http.StatusSeeOther)
		return
	}

	msg, err := h.scheduler.TriggerScheduleManually(id, dryRun)
	if err != nil {
		h.hub.BroadcastToast("error", "Gagal memicu jadwal: "+err.Error())
		http.Redirect(w, r, "/schedules", http.StatusSeeOther)
		return
	}

	h.hub.BroadcastToast("success", fmt.Sprintf("Pesan jadwal berhasil dimasukkan antrian (ID: %d)", msg.ID))
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

func (h *Handlers) DeleteScheduleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := h.getActiveUserID(r)
	if err := h.store.DeleteSchedule(id, userID); err != nil {
		h.hub.BroadcastToast("error", "Gagal menghapus jadwal.")
	} else {
		h.store.AddActivityLog("scheduler", "Jadwal Dihapus", fmt.Sprintf("ID: %d", id))
		h.hub.BroadcastToast("info", "Jadwal telah dihapus.")
	}
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}


// ImportSchedulesHandler menerima unggahan CSV jadwal dan membuatnya
// sekaligus untuk akun yang sedang login. Format kolom (header wajib):
// hari,matkul,jam_mulai,jam_selesai,dosen,telepon,ruangan,mode
func (h *Handlers) ImportSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)

	file, _, err := r.FormFile("csv_file")
	if err != nil {
		h.hub.BroadcastToast("error", "Berkas CSV tidak terbaca. Pastikan memilih berkas terlebih dahulu.")
		http.Redirect(w, r, "/schedules", http.StatusSeeOther)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // baris boleh bervariasi; validasi manual

	records, err := reader.ReadAll()
	if err != nil {
		h.hub.BroadcastToast("error", "Format CSV tidak valid: "+err.Error())
		http.Redirect(w, r, "/schedules", http.StatusSeeOther)
		return
	}
	if len(records) < 2 {
		h.hub.BroadcastToast("error", "CSV kosong — sertakan baris header dan minimal satu jadwal.")
		http.Redirect(w, r, "/schedules", http.StatusSeeOther)
		return
	}

	// Peta header -> index kolom (fleksibel terhadap urutan)
	col := map[string]int{}
	for i, name := range records[0] {
		col[strings.ToLower(strings.TrimSpace(name))] = i
	}
	get := func(row []string, key string) string {
		idx, ok := col[key]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	// Muat kontak user sekali untuk pencocokan dosen/telepon
	contacts, _ := h.store.ListContacts(userID, "")
	byPhone := map[string]store.Contact{}
	byName := map[string]store.Contact{}
	for _, c := range contacts {
		if c.Phone != "" {
			byPhone[normalizePhoneCSV(c.Phone)] = c
		}
		byName[strings.ToLower(c.Name)] = c
	}

	dayIndex := map[string]int{
		"minggu": 0, "senin": 1, "selasa": 2, "rabu": 3,
		"kamis": 4, "jumat": 5, "jum'at": 5, "sabtu": 6,
	}

	created, skipped := 0, 0
	var rowErrors []string
	for lineNo, row := range records[1:] {
		lineNum := lineNo + 2 // +1 header, +1 base-1
		dayRaw := strings.ToLower(get(row, "hari"))
		day, ok := dayIndex[dayRaw]
		if !ok {
			if n, err := strconv.Atoi(dayRaw); err == nil && n >= 0 && n <= 6 {
				day = n
				ok = true
			}
		}
		matkul := get(row, "matkul")
		start := get(row, "jam_mulai")
		end := get(row, "jam_selesai")

		if !ok || matkul == "" || start == "" {
			rowErrors = append(rowErrors, fmt.Sprintf("baris %d: kolom hari/matkul/jam_mulai wajib diisi", lineNum))
			skipped++
			continue
		}
		if !isValidTimeCSV(start) || (end != "" && !isValidTimeCSV(end)) {
			rowErrors = append(rowErrors, fmt.Sprintf("baris %d: format jam harus HH:MM", lineNum))
			skipped++
			continue
		}

		mode := strings.ToUpper(get(row, "mode"))
		if mode != "H-0" {
			mode = "H-1"
		}
		sendAt := get(row, "jam_kirim")
		if !isValidTimeCSV(sendAt) {
			if mode == "H-1" {
				sendAt = "20:00"
			} else {
				sendAt = "06:30"
			}
		}

		sc := &store.Schedule{
			UserID:      userID,
			Title:       matkul,
			Matkul:      matkul,
			DayOfWeek:   day,
			StartTime:   start,
			EndTime:     end,
			Location:    get(row, "ruangan"),
			Mode:        mode,
			SendAtTime:  sendAt,
			IsActive:    true,
			TargetPhone: get(row, "telepon"),
		}

		// Tautkan dosen/penerima bila cocok dengan kontak yang ada
		dosenName := get(row, "dosen")
		phone := normalizePhoneCSV(get(row, "telepon"))
		if c, found := byPhone[phone]; found && phone != "" {
			sc.RecipientID = sql.NullInt64{Int64: c.ID, Valid: true}
			sc.TargetPhone = c.Phone
		}
		if dosenName != "" {
			if c, found := byName[strings.ToLower(dosenName)]; found {
				sc.DosenID = sql.NullInt64{Int64: c.ID, Valid: true}
				if !sc.RecipientID.Valid {
					sc.RecipientID = sql.NullInt64{Int64: c.ID, Valid: true}
					sc.TargetPhone = c.Phone
				}
			}
		}

		if _, err := h.store.CreateSchedule(sc); err != nil {
			rowErrors = append(rowErrors, fmt.Sprintf("baris %d: gagal menyimpan (%v)", lineNum, err))
			skipped++
			continue
		}
		created++
	}

	h.store.AddActivityLog("scheduler", "Import Jadwal CSV",
		fmt.Sprintf("Dibuat: %d, dilewati: %d", created, skipped))

	msg := fmt.Sprintf("Import selesai: %d jadwal dibuat.", created)
	if skipped > 0 {
		msg += fmt.Sprintf(" %d baris dilewati.", skipped)
	}
	if created > 0 {
		h.hub.BroadcastToast("success", msg)
	} else {
		h.hub.BroadcastToast("error", msg+" Periksa format CSV Anda.")
	}
	if len(rowErrors) > 0 {
		maxDetail := 3
		if len(rowErrors) < maxDetail {
			maxDetail = len(rowErrors)
		}
		for i := 0; i < maxDetail; i++ {
			h.hub.BroadcastToast("warning", rowErrors[i])
		}
	}
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}

// normalizePhoneCSV menyamakan format nomor untuk pencocokan kontak
func normalizePhoneCSV(p string) string {
	p = strings.ReplaceAll(p, " ", "")
	p = strings.ReplaceAll(p, "-", "")
	p = strings.TrimPrefix(p, "+")
	if strings.HasPrefix(p, "0") && len(p) > 1 {
		p = "62" + p[1:]
	}
	return p
}

// isValidTimeCSV memvalidasi format HH:MM sederhana
func isValidTimeCSV(t string) bool {
	if len(t) != 5 || t[2] != ':' {
		return false
	}
	h, err1 := strconv.Atoi(t[:2])
	m, err2 := strconv.Atoi(t[3:])
	return err1 == nil && err2 == nil && h >= 0 && h < 24 && m >= 0 && m < 60
}

// ==========================================
// Templates Handlers
// ==========================================

func (h *Handlers) TemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates, _ := h.store.ListTemplates()

	var selected *store.Template
	selIDStr := r.URL.Query().Get("id")
	if selIDStr != "" {
		selID, _ := strconv.ParseInt(selIDStr, 10, 64)
		selected, _ = h.store.GetTemplateByID(selID)
	}
	if selected == nil && len(templates) > 0 {
		selected = &templates[0]
	}

	var versions []store.TemplateVersion
	previewText := ""
	if selected != nil {
		versions, _ = h.store.ListTemplateVersions(selected.ID)
		dummyCtx := h.tmplEngine.DummyContext()
		previewText, _ = h.tmplEngine.Render(selected.Content, dummyCtx)
	}

	data := map[string]interface{}{
		"Templates":   templates,
		"Selected":    selected,
		"Versions":    versions,
		"PreviewText": previewText,
	}

	h.renderer.Render(w, r, "pages/templates.html", PageData{
		Title:      "Template Pesan",
		ActivePage: "templates",
		Data:       data,
	})
}

func (h *Handlers) CreateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	userID := h.getActiveUserID(r)

	t := &store.Template{
		UserID:    sql.NullInt64{Int64: userID, Valid: true},
		Name:      strings.TrimSpace(r.FormValue("name")),
		Content:   strings.TrimSpace(r.FormValue("content")),
		IsDefault: r.FormValue("is_default") == "on" || r.FormValue("is_default") == "1",
	}

	if t.Name == "" || t.Content == "" {
		http.Redirect(w, r, "/templates", http.StatusSeeOther)
		return
	}

	created, err := h.store.CreateTemplate(t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.store.AddActivityLog("template", "Template Baru Dibuat", "Nama: "+t.Name)
	h.hub.BroadcastToast("success", fmt.Sprintf("Template %s berhasil dibuat!", t.Name))
	http.Redirect(w, r, fmt.Sprintf("/templates?id=%d", created.ID), http.StatusSeeOther)
}

func (h *Handlers) UpdateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	userID := h.getActiveUserID(r)

	existing, err := h.store.GetTemplateByID(id)
	if err != nil {
		h.hub.BroadcastToast("error", "Template tidak ditemukan.")
		http.Redirect(w, r, "/templates", http.StatusSeeOther)
		return
	}
	if existing.UserID.Valid && existing.UserID.Int64 != userID && !isPrivileged(h.currentUser(r)) {
		h.hub.BroadcastToast("error", "Akses ditolak: template ini bukan milik akun Anda.")
		http.Redirect(w, r, "/templates", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	t := &store.Template{
		ID:        id,
		UserID:    existing.UserID,
		Name:      strings.TrimSpace(r.FormValue("name")),
		Content:   strings.TrimSpace(r.FormValue("content")),
		IsDefault: r.FormValue("is_default") == "on" || r.FormValue("is_default") == "1",
	}

	if err := h.store.UpdateTemplate(t); err != nil {
		h.hub.BroadcastToast("error", "Gagal memperbarui template.")
		http.Redirect(w, r, fmt.Sprintf("/templates?id=%d", id), http.StatusSeeOther)
		return
	}
	h.store.AddActivityLog("template", "Template Diperbarui (Versi Baru Disimpan)", fmt.Sprintf("ID: %d, Nama: %s", id, t.Name))
	h.hub.BroadcastToast("success", "Template dan riwayat versi berhasil diperbarui!")
	http.Redirect(w, r, fmt.Sprintf("/templates?id=%d", id), http.StatusSeeOther)
}

func (h *Handlers) DeleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := h.getActiveUserID(r)
	if existing, err := h.store.GetTemplateByID(id); err == nil &&
		existing.UserID.Valid && existing.UserID.Int64 != userID && !isPrivileged(h.currentUser(r)) {
		h.hub.BroadcastToast("error", "Akses ditolak: template ini bukan milik akun Anda.")
		http.Redirect(w, r, "/templates", http.StatusSeeOther)
		return
	}

	_ = h.store.DeleteTemplate(id)
	h.store.AddActivityLog("template", "Template Dihapus", fmt.Sprintf("ID: %d", id))
	h.hub.BroadcastToast("info", "Template telah dihapus.")
	http.Redirect(w, r, "/templates", http.StatusSeeOther)
}

func (h *Handlers) PreviewTemplateHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	content := r.FormValue("content")

	dummyCtx := h.tmplEngine.DummyContext()
	rendered, err := h.tmplEngine.Render(content, dummyCtx)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="alert alert-error text-xs p-3"><strong>Format Error:</strong> %s</div>`, html.EscapeString(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="whitespace-pre-wrap font-sans text-sm leading-relaxed text-slate-200 bg-slate-900/90 p-4 rounded-xl border border-slate-800 shadow-inner">%s</div>`, html.EscapeString(rendered))
}

// ==========================================
// History Handlers
// ==========================================

func (h *Handlers) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	status := r.URL.Query().Get("status")
	messages, _ := h.store.ListQueueMessages(userID, status, 100)

	data := map[string]interface{}{
		"Messages":       messages,
		"SelectedStatus": status,
	}

	h.renderer.Render(w, r, "pages/history.html", PageData{
		Title:      "Riwayat Pengiriman",
		ActivePage: "history",
		Data:       data,
	})
}

// ==========================================
// Queue Handlers
// ==========================================

func (h *Handlers) QueueHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	pending, _ := h.store.ListQueueMessages(userID, "pending", 50)
	processing, _ := h.store.ListQueueMessages(userID, "processing", 10)

	data := map[string]interface{}{
		"Pending":    pending,
		"Processing": processing,
	}

	h.renderer.Render(w, r, "pages/queue.html", PageData{
		Title:      "Antrian Pesan",
		ActivePage: "queue",
		Data:       data,
	})
}

func (h *Handlers) QueueSendNowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	if msg, err := h.store.GetQueueMessageByID(id); err != nil || msg.UserID != h.getActiveUserID(r) {
		h.hub.BroadcastToast("error", "Pesan antrian tidak ditemukan pada akun Anda.")
		http.Redirect(w, r, "/queue", http.StatusSeeOther)
		return
	}

	_ = h.queueMgr.ProcessImmediate(id)
	h.hub.BroadcastToast("info", "Pesan sedang diproses untuk dikirim sekarang...")
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

func (h *Handlers) QueueCancelHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := h.getActiveUserID(r)
	if msg, err := h.store.GetQueueMessageByID(id); err != nil || msg.UserID != userID {
		h.hub.BroadcastToast("error", "Pesan antrian tidak ditemukan pada akun Anda.")
		http.Redirect(w, r, "/queue", http.StatusSeeOther)
		return
	}
	_ = h.store.CancelQueueMessage(id, userID)
	h.hub.BroadcastToast("info", "Pesan dalam antrian telah dibatalkan.")
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

func (h *Handlers) QueueDeleteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	userID := h.getActiveUserID(r)
	if msg, err := h.store.GetQueueMessageByID(id); err != nil || msg.UserID != userID {
		http.Redirect(w, r, "/queue", http.StatusSeeOther)
		return
	}
	_ = h.store.DeleteQueueMessage(id, userID)
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

// ==========================================
// Settings & Backup Handlers
// ==========================================

func (h *Handlers) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.store.GetSettings()
	holidays, _ := h.store.ListHolidays()
	users, _ := h.store.ListUsers()
	currentUser := auth.GetUserFromContext(r.Context())
	tunnelStatus := h.tunnelMgr.Status()

	data := map[string]interface{}{
		"Settings":    settings,
		"Holidays":    holidays,
		"Users":       users,
		"CurrentUser": currentUser,
		"Tunnel":      tunnelStatus,
	}

	h.renderer.Render(w, r, "pages/settings.html", PageData{
		Title:      "Pengaturan Sistem",
		ActivePage: "settings",
		Data:       data,
	})
}

func (h *Handlers) UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	rateMin, _ := strconv.Atoi(r.FormValue("rate_limit_min_sec"))
	rateMax, _ := strconv.Atoi(r.FormValue("rate_limit_max_sec"))
	retries, _ := strconv.Atoi(r.FormValue("max_retries"))

	st := &store.Settings{
		GlobalDryRun:     r.FormValue("global_dry_run") == "on" || r.FormValue("global_dry_run") == "1",
		RegistrationOpen: r.FormValue("registration_open") == "on" || r.FormValue("registration_open") == "1",
		SendWindowStart:  r.FormValue("send_window_start"),
		SendWindowEnd:    r.FormValue("send_window_end"),
		Timezone:         r.FormValue("timezone"),
		RateLimitMinSec:  rateMin,
		RateLimitMaxSec:  rateMax,
		MaxRetries:       retries,
	}

	_ = h.store.UpdateSettings(st)
	h.store.AddActivityLog("settings", "Pengaturan Sistem Diperbarui",
		fmt.Sprintf("DryRun: %v, Window: %s-%s, Timezone: %s", st.GlobalDryRun, st.SendWindowStart, st.SendWindowEnd, st.Timezone))

	h.hub.BroadcastToast("success", "Pengaturan sistem berhasil disimpan!")
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handlers) CreateHolidayHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	dateStr := strings.TrimSpace(r.FormValue("date"))
	desc := strings.TrimSpace(r.FormValue("description"))

	if dateStr != "" && desc != "" {
		_ = h.store.CreateHoliday(dateStr, desc)
		h.hub.BroadcastToast("success", "Hari libur berhasil ditambahkan.")
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handlers) DeleteHolidayHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_ = h.store.DeleteHoliday(id)
	h.hub.BroadcastToast("info", "Hari libur telah dihapus.")
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handlers) ToggleRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.GetSettings()
	if err == nil {
		_ = h.store.SetRegistrationOpen(!settings.RegistrationOpen)
		h.hub.BroadcastToast("info", "Status pendaftaran pengguna telah diperbarui.")
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handlers) BackupDownloadHandler(w http.ResponseWriter, r *http.Request) {
	dbFile, err := os.Open(h.cfg.DBPath)
	if err != nil {
		http.Error(w, "Gagal membaca database untuk backup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dbFile.Close()

	filename := fmt.Sprintf("sipen_backup_%s.db", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")

	_, _ = io.Copy(w, dbFile)
	h.store.AddActivityLog("system", "Download Backup Database", "File: "+filename)
}

// ==========================================
// Cloudflare Tunnel Handlers
// ==========================================

func (h *Handlers) ToggleTunnelHandler(w http.ResponseWriter, r *http.Request) {
	status := h.tunnelMgr.Status()
	if status.State == tunnel.StateActive {
		_ = h.tunnelMgr.Stop()
		h.hub.BroadcastToast("info", "Cloudflare Tunnel telah dimatikan.")
	} else {
		go func() {
			err := h.tunnelMgr.Start("")
			if err != nil {
				h.hub.BroadcastToast("error", "Gagal mengaktifkan Cloudflare Tunnel: "+err.Error())
			} else {
				h.hub.BroadcastToast("success", "Cloudflare Tunnel berhasil aktif!")
			}
		}()
		h.hub.BroadcastToast("info", "Memulai Cloudflare Quick Tunnel...")
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (h *Handlers) TunnelStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.tunnelMgr.Status())
}

// ==========================================
// Public Academic Schedule Portal & iCalendar ICS
// ==========================================

func (h *Handlers) PublicScheduleHandler(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.store.ListPublicSchedules()
	if err != nil {
		schedules = []store.ScheduleDetail{}
	}

	h.renderer.RenderPlain(w, "pages/schedule_public.html", PageData{
		Title: "Portal Jadwal Kuliah Publik — SiPenDosa",
		Data: map[string]interface{}{
			"Schedules": schedules,
		},
	})
}

func (h *Handlers) PublicCalendarICSHandler(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.store.ListPublicSchedules()
	if err != nil {
		http.Error(w, "Gagal mengambil jadwal publik", http.StatusInternalServerError)
		return
	}

	// Build RFC 5545 iCalendar stream
	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//SiPenDosa//Jadwal Kuliah Mahasiswa//ID\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:Jadwal Kuliah SiPenDosa\r\n")
	sb.WriteString("X-WR-TIMEZONE:Asia/Jakarta\r\n")

	dayMap := map[int]string{
		1: "MO", 2: "TU", 3: "WE", 4: "TH", 5: "FR", 6: "SA", 7: "SU",
	}

	nowStr := time.Now().UTC().Format("20060102T150405Z")

	for _, sc := range schedules {
		byDay, ok := dayMap[sc.DayOfWeek]
		if !ok {
			byDay = "MO"
		}

		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		diff := sc.DayOfWeek - weekday
		if diff < 0 {
			diff += 7
		}
		targetDate := now.AddDate(0, 0, diff)

		startParts := strings.Split(sc.StartTime, ":")
		endParts := strings.Split(sc.EndTime, ":")
		startH, startM := "08", "00"
		endH, endM := "10", "00"
		if len(startParts) >= 2 {
			startH, startM = startParts[0], startParts[1]
		}
		if len(endParts) >= 2 {
			endH, endM = endParts[0], endParts[1]
		}

		dtStart := fmt.Sprintf("%sT%s%s00", targetDate.Format("20060102"), startH, startM)
		dtEnd := fmt.Sprintf("%sT%s%s00", targetDate.Format("20060102"), endH, endM)

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:sipen-sched-%d@sipendosa\r\n", sc.ID))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", nowStr))
		sb.WriteString(fmt.Sprintf("DTSTART;TZID=Asia/Jakarta:%s\r\n", dtStart))
		sb.WriteString(fmt.Sprintf("DTEND;TZID=Asia/Jakarta:%s\r\n", dtEnd))
		sb.WriteString(fmt.Sprintf("RRULE:FREQ=WEEKLY;BYDAY=%s\r\n", byDay))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", sc.Matkul))
		if sc.DosenName != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:Dosen: %s\\nRuang: %s\\nMode: %s\r\n", sc.DosenName, sc.Location, sc.Mode))
		} else {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:Ruang: %s\\nMode: %s\r\n", sc.Location, sc.Mode))
		}
		if sc.Location != "" {
			sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", sc.Location))
		}
		sb.WriteString("STATUS:CONFIRMED\r\n")
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"jadwal_kuliah.ics\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sb.String()))
}

// ==========================================
// Logs Handlers
// ==========================================

func (h *Handlers) LogsHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logs, _ := h.store.ListActivityLogs(150, category)

	data := map[string]interface{}{
		"Logs":             logs,
		"SelectedCategory": category,
		"Timezone":         h.cfg.DefaultTimezone,
	}

	h.renderer.Render(w, r, "pages/logs.html", PageData{
		Title:      "Log & Observabilitas",
		ActivePage: "logs",
		Data:       data,
	})
}

// ==========================================
// WhatsApp Actions API (Scoped to User)
// ==========================================

func (h *Handlers) WhatsAppQRHandler(w http.ResponseWriter, r *http.Request) {
	waClient := h.getActiveWAClient(r)
	if waClient == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"state": "disconnected",
			"error": "WhatsApp client not initialized",
		})
		return
	}

	state, phone, pushName, qr := waClient.Status()
	if qr == "" && state != whatsapp.StateConnected {
		_ = waClient.Reconnect()
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			state, phone, pushName, qr = waClient.Status()
			if qr != "" || state == whatsapp.StateConnected {
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"qr":           qr,
		"state":        string(state),
		"phone":        phone,
		"push_name":    pushName,
		"pairing_code": waClient.GetPairingCode(),
	})
}

func (h *Handlers) WhatsAppReconnectHandler(w http.ResponseWriter, r *http.Request) {
	waClient := h.getActiveWAClient(r)
	if waClient != nil {
		err := waClient.Reconnect()
		if err != nil {
			h.hub.BroadcastToast("error", "Gagal menyambung ulang: "+err.Error())
		} else {
			h.hub.BroadcastToast("info", "Memulai proses rekoneksi WhatsApp...")
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) WhatsAppDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	waClient := h.getActiveWAClient(r)
	if waClient != nil {
		waClient.Disconnect()
		h.hub.BroadcastToast("info", "WhatsApp telah diputus koneksinya.")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) WhatsAppPairPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	phone := strings.TrimSpace(r.FormValue("phone"))
	if phone == "" {
		var body struct {
			Phone string `json:"phone"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		phone = strings.TrimSpace(body.Phone)
	}

	if phone == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Nomor WhatsApp wajib diisi (contoh: 08123456789 atau 628...)",
		})
		return
	}

	waClient := h.getActiveWAClient(r)
	if waClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "WhatsApp client not available",
		})
		return
	}

	code, err := waClient.PairPhone(r.Context(), phone)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	h.hub.BroadcastToast("success", fmt.Sprintf("Kode pairing WhatsApp: %s (masukkan di WhatsApp > Tautkan Perangkat)", code))
	h.store.AddActivityLog("whatsapp", "Request Pairing Code", fmt.Sprintf("Phone: %s, Code: %s", phone, code))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"code":    code,
		"message": "Kode pairing berhasil dibuat. Masukkan di WhatsApp > Perangkat Tertaut > Tautkan dengan nomor telepon.",
	})
}

func (h *Handlers) WhatsAppTestSendHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	target := strings.TrimSpace(r.FormValue("target_phone"))
	message := strings.TrimSpace(r.FormValue("message"))
	userID := h.getActiveUserID(r)

	if target == "" || message == "" {
		h.hub.BroadcastToast("error", "Nomor tujuan dan pesan tes tidak boleh kosong.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	qm := &store.QueueMessage{
		UserID:        userID,
		RecipientJID:  target,
		RecipientName: "Uji Coba WhatsApp",
		Message:       message,
		Status:        "pending",
		ScheduledFor:  time.Now(),
	}

	_, err := h.queueMgr.Enqueue(qm)
	if err != nil {
		h.hub.BroadcastToast("error", "Gagal memasukkan pesan uji coba ke antrian: "+err.Error())
	} else {
		h.hub.BroadcastToast("success", "Pesan uji coba telah masuk antrian dan akan segera dikirim!")
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// WhatsAppGroupsHandler returns WhatsApp groups the current account has joined (Issue #2)
func (h *Handlers) WhatsAppGroupsHandler(w http.ResponseWriter, r *http.Request) {
	waClient := h.getActiveWAClient(r)
	if waClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "WhatsApp client not available",
		})
		return
	}

	groups, err := waClient.GetJoinedGroups(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"groups":  groups,
	})
}

// WhatsAppContactsHandler returns contacts cached in the user's WhatsApp session (Issue #2)
func (h *Handlers) WhatsAppContactsHandler(w http.ResponseWriter, r *http.Request) {
	waClient := h.getActiveWAClient(r)
	if waClient == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "WhatsApp client not available",
		})
		return
	}

	contacts, err := waClient.GetStoredContacts(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"contacts": contacts,
	})
}

// ==========================================
// Health check
// ==========================================

func (h *Handlers) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	// Endpoint publik: jangan buat client WA dan jangan bocorkan nomor telepon.
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// ==========================================
// Error & Informational Pages
// ==========================================

func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.RenderWithStatus(w, r, "errors/404.html", PageData{
		Title:      "404 Halaman Tidak Ditemukan — SiPenDosa",
		ActivePage: "404",
	}, http.StatusNotFound)
}

func (h *Handlers) InternalServerErrorHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.RenderWithStatus(w, r, "errors/500.html", PageData{
		Title:      "500 Kesalahan Server — SiPenDosa",
		ActivePage: "500",
	}, http.StatusInternalServerError)
}

func (h *Handlers) TermsHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "pages/terms.html", PageData{
		Title:      "Ketentuan Layanan (ToS) — SiPenDosa",
		ActivePage: "terms",
	})
}

func (h *Handlers) PrivacyHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "pages/privacy.html", PageData{
		Title:      "Kebijakan Privasi — SiPenDosa",
		ActivePage: "privacy",
	})
}

func (h *Handlers) AboutHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "pages/about.html", PageData{
		Title:      "Tentang & Filosofi — SiPenDosa",
		ActivePage: "about",
	})
}

func (h *Handlers) ChangelogHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "pages/changelog.html", PageData{
		Title:      "Catatan Rilis & Changelog — SiPenDosa",
		ActivePage: "changelog",
	})
}

func (h *Handlers) TerminalHandler(w http.ResponseWriter, r *http.Request) {
	h.renderer.Render(w, r, "pages/terminal.html", PageData{
		Title:      "Terminal Android & Service — SiPenDosa",
		ActivePage: "terminal",
	})
}

// ==========================================
// SEO, PWA & Discovery Endpoints
// ==========================================

func (h *Handlers) RobotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	host := r.Host
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	content := fmt.Sprintf("User-agent: *\nAllow: /\nAllow: /jadwal\nAllow: /login\nAllow: /register\nAllow: /terms\nAllow: /privacy\nAllow: /about\nDisallow: /contacts/\nDisallow: /schedules/\nDisallow: /templates/\nDisallow: /queue/\nDisallow: /history/\nDisallow: /settings/\nDisallow: /logs/\n\nSitemap: %s://%s/sitemap.xml\n", scheme, host)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (h *Handlers) SitemapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	host := r.Host
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	now := time.Now().Format("2006-01-02")
	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>%s://%s/</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>%s://%s/jadwal</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>%s://%s/login</loc>
    <lastmod>%s</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>%s://%s/about</loc>
    <lastmod>%s</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>%s://%s/terms</loc>
    <lastmod>%s</lastmod>
    <changefreq>yearly</changefreq>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>%s://%s/privacy</loc>
    <lastmod>%s</lastmod>
    <changefreq>yearly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>`, scheme, host, now, scheme, host, now, scheme, host, now, scheme, host, now, scheme, host, now, scheme, host, now)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xmlContent))
}

func (h *Handlers) ManifestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, "web/static/manifest.json")
}

func (h *Handlers) FaviconHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, "web/static/favicon.ico")
}

// SystemVersionHandler returns current version and environment info
func (h *Handlers) SystemVersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"version":    version.CurrentVersion,
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"go_version": runtime.Version(),
	})
}

// CheckUpdateHandler checks GitHub releases for newer version
func (h *Handlers) CheckUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	force := r.URL.Query().Get("force") == "true"
	isTest := r.URL.Query().Get("test") == "true"

	res, err := h.updater.CheckUpdate(r.Context(), force)
	if err != nil {
		slog.Warn("Gagal memeriksa update GitHub", "err", err)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"current_version": "v" + version.CurrentVersion,
			"latest_version":  "v" + version.CurrentVersion,
			"has_update":      false,
			"error":           err.Error(),
		})
		return
	}

	if isTest {
		res.HasUpdate = true
		res.LatestVersion = "v1.2.0"
		res.ReleaseTitle = "SiPenDosa v1.2.0 — (Simulasi Uji Coba Pembaruan)"
		res.ReleaseNotes = "• Mode pengujian instalasi pembaruan otomatis\n• Peningkatan antarmuka dan kestabilan sistem"
	}

	_ = json.NewEncoder(w).Encode(res)
}

// DownloadApkHandler redirects directly to latest APK release download
func (h *Handlers) DownloadApkHandler(w http.ResponseWriter, r *http.Request) {
	res, err := h.updater.CheckUpdate(r.Context(), false)
	if err != nil || res.ApkURL == "" {
		fallbackURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/SiPenDosa-Android.apk", version.GitRepoOwner, version.GitRepoName)
		http.Redirect(w, r, fallbackURL, http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, res.ApkURL, http.StatusTemporaryRedirect)
}

// ApplyUpdateHandler performs hot self-update of the binary
func (h *Handlers) ApplyUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var req struct {
		DownloadURL string `json:"download_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.DownloadURL == "" {
		res, err := h.updater.CheckUpdate(r.Context(), false)
		if err == nil && res.DownloadURL != "" {
			req.DownloadURL = res.DownloadURL
		}
	}

	if req.DownloadURL == "" {
		http.Error(w, `{"error":"URL pembaruan tidak ditemukan"}`, http.StatusBadRequest)
		return
	}

	u, err := url.Parse(req.DownloadURL)
	if err != nil || u.Scheme != "https" || !isTrustedUpdateHost(u.Host) {
		http.Error(w, `{"error":"URL pembaruan tidak tepercaya (hanya GitHub releases resmi yang diizinkan)"}`, http.StatusBadRequest)
		return
	}

	go func(dlURL string) {
		time.Sleep(500 * time.Millisecond)
		err := h.updater.ApplyBinarySelfUpdate(context.Background(), dlURL)
		if err != nil {
			slog.Error("Gagal memasang self-update binary", "err", err)
		} else {
			slog.Info("Self-update binary berhasil dipasang")
		}
	}(req.DownloadURL)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pembaruan sedang diunduh dan dipasang di latar belakang...",
	})
}

// isTrustedUpdateHost membatasi sumber binary update ke GitHub resmi proyek
func isTrustedUpdateHost(host string) bool {
	switch host {
	case "github.com", "api.github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com":
		return true
	}
	return strings.HasSuffix(host, ".github.com") || strings.HasSuffix(host, ".githubusercontent.com")
}

// ==========================================
// Mobile & External REST JSON API Handlers
// ==========================================

// APITokenLoginHandler menukar username/password menjadi session bearer token
// untuk aplikasi companion mobile (Flutter/iOS/Android).
func (h *Handlers) APITokenLoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "username dan password wajib diisi"})
		return
	}

	user, err := h.authSvc.Authenticate(req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "username atau password salah"})
		return
	}

	token, err := h.authSvc.CreateSession(user.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "gagal membuat sesi"})
		return
	}

	h.store.AddActivityLog("auth", "Login API Companion", "User: "+user.Username)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":    token,
		"username": user.Username,
		"role":     user.Role,
	})
}

// APIStatsHandler provides JSON overview metrics for Flutter & iOS companion apps
func (h *Handlers) APIStatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	stats, err := h.store.GetDashboardStats(userID)
	if err != nil || stats == nil {
		stats = &store.DashboardStats{
			SuccessRate: 100.0,
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(stats)
}

// APIWhatsAppStatusHandler provides real-time WhatsApp connection status in JSON
func (h *Handlers) APIWhatsAppStatusHandler(w http.ResponseWriter, r *http.Request) {
	waClient := h.getActiveWAClient(r)
	var waState whatsapp.State = whatsapp.StateDisconnected
	var waPhone, waPushName, waQR string
	if waClient != nil {
		waState, waPhone, waPushName, waQR = waClient.Status()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"state":     string(waState),
		"phone":     waPhone,
		"push_name": waPushName,
		"qr":        waQR,
	})
}

// APIContactsHandler returns contacts list in JSON
func (h *Handlers) APIContactsHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	filterType := r.URL.Query().Get("type")
	contacts, err := h.store.ListContacts(userID, filterType)
	if err != nil || contacts == nil {
		contacts = []store.Contact{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(contacts)
}

// APISchedulesHandler returns schedules list in JSON
func (h *Handlers) APISchedulesHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	schedules, err := h.store.ListSchedules(userID)
	if err != nil || schedules == nil {
		schedules = []store.ScheduleDetail{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(schedules)
}

// APIQueueHandler returns queue messages list in JSON with pending counter
func (h *Handlers) APIQueueHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.getActiveUserID(r)
	status := r.URL.Query().Get("status")
	messages, err := h.store.ListQueueMessages(userID, status, 100)
	if err != nil || messages == nil {
		messages = []store.QueueMessage{}
	}

	pendingCount := 0
	for _, m := range messages {
		if m.Status == "pending" {
			pendingCount++
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"data":          messages,
		"pending_count": pendingCount,
	})
}

// APIQueueProcessHandler forces immediate sending of a queued message
func (h *Handlers) APIQueueProcessHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ID antrian tidak valid"})
		return
	}

	userID := h.getActiveUserID(r)
	if msg, err := h.store.GetQueueMessageByID(id); err != nil || msg.UserID != userID {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Pesan antrian tidak ditemukan pada akun Anda"})
		return
	}

	if err := h.queueMgr.ProcessImmediate(id); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Pesan antrian sedang diproses untuk dikirim langsung",
		"id":      id,
	})
}
