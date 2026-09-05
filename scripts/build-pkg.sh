#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Apple Installer Package (.pkg) Builder
# Menghasilkan installer wizard resmi macOS menggunakan pkgbuild & productbuild
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${SCRIPT_DIR}/dist"
APP_PATH="${DIST_DIR}/SiPenDosa.app"
PKG_OUTPUT="${DIST_DIR}/SiPenDosa-1.0.0-Installer.pkg"
TMP_DIR="${DIST_DIR}/.pkg_build_tmp"
SCRIPTS_DIR="${TMP_DIR}/scripts"
RESOURCES_DIR="${TMP_DIR}/resources"

echo "========================================================================"
echo "📦 MEMBANGUN APPLE INSTALLER PACKAGE: SiPenDosa.pkg"
echo "========================================================================"

# 1. Pastikan SiPenDosa.app sudah terbangun
if [ ! -d "$APP_PATH" ]; then
    "${SCRIPT_DIR}/scripts/build-macos-app.sh"
fi

rm -rf "$TMP_DIR"
mkdir -p "$SCRIPTS_DIR" "$RESOURCES_DIR"

# 2. Siapkan skrip postinstall
cat <<'EOF' > "${SCRIPTS_DIR}/postinstall"
#!/bin/bash
# postinstall untuk SiPenDosa macOS Installer

# Ambil user aktif konsol saat ini
CURRENT_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "$USER")
USER_HOME=$(eval echo "~$CURRENT_USER")

SIPEN_DATA_DIR="${USER_HOME}/.local/share/sipendosa"
LAUNCH_AGENTS_DIR="${USER_HOME}/Library/LaunchAgents"
PLIST_FILE="${LAUNCH_AGENTS_DIR}/com.sipendosa.daemon.plist"

mkdir -p "${SIPEN_DATA_DIR}/data"
mkdir -p "${SIPEN_DATA_DIR}/session"
mkdir -p "${LAUNCH_AGENTS_DIR}"

# Generate .env jika belum ada
if [ ! -f "${SIPEN_DATA_DIR}/.env" ]; then
    SECRET=$(head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n' 2>/dev/null || echo "sipen-secure-random-token-32char-len")
    cat <<ENV > "${SIPEN_DATA_DIR}/.env"
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
    chmod 600 "${SIPEN_DATA_DIR}/.env"
fi

# Salin binary backend ke data dir sebagai referensi fallback jika diperlukan
if [ -f "/Applications/SiPenDosa.app/Contents/Resources/sipen" ]; then
    cp -f "/Applications/SiPenDosa.app/Contents/Resources/sipen" "${SIPEN_DATA_DIR}/sipen"
    chmod 755 "${SIPEN_DATA_DIR}/sipen"
fi

# Buat LaunchAgent plist
cat <<PLIST > "${PLIST_FILE}"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sipendosa.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/SiPenDosa.app/Contents/Resources/sipen</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${SIPEN_DATA_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${SIPEN_DATA_DIR}/sipen.log</string>
    <key>StandardErrorPath</key>
    <string>${SIPEN_DATA_DIR}/sipen.log</string>
</dict>
</plist>
PLIST

chown -R "${CURRENT_USER}" "${SIPEN_DATA_DIR}" "${PLIST_FILE}" 2>/dev/null || true

exit 0
EOF
chmod 755 "${SCRIPTS_DIR}/postinstall"

# 3. Siapkan Dokumen Welcome & Filosofi (HTML/RichText)
cat <<'EOF' > "${RESOURCES_DIR}/welcome.html"
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
body {
    font-family: -apple-system, BlinkMacSystemFont, "Helvetica Neue", sans-serif;
    color: #1e293b;
    background-color: #ffffff;
    line-height: 1.5;
    padding: 10px;
}
h1 {
    color: #e11d48;
    font-size: 22px;
    margin-bottom: 4px;
}
h3 {
    color: #f59e0b;
    font-size: 14px;
    margin-top: 0;
    font-weight: 600;
}
p {
    font-size: 13px;
    color: #334155;
}
.box {
    background-color: #fff1f2;
    border-left: 4px solid #e11d48;
    padding: 10px 14px;
    margin: 14px 0;
    border-radius: 4px;
    font-size: 12.5px;
}
ul {
    font-size: 12.5px;
    color: #334155;
    padding-left: 20px;
}
li {
    margin-bottom: 4px;
}
</style>
</head>
<body>
<h1>SiPenDosa — OverPower Edition</h1>
<h3>Sistem Pengingat Dosen Saatnya (macOS Native Universal)</h3>

<p>Selamat datang di penginstal resmi <strong>SiPenDosa</strong> untuk Apple macOS.</p>

<div class="box">
<strong>Filosofi & Karakter:</strong><br>
<em>"Asisten otomasi yang rela menanggung beban 'berdosa' demi mengingatkan dosen dengan sopan dan terjadwal, agar mahasiswa tidak perlu lagi merasa sungkan!"</em>
</div>

<p>Paket instalasi ini akan memasang:</p>
<ul>
    <li><strong>SiPenDosa.app</strong> di folder <code>/Applications</code> (Companion Menu Bar App).</li>
    <li><strong>Go Core Backend Engine</strong> + C++17 Low-Latency Socket Bridge.</li>
    <li>Konfigurasi otomatis <code>~/.local/share/sipendosa/.env</code> dengan token sesi unik 32-karakter.</li>
    <li>Registrasi <strong>macOS LaunchAgent</strong> untuk opsi layanan pengingat 24/7 di latar belakang.</li>
</ul>

<p>Klik <strong>Lanjutkan (Continue)</strong> untuk memulai proses instalasi.</p>
</body>
</html>
EOF

# 4. Bangun Component Package menggunakan pkgbuild
COMPONENT_PKG="${TMP_DIR}/SiPenDosaComponent.pkg"
echo "==> Mengemas SiPenDosa.app dengan pkgbuild..."
pkgbuild \
    --component "$APP_PATH" \
    --install-location "/Applications" \
    --scripts "$SCRIPTS_DIR" \
    --identifier "com.sipendosa.app.pkg" \
    --version "1.0.0" \
    "$COMPONENT_PKG"

# 5. Buat Distribution.xml
DISTRIBUTION_XML="${TMP_DIR}/Distribution.xml"
cat <<EOF > "$DISTRIBUTION_XML"
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
    <title>SiPenDosa (OverPower Edition)</title>
    <welcome file="welcome.html" mime-type="text/html"/>
    <options customize="never" require-scripts="false" hostArchitectures="x86_64,arm64"/>
    <domains enable_anywhere="false" enable_currentUserHome="false" enable_localSystem="true"/>
    <choices-outline>
        <line choice="default">
            <line choice="com.sipendosa.app.pkg"/>
        </line>
    </choices-outline>
    <choice id="default"/>
    <choice id="com.sipendosa.app.pkg" visible="false">
        <pkg-ref id="com.sipendosa.app.pkg"/>
    </choice>
    <pkg-ref id="com.sipendosa.app.pkg" version="1.0.0" onConclusion="none">SiPenDosaComponent.pkg</pkg-ref>
</installer-gui-script>
EOF

# 6. Bangun Installer Package akhir menggunakan productbuild
echo "==> Mengompilasi Distribution Package dengan productbuild..."
productbuild \
    --distribution "$DISTRIBUTION_XML" \
    --package-path "$TMP_DIR" \
    --resources "$RESOURCES_DIR" \
    "$PKG_OUTPUT"

rm -rf "$TMP_DIR"
echo "✓ Apple Installer Package (.pkg) berhasil dibuat: ${PKG_OUTPUT}"
