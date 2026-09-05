#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — macOS Native App Builder (.app)
# Menggabungkan Go Core Engine, C++17 Socket Bridge, dan Swift AppKit Controller
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${SCRIPT_DIR}/dist"
APP_DIR="${DIST_DIR}/SiPenDosa.app"
CONTENTS_DIR="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"
ASSETS_DIR="${SCRIPT_DIR}/assets/macos"

echo "========================================================================"
echo "⚡ MEMBANGUN NATIVE MACOS APP: SiPenDosa.app"
echo "========================================================================"

mkdir -p "${DIST_DIR}" "${ASSETS_DIR}"

# 1. Pastikan icon dan assets visual tersedia
if [ ! -f "${ASSETS_DIR}/AppIcon.icns" ]; then
    echo "==> Men-generate aset grafis macOS (AppIcon & DMG Background)..."
    go run "${SCRIPT_DIR}/scripts/generate_assets/main.go" "${ASSETS_DIR}/AppIcon.png" "${ASSETS_DIR}/dmg_background.png"

    mkdir -p "${ASSETS_DIR}/AppIcon.iconset"
    sips -z 16 16     "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_16x16.png" >/dev/null 2>&1
    sips -z 32 32     "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_16x16@2x.png" >/dev/null 2>&1
    sips -z 32 32     "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_32x32.png" >/dev/null 2>&1
    sips -z 64 64     "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_32x32@2x.png" >/dev/null 2>&1
    sips -z 128 128   "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_128x128.png" >/dev/null 2>&1
    sips -z 256 256   "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_128x128@2x.png" >/dev/null 2>&1
    sips -z 256 256   "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_256x256.png" >/dev/null 2>&1
    sips -z 512 512   "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_256x256@2x.png" >/dev/null 2>&1
    sips -z 512 512   "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_512x512.png" >/dev/null 2>&1
    sips -z 1024 1024 "${ASSETS_DIR}/AppIcon.png" --out "${ASSETS_DIR}/AppIcon.iconset/icon_512x512@2x.png" >/dev/null 2>&1
    iconutil -c icns "${ASSETS_DIR}/AppIcon.iconset" -o "${ASSETS_DIR}/AppIcon.icns"
    rm -rf "${ASSETS_DIR}/AppIcon.iconset"
fi

# 2. Pastikan Go binary untuk macOS tersedia
mkdir -p "${DIST_DIR}/bin"
GO_ARM64="${DIST_DIR}/bin/sipen_darwin_arm64"
GO_AMD64="${DIST_DIR}/bin/sipen_darwin_amd64"
GO_UNIVERSAL="${DIST_DIR}/bin/sipen_darwin_universal"

if [ ! -f "$GO_ARM64" ]; then
    echo "==> Mengompilasi Go backend untuk darwin/arm64..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "$GO_ARM64" ./cmd/sipen
fi

if [ ! -f "$GO_AMD64" ]; then
    echo "==> Mengompilasi Go backend untuk darwin/amd64..."
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "$GO_AMD64" ./cmd/sipen
fi

# Gabungkan menjadi Universal Mach-O Binary menggunakan lipo
if [ -f "$GO_ARM64" ] && [ -f "$GO_AMD64" ]; then
    echo "==> Menggabungkan binary Go menjadi Universal Mach-O (arm64 + amd64)..."
    lipo -create -output "$GO_UNIVERSAL" "$GO_ARM64" "$GO_AMD64"
else
    cp -f "$GO_AMD64" "$GO_UNIVERSAL"
fi

# 3. Kompilasi C++17 Bridge
echo "==> Mengompilasi C++17 High-Performance Bridge..."
clang++ -std=c++17 -O2 -c "${SCRIPT_DIR}/native/bridge/sipen_bridge.cpp" -o "${SCRIPT_DIR}/native/bridge/sipen_bridge.o"

# 4. Kompilasi Swift Companion App
echo "==> Mengompilasi Swift Native Controller..."
SWIFT_BIN="${DIST_DIR}/SiPenDosa_swift_bin"
swiftc \
    -import-objc-header "${SCRIPT_DIR}/native/macos/SiPenDosa-Bridging-Header.h" \
    "${SCRIPT_DIR}/native/bridge/sipen_bridge.o" \
    "${SCRIPT_DIR}/native/macos/SiPenDosaApp.swift" \
    -Xlinker -lc++ \
    -O \
    -o "$SWIFT_BIN"

# 5. Susun struktur bundle SiPenDosa.app
echo "==> Menyusun struktur direktori SiPenDosa.app..."
rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

# Salin binary controller
cp -f "$SWIFT_BIN" "${MACOS_DIR}/SiPenDosa"
chmod 755 "${MACOS_DIR}/SiPenDosa"
rm -f "$SWIFT_BIN"

# Salin binary backend Go core ke Resources
cp -f "$GO_UNIVERSAL" "${RESOURCES_DIR}/sipen"
chmod 755 "${RESOURCES_DIR}/sipen"

# Salin Icon
cp -f "${ASSETS_DIR}/AppIcon.icns" "${RESOURCES_DIR}/AppIcon.icns"

# Buat Info.plist
cat <<EOF > "${CONTENTS_DIR}/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>id</string>
    <key>CFBundleExecutable</key>
    <string>SiPenDosa</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.sipendosa.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>SiPenDosa</string>
    <key>CFBundleDisplayName</key>
    <string>SiPenDosa</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSHumanReadableCopyright</key>
    <string>© 2026 SiPenDosa Team. OverPower Edition.</string>
</dict>
</plist>
EOF

echo "✓ Native macOS App berhasil dibuat: ${APP_DIR}"
