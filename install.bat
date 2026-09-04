@echo off
title SiPenDosa Installer Launcher
cd /d "%~dp0"

if exist "SiPenDosa-Setup.exe" (
    echo Menjalankan SiPenDosa-Setup.exe...
    start "" "SiPenDosa-Setup.exe"
    exit /b
)

echo Menjalankan skrip instalasi PowerShell...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install.ps1"
pause
