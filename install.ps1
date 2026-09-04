# ========================================================================
# SiPenDosa — One-Click Windows PowerShell Installer
# "Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan"
# ========================================================================

[CmdletBinding()]
param (
    [string]$TargetDir = "$env:LOCALAPPDATA\Programs\SiPenDosa",
    [switch]$Silent,
    [switch]$NoLaunch
)

$ErrorActionPreference = "Stop"

function Write-CyberHeader {
    Clear-Host
    Write-Host ""
    Write-Host "  ███████╗██╗██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ███████╗ █████╗ " -ForegroundColor Cyan
    Write-Host "  ██╔════╝██║██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗" -ForegroundColor Cyan
    Write-Host "  ███████╗██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║███████╗███████║" -ForegroundColor Magenta
    Write-Host "  ╚════██║██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║╚════██║██╔══██║" -ForegroundColor Magenta
    Write-Host "  ███████║██║██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝███████║██║  ██║" -ForegroundColor Cyan
    Write-Host "  ╚══════╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝" -ForegroundColor Cyan
    Write-Host "  ⚡ SIPENDOSA ONE-CLICK POWERSHELL INSTALLER ⚡" -ForegroundColor Yellow
    Write-Host "  `"Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan`"" -ForegroundColor Gray
    Write-Host ""
    Write-Host ("─" * 74) -ForegroundColor DarkGray
    Write-Host ""
}

if (-not $Silent) {
    Write-CyberHeader
    Write-Host "[1/6] Menentukan target folder instalasi..." -ForegroundColor Cyan
    Write-Host "      Default: $TargetDir" -ForegroundColor Green
    $userInput = Read-Host "      Tekan ENTER untuk menggunakan lokasi ini, atau ketik lokasi baru"
    if ($userInput.Trim() -ne "") {
        $TargetDir = $userInput.Trim()
    }
    Write-Host ""
}

# 1. Pastikan proses lama ditutup
Get-Process "sipen" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500

# 2. Buat direktori target
if (-not $Silent) { Write-Host "[2/6] Menyiapkan direktori aplikasi di $TargetDir..." -ForegroundColor Cyan }
New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $TargetDir "data") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $TargetDir "session") | Out-Null

# 3. Salin executable
$scriptDir = $PSScriptRoot
$sourceExe = Join-Path $scriptDir "sipen.exe"
if (-not (Test-Path $sourceExe)) {
    $sourceExe = Join-Path $scriptDir "bin\sipen.exe"
}

if (Test-Path $sourceExe) {
    if (-not $Silent) { Write-Host "[3/6] Menyalin file binary sipen.exe..." -ForegroundColor Cyan }
    Copy-Item $sourceExe -Destination (Join-Path $TargetDir "sipen.exe") -Force
} else {
    Write-Host "[ERROR] Binary sipen.exe tidak ditemukan di folder instalasi!" -ForegroundColor Red
    exit 1
}

# 4. Generate .env jika belum ada
$envFile = Join-Path $TargetDir ".env"
if (-not (Test-Path $envFile)) {
    if (-not $Silent) { Write-Host "[4/6] Meng-generate file konfigurasi produksi (.env)..." -ForegroundColor Cyan }
    $secretBytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($secretBytes)
    $sessionSecret = ($secretBytes | ForEach-Object { "{0:x2}" -f $_ }) -join ""

    $envContent = @"
# ==========================================
# SiPenDosa — Sistem Pengingat Dosen Saatnya
# Production Environment Configuration
# ==========================================

PORT=8473
HOST=0.0.0.0
APP_ENV=production
SESSION_SECRET=$sessionSecret

DB_PATH=data/sipen.db
WA_SESSION_PATH=session/whatsapp.db

DEFAULT_TIMEZONE=Asia/Makassar
SEND_WINDOW_START=08:00
SEND_WINDOW_END=16:00

RATE_LIMIT_MIN_SEC=5
RATE_LIMIT_MAX_SEC=15
MAX_RETRIES=3

GLOBAL_DRY_RUN=false
"@
    Set-Content -Path $envFile -Value $envContent -Encoding UTF8
} else {
    if (-not $Silent) { Write-Host "[4/6] Mempertahankan konfigurasi .env yang sudah ada..." -ForegroundColor Cyan }
}

# 5. Buat Helper Scripts & Uninstaller
if (-not $Silent) { Write-Host "[5/6] Membuat skrip pendukung (background runner & uninstaller)..." -ForegroundColor Cyan }

# run-background.vbs
$vbsContent = @"
Set WshShell = CreateObject("WScript.Shell")
WshShell.CurrentDirectory = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)
WshShell.Run chr(34) & WshShell.CurrentDirectory & "\sipen.exe" & Chr(34), 0
Set WshShell = Nothing
"@
Set-Content -Path (Join-Path $TargetDir "run-background.vbs") -Value $vbsContent -Encoding UTF8

# stop-background.bat
$stopBat = @"
@echo off
echo Menghentikan proses SiPenDosa...
taskkill /f /im sipen.exe >nul 2>&1
echo SiPenDosa telah dihentikan.
timeout /t 2 >nul
"@
Set-Content -Path (Join-Path $TargetDir "stop-background.bat") -Value $stopBat -Encoding ASCII

# open-dashboard.bat
$openBat = @"
@echo off
start http://localhost:8473
"@
Set-Content -Path (Join-Path $TargetDir "open-dashboard.bat") -Value $openBat -Encoding ASCII

# uninstall.bat
$uninstallBat = @"
@echo off
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
"@
Set-Content -Path (Join-Path $TargetDir "uninstall.bat") -Value $uninstallBat -Encoding ASCII

# 6. Buat Windows Shortcuts
if (-not $Silent) { Write-Host "[6/6] Membuat shortcut Desktop & Start Menu..." -ForegroundColor Cyan }

$wsh = New-Object -ComObject WScript.Shell

# Desktop Shortcut
$desktopPath = [Environment]::GetFolderPath("Desktop")
$s1 = $wsh.CreateShortcut((Join-Path $desktopPath "SiPenDosa.lnk"))
$s1.TargetPath = (Join-Path $TargetDir "sipen.exe")
$s1.WorkingDirectory = $TargetDir
$s1.Description = "SiPenDosa — Sistem Pengingat Dosen Saatnya"
$s1.Save()

# Start Menu Folder
$startMenuDir = Join-Path ([Environment]::GetFolderPath("Programs")) "SiPenDosa"
New-Item -ItemType Directory -Force -Path $startMenuDir | Out-Null

$s2 = $wsh.CreateShortcut((Join-Path $startMenuDir "SiPenDosa (Terminal).lnk"))
$s2.TargetPath = (Join-Path $TargetDir "sipen.exe")
$s2.WorkingDirectory = $TargetDir
$s2.Description = "Jalankan SiPenDosa dengan Jendela Terminal dan Live Logs"
$s2.Save()

$s3 = $wsh.CreateShortcut((Join-Path $startMenuDir "SiPenDosa (Background).lnk"))
$s3.TargetPath = "wscript.exe"
$s3.Arguments = "`"" + (Join-Path $TargetDir "run-background.vbs") + "`""
$s3.WorkingDirectory = $TargetDir
$s3.Description = "Jalankan SiPenDosa di Latar Belakang (Hening Tanpa Jendela Hitam)"
$s3.Save()

$s4 = $wsh.CreateShortcut((Join-Path $startMenuDir "Hentikan SiPenDosa.lnk"))
$s4.TargetPath = (Join-Path $TargetDir "stop-background.bat")
$s4.WorkingDirectory = $TargetDir
$s4.Save()

$s5 = $wsh.CreateShortcut((Join-Path $startMenuDir "Uninstall SiPenDosa.lnk"))
$s5.TargetPath = (Join-Path $TargetDir "uninstall.bat")
$s5.WorkingDirectory = $TargetDir
$s5.Save()

# Web Dashboard URL Shortcut
$urlContent = "[InternetShortcut]`nURL=http://localhost:8473`n"
Set-Content -Path (Join-Path $startMenuDir "Buka Web Dashboard.url") -Value $urlContent -Encoding ASCII

# Registrasi Registry
$regKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\SiPenDosa'
if (-not (Test-Path $regKey)) {
    New-Item -Path $regKey -Force | Out-Null
}
Set-ItemProperty -Path $regKey -Name 'DisplayName' -Value 'SiPenDosa (Sistem Pengingat Dosen Saatnya)'
Set-ItemProperty -Path $regKey -Name 'DisplayVersion' -Value '1.0.0'
Set-ItemProperty -Path $regKey -Name 'Publisher' -Value 'SiPenDosa Team'
Set-ItemProperty -Path $regKey -Name 'InstallLocation' -Value $TargetDir
Set-ItemProperty -Path $regKey -Name 'UninstallString' -Value "`"$TargetDir\uninstall.bat`""
Set-ItemProperty -Path $regKey -Name 'URLInfoAbout' -Value 'http://localhost:8473'
Set-ItemProperty -Path $regKey -Name 'DisplayIcon' -Value "$TargetDir\sipen.exe"

if (-not $Silent) {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════════════════════╗" -ForegroundColor Green
    Write-Host "║  ✓ INSTALASI SIPENDOSA BERHASIL DISELESAIKAN DENGAN SEMPURNA!          ║" -ForegroundColor Green
    Write-Host "╠════════════════════════════════════════════════════════════════════════╣" -ForegroundColor Green
    Write-Host "║  • Lokasi Program : $TargetDir" -ForegroundColor White
    Write-Host "║  • Web Dashboard  : http://localhost:8473" -ForegroundColor Cyan
    Write-Host "║  • Desktop Icon   : Dibuat (SiPenDosa.lnk)" -ForegroundColor White
    Write-Host "║  • Start Menu     : Terdaftar di Start Menu > Programs > SiPenDosa" -ForegroundColor White
    Write-Host "╚════════════════════════════════════════════════════════════════════════╝" -ForegroundColor Green
    Write-Host ""

    if (-not $NoLaunch) {
        $runNow = Read-Host "Apakah Anda ingin langsung menjalankan SiPenDosa dan membuka browser? [Y/n]"
        if ($runNow.Trim() -eq "" -or $runNow.Trim() -eq "y" -or $runNow.Trim() -eq "Y") {
            Start-Process -FilePath (Join-Path $TargetDir "sipen.exe") -WorkingDirectory $TargetDir
            Start-Sleep -Seconds 2
            Start-Process "http://localhost:8473"
        }
    }
}
