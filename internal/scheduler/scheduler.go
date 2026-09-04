package scheduler

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"sipen/internal/queue"
	"sipen/internal/realtime"
	"sipen/internal/store"
	"sipen/internal/template"
)

// NextDeliveryInfo holds information about the upcoming scheduled message
type NextDeliveryInfo struct {
	ScheduleID     int64  `json:"schedule_id"`
	Matkul         string `json:"matkul"`
	RecipientName  string `json:"recipient_name"`
	TargetTime     string `json:"target_time"`
	RemainingSecs  int64  `json:"remaining_seconds"`
	RemainingHuman string `json:"remaining_human"`
	MessagePreview string `json:"message_preview"`
	IsDryRun       bool   `json:"is_dry_run"`
}

// Scheduler handles periodic schedule checking, countdown calculation, and holiday checks
type Scheduler struct {
	store       *store.Store
	queueMgr    *queue.Manager
	tmplEngine  *template.Engine
	hub         *realtime.Hub
	stopChan    chan struct{}
	mu          sync.RWMutex
	running     bool
	nextInfo    *NextDeliveryInfo
}

// NewScheduler creates a new scheduler instance
func NewScheduler(s *store.Store, q *queue.Manager, t *template.Engine, hub *realtime.Hub) *Scheduler {
	return &Scheduler{
		store:      s,
		queueMgr:   q,
		tmplEngine: t,
		hub:        hub,
		stopChan:   make(chan struct{}),
	}
}

// Start runs the minute evaluation loop and countdown broadcaster
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.loop()
}

// Stop halts the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopChan)
}

func (s *Scheduler) loop() {
	// Immediate initial calculation
	s.evaluateTick()
	s.updateCountdown()

	tickerMinute := time.NewTicker(time.Minute)
	tickerCountdown := time.NewTicker(15 * time.Second)
	defer tickerMinute.Stop()
	defer tickerCountdown.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-tickerMinute.C:
			s.evaluateTick()
			s.updateCountdown()
		case <-tickerCountdown.C:
			s.updateCountdown()
		}
	}
}

func (s *Scheduler) evaluateTick() {
	settings, err := s.store.GetSettings()
	if err != nil {
		slog.Error("Scheduler failed to load settings", "err", err)
		return
	}

	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		loc = time.FixedZone("WITA", 8*3600)
	}

	now := time.Now().In(loc)
	currentTimeStr := now.Format("15:04")
	todayDateStr := now.Format("2006-01-02")

	// Check if current time is within sending window (e.g. 08:00 - 16:00)
	if currentTimeStr < settings.SendWindowStart || currentTimeStr > settings.SendWindowEnd {
		return
	}

	schedules, err := s.store.ListActiveSchedules()
	if err != nil {
		slog.Error("Scheduler failed to list active schedules", "err", err)
		return
	}

	for _, sc := range schedules {
		if sc.SendAtTime != currentTimeStr {
			continue
		}

		// Check if already sent today
		if sc.LastSentAt != nil && sc.LastSentAt.In(loc).Format("2006-01-02") == todayDateStr {
			continue
		}

		var targetLectureDate time.Time
		var isEligibleDay bool

		if sc.Mode == "H-1" {
			tomorrow := now.AddDate(0, 0, 1)
			if int(tomorrow.Weekday()) == sc.DayOfWeek {
				targetLectureDate = tomorrow
				isEligibleDay = true
			}
		} else { // H-0
			if int(now.Weekday()) == sc.DayOfWeek {
				targetLectureDate = now
				isEligibleDay = true
			}
		}

		if !isEligibleDay {
			continue
		}

		// Holiday verification
		lectureDateStr := targetLectureDate.Format("2006-01-02")
		isHoliday, holidayDesc, _ := s.store.IsHoliday(lectureDateStr)
		if isHoliday {
			slog.Info("Skipping reminder because lecture date is a holiday", "matkul", sc.Matkul, "holiday", holidayDesc)
			s.store.AddActivityLog("scheduler", "Jadwal Dilewati (Hari Libur)",
				fmt.Sprintf("Matkul: %s, Tanggal: %s, Libur: %s", sc.Matkul, lectureDateStr, holidayDesc))
			continue
		}

		// Trigger and enqueue
		s.fireSchedule(sc, targetLectureDate, now)
	}
}

func (s *Scheduler) fireSchedule(sc store.ScheduleDetail, lectureDate time.Time, now time.Time) {
	// Resolve template
	tmplContent := ""
	if sc.TemplateID.Valid {
		t, err := s.store.GetTemplateByID(sc.TemplateID.Int64)
		if err == nil {
			tmplContent = t.Content
		}
	}
	if tmplContent == "" {
		def, err := s.store.GetDefaultTemplate()
		if err == nil {
			tmplContent = def.Content
		}
	}
	if tmplContent == "" {
		slog.Error("No template available for schedule", "schedule_id", sc.ID)
		return
	}

	// Resolve student/komti contact
	var studentContact *store.Contact
	contacts, _ := s.store.ListContacts("komti")
	if len(contacts) > 0 {
		studentContact = &contacts[0]
	}

	ctx := s.tmplEngine.BuildContext(&sc, lectureDate, studentContact)
	rendered, err := s.tmplEngine.Render(tmplContent, ctx)
	if err != nil {
		slog.Error("Failed to render message for schedule", "schedule_id", sc.ID, "err", err)
		return
	}

	// Determine recipient phone / JID
	targetPhone := sc.TargetPhone
	if targetPhone == "" && sc.RecipientID.Valid {
		recContact, err := s.store.GetContactByID(sc.RecipientID.Int64)
		if err == nil {
			targetPhone = recContact.Phone
		}
	}
	if targetPhone == "" && sc.DosenID.Valid {
		dosenContact, err := s.store.GetContactByID(sc.DosenID.Int64)
		if err == nil {
			targetPhone = dosenContact.Phone
		}
	}

	if targetPhone == "" {
		slog.Warn("Schedule has no valid recipient phone", "schedule_id", sc.ID)
		return
	}

	qm := &store.QueueMessage{
		ScheduleID:    sql.NullInt64{Int64: sc.ID, Valid: true},
		RecipientJID:  targetPhone,
		RecipientName: sc.RecipientName,
		Message:       rendered,
		Status:        "pending",
		ScheduledFor:  now,
	}

	if qm.RecipientName == "" {
		qm.RecipientName = sc.DosenName
	}
	if qm.RecipientName == "" {
		qm.RecipientName = "Penerima WhatsApp"
	}

	_, err = s.queueMgr.Enqueue(qm)
	if err != nil {
		slog.Error("Failed to enqueue scheduled message", "schedule_id", sc.ID, "err", err)
		return
	}

	_ = s.store.UpdateScheduleLastSent(sc.ID, now)
	s.store.AddActivityLog("scheduler", "Pemicu Otomatis Jadwal Masuk Antrian",
		fmt.Sprintf("Matkul: %s, Penerima: %s, Jam Kuliah: %s", sc.Matkul, qm.RecipientName, sc.StartTime))
}

// TriggerScheduleManually allows instant trigger from web dashboard
func (s *Scheduler) TriggerScheduleManually(scheduleID int64, dryRun bool) (*store.QueueMessage, error) {
	sc, err := s.store.GetScheduleDetailByID(scheduleID)
	if err != nil {
		return nil, fmt.Errorf("jadwal tidak ditemukan: %w", err)
	}

	settings, _ := s.store.GetSettings()
	loc, _ := time.LoadLocation(settings.Timezone)
	if loc == nil {
		loc = time.FixedZone("WITA", 8*3600)
	}
	now := time.Now().In(loc)

	// Calculate next lecture occurrence date
	lectureDate := s.calculateNextLectureDate(sc.DayOfWeek, now)

	tmplContent := ""
	if sc.TemplateID.Valid {
		t, err := s.store.GetTemplateByID(sc.TemplateID.Int64)
		if err == nil {
			tmplContent = t.Content
		}
	}
	if tmplContent == "" {
		def, _ := s.store.GetDefaultTemplate()
		if def != nil {
			tmplContent = def.Content
		}
	}

	var studentContact *store.Contact
	contacts, _ := s.store.ListContacts("komti")
	if len(contacts) > 0 {
		studentContact = &contacts[0]
	}

	ctx := s.tmplEngine.BuildContext(sc, lectureDate, studentContact)
	rendered, err := s.tmplEngine.Render(tmplContent, ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal me-render pesan: %w", err)
	}

	targetPhone := sc.TargetPhone
	if targetPhone == "" && sc.RecipientID.Valid {
		recContact, err := s.store.GetContactByID(sc.RecipientID.Int64)
		if err == nil {
			targetPhone = recContact.Phone
		}
	}
	if targetPhone == "" && sc.DosenID.Valid {
		dosenContact, err := s.store.GetContactByID(sc.DosenID.Int64)
		if err == nil {
			targetPhone = dosenContact.Phone
		}
	}

	recipientName := sc.RecipientName
	if recipientName == "" {
		recipientName = sc.DosenName
	}
	if recipientName == "" {
		recipientName = "Penerima Jadwal"
	}

	qm := &store.QueueMessage{
		ScheduleID:    sql.NullInt64{Int64: sc.ID, Valid: true},
		RecipientJID:  targetPhone,
		RecipientName: recipientName,
		Message:       rendered,
		Status:        "pending",
		ScheduledFor:  now,
	}

	// If manual dry run requested, set temporary flag
	if dryRun {
		origDryRun := sc.DryRun
		sc.DryRun = true
		defer func() {
			sc.DryRun = origDryRun
		}()
	}

	res, err := s.queueMgr.Enqueue(qm)
	if err != nil {
		return nil, err
	}

	s.store.AddActivityLog("scheduler", "Manual Trigger Jadwal",
		fmt.Sprintf("Matkul: %s, Penerima: %s, Dry Run: %v", sc.Matkul, recipientName, dryRun))

	return res, nil
}

func (s *Scheduler) calculateNextLectureDate(targetDayOfWeek int, fromTime time.Time) time.Time {
	currentDay := int(fromTime.Weekday())
	daysUntil := (targetDayOfWeek - currentDay + 7) % 7
	if daysUntil == 0 {
		daysUntil = 7
	}
	return fromTime.AddDate(0, 0, daysUntil)
}

func (s *Scheduler) updateCountdown() {
	settings, err := s.store.GetSettings()
	if err != nil {
		return
	}

	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		loc = time.FixedZone("WITA", 8*3600)
	}

	now := time.Now().In(loc)
	schedules, err := s.store.ListActiveSchedules()
	if err != nil || len(schedules) == 0 {
		s.mu.Lock()
		s.nextInfo = nil
		s.mu.Unlock()
		s.hub.Broadcast("countdown", nil)
		return
	}

	var nearestTime time.Time
	var nearestSchedule *store.ScheduleDetail

	for i := range schedules {
		sc := &schedules[i]
		fireTime := s.calculateNextFireTime(sc, now, loc)
		if nearestTime.IsZero() || fireTime.Before(nearestTime) {
			nearestTime = fireTime
			nearestSchedule = sc
		}
	}

	if nearestSchedule == nil {
		return
	}

	remSecs := int64(nearestTime.Sub(now).Seconds())
	if remSecs < 0 {
		remSecs = 0
	}

	hours := remSecs / 3600
	minutes := (remSecs % 3600) / 60
	seconds := remSecs % 60
	remHuman := fmt.Sprintf("%02d jam %02d menit %02d detik", hours, minutes, seconds)
	if hours > 24 {
		days := hours / 24
		remHuman = fmt.Sprintf("%d hari %02d jam %02d menit", days, hours%24, minutes)
	}

	// Preview message
	lectureDate := s.calculateNextLectureDate(nearestSchedule.DayOfWeek, now)
	ctx := s.tmplEngine.BuildContext(nearestSchedule, lectureDate, nil)
	previewTmpl := ""
	if nearestSchedule.TemplateID.Valid {
		t, err := s.store.GetTemplateByID(nearestSchedule.TemplateID.Int64)
		if err == nil {
			previewTmpl = t.Content
		}
	}
	if previewTmpl == "" {
		def, _ := s.store.GetDefaultTemplate()
		if def != nil {
			previewTmpl = def.Content
		}
	}
	previewMsg, _ := s.tmplEngine.Render(previewTmpl, ctx)

	recName := nearestSchedule.RecipientName
	if recName == "" {
		recName = nearestSchedule.DosenName
	}

	info := &NextDeliveryInfo{
		ScheduleID:     nearestSchedule.ID,
		Matkul:         nearestSchedule.Matkul,
		RecipientName:  recName,
		TargetTime:     nearestTime.Format("02 Jan 15:04 WITA"),
		RemainingSecs:  remSecs,
		RemainingHuman: remHuman,
		MessagePreview: previewMsg,
		IsDryRun:       nearestSchedule.DryRun || settings.GlobalDryRun,
	}

	s.mu.Lock()
	s.nextInfo = info
	s.mu.Unlock()

	s.hub.Broadcast("countdown", info)
}

func (s *Scheduler) calculateNextFireTime(sc *store.ScheduleDetail, now time.Time, loc *time.Location) time.Time {
	// Parse SendAtTime
	var sendHour, sendMinute int
	_, _ = fmt.Sscanf(sc.SendAtTime, "%d:%d", &sendHour, &sendMinute)

	// In H-1 mode, fire day is 1 day before lecture day
	targetFireDayOfWeek := sc.DayOfWeek
	if sc.Mode == "H-1" {
		targetFireDayOfWeek = (sc.DayOfWeek - 1 + 7) % 7
	}

	currentDayOfWeek := int(now.Weekday())
	daysUntil := (targetFireDayOfWeek - currentDayOfWeek + 7) % 7

	fireTime := time.Date(now.Year(), now.Month(), now.Day(), sendHour, sendMinute, 0, 0, loc).AddDate(0, 0, daysUntil)
	if fireTime.Before(now) {
		fireTime = fireTime.AddDate(0, 0, 7)
	}

	return fireTime
}

// GetNextDeliveryInfo returns cached next delivery calculation
func (s *Scheduler) GetNextDeliveryInfo() *NextDeliveryInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextInfo
}
