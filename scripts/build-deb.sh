#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Debian / Ubuntu (.deb) Package Builder
# Menghasilkan file .deb standar menggunakan dpkg-deb atau native ar/tar
# ==============================================================================

set -e

APP_NAME="sipen"
PACKAGE_NAME="sipendosa"
VERSION="1.1.2"
ARCH="${1:-amd64}" # amd64 atau arm64

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${SCRIPT_DIR}/dist"
PKG_ROOT="${DIST_DIR}/pkg_deb_${ARCH}"
DEB_FILE="${DIST_DIR}/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

echo "==> Membangun paket Debian/Ubuntu (${ARCH}): ${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

# 1. Pastikan binary Linux tersedia
mkdir -p "${DIST_DIR}/bin"
BIN_SOURCE="${DIST_DIR}/bin/sipen_linux_${ARCH}"

if [ ! -f "$BIN_SOURCE" ]; then
    echo "==> Mengompilasi binary Linux (${ARCH})..."
    CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build -ldflags="-s -w" -o "$BIN_SOURCE" ./cmd/sipen
fi

# 2. Bersihkan dan susun pohon direktori paket
rm -rf "$PKG_ROOT"
mkdir -p "${PKG_ROOT}/opt/sipen/data"
mkdir -p "${PKG_ROOT}/opt/sipen/session"
mkdir -p "${PKG_ROOT}/etc/systemd/system"
mkdir -p "${PKG_ROOT}/DEBIAN"

# Salin binary
cp -f "$BIN_SOURCE" "${PKG_ROOT}/opt/sipen/sipen"
chmod 755 "${PKG_ROOT}/opt/sipen/sipen"

# Salin systemd service
cp -f "${SCRIPT_DIR}/sipen.service" "${PKG_ROOT}/etc/systemd/system/sipen.service"

# Buat DEBIAN/control
cat <<EOF > "${PKG_ROOT}/DEBIAN/control"
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${ARCH}
Maintainer: SiPenDosa Team <support@sipendosa.local>
Description: Sistem Pengingat Dosen Saatnya (SiPenDosa)
 Advanced WhatsApp Assistant Bot untuk pengingat jadwal kuliah otomatis.
 Dilengkapi rate-limiter anti-ban, pure-Go SQLite WAL, dan Web Dashboard modern.
EOF

# Buat DEBIAN/postinst
cat <<'EOF' > "${PKG_ROOT}/DEBIAN/postinst"
#!/bin/sh
set -e

# Buat user sistem sipen jika belum ada
if ! id -u sipen >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin sipen || true
fi

chown -R sipen:sipen /opt/sipen 2>/dev/null || true

# Buat .env default jika belum ada
if [ ! -f /opt/sipen/.env ]; then
    SECRET=$(head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n' 2>/dev/null || echo "sipen-secure-random-token-32char-len")
    cat <<ENV > /opt/sipen/.env
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
    chmod 600 /opt/sipen/.env
    chown sipen:sipen /opt/sipen/.env 2>/dev/null || true
fi

# Reload systemd
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl enable sipen || true
    systemctl restart sipen || true
fi

exit 0
EOF
chmod 755 "${PKG_ROOT}/DEBIAN/postinst"

# Buat DEBIAN/prerm
cat <<'EOF' > "${PKG_ROOT}/DEBIAN/prerm"
#!/bin/sh
set -e
if [ -d /run/systemd/system ]; then
    systemctl stop sipen || true
fi
exit 0
EOF
chmod 755 "${PKG_ROOT}/DEBIAN/prerm"

# 3. Kemas menjadi file .deb
if command -v dpkg-deb >/dev/null 2>&1; then
    dpkg-deb --build "$PKG_ROOT" "$DEB_FILE"
else
    # Gunakan pure Go packager (kompatibel 100% dengan standard ar & dpkg lintas platform)
    echo "==> Mengemas .deb menggunakan pure Go Debian packager..."
    TEMP_BUILD="${DIST_DIR}/.deb_tmp_${ARCH}"
    rm -rf "$TEMP_BUILD"
    mkdir -p "$TEMP_BUILD"

    echo "2.0" > "${TEMP_BUILD}/debian-binary"

    # Buat control.tar.gz
    (cd "${PKG_ROOT}/DEBIAN" && tar -czf "${TEMP_BUILD}/control.tar.gz" ./*)

    # Buat data.tar.gz
    (cd "$PKG_ROOT" && tar -czf "${TEMP_BUILD}/data.tar.gz" --exclude='./DEBIAN' opt etc)

    # Gabungkan menjadi file .deb menggunakan helper Go murni
    go run "${SCRIPT_DIR}/scripts/debpack/debpack.go" "$DEB_FILE" "${TEMP_BUILD}/debian-binary" "${TEMP_BUILD}/control.tar.gz" "${TEMP_BUILD}/data.tar.gz"
    rm -rf "$TEMP_BUILD"
fi

rm -rf "$PKG_ROOT"
echo "==> Paket Debian/Ubuntu berhasil dibuat: ${DEB_FILE}"
