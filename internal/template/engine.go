package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"sipen/internal/store"
)

var indonesianDays = []string{
	"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu",
}

var indonesianMonths = []string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// Context contains all variables available in templates
type Context struct {
	NamaDosen       string
	NamaMahasiswa   string
	NIM             string
	Matkul          string
	Hari            string
	Tanggal         string
	JamMulai        string
	JamSelesai      string
	Lokasi          string
	LinkGroup       string
	WaktuSekarang   string
	HariDalamBahasa string
}

// Engine handles parsing and executing templates
type Engine struct {
	funcMap template.FuncMap
}

// NewEngine creates a new template engine with rich helper functions
func NewEngine() *Engine {
	return &Engine{
		funcMap: template.FuncMap{
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
			"title": strings.Title,
			"trim":  strings.TrimSpace,
			"formatDate": func(t time.Time, layout string) string {
				return t.Format(layout)
			},
			"default": func(def, val string) string {
				if strings.TrimSpace(val) == "" {
					return def
				}
				return val
			},
		},
	}
}

// Render compiles and executes a template string with the provided context
func (e *Engine) Render(tmplStr string, ctx Context) (string, error) {
	tmpl, err := template.New("sipen_tmpl").Funcs(e.funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("gagal memvalidasi template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("gagal me-render template: %w", err)
	}

	return buf.String(), nil
}

// BuildContext creates a Context object for a specific schedule and lecture date
func (e *Engine) BuildContext(sc *store.ScheduleDetail, lectureDate time.Time, studentContact *store.Contact) Context {
	dayName := DayName(lectureDate.Weekday())
	dateFormatted := FormatIndonesianDate(lectureDate)

	namaMahasiswa := ""
	nim := ""
	if studentContact != nil {
		namaMahasiswa = studentContact.Name
		nim = studentContact.NIM
	} else if sc.RecipientType == "komti" || sc.RecipientType == "mahasiswa" {
		namaMahasiswa = sc.RecipientName
	}

	namaDosen := sc.DosenName
	if namaDosen == "" {
		namaDosen = "Bapak/Ibu Dosen Pengampu"
	}

	return Context{
		NamaDosen:       namaDosen,
		NamaMahasiswa:   namaMahasiswa,
		NIM:             nim,
		Matkul:          sc.Matkul,
		Hari:            dayName,
		Tanggal:         dateFormatted,
		JamMulai:        sc.StartTime,
		JamSelesai:      sc.EndTime,
		Lokasi:          sc.Location,
		LinkGroup:       sc.LinkGroup,
		WaktuSekarang:   time.Now().Format("15:04:05 WITA"),
		HariDalamBahasa: dayName,
	}
}

// DummyContext returns a realistic preview context with sample Indonesian university data
func (e *Engine) DummyContext() Context {
	tomorrow := time.Now().Add(24 * time.Hour)
	dayName := DayName(tomorrow.Weekday())

	return Context{
		NamaDosen:       "Dr. Eng. Ir. Hendra Gunawan, S.T., M.T.",
		NamaMahasiswa:   "Muhammad Fajar Siddiq",
		NIM:             "210211048",
		Matkul:          "Sistem Operasi & Jaringan Terdistribusi",
		Hari:            dayName,
		Tanggal:         FormatIndonesianDate(tomorrow),
		JamMulai:        "08:00",
		JamSelesai:      "10:30",
		Lokasi:          "Gedung Kuliah Bersama R.304 / Lab Cloud",
		LinkGroup:       "https://chat.whatsapp.com/GXYZ1234567890",
		WaktuSekarang:   time.Now().Format("15:04:05 WITA"),
		HariDalamBahasa: dayName,
	}
}

// DayName returns the Indonesian day name for a given weekday
func DayName(w time.Weekday) string {
	idx := int(w)
	if idx >= 0 && idx < len(indonesianDays) {
		return indonesianDays[idx]
	}
	return "Senin"
}

// FormatIndonesianDate formats a date like "15 Oktober 2026"
func FormatIndonesianDate(t time.Time) string {
	day := t.Day()
	month := indonesianMonths[int(t.Month())]
	year := t.Year()
	return fmt.Sprintf("%d %s %d", day, month, year)
}
