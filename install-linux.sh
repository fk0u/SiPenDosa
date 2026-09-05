#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Universal Linux Standalone Installer
# Kompatibel dengan: Ubuntu, Debian, RedHat, CentOS, Fedora, Rocky, AlmaLinux, Arch
# "Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan"
# ==============================================================================

set -e

# Warna Terminal (Bebas Biru/Ungu — Ember Crimson & Gold Brand SiPenDosa)
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
APP_DESC="Sistem Pengingat Dosen Saatnya"
INSTALL_DIR="/opt/sipen"
DEFAULT_PORT="8473"

print_banner() {
    echo -e "${BRED}"
    echo '  ███████╗██╗██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ███████╗ █████╗ '
    echo '  ██╔════╝██║██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗'
    echo '  ███████╗██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║███████╗███████║'
    echo '  ╚════██║██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║╚════██║██╔══██║'
    echo '  ███████║██║██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝███████║██║  ██║'
    echo '  ╚══════╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝'
    echo -e "${NC}"
    echo -e "  ${BYELLOW}⚡ ${APP_TITLE} — UNIVERSAL LINUX STANDALONE INSTALLER ⚡${NC}"
    echo -e "  ${GRAY}\"Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak sungkan\"${NC}"
    echo
    echo -e "${GRAY}────────────────────────────────────────────────────────────────────────${NC}"
    echo
}

# 1. Pengecekan Hak Akses Root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${BYELLOW}[PERHATIAN]${NC} Skrip ini membutuhkan hak akses root / sudo untuk memasang service systemd."
        echo -e "Jalankan ulang dengan: ${BGREEN}sudo bash $0${NC}"
        exit 1
    fi
}

# 2. Deteksi Arsitektur Mesin
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64)
            TARGET_ARCH="amd64"
            ;;
        aarch64|arm64)
            TARGET_ARCH="arm64"
            ;;
        *)
            echo -e "${RED}[ERROR] Arsitektur $ARCH saat ini belum didukung secara resmi.${NC}"
            exit 1
            ;;
    esac
    echo -e "${BYELLOW}[1/6]${NC} Arsitektur terdeteksi: ${BGREEN}${ARCH} (${TARGET_ARCH})${NC}"
}

# 3. Temukan Binary Sumber
find_binary() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SRC_BIN=""

    if [ -f "${SCRIPT_DIR}/bin/linux_${TARGET_ARCH}/${APP_NAME}" ]; then
        SRC_BIN="${SCRIPT_DIR}/bin/linux_${TARGET_ARCH}/${APP_NAME}"
    elif [ -f "${SCRIPT_DIR}/${APP_NAME}" ]; then
        SRC_BIN="${SCRIPT_DIR}/${APP_NAME}"
    elif [ -f "${SCRIPT_DIR}/${APP_NAME}_linux_${TARGET_ARCH}" ]; then
        SRC_BIN="${SCRIPT_DIR}/${APP_NAME}_linux_${TARGET_ARCH}"
    elif command -v go >/dev/null 2>&1; then
        echo -e "${YELLOW}Binary siap pakai tidak ditemukan, mengompilasi dari sumber menggunakan Go lokal...${NC}"
        (cd "$SCRIPT_DIR" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "${APP_NAME}" ./cmd/sipen)
        SRC_BIN="${SCRIPT_DIR}/${APP_NAME}"
    else
        echo -e "${RED}[ERROR] Binary ${APP_NAME} tidak ditemukan di folder paket dan compiler Go tidak terpasang.${NC}"
        exit 1
    fi

    echo -e "${BYELLOW}[2/6]${NC} Binary sumber: ${BGREEN}${SRC_BIN}${NC}"
}

# 4. Siapkan Direktori dan Salin Binary
setup_directories() {
    echo -e "${BYELLOW}[3/6]${NC} Menyiapkan direktori instalasi di ${BGREEN}${INSTALL_DIR}${NC}..."
    mkdir -p "${INSTALL_DIR}/data"
    mkdir -p "${INSTALL_DIR}/session"

    cp -f "$SRC_BIN" "${INSTALL_DIR}/${APP_NAME}"
    chmod 755 "${INSTALL_DIR}/${APP_NAME}"

    # Buat akun sistem khusus sipen jika belum ada
    if ! id -u sipen >/dev/null 2>&1; then
        if command -v useradd >/dev/null 2>&1; then
            useradd --system --no-create-home --shell /usr/sbin/nologin sipen || true
        elif command -v adduser >/dev/null 2>&1; then
            adduser -S -D -H sipen || true
        fi
    fi

    # Berikan izin direktori
    chown -R sipen:sipen "${INSTALL_DIR}" 2>/dev/null || chown -R root:root "${INSTALL_DIR}"
}

# 5. Buat Konfigurasi .env Produksi Jika Belum Ada
setup_env() {
    echo -e "${BYELLOW}[4/6]${NC} Memeriksa konfigurasi environment (${INSTALL_DIR}/.env)..."
    ENV_FILE="${INSTALL_DIR}/.env"

    if [ ! -f "$ENV_FILE" ]; then
        # Generate random secret 32 bytes hex
        if command -v openssl >/dev/null 2>&1; then
            SECRET=$(openssl rand -hex 32)
        else
            SECRET=$(head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n')
        fi

        cat <<EOF > "$ENV_FILE"
# ==============================================================================
# SiPenDosa — Sistem Pengingat Dosen Saatnya (Production Configuration)
# ==============================================================================
PORT=${DEFAULT_PORT}
HOST=0.0.0.0
APP_ENV=production
SESSION_SECRET=${SECRET}

DB_PATH=data/sipen.db
WA_SESSION_PATH=session/whatsapp.db

DEFAULT_TIMEZONE=Asia/Makassar
SEND_WINDOW_START=08:00
SEND_WINDOW_END=16:00

RATE_LIMIT_MIN_SEC=5
RATE_LIMIT_MAX_SEC=15
MAX_RETRIES=3

GLOBAL_DRY_RUN=false
EOF
        chmod 600 "$ENV_FILE"
        chown sipen:sipen "$ENV_FILE" 2>/dev/null || true
        echo -e "      ${BGREEN}✓ File .env produksi berhasil dibuat dengan secret acak 32 karakter.${NC}"
    else
        echo -e "      ${YELLOW}✓ Konfigurasi .env yang sudah ada dipertahankan.${NC}"
    fi
}

# 6. Pasang dan Aktifkan Systemd Service
setup_systemd() {
    echo -e "${BYELLOW}[5/6]${NC} Memasang systemd service (${APP_NAME}.service)..."

    # Tentukan user service (sipen atau root jika sipen gagal)
    SVC_USER="sipen"
    if ! id -u sipen >/dev/null 2>&1; then
        SVC_USER="root"
    fi

    cat <<EOF > "/etc/systemd/system/${APP_NAME}.service"
[Unit]
Description=${APP_TITLE} — ${APP_DESC} (Daemon Engine)
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SVC_USER}
Group=${SVC_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${APP_NAME}
Restart=always
RestartSec=5s
LimitNOFILE=65535
EnvironmentFile=${INSTALL_DIR}/.env

# Hardening
PrivateTmp=true
ProtectSystem=full
NoNewPrivileges=true

StandardOutput=journal
StandardError=journal
SyslogIdentifier=${APP_NAME}

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "${APP_NAME}"
    systemctl restart "${APP_NAME}"
    echo -e "      ${BGREEN}✓ Service systemd berhasil dipasang dan dijalankan.${NC}"
}

# 7. Pengaturan Firewall (UFW / Firewalld)
setup_firewall() {
    echo -e "${BYELLOW}[6/6]${NC} Memeriksa status firewall..."
    if command -v ufw >/dev/null 2>&1 && ufw status | grep -q "Status: active"; then
        echo -e "      Membuka port ${DEFAULT_PORT}/tcp pada UFW..."
        ufw allow ${DEFAULT_PORT}/tcp comment "SiPenDosa Web Dashboard" >/dev/null 2>&1 || true
        echo -e "      ${BGREEN}✓ Port ${DEFAULT_PORT} diizinkan pada UFW.${NC}"
    elif command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld; then
        echo -e "      Membuka port ${DEFAULT_PORT}/tcp pada firewalld (RHEL/CentOS/Fedora)..."
        firewall-cmd --permanent --add-port=${DEFAULT_PORT}/tcp >/dev/null 2>&1 || true
        firewall-cmd --reload >/dev/null 2>&1 || true
        echo -e "      ${BGREEN}✓ Port ${DEFAULT_PORT} diizinkan pada firewalld.${NC}"
    fi
}

print_success() {
    # Dapatkan IP server
    SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}') || SERVER_IP="127.0.0.1"

    echo
    echo -e "${BGREEN}╔════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BGREEN}║${NC}  ${BGREEN}${BOLD}✓ INSTALASI SIPENDOSA DI SERVER LINUX BERHASIL DILAKUKAN!${NC}             ${BGREEN}║${NC}"
    echo -e "${BGREEN}╠════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BGREEN}║${NC}  • Lokasi Direktori : ${INSTALL_DIR}                                            ${BGREEN}║${NC}"
    echo -e "${BGREEN}║${NC}  • Status Service   : ${BGREEN}AKTIF (systemctl status ${APP_NAME})${NC}                         ${BGREEN}║${NC}"
    echo -e "${BGREEN}║${NC}  • Web Dashboard    : ${BYELLOW}http://${SERVER_IP}:${DEFAULT_PORT}${NC}                                  ${BGREEN}║${NC}"
    echo -e "${BGREEN}║${NC}  • Cek Log Realtime : ${GRAY}sudo journalctl -u ${APP_NAME} -f${NC}                              ${BGREEN}║${NC}"
    echo -e "${BGREEN}╚════════════════════════════════════════════════════════════════════════╝${NC}"
    echo
    echo -e "${BYELLOW}Langkah Berikutnya:${NC}"
    echo -e "1. Buka browser Anda di ${BGREEN}http://${SERVER_IP}:${DEFAULT_PORT}${NC}"
    echo -e "2. Inisialisasi akun SuperAdmin pertama Anda."
    echo -e "3. Tautkan WhatsApp melalui pemindaian QR Code di browser atau log terminal."
    echo
}

main() {
    print_banner
    check_root
    detect_arch
    find_binary
    setup_directories
    setup_env
    setup_systemd
    setup_firewall
    print_success
}

main "$@"
