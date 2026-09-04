package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"sipen/internal/auth"
	"sipen/internal/store"
)

// ViewRenderer manages HTML template parsing and execution
type ViewRenderer struct {
	templatesDir string
	funcMap      template.FuncMap
}

// PageData contains the standard context passed into HTML templates
type PageData struct {
	Title           string
	ActivePage      string
	User            *store.User
	Data            interface{}
	FlashSuccess    string
	FlashError      string
	IsRegistration  bool
	CurrentYear     int
}

// NewViewRenderer creates a new view renderer
func NewViewRenderer(templatesDir string) *ViewRenderer {
	return &ViewRenderer{
		templatesDir: templatesDir,
		funcMap: template.FuncMap{
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
			"title": strings.Title,
			"trim":  strings.TrimSpace,
			"formatTime": func(t *time.Time, layout string) string {
				if t == nil {
					return "-"
				}
				return t.Format(layout)
			},
			"formatDateIndo": func(t time.Time) string {
				months := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
				return fmt.Sprintf("%d %s %d", t.Day(), months[int(t.Month())], t.Year())
			},
			"dayName": func(day int) string {
				days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
				if day >= 0 && day < len(days) {
					return days[day]
				}
				return "-"
			},
			"safeHTML": func(s string) template.HTML {
				return template.HTML(s)
			},
			"truncate": func(s string, maxLen int) string {
				if len(s) > maxLen {
					return s[:maxLen] + "..."
				}
				return s
			},
			"statusBadge": func(status string) template.HTML {
				switch status {
				case "sent":
					return `<span class="badge badge-success badge-sm gap-1">Terkirim</span>`
				case "pending":
					return `<span class="badge badge-warning badge-sm gap-1">Menunggu</span>`
				case "processing":
					return `<span class="badge badge-info badge-sm gap-1 animate-pulse">Memproses</span>`
				case "failed":
					return `<span class="badge badge-error badge-sm gap-1">Gagal</span>`
				case "cancelled":
					return `<span class="badge badge-neutral badge-sm gap-1">Dibatalkan</span>`
				case "dry_run":
					return `<span class="badge badge-secondary badge-sm gap-1">Dry Run</span>`
				default:
					return template.HTML(fmt.Sprintf(`<span class="badge badge-ghost badge-sm">%s</span>`, status))
				}
			},
		},
	}
}

// Render renders a full page embedded within the base layout
func (v *ViewRenderer) Render(w http.ResponseWriter, r *http.Request, pageTemplate string, data PageData) {
	if data.User == nil {
		data.User = auth.GetUserFromContext(r.Context())
	}
	data.CurrentYear = time.Now().Year()

	layoutPath := filepath.Join(v.templatesDir, "layouts", "base.html")
	pagePath := filepath.Join(v.templatesDir, pageTemplate)

	tmpl, err := template.New("base.html").Funcs(v.funcMap).ParseFiles(layoutPath, pagePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, fmt.Sprintf("Template render error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &buf)
}

// RenderPlain renders a standalone page without the main dashboard sidebar (e.g. login, register)
func (v *ViewRenderer) RenderPlain(w http.ResponseWriter, pageTemplate string, data PageData) {
	data.CurrentYear = time.Now().Year()
	pagePath := filepath.Join(v.templatesDir, pageTemplate)

	tmpl, err := template.New(filepath.Base(pageTemplate)).Funcs(v.funcMap).ParseFiles(pagePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		http.Error(w, fmt.Sprintf("Template render error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &buf)
}

// RenderPartial renders an HTML snippet (for HTMX partial responses)
func (v *ViewRenderer) RenderPartial(w http.ResponseWriter, partialTemplate string, data interface{}) {
	tmplPath := filepath.Join(v.templatesDir, partialTemplate)
	tmpl, err := template.New(filepath.Base(partialTemplate)).Funcs(v.funcMap).ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Partial parse error: %v", err), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		http.Error(w, fmt.Sprintf("Partial render error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &buf)
}
