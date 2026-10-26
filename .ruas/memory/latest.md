---
ruas_dialect: ai-native/v1
target: "Autonomous AI Coding Agents (Cursor, Claude, Antigravity, Cline)"
generated_at: "2026-10-26T00:53:15.102Z"
project_root: "/Users/kou/Project/SiPenDosa"
total_symbols: 660
total_edges: 1561
---

# 🌿 Ruas AI-Native Memory & Working Context
> **LLM NOTICE:** This document is pre-indexed semantic ground truth. Do NOT rescan ASTs or hallucinate unverified relationships.

## 🏛️ Central Hub Symbols (God Nodes)
| Symbol | Kind | Inbound Callers | Outbound Calls | Risk Assessment |
| :--- | :--- | :---: | :---: | :--- |
| `Close` | `method` | 54 | 2 | 🔴 HIGH BLAST RADIUS |
| `Close` | `method` | 53 | 2 | 🔴 HIGH BLAST RADIUS |
| `Close` | `method` | 52 | 3 | 🔴 HIGH BLAST RADIUS |
| `main` | `function` | 0 | 40 | 🟡 MODERATE |
| `BroadcastToast` | `method` | 30 | 1 | 🔴 HIGH BLAST RADIUS |
| `getActiveUserID` | `method` | 26 | 2 | 🔴 HIGH BLAST RADIUS |
| `setupTestWeb` | `function` | 11 | 16 | 🔴 HIGH BLAST RADIUS |
| `AddActivityLog` | `method` | 26 | 0 | 🔴 HIGH BLAST RADIUS |
| `Context` | `class` | 22 | 0 | 🔴 HIGH BLAST RADIUS |
| `New` | `function` | 19 | 2 | 🔴 HIGH BLAST RADIUS |

## ⚡ Bayesian Critical Call Trajectories
- `[main] --(CALLS:89%)--> [printBanner]`
- `[main] --(CALLS:89%)--> [createHelperScripts]`
- `[main] --(CALLS:89%)--> [createShortcuts]`
- `[main] --(CALLS:89%)--> [registerUninstall]`
- `[main] --(CALLS:89%)--> [printSuccessBox]`
- `[main] --(CALLS:89%)--> [launchApp]`
- `[createShortcuts] --(CALLS:89%)--> [escapePS]`
- `[registerUninstall] --(CALLS:89%)--> [escapePS]`
- `[main] --(CALLS:89%)--> [runPairCli]`
- `[main] --(CALLS:89%)--> [runModeCli]`
- `[main] --(CALLS:89%)--> [printCliHelp]`
- `[main] --(CALLS:89%)--> [resolveUIMode]`
- `[main] --(CALLS:89%)--> [openDesktopApp]`
- `[resolveUIMode] --(CALLS:89%)--> [isTTY]`
- `[resolveUIMode] --(CALLS:89%)--> [runFirstRunWizard]`
- `[runFirstRunWizard] --(CALLS:89%)--> [persistUIMode]`
- `[runModeCli] --(CALLS:89%)--> [persistUIMode]`
- `[openDesktopApp] --(CALLS:89%)--> [openBrowserAppMode]`
- `[RequireAuth] --(CALLS:89%)--> [ValidateSession]`
- `[RequireAuth] --(CALLS:89%)--> [ClearSessionCookie]`

## 📐 Compact AST Signatures (Zero-Fluff Token Layout)
### `web/assets.go`
- [module] `webassets` :: package webassets (L1-L1)
- [function] `Templates` :: func Templates() fs.FS (L12-L18)
- [function] `Static` :: func Static() fs.FS (L21-L27)

### `cmd/installer/main.go`
- [module] `main` :: package main (L3-L3)
- [function] `main` :: func main() (L44-L196)
- [function] `printBanner` :: func printBanner() (L198-L220)
- [function] `printSuccessBox` :: func printSuccessBox(installDir string) (L222-L235)
- [function] `createHelperScripts` :: func createHelperScripts(installDir string) (L237-L299)
- [function] `createShortcuts` :: func createShortcuts(installDir, targetExe string) (L301-L375)
- [function] `registerUninstall` :: func registerUninstall(installDir string) (L377-L401)
- [function] `launchApp` :: func launchApp(installDir, targetExe string) (L403-L414)
- [function] `escapePS` :: func escapePS(s string) string (L416-L418)

### `cmd/sipen/main.go`
- [module] `main` :: package main (L1-L1)
- [function] `main` :: func main() (L42-L245)
- [function] `runPairCli` :: func runPairCli(cfg *config.Config, phone string) (L247-L302)
- [function] `printCliHelp` :: func printCliHelp() (L304-L317)
- [function] `isTTY` :: func isTTY() bool (L320-L326)
- [function] `resolveUIMode` :: func resolveUIMode(cfg *config.Config, forced string) string (L330-L355)
- [function] `runFirstRunWizard` :: func runFirstRunWizard(cfg *config.Config) string (L358-L389)
- [function] `persistUIMode` :: func persistUIMode(cfg *config.Config, mode string) error (L392-L421)
- [function] `runModeCli` :: func runModeCli(cfg *config.Config, args []string) (L424-L442)
- [function] `openDesktopApp` :: func openDesktopApp(serverAddr string) (L447-L464)
- [function] `openBrowserAppMode` :: func openBrowserAppMode(url string) (L468-L493)

### `internal/auth/auth.go`
- [module] `auth` :: package auth (L1-L1)
- [class] `contextKey` :: type contextKey type_identifier (L18-L18)
- [class] `Service` :: type Service struct (L34-L36)
- [function] `NewService` :: func NewService(s *store.Store) *Service (L39-L41)
- [method] `RegisterFirstUser` :: func (s *Service) RegisterFirstUser(username, password string) (*store.User, error) (L44-L68)
- [method] `RegisterUser` :: func (s *Service) RegisterUser(username, password, role string, allowOverride bool) (*store.User, error) (L71-L95)
- [method] `Authenticate` :: func (s *Service) Authenticate(username, password string) (*store.User, error) (L98-L109)
- [method] `CreateSession` :: func (s *Service) CreateSession(userID int64) (string, error) (L112-L125)
- [method] `ValidateSession` :: func (s *Service) ValidateSession(token string) (*store.User, error) (L128-L140)
- [method] `RevokeSession` :: func (s *Service) RevokeSession(token string) error (L143-L145)
- [function] `SetSessionCookie` :: func SetSessionCookie(w http.ResponseWriter, token string, secure bool) (L149-L159)
- [function] `ClearSessionCookie` :: func ClearSessionCookie(w http.ResponseWriter) (L162-L173)
- [method] `RequireAuth` :: func (s *Service) RequireAuth(next http.Handler) http.Handler (L176-L217)
- [method] `RequireAuthWS` :: func (s *Service) RequireAuthWS(next http.Handler) http.Handler (L222-L246)
- [method] `Create2FAChallenge` :: func (s *Service) Create2FAChallenge(userID int64) (string, error) (L249-L261)
- [method] `Verify2FA` :: func (s *Service) Verify2FA(challengeToken, passcode string) (*store.User, error) (L264-L282)
- [function] `RequireAdmin` :: func RequireAdmin(next http.Handler) http.Handler (L285-L294)
- [function] `RequireSuperAdmin` :: func RequireSuperAdmin(next http.Handler) http.Handler (L297-L306)
- [function] `GetUserFromContext` :: func GetUserFromContext(ctx context.Context) *store.User (L309-L314)

### `internal/auth/auth_test.go`
- [module] `auth_test` :: package auth_test (L1-L1)
- [function] `setupTestStore` :: func setupTestStore(t *testing.T) (*store.Store, *auth.Service) (L14-L25)
- [function] `TestAuthFlowWith2FA` :: func TestAuthFlowWith2FA(t *testing.T) (L27-L108)
- [function] `TestClearSessionCookie` :: func TestClearSessionCookie(t *testing.T) (L110-L133)

### `internal/banner/banner.go`
- [module] `banner` :: package banner (L1-L1)
- [function] `PrintCompact` :: func PrintCompact(cfg *config.Config) (L30-L51)
- [function] `PrintBanner` :: func PrintBanner(cfg *config.Config) (L53-L122)
- [function] `formatRow` :: func formatRow(label, value string) (L124-L128)
- [function] `LogStep` :: func LogStep(tag, message string) (L131-L152)
- [function] `PrintQRHeader` :: func PrintQRHeader() (L155-L168)
- [function] `PrintQRFooter` :: func PrintQRFooter(port string) (L171-L176)
- [function] `PrintConnectedBox` :: func PrintConnectedBox(jid, pushName string) (L179-L194)

### `internal/config/config.go`
- [module] `config` :: package config (L1-L1)
- [class] `Config` :: type Config struct (L21-L37)
- [function] `Load` :: func Load() (*Config, error) (L40-L119)
- [function] `resolvePath` :: func resolvePath(p, baseDir string) string (L121-L155)
- [method] `GetLocation` :: func (c *Config) GetLocation() *time.Location (L158-L165)
- [function] `getEnv` :: func getEnv(key, fallback string) string (L167-L172)
- [function] `getEnvInt` :: func getEnvInt(key string, fallback int) int (L174-L181)
- [function] `getEnvBool` :: func getEnvBool(key string, fallback bool) bool (L183-L190)

### `internal/config/config_test.go`
- [module] `config` :: package config (L1-L1)
- [function] `TestConfigDefaults` :: func TestConfigDefaults(t *testing.T) (L9-L27)
- [function] `TestResolvePath` :: func TestResolvePath(t *testing.T) (L29-L52)

### `internal/netutil/resolver.go`
- [module] `netutil` :: package netutil (L1-L1)
- [function] `SetupNetworkEnvironment` :: func SetupNetworkEnvironment() (L30-L35)
- [function] `IsAndroid` :: func IsAndroid() bool (L38-L49)
- [function] `setupDNSResolver` :: func setupDNSResolver() (L51-L106)
- [function] `getAndroidSystemDNS` :: func getAndroidSystemDNS() []string (L108-L129)
- [function] `setupRootCerts` :: func setupRootCerts() (L131-L179)

### `internal/queue/queue.go`
- [module] `queue` :: package queue (L1-L1)
- [class] `Manager` :: type Manager struct (L18-L26)
- [function] `NewManager` :: func NewManager(s *store.Store, waMgr *whatsapp.Manager, hub *realtime.Hub) *Manager (L29-L37)
- [method] `Start` :: func (m *Manager) Start() (L40-L50)
- [method] `Stop` :: func (m *Manager) Stop() (L53-L61)
- [method] `TriggerProcessing` :: func (m *Manager) TriggerProcessing() (L64-L69)
- [method] `Enqueue` :: func (m *Manager) Enqueue(qm *store.QueueMessage) (*store.QueueMessage, error) (L72-L93)
- [method] `workerLoop` :: func (m *Manager) workerLoop() (L95-L109)
- [method] `processPending` :: func (m *Manager) processPending() (L111-L129)
- [method] `processSingleMessage` :: func (m *Manager) processSingleMessage(msg store.QueueMessage, settings *store.Settings) (L131-L251)
- [method] `ProcessImmediate` :: func (m *Manager) ProcessImmediate(id int64) error (L254-L272)

### `internal/scheduler/scheduler.go`
- [module] `scheduler` :: package scheduler (L1-L1)
- [class] `NextDeliveryInfo` :: type NextDeliveryInfo struct (L17-L26)
- [class] `Scheduler` :: type Scheduler struct (L29-L38)
- [function] `NewScheduler` :: func NewScheduler(s *store.Store, q *queue.Manager, t *template.Engine, hub *realtime.Hub) *Scheduler (L41-L49)
- [method] `Start` :: func (s *Scheduler) Start() (L52-L62)
- [method] `Stop` :: func (s *Scheduler) Stop() (L65-L73)
- [method] `loop` :: func (s *Scheduler) loop() (L75-L96)
- [method] `evaluateTick` :: func (s *Scheduler) evaluateTick() (L98-L176)
- [method] `fireSchedule` :: func (s *Scheduler) fireSchedule(sc store.ScheduleDetail, lectureDate time.Time, now time.Time) (L178-L258)
- [method] `TriggerScheduleManually` :: func (s *Scheduler) TriggerScheduleManually(scheduleID int64, dryRun bool) (*store.QueueMessage, error) (L261-L354)
- [method] `calculateNextLectureDate` :: func (s *Scheduler) calculateNextLectureDate(targetDayOfWeek int, fromTime time.Time) time.Time (L356-L363)
- [method] `updateCountdown` :: func (s *Scheduler) updateCountdown() (L365-L461)
- [method] `calculateNextFireTime` :: func (s *Scheduler) calculateNextFireTime(sc *store.ScheduleDetail, now time.Time, loc *time.Location) time.Time (L463-L502)
- [method] `GetNextDeliveryInfo` :: func (s *Scheduler) GetNextDeliveryInfo() *NextDeliveryInfo (L505-L509)

### `internal/scheduler/scheduler_test.go`
- [module] `scheduler_test` :: package scheduler_test (L1-L1)
- [function] `setupTestScheduler` :: func setupTestScheduler(t *testing.T) (*store.Store, *scheduler.Scheduler, *store.User) (L18-L41)
- [function] `TestManualTriggerUsesTodayForSameDayLecture` :: func TestManualTriggerUsesTodayForSameDayLecture(t *testing.T) (L43-L90)
- [function] `containsSubstring` :: func containsSubstring(s, sub string) bool (L92-L99)

### `internal/realtime/hub.go`
- [module] `realtime` :: package realtime (L1-L1)
- [class] `Event` :: type Event struct (L33-L36)
- [class] `ClientConn` :: type ClientConn struct (L39-L42)
- [method] `WriteJSON` :: func (c *ClientConn) WriteJSON(v interface{}) error (L44-L49)
- [method] `WritePing` :: func (c *ClientConn) WritePing() error (L51-L56)
- [method] `WritePong` :: func (c *ClientConn) WritePong(appData []byte) error (L58-L63)
- [method] `Close` :: func (c *ClientConn) Close() error (L65-L69)
- [class] `Hub` :: type Hub struct (L72-L79)
- [function] `NewHub` :: func NewHub() *Hub (L82-L90)
- [method] `Run` :: func (h *Hub) Run() (L93-L152)
- [method] `Broadcast` :: func (h *Hub) Broadcast(eventType string, payload interface{}) (L155-L165)
- [method] `BroadcastToast` :: func (h *Hub) BroadcastToast(level, message string) (L168-L173)
- [method] `HandleWS` :: func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) (L176-L202)

### `internal/store/db.go`
- [module] `store` :: package store (L1-L1)
- [class] `Store` :: type Store struct (L14-L16)
- [function] `New` :: func New(dbPath string) (*Store, error) (L19-L43)
- [method] `DB` :: func (s *Store) DB() *sql.DB (L46-L48)
- [method] `Close` :: func (s *Store) Close() error (L51-L53)
- [method] `migrate` :: func (s *Store) migrate() error (L55-L270)
- [method] `addColumnIfNotExists` :: func (s *Store) addColumnIfNotExists(table, column, colDef string) error (L272-L302)

### `internal/store/models.go`
- [module] `store` :: package store (L1-L1)
- [class] `User` :: type User struct (L9-L19)
- [class] `TwoFactorChallenge` :: type TwoFactorChallenge struct (L22-L27)
- [class] `Session` :: type Session struct (L30-L35)
- [class] `Settings` :: type Settings struct (L38-L48)
- [class] `Contact` :: type Contact struct (L51-L62)
- [class] `Template` :: type Template struct (L65-L73)
- [class] `TemplateVersion` :: type TemplateVersion struct (L76-L82)
- [class] `Schedule` :: type Schedule struct (L85-L107)
- [class] `ScheduleDetail` :: type ScheduleDetail struct (L110-L117)
- [class] `Holiday` :: type Holiday struct (L120-L124)
- [class] `QueueMessage` :: type QueueMessage struct (L127-L142)
- [class] `ActivityLog` :: type ActivityLog struct (L145-L152)
- [class] `DashboardStats` :: type DashboardStats struct (L155-L164)

### `internal/store/repository.go`
- [module] `store` :: package store (L1-L1)
- [method] `CountUsers` :: func (s *Store) CountUsers() (int, error) (L13-L17)
- [method] `CreateUser` :: func (s *Store) CreateUser(username, passwordHash, role string) (*User, error) (L19-L35)
- [method] `GetUserByUsername` :: func (s *Store) GetUserByUsername(username string) (*User, error) (L37-L52)
- [method] `GetUserByID` :: func (s *Store) GetUserByID(id int64) (*User, error) (L54-L69)
- [method] `ListUsers` :: func (s *Store) ListUsers() ([]User, error) (L71-L91)
- [method] `UpdateUser2FA` :: func (s *Store) UpdateUser2FA(userID int64, secret string, enabled bool) error (L93-L101)
- [method] `UpdateUserPassword` :: func (s *Store) UpdateUserPassword(userID int64, passwordHash string) error (L103-L107)
- [method] `DeleteUser` :: func (s *Store) DeleteUser(id int64) error (L109-L112)
- [method] `Create2FAChallenge` :: func (s *Store) Create2FAChallenge(token string, userID int64, expiresAt time.Time) error (L114-L118)
- [method] `Get2FAChallenge` :: func (s *Store) Get2FAChallenge(token string) (*TwoFactorChallenge, error) (L120-L129)
- [method] `Delete2FAChallenge` :: func (s *Store) Delete2FAChallenge(token string) error (L131-L134)
- [method] `CreateSession` :: func (s *Store) CreateSession(token string, userID int64, expiresAt time.Time) error (L136-L140)
- [method] `GetSession` :: func (s *Store) GetSession(token string) (*Session, error) (L142-L151)
- [method] `DeleteSession` :: func (s *Store) DeleteSession(token string) error (L153-L156)
- [method] `CleanExpiredSessions` :: func (s *Store) CleanExpiredSessions() error (L158-L161)
- [method] `GetSettings` :: func (s *Store) GetSettings() (*Settings, error) (L167-L181)
- [method] `UpdateSettings` :: func (s *Store) UpdateSettings(st *Settings) error (L183-L199)
- [method] `SetRegistrationOpen` :: func (s *Store) SetRegistrationOpen(open bool) error (L201-L208)
- [method] `SetGlobalDryRun` :: func (s *Store) SetGlobalDryRun(dryRun bool) error (L210-L217)
- [method] `ListContacts` :: func (s *Store) ListContacts(userID int64, contactType string) ([]Contact, error) (L223-L251)
- [method] `GetContactByID` :: func (s *Store) GetContactByID(id int64) (*Contact, error) (L253-L262)
- [method] `CreateContact` :: func (s *Store) CreateContact(c *Contact) (*Contact, error) (L264-L279)
- [method] `UpdateContact` :: func (s *Store) UpdateContact(c *Contact) error (L281-L287)
- [method] `DeleteContact` :: func (s *Store) DeleteContact(id, userID int64) error (L289-L292)
- [method] `ListTemplates` :: func (s *Store) ListTemplates() ([]Template, error) (L298-L316)
- [method] `GetTemplateByID` :: func (s *Store) GetTemplateByID(id int64) (*Template, error) (L318-L328)
- [method] `GetDefaultTemplate` :: func (s *Store) GetDefaultTemplate() (*Template, error) (L330-L345)
- [method] `CreateTemplate` :: func (s *Store) CreateTemplate(t *Template) (*Template, error) (L347-L369)
- [method] `UpdateTemplate` :: func (s *Store) UpdateTemplate(t *Template) error (L371-L392)
- [method] `DeleteTemplate` :: func (s *Store) DeleteTemplate(id int64) error (L394-L397)
- [method] `ListTemplateVersions` :: func (s *Store) ListTemplateVersions(templateID int64) ([]TemplateVersion, error) (L399-L416)
- [method] `ListSchedules` :: func (s *Store) ListSchedules(userID int64) ([]ScheduleDetail, error) (L422-L475)
- [method] `ListActiveSchedules` :: func (s *Store) ListActiveSchedules() ([]ScheduleDetail, error) (L477-L525)
- [method] `ListPublicSchedules` :: func (s *Store) ListPublicSchedules() ([]ScheduleDetail, error) (L527-L575)
- [method] `GetScheduleByID` :: func (s *Store) GetScheduleByID(id int64) (*Schedule, error) (L577-L602)
- [method] `GetScheduleByIDForUser` :: func (s *Store) GetScheduleByIDForUser(id, userID int64) (*Schedule, error) (L605-L624)
- [method] `GetScheduleDetailByID` :: func (s *Store) GetScheduleDetailByID(id int64) (*ScheduleDetail, error) (L626-L663)
- [method] `CreateSchedule` :: func (s *Store) CreateSchedule(sc *Schedule) (*Schedule, error) (L665-L699)
- [method] `UpdateSchedule` :: func (s *Store) UpdateSchedule(sc *Schedule) error (L701-L725)
- [method] `UpdateScheduleLastSent` :: func (s *Store) UpdateScheduleLastSent(id int64, t time.Time) error (L727-L730)
- [method] `ToggleScheduleActive` :: func (s *Store) ToggleScheduleActive(id, userID int64, active bool) error (L732-L739)
- [method] `DeleteSchedule` :: func (s *Store) DeleteSchedule(id, userID int64) error (L741-L744)
- [method] `ListHolidays` :: func (s *Store) ListHolidays() ([]Holiday, error) (L750-L766)
- [method] `IsHoliday` :: func (s *Store) IsHoliday(dateStr string) (bool, string, error) (L768-L778)
- [method] `CreateHoliday` :: func (s *Store) CreateHoliday(dateStr, description string) error (L780-L783)
- [method] `DeleteHoliday` :: func (s *Store) DeleteHoliday(id int64) error (L785-L788)
- [method] `CreateQueueMessage` :: func (s *Store) CreateQueueMessage(qm *QueueMessage) (*QueueMessage, error) (L794-L813)
- [method] `ListQueueMessages` :: func (s *Store) ListQueueMessages(userID int64, status string, limit int) ([]QueueMessage, error) (L815-L855)
- [method] `GetPendingQueueMessages` :: func (s *Store) GetPendingQueueMessages(now time.Time, limit int) ([]QueueMessage, error) (L857-L884)
- [method] `GetQueueMessageByID` :: func (s *Store) GetQueueMessageByID(id int64) (*QueueMessage, error) (L886-L903)
- [method] `ClaimPendingQueueMessage` :: func (s *Store) ClaimPendingQueueMessage(id int64) (bool, error) (L907-L916)
- [method] `UpdateQueueStatus` :: func (s *Store) UpdateQueueStatus(id int64, status, errorMsg string, sentAt *time.Time) error (L918-L923)
- [method] `IncrementRetry` :: func (s *Store) IncrementRetry(id int64, nextTime time.Time, errorMsg string) error (L925-L931)
- [method] `CancelQueueMessage` :: func (s *Store) CancelQueueMessage(id, userID int64) error (L933-L938)
- [method] `DeleteQueueMessage` :: func (s *Store) DeleteQueueMessage(id, userID int64) error (L940-L943)
- [method] `AddActivityLog` :: func (s *Store) AddActivityLog(category, message, details string) (L949-L952)
- [method] `ListActivityLogs` :: func (s *Store) ListActivityLogs(limit int, category string) ([]ActivityLog, error) (L954-L981)
- [method] `GetDashboardStats` :: func (s *Store) GetDashboardStats(userID int64) (*DashboardStats, error) (L987-L1048)

### `internal/template/engine.go`
- [module] `template` :: package template (L1-L1)
- [class] `Context` :: type Context struct (L26-L39)
- [class] `Engine` :: type Engine struct (L42-L44)
- [function] `NewEngine` :: func NewEngine() *Engine (L47-L65)
- [method] `Render` :: func (e *Engine) Render(tmplStr string, ctx Context) (string, error) (L68-L80)
- [method] `BuildContext` :: func (e *Engine) BuildContext(sc *store.ScheduleDetail, lectureDate time.Time, studentContact *store.Contact) Context (L83-L119)
- [method] `DummyContext` :: func (e *Engine) DummyContext() Context (L122-L144)
- [function] `DayName` :: func DayName(w time.Weekday) string (L147-L153)
- [function] `FormatIndonesianDate` :: func FormatIndonesianDate(t time.Time) string (L156-L161)

### `internal/template/engine_test.go`
- [module] `template` :: package template (L1-L1)
- [function] `TestEngineRender` :: func TestEngineRender(t *testing.T) (L11-L43)
- [function] `TestIndonesianDates` :: func TestIndonesianDates(t *testing.T) (L45-L57)
- [function] `TestBuildContext` :: func TestBuildContext(t *testing.T) (L59-L81)

### `internal/termui/termui.go`
- [module] `termui` :: package termui (L3-L3)
- [class] `UI` :: type UI struct (L37-L45)
- [function] `New` :: func New(cfg *config.Config, s *store.Store, wam *whatsapp.Manager, qm *queue.Manager, sch *scheduler.Scheduler) *UI (L48-L57)
- [method] `Run` :: func (u *UI) Run() (L60-L89)
- [method] `dispatch` :: func (u *UI) dispatch(raw string) (quit bool) (L91-L120)
- [method] `cmdHelp` :: func (u *UI) cmdHelp() (L122-L140)
- [method] `refreshLoop` :: func (u *UI) refreshLoop() (L143-L154)
- [method] `cmdQuit` :: func (u *UI) cmdQuit() (L156-L158)
- [method] `renderStatus` :: func (u *UI) renderStatus() (L161-L192)
- [method] `cmdJadwal` :: func (u *UI) cmdJadwal() (L194-L225)
- [method] `cmdAntrian` :: func (u *UI) cmdAntrian() (L227-L256)
- [method] `cmdKirim` :: func (u *UI) cmdKirim(args []string) (L258-L280)
- [method] `cmdPair` :: func (u *UI) cmdPair(args []string) (L282-L317)
- [method] `cmdAkun` :: func (u *UI) cmdAkun() (L319-L343)
- [method] `systemStats` :: func (u *UI) systemStats() (pending, sentToday, failed int) (L346-L359)
- [function] `truncate` :: func truncate(s string, n int) string (L363-L369)
- [function] `orDash` :: func orDash(s string) string (L371-L376)

### `internal/totp/totp.go`
- [module] `totp` :: package totp (L1-L1)
- [function] `GenerateSecret` :: func GenerateSecret(byteLength int) (string, error) (L28-L41)
- [function] `GeneratePasscode` :: func GeneratePasscode(secret string, t time.Time) (string, error) (L44-L72)
- [function] `ValidatePasscode` :: func ValidatePasscode(secret, passcode string, skew int) bool (L76-L96)
- [function] `GenerateOTPAuthURL` :: func GenerateOTPAuthURL(username, secret, issuer string) string (L99-L112)
- [function] `GenerateQRCodeDataURL` :: func GenerateQRCodeDataURL(username, secret, issuer string) (string, error) (L115-L127)

### `internal/totp/totp_test.go`
- [module] `totp` :: package totp (L1-L1)
- [function] `TestTOTPFlow` :: func TestTOTPFlow(t *testing.T) (L9-L61)

### `internal/tunnel/tunnel.go`
- [module] `tunnel` :: package tunnel (L1-L1)
- [class] `State` :: type State type_identifier (L30-L30)
- [class] `Status` :: type Status struct (L40-L46)
- [class] `Manager` :: type Manager struct (L49-L60)
- [function] `NewManager` :: func NewManager(binDir, port string) *Manager (L65-L72)
- [method] `Status` :: func (m *Manager) Status() Status (L75-L91)
- [method] `Start` :: func (m *Manager) Start(localPort string) error (L94-L170)
- [method] `Stop` :: func (m *Manager) Stop() error (L173-L191)
- [method] `monitorOutput` :: func (m *Manager) monitorOutput(r io.Reader) (L193-L218)
- [method] `ensureCloudflared` :: func (m *Manager) ensureCloudflared() (string, error) (L220-L335)
- [function] `fetchExpectedCloudflaredChecksum` :: func fetchExpectedCloudflaredChecksum(assetName string) (string, error) (L338-L374)
- [function] `copyExecutableFile` :: func copyExecutableFile(src, dst string) error (L376-L393)
- [function] `getCloudflaredDownloadURL` :: func getCloudflaredDownloadURL() (string, error) (L395-L415)

### `internal/updater/restart_unix.go`
- [module] `updater` :: package updater (L3-L3)
- [function] `RestartSelf` :: func RestartSelf() error (L14-L24)

### `internal/updater/restart_windows.go`
- [module] `updater` :: package updater (L3-L3)
- [function] `RestartSelf` :: func RestartSelf() error (L14-L32)

### `internal/updater/updater.go`
- [module] `updater` :: package updater (L1-L1)
- [class] `GitHubRelease` :: type GitHubRelease struct (L25-L33)
- [class] `GitHubAsset` :: type GitHubAsset struct (L36-L42)
- [class] `CheckResult` :: type CheckResult struct (L45-L58)
- [class] `Manager` :: type Manager struct (L61-L66)
- [function] `NewManager` :: func NewManager() *Manager (L69-L75)
- [method] `CheckUpdate` :: func (m *Manager) CheckUpdate(ctx context.Context, force bool) (*CheckResult, error) (L78-L168)
- [function] `CompareVersions` :: func CompareVersions(vA, vB string) int (L176-L196)
- [function] `parseSemver` :: func parseSemver(v string) []int (L198-L213)
- [function] `matchPlatformAsset` :: func matchPlatformAsset(name, goos, goarch string) bool (L215-L242)
- [method] `ApplyBinarySelfUpdate` :: func (m *Manager) ApplyBinarySelfUpdate(ctx context.Context, downloadURL string) error (L245-L402)
- [function] `copyFile` :: func copyFile(src, dst string) error (L404-L421)

### `internal/updater/updater_test.go`
- [module] `updater` :: package updater (L1-L1)
- [function] `TestCompareVersions` :: func TestCompareVersions(t *testing.T) (L7-L29)

### `internal/web/features_test.go`
- [module] `web_test` :: package web_test (L1-L1)
- [function] `setupTestWeb` :: func setupTestWeb(t *testing.T) (http.Handler, *store.Store, *auth.Service, *store.User) (L31-L66)
- [function] `TestPublicSchedulePortalAndICS` :: func TestPublicSchedulePortalAndICS(t *testing.T) (L68-L131)
- [function] `TestRegistrationClosedWhenSuperAdminExists` :: func TestRegistrationClosedWhenSuperAdminExists(t *testing.T) (L133-L162)
- [function] `TestAdminCanCreateNewUserAndIsolatedSession` :: func TestAdminCanCreateNewUserAndIsolatedSession(t *testing.T) (L164-L199)
- [function] `TestTwoFactorAPIFlow` :: func TestTwoFactorAPIFlow(t *testing.T) (L201-L248)
- [function] `TestScheduleCSVImport` :: func TestScheduleCSVImport(t *testing.T) (L252-L309)
- [function] `TestAPITokenLoginEnforces2FA` :: func TestAPITokenLoginEnforces2FA(t *testing.T) (L311-L370)
- [function] `TestLoopbackPairPhoneAuth` :: func TestLoopbackPairPhoneAuth(t *testing.T) (L372-L398)

### `internal/web/handlers.go`
- [module] `web` :: package web (L1-L1)
- [class] `Handlers` :: type Handlers struct (L38-L51)
- [function] `NewHandlers` :: func NewHandlers(
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
) *Handlers (L54-L79)
- [method] `SetStaticFS` :: func (h *Handlers) SetStaticFS(fsys fs.FS) (L82-L84)
- [method] `getActiveWAClient` :: func (h *Handlers) getActiveWAClient(r *http.Request) *whatsapp.Client (L88-L98)
- [method] `getActiveUserID` :: func (h *Handlers) getActiveUserID(r *http.Request) int64 (L101-L107)
- [method] `currentUser` :: func (h *Handlers) currentUser(r *http.Request) *store.User (L110-L112)
- [function] `isPrivileged` :: func isPrivileged(user *store.User) bool (L115-L117)
- [method] `LoginHandler` :: func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) (L123-L142)
- [method] `LoginPostHandler` :: func (h *Handlers) LoginPostHandler(w http.ResponseWriter, r *http.Request) (L144-L184)
- [method] `Login2FAHandler` :: func (h *Handlers) Login2FAHandler(w http.ResponseWriter, r *http.Request) (L186-L199)
- [method] `Login2FAPostHandler` :: func (h *Handlers) Login2FAPostHandler(w http.ResponseWriter, r *http.Request) (L201-L233)
- [method] `RegisterHandler` :: func (h *Handlers) RegisterHandler(w http.ResponseWriter, r *http.Request) (L235-L249)
- [method] `RegisterPostHandler` :: func (h *Handlers) RegisterPostHandler(w http.ResponseWriter, r *http.Request) (L251-L298)
- [method] `LogoutHandler` :: func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) (L300-L307)
- [method] `TwoFactorSetupAPIHandler` :: func (h *Handlers) TwoFactorSetupAPIHandler(w http.ResponseWriter, r *http.Request) (L313-L338)
- [method] `TwoFactorEnableAPIHandler` :: func (h *Handlers) TwoFactorEnableAPIHandler(w http.ResponseWriter, r *http.Request) (L340-L387)
- [method] `TwoFactorDisableAPIHandler` :: func (h *Handlers) TwoFactorDisableAPIHandler(w http.ResponseWriter, r *http.Request) (L389-L433)
- [method] `CreateUserHandler` :: func (h *Handlers) CreateUserHandler(w http.ResponseWriter, r *http.Request) (L439-L476)
- [method] `DeleteUserHandler` :: func (h *Handlers) DeleteUserHandler(w http.ResponseWriter, r *http.Request) (L478-L496)
- [method] `OverviewHandler` :: func (h *Handlers) OverviewHandler(w http.ResponseWriter, r *http.Request) (L502-L553)
- [method] `ContactsHandler` :: func (h *Handlers) ContactsHandler(w http.ResponseWriter, r *http.Request) (L559-L578)
- [method] `CreateContactHandler` :: func (h *Handlers) CreateContactHandler(w http.ResponseWriter, r *http.Request) (L580-L608)
- [method] `UpdateContactHandler` :: func (h *Handlers) UpdateContactHandler(w http.ResponseWriter, r *http.Request) (L610-L635)
- [method] `DeleteContactHandler` :: func (h *Handlers) DeleteContactHandler(w http.ResponseWriter, r *http.Request) (L637-L649)
- [method] `SchedulesHandler` :: func (h *Handlers) SchedulesHandler(w http.ResponseWriter, r *http.Request) (L655-L672)
- [method] `CreateScheduleHandler` :: func (h *Handlers) CreateScheduleHandler(w http.ResponseWriter, r *http.Request) (L674-L731)
- [method] `UpdateScheduleHandler` :: func (h *Handlers) UpdateScheduleHandler(w http.ResponseWriter, r *http.Request) (L733-L792)
- [method] `ToggleScheduleHandler` :: func (h *Handlers) ToggleScheduleHandler(w http.ResponseWriter, r *http.Request) (L794-L808)
- [method] `TriggerScheduleHandler` :: func (h *Handlers) TriggerScheduleHandler(w http.ResponseWriter, r *http.Request) (L810-L833)
- [method] `DeleteScheduleHandler` :: func (h *Handlers) DeleteScheduleHandler(w http.ResponseWriter, r *http.Request) (L835-L847)
- [method] `ImportSchedulesHandler` :: func (h *Handlers) ImportSchedulesHandler(w http.ResponseWriter, r *http.Request) (L853-L1010)
- [function] `normalizePhoneCSV` :: func normalizePhoneCSV(p string) string (L1013-L1021)
- [function] `isValidTimeCSV` :: func isValidTimeCSV(t string) bool (L1024-L1031)
- [method] `TemplatesHandler` :: func (h *Handlers) TemplatesHandler(w http.ResponseWriter, r *http.Request) (L1037-L1070)
- [method] `CreateTemplateHandler` :: func (h *Handlers) CreateTemplateHandler(w http.ResponseWriter, r *http.Request) (L1072-L1097)
- [method] `UpdateTemplateHandler` :: func (h *Handlers) UpdateTemplateHandler(w http.ResponseWriter, r *http.Request) (L1099-L1133)
- [method] `DeleteTemplateHandler` :: func (h *Handlers) DeleteTemplateHandler(w http.ResponseWriter, r *http.Request) (L1135-L1151)
- [method] `PreviewTemplateHandler` :: func (h *Handlers) PreviewTemplateHandler(w http.ResponseWriter, r *http.Request) (L1153-L1167)
- [method] `HistoryHandler` :: func (h *Handlers) HistoryHandler(w http.ResponseWriter, r *http.Request) (L1173-L1188)
- [method] `QueueHandler` :: func (h *Handlers) QueueHandler(w http.ResponseWriter, r *http.Request) (L1194-L1209)
- [method] `QueueSendNowHandler` :: func (h *Handlers) QueueSendNowHandler(w http.ResponseWriter, r *http.Request) (L1211-L1224)
- [method] `QueueCancelHandler` :: func (h *Handlers) QueueCancelHandler(w http.ResponseWriter, r *http.Request) (L1226-L1239)
- [method] `QueueDeleteHandler` :: func (h *Handlers) QueueDeleteHandler(w http.ResponseWriter, r *http.Request) (L1241-L1252)
- [method] `SettingsHandler` :: func (h *Handlers) SettingsHandler(w http.ResponseWriter, r *http.Request) (L1258-L1278)
- [method] `UpdateSettingsHandler` :: func (h *Handlers) UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) (L1280-L1304)
- [method] `CreateHolidayHandler` :: func (h *Handlers) CreateHolidayHandler(w http.ResponseWriter, r *http.Request) (L1306-L1316)
- [method] `DeleteHolidayHandler` :: func (h *Handlers) DeleteHolidayHandler(w http.ResponseWriter, r *http.Request) (L1318-L1325)
- [method] `ToggleRegistrationHandler` :: func (h *Handlers) ToggleRegistrationHandler(w http.ResponseWriter, r *http.Request) (L1327-L1334)
- [method] `BackupDownloadHandler` :: func (h *Handlers) BackupDownloadHandler(w http.ResponseWriter, r *http.Request) (L1336-L1350)
- [method] `ToggleTunnelHandler` :: func (h *Handlers) ToggleTunnelHandler(w http.ResponseWriter, r *http.Request) (L1356-L1373)
- [method] `TunnelStatusHandler` :: func (h *Handlers) TunnelStatusHandler(w http.ResponseWriter, r *http.Request) (L1375-L1378)
- [method] `PublicScheduleHandler` :: func (h *Handlers) PublicScheduleHandler(w http.ResponseWriter, r *http.Request) (L1384-L1396)
- [method] `PublicCalendarICSHandler` :: func (h *Handlers) PublicCalendarICSHandler(w http.ResponseWriter, r *http.Request) (L1398-L1477)
- [method] `LogsHandler` :: func (h *Handlers) LogsHandler(w http.ResponseWriter, r *http.Request) (L1483-L1498)
- [method] `WhatsAppQRHandler` :: func (h *Handlers) WhatsAppQRHandler(w http.ResponseWriter, r *http.Request) (L1504-L1535)
- [method] `WhatsAppReconnectHandler` :: func (h *Handlers) WhatsAppReconnectHandler(w http.ResponseWriter, r *http.Request) (L1537-L1548)
- [method] `WhatsAppDisconnectHandler` :: func (h *Handlers) WhatsAppDisconnectHandler(w http.ResponseWriter, r *http.Request) (L1550-L1557)
- [method] `WhatsAppPairPhoneHandler` :: func (h *Handlers) WhatsAppPairPhoneHandler(w http.ResponseWriter, r *http.Request) (L1559-L1615)
- [method] `WhatsAppTestSendHandler` :: func (h *Handlers) WhatsAppTestSendHandler(w http.ResponseWriter, r *http.Request) (L1617-L1646)
- [method] `WhatsAppGroupsHandler` :: func (h *Handlers) WhatsAppGroupsHandler(w http.ResponseWriter, r *http.Request) (L1649-L1677)
- [method] `WhatsAppContactsHandler` :: func (h *Handlers) WhatsAppContactsHandler(w http.ResponseWriter, r *http.Request) (L1680-L1708)
- [method] `HealthzHandler` :: func (h *Handlers) HealthzHandler(w http.ResponseWriter, r *http.Request) (L1714-L1721)
- [method] `NotFoundHandler` :: func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) (L1727-L1732)
- [method] `InternalServerErrorHandler` :: func (h *Handlers) InternalServerErrorHandler(w http.ResponseWriter, r *http.Request) (L1734-L1739)
- [method] `TermsHandler` :: func (h *Handlers) TermsHandler(w http.ResponseWriter, r *http.Request) (L1741-L1746)
- [method] `PrivacyHandler` :: func (h *Handlers) PrivacyHandler(w http.ResponseWriter, r *http.Request) (L1748-L1753)
- [method] `AboutHandler` :: func (h *Handlers) AboutHandler(w http.ResponseWriter, r *http.Request) (L1755-L1760)
- [method] `ChangelogHandler` :: func (h *Handlers) ChangelogHandler(w http.ResponseWriter, r *http.Request) (L1762-L1767)
- [method] `TerminalHandler` :: func (h *Handlers) TerminalHandler(w http.ResponseWriter, r *http.Request) (L1769-L1774)
- [method] `RobotsHandler` :: func (h *Handlers) RobotsHandler(w http.ResponseWriter, r *http.Request) (L1780-L1791)
- [method] `SitemapHandler` :: func (h *Handlers) SitemapHandler(w http.ResponseWriter, r *http.Request) (L1793-L1843)
- [method] `ManifestHandler` :: func (h *Handlers) ManifestHandler(w http.ResponseWriter, r *http.Request) (L1845-L1849)
- [method] `FaviconHandler` :: func (h *Handlers) FaviconHandler(w http.ResponseWriter, r *http.Request) (L1851-L1854)
- [method] `serveEmbedded` :: func (h *Handlers) serveEmbedded(w http.ResponseWriter, r *http.Request, name string) (L1858-L1869)
- [method] `SystemVersionHandler` :: func (h *Handlers) SystemVersionHandler(w http.ResponseWriter, r *http.Request) (L1872-L1880)
- [method] `CheckUpdateHandler` :: func (h *Handlers) CheckUpdateHandler(w http.ResponseWriter, r *http.Request) (L1883-L1910)
- [method] `DownloadApkHandler` :: func (h *Handlers) DownloadApkHandler(w http.ResponseWriter, r *http.Request) (L1913-L1921)
- [method] `ApplyUpdateHandler` :: func (h *Handlers) ApplyUpdateHandler(w http.ResponseWriter, r *http.Request) (L1924-L1970)
- [function] `isTrustedUpdateHost` :: func isTrustedUpdateHost(host string) bool (L1973-L1979)
- [method] `APITokenLoginHandler` :: func (h *Handlers) APITokenLoginHandler(w http.ResponseWriter, r *http.Request) (L1987-L2045)
- [method] `APIStatsHandler` :: func (h *Handlers) APIStatsHandler(w http.ResponseWriter, r *http.Request) (L2048-L2059)
- [method] `APIWhatsAppStatusHandler` :: func (h *Handlers) APIWhatsAppStatusHandler(w http.ResponseWriter, r *http.Request) (L2062-L2077)
- [method] `APIContactsHandler` :: func (h *Handlers) APIContactsHandler(w http.ResponseWriter, r *http.Request) (L2080-L2090)
- [method] `APISchedulesHandler` :: func (h *Handlers) APISchedulesHandler(w http.ResponseWriter, r *http.Request) (L2093-L2102)
- [method] `APIQueueHandler` :: func (h *Handlers) APIQueueHandler(w http.ResponseWriter, r *http.Request) (L2105-L2126)
- [method] `APIQueueProcessHandler` :: func (h *Handlers) APIQueueProcessHandler(w http.ResponseWriter, r *http.Request) (L2129-L2163)

### `internal/web/renderer.go`
- [module] `web` :: package web (L1-L1)
- [class] `ViewRenderer` :: type ViewRenderer struct (L24-L28)
- [class] `PageData` :: type PageData struct (L31-L40)
- [function] `NewViewRenderer` :: func NewViewRenderer(templateFS fs.FS, diskDir string) *ViewRenderer (L43-L138)
- [method] `Render` :: func (v *ViewRenderer) Render(w http.ResponseWriter, r *http.Request, pageTemplate string, data PageData) (L141-L143)
- [method] `RenderWithStatus` :: func (v *ViewRenderer) RenderWithStatus(w http.ResponseWriter, r *http.Request, pageTemplate string, data PageData, statusCode int) (L146-L189)
- [method] `RenderPlain` :: func (v *ViewRenderer) RenderPlain(w http.ResponseWriter, pageTemplate string, data PageData) (L192-L230)
- [method] `RenderPartial` :: func (v *ViewRenderer) RenderPartial(w http.ResponseWriter, partialTemplate string, data interface{}) (L233-L269)

### `internal/web/renderer_test.go`
- [module] `web` :: package web (L1-L1)
- [function] `TestAllTemplatesRender` :: func TestAllTemplatesRender(t *testing.T) (L14-L179)
- [function] `TestRenderStatusPages` :: func TestRenderStatusPages(t *testing.T) (L181-L201)

### `internal/web/router.go`
- [module] `web` :: package web (L1-L1)
- [function] `devAssetsEnabled` :: func devAssetsEnabled() bool (L24-L26)
- [function] `SetupRouter` :: func SetupRouter(h *Handlers, staticFS fs.FS, staticDiskDir string) http.Handler (L29-L292)
- [function] `FileServer` :: func FileServer(r chi.Router, path string, root http.FileSystem) (L295-L308)

### `internal/web/security.go`
- [module] `web` :: package web (L1-L1)
- [function] `SecurityHeadersMiddleware` :: func SecurityHeadersMiddleware(next http.Handler) http.Handler (L13-L28)
- [function] `MaxBodyBytesMiddleware` :: func MaxBodyBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler (L31-L40)
- [function] `CSRFOriginMiddleware` :: func CSRFOriginMiddleware(next http.Handler) http.Handler (L43-L98)
- [function] `isCSRFExempt` :: func isCSRFExempt(r *http.Request) bool (L102-L116)
- [function] `isAllowedHost` :: func isAllowedHost(clientHost, serverHost string) bool (L118-L141)
- [class] `RateLimiter` :: type RateLimiter struct (L144-L149)
- [function] `NewRateLimiter` :: func NewRateLimiter(limit int, window time.Duration) *RateLimiter (L152-L184)
- [method] `Middleware` :: func (rl *RateLimiter) Middleware(next http.Handler) http.Handler (L187-L218)
- [function] `extractIP` :: func extractIP(r *http.Request) string (L220-L239)

### `internal/web/security_regression_test.go`
- [module] `web_test` :: package web_test (L1-L1)
- [function] `TestMobileAPIAndWSRequireAuth` :: func TestMobileAPIAndWSRequireAuth(t *testing.T) (L15-L36)
- [function] `TestTokenLoginGrantsMobileAPI` :: func TestTokenLoginGrantsMobileAPI(t *testing.T) (L39-L67)
- [function] `TestContactDeletionIsScopedToOwner` :: func TestContactDeletionIsScopedToOwner(t *testing.T) (L70-L128)
- [function] `TestHealthzDoesNotLeakPhone` :: func TestHealthzDoesNotLeakPhone(t *testing.T) (L131-L145)
- [function] `itoa` :: func itoa(n int64) string (L147-L149)

### `internal/web/security_test.go`
- [module] `web` :: package web (L1-L1)
- [function] `TestSecurityHeadersMiddleware` :: func TestSecurityHeadersMiddleware(t *testing.T) (L10-L31)
- [function] `TestCSRFOriginMiddleware` :: func TestCSRFOriginMiddleware(t *testing.T) (L33-L57)
- [function] `TestRateLimiterMiddleware` :: func TestRateLimiterMiddleware(t *testing.T) (L59-L90)

### `internal/version/version.go`
- [module] `version` :: package version (L1-L1)
- [function] `FullVersion` :: func FullVersion() string (L16-L18)

### `internal/whatsapp/client.go`
- [module] `whatsapp` :: package whatsapp (L1-L1)
- [class] `State` :: type State type_identifier (L33-L33)
- [class] `Client` :: type Client struct (L44-L62)
- [function] `NewClient` :: func NewClient(dbPath string, hub *realtime.Hub, port string, userID int64) (*Client, error) (L65-L101)
- [method] `Start` :: func (c *Client) Start() error (L104-L141)
- [method] `listenQR` :: func (c *Client) listenQR(qrChan <-chan whatsmeow.QRChannelItem) (L143-L183)
- [method] `handleEvent` :: func (c *Client) handleEvent(evt interface{}) (L185-L216)
- [method] `keepAliveLoop` :: func (c *Client) keepAliveLoop() (L218-L253)
- [method] `SendMessage` :: func (c *Client) SendMessage(ctx context.Context, to string, message string) error (L256-L300)
- [method] `Disconnect` :: func (c *Client) Disconnect() (L303-L310)
- [method] `Reconnect` :: func (c *Client) Reconnect() error (L313-L316)
- [method] `Logout` :: func (c *Client) Logout() error (L319-L333)
- [method] `Close` :: func (c *Client) Close() (L336-L343)
- [method] `UserID` :: func (c *Client) UserID() int64 (L345-L347)
- [method] `setState` :: func (c *Client) setState(s State) (L349-L359)
- [method] `Status` :: func (c *Client) Status() (State, string, string, string) (L362-L366)
- [method] `PairPhone` :: func (c *Client) PairPhone(ctx context.Context, phone string) (string, error) (L369-L420)
- [method] `GetPairingCode` :: func (c *Client) GetPairingCode() string (L423-L427)
- [class] `GroupItem` :: type GroupItem struct (L430-L435)
- [class] `ContactItem` :: type ContactItem struct (L438-L443)
- [method] `GetJoinedGroups` :: func (c *Client) GetJoinedGroups(ctx context.Context) ([]GroupItem, error) (L446-L482)
- [method] `GetStoredContacts` :: func (c *Client) GetStoredContacts(ctx context.Context) ([]ContactItem, error) (L485-L519)
- [function] `FormatJID` :: func FormatJID(target string) (types.JID, error) (L522-L546)

### `internal/whatsapp/client_test.go`
- [module] `whatsapp` :: package whatsapp (L1-L1)
- [function] `TestFormatJID` :: func TestFormatJID(t *testing.T) (L7-L53)

### `internal/whatsapp/manager.go`
- [module] `whatsapp` :: package whatsapp (L1-L1)
- [class] `Manager` :: type Manager struct (L15-L21)
- [function] `NewManager` :: func NewManager(sessionsDir string, hub *realtime.Hub, port string) *Manager (L24-L41)
- [method] `GetClient` :: func (m *Manager) GetClient(userID int64) (*Client, error) (L44-L96)
- [method] `GetClientIfExists` :: func (m *Manager) GetClientIfExists(userID int64) *Client (L100-L104)
- [method] `CloseClient` :: func (m *Manager) CloseClient(userID int64) (L107-L115)
- [method] `CloseAll` :: func (m *Manager) CloseAll() (L118-L127)

### `scripts/apkbuilder/main.go`
- [module] `main` :: package main (L1-L1)
- [function] `buildMinimalDex` :: func buildMinimalDex() []byte (L25-L104)
- [function] `buildBinaryAXML` :: func buildBinaryAXML() []byte (L107-L229)
- [function] `signAPK` :: func signAPK(apkPath string, entries map[string][]byte) error (L231-L366)
- [class] `pkcs7SignedData` :: type pkcs7SignedData struct (L285-L305)
- [function] `main` :: func main() (L368-L414)

### `scripts/create_issues/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `IssueRequest` :: type IssueRequest struct (L15-L19)
- [class] `IssueResponse` :: type IssueResponse struct (L21-L26)
- [function] `getGitToken` :: func getGitToken() (string, error) (L28-L42)
- [function] `main` :: func main() (L44-L174)

### `scripts/create_release/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseResponse` :: type ReleaseResponse struct (L15-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L194)

### `scripts/debpack/debpack.go`
- [module] `main` :: package main (L1-L1)
- [function] `writeArMember` :: func writeArMember(w *os.File, name string, data []byte) error (L11-L30)
- [function] `main` :: func main() (L32-L90)

### `scripts/generate_assets/main.go`
- [module] `main` :: package main (L1-L1)
- [function] `blend` :: func blend(c1, c2 color.RGBA, t float64) color.RGBA (L29-L42)
- [function] `renderIconImage` :: func renderIconImage(size int) *image.RGBA (L45-L154)
- [function] `savePNG` :: func savePNG(path string, img image.Image) error (L156-L166)
- [function] `saveICO` :: func saveICO(path string, img image.Image) error (L169-L209)
- [function] `generateOGPreview` :: func generateOGPreview(outPath string) error (L212-L279)
- [function] `generateDMGBackground` :: func generateDMGBackground(outPath string) error (L282-L350)
- [function] `main` :: func main() (L352-L372)

### `scripts/publish/main.go`
- [module] `main` :: package main (L12-L12)
- [class] `releaseResponse` :: type releaseResponse struct (L28-L32)
- [function] `getGitToken` :: func getGitToken() (string, error) (L34-L47)
- [function] `main` :: func main() (L49-L190)

### `scripts/publish_v110/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseResponse` :: type ReleaseResponse struct (L16-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L211)

### `scripts/publish_v112/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseResponse` :: type ReleaseResponse struct (L16-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L212)

### `scripts/publish_v111/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseResponse` :: type ReleaseResponse struct (L16-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L212)

### `scripts/publish_v120/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseResponse` :: type ReleaseResponse struct (L16-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L222)

### `scripts/publish_v121/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseResponse` :: type ReleaseResponse struct (L16-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L202)

### `scripts/update_release_apk/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseInfo` :: type ReleaseInfo struct (L13-L20)
- [function] `getGitToken` :: func getGitToken() (string, error) (L22-L36)
- [function] `main` :: func main() (L38-L128)

### `scripts/update_release_v111_assets/main.go`
- [module] `main` :: package main (L1-L1)
- [class] `ReleaseInfo` :: type ReleaseInfo struct (L15-L22)
- [function] `getGitToken` :: func getGitToken() (string, error) (L24-L38)
- [function] `main` :: func main() (L40-L130)

### `web/static/js/app.js`
- [function] `initWebSocket` :: () (L8-L65)
- [function] `scheduleReconnect` :: () (L67-L74)
- [function] `startFallbackPolling` :: () (L76-L80)
- [function] `stopFallbackPolling` :: () (L82-L87)
- [function] `pollStatusFallback` :: () (L89-L117)
- [function] `handleEvent` :: (event) (L119-L146)
- [function] `updateWhatsAppStatus` :: (payload) (L148-L182)
- [function] `updateWhatsAppQR` :: (qrDataUrl) (L184-L197)
- [function] `updateCountdown` :: (info) (L199-L219)
- [function] `renderPairingCode` :: (code) (L378-L389)
- [function] `renderUpdateAvailable` :: (data, isManual) (L446-L491)
- [function] `renderPickerList` :: (items) (L746-L780)
- [function] `selectWAPickerItem` :: (item) (L782-L790)
- [function] `escapeHtml` :: (str) (L808-L811)
- [function] `bootstrapApp` :: () (L814-L818)

### `mobile/android/app/src/main/java/com/sipendosa/app/ActionTask.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `ActionTask` :: public class ActionTask (L5-L93)
- [method] `ActionTask` :: ActionTask(MainActivity activity, int action) (L20-L23)
- [method] `progress` :: ActionTask progress(MainActivity act, int progress) (L25-L29)
- [method] `success` :: ActionTask success(MainActivity act, File file) (L31-L35)
- [method] `failed` :: ActionTask failed(MainActivity act, String error) (L37-L41)
- [method] `startDownload` :: ActionTask startDownload(MainActivity act, String url, String ver) (L43-L47)
- [method] `doDownload` :: ActionTask doDownload(MainActivity act, String url) (L49-L53)
- [method] `doPairCode` :: ActionTask doPairCode(MainActivity act, String phone) (L55-L59)
- [method] `pairCodeResult` :: ActionTask pairCodeResult(MainActivity act, String result) (L61-L65)
- [method] `run` :: void run() (L67-L92)

### `mobile/android/app/src/main/java/com/sipendosa/app/AndroidBridge.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `AndroidBridge` :: public class AndroidBridge (L5-L26)
- [method] `AndroidBridge` :: AndroidBridge(MainActivity activity) (L8-L10)
- [method] `isAndroidApp` :: boolean isAndroidApp() (L12-L15)
- [method] `getAppVersion` :: String getAppVersion() (L17-L20)
- [method] `downloadAndInstallUpdate` :: void downloadAndInstallUpdate(final String apkUrl, final String versionName) (L22-L25)

### `mobile/android/app/src/main/java/com/sipendosa/app/ApkFileProvider.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `ApkFileProvider` :: public class ApkFileProvider (L18-L113)
- [method] `onCreate` :: boolean onCreate() (L21-L24)
- [method] `getFileForUri` :: File getFileForUri(Uri uri) (L26-L60)
- [method] `openFile` :: ParcelFileDescriptor openFile(Uri uri, String mode) (L62-L66)
- [method] `query` :: Cursor query(Uri uri, String[] projection, String selection, String[] selectionArgs, String sortOrder) (L68-L92)
- [method] `getType` :: String getType(Uri uri) (L94-L97)
- [method] `insert` :: Uri insert(Uri uri, ContentValues values) (L99-L102)
- [method] `delete` :: int delete(Uri uri, String selection, String[] selectionArgs) (L104-L107)
- [method] `update` :: int update(Uri uri, ContentValues values, String selection, String[] selectionArgs) (L109-L112)

### `mobile/android/app/src/main/java/com/sipendosa/app/BootCompletedReceiver.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `BootCompletedReceiver` :: public class BootCompletedReceiver (L14-L42)
- [method] `onReceive` :: void onReceive(Context context, Intent intent) (L17-L41)

### `mobile/android/app/src/main/java/com/sipendosa/app/CustomWebChromeClient.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `CustomWebChromeClient` :: public class CustomWebChromeClient (L11-L50)
- [method] `CustomWebChromeClient` :: CustomWebChromeClient(Activity activity) (L14-L16)
- [method] `onConsoleMessage` :: boolean onConsoleMessage(ConsoleMessage consoleMessage) (L18-L23)
- [method] `onJsAlert` :: boolean onJsAlert(WebView view, String url, String message, final JsResult result) (L25-L36)
- [method] `onJsConfirm` :: boolean onJsConfirm(WebView view, String url, String message, final JsResult result) (L38-L49)

### `mobile/android/app/src/main/java/com/sipendosa/app/InternalWebClient.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `InternalWebClient` :: public class InternalWebClient (L10-L45)
- [method] `InternalWebClient` :: InternalWebClient(MainActivity activity) (L14-L16)
- [method] `shouldOverrideUrlLoading` :: boolean shouldOverrideUrlLoading(WebView view, String url) (L18-L28)
- [method] `onPageStarted` :: void onPageStarted(WebView view, String url, Bitmap favicon) (L30-L36)
- [method] `onPageFinished` :: void onPageFinished(WebView view, String url) (L38-L44)

### `mobile/android/app/src/main/java/com/sipendosa/app/JsDialogClickListener.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `JsDialogClickListener` :: public class JsDialogClickListener (L6-L28)
- [method] `JsDialogClickListener` :: JsDialogClickListener(JsResult result, boolean isConfirm) (L10-L13)
- [method] `onClick` :: void onClick(DialogInterface dialog, int which) (L15-L22)
- [method] `onCancel` :: void onCancel(DialogInterface dialog) (L24-L27)

### `mobile/android/app/src/main/java/com/sipendosa/app/LanDialogClickListener.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `LanDialogClickListener` :: public class LanDialogClickListener (L5-L24)
- [method] `LanDialogClickListener` :: LanDialogClickListener(MainActivity activity, int action) (L15-L18)
- [method] `onClick` :: void onClick(DialogInterface dialog, int which) (L20-L23)

### `mobile/android/app/src/main/java/com/sipendosa/app/MainActivity.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `MainActivity` :: public class MainActivity (L56-L791)
- [method] `onCreate` :: void onCreate(Bundle savedInstanceState) (L92-L221)
- [method] `buildTopNavBar` :: void buildTopNavBar() (L223-L296)
- [method] `createTopButton` :: Button createTopButton(String text, int textColor) (L298-L319)
- [method] `onPageStarted` :: void onPageStarted(String url) (L321-L326)
- [method] `onPageNavigated` :: void onPageNavigated(String url) (L328-L332)
- [method] `createProgressDrawable` :: GradientDrawable createProgressDrawable() (L334-L338)
- [method] `buildSplashOverlay` :: void buildSplashOverlay() (L340-L390)
- [method] `onClick` :: void onClick(View v) (L392-L404)
- [method] `showLanDialog` :: void showLanDialog() (L406-L418)
- [method] `showWaPairDialog` :: void showWaPairDialog() (L420-L427)
- [method] `handleDialogAction` :: void handleDialogAction(int action, int which) (L429-L466)
- [method] `promptPhoneNumberPairing` :: void promptPhoneNumberPairing() (L468-L484)
- [method] `requestPairingCode` :: void requestPairingCode(String phone) (L486-L492)
- [method] `requestPairingCodeInternal` :: void requestPairingCodeInternal(String phone) (L494-L540)
- [method] `onPairCodeResult` :: void onPairCodeResult(String result) (L542-L562)
- [method] `run` :: void run() (L564-L588)
- [method] `onServerStatusChecked` :: void onServerStatusChecked(boolean ready) (L590-L611)
- [method] `startApkDownloadAndInstall` :: void startApkDownloadAndInstall(String apkUrl, String versionName) (L616-L632)
- [method] `onDownloadProgress` :: void onDownloadProgress(int progress) (L634-L638)
- [method] `onDownloadSuccess` :: void onDownloadSuccess(File apkFile) (L640-L646)
- [method] `onDownloadError` :: void onDownloadError(String error) (L648-L653)
- [method] `downloadApkInternal` :: void downloadApkInternal(String targetUrl) (L655-L707)
- [method] `installApk` :: void installApk(File apkFile) (L709-L739)
- [method] `getLanIpAddress` :: String getLanIpAddress() (L741-L767)
- [method] `dp` :: int dp(int value) (L769-L775)
- [method] `onBackPressed` :: void onBackPressed() (L777-L790)

### `mobile/android/app/src/main/java/com/sipendosa/app/SiPenDosaServerService.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `SiPenDosaServerService` :: public class SiPenDosaServerService (L30-L291)
- [method] `onCreate` :: void onCreate() (L42-L46)
- [method] `onStartCommand` :: int onStartCommand(Intent intent, int flags, int startId) (L48-L59)
- [method] `acquireLocks` :: void acquireLocks() (L61-L99)
- [method] `releaseLocks` :: void releaseLocks() (L101-L115)
- [method] `createNotificationChannel` :: void createNotificationChannel() (L117-L131)
- [method] `startForegroundServiceNotification` :: void startForegroundServiceNotification() (L133-L156)
- [method] `run` :: void run() (L158-L182)
- [method] `runServerProcessOnce` :: void runServerProcessOnce() (L184-L243)
- [method] `extractAsset` :: void extractAsset(String assetName, File dest) (L245-L263)
- [method] `getRecentLogs` :: List<String> getRecentLogs() (L265-L269)
- [method] `onDestroy` :: void onDestroy() (L271-L285)
- [method] `onBind` :: IBinder onBind(Intent intent) (L287-L290)

### `mobile/android/app/src/main/java/com/sipendosa/app/UpdateUiTask.java`
- [module] `com.sipendosa.app` :: package com.sipendosa.app (L1-L1)
- [class] `UpdateUiTask` :: public class UpdateUiTask (L3-L16)
- [method] `UpdateUiTask` :: UpdateUiTask(MainActivity activity, boolean isReady) (L7-L10)
- [method] `run` :: void run() (L12-L15)
