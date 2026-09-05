package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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
	"sipen/internal/whatsapp"

	"github.com/go-chi/chi/v5"
)

// Handlers bundles all web application controller dependencies
type Handlers struct {
	cfg        *config.Config
	store      *store.Store
	authSvc    *auth.Service
	waClient   *whatsapp.Client
	queueMgr   *queue.Manager
	scheduler  *scheduler.Scheduler
	tmplEngine *template.Engine
	hub        *realtime.Hub
	renderer   *ViewRenderer
}

// NewHandlers creates an instance of application HTTP handlers
func NewHandlers(
	cfg *config.Config,
	s *store.Store,
	a *auth.Service,
	wa *whatsapp.Client,
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
		waClient:   wa,
		queueMgr:   q,
		scheduler:  sch,
		tmplEngine: t,
		hub:        h,
		renderer:   r,
	}
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

	token, err := h.authSvc.CreateSession(user.ID)
	if err != nil {
		h.renderer.RenderPlain(w, "auth/login.html", PageData{
			Title:      "Masuk — SiPenDosa",
			FlashError: "Gagal membuat sesi login.",
		})
		return
	}

	auth.SetSessionCookie(w, token)
	h.store.AddActivityLog("auth", "User Login Berhasil", "Username: "+user.Username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	count, _ := h.store.CountUsers()
	isFirstUser := count == 0

	if !isFirstUser {
		settings, err := h.store.GetSettings()
		if err != nil || !settings.RegistrationOpen {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
	}

	h.renderer.RenderPlain(w, "auth/register.html", PageData{
		Title:          "Pendaftaran — SiPenDosa",
		IsRegistration: isFirstUser,
	})
}

func (h *Handlers) RegisterPostHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	count, _ := h.store.CountUsers()
	isFirstUser := count == 0

	if username == "" || len(password) < 6 {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:          "Pendaftaran — SiPenDosa",
			IsRegistration: isFirstUser,
			FlashError:     "Username wajib diisi dan kata sandi minimal 6 karakter.",
		})
		return
	}

	if password != confirmPassword {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:          "Pendaftaran — SiPenDosa",
			IsRegistration: isFirstUser,
			FlashError:     "Konfirmasi kata sandi tidak cocok.",
		})
		return
	}

	var err error
	if isFirstUser {
		_, err = h.authSvc.RegisterFirstUser(username, password)
	} else {
		_, err = h.authSvc.RegisterUser(username, password, "admin", false)
	}

	if err != nil {
		h.renderer.RenderPlain(w, "auth/register.html", PageData{
			Title:          "Pendaftaran — SiPenDosa",
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
// Overview Dashboard
// ==========================================

func (h *Handlers) OverviewHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetDashboardStats()
	if err != nil {
		slog.Error("Failed getting dashboard stats", "err", err)
	}

	waState, waPhone, waPushName, waQR := h.waClient.Status()
	recentMessages, _ := h.store.ListQueueMessages("", 6)
	nextInfo := h.scheduler.GetNextDeliveryInfo()
	settings, _ := h.store.GetSettings()

	data := map[string]interface{}{
		"Stats":          stats,
		"WAState":        string(waState),
		"WAPhone":        waPhone,
		"WAPushName":     waPushName,
		"WAQR":           waQR,
		"RecentMessages": recentMessages,
		"NextDelivery":   nextInfo,
		"Settings":       settings,
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
	filterType := r.URL.Query().Get("type")
	contacts, err := h.store.ListContacts(filterType)
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
	c := &store.Contact{
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

	_ = r.ParseForm()
	c := &store.Contact{
		ID:          id,
		Name:        strings.TrimSpace(r.FormValue("name")),
		Phone:       strings.TrimSpace(r.FormValue("phone")),
		ContactType: r.FormValue("contact_type"),
		NIM:         strings.TrimSpace(r.FormValue("nim")),
		Email:       strings.TrimSpace(r.FormValue("email")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
	}

	_ = h.store.UpdateContact(c)
	h.store.AddActivityLog("contacts", "Kontak Diperbarui", fmt.Sprintf("ID: %d, Nama: %s", id, c.Name))
	h.hub.BroadcastToast("success", "Kontak berhasil diperbarui!")
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

func (h *Handlers) DeleteContactHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_ = h.store.DeleteContact(id)
	h.store.AddActivityLog("contacts", "Kontak Dihapus", fmt.Sprintf("ID: %d", id))
	h.hub.BroadcastToast("info", "Kontak telah dihapus.")
	http.Redirect(w, r, "/contacts", http.StatusSeeOther)
}

// ==========================================
// Schedules Handlers
// ==========================================

func (h *Handlers) SchedulesHandler(w http.ResponseWriter, r *http.Request) {
	schedules, _ := h.store.ListSchedules()
	contacts, _ := h.store.ListContacts("")
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

	dayOfWeek, _ := strconv.Atoi(r.FormValue("day_of_week"))
	dosenID, _ := strconv.ParseInt(r.FormValue("dosen_id"), 10, 64)
	recipientID, _ := strconv.ParseInt(r.FormValue("recipient_id"), 10, 64)
	templateID, _ := strconv.ParseInt(r.FormValue("template_id"), 10, 64)

	sc := &store.Schedule{
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
		DryRun:      r.FormValue("dry_run") == "on" || r.FormValue("dry_run") == "1",
	}

	if dosenID > 0 {
		sc.DosenID = sql.NullInt64{Int64: dosenID, Valid: true}
	}
	if recipientID > 0 {
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

	_ = r.ParseForm()
	dayOfWeek, _ := strconv.Atoi(r.FormValue("day_of_week"))
	dosenID, _ := strconv.ParseInt(r.FormValue("dosen_id"), 10, 64)
	recipientID, _ := strconv.ParseInt(r.FormValue("recipient_id"), 10, 64)
	templateID, _ := strconv.ParseInt(r.FormValue("template_id"), 10, 64)

	sc := &store.Schedule{
		ID:          id,
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
		DryRun:      r.FormValue("dry_run") == "on" || r.FormValue("dry_run") == "1",
	}

	if dosenID > 0 {
		sc.DosenID = sql.NullInt64{Int64: dosenID, Valid: true}
	}
	if recipientID > 0 {
		sc.RecipientID = sql.NullInt64{Int64: recipientID, Valid: true}
	}
	if templateID > 0 {
		sc.TemplateID = sql.NullInt64{Int64: templateID, Valid: true}
	}

	_ = h.store.UpdateSchedule(sc)
	h.store.AddActivityLog("scheduler", "Jadwal Diperbarui", fmt.Sprintf("ID: %d, Matkul: %s", id, sc.Matkul))
	h.hub.BroadcastToast("success", "Jadwal berhasil diperbarui!")
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}

func (h *Handlers) ToggleScheduleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	sc, err := h.store.GetScheduleByID(id)
	if err == nil {
		_ = h.store.ToggleScheduleActive(id, !sc.IsActive)
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

	_ = h.store.DeleteSchedule(id)
	h.store.AddActivityLog("scheduler", "Jadwal Dihapus", fmt.Sprintf("ID: %d", id))
	h.hub.BroadcastToast("info", "Jadwal telah dihapus.")
	http.Redirect(w, r, "/schedules", http.StatusSeeOther)
}

// ==========================================
// Templates Handlers
// ==========================================

func (h *Handlers) TemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templates, _ := h.store.ListTemplates()

	// Select first template or requested template
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
	t := &store.Template{
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

	_ = r.ParseForm()
	t := &store.Template{
		ID:        id,
		Name:      strings.TrimSpace(r.FormValue("name")),
		Content:   strings.TrimSpace(r.FormValue("content")),
		IsDefault: r.FormValue("is_default") == "on" || r.FormValue("is_default") == "1",
	}

	_ = h.store.UpdateTemplate(t)
	h.store.AddActivityLog("template", "Template Diperbarui (Versi Baru Disimpan)", fmt.Sprintf("ID: %d, Nama: %s", id, t.Name))
	h.hub.BroadcastToast("success", "Template dan riwayat versi berhasil diperbarui!")
	http.Redirect(w, r, fmt.Sprintf("/templates?id=%d", id), http.StatusSeeOther)
}

func (h *Handlers) DeleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

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
		fmt.Fprintf(w, `<div class="alert alert-error text-xs p-3"><strong>Format Error:</strong> %s</div>`, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="whitespace-pre-wrap font-sans text-sm leading-relaxed text-slate-200 bg-slate-900/90 p-4 rounded-xl border border-slate-800 shadow-inner">%s</div>`, rendered)
}

// ==========================================
// History Handlers
// ==========================================

func (h *Handlers) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	messages, _ := h.store.ListQueueMessages(status, 100)

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
	pending, _ := h.store.ListQueueMessages("pending", 50)
	processing, _ := h.store.ListQueueMessages("processing", 10)

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

	_ = h.queueMgr.ProcessImmediate(id)
	h.hub.BroadcastToast("info", "Pesan sedang diproses untuk dikirim sekarang...")
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

func (h *Handlers) QueueCancelHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_ = h.store.CancelQueueMessage(id)
	h.hub.BroadcastToast("info", "Pesan dalam antrian telah dibatalkan.")
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

func (h *Handlers) QueueDeleteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_ = h.store.DeleteQueueMessage(id)
	http.Redirect(w, r, "/queue", http.StatusSeeOther)
}

// ==========================================
// Settings & Backup Handlers
// ==========================================

func (h *Handlers) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings, _ := h.store.GetSettings()
	holidays, _ := h.store.ListHolidays()
	users, _ := h.store.ListUsers()

	data := map[string]interface{}{
		"Settings": settings,
		"Holidays": holidays,
		"Users":    users,
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

func (h *Handlers) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser != nil && currentUser.ID == id {
		h.hub.BroadcastToast("error", "Anda tidak dapat menghapus akun Anda sendiri.")
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	_ = h.store.DeleteUser(id)
	h.hub.BroadcastToast("info", "Pengguna telah dihapus.")
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
// WhatsApp Actions API
// ==========================================

func (h *Handlers) WhatsAppQRHandler(w http.ResponseWriter, r *http.Request) {
	_, _, _, qr := h.waClient.Status()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"qr": qr,
	})
}

func (h *Handlers) WhatsAppReconnectHandler(w http.ResponseWriter, r *http.Request) {
	err := h.waClient.Reconnect()
	if err != nil {
		h.hub.BroadcastToast("error", "Gagal menyambung ulang: "+err.Error())
	} else {
		h.hub.BroadcastToast("info", "Memulai proses rekoneksi WhatsApp...")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) WhatsAppDisconnectHandler(w http.ResponseWriter, r *http.Request) {
	h.waClient.Disconnect()
	h.hub.BroadcastToast("info", "WhatsApp telah diputus koneksinya.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) WhatsAppTestSendHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	target := strings.TrimSpace(r.FormValue("target_phone"))
	message := strings.TrimSpace(r.FormValue("message"))

	if target == "" || message == "" {
		h.hub.BroadcastToast("error", "Nomor tujuan dan pesan tes tidak boleh kosong.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	qm := &store.QueueMessage{
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

// ==========================================
// Health check
// ==========================================

func (h *Handlers) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	state, phone, _, _ := h.waClient.Status()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "ok",
		"timestamp":       time.Now().Unix(),
		"whatsapp_status": string(state),
		"whatsapp_phone":  phone,
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
	content := fmt.Sprintf("User-agent: *\nAllow: /\nAllow: /login\nAllow: /register\nAllow: /terms\nAllow: /privacy\nAllow: /about\nDisallow: /contacts/\nDisallow: /schedules/\nDisallow: /templates/\nDisallow: /queue/\nDisallow: /history/\nDisallow: /settings/\nDisallow: /logs/\n\nSitemap: %s://%s/sitemap.xml\n", scheme, host)
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
</urlset>`, scheme, host, now, scheme, host, now, scheme, host, now, scheme, host, now, scheme, host, now)
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
