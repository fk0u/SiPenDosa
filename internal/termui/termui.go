// Package termui menyediakan antarmuka terminal interaktif SiPenDosa —
// mode akses default aplikasi: dasbor langsung di konsol tanpa web.
package termui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sipen/internal/config"
	"sipen/internal/queue"
	"sipen/internal/scheduler"
	"sipen/internal/store"
	"sipen/internal/whatsapp"
)

// ANSI helpers
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
)

var dayNames = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

// UI merangkum dependensi antarmuka terminal
type UI struct {
	cfg      *config.Config
	store    *store.Store
	waMgr    *whatsapp.Manager
	queueMgr *queue.Manager
	sched    *scheduler.Scheduler

	stopRefresh chan struct{}
}

// New membuat instance antarmuka terminal
func New(cfg *config.Config, s *store.Store, wam *whatsapp.Manager, qm *queue.Manager, sch *scheduler.Scheduler) *UI {
	return &UI{
		cfg:         cfg,
		store:       s,
		waMgr:       wam,
		queueMgr:    qm,
		sched:       sch,
		stopRefresh: make(chan struct{}),
	}
}

// Run menjalankan sesi terminal interaktif hingga pengguna keluar.
func (u *UI) Run() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Printf("  %s%sMODE TERMINAL%s — akses penuh tanpa web. Ketik %s%shelp%s%s untuk daftar perintah.\n",
		bold, cyan, reset, bold, green, reset, reset)
	fmt.Println()

	// Refresh status periodik di latar belakang
	go u.refreshLoop()

	for {
		fmt.Printf("%s%ssipen>%s ", bold, magenta, reset)
		line, err := reader.ReadString('\n')
		if err != nil {
			// stdin ditutup (mis. Ctrl+D) — keluar dengan rapi
			fmt.Println()
			u.cmdQuit()
			return
		}
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		quit := u.dispatch(cmd)
		if quit {
			return
		}
	}
}

func (u *UI) dispatch(raw string) (quit bool) {
	parts := strings.Fields(raw)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "help", "?":
		u.cmdHelp()
	case "status", "s":
		u.renderStatus()
	case "jadwal", "j":
		u.cmdJadwal()
	case "antrian", "a":
		u.cmdAntrian()
	case "kirim":
		u.cmdKirim(args)
	case "pair":
		u.cmdPair(args)
	case "akun":
		u.cmdAkun()
	case "clear", "cls":
		fmt.Print("\033[2J\033[H")
	case "quit", "exit", "q":
		u.cmdQuit()
		return true
	default:
		fmt.Printf("  %sPerintah tidak dikenal: %q%s — ketik %shelp%s untuk bantuan.\n", red, cmd, reset, bold, reset)
	}
	return false
}

func (u *UI) cmdHelp() {
	fmt.Println()
	fmt.Printf("  %s%sPerintah SiPenDosa%s\n", bold, cyan, reset)
	rows := [][2]string{
		{"status", "Tampilkan status engine, WhatsApp, dan countdown"},
		{"jadwal", "Daftar jadwal pengingat aktif"},
		{"antrian", "Lihat isi antrian pesan terbaru"},
		{"kirim <id>", "Picu pengiriman jadwal ber-ID sekarang"},
		{"pair <nomor>", "Minta kode pairing 8-digit WhatsApp"},
		{"akun", "Daftar akun kelas & status sesi WA-nya"},
		{"clear", "Bersihkan layar"},
		{"quit", "Matikan SiPenDosa dan keluar"},
	}
	for _, r := range rows {
		fmt.Printf("    %s%-13s%s %s\n", green, r[0], reset, dim+r[1]+reset)
	}
	fmt.Printf("  %sTips: mode akses bisa diganti kapan saja lewat file .env (SIPEN_UI=terminal|web|desktop).%s\n", dim, reset)
	fmt.Println()
}

// refreshLoop menampilkan status ringkas setiap 60 detik saat pengguna idle
func (u *UI) refreshLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			u.renderStatus()
		case <-u.stopRefresh:
			return
		}
	}
}

func (u *UI) cmdQuit() {
	fmt.Printf("\n  %sSiPenDosa dimatikan dengan rapi. Sampai jumpa!%s\n\n", cyan, reset)
}

// renderStatus mencetak dasbor ringkas ke terminal
func (u *UI) renderStatus() {
	loc := u.cfg.GetLocation()
	now := time.Now().In(loc)

	fmt.Println()
	fmt.Printf("  %s%sSTATUS ENGINE — %s%s\n", bold, cyan, now.Format("Mon, 02 Jan 2006 15:04"), reset)

	// WhatsApp per akun
	userCount, _ := u.store.CountUsers()
	fmt.Printf("  %s●%s Akun terdaftar     : %s%d%s\n", green, reset, bold, userCount, reset)

	// Countdown pengiriman berikutnya
	if info := u.sched.GetNextDeliveryInfo(); info != nil {
		fmt.Printf("  %s●%s Kirim berikutnya  : %s%s%s\n", yellow, reset, bold+yellow, info.RemainingHuman, reset)
		fmt.Printf("    Mata kuliah      : %s\n", info.Matkul)
		fmt.Printf("    Target waktu     : %s\n", info.TargetTime)
	} else {
		fmt.Printf("  %s●%s Kirim berikutnya  : %stidak ada jadwal aktif%s\n", dim, reset, dim, reset)
	}

	// Statistik antrian (agregat seluruh akun)
	if pending, sentToday, failed := u.systemStats(); pending >= 0 {
		fmt.Printf("  %s●%s Antrian menunggu  : %s%d%s | Terkirim hari ini: %s%d%s | Gagal: %s%d%s\n",
			blue, reset, bold, pending, reset,
			green+bold, sentToday, reset,
			red+bold, failed, reset)
	}

	fmt.Printf("  %s●%s Zona waktu        : %s%s%s\n", cyan, reset, bold, u.cfg.DefaultTimezone, reset)
	fmt.Printf("  %s●%s Dry-run global     : %s%v%s\n", cyan, reset, bold, u.cfg.GlobalDryRun, reset)
	fmt.Println()
}

func (u *UI) cmdJadwal() {
	// Agregasi jadwal seluruh akun (mode terminal = mode admin sistem)
	users, err := u.store.ListUsers()
	if err != nil {
		fmt.Printf("  %sGagal membaca akun: %v%s\n", red, err, reset)
		return
	}

	fmt.Println()
	fmt.Printf("  %s%sJADWAL PENGINGAT AKTIF%s\n", bold, cyan, reset)
	shown := 0
	for _, usr := range users {
		schedules, _ := u.store.ListSchedules(usr.ID)
		for _, sc := range schedules {
			if !sc.IsActive {
				continue
			}
			shown++
			fmt.Printf("  %s[%s#%d%s]%s %s%-28s%s %s%s %s-%s %-5s  %s→ %s%s\n",
				dim, reset, sc.ID, dim, reset,
				bold, truncate(sc.Matkul, 28), reset,
				green, dayNames[sc.DayOfWeek%7], reset, sc.StartTime, sc.EndTime,
				dim, sc.RecipientName, reset)
			fmt.Printf("        %sMode %s · kirim %s · lokasi %s%s\n",
				dim, sc.Mode, sc.SendAtTime, orDash(sc.Location), reset)
		}
	}
	if shown == 0 {
		fmt.Printf("  %sBelum ada jadwal aktif. Tambahkan lewat web (sipen web) lalu buka /schedules.%s\n", dim, reset)
	}
	fmt.Println()
}

func (u *UI) cmdAntrian() {
	users, _ := u.store.ListUsers()
	fmt.Println()
	fmt.Printf("  %s%sANTRIAN PESAN (20 TERBARU)%s\n", bold, cyan, reset)
	shown := 0
	for _, usr := range users {
		msgs, _ := u.store.ListQueueMessages(usr.ID, "", 20)
		for _, m := range msgs {
			shown++
			color := cyan
			switch m.Status {
			case "sent":
				color = green
			case "failed":
				color = red
			case "pending":
				color = yellow
			}
			fmt.Printf("  %s[%s#%d%s]%s %-9s %s%-22s%s %s\n",
				dim, reset, m.ID, dim, reset,
				color+bold+m.Status+reset,
				bold, truncate(m.RecipientName, 22), reset,
				dim+truncate(strings.ReplaceAll(m.Message, "\n", " "), 40)+reset)
		}
	}
	if shown == 0 {
		fmt.Printf("  %sAntrian kosong.%s\n", dim, reset)
	}
	fmt.Println()
}

func (u *UI) cmdKirim(args []string) {
	if len(args) == 0 {
		fmt.Printf("  Penggunaan: %skirim <id_jadwal>%s (lihat id lewat %sjadwal%s)\n", bold, reset, bold, reset)
		return
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Printf("  %sID tidak valid: %s%s\n", red, args[0], reset)
		return
	}

	if _, err := u.store.GetScheduleByID(id); err != nil {
		fmt.Printf("  %sJadwal #%d tidak ditemukan.%s\n", red, id, reset)
		return
	}

	msg, err := u.sched.TriggerScheduleManually(id, false)
	if err != nil {
		fmt.Printf("  %sGagal memicu jadwal: %v%s\n", red, err, reset)
		return
	}
	fmt.Printf("  %s✓ Pesan dimasukkan antrian (ID: %d) — anti-ban rate limiter aktif.%s\n", green, msg.ID, reset)
}

func (u *UI) cmdPair(args []string) {
	if len(args) == 0 {
		fmt.Printf("  Penggunaan: %spair <nomor_whatsapp>%s\n", bold, reset)
		return
	}
	phone := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("  %sMeminta kode pairing untuk %s ...%s\n", cyan, phone, reset)
	client, err := u.waMgr.GetClient(1)
	if err != nil {
		fmt.Printf("  %sGagal menyiapkan sesi WA: %v%s\n", red, err, reset)
		return
	}
	code, err := client.PairPhone(ctx, phone)
	if err != nil {
		fmt.Printf("  %sGagal: %v%s\n", red, err, reset)
		return
	}
	if code == "" {
		code = client.GetPairingCode()
	}
	if code == "" {
		fmt.Printf("  %sKode belum tersedia — coba lagi beberapa saat.%s\n", yellow, reset)
		return
	}
	formatted := code
	if len(formatted) == 8 {
		formatted = formatted[:4] + " - " + formatted[4:]
	}
	fmt.Println()
	fmt.Printf("  %s%sKODE PAIRING ANDA:%s  %s%s%s\n", bold, green, reset, bold+yellow, formatted, reset)
	fmt.Printf("  %sMasukkan kode ini di WhatsApp: Perangkat tertaut → Tautkan dengan nomor telepon saja.%s\n", dim, reset)
	fmt.Println()
}

func (u *UI) cmdAkun() {
	users, _ := u.store.ListUsers()
	fmt.Println()
	fmt.Printf("  %s%sAKUN KELAS & SESI WHATSAPP%s\n", bold, cyan, reset)
	for _, usr := range users {
		client := u.waMgr.GetClientIfExists(usr.ID)
		var state whatsapp.State
		var phone, push string
		if client != nil {
			state, phone, push, _ = client.Status()
		}
		stateStr := dim + "tidak aktif" + reset
		if state == whatsapp.StateConnected {
			stateStr = green + "terhubung" + reset + " " + dim + phone + reset
		} else if state != "" {
			stateStr = yellow + string(state) + reset
		}
		fmt.Printf("  %s[%s#%d%s]%s %-16s %-9s %s\n",
			dim, reset, usr.ID, dim, reset,
			bold+usr.Username+reset,
			dim+usr.Role+reset, stateStr)
		_ = push
	}
	fmt.Println()
}

// systemStats mengagregasi statistik seluruh akun; pending=-1 saat gagal
func (u *UI) systemStats() (pending, sentToday, failed int) {
	users, err := u.store.ListUsers()
	if err != nil {
		return -1, 0, 0
	}
	for _, usr := range users {
		if stats, err := u.store.GetDashboardStats(usr.ID); err == nil && stats != nil {
			pending += stats.TotalPending
			sentToday += stats.TotalSentToday
			failed += stats.TotalFailedAll
		}
	}
	return pending, sentToday, failed
}

// ---------- util ----------

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
