#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Apple Installer Package (.pkg) Builder
# Menghasilkan installer wizard resmi macOS menggunakan pkgbuild & productbuild
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${SCRIPT_DIR}/dist"
APP_PATH="${DIST_DIR}/SiPenDosa.app"
PKG_OUTPUT="${DIST_DIR}/SiPenDosa-1.4.1-Installer.pkg"
TMP_DIR="${DIST_DIR}/.pkg_build_tmp"
SCRIPTS_DIR="${TMP_DIR}/scripts"
RESOURCES_DIR="${TMP_DIR}/resources"

echo "========================================================================"
echo "📦 MEMBANGUN APPLE INSTALLER PACKAGE: SiPenDosa.pkg"
echo "========================================================================"

if [ -x "${HOME}/go/bin/go" ]; then
    export PATH="${HOME}/go/bin:${PATH}"
elif [ -x "/usr/local/go/bin/go" ]; then
    export PATH="/usr/local/go/bin:${PATH}"
elif [ -x "/opt/homebrew/bin/go" ]; then
    export PATH="/opt/homebrew/bin:${PATH}"
fi

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
USER_UID=$(id -u "$CURRENT_USER" 2>/dev/null || echo "501")

SIPEN_DATA_DIR="${USER_HOME}/.local/share/sipendosa"
LAUNCH_AGENTS_DIR="${USER_HOME}/Library/LaunchAgents"
PLIST_FILE="${LAUNCH_AGENTS_DIR}/com.sipendosa.daemon.plist"

mkdir -p "${SIPEN_DATA_DIR}/data"
mkdir -p "${SIPEN_DATA_DIR}/session"
mkdir -p "${LAUNCH_AGENTS_DIR}"

# 1. Hentikan seluruh instance SiPenDosa dan backend lama yang sedang berjalan
killall -9 SiPenDosa 2>/dev/null || true
killall -9 sipen 2>/dev/null || true
lsof -ti :8473 | xargs kill -9 2>/dev/null || true

# 2. Unload daemon lama jika terdaftar di launchd
if [ -f "${PLIST_FILE}" ]; then
    launchctl asuser "${USER_UID}" launchctl unload "${PLIST_FILE}" 2>/dev/null || true
    launchctl asuser "${USER_UID}" launchctl bootout "gui/${USER_UID}" "${PLIST_FILE}" 2>/dev/null || true
fi

# 3. Generate .env jika belum ada
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
SIPEN_UI=web
ENV
    chmod 600 "${SIPEN_DATA_DIR}/.env"
fi

# 4. Sinkronisasikan binary backend terbaru ke seluruh lokasi instalasi yang mungkin ada
NEW_SIPEN="/Applications/SiPenDosa.app/Contents/Resources/sipen"
if [ -f "${NEW_SIPEN}" ]; then
    cp -f "${NEW_SIPEN}" "${SIPEN_DATA_DIR}/sipen"
    chmod 755 "${SIPEN_DATA_DIR}/sipen"

    # Jika user sebelumnya menginstal lewat install-macos.sh di ~/Applications/SiPenDosa
    if [ -d "${USER_HOME}/Applications/SiPenDosa" ]; then
        cp -f "${NEW_SIPEN}" "${USER_HOME}/Applications/SiPenDosa/sipen"
        chmod 755 "${USER_HOME}/Applications/SiPenDosa/sipen"
    fi
fi

# 5. Pastikan .env yang ada memiliki mode web (bukan terminal yang mati saat background)
if [ -f "${SIPEN_DATA_DIR}/.env" ]; then
    sed -i '' 's/^SIPEN_UI=terminal/SIPEN_UI=web/g' "${SIPEN_DATA_DIR}/.env" 2>/dev/null || true
fi

# 6. Buat dan perbarui LaunchAgent plist resmi
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
        <string>web</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>SIPEN_UI</key>
        <string>web</string>
    </dict>
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
if [ -d "${USER_HOME}/Applications/SiPenDosa" ]; then
    chown -R "${CURRENT_USER}" "${USER_HOME}/Applications/SiPenDosa" 2>/dev/null || true
fi

# 7. Muat ulang LaunchAgent di domain user yang aktif
launchctl asuser "${USER_UID}" launchctl load -w "${PLIST_FILE}" 2>/dev/null || true
launchctl asuser "${USER_UID}" launchctl bootstrap "gui/${USER_UID}" "${PLIST_FILE}" 2>/dev/null || true
launchctl asuser "${USER_UID}" launchctl kickstart -k "gui/${USER_UID}/com.sipendosa.daemon" 2>/dev/null || true

# 7. Luncurkan aplikasi native companion SiPenDosa.app
launchctl asuser "${USER_UID}" open -a "/Applications/SiPenDosa.app" 2>/dev/null || true

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
<h1>SiPenDosa — Universal Edition</h1>
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
    --version "1.4.1" \
    "$COMPONENT_PKG"

# 5. Buat Distribution.xml
DISTRIBUTION_XML="${TMP_DIR}/Distribution.xml"
cat <<EOF > "$DISTRIBUTION_XML"
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
    <title>SiPenDosa Universal Edition (macOS 12+)</title>
    <welcome file="welcome.html" mime-type="text/html"/>
    <options customize="never" require-scripts="false" hostArchitectures="x86_64,arm64"/>
    <volume-check>
        <allowed-os-versions>
            <os-version min="12.0"/>
        </allowed-os-versions>
    </volume-check>
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
    <pkg-ref id="com.sipendosa.app.pkg" version="1.4.1" onConclusion="none">SiPenDosaComponent.pkg</pkg-ref>
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
