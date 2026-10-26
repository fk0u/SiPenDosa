package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"sipen/internal/auth"
	"sipen/internal/banner"
	"sipen/internal/config"
	"sipen/internal/netutil"
	"sipen/internal/queue"
	"sipen/internal/realtime"
	"sipen/internal/scheduler"
	"sipen/internal/store"
	"sipen/internal/template"
	"sipen/internal/termui"
	"sipen/internal/tunnel"
	"sipen/internal/web"
	"sipen/internal/whatsapp"
	webassets "sipen/web"
)

// ANSI codes untuk output CLI
const (
	bold  = "\033[1m"
	reset = "\033[0m"
)

func main() {
	// Inisialisasi Lingkungan Jaringan (DNS Fallback & Root CA untuk Android/Embedded)
	netutil.SetupNetworkEnvironment()

	// Muat Konfigurasi Lingkungan
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Gagal memuat konfigurasi sistem", "error", err)
		os.Exit(1)
	}

	// Tangani Perintah CLI Khusus (seperti 'sipen pair <nomor>')
	forcedMode := ""
	if len(os.Args) > 1 {
		cmd := strings.ToLower(os.Args[1])
		switch cmd {
		case "pair":
			if len(os.Args) < 3 {
				fmt.Println("Penggunaan: sipen pair <nomor_whatsapp>")
				fmt.Println("Contoh: sipen pair 08123456789")
				os.Exit(1)
			}
			runPairCli(cfg, os.Args[2])
			return
		case "web", "serve":
			forcedMode = config.UIWeb
		case "desktop":
			forcedMode = config.UIDesktop
		case "mode":
			// sipen mode [terminal|web|desktop] — lihat/mengubah mode akses
			runModeCli(cfg, os.Args[2:])
			return
		case "help", "--help", "-h":
			printCliHelp()
			return
		}
	}

	// Resolusi mode akses: argumen CLI > SIPEN_UI (.env) > wizard first-run
	mode := resolveUIMode(cfg, forcedMode)
	cfg.UIMode = mode

	// 1. Inisialisasi Logger Terstruktur
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Tampilkan Banner — penuh untuk web/desktop, ringkas untuk terminal
	if forcedMode == "" && (cfg.UIMode == config.UITerminal || cfg.UIMode == "") {
		banner.PrintCompact(cfg)
	} else {
		banner.PrintBanner(cfg)
	}
	banner.LogStep("DAEMON", "Memulai inisialisasi subsistem SiPenDosa...")

	// 4. Inisialisasi SQLite Database Store (Zero-CGO Pure-Go WAL)
	banner.LogStep("DATABASE", fmt.Sprintf("Membuka database penyimpanan di %s...", cfg.DBPath))
	appStore, err := store.New(cfg.DBPath)
	if err != nil {
		slog.Error("Gagal menginisialisasi database store", "error", err)
		os.Exit(1)
	}
	defer appStore.Close()

	// 5. Inisialisasi Service Autentikasi Pengguna
	authSvc := auth.NewService(appStore)

	// 6. Inisialisasi WebSocket Realtime Hub
	hub := realtime.NewHub()
	go hub.Run()

	// 7. Inisialisasi WhatsApp Multi-Account Engine (whatsmeow)
	banner.LogStep("WHATSAPP", fmt.Sprintf("Menginisialisasi WhatsApp Multi-Account Engine di %s...", cfg.WASessionPath))
	waManager := whatsapp.NewManager(cfg.WASessionPath, hub, cfg.Port)

	// 8. Inisialisasi Cloudflare Quick Tunnel Manager (Issue #3)
	tunnelMgr := tunnel.NewManager("data/bin", cfg.Port)

	// 9. Inisialisasi Antrian Pesan & Anti-Ban Rate Limiter (Multi-Account)
	queueMgr := queue.NewManager(appStore, waManager, hub)
	queueMgr.Start()
	defer queueMgr.Stop()

	// 10. Inisialisasi Template Engine & Dynamic Parser
	tmplEngine := template.NewEngine()

	// 11. Inisialisasi Smart Scheduler
	banner.LogStep("SCHEDULER", fmt.Sprintf("Mengaktifkan scheduler zona waktu %s (Jendela: %s - %s)...",
		cfg.DefaultTimezone, cfg.SendWindowStart, cfg.SendWindowEnd))
	sch := scheduler.NewScheduler(appStore, queueMgr, tmplEngine, hub)
	sch.Start()
	defer sch.Stop()

	// 14. Sambungkan WhatsApp Superadmin Default (User ID 1) Secara Latar Belakang
	go func() {
		banner.LogStep("WHATSAPP", "Memulai koneksi sesi WhatsApp akun utama...")
		client1, err := waManager.GetClient(1)
		if err != nil {
			slog.Warn("Gagal memuat sesi WhatsApp akun utama", "detail", err)
			return
		}
		if err := client1.Start(); err != nil {
			slog.Warn("WhatsApp belum tertaut atau sesi terputus. Silakan scan QR via terminal atau web.", "detail", err)
		}
	}()

	// ===== MODE TERMINAL: tanpa HTTP sama sekali =====
	if mode == config.UITerminal {
		banner.LogStep("MODE", "Mode akses: TERMINAL — dashboard web dimatikan (jalankan 'sipen web' bila diperlukan).")
		ui := termui.New(cfg, appStore, waManager, queueMgr, sch)

		// Tetap tanggapi sinyal agar shutdown rapi saat Ctrl+C di tengah prompt
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println()
			banner.LogStep("DAEMON", "Menerima sinyal terminasi. Mematikan subsistem...")
			queueMgr.Stop()
			sch.Stop()
			waManager.CloseAll()
			appStore.Close()
			os.Exit(0)
		}()

		ui.Run()
		return
	}

	// ===== MODE WEB / DESKTOP: HTTP dashboard aktif =====
	// 12. Inisialisasi View Renderer & HTTP Handlers
	viewRenderer := web.NewViewRenderer(webassets.Templates(), "web/templates")
	handlers := web.NewHandlers(cfg, appStore, authSvc, waManager, tunnelMgr, queueMgr, sch, tmplEngine, hub, viewRenderer)

	// 13. Rakit Router HTTP (Chi)
	router := web.SetupRouter(handlers, webassets.Static(), "web/static")

	serverAddr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:              serverAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 15. Jalankan Server HTTP di Goroutine Terpisah
	go func() {
		banner.LogStep("SYSTEM", fmt.Sprintf("HTTP Web Dashboard aktif melayani di http://%s", serverAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server HTTP berhenti secara tidak normal", "error", err)
			os.Exit(1)
		}
	}()

	// 15a. Mode desktop: buka jendela aplikasi desktop (opsional)
	if mode == config.UIDesktop {
		go func() {
			time.Sleep(800 * time.Millisecond) // beri waktu server siap
			openDesktopApp(serverAddr)
		}()
	}

	// 15b. Bersihkan sesi & tantangan 2FA kedaluwarsa setiap 24 jam
	ctx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := appStore.CleanExpiredSessions(); err != nil {
					slog.Warn("Gagal membersihkan sesi kedaluwarsa", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 16. Tangani Sinyal Shutdown (Graceful Termination)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	banner.LogStep("DAEMON", fmt.Sprintf("Menerima sinyal terminasi (%v). Menjalankan proses shutdown yang aman...", sig))

	// Konteks batas waktu shutdown 10 detik
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Gagal mematikan server HTTP secara mulus", "error", err)
	}

	// Matikan WhatsApp dan Tunnel secara aman
	waManager.CloseAll()
	_ = tunnelMgr.Stop()

	banner.LogStep("DAEMON", "Seluruh subsistem SiPenDosa berhasil dimatikan dengan aman. Sampai jumpa!")
}

func runPairCli(cfg *config.Config, phone string) {
	fmt.Printf("\n⚡ Menghubungi server SiPenDosa di port %s...\n", cfg.Port)
	payload, _ := json.Marshal(map[string]string{"phone": phone})

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s/api/internal/pair-phone", cfg.Port), bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Gagal membuat request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("\n❌ Gagal terhubung ke daemon server SiPenDosa!")
		fmt.Println("Pastikan server sedang berjalan di latar belakang:")
		fmt.Println("  1. Di Termux: jalankan 'sipen-start' atau 'sipen-bg'")
		fmt.Println("  2. Lalu jalankan kembali: 'sipen pair " + phone + "'")
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Gagal membaca respon server: %v\n", err)
		os.Exit(1)
	}

	if !result.Success || result.Code == "" {
		fmt.Printf("\n❌ Gagal meminta kode pairing: %s\n", result.Error)
		os.Exit(1)
	}

	codeFormatted := result.Code
	if len(codeFormatted) == 8 && !strings.Contains(codeFormatted, "-") {
		codeFormatted = codeFormatted[:4] + " - " + codeFormatted[4:]
	}

	fmt.Println("\n==================================================================")
	fmt.Println("⚡ KODE PAIRING WHATSAPP BERHASIL DIBUAT!")
	fmt.Println("==================================================================")
	fmt.Printf("• Nomor WhatsApp : %s\n", phone)
	fmt.Printf("• KODE PAIRING   : \033[1;33m[ %s ]\033[0m\n", codeFormatted)
	fmt.Println("==================================================================")
	fmt.Println("Petunjuk Login di Aplikasi WhatsApp HP Anda:")
	fmt.Println("1. Buka aplikasi WhatsApp di HP Anda.")
	fmt.Println("2. Ketuk Titik Tiga (⋮) > Perangkat Tertaut > Tautkan Perangkat.")
	fmt.Println("3. Di bawah pemindai kamera, ketuk: 'Tautkan dengan nomor telepon saja'.")
	fmt.Printf("4. Masukkan 8 karakter kode di atas: %s\n", result.Code)
	fmt.Println("5. Selesai! WhatsApp akan otomatis tersambung ke SiPenDosa.")
	fmt.Println("==================================================================")
}

func printCliHelp() {
	fmt.Println("SiPenDosa — Sistem Pengingat Dosen Saatnya (Advanced WhatsApp Reminder Daemon)")
	fmt.Println("\nPenggunaan:")
	fmt.Println("  sipen                     Jalankan sesuai mode akses tersimpan (SIPEN_UI; default terminal)")
	fmt.Println("  sipen terminal            Paksa mode terminal interaktif (tanpa web)")
	fmt.Println("  sipen web                 Paksa mode web (daemon + dashboard)")
	fmt.Println("  sipen desktop             Mode web + buka jendela aplikasi desktop")
	fmt.Println("  sipen mode [terminal|web|desktop]  Lihat atau ubah mode akses tersimpan")
	fmt.Println("  sipen pair <nomor_hp>     Minta kode pairing WhatsApp via nomor HP (butuh mode web)")
	fmt.Println("  sipen help                Tampilkan bantuan ini")
	fmt.Println("\nContoh:")
	fmt.Println("  sipen pair 08123456789")
	fmt.Println("  SIPEN_UI=desktop sipen")
}

// isTTY melaporkan apakah stdin adalah terminal interaktif
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// resolveUIMode menentukan mode akses efektif:
// argumen CLI > SIPEN_UI di .env > wizard first-run (TTY) > migrasi instalasi lama (web) > terminal
func resolveUIMode(cfg *config.Config, forced string) string {
	switch forced {
	case config.UITerminal, config.UIWeb, config.UIDesktop:
		return forced
	}
	switch cfg.UIMode {
	case config.UITerminal, config.UIWeb, config.UIDesktop:
		return cfg.UIMode
	}

	// Lingkungan Android / Embedded selalu berjalan dalam mode web
	if netutil.IsAndroid() {
		return config.UIWeb
	}

	// Nilai SIPEN_UI belum tersimpan.
	if isTTY() {
		return runFirstRunWizard(cfg)
	}

	// Non-interaktif: instalasi lama (sudah punya .env) dipertahankan di mode web;
	// instalasi fresh tanpa TTY jatuh ke terminal sesuai desain produk.
	if cfg.EnvFile != "" {
		if _, err := os.Stat(cfg.EnvFile); err == nil {
			banner.LogStep("MODE", "SIPEN_UI belum diatur — instalasi lama dipertahankan di mode WEB. Jalankan 'sipen mode' untuk mengubah.")
			return config.UIWeb
		}
	}
	banner.LogStep("MODE", "SIPEN_UI belum diatur & tanpa terminal interaktif — memakai mode TERMINAL.")
	return config.UITerminal
}

// runFirstRunWizard menanyakan mode akses pada jalankan pertama dan menyimpannya ke .env
func runFirstRunWizard(cfg *config.Config) string {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("  │  %sSELAMAT DATANG DI SIPENDOSA%s — pilih mode akses Anda:        │\n", bold, reset)
	fmt.Println("  │                                                             │")
	fmt.Printf("  │  %s[1]%s Terminal saja   — konsol interaktif, tanpa web         │\n", bold, reset)
	fmt.Printf("  │  %s[2]%s Web           — dashboard browser di jaringan lokal    │\n", bold, reset)
	fmt.Printf("  │  %s[3]%s Desktop + Web  — dashboard + jendela aplikasi desktop  │\n", bold, reset)
	fmt.Println("  │                                                             │")
	fmt.Println("  └─────────────────────────────────────────────────────────────┘")
	fmt.Printf("  Pilihan %s[1-3]%s (default 1): ", bold, reset)

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	mode := config.UITerminal
	switch choice {
	case "2":
		mode = config.UIWeb
	case "3":
		mode = config.UIDesktop
	}

	// Simpan pilihan ke .env agar tidak ditanya lagi
	if err := persistUIMode(cfg, mode); err != nil {
		banner.LogStep("MODE", fmt.Sprintf("Mode %s dipakai untuk sesi ini (gagal menyimpan ke .env: %v).", mode, err))
	} else {
		banner.LogStep("MODE", fmt.Sprintf("Mode akses %s%s%s tersimpan di .env — ubah kapan saja lewat 'sipen mode'.", bold, mode, reset))
	}
	return mode
}

// persistUIMode menulis/memperbarui baris SIPEN_UI pada file .env
func persistUIMode(cfg *config.Config, mode string) error {
	if cfg.EnvFile == "" {
		return fmt.Errorf("lokasi .env tidak diketahui")
	}
	content, err := os.ReadFile(cfg.EnvFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = []byte("")
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "SIPEN_UI=") {
			lines[i] = "SIPEN_UI=" + mode
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "", "# Mode akses antarmuka: terminal | web | desktop", "SIPEN_UI="+mode)
	}
	if err := os.WriteFile(cfg.EnvFile, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		return err
	}
	cfg.UIMode = mode
	return nil
}

// runModeCli: 'sipen mode' — tampilkan, atau ubah mode akses tersimpan
func runModeCli(cfg *config.Config, args []string) {
	if len(args) == 0 {
		fmt.Printf("Mode akses saat ini: %s%s%s\n", bold, cfg.UIMode+" (belum tersimpan — akan ditanya saat first-run)", reset)
		fmt.Println("Ubah dengan: sipen mode <terminal|web|desktop>")
		return
	}
	mode := strings.ToLower(args[0])
	switch mode {
	case config.UITerminal, config.UIWeb, config.UIDesktop:
	default:
		fmt.Printf("Mode tidak dikenal: %q (pilihan: terminal, web, desktop)\n", mode)
		os.Exit(1)
	}
	if err := persistUIMode(cfg, mode); err != nil {
		fmt.Printf("Gagal menyimpan mode: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Mode akses tersimpan: %s%s%s — berlaku pada jalankan berikutnya.\n", bold, mode, reset)
}

// openDesktopApp membuka jendela aplikasi desktop sesuai platform.
// macOS: aplikasi companion SiPenDosa.app (jendela WKWebView native).
// Platform lain: jendela app-mode Edge/Chrome (tanpa chrome browser UI).
func openDesktopApp(serverAddr string) {
	url := "http://" + serverAddr
	switch runtime.GOOS {
	case "darwin":
		// Buka aplikasi companion bila terpasang; ia mendeteksi daemon yang
		// sudah berjalan dan menampilkan jendela desktopnya sendiri.
		for _, app := range []string{"SiPenDosa", "SiPenDosa.app"} {
			if err := exec.Command("open", "-a", app).Start(); err == nil {
				return
			}
		}
		openBrowserAppMode(url)
	case "windows":
		openBrowserAppMode(url)
	default:
		openBrowserAppMode(url)
	}
}

// openBrowserAppMode mencoba Edge/Chrome dalam mode --app (jendela mandiri);
// bila tidak ada, buka browser default biasa.
func openBrowserAppMode(url string) {
	candidates := [][]string{
		{"msedge", "--app=" + url},
		{"microsoft-edge", "--app=" + url},
		{"google-chrome", "--app=" + url},
		{"chromium", "--app=" + url},
		{"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", "--app=" + url},
		{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--app=" + url},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err == nil {
			if err := exec.Command(c[0], c[1:]...).Start(); err == nil {
				return
			}
		}
	}
	// Fallback: browser default
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "windows":
		_ = exec.Command("cmd", "/c", "start", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}
