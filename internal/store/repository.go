package store

import (
	"database/sql"
	"fmt"
	"time"
)

// ==========================================
// Users & Sessions
// ==========================================

func (s *Store) CountUsers() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (s *Store) CreateUser(username, passwordHash, role string) (*User, error) {
	now := time.Now()
	res, err := s.db.Exec(`INSERT INTO users (username, password_hash, role, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?)`, username, passwordHash, role, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	var u User
	var tfaEnabled int
	err := s.db.QueryRow(`SELECT id, username, password_hash, role, 
		COALESCE(two_factor_secret, ''), COALESCE(two_factor_enabled, 0), COALESCE(two_factor_backup_codes, ''),
		created_at, updated_at 
		FROM users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, 
			&u.TwoFactorSecret, &tfaEnabled, &u.TwoFactorBackupCodes,
			&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.TwoFactorEnabled = tfaEnabled == 1
	return &u, nil
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	var u User
	var tfaEnabled int
	err := s.db.QueryRow(`SELECT id, username, password_hash, role, 
		COALESCE(two_factor_secret, ''), COALESCE(two_factor_enabled, 0), COALESCE(two_factor_backup_codes, ''),
		created_at, updated_at 
		FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, 
			&u.TwoFactorSecret, &tfaEnabled, &u.TwoFactorBackupCodes,
			&u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.TwoFactorEnabled = tfaEnabled == 1
	return &u, nil
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, username, role, 
		COALESCE(two_factor_secret, ''), COALESCE(two_factor_enabled, 0), COALESCE(two_factor_backup_codes, ''),
		created_at, updated_at FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var tfaEnabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.TwoFactorSecret, &tfaEnabled, &u.TwoFactorBackupCodes, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.TwoFactorEnabled = tfaEnabled == 1
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) UpdateUser2FA(userID int64, secret string, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.db.Exec(`UPDATE users SET two_factor_secret = ?, two_factor_enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		secret, val, userID)
	return err
}

func (s *Store) UpdateUserPassword(userID int64, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		passwordHash, userID)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *Store) Create2FAChallenge(token string, userID int64, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO two_factor_challenges (token, user_id, expires_at, created_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, token, userID, expiresAt)
	return err
}

func (s *Store) Get2FAChallenge(token string) (*TwoFactorChallenge, error) {
	var c TwoFactorChallenge
	err := s.db.QueryRow(`SELECT token, user_id, expires_at, created_at 
		FROM two_factor_challenges WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`, token).
		Scan(&c.Token, &c.UserID, &c.ExpiresAt, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) Delete2FAChallenge(token string) error {
	_, err := s.db.Exec("DELETE FROM two_factor_challenges WHERE token = ?", token)
	return err
}

func (s *Store) CreateSession(token string, userID int64, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions (token, user_id, expires_at, created_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, token, userID, expiresAt)
	return err
}

func (s *Store) GetSession(token string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(`SELECT token, user_id, expires_at, created_at 
		FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`, token).
		Scan(&sess.Token, &sess.UserID, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *Store) CleanExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP")
	return err
}

// ==========================================
// Settings
// ==========================================

func (s *Store) GetSettings() (*Settings, error) {
	var st Settings
	var globalDryRun, regOpen int
	err := s.db.QueryRow(`SELECT global_dry_run, registration_open, send_window_start, send_window_end, 
		timezone, rate_limit_min_sec, rate_limit_max_sec, max_retries, updated_at 
		FROM settings WHERE id = 1`).
		Scan(&globalDryRun, &regOpen, &st.SendWindowStart, &st.SendWindowEnd,
			&st.Timezone, &st.RateLimitMinSec, &st.RateLimitMaxSec, &st.MaxRetries, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	st.GlobalDryRun = globalDryRun == 1
	st.RegistrationOpen = regOpen == 1
	return &st, nil
}

func (s *Store) UpdateSettings(st *Settings) error {
	globalDryRun := 0
	if st.GlobalDryRun {
		globalDryRun = 1
	}
	regOpen := 0
	if st.RegistrationOpen {
		regOpen = 1
	}
	_, err := s.db.Exec(`UPDATE settings SET 
		global_dry_run = ?, registration_open = ?, send_window_start = ?, send_window_end = ?, 
		timezone = ?, rate_limit_min_sec = ?, rate_limit_max_sec = ?, max_retries = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = 1`,
		globalDryRun, regOpen, st.SendWindowStart, st.SendWindowEnd,
		st.Timezone, st.RateLimitMinSec, st.RateLimitMaxSec, st.MaxRetries)
	return err
}

func (s *Store) SetRegistrationOpen(open bool) error {
	val := 0
	if open {
		val = 1
	}
	_, err := s.db.Exec("UPDATE settings SET registration_open = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1", val)
	return err
}

func (s *Store) SetGlobalDryRun(dryRun bool) error {
	val := 0
	if dryRun {
		val = 1
	}
	_, err := s.db.Exec("UPDATE settings SET global_dry_run = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1", val)
	return err
}

// ==========================================
// Contacts
// ==========================================

func (s *Store) ListContacts(userID int64, contactType string) ([]Contact, error) {
	query := `SELECT id, user_id, name, phone, contact_type, nim, email, notes, created_at, updated_at FROM contacts WHERE 1=1`
	var args []interface{}
	if userID > 0 {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	if contactType != "" {
		query += ` AND contact_type = ?`
		args = append(args, contactType)
	}
	query += ` ORDER BY name ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.ContactType, &c.NIM, &c.Email, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (s *Store) GetContactByID(id int64) (*Contact, error) {
	var c Contact
	err := s.db.QueryRow(`SELECT id, user_id, name, phone, contact_type, nim, email, notes, created_at, updated_at 
		FROM contacts WHERE id = ?`, id).
		Scan(&c.ID, &c.UserID, &c.Name, &c.Phone, &c.ContactType, &c.NIM, &c.Email, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateContact(c *Contact) (*Contact, error) {
	now := time.Now()
	if c.UserID <= 0 {
		c.UserID = 1
	}
	res, err := s.db.Exec(`INSERT INTO contacts (user_id, name, phone, contact_type, nim, email, notes, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.UserID, c.Name, c.Phone, c.ContactType, c.NIM, c.Email, c.Notes, now, now)
	if err != nil {
		return nil, err
	}
	c.ID, _ = res.LastInsertId()
	c.CreatedAt = now
	c.UpdatedAt = now
	return c, nil
}

func (s *Store) UpdateContact(c *Contact) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE contacts SET name = ?, phone = ?, contact_type = ?, nim = ?, email = ?, notes = ?, updated_at = ? 
		WHERE id = ?`,
		c.Name, c.Phone, c.ContactType, c.NIM, c.Email, c.Notes, now, c.ID)
	return err
}

func (s *Store) DeleteContact(id int64) error {
	_, err := s.db.Exec("DELETE FROM contacts WHERE id = ?", id)
	return err
}

// ==========================================
// Templates & Versions
// ==========================================

func (s *Store) ListTemplates() ([]Template, error) {
	rows, err := s.db.Query(`SELECT id, user_id, name, content, is_default, created_at, updated_at FROM templates ORDER BY is_default DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		var isDefault int
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Content, &isDefault, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.IsDefault = isDefault == 1
		templates = append(templates, t)
	}
	return templates, nil
}

func (s *Store) GetTemplateByID(id int64) (*Template, error) {
	var t Template
	var isDefault int
	err := s.db.QueryRow(`SELECT id, user_id, name, content, is_default, created_at, updated_at FROM templates WHERE id = ?`, id).
		Scan(&t.ID, &t.UserID, &t.Name, &t.Content, &isDefault, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.IsDefault = isDefault == 1
	return &t, nil
}

func (s *Store) GetDefaultTemplate() (*Template, error) {
	var t Template
	var isDefault int
	err := s.db.QueryRow(`SELECT id, user_id, name, content, is_default, created_at, updated_at FROM templates WHERE is_default = 1 LIMIT 1`).
		Scan(&t.ID, &t.UserID, &t.Name, &t.Content, &isDefault, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		// Fallback to first template
		err = s.db.QueryRow(`SELECT id, user_id, name, content, is_default, created_at, updated_at FROM templates ORDER BY id ASC LIMIT 1`).
			Scan(&t.ID, &t.UserID, &t.Name, &t.Content, &isDefault, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
	}
	t.IsDefault = isDefault == 1
	return &t, nil
}

func (s *Store) CreateTemplate(t *Template) (*Template, error) {
	now := time.Now()
	isDefault := 0
	if t.IsDefault {
		isDefault = 1
		_, _ = s.db.Exec("UPDATE templates SET is_default = 0")
	}

	res, err := s.db.Exec(`INSERT INTO templates (user_id, name, content, is_default, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`, t.UserID, t.Name, t.Content, isDefault, now, now)
	if err != nil {
		return nil, err
	}
	t.ID, _ = res.LastInsertId()
	t.CreatedAt = now
	t.UpdatedAt = now

	// Save initial version
	_, _ = s.db.Exec(`INSERT INTO template_versions (template_id, content, version, created_at) VALUES (?, ?, 1, ?)`,
		t.ID, t.Content, now)

	return t, nil
}

func (s *Store) UpdateTemplate(t *Template) error {
	now := time.Now()
	isDefault := 0
	if t.IsDefault {
		isDefault = 1
		_, _ = s.db.Exec("UPDATE templates SET is_default = 0 WHERE id != ?", t.ID)
	}

	_, err := s.db.Exec(`UPDATE templates SET name = ?, content = ?, is_default = ?, updated_at = ? WHERE id = ?`,
		t.Name, t.Content, isDefault, now, t.ID)
	if err != nil {
		return err
	}

	// Calculate next version
	var nextVer int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) + 1 FROM template_versions WHERE template_id = ?`, t.ID).Scan(&nextVer)
	_, _ = s.db.Exec(`INSERT INTO template_versions (template_id, content, version, created_at) VALUES (?, ?, ?, ?)`,
		t.ID, t.Content, nextVer, now)

	return nil
}

func (s *Store) DeleteTemplate(id int64) error {
	_, err := s.db.Exec("DELETE FROM templates WHERE id = ?", id)
	return err
}

func (s *Store) ListTemplateVersions(templateID int64) ([]TemplateVersion, error) {
	rows, err := s.db.Query(`SELECT id, template_id, content, version, created_at 
		FROM template_versions WHERE template_id = ? ORDER BY version DESC`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []TemplateVersion
	for rows.Next() {
		var v TemplateVersion
		if err := rows.Scan(&v.ID, &v.TemplateID, &v.Content, &v.Version, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// ==========================================
// Schedules
// ==========================================

func (s *Store) ListSchedules(userID int64) ([]ScheduleDetail, error) {
	query := `SELECT 
		s.id, s.user_id, s.title, s.matkul, s.dosen_id, s.recipient_id, s.target_phone, s.template_id,
		s.day_of_week, s.start_time, s.end_time, s.location, s.link_group, s.mode, s.send_at_time,
		s.is_active, s.is_public, s.dry_run, s.last_sent_at, s.created_at, s.updated_at,
		COALESCE(d.name, '') as dosen_name,
		COALESCE(r.name, '') as recipient_name,
		COALESCE(r.contact_type, '') as recipient_type,
		COALESCE(t.name, '') as template_name,
		COALESCE(u.username, '') as creator_name
	FROM schedules s
	LEFT JOIN contacts d ON s.dosen_id = d.id
	LEFT JOIN contacts r ON s.recipient_id = r.id
	LEFT JOIN templates t ON s.template_id = t.id
	LEFT JOIN users u ON s.user_id = u.id`

	var args []interface{}
	if userID > 0 {
		query += ` WHERE s.user_id = ?`
		args = append(args, userID)
	}
	query += ` ORDER BY s.day_of_week ASC, s.start_time ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleDetail
	for rows.Next() {
		var sd ScheduleDetail
		var isActive, isPublic, dryRun int
		var lastSentAt sql.NullTime

		if err := rows.Scan(
			&sd.ID, &sd.UserID, &sd.Title, &sd.Matkul, &sd.DosenID, &sd.RecipientID, &sd.TargetPhone, &sd.TemplateID,
			&sd.DayOfWeek, &sd.StartTime, &sd.EndTime, &sd.Location, &sd.LinkGroup, &sd.Mode, &sd.SendAtTime,
			&isActive, &isPublic, &dryRun, &lastSentAt, &sd.CreatedAt, &sd.UpdatedAt,
			&sd.DosenName, &sd.RecipientName, &sd.RecipientType, &sd.TemplateName, &sd.CreatorName,
		); err != nil {
			return nil, err
		}

		sd.IsActive = isActive == 1
		sd.IsPublic = isPublic == 1
		sd.DryRun = dryRun == 1
		if lastSentAt.Valid {
			sd.LastSentAt = &lastSentAt.Time
		}
		list = append(list, sd)
	}
	return list, nil
}

func (s *Store) ListActiveSchedules() ([]ScheduleDetail, error) {
	query := `SELECT 
		s.id, s.user_id, s.title, s.matkul, s.dosen_id, s.recipient_id, s.target_phone, s.template_id,
		s.day_of_week, s.start_time, s.end_time, s.location, s.link_group, s.mode, s.send_at_time,
		s.is_active, s.is_public, s.dry_run, s.last_sent_at, s.created_at, s.updated_at,
		COALESCE(d.name, '') as dosen_name,
		COALESCE(r.name, '') as recipient_name,
		COALESCE(r.contact_type, '') as recipient_type,
		COALESCE(t.name, '') as template_name,
		COALESCE(u.username, '') as creator_name
	FROM schedules s
	LEFT JOIN contacts d ON s.dosen_id = d.id
	LEFT JOIN contacts r ON s.recipient_id = r.id
	LEFT JOIN templates t ON s.template_id = t.id
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.is_active = 1
	ORDER BY s.day_of_week ASC, s.start_time ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleDetail
	for rows.Next() {
		var sd ScheduleDetail
		var isActive, isPublic, dryRun int
		var lastSentAt sql.NullTime

		if err := rows.Scan(
			&sd.ID, &sd.UserID, &sd.Title, &sd.Matkul, &sd.DosenID, &sd.RecipientID, &sd.TargetPhone, &sd.TemplateID,
			&sd.DayOfWeek, &sd.StartTime, &sd.EndTime, &sd.Location, &sd.LinkGroup, &sd.Mode, &sd.SendAtTime,
			&isActive, &isPublic, &dryRun, &lastSentAt, &sd.CreatedAt, &sd.UpdatedAt,
			&sd.DosenName, &sd.RecipientName, &sd.RecipientType, &sd.TemplateName, &sd.CreatorName,
		); err != nil {
			return nil, err
		}

		sd.IsActive = isActive == 1
		sd.IsPublic = isPublic == 1
		sd.DryRun = dryRun == 1
		if lastSentAt.Valid {
			sd.LastSentAt = &lastSentAt.Time
		}
		list = append(list, sd)
	}
	return list, nil
}

func (s *Store) ListPublicSchedules() ([]ScheduleDetail, error) {
	query := `SELECT 
		s.id, s.user_id, s.title, s.matkul, s.dosen_id, s.recipient_id, s.target_phone, s.template_id,
		s.day_of_week, s.start_time, s.end_time, s.location, s.link_group, s.mode, s.send_at_time,
		s.is_active, s.is_public, s.dry_run, s.last_sent_at, s.created_at, s.updated_at,
		COALESCE(d.name, '') as dosen_name,
		COALESCE(r.name, '') as recipient_name,
		COALESCE(r.contact_type, '') as recipient_type,
		COALESCE(t.name, '') as template_name,
		COALESCE(u.username, '') as creator_name
	FROM schedules s
	LEFT JOIN contacts d ON s.dosen_id = d.id
	LEFT JOIN contacts r ON s.recipient_id = r.id
	LEFT JOIN templates t ON s.template_id = t.id
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.is_public = 1 AND s.is_active = 1
	ORDER BY s.day_of_week ASC, s.start_time ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleDetail
	for rows.Next() {
		var sd ScheduleDetail
		var isActive, isPublic, dryRun int
		var lastSentAt sql.NullTime

		if err := rows.Scan(
			&sd.ID, &sd.UserID, &sd.Title, &sd.Matkul, &sd.DosenID, &sd.RecipientID, &sd.TargetPhone, &sd.TemplateID,
			&sd.DayOfWeek, &sd.StartTime, &sd.EndTime, &sd.Location, &sd.LinkGroup, &sd.Mode, &sd.SendAtTime,
			&isActive, &isPublic, &dryRun, &lastSentAt, &sd.CreatedAt, &sd.UpdatedAt,
			&sd.DosenName, &sd.RecipientName, &sd.RecipientType, &sd.TemplateName, &sd.CreatorName,
		); err != nil {
			return nil, err
		}

		sd.IsActive = isActive == 1
		sd.IsPublic = isPublic == 1
		sd.DryRun = dryRun == 1
		if lastSentAt.Valid {
			sd.LastSentAt = &lastSentAt.Time
		}
		list = append(list, sd)
	}
	return list, nil
}

func (s *Store) GetScheduleByID(id int64) (*Schedule, error) {
	var sc Schedule
	var isActive, isPublic, dryRun int
	var lastSentAt sql.NullTime

	err := s.db.QueryRow(`SELECT 
		id, user_id, title, matkul, dosen_id, recipient_id, target_phone, template_id,
		day_of_week, start_time, end_time, location, link_group, mode, send_at_time,
		is_active, is_public, dry_run, last_sent_at, created_at, updated_at
		FROM schedules WHERE id = ?`, id).
		Scan(
			&sc.ID, &sc.UserID, &sc.Title, &sc.Matkul, &sc.DosenID, &sc.RecipientID, &sc.TargetPhone, &sc.TemplateID,
			&sc.DayOfWeek, &sc.StartTime, &sc.EndTime, &sc.Location, &sc.LinkGroup, &sc.Mode, &sc.SendAtTime,
			&isActive, &isPublic, &dryRun, &lastSentAt, &sc.CreatedAt, &sc.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	sc.IsActive = isActive == 1
	sc.IsPublic = isPublic == 1
	sc.DryRun = dryRun == 1
	if lastSentAt.Valid {
		sc.LastSentAt = &lastSentAt.Time
	}
	return &sc, nil
}

func (s *Store) GetScheduleDetailByID(id int64) (*ScheduleDetail, error) {
	query := `SELECT 
		s.id, s.user_id, s.title, s.matkul, s.dosen_id, s.recipient_id, s.target_phone, s.template_id,
		s.day_of_week, s.start_time, s.end_time, s.location, s.link_group, s.mode, s.send_at_time,
		s.is_active, s.is_public, s.dry_run, s.last_sent_at, s.created_at, s.updated_at,
		COALESCE(d.name, '') as dosen_name,
		COALESCE(r.name, '') as recipient_name,
		COALESCE(r.contact_type, '') as recipient_type,
		COALESCE(t.name, '') as template_name,
		COALESCE(u.username, '') as creator_name
	FROM schedules s
	LEFT JOIN contacts d ON s.dosen_id = d.id
	LEFT JOIN contacts r ON s.recipient_id = r.id
	LEFT JOIN templates t ON s.template_id = t.id
	LEFT JOIN users u ON s.user_id = u.id
	WHERE s.id = ?`

	var sd ScheduleDetail
	var isActive, isPublic, dryRun int
	var lastSentAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&sd.ID, &sd.UserID, &sd.Title, &sd.Matkul, &sd.DosenID, &sd.RecipientID, &sd.TargetPhone, &sd.TemplateID,
		&sd.DayOfWeek, &sd.StartTime, &sd.EndTime, &sd.Location, &sd.LinkGroup, &sd.Mode, &sd.SendAtTime,
		&isActive, &isPublic, &dryRun, &lastSentAt, &sd.CreatedAt, &sd.UpdatedAt,
		&sd.DosenName, &sd.RecipientName, &sd.RecipientType, &sd.TemplateName, &sd.CreatorName,
	)
	if err != nil {
		return nil, err
	}
	sd.IsActive = isActive == 1
	sd.IsPublic = isPublic == 1
	sd.DryRun = dryRun == 1
	if lastSentAt.Valid {
		sd.LastSentAt = &lastSentAt.Time
	}
	return &sd, nil
}

func (s *Store) CreateSchedule(sc *Schedule) (*Schedule, error) {
	now := time.Now()
	if sc.UserID <= 0 {
		sc.UserID = 1
	}
	isActive := 0
	if sc.IsActive {
		isActive = 1
	}
	isPublic := 0
	if sc.IsPublic {
		isPublic = 1
	}
	dryRun := 0
	if sc.DryRun {
		dryRun = 1
	}

	res, err := s.db.Exec(`INSERT INTO schedules (
		user_id, title, matkul, dosen_id, recipient_id, target_phone, template_id,
		day_of_week, start_time, end_time, location, link_group, mode, send_at_time,
		is_active, is_public, dry_run, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.UserID, sc.Title, sc.Matkul, sc.DosenID, sc.RecipientID, sc.TargetPhone, sc.TemplateID,
		sc.DayOfWeek, sc.StartTime, sc.EndTime, sc.Location, sc.LinkGroup, sc.Mode, sc.SendAtTime,
		isActive, isPublic, dryRun, now, now,
	)
	if err != nil {
		return nil, err
	}
	sc.ID, _ = res.LastInsertId()
	sc.CreatedAt = now
	sc.UpdatedAt = now
	return sc, nil
}

func (s *Store) UpdateSchedule(sc *Schedule) error {
	now := time.Now()
	isActive := 0
	if sc.IsActive {
		isActive = 1
	}
	isPublic := 0
	if sc.IsPublic {
		isPublic = 1
	}
	dryRun := 0
	if sc.DryRun {
		dryRun = 1
	}

	_, err := s.db.Exec(`UPDATE schedules SET 
		title = ?, matkul = ?, dosen_id = ?, recipient_id = ?, target_phone = ?, template_id = ?,
		day_of_week = ?, start_time = ?, end_time = ?, location = ?, link_group = ?, mode = ?, send_at_time = ?,
		is_active = ?, is_public = ?, dry_run = ?, updated_at = ?
		WHERE id = ?`,
		sc.Title, sc.Matkul, sc.DosenID, sc.RecipientID, sc.TargetPhone, sc.TemplateID,
		sc.DayOfWeek, sc.StartTime, sc.EndTime, sc.Location, sc.LinkGroup, sc.Mode, sc.SendAtTime,
		isActive, isPublic, dryRun, now, sc.ID)
	return err
}

func (s *Store) UpdateScheduleLastSent(id int64, t time.Time) error {
	_, err := s.db.Exec("UPDATE schedules SET last_sent_at = ? WHERE id = ?", t, id)
	return err
}

func (s *Store) ToggleScheduleActive(id int64, active bool) error {
	val := 0
	if active {
		val = 1
	}
	_, err := s.db.Exec("UPDATE schedules SET is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", val, id)
	return err
}

func (s *Store) DeleteSchedule(id int64) error {
	_, err := s.db.Exec("DELETE FROM schedules WHERE id = ?", id)
	return err
}

// ==========================================
// Holidays
// ==========================================

func (s *Store) ListHolidays() ([]Holiday, error) {
	rows, err := s.db.Query(`SELECT id, date, description FROM holidays ORDER BY date ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []Holiday
	for rows.Next() {
		var h Holiday
		if err := rows.Scan(&h.ID, &h.Date, &h.Description); err != nil {
			return nil, err
		}
		holidays = append(holidays, h)
	}
	return holidays, nil
}

func (s *Store) IsHoliday(dateStr string) (bool, string, error) {
	var desc string
	err := s.db.QueryRow("SELECT description FROM holidays WHERE date = ?", dateStr).Scan(&desc)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, desc, nil
}

func (s *Store) CreateHoliday(dateStr, description string) error {
	_, err := s.db.Exec(`INSERT INTO holidays (date, description) VALUES (?, ?)`, dateStr, description)
	return err
}

func (s *Store) DeleteHoliday(id int64) error {
	_, err := s.db.Exec("DELETE FROM holidays WHERE id = ?", id)
	return err
}

// ==========================================
// Queue & Messages History
// ==========================================

func (s *Store) CreateQueueMessage(qm *QueueMessage) (*QueueMessage, error) {
	now := time.Now()
	if qm.UserID <= 0 {
		qm.UserID = 1
	}
	res, err := s.db.Exec(`INSERT INTO queue_messages (
		user_id, schedule_id, recipient_jid, recipient_name, message, status, 
		retry_count, max_retries, error_message, scheduled_for, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		qm.UserID, qm.ScheduleID, qm.RecipientJID, qm.RecipientName, qm.Message, qm.Status,
		qm.RetryCount, qm.MaxRetries, qm.ErrorMessage, qm.ScheduledFor, now, now,
	)
	if err != nil {
		return nil, err
	}
	qm.ID, _ = res.LastInsertId()
	qm.CreatedAt = now
	qm.UpdatedAt = now
	return qm, nil
}

func (s *Store) ListQueueMessages(userID int64, status string, limit int) ([]QueueMessage, error) {
	query := `SELECT id, user_id, schedule_id, recipient_jid, recipient_name, message, status, 
		retry_count, max_retries, error_message, scheduled_for, sent_at, created_at, updated_at 
		FROM queue_messages WHERE 1=1`
	var args []interface{}
	if userID > 0 {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY scheduled_for DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []QueueMessage
	for rows.Next() {
		var qm QueueMessage
		var sentAt sql.NullTime
		if err := rows.Scan(
			&qm.ID, &qm.UserID, &qm.ScheduleID, &qm.RecipientJID, &qm.RecipientName, &qm.Message, &qm.Status,
			&qm.RetryCount, &qm.MaxRetries, &qm.ErrorMessage, &qm.ScheduledFor, &sentAt, &qm.CreatedAt, &qm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if sentAt.Valid {
			qm.SentAt = &sentAt.Time
		}
		list = append(list, qm)
	}
	return list, nil
}

func (s *Store) GetPendingQueueMessages(now time.Time, limit int) ([]QueueMessage, error) {
	rows, err := s.db.Query(`SELECT id, user_id, schedule_id, recipient_jid, recipient_name, message, status, 
		retry_count, max_retries, error_message, scheduled_for, sent_at, created_at, updated_at 
		FROM queue_messages 
		WHERE status = 'pending' AND scheduled_for <= ? 
		ORDER BY scheduled_for ASC LIMIT ?`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []QueueMessage
	for rows.Next() {
		var qm QueueMessage
		var sentAt sql.NullTime
		if err := rows.Scan(
			&qm.ID, &qm.UserID, &qm.ScheduleID, &qm.RecipientJID, &qm.RecipientName, &qm.Message, &qm.Status,
			&qm.RetryCount, &qm.MaxRetries, &qm.ErrorMessage, &qm.ScheduledFor, &sentAt, &qm.CreatedAt, &qm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if sentAt.Valid {
			qm.SentAt = &sentAt.Time
		}
		list = append(list, qm)
	}
	return list, nil
}

func (s *Store) GetQueueMessageByID(id int64) (*QueueMessage, error) {
	var qm QueueMessage
	var sentAt sql.NullTime
	err := s.db.QueryRow(`SELECT id, user_id, schedule_id, recipient_jid, recipient_name, message, status, 
		retry_count, max_retries, error_message, scheduled_for, sent_at, created_at, updated_at 
		FROM queue_messages WHERE id = ?`, id).
		Scan(
			&qm.ID, &qm.UserID, &qm.ScheduleID, &qm.RecipientJID, &qm.RecipientName, &qm.Message, &qm.Status,
			&qm.RetryCount, &qm.MaxRetries, &qm.ErrorMessage, &qm.ScheduledFor, &sentAt, &qm.CreatedAt, &qm.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	if sentAt.Valid {
		qm.SentAt = &sentAt.Time
	}
	return &qm, nil
}

func (s *Store) UpdateQueueStatus(id int64, status, errorMsg string, sentAt *time.Time) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE queue_messages SET status = ?, error_message = ?, sent_at = ?, updated_at = ? WHERE id = ?`,
		status, errorMsg, sentAt, now, id)
	return err
}

func (s *Store) IncrementRetry(id int64, nextTime time.Time, errorMsg string) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE queue_messages SET 
		retry_count = retry_count + 1, status = 'pending', scheduled_for = ?, error_message = ?, updated_at = ? 
		WHERE id = ?`, nextTime, errorMsg, now, id)
	return err
}

func (s *Store) CancelQueueMessage(id int64) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE queue_messages SET status = 'cancelled', updated_at = ? WHERE id = ? AND status = 'pending'`,
		now, id)
	return err
}

func (s *Store) DeleteQueueMessage(id int64) error {
	_, err := s.db.Exec("DELETE FROM queue_messages WHERE id = ?", id)
	return err
}

// ==========================================
// Activity & System Logs
// ==========================================

func (s *Store) AddActivityLog(category, message, details string) {
	_, _ = s.db.Exec(`INSERT INTO activity_logs (category, message, details, created_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`, category, message, details)
}

func (s *Store) ListActivityLogs(limit int, category string) ([]ActivityLog, error) {
	query := `SELECT id, category, message, details, created_at FROM activity_logs`
	var args []interface{}
	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ActivityLog
	for rows.Next() {
		var l ActivityLog
		if err := rows.Scan(&l.ID, &l.Category, &l.Message, &l.Details, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ==========================================
// Dashboard Statistics
// ==========================================

func (s *Store) GetDashboardStats(userID int64) (*DashboardStats, error) {
	var stats DashboardStats

	todayStart := time.Now().Format("2006-01-02") + " 00:00:00"

	if userID > 0 {
		// Sent today
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE user_id = ? AND status = 'sent' AND sent_at >= ?`, userID, todayStart).
			Scan(&stats.TotalSentToday)

		// Total sent all time (including dry_run for statistics)
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE user_id = ? AND status IN ('sent', 'dry_run')`, userID).
			Scan(&stats.TotalSentAll)

		// Total failed all time
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE user_id = ? AND status = 'failed'`, userID).
			Scan(&stats.TotalFailedAll)

		// Total pending
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE user_id = ? AND status = 'pending'`, userID).
			Scan(&stats.TotalPending)

		// Total active schedules
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM schedules WHERE user_id = ? AND is_active = 1`, userID).
			Scan(&stats.TotalSchedules)
		stats.ActiveSchedules = stats.TotalSchedules

		// Total contacts
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM contacts WHERE user_id = ?`, userID).
			Scan(&stats.TotalContacts)
	} else {
		// System-wide
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE status = 'sent' AND sent_at >= ?`, todayStart).
			Scan(&stats.TotalSentToday)

		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE status IN ('sent', 'dry_run')`).
			Scan(&stats.TotalSentAll)

		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE status = 'failed'`).
			Scan(&stats.TotalFailedAll)

		_ = s.db.QueryRow(`SELECT COUNT(*) FROM queue_messages WHERE status = 'pending'`).
			Scan(&stats.TotalPending)

		_ = s.db.QueryRow(`SELECT COUNT(*) FROM schedules WHERE is_active = 1`).
			Scan(&stats.TotalSchedules)
		stats.ActiveSchedules = stats.TotalSchedules

		_ = s.db.QueryRow(`SELECT COUNT(*) FROM contacts`).
			Scan(&stats.TotalContacts)
	}

	// Calculate success rate
	totalAttempts := stats.TotalSentAll + stats.TotalFailedAll
	if totalAttempts > 0 {
		stats.SuccessRate = (float64(stats.TotalSentAll) / float64(totalAttempts)) * 100
	} else {
		stats.SuccessRate = 100.0
	}

	return &stats, nil
}
