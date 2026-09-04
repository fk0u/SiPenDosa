package banner

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"sipen/internal/config"
)

// ANSI color codes
const (
	Reset        = "\033[0m"
	Bold         = "\033[1m"
	Dim          = "\033[2m"
	Cyan         = "\033[36m"
	BrightCyan   = "\033[96m"
	Magenta      = "\033[35m"
	BrightPurple = "\033[95m"
	Green        = "\033[32m"
	BrightGreen  = "\033[92m"
	Yellow       = "\033[33m"
	BrightYellow = "\033[93m"
	Red          = "\033[31m"
	BrightRed    = "\033[91m"
	White        = "\033[97m"
	Gray         = "\033[90m"
)

// PrintBanner prints a futuristic, cyberpunk-style ASCII banner and telemetry box for SiPenDosa
func PrintBanner(cfg *config.Config) {
	logo := []string{
		`  ███████╗██╗██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ███████╗ █████╗ `,
		`  ██╔════╝██║██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗`,
		`  ███████╗██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║███████╗███████║`,
		`  ╚════██║██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║╚════██║██╔══██║`,
		`  ███████║██║██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝███████║██║  ██║`,
		`  ╚══════╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝`,
	}

	fmt.Println()
	// Print gradient ASCII
	colors := []string{BrightCyan, Cyan, BrightPurple, Magenta, BrightPurple, BrightCyan}
	for i, line := range logo {
		c := colors[i%len(colors)]
		fmt.Printf("%s%s%s%s\n", Bold, c, line, Reset)
	}

	fmt.Printf("  %s%s⚡ SIPENDOSA — SISTEM PENGINGAT DOSEN SAATNYA ⚡%s\n", Bold, BrightYellow, Reset)
	fmt.Printf("  %s%s\"Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan\"%s\n", Dim, Gray, Reset)
	fmt.Println()

	// Telemetry Box
	boxWidth := 74
	borderH := strings.Repeat("═", boxWidth-2)
	dividerH := strings.Repeat("─", boxWidth-2)

	fmt.Printf("%s%s╔%s╗%s\n", Bold, Cyan, borderH, Reset)
	fmt.Printf("%s%s║%s %s%-70s%s %s%s║%s\n", Bold, Cyan, Reset, BrightPurple+Bold, "SIPENDOSA DAEMON SYSTEM TELEMETRY & ENGINE STATUS", Reset, Bold, Cyan, Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Bold, Cyan, borderH, Reset)

	formatRow("App Name", "SiPenDosa (Sistem Pengingat Dosen Saatnya)")
	formatRow("Engine Version", "v1.0.0-PROD (Build 2026.09-Release)")
	formatRow("Core Environment", fmt.Sprintf("%s (%s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH))
	formatRow("Web Dashboard", fmt.Sprintf("http://localhost:%s", cfg.Port))
	formatRow("Listen Address", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port))
	formatRow("Database Engine", "SQLite Pure-Go WAL (Zero-CGO Isolated)")
	formatRow("Timezone Target", fmt.Sprintf("%s (WITA, UTC+8)", cfg.DefaultTimezone))
	formatRow("Anti-Ban Shield", fmt.Sprintf("ACTIVE (Human Presence + Jitter %d-%ds)", cfg.RateLimitMinSec, cfg.RateLimitMaxSec))
	formatRow("Sending Window", fmt.Sprintf("%s — %s (H-1 Reminder Mode)", cfg.SendWindowStart, cfg.SendWindowEnd))
	dryRunStr := "DISABLED (Real WhatsApp Delivery)"
	if cfg.GlobalDryRun {
		dryRunStr = "ENABLED (Simulation Only)"
	}
	formatRow("Global Dry-Run", dryRunStr)

	fmt.Printf("%s%s╚%s╝%s\n", Bold, Cyan, borderH, Reset)
	fmt.Println()

	// Dashboard quick link callout
	fmt.Printf("%s%s┌%s┐%s\n", Bold, Green, dividerH, Reset)

	urlText := fmt.Sprintf("http://localhost:%s", cfg.Port)
	line1 := fmt.Sprintf("  🚀 WEB DASHBOARD SIAP DIAKSES: %s", urlText)
	pad1 := boxWidth - 2 - len(line1)
	if pad1 < 0 {
		pad1 = 0
	}
	fmt.Println(Bold + Green + "│" + Reset + Bold + BrightGreen + line1 + strings.Repeat(" ", pad1) + Bold + Green + "│" + Reset)

	line2 := "  Tekan Ctrl+C di konsol ini untuk mematikan daemon secara aman (graceful)"
	pad2 := boxWidth - 2 - len(line2)
	if pad2 < 0 {
		pad2 = 0
	}
	fmt.Println(Bold + Green + "│" + Reset + Gray + line2 + strings.Repeat(" ", pad2) + Bold + Green + "│" + Reset)

	fmt.Printf("%s%s└%s┘%s\n", Bold, Green, dividerH, Reset)
	fmt.Println()
}

func formatRow(label, value string) {
	labelFormatted := fmt.Sprintf("%s%s%-17s%s :", Gray, Bold, label, Reset)
	valFormatted := fmt.Sprintf("%s%-51s%s", White, value, Reset)
	fmt.Printf("%s%s║%s  %s %s %s%s║%s\n", Bold, Cyan, Reset, labelFormatted, valFormatted, Bold, Cyan, Reset)
}

// LogStep prints a cyber-themed formatted activity step
func LogStep(tag, message string) {
	timeStr := time.Now().Format("15:04:05")
	var tagColor string
	switch tag {
	case "SYSTEM", "DAEMON":
		tagColor = BrightCyan
	case "DATABASE", "STORE":
		tagColor = BrightGreen
	case "WHATSAPP", "WA":
		tagColor = BrightPurple
	case "QUEUE", "ANTI-BAN":
		tagColor = BrightYellow
	case "SCHEDULER":
		tagColor = BrightCyan
	case "ERROR":
		tagColor = BrightRed
	default:
		tagColor = White
	}

	fmt.Printf("%s[%s]%s %s[%-9s]%s %s\n", Gray, timeStr, Reset, tagColor+Bold, tag, Reset, message)
}

// PrintQRHeader prints a stylish container header for terminal QR code
func PrintQRHeader() {
	boxWidth := 70
	borderH := strings.Repeat("═", boxWidth-2)
	fmt.Println()
	fmt.Printf("%s%s╔%s╗%s\n", Bold, Yellow, borderH, Reset)
	fmt.Printf("%s%s║%s  %s%s📱 PINDAI WHATSAPP QR CODE UNTUK MENAUTKAN PERANGKAT SIPENDOSA%s     %s%s║%s\n",
		Bold, Yellow, Reset, Bold, BrightYellow, Reset, Bold, Yellow, Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Bold, Yellow, borderH, Reset)
	fmt.Printf("%s%s║%s  1. Buka WhatsApp di smartphone Anda                              %s%s║%s\n", Bold, Yellow, Reset, Bold, Yellow, Reset)
	fmt.Printf("%s%s║%s  2. Buka Menu (⋮) atau Pengaturan > Perangkat Tertaut              %s%s║%s\n", Bold, Yellow, Reset, Bold, Yellow, Reset)
	fmt.Printf("%s%s║%s  3. Pilih 'Tautkan Perangkat' dan arahkan kamera ke QR berikut:    %s%s║%s\n", Bold, Yellow, Reset, Bold, Yellow, Reset)
	fmt.Printf("%s%s╚%s╝%s\n", Bold, Yellow, borderH, Reset)
	fmt.Println()
}

// PrintQRFooter prints tips below the terminal QR code
func PrintQRFooter(port string) {
	fmt.Println()
	fmt.Printf("%s%s[💡 TIP]%s Anda juga dapat memindai QR Code melalui browser di: %shttp://localhost:%s%s\n",
		Bold, BrightCyan, Reset, BrightGreen+Bold, port, Reset)
	fmt.Printf("%s%s         (Buka menu Overview > Klik tombol 'Scan QR Code')%s\n\n", Gray, Reset, Reset)
}

// PrintConnectedBox prints a celebratory connected box
func PrintConnectedBox(jid, pushName string) {
	boxWidth := 70
	borderH := strings.Repeat("═", boxWidth-2)
	fmt.Println()
	fmt.Printf("%s%s╔%s╗%s\n", Bold, Green, borderH, Reset)
	fmt.Printf("%s%s║%s  %s%s✓ WHATSAPP ENGINE BERHASIL TERHUBUNG SECARA ONLINE%s               %s%s║%s\n",
		Bold, Green, Reset, Bold, BrightGreen, Reset, Bold, Green, Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Bold, Green, borderH, Reset)
	fmt.Printf("%s%s║%s  • Akun JID    : %s%-50s%s%s%s║%s\n", Bold, Green, Reset, BrightCyan, jid, Reset, Bold, Green, Reset)
	if pushName != "" {
		fmt.Printf("%s%s║%s  • Nama Akun   : %s%-50s%s%s%s║%s\n", Bold, Green, Reset, White, pushName, Reset, Bold, Green, Reset)
	}
	fmt.Printf("%s%s║%s  • Status Sesi : %sAKTIF (SiPenDosa siaga mengirim pengingat jadwal)%s   %s%s║%s\n", Bold, Green, Reset, BrightGreen, Reset, Bold, Green, Reset)
	fmt.Printf("%s%s╚%s╝%s\n", Bold, Green, borderH, Reset)
	fmt.Println()
}
