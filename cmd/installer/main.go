package main

import (
	"bufio"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed sipen.exe
var embeddedBinary []byte

const (
	AppName        = "SiPenDosa"
	AppDescription = "Sistem Pengingat Dosen Saatnya"
	AppVersion     = "1.0.0"
	DefaultPort    = "8473"
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
	White        = "\033[97m"
	Gray         = "\033[90m"
)

func main() {
	isSilent := false
	noLaunch := false
	customDir := ""

	for i := 1; i < len(os.Args); i++ {
		arg := strings.ToLower(os.Args[i])
		if arg == "-silent" || arg == "/s" || arg == "--silent" {
			isSilent = true
		} else if arg == "-no-launch" || arg == "--no-launch" {
			noLaunch = true
		} else if (arg == "-dir" || arg == "--dir") && i+1 < len(os.Args) {
			customDir = os.Args[i+1]
			i++
		}
	}

	if !isSilent {
		printBanner()
	}

	// 1. Determine Target Installation Directory
	installDir := customDir
	if installDir == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			homeDir, _ := os.UserHomeDir()
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		installDir = filepath.Join(localAppData, "Programs", "SiPenDosa")
	}

	if !isSilent {
		fmt.Printf("%s[1/6]%s Menentukan lokasi instalasi...\n", BrightCyan+Bold, Reset)
		fmt.Printf("      Lokasi target default : %s%s%s\n", BrightGreen, installDir, Reset)
		fmt.Printf("      Tekan %sENTER%s untuk menggunakan lokasi ini, atau ketik lokasi baru: ", Bold, Reset)

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			installDir = input
		}
		fmt.Println()
	}

	// 2. Create Target Directories
	if !isSilent {
		fmt.Printf("%s[2/6]%s Membuat direktori aplikasi di: %s...\n", BrightCyan+Bold, Reset, installDir)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		fmt.Printf("%s[ERROR] Gagal membuat direktori instalasi: %v%s\n", Yellow, err, Reset)
		os.Exit(1)
	}
	_ = os.MkdirAll(filepath.Join(installDir, "data"), 0755)
	_ = os.MkdirAll(filepath.Join(installDir, "session"), 0755)

	// 3. Extract Executable
	if !isSilent {
		fmt.Printf("%s[3/6]%s Mengekstrak binary %s.exe (%d MB)...\n", BrightCyan+Bold, Reset, AppName, len(embeddedBinary)/(1024*1024))
	}
	targetExe := filepath.Join(installDir, "sipen.exe")

	// Ensure no old process is locking the file
	_ = exec.Command("taskkill", "/f", "/im", "sipen.exe").Run()
	time.Sleep(500 * time.Millisecond)

	if err := os.WriteFile(targetExe, embeddedBinary, 0755); err != nil {
		fmt.Printf("%s[ERROR] Gagal mengekstrak binary: %v%s\n", Yellow, err, Reset)
		os.Exit(1)
	}

	// 4. Create Production .env if not exists
	envPath := filepath.Join(installDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if !isSilent {
			fmt.Printf("%s[4/6]%s Meng-generate file konfigurasi produksi (.env) dengan secret baru...\n", BrightCyan+Bold, Reset)
		}
		secretBytes := make([]byte, 32)
		_, _ = rand.Read(secretBytes)
		sessionSecret := hex.EncodeToString(secretBytes)

		envContent := fmt.Sprintf(`# ==========================================
# SiPenDosa — Sistem Pengingat Dosen Saatnya
# Production Environment Configuration
# ==========================================

PORT=%s
HOST=0.0.0.0
APP_ENV=production
SESSION_SECRET=%s

DB_PATH=data/sipen.db
WA_SESSION_PATH=session/whatsapp.db

DEFAULT_TIMEZONE=Asia/Makassar
SEND_WINDOW_START=08:00
SEND_WINDOW_END=16:00

RATE_LIMIT_MIN_SEC=5
RATE_LIMIT_MAX_SEC=15
MAX_RETRIES=3

GLOBAL_DRY_RUN=false
`, DefaultPort, sessionSecret)

		_ = os.WriteFile(envPath, []byte(envContent), 0644)
	} else {
		if !isSilent {
			fmt.Printf("%s[4/6]%s Mempertahankan konfigurasi .env yang sudah ada...\n", BrightCyan+Bold, Reset)
		}
	}

	// 5. Generate Helper Scripts & Uninstaller
	if !isSilent {
		fmt.Printf("%s[5/6]%s Membuat skrip launcher dan uninstaller...\n", BrightCyan+Bold, Reset)
	}
	createHelperScripts(installDir)

	// 6. Create Desktop & Start Menu Shortcuts & Registry entry
	if !isSilent {
		fmt.Printf("%s[6/6]%s Membuat shortcut Desktop & Start Menu Windows...\n", BrightCyan+Bold, Reset)
	}
	createShortcuts(installDir, targetExe)
	registerUninstall(installDir)

	if !isSilent {
		fmt.Println()
		printSuccessBox(installDir)

		if !noLaunch {
			fmt.Printf("\n%sApakah Anda ingin langsung menjalankan SiPenDosa dan membuka Web Dashboard? [Y/n]: %s", Bold+BrightYellow, Reset)
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans == "" || ans == "y" || ans == "yes" {
				launchApp(installDir, targetExe)
			}
		}
	}
}

func printBanner() {
	logo := []string{
		`  ███████╗██╗██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ███████╗ █████╗ `,
		`  ██╔════╝██║██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗`,
		`  ███████╗██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║███████╗███████║`,
		`  ╚════██║██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║╚════██║██╔══██║`,
		`  ███████║██║██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝███████║██║  ██║`,
		`  ╚══════╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝`,
	}

	fmt.Println()
	colors := []string{BrightCyan, Cyan, BrightPurple, Magenta, BrightPurple, BrightCyan}
	for i, line := range logo {
		c := colors[i%len(colors)]
		fmt.Printf("%s%s%s%s\n", Bold, c, line, Reset)
	}

	fmt.Printf("  %s%s⚡ SIPENDOSA WINDOWS STANDALONE INSTALLER (v%s) ⚡%s\n", Bold, BrightYellow, AppVersion, Reset)
	fmt.Printf("  %s%s\"Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan\"%s\n", Dim, Gray, Reset)
	fmt.Println()
	fmt.Println(strings.Repeat("─", 74))
	fmt.Println()
}

func printSuccessBox(installDir string) {
	boxWidth := 74
	borderH := strings.Repeat("═", boxWidth-2)

	fmt.Printf("%s%s╔%s╗%s\n", Bold, Green, borderH, Reset)
	fmt.Printf("%s%s║%s  %s%s✓ INSTALASI SIPENDOSA BERHASIL DISELESAIKAN DENGAN SEMPURNA!%s      %s%s║%s\n",
		Bold, Green, Reset, Bold, BrightGreen, Reset, Bold, Green, Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Bold, Green, borderH, Reset)
	fmt.Printf("%s%s║%s  • Lokasi Program : %-51s%s║%s\n", Bold, Green, Reset, installDir, Bold, Green, Reset)
	fmt.Printf("%s%s║%s  • Web Dashboard  : %shttp://localhost:%s%-38s%s%s║%s\n", Bold, Green, Reset, BrightCyan+Bold, DefaultPort, Reset, Bold, Green, Reset)
	fmt.Printf("%s%s║%s  • Desktop Icon   : Dibuat (SiPenDosa.lnk)%-35s%s║%s\n", Bold, Green, Reset, "", Bold, Green, Reset)
	fmt.Printf("%s%s║%s  • Start Menu     : Terdaftar di Start Menu > Programs > SiPenDosa    %s%s║%s\n", Bold, Green, Reset, Bold, Green, Reset)
	fmt.Printf("%s%s╚%s╝%s\n", Bold, Green, borderH, Reset)
}

func createHelperScripts(installDir string) {
	// 1. Silent Background Runner (VBS)
	vbsContent := `Set WshShell = CreateObject("WScript.Shell")
WshShell.CurrentDirectory = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)
WshShell.Run chr(34) & WshShell.CurrentDirectory & "\sipen.exe" & Chr(34), 0
Set WshShell = Nothing
`
	_ = os.WriteFile(filepath.Join(installDir, "run-background.vbs"), []byte(vbsContent), 0644)

	// 2. Stop Background Process
	stopBat := `@echo off
echo Menghentikan proses SiPenDosa...
taskkill /f /im sipen.exe >nul 2>&1
echo SiPenDosa telah dihentikan.
timeout /t 2 >nul
`
	_ = os.WriteFile(filepath.Join(installDir, "stop-background.bat"), []byte(stopBat), 0644)

	// 3. Open Browser Dashboard
	openBat := `@echo off
start http://localhost:8473
`
	_ = os.WriteFile(filepath.Join(installDir, "open-dashboard.bat"), []byte(openBat), 0644)

	// 4. Full Uninstaller Script
	uninstallBat := `@echo off
title SiPenDosa Uninstaller
echo ========================================================
echo        SIPENDOSA - SISTEM PENGINGAT DOSEN SAATNYA
echo                    UNINSTALLER WIZARD
echo ========================================================
echo.
echo [1/3] Menutup service/proses SiPenDosa jika sedang berjalan...
taskkill /f /im sipen.exe >nul 2>&1

echo [2/3] Menghapus shortcut Desktop dan Start Menu...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Remove-Item -Force -ErrorAction SilentlyContinue \"$([Environment]::GetFolderPath('Desktop'))\SiPenDosa.lnk\""
powershell -NoProfile -ExecutionPolicy Bypass -Command "Remove-Item -Recurse -Force -ErrorAction SilentlyContinue \"$([Environment]::GetFolderPath('Programs'))\SiPenDosa\""

echo [3/3] Menghapus registrasi sistem Windows...
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\SiPenDosa" /f >nul 2>&1

echo.
set /p DELDATA="Apakah Anda ingin menghapus seluruh database dan sesi WhatsApp? (Y/T, default T): "
if /i "%DELDATA%"=="Y" (
    echo Menghapus seluruh folder aplikasi...
    cd ..
    rmdir /s /q "%~dp0"
    echo Seluruh file dan database telah dihapus.
) else (
    echo Folder database data/ dan session/ tetap disimpan untuk backup.
    del /q "%~dp0sipen.exe" >nul 2>&1
    del /q "%~dp0run-background.vbs" >nul 2>&1
    del /q "%~dp0stop-background.bat" >nul 2>&1
    del /q "%~dp0open-dashboard.bat" >nul 2>&1
)

echo.
echo SiPenDosa berhasil di-uninstall dari komputer Anda. Terima kasih!
pause
`
	_ = os.WriteFile(filepath.Join(installDir, "uninstall.bat"), []byte(uninstallBat), 0644)
}

func createShortcuts(installDir, targetExe string) {
	desktopShortcut := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "SiPenDosa.lnk")
	startMenuDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "SiPenDosa")
	_ = os.MkdirAll(startMenuDir, 0755)

	startMenuTerminal := filepath.Join(startMenuDir, "SiPenDosa (Terminal).lnk")
	startMenuBg := filepath.Join(startMenuDir, "SiPenDosa (Background).lnk")
	startMenuStop := filepath.Join(startMenuDir, "Hentikan SiPenDosa.lnk")
	startMenuUninstall := filepath.Join(startMenuDir, "Uninstall SiPenDosa.lnk")

	// PowerShell script to create Windows shortcuts cleanly via WScript.Shell
	psScript := fmt.Sprintf(`
$wsh = New-Object -COM WScript.Shell

# Desktop Shortcut
$s1 = $wsh.CreateShortcut('%s')
$s1.TargetPath = '%s'
$s1.WorkingDirectory = '%s'
$s1.Description = 'SiPenDosa — Sistem Pengingat Dosen Saatnya'
$s1.Save()

# Start Menu Terminal
$s2 = $wsh.CreateShortcut('%s')
$s2.TargetPath = '%s'
$s2.WorkingDirectory = '%s'
$s2.Description = 'Jalankan SiPenDosa dengan Jendela Terminal dan Live Logs'
$s2.Save()

# Start Menu Background
$s3 = $wsh.CreateShortcut('%s')
$s3.TargetPath = 'wscript.exe'
$s3.Arguments = '"%s"'
$s3.WorkingDirectory = '%s'
$s3.Description = 'Jalankan SiPenDosa di Latar Belakang (Hening Tanpa Jendela Hitam)'
$s3.Save()

# Start Menu Stop
$s4 = $wsh.CreateShortcut('%s')
$s4.TargetPath = '%s'
$s4.WorkingDirectory = '%s'
$s4.Description = 'Hentikan Daemon SiPenDosa'
$s4.Save()

# Start Menu Uninstall
$s5 = $wsh.CreateShortcut('%s')
$s5.TargetPath = '%s'
$s5.WorkingDirectory = '%s'
$s5.Description = 'Uninstall SiPenDosa'
$s5.Save()
`,
		escapePS(desktopShortcut),
		escapePS(targetExe),
		escapePS(installDir),
		escapePS(startMenuTerminal),
		escapePS(targetExe),
		escapePS(installDir),
		escapePS(startMenuBg),
		escapePS(filepath.Join(installDir, "run-background.vbs")),
		escapePS(installDir),
		escapePS(startMenuStop),
		escapePS(filepath.Join(installDir, "stop-background.bat")),
		escapePS(installDir),
		escapePS(startMenuUninstall),
		escapePS(filepath.Join(installDir, "uninstall.bat")),
		escapePS(installDir),
	)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	_ = cmd.Run()

	// Web URL Shortcut in Start Menu
	urlShortcut := filepath.Join(startMenuDir, "Buka Web Dashboard.url")
	urlContent := fmt.Sprintf("[InternetShortcut]\nURL=http://localhost:%s\n", DefaultPort)
	_ = os.WriteFile(urlShortcut, []byte(urlContent), 0644)
}

func registerUninstall(installDir string) {
	// Register in HKCU Software\Microsoft\Windows\CurrentVersion\Uninstall\SiPenDosa
	psScript := fmt.Sprintf(`
$regKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\SiPenDosa'
if (-not (Test-Path $regKey)) {
    New-Item -Path $regKey -Force | Out-Null
}
Set-ItemProperty -Path $regKey -Name 'DisplayName' -Value 'SiPenDosa (Sistem Pengingat Dosen Saatnya)'
Set-ItemProperty -Path $regKey -Name 'DisplayVersion' -Value '%s'
Set-ItemProperty -Path $regKey -Name 'Publisher' -Value 'SiPenDosa Team'
Set-ItemProperty -Path $regKey -Name 'InstallLocation' -Value '%s'
Set-ItemProperty -Path $regKey -Name 'UninstallString' -Value '"%s"'
Set-ItemProperty -Path $regKey -Name 'URLInfoAbout' -Value 'http://localhost:%s'
Set-ItemProperty -Path $regKey -Name 'DisplayIcon' -Value '%s'
`,
		AppVersion,
		escapePS(installDir),
		escapePS(filepath.Join(installDir, "uninstall.bat")),
		DefaultPort,
		escapePS(filepath.Join(installDir, "sipen.exe")),
	)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	_ = cmd.Run()
}

func launchApp(installDir, targetExe string) {
	fmt.Printf("%sMenjalankan SiPenDosa dan membuka browser...%s\n", BrightCyan, Reset)

	// Start SiPenDosa in new window
	cmd := exec.Command("cmd.exe", "/c", "start", "SiPenDosa Daemon", targetExe)
	cmd.Dir = installDir
	_ = cmd.Start()

	// Give it 1.5 seconds to start HTTP listener, then launch browser
	time.Sleep(1500 * time.Millisecond)
	_ = exec.Command("cmd.exe", "/c", "start", fmt.Sprintf("http://localhost:%s", DefaultPort)).Run()
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
