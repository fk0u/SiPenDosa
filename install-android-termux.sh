#!/data/data/com.termux/files/usr/bin/bash
# ==============================================================================
# SiPenDosa — Android Handphone Server Setup (Termux Edition)
# Menjadikan smartphone Android sebagai server pengingat WhatsApp 24/7 mandiri
# ==============================================================================

set -e

APP_NAME="sipen"
INSTALL_DIR="$HOME/.sipendosa"
DATA_DIR="$INSTALL_DIR/data"
SESSION_DIR="$INSTALL_DIR/session"

# Warna Terminal (Scarlet Ember & Burnished Gold)
RED='\033[0;31m'
BRED='\033[1;31m'
YELLOW='\033[0;33m'
BYELLOW='\033[1;33m'
GREEN='\033[0;32m'
GRAY='\033[0;90m'
NC='\033[0m'

clear
echo -e "${BRED}  ███████╗██╗██████╗ ███████╗███╗   ██╗██████╗  ██████╗ ███████╗ █████╗ ${NC}"
echo -e "${RED}  ██╔════╝██║██╔══██╗██╔════╝████╗  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗${NC}"
echo -e "${BYELLOW}  ███████╗██║██████╔╝█████╗  ██╔██╗ ██║██║  ██║██║   ██║███████╗███████║${NC}"
echo -e "${YELLOW}  ╚════██║██║██╔═══╝ ██╔══╝  ██║╚██╗██║██║  ██║██║   ██║╚════██║██╔══██║${NC}"
echo -e "${RED}  ███████║██║██║     ███████╗██║ ╚████║██████╔╝╚██████╔╝███████║██║  ██║${NC}"
echo -e "${BRED}  ╚══════╝╚═╝╚═╝     ╚══════╝╚═╝  ╚═══╝╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═╝${NC}"
echo -e "${BYELLOW}  📱 ANDROID HANDPHONE 24/7 SERVER INSTALLER${NC}"
echo -e "${GRAY}  \"Jadikan smartphone Android Anda sebagai server pengingat dosen mandiri\"${NC}"
echo -e "${GRAY}  ----------------------------------------------------------------------${NC}"
echo ""

# 1. Periksa Arsitektur CPU Android
echo -e "${BYELLOW}[1/5] Memeriksa spesifikasi arsitektur Android...${NC}"
ARCH=$(uname -m)
case "$ARCH" in
    aarch64|arm64)
        BIN_ARCH="arm64"
        echo -e "      ${GREEN}✓ Arsitektur terdeteksi: 64-bit ARM (aarch64)${NC}"
        ;;
    armv7l|armv8l|arm)
        BIN_ARCH="arm"
        echo -e "      ${YELLOW}! Arsitektur terdeteksi: 32-bit ARM (${ARCH})${NC}"
        ;;
    x86_64)
        BIN_ARCH="amd64"
        echo -e "      ${GREEN}✓ Arsitektur terdeteksi: x86_64 (Emulator/Android-x86)${NC}"
        ;;
    *)
        echo -e "      ${RED}Arsitektur tidak dikenali: ${ARCH}${NC}"
        exit 1
        ;;
esac

# 2. Siapkan direktori penyimpanan
echo -e "${BYELLOW}[2/5] Menyiapkan direktori penyimpanan di $INSTALL_DIR...${NC}"
mkdir -p "$DATA_DIR" "$SESSION_DIR"

# 3. Salin / Pasang Binary
echo -e "${BYELLOW}[3/5] Memasang binary SiPenDosa engine...${NC}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ -f "$SCRIPT_DIR/dist/bin/sipen_linux_${BIN_ARCH}" ]; then
    cp -f "$SCRIPT_DIR/dist/bin/sipen_linux_${BIN_ARCH}" "$INSTALL_DIR/$APP_NAME"
elif [ -f "$SCRIPT_DIR/sipen" ]; then
    cp -f "$SCRIPT_DIR/sipen" "$INSTALL_DIR/$APP_NAME"
else
    # Fallback: build jika Go terinstal di Termux
    if command -v go >/dev/null 2>&1; then
        echo -e "      Mengompilasi binary langsung di Termux..."
        CGO_ENABLED=0 go build -ldflags="-s -w" -o "$INSTALL_DIR/$APP_NAME" "$SCRIPT_DIR/cmd/sipen"
    else
        echo -e "      ${RED}[ERROR] Binary sipen_linux_${BIN_ARCH} tidak ditemukan di folder rilis!${NC}"
        echo -e "      Pastikan Anda menyertakan binary hasil build 'make build-linux'."
        exit 1
    fi
fi
chmod +x "$INSTALL_DIR/$APP_NAME"
echo -e "      ${GREEN}✓ Binary server terpasang di: $INSTALL_DIR/$APP_NAME${NC}"

# 4. Generate .env jika belum ada
echo -e "${BYELLOW}[4/5] Menyiapkan file konfigurasi (.env)...${NC}"
ENV_FILE="$INSTALL_DIR/.env"
if [ ! -f "$ENV_FILE" ]; then
    SECRET=$(head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n' 2>/dev/null || echo "android-secret-sipendosa-key-32chars")
    cat <<ENV > "$ENV_FILE"
PORT=8473
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
ENV
    chmod 600 "$ENV_FILE"
    echo -e "      ${GREEN}✓ Token rahasia .env berhasil di-generate.${NC}"
fi

# 5. Buat Skrip Kontrol (Start, Stop, Status) & Wakelock
echo -e "${BYELLOW}[5/5] Membuat skrip pengontrol server Android...${NC}"

# Skrip Start (dengan auto-wakelock agar tidak tidur saat layar mati)
cat <<'EOF' > "$INSTALL_DIR/start.sh"
#!/data/data/com.termux/files/usr/bin/bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

# Aktifkan Android Wakelock jika termux-wake-lock tersedia
if command -v termux-wake-lock >/dev/null 2>&1; then
    termux-wake-lock
    echo "[INFO] Android Wakelock diaktifkan (Server tetap menyala saat layar mati)."
fi

# Deteksi IP lokal Wi-Fi Android
WIFI_IP=$(ip -4 addr show wlan0 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' || echo "localhost")

echo ""
echo "=================================================================="
echo "⚡ SIPENDOSA SERVER AKTIF DI SMARTPHONE ANDROID!"
echo "=================================================================="
echo "• Akses dari Browser HP Ini : http://localhost:8473"
echo "• Akses dari Laptop / Wi-Fi  : http://${WIFI_IP}:8473"
echo "• Tekan Ctrl + C untuk menghentikan server."
echo "=================================================================="
echo ""

exec ./sipen
EOF
chmod +x "$INSTALL_DIR/start.sh"

# Skrip Background Runner (daemon hening)
cat <<'EOF' > "$INSTALL_DIR/start-bg.sh"
#!/data/data/com.termux/files/usr/bin/bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

if pgrep -f "./sipen" >/dev/null 2>&1; then
    echo "SiPenDosa server sudah berjalan!"
    exit 0
fi

if command -v termux-wake-lock >/dev/null 2>&1; then
    termux-wake-lock
fi

nohup ./sipen > sipen.log 2>&1 &
echo "✓ SiPenDosa berjalan di latar belakang! PID: $!"
echo "• Buka http://localhost:8473 di browser."
echo "• Periksa log: tail -f $DIR/sipen.log"
EOF
chmod +x "$INSTALL_DIR/start-bg.sh"

# Skrip Stop
cat <<'EOF' > "$INSTALL_DIR/stop.sh"
#!/data/data/com.termux/files/usr/bin/bash
pkill -f "./sipen" || true
if command -v termux-wake-unlock >/dev/null 2>&1; then
    termux-wake-unlock
fi
echo "✓ SiPenDosa server dihentikan dan Wakelock dilepas."
EOF
chmod +x "$INSTALL_DIR/stop.sh"

# Shortcut di PATH Termux jika ada $PREFIX/bin
if [ -d "$PREFIX/bin" ]; then
    ln -sf "$INSTALL_DIR/start.sh" "$PREFIX/bin/sipen-start"
    ln -sf "$INSTALL_DIR/stop.sh" "$PREFIX/bin/sipen-stop"
fi

# Ambil IP Wi-Fi
WIFI_IP=$(ip -4 addr show wlan0 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' || echo "IP_WIFI_HANDPHONE")

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✓ SMARTPHONE ANDROID ANDA SIAP MENJADI SERVER SIPENDOSA!             ║${NC}"
echo -e "${GREEN}╠════════════════════════════════════════════════════════════════════════╣${NC}"
echo -e "║  • Lokasi Server   : $INSTALL_DIR"
echo -e "║  • Akses dari HP   : ${BYELLOW}http://localhost:8473${NC}"
echo -e "║  • Akses dr Laptop : ${BYELLOW}http://${WIFI_IP}:8473${NC}"
echo -e "║"
echo -e "║  Perintah Penggunaan di Termux:"
echo -e "║  - Nyalakan Server : ${BRED}$INSTALL_DIR/start.sh${NC} (atau ketik: sipen-start)"
echo -e "║  - Matikan Server  : ${YELLOW}$INSTALL_DIR/stop.sh${NC} (atau ketik: sipen-stop)"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

read -p "Apakah Anda ingin langsung menyalakan server SiPenDosa sekarang? (y/N) " JAWABAN
case "$JAWABAN" in
    [yY][eE][sS]|[yY])
        "$INSTALL_DIR/start.sh"
        ;;
    *)
        echo "Untuk menyalakan sewaktu-waktu, jalankan: $INSTALL_DIR/start.sh"
        ;;
esac
