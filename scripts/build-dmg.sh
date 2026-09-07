#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Apple Disk Image (.dmg) Builder
# Membuat disk image premium drag-and-drop dengan background visual dan icon retina
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${SCRIPT_DIR}/dist"
APP_PATH="${DIST_DIR}/SiPenDosa.app"
DMG_FINAL="${DIST_DIR}/SiPenDosa-1.1.1.dmg"
DMG_TMP="${DIST_DIR}/.tmp_sipen.dmg"
STAGING_DIR="${DIST_DIR}/.dmg_staging"
ASSETS_DIR="${SCRIPT_DIR}/assets/macos"

VOLUME_NAME="SiPenDosa"

echo "========================================================================"
echo "💿 MEMBANGUN APPLE DISK IMAGE: SiPenDosa.dmg"
echo "========================================================================"

# 1. Pastikan SiPenDosa.app sudah terbangun
if [ ! -d "$APP_PATH" ]; then
    "${SCRIPT_DIR}/scripts/build-macos-app.sh"
fi

# 2. Pastikan aset background & icon tersedia
if [ ! -f "${ASSETS_DIR}/dmg_background.png" ]; then
    go run "${SCRIPT_DIR}/scripts/generate_assets/main.go" "${ASSETS_DIR}/AppIcon.png" "${ASSETS_DIR}/dmg_background.png"
fi

# 3. Siapkan staging directory
rm -rf "$STAGING_DIR" "$DMG_TMP" "$DMG_FINAL"
mkdir -p "$STAGING_DIR"

echo "==> Menyalin SiPenDosa.app ke staging directory..."
cp -R "$APP_PATH" "$STAGING_DIR/"

echo "==> Membuat symlink Applications..."
ln -s /Applications "$STAGING_DIR/Applications"

echo "==> Memasang background image kustom..."
mkdir -p "$STAGING_DIR/.background"
cp -f "${ASSETS_DIR}/dmg_background.png" "$STAGING_DIR/.background/background.png"

if [ -f "${ASSETS_DIR}/AppIcon.icns" ]; then
    cp -f "${ASSETS_DIR}/AppIcon.icns" "$STAGING_DIR/.VolumeIcon.icns"
fi

# 4. Buat read-write DMG sementara
echo "==> Membuat master disk image sementara (UDRW)..."
hdiutil create \
    -srcfolder "$STAGING_DIR" \
    -volname "$VOLUME_NAME" \
    -fs HFS+ \
    -fsargs "-c c=64,a=16,e=16" \
    -format UDRW \
    -size 150m \
    "$DMG_TMP"

# 5. Mount DMG sementara untuk kustomisasi layout Finder
echo "==> Me-mount disk image sementara untuk pengaturan layout Finder..."
MOUNT_DIR=$(mktemp -d /tmp/sipen_mount.XXXXXX)
ATTACH_INFO=$(hdiutil attach -readwrite -noverify -noautoopen "$DMG_TMP")
DEV_NAME=$(echo "$ATTACH_INFO" | egrep '^/dev/' | head -n 1 | awk '{print $1}')
VOLUME_PATH="/Volumes/${VOLUME_NAME}"

# Set Volume Icon jika ada
if [ -f "${VOLUME_PATH}/.VolumeIcon.icns" ]; then
    SetFile -a C "${VOLUME_PATH}" 2>/dev/null || true
fi

# Atur tata letak Finder menggunakan AppleScript
echo "==> Menerapkan tata letak visual Finder (Ukuran: 640x440, Posisi: 160,240 -> 480,240)..."
osascript <<APPLESCRIPT || true
tell application "Finder"
    tell disk "${VOLUME_NAME}"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set the bounds of container window to {300, 150, 940, 590}
        set theViewOptions to the icon view options of container window
        set arrangement of theViewOptions to not arranged
        set icon size of theViewOptions to 110
        try
            set background picture of theViewOptions to file ".background:background.png"
        end try
        set position of item "SiPenDosa.app" of container window to {160, 240}
        set position of item "Applications" of container window to {480, 240}
        update without registering applications
        delay 1
        close
    end tell
end tell
APPLESCRIPT

# Pastikan cache tersinkron
sync
sleep 1

echo "==> Melepaskan mount disk image..."
hdiutil detach "$DEV_NAME" -force || true
sleep 1

# 6. Konversi ke Compressed Read-Only DMG (UDZO)
echo "==> Mengompresi ke format final Apple Disk Image (UDZO)..."
hdiutil convert \
    "$DMG_TMP" \
    -format UDZO \
    -imagekey zlib-level=9 \
    -o "$DMG_FINAL"

# Bersihkan temporary files
rm -rf "$DMG_TMP" "$STAGING_DIR" "$MOUNT_DIR"

echo "✓ Apple Disk Image (.dmg) premium berhasil dibuat: ${DMG_FINAL}"
ls -lh "$DMG_FINAL"
