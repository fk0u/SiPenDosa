#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Android Handphone Server Setup (Termux Edition)
# Menjadikan smartphone Android sebagai server pengingat WhatsApp 24/7 mandiri
# ==============================================================================

# Re-eksekusi dengan Termux bash jika dipanggil dengan sh biasa
if [ -z "$BASH_VERSION" ]; then
    if [ -x "/data/data/com.termux/files/usr/bin/bash" ]; then
        exec /data/data/com.termux/files/usr/bin/bash "$0" "$@"
    fi
fi

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

clear 2>/dev/null || true
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

# 3. Deteksi Direktori Skrip Secara Tangguh (Mendukung bash, sh, curl | bash, dan symlink)
SCRIPT_SOURCE="${BASH_SOURCE[0]:-$0}"
if [ -n "$SCRIPT_SOURCE" ] && [ "$SCRIPT_SOURCE" != "bash" ] && [ "$SCRIPT_SOURCE" != "sh" ] && [ -e "$SCRIPT_SOURCE" ]; then
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" 2>/dev/null && pwd)"
else
    SCRIPT_DIR="$(pwd)"
fi

# 4. Salin / Pasang Binary SiPenDosa
echo -e "${BYELLOW}[3/5] Memasang binary SiPenDosa engine...${NC}"

INSTALLED=0

# A. Cek file binary prekompilasi lokal yang ada di folder rilis / build
CANDIDATE_BINS=(
    "$SCRIPT_DIR/dist/bin/sipen_linux_${BIN_ARCH}"
    "$SCRIPT_DIR/dist/bin/sipen_android_${BIN_ARCH}"
    "$SCRIPT_DIR/bin/sipen_linux_${BIN_ARCH}"
    "$SCRIPT_DIR/bin/sipen_android_${BIN_ARCH}"
    "$SCRIPT_DIR/${APP_NAME}_linux_${BIN_ARCH}"
    "$SCRIPT_DIR/${APP_NAME}_android_${BIN_ARCH}"
    "$SCRIPT_DIR/${APP_NAME}"
)

for bin_path in "${CANDIDATE_BINS[@]}"; do
    if [ -f "$bin_path" ]; then
        echo -e "      ${GREEN}✓ Menggunakan binary lokal: $bin_path${NC}"
        cp -f "$bin_path" "$INSTALL_DIR/$APP_NAME"
        INSTALLED=1
        break
    fi
done

# B. Jika binary lokal tidak ada, cek apakah repositori sumber tersedia dan Go terpasang
if [ "$INSTALLED" -eq 0 ]; then
    if [ -d "$SCRIPT_DIR/cmd/sipen" ] && [ -f "$SCRIPT_DIR/go.mod" ] && command -v go >/dev/null 2>&1; then
        echo -e "      Mengompilasi binary langsung di Termux dari kode sumber..."
        (cd "$SCRIPT_DIR" && CGO_ENABLED=0 go build -ldflags="-s -w" -o "$INSTALL_DIR/$APP_NAME" ./cmd/sipen)
        INSTALLED=1
    fi
fi

# C. Jika belum terpasang, coba unduh paket rilis resmi dari GitHub
if [ "$INSTALLED" -eq 0 ]; then
    echo -e "      Mengunduh paket rilis resmi SiPenDosa untuk ${BIN_ARCH} dari GitHub..."
    RELEASE_TAG="v1.0.0"
    TMP_DL="/tmp/sipen_dl_$$"
    mkdir -p "$TMP_DL"

    RAW_BIN_URL="https://github.com/fk0u/SiPenDosa/releases/download/${RELEASE_TAG}/sipen_linux_${BIN_ARCH}"
    DEB_URL="https://github.com/fk0u/SiPenDosa/releases/download/${RELEASE_TAG}/sipendosa_1.0.0_${BIN_ARCH}.deb"

    # 1. Coba download standalone binary langsung
    if curl -fsSL -o "$TMP_DL/sipen_bin" "$RAW_BIN_URL" 2>/dev/null && [ -s "$TMP_DL/sipen_bin" ]; then
        mv -f "$TMP_DL/sipen_bin" "$INSTALL_DIR/$APP_NAME"
        INSTALLED=1
        echo -e "      ${GREEN}✓ Berhasil mengunduh standalone binary resmi!${NC}"
    # 2. Jika tidak ada, unduh paket .deb dan ekstrak binary
    elif curl -fsSL -o "$TMP_DL/sipen.deb" "$DEB_URL" 2>/dev/null && [ -s "$TMP_DL/sipen.deb" ]; then
        if command -v dpkg >/dev/null 2>&1; then
            dpkg -x "$TMP_DL/sipen.deb" "$TMP_DL/extracted"
            if [ -f "$TMP_DL/extracted/opt/sipen/sipen" ]; then
                cp -f "$TMP_DL/extracted/opt/sipen/sipen" "$INSTALL_DIR/$APP_NAME"
                INSTALLED=1
                echo -e "      ${GREEN}✓ Berhasil mengekstrak binary dari paket resmi .deb!${NC}"
            fi
        elif command -v ar >/dev/null 2>&1 && command -v tar >/dev/null 2>&1; then
            (cd "$TMP_DL" && ar -x sipen.deb data.tar.gz && tar -xzf data.tar.gz)
            if [ -f "$TMP_DL/opt/sipen/sipen" ]; then
                cp -f "$TMP_DL/opt/sipen/sipen" "$INSTALL_DIR/$APP_NAME"
                INSTALLED=1
                echo -e "      ${GREEN}✓ Berhasil mengekstrak binary dari paket resmi .deb!${NC}"
            fi
        fi
    fi
    rm -rf "$TMP_DL"
fi

# D. Validasi akhir keberadaan binary
if [ "$INSTALLED" -eq 0 ] || [ ! -f "$INSTALL_DIR/$APP_NAME" ]; then
    echo -e "      ${RED}[ERROR] Binary sipen untuk arsitektur ${BIN_ARCH} tidak berhasil dipasang!${NC}"
    echo -e "      Petunjuk penyelesaian:"
    echo -e "      1. Pastikan koneksi internet aktif agar installer dapat mengunduh paket resmi."
    echo -e "      2. Jika ingin build mandiri, instal Go: 'pkg install golang git', lalu clone repo dan jalankan skrip."
    exit 1
fi

chmod +x "$INSTALL_DIR/$APP_NAME"
echo -e "      ${GREEN}✓ Binary server terpasang di: $INSTALL_DIR/$APP_NAME${NC}"

# 5. Generate .env jika belum ada
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

# 6. Buat Skrip Kontrol (Start, Stop, Status) & Wakelock
echo -e "${BYELLOW}[5/5] Membuat skrip pengontrol server Android...${NC}"

TERMUX_BASH="$(command -v bash 2>/dev/null || echo "/data/data/com.termux/files/usr/bin/bash")"

# Skrip Start (dengan auto-wakelock agar tidak tidur saat layar mati)
cat <<EOF > "$INSTALL_DIR/start.sh"
#!$TERMUX_BASH
cd "$INSTALL_DIR"

# Aktifkan Android Wakelock jika termux-wake-lock tersedia
if command -v termux-wake-lock >/dev/null 2>&1; then
    termux-wake-lock
    echo "[INFO] Android Wakelock diaktifkan (Server tetap menyala saat layar mati)."
fi

# Deteksi IP lokal Wi-Fi Android
WIFI_IP=\$( (ip -4 addr show wlan0 2>/dev/null || ifconfig wlan0 2>/dev/null) | awk '{for(i=1;i<=NF;i++) if(\$i=="inet") {split(\$(i+1),a,"/"); print a[1]; exit}}' )
[ -z "\$WIFI_IP" ] && WIFI_IP="localhost"

echo ""
echo "=================================================================="
echo "⚡ SIPENDOSA SERVER AKTIF DI SMARTPHONE ANDROID!"
echo "=================================================================="
echo "• Akses dari Browser HP Ini : http://localhost:8473"
echo "• Akses dari Laptop / Wi-Fi  : http://\${WIFI_IP}:8473"
echo "• Tekan Ctrl + C untuk menghentikan server."
echo "=================================================================="
echo ""

exec "$INSTALL_DIR/$APP_NAME"
EOF
chmod +x "$INSTALL_DIR/start.sh"

# Skrip Background Runner (daemon hening)
cat <<EOF > "$INSTALL_DIR/start-bg.sh"
#!$TERMUX_BASH
cd "$INSTALL_DIR"

if pgrep -f "$INSTALL_DIR/$APP_NAME" >/dev/null 2>&1; then
    echo "SiPenDosa server sudah berjalan!"
    exit 0
fi

if command -v termux-wake-lock >/dev/null 2>&1; then
    termux-wake-lock
fi

nohup "$INSTALL_DIR/$APP_NAME" > "$INSTALL_DIR/sipen.log" 2>&1 &
echo "✓ SiPenDosa berjalan di latar belakang! PID: \$!"
echo "• Buka http://localhost:8473 di browser."
echo "• Periksa log: tail -f $INSTALL_DIR/sipen.log"
EOF
chmod +x "$INSTALL_DIR/start-bg.sh"

# Skrip Stop
cat <<EOF > "$INSTALL_DIR/stop.sh"
#!$TERMUX_BASH
pkill -f "$INSTALL_DIR/$APP_NAME" || true
if command -v termux-wake-unlock >/dev/null 2>&1; then
    termux-wake-unlock
fi
echo "✓ SiPenDosa server dihentikan dan Wakelock dilepas."
EOF
chmod +x "$INSTALL_DIR/stop.sh"

# Shortcut di PATH Termux jika ada $PREFIX/bin
PREFIX_DIR="${PREFIX:-/data/data/com.termux/files/usr}"
if [ -d "$PREFIX_DIR/bin" ]; then
    ln -sf "$INSTALL_DIR/start.sh" "$PREFIX_DIR/bin/sipen-start"
    ln -sf "$INSTALL_DIR/start-bg.sh" "$PREFIX_DIR/bin/sipen-bg"
    ln -sf "$INSTALL_DIR/stop.sh" "$PREFIX_DIR/bin/sipen-stop"

    # Shortcut perintah langsung 'sipen'
    cat <<EOF > "$PREFIX_DIR/bin/sipen"
#!$TERMUX_BASH
cd "$INSTALL_DIR"
exec "$INSTALL_DIR/$APP_NAME" "\$@"
EOF
    chmod +x "$PREFIX_DIR/bin/sipen"
fi

# Ambil IP Wi-Fi
WIFI_IP=$( (ip -4 addr show wlan0 2>/dev/null || ifconfig wlan0 2>/dev/null) | awk '{for(i=1;i<=NF;i++) if($i=="inet") {split($(i+1),a,"/"); print a[1]; exit}}' )
[ -z "$WIFI_IP" ] && WIFI_IP="IP_WIFI_HANDPHONE"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   ✓ SMARTPHONE ANDROID ANDA SIAP MENJADI SERVER SIPENDOSA!             ║${NC}"
echo -e "${GREEN}╠════════════════════════════════════════════════════════════════════════╣${NC}"
echo -e "║  • Lokasi Server   : $INSTALL_DIR"
echo -e "║  • Akses dari HP   : ${BYELLOW}http://localhost:8473${NC}"
echo -e "║  • Akses dr Laptop : ${BYELLOW}http://${WIFI_IP}:8473${NC}"
echo -e "║"
echo -e "║  Perintah Penggunaan di Termux:"
echo -e "║  - Jalankan Foreground : ${BRED}sipen-start${NC} (atau $INSTALL_DIR/start.sh)"
echo -e "║  - Jalankan Background : ${BYELLOW}sipen-bg${NC}    (atau $INSTALL_DIR/start-bg.sh)"
echo -e "║  - Matikan Server      : ${YELLOW}sipen-stop${NC}  (atau $INSTALL_DIR/stop.sh)"
echo -e "║  - Perintah Langsung   : ${GREEN}sipen${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

read -p "Apakah Anda ingin langsung menyalakan server SiPenDosa sekarang? (y/N) " JAWABAN || JAWABAN="n"
case "$JAWABAN" in
    [yY][eE][sS]|[yY])
        "$INSTALL_DIR/start.sh"
        ;;
    *)
        echo "Untuk menyalakan sewaktu-waktu, jalankan: sipen-start"
        ;;
esac
