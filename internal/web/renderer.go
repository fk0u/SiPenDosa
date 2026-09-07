package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"sipen/internal/auth"
	"sipen/internal/store"
)

// ViewRenderer manages HTML template parsing and execution
type ViewRenderer struct {
	templateFS fs.FS
	diskDir    string
	funcMap    template.FuncMap
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

// NewViewRenderer creates a new view renderer supporting embedded fs and optional disk fallback
func NewViewRenderer(templateFS fs.FS, diskDir string) *ViewRenderer {
	return &ViewRenderer{
		templateFS: templateFS,
		diskDir:    diskDir,
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
			"statVal": func(stats interface{}, field string) interface{} {
				if stats == nil {
					return 0
				}
				if ds, ok := stats.(*store.DashboardStats); ok && ds != nil {
					switch field {
					case "ActiveSchedules":
						if ds.ActiveSchedules > 0 {
							return ds.ActiveSchedules
						}
						return ds.TotalSchedules
					case "TotalSchedules":
						return ds.TotalSchedules
					case "TotalContacts":
						return ds.TotalContacts
					case "TotalSentToday":
						return ds.TotalSentToday
					case "TotalSentAll":
						return ds.TotalSentAll
					case "TotalPending":
						return ds.TotalPending
					case "TotalFailedAll":
						return ds.TotalFailedAll
					case "SuccessRate":
						return ds.SuccessRate
					}
				}
				if m, ok := stats.(map[string]interface{}); ok && m != nil {
					if v, exists := m[field]; exists && v != nil {
						return v
					}
					// Check lowercase / snake_case alternative
					snake := strings.ToLower(field)
					if v, exists := m[snake]; exists && v != nil {
						return v
					}
				}
				return 0
			},
		},
	}
}

// Render renders a full page embedded within the base layout with 200 OK
func (v *ViewRenderer) Render(w http.ResponseWriter, r *http.Request, pageTemplate string, data PageData) {
	v.RenderWithStatus(w, r, pageTemplate, data, http.StatusOK)
}

// RenderWithStatus renders a full page embedded within the base layout with an explicit HTTP status code
func (v *ViewRenderer) RenderWithStatus(w http.ResponseWriter, r *http.Request, pageTemplate string, data PageData, statusCode int) {
	if data.User == nil {
		data.User = auth.GetUserFromContext(r.Context())
	}
	data.CurrentYear = time.Now().Year()

	var tmpl *template.Template
	var err error

	if v.diskDir != "" {
		diskLayout := filepath.Join(v.diskDir, "layouts", "base.html")
		diskPage := filepath.Join(v.diskDir, pageTemplate)
		if _, statErr := os.Stat(diskLayout); statErr == nil {
			if _, statErr := os.Stat(diskPage); statErr == nil {
				tmpl, err = template.New("base.html").Funcs(v.funcMap).ParseFiles(diskLayout, diskPage)
			}
		}
	}

	if tmpl == nil && v.templateFS != nil {
		layoutPath := "layouts/base.html"
		pagePath := filepath.ToSlash(pageTemplate)
		tmpl, err = template.New("base.html").Funcs(v.funcMap).ParseFS(v.templateFS, layoutPath, pagePath)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
		return
	}
	if tmpl == nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		http.Error(w, fmt.Sprintf("Template render error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = io.Copy(w, &buf)
}

// RenderPlain renders a standalone page without the main dashboard sidebar (e.g. login, register)
func (v *ViewRenderer) RenderPlain(w http.ResponseWriter, pageTemplate string, data PageData) {
	data.CurrentYear = time.Now().Year()
	baseName := path.Base(filepath.ToSlash(pageTemplate))

	var tmpl *template.Template
	var err error

	if v.diskDir != "" {
		diskFile := filepath.Join(v.diskDir, pageTemplate)
		if _, statErr := os.Stat(diskFile); statErr == nil {
			tmpl, err = template.New(baseName).Funcs(v.funcMap).ParseFiles(diskFile)
		}
	}

	if tmpl == nil && v.templateFS != nil {
		pagePath := filepath.ToSlash(pageTemplate)
		tmpl, err = template.New(baseName).Funcs(v.funcMap).ParseFS(v.templateFS, pagePath)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
		return
	}
	if tmpl == nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, baseName, data); err != nil {
		if err2 := tmpl.Execute(&buf, data); err2 != nil {
			http.Error(w, fmt.Sprintf("Template render error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &buf)
}

// RenderPartial renders an HTML snippet (for HTMX partial responses)
func (v *ViewRenderer) RenderPartial(w http.ResponseWriter, partialTemplate string, data interface{}) {
	baseName := path.Base(filepath.ToSlash(partialTemplate))
	var tmpl *template.Template
	var err error

	if v.diskDir != "" {
		diskFile := filepath.Join(v.diskDir, partialTemplate)
		if _, statErr := os.Stat(diskFile); statErr == nil {
			tmpl, err = template.New(baseName).Funcs(v.funcMap).ParseFiles(diskFile)
		}
	}

	if tmpl == nil && v.templateFS != nil {
		pagePath := filepath.ToSlash(partialTemplate)
		tmpl, err = template.New(baseName).Funcs(v.funcMap).ParseFS(v.templateFS, pagePath)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Partial parse error: %v", err), http.StatusInternalServerError)
		return
	}
	if tmpl == nil {
		http.Error(w, "Partial template not found", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, baseName, data); err != nil {
		if err2 := tmpl.Execute(&buf, data); err2 != nil {
			http.Error(w, fmt.Sprintf("Partial render error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &buf)
}
