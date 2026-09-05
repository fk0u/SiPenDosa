#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — macOS Standalone Installer & LaunchAgent Setup
# Mendukung: Apple Silicon (M1/M2/M3/M4/arm64) & Intel (x86_64)
# "Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan"
# ==============================================================================

set -e

RED='\033[0;31m'
BRED='\033[1;31m'
GREEN='\033[0;32m'
BGREEN='\033[1;32m'
YELLOW='\033[0;33m'
BYELLOW='\033[1;33m'
GRAY='\033[0;90m'
BOLD='\033[1m'
NC='\033[0m'

APP_NAME="sipen"
APP_TITLE="SiPenDosa"
DEFAULT_PORT="8473"

INSTALL_DIR="${HOME}/Applications/SiPenDosa"
APP_SUPPORT_DIR="${HOME}/Library/Application Support/SiPenDosa"
PLIST_PATH="${HOME}/Library/LaunchAgents/com.sipendosa.daemon.plist"

print_banner() {
    echo -e "${BRED}"
    echo '  ███████╗██╗██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ███████╗ █████╗ '
    echo '  ██╔════╝██║██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗'
    echo '  ███████╗██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║███████╗███████║'
    echo '  ╚════██║██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║╚════██║██╔══██║'
    echo '  ███████║██║██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝███████║██║  ██║'
    echo '  ╚══════╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝'
    echo -e "${NC}"
    echo -e "  ${BYELLOW}⚡ ${APP_TITLE} — MACOS STANDALONE INSTALLER ⚡${NC}"
    echo -e "  ${GRAY}\"Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan\"${NC}"
    echo
    echo -e "${GRAY}────────────────────────────────────────────────────────────────────────${NC}"
    echo
}

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        arm64)
            TARGET_ARCH="arm64"
            ;;
        x86_64)
            TARGET_ARCH="amd64"
            ;;
        *)
            echo -e "${RED}[ERROR] Arsitektur macOS $ARCH belum didukung.${NC}"
            exit 1
            ;;
    esac
    echo -e "${BYELLOW}[1/5]${NC} Arsitektur Mac terdeteksi: ${BGREEN}${ARCH} (${TARGET_ARCH})${NC}"
}

find_binary() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SRC_BIN=""

    if [ -f "${SCRIPT_DIR}/bin/darwin_${TARGET_ARCH}/${APP_NAME}" ]; then
        SRC_BIN="${SCRIPT_DIR}/bin/darwin_${TARGET_ARCH}/${APP_NAME}"
    elif [ -f "${SCRIPT_DIR}/${APP_NAME}" ]; then
        SRC_BIN="${SCRIPT_DIR}/${APP_NAME}"
    elif command -v go >/dev/null 2>&1; then
        echo -e "${YELLOW}Binary macOS belum terkompilasi, mengompilasi dari sumber menggunakan Go lokal...${NC}"
        (cd "$SCRIPT_DIR" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${APP_NAME}" ./cmd/sipen)
        SRC_BIN="${SCRIPT_DIR}/${APP_NAME}"
    else
        echo -e "${RED}[ERROR] Binary ${APP_NAME} tidak ditemukan di paket ini.${NC}"
        exit 1
    fi

    echo -e "${BYELLOW}[2/5]${NC} Menggunakan binary: ${BGREEN}${SRC_BIN}${NC}"
}

setup_app_files() {
    echo -e "${BYELLOW}[3/5]${NC} Menyiapkan file aplikasi di ${BGREEN}${INSTALL_DIR}${NC}..."
    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${APP_SUPPORT_DIR}/data"
    mkdir -p "${APP_SUPPORT_DIR}/session"

    # Salin binary
    cp -f "$SRC_BIN" "${INSTALL_DIR}/${APP_NAME}"
    chmod 755 "${INSTALL_DIR}/${APP_NAME}"

    # Generate .env jika belum ada
    ENV_FILE="${INSTALL_DIR}/.env"
    if [ ! -f "$ENV_FILE" ]; then
        SECRET=$(head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n' 2>/dev/null || echo "mac-sipen-secure-random-token-32")
        cat <<EOF > "$ENV_FILE"
PORT=${DEFAULT_PORT}
HOST=127.0.0.1
APP_ENV=production
SESSION_SECRET=${SECRET}

DB_PATH=${APP_SUPPORT_DIR}/data/sipen.db
WA_SESSION_PATH=${APP_SUPPORT_DIR}/session/whatsapp.db

DEFAULT_TIMEZONE=Asia/Makassar
SEND_WINDOW_START=08:00
SEND_WINDOW_END=16:00

RATE_LIMIT_MIN_SEC=5
RATE_LIMIT_MAX_SEC=15
MAX_RETRIES=3

GLOBAL_DRY_RUN=false
EOF
        echo -e "      ${BGREEN}✓ Konfigurasi .env produksi berhasil dibuat.${NC}"
    else
        echo -e "      ${YELLOW}✓ Konfigurasi .env yang sudah ada dipertahankan.${NC}"
    fi

    # Buat launcher script
    cat <<'EOF' > "${INSTALL_DIR}/start-terminal.sh"
#!/bin/bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR" && ./sipen
EOF
    chmod +x "${INSTALL_DIR}/start-terminal.sh"
}

setup_launchagent() {
    echo -e "${BYELLOW}[4/5]${NC} Memasang LaunchAgent macOS di ${BGREEN}${PLIST_PATH}${NC}..."
    mkdir -p "${HOME}/Library/LaunchAgents"

    # Unload daemon jika sedang berjalan
    launchctl unload "$PLIST_PATH" 2>/dev/null || true

    cat <<EOF > "$PLIST_PATH"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sipendosa.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${APP_NAME}</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${APP_SUPPORT_DIR}/daemon.log</string>
    <key>StandardErrorPath</key>
    <string>${APP_SUPPORT_DIR}/daemon.log</string>
</dict>
</plist>
EOF

    launchctl load "$PLIST_PATH"
    echo -e "      ${BGREEN}✓ Daemon LaunchAgent berhasil dipasang dan dijalankan.${NC}"
}

launch_browser() {
    echo -e "${BYELLOW}[5/5]${NC} Membuka Web Dashboard SiPenDosa di browser..."
    sleep 1
    open "http://localhost:${DEFAULT_PORT}" || true
}

print_success() {
    echo
    echo -e "${BGREEN}╔════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BGREEN}║${NC}  ${BGREEN}${BOLD}✓ INSTALASI SIPENDOSA DI MACOS BERHASIL DISELESAIKAN!${NC}                 ${BGREEN}║${NC}"
    echo -e "${BGREEN}╠════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BGREEN}║${NC}  • Lokasi Program : ${INSTALL_DIR}                             ${BGREEN}║${NC}"
    echo -e "${BGREEN}║${NC}  • Web Dashboard  : ${BYELLOW}http://localhost:${DEFAULT_PORT}${NC}                                  ${BGREEN}║${NC}"
    echo -e "${BGREEN}║${NC}  • Status Daemon  : ${BGREEN}AKTIF di latar belakang (LaunchAgent)${NC}               ${BGREEN}║${NC}"
    echo -e "${BGREEN}║${NC}  • File Log       : ${GRAY}${APP_SUPPORT_DIR}/daemon.log${NC}       ${BGREEN}║${NC}"
    echo -e "${BGREEN}╚════════════════════════════════════════════════════════════════════════╝${NC}"
    echo
    echo -e "${BYELLOW}Perintah Kontrol Daemon macOS:${NC}"
    echo -e "  • Hentikan:  ${GRAY}launchctl unload ${PLIST_PATH}${NC}"
    echo -e "  • Nyalakan:  ${GRAY}launchctl load ${PLIST_PATH}${NC}"
    echo -e "  • Pantau Log: ${GRAY}tail -f \"${APP_SUPPORT_DIR}/daemon.log\"${NC}"
    echo
}

main() {
    print_banner
    detect_arch
    find_binary
    setup_app_files
    setup_launchagent
    launch_browser
    print_success
}

main "$@"
