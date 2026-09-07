package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sipen/internal/auth"
	"sipen/internal/banner"
	"sipen/internal/config"
	"sipen/internal/queue"
	"sipen/internal/realtime"
	"sipen/internal/scheduler"
	"sipen/internal/store"
	"sipen/internal/template"
	"sipen/internal/web"
	"sipen/internal/whatsapp"
	webassets "sipen/web"
)

func main() {
	// Muat Konfigurasi Lingkungan
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Gagal memuat konfigurasi sistem", "error", err)
		os.Exit(1)
	}

	// Tangani Perintah CLI Khusus (seperti 'sipen pair <nomor>')
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
		case "help", "--help", "-h":
			printCliHelp()
			return
		}
	}

	// 1. Inisialisasi Logger Terstruktur
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Tampilkan Banner Terminal Khas SiPenDosa (Ember & Gold)
	banner.PrintBanner(cfg)
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

	// 7. Inisialisasi WhatsApp Engine (whatsmeow)
	banner.LogStep("WHATSAPP", fmt.Sprintf("Menginisialisasi engine WhatsApp di %s...", cfg.WASessionPath))
	waClient, err := whatsapp.NewClient(cfg.WASessionPath, hub, cfg.Port)
	if err != nil {
		slog.Error("Gagal menginisialisasi engine WhatsApp", "error", err)
		os.Exit(1)
	}

	// 8. Inisialisasi Antrian Pesan & Anti-Ban Rate Limiter
	queueMgr := queue.NewManager(appStore, waClient, hub)
	queueMgr.Start()
	defer queueMgr.Stop()

	// 9. Inisialisasi Template Engine & Dynamic Parser
	tmplEngine := template.NewEngine()

	// 10. Inisialisasi Smart Scheduler
	banner.LogStep("SCHEDULER", fmt.Sprintf("Mengaktifkan scheduler zona waktu %s (Jendela: %s - %s)...",
		cfg.DefaultTimezone, cfg.SendWindowStart, cfg.SendWindowEnd))
	sch := scheduler.NewScheduler(appStore, queueMgr, tmplEngine, hub)
	sch.Start()
	defer sch.Stop()

	// 11. Inisialisasi View Renderer & HTTP Handlers
	viewRenderer := web.NewViewRenderer(webassets.Templates(), "web/templates")
	handlers := web.NewHandlers(cfg, appStore, authSvc, waClient, queueMgr, sch, tmplEngine, hub, viewRenderer)

	// 12. Rakit Router HTTP (Chi)
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

	// 13. Sambungkan WhatsApp Secara Latar Belakang (Auto-Reconnect)
	go func() {
		banner.LogStep("WHATSAPP", "Memulai koneksi sesi WhatsApp...")
		if err := waClient.Start(); err != nil {
			slog.Warn("WhatsApp belum tertaut atau sesi terputus. Silakan scan QR via terminal atau web.", "detail", err)
		}
	}()

	// 14. Jalankan Server HTTP di Goroutine Terpisah
	go func() {
		banner.LogStep("SYSTEM", fmt.Sprintf("HTTP Web Dashboard aktif melayani di http://%s", serverAddr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server HTTP berhenti secara tidak normal", "error", err)
			os.Exit(1)
		}
	}()

	// 15. Tangani Sinyal Shutdown (Graceful Termination)
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

	// Putuskan koneksi WhatsApp secara aman
	waClient.Close()

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
	fmt.Println("==================================================================\n")
}

func printCliHelp() {
	fmt.Println("SiPenDosa — Advanced Automated WhatsApp Reminder Daemon")
	fmt.Println("\nPenggunaan:")
	fmt.Println("  sipen                   Menjalankan server daemon utama")
	fmt.Println("  sipen pair <nomor_hp>   Minta kode pairing login WhatsApp via nomor HP")
	fmt.Println("  sipen help              Menampilkan bantuan ini")
	fmt.Println("\nContoh:")
	fmt.Println("  sipen pair 08123456789")
}
