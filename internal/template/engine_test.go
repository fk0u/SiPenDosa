package template

import (
	"strings"
	"testing"
	"time"

	"sipen/internal/store"
)

func TestEngineRender(t *testing.T) {
	engine := NewEngine()

	tmplStr := "Halo Yth. {{.NamaDosen}}, besok ada kuliah {{.Matkul}} pada hari {{.Hari}}, {{.Tanggal}} pukul {{.JamMulai}} di {{.Lokasi}}."
	ctx := Context{
		NamaDosen: "Dr. Budi Santoso",
		Matkul:    "Algoritma & Struktur Data",
		Hari:      "Senin",
		Tanggal:   "15 Oktober 2026",
		JamMulai:  "08:00",
		Lokasi:    "Lab Komputer 3",
	}

	result, err := engine.Render(tmplStr, ctx)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	expectedParts := []string{
		"Dr. Budi Santoso",
		"Algoritma & Struktur Data",
		"Senin",
		"15 Oktober 2026",
		"08:00",
		"Lab Komputer 3",
	}

	for _, p := range expectedParts {
		if !strings.Contains(result, p) {
			t.Errorf("Expected output to contain %q, got %q", p, result)
		}
	}
}

func TestIndonesianDates(t *testing.T) {
	sampleDate := time.Date(2026, time.August, 17, 10, 0, 0, 0, time.UTC)
	day := DayName(sampleDate.Weekday())
	formatted := FormatIndonesianDate(sampleDate)

	if day != "Senin" {
		t.Errorf("Expected 'Senin', got %s", day)
	}

	if formatted != "17 Agustus 2026" {
		t.Errorf("Expected '17 Agustus 2026', got %s", formatted)
	}
}

func TestBuildContext(t *testing.T) {
	engine := NewEngine()

	sc := &store.ScheduleDetail{
		Schedule: store.Schedule{
			Matkul:    "Pemrograman Web",
			StartTime: "08:00",
			EndTime:   "09:40",
			Location:  "R.302",
		},
		DosenName: "Prof. Ir. Hendra",
	}

	lectureDate := time.Date(2026, time.September, 7, 8, 0, 0, 0, time.UTC)
	ctx := engine.BuildContext(sc, lectureDate, nil)

	if ctx.NamaDosen != "Prof. Ir. Hendra" {
		t.Errorf("Unexpected lecturer name: %s", ctx.NamaDosen)
	}
	if ctx.Matkul != "Pemrograman Web" {
		t.Errorf("Unexpected course name: %s", ctx.Matkul)
	}
}
