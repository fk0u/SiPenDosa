#!/usr/bin/env bash
# ==============================================================================
# SiPenDosa — Official Universal Multi-Arch Android Builder (APK & AAB)
# Signed by: KOU
# Target: API 21+ (Android 5.0 Minimum, Up to Android 15)
# Architectures: ARM64 (arm64-v8a), ARMv7 (armeabi-v7a), x86_64, x86
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

echo "========================================================================"
echo "🤖 MEMBANGUN UNIVERSAL ANDROID APK & AAB (OFFICIAL TOOLCHAIN & SIGNED: KOU)"
echo "   Target: API 21+ (Android 5.0+ Universal) • Multi-Arch: arm7 / arm64 / x86"
echo "========================================================================"

JAVA_BIN="/usr/local/opt/openjdk/bin"
if [ ! -x "${JAVA_BIN}/javac" ]; then
    JAVA_BIN="$(dirname "$(command -v javac 2>/dev/null)")"
fi
export PATH="${JAVA_BIN}:${PATH}"

TOOLS_DIR="${HOME}/.android-sdk-tools"
mkdir -p "${TOOLS_DIR}"

BUILD_TOOLS_DIR="${TOOLS_DIR}/android-13"
ANDROID_JAR="${TOOLS_DIR}/android-33.jar"
BUNDLETOOL_JAR="${TOOLS_DIR}/bundletool.jar"

# 1. Pastikan Official Android SDK Tools & Platform Jar Tersedia
if [ ! -f "${ANDROID_JAR}" ]; then
    echo "==> Mengunduh Android API 33/32 Platform SDK (android.jar)..."
    curl -fsSL -o "${TOOLS_DIR}/platform-33.zip" https://dl.google.com/android/repository/platform-33_r02.zip
    unzip -q -o "${TOOLS_DIR}/platform-33.zip" "android-13/android.jar" -d "${TOOLS_DIR}/"
    mv -f "${TOOLS_DIR}/android-13/android.jar" "${ANDROID_JAR}"
    rm -f "${TOOLS_DIR}/platform-33.zip"
fi

if [ ! -x "${BUILD_TOOLS_DIR}/aapt" ] || [ ! -x "${BUILD_TOOLS_DIR}/d8" ] || [ ! -x "${BUILD_TOOLS_DIR}/aapt2" ]; then
    echo "==> Mengunduh Google Android Build-Tools v33.0.2..."
    curl -fsSL -o "${TOOLS_DIR}/build-tools.zip" https://dl.google.com/android/repository/build-tools_r33.0.2-macosx.zip
    unzip -q -o "${TOOLS_DIR}/build-tools.zip" -d "${TOOLS_DIR}/"
    rm -f "${TOOLS_DIR}/build-tools.zip"
fi

if [ ! -f "${BUNDLETOOL_JAR}" ]; then
    echo "==> Mengunduh Google Official Bundletool (untuk AAB)..."
    curl -fsSL -o "${BUNDLETOOL_JAR}" https://github.com/google/bundletool/releases/download/1.17.2/bundletool-all-1.17.2.jar
fi

# Buat tools executable
chmod +x "${BUILD_TOOLS_DIR}/aapt" "${BUILD_TOOLS_DIR}/aapt2" "${BUILD_TOOLS_DIR}/d8" "${BUILD_TOOLS_DIR}/zipalign" "${BUILD_TOOLS_DIR}/apksigner" 2>/dev/null || true

DIST_DIR="${ROOT_DIR}/dist"
BIN_DIR="${DIST_DIR}/bin"
mkdir -p "${BIN_DIR}"
mkdir -p mobile/android/app/src/main/res/drawable

# 2. Kompilasi 4 Arsitektur Go Binary (Universal Support)
echo "==> Mengompilasi binary core SiPenDosa untuk 4 arsitektur Android..."

# 2a. ARM64 (arm64-v8a)
echo "    • Mengompilasi ARM64 (arm64-v8a)..."
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags="-s -w" -o "${BIN_DIR}/sipen_android_arm64" ./cmd/sipen

# 2b. ARMv7 32-bit (armeabi-v7a)
echo "    • Mengompilasi ARMv7 32-bit (armeabi-v7a)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o "${BIN_DIR}/sipen_android_arm7" ./cmd/sipen

# 2c. x86_64 64-bit (Intel/AMD)
echo "    • Mengompilasi x86_64 (Intel 64-bit / Emulators)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${BIN_DIR}/sipen_android_x86_64" ./cmd/sipen

# 2d. x86 32-bit (Intel 32-bit)
echo "    • Mengompilasi x86 (Intel 32-bit)..."
CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o "${BIN_DIR}/sipen_android_x86" ./cmd/sipen

# 3. Salin ikon aplikasi
if [ -f "web/static/img/icon-192.png" ]; then
    cp web/static/img/icon-192.png mobile/android/app/src/main/res/drawable/icon.png
fi

# 4. Buat Keystore Resmi Bertanda Tangan 'KOU'
KEYSTORE_DIR="${ROOT_DIR}/mobile/android/keystore"
KEYSTORE="${KEYSTORE_DIR}/kou-release.keystore"
mkdir -p "${KEYSTORE_DIR}"

if [ ! -f "${KEYSTORE}" ]; then
    echo "==> Men-generate signing keystore resmi bertanda tangan 'KOU'..."
    "${JAVA_BIN}/keytool" -genkeypair \
        -keystore "${KEYSTORE}" \
        -storepass "sipendosa2026" \
        -keypass "sipendosa2026" \
        -alias "KOU" \
        -keyalg RSA \
        -keysize 2048 \
        -validity 36500 \
        -dname "CN=KOU, OU=KOU, O=KOU, L=Makassar, ST=Sulawesi Selatan, C=ID"
fi

# 5. Kompilasi Java Sources & Dalvik Executable (DEX)
echo "==> Mengompilasi Java classes dengan OpenJDK javac..."
TMP_BUILD="/tmp/sipen_android_build_$$"
rm -rf "${TMP_BUILD}"
mkdir -p "${TMP_BUILD}/classes"
mkdir -p "${TMP_BUILD}/dex"
mkdir -p "${TMP_BUILD}/apk_staging"
mkdir -p "${TMP_BUILD}/apk_staging/lib/arm64-v8a"
mkdir -p "${TMP_BUILD}/apk_staging/lib/armeabi-v7a"
mkdir -p "${TMP_BUILD}/apk_staging/lib/x86_64"
mkdir -p "${TMP_BUILD}/apk_staging/lib/x86"

"${JAVA_BIN}/javac" --release 8 -g:lines,source \
    -cp "${ANDROID_JAR}" \
    -d "${TMP_BUILD}/classes" \
    mobile/android/app/src/main/java/com/sipendosa/app/*.java

echo "==> Mengonversi bytecode ke Dalvik Executable (classes.dex) dengan D8..."
"${BUILD_TOOLS_DIR}/d8" \
    --min-api 21 \
    --lib "${ANDROID_JAR}" \
    --output "${TMP_BUILD}/dex" \
    "${TMP_BUILD}/classes/com/sipendosa/app"/*.class

# Susun seluruh biner arsitektur ke lib/<abi>/libsipen.so
cp "${BIN_DIR}/sipen_android_arm64" "${TMP_BUILD}/apk_staging/lib/arm64-v8a/libsipen.so"
cp "${BIN_DIR}/sipen_android_arm7" "${TMP_BUILD}/apk_staging/lib/armeabi-v7a/libsipen.so"
cp "${BIN_DIR}/sipen_android_x86_64" "${TMP_BUILD}/apk_staging/lib/x86_64/libsipen.so"
cp "${BIN_DIR}/sipen_android_x86" "${TMP_BUILD}/apk_staging/lib/x86/libsipen.so"

# ==============================================================================
# BAGIAN A: MEMBANGUN DAN MENANDATANGANI STANDALONE UNIVERSAL APK
# ==============================================================================
echo "==> [A] Memaketkan Universal APK dengan AAPT..."
"${BUILD_TOOLS_DIR}/aapt" package -f \
    -M mobile/android/app/src/main/AndroidManifest.xml \
    -S mobile/android/app/src/main/res \
    -I "${ANDROID_JAR}" \
    -F "${TMP_BUILD}/unaligned.apk"

cd "${TMP_BUILD}"
cp "${TMP_BUILD}/dex/classes.dex" ./classes.dex
"${BUILD_TOOLS_DIR}/aapt" add unaligned.apk classes.dex

# Masukkan seluruh native libraries ke APK
"${BUILD_TOOLS_DIR}/aapt" add unaligned.apk \
    apk_staging/lib/arm64-v8a/libsipen.so \
    apk_staging/lib/armeabi-v7a/libsipen.so \
    apk_staging/lib/x86_64/libsipen.so \
    apk_staging/lib/x86/libsipen.so

# Reorganize lib directory inside APK jika dibutuhkan oleh AAPT
(
    cd "${TMP_BUILD}/apk_staging"
    zip -q -u "${TMP_BUILD}/unaligned.apk" \
        lib/arm64-v8a/libsipen.so \
        lib/armeabi-v7a/libsipen.so \
        lib/x86_64/libsipen.so \
        lib/x86/libsipen.so
)

cd "${ROOT_DIR}"

echo "==> Menyelaraskan ZIP archive APK dengan zipalign (4-byte alignment)..."
"${BUILD_TOOLS_DIR}/zipalign" -p -f 4 "${TMP_BUILD}/unaligned.apk" "${TMP_BUILD}/aligned.apk"

echo "==> Menandatangani APK dengan APKSIGNER atas nama 'KOU' (V1, V2, V3 Scheme)..."
"${BUILD_TOOLS_DIR}/apksigner" sign \
    --ks "${KEYSTORE}" \
    --ks-pass "pass:sipendosa2026" \
    --key-pass "pass:sipendosa2026" \
    --ks-key-alias "KOU" \
    --v1-signing-enabled true \
    --v2-signing-enabled true \
    --v3-signing-enabled true \
    --out "${DIST_DIR}/SiPenDosa-Android.apk" \
    "${TMP_BUILD}/aligned.apk"

echo "==> Memverifikasi integritas tanda tangan APK..."
"${BUILD_TOOLS_DIR}/apksigner" verify --verbose "${DIST_DIR}/SiPenDosa-Android.apk"

# ==============================================================================
# BAGIAN B: MEMBANGUN DAN MENANDATANGANI ANDROID APP BUNDLE (.AAB)
# ==============================================================================
echo "==> [B] Membangun Android App Bundle (.aab) dengan AAPT2 & Bundletool..."

# B1. Compile resources dengan aapt2
mkdir -p "${TMP_BUILD}/aab/compiled_res"
"${BUILD_TOOLS_DIR}/aapt2" compile \
    --dir mobile/android/app/src/main/res \
    -o "${TMP_BUILD}/aab/compiled_res.zip"

# B2. Link resources dalam format protobuf (--proto-format)
mkdir -p "${TMP_BUILD}/aab/base"
"${BUILD_TOOLS_DIR}/aapt2" link \
    --proto-format \
    -o "${TMP_BUILD}/aab/base.zip" \
    -I "${ANDROID_JAR}" \
    --manifest mobile/android/app/src/main/AndroidManifest.xml \
    -R "${TMP_BUILD}/aab/compiled_res.zip" \
    --auto-add-overlay

# B3. Susun struktur bundle module (base/)
mkdir -p "${TMP_BUILD}/aab/staging"
unzip -q "${TMP_BUILD}/aab/base.zip" -d "${TMP_BUILD}/aab/staging"

# Buat folder manifest, dex, lib
mkdir -p "${TMP_BUILD}/aab/staging/manifest"
mkdir -p "${TMP_BUILD}/aab/staging/dex"
mkdir -p "${TMP_BUILD}/aab/staging/lib/arm64-v8a"
mkdir -p "${TMP_BUILD}/aab/staging/lib/armeabi-v7a"
mkdir -p "${TMP_BUILD}/aab/staging/lib/x86_64"
mkdir -p "${TMP_BUILD}/aab/staging/lib/x86"

mv -f "${TMP_BUILD}/aab/staging/AndroidManifest.xml" "${TMP_BUILD}/aab/staging/manifest/AndroidManifest.xml"
cp -f "${TMP_BUILD}/dex/classes.dex" "${TMP_BUILD}/aab/staging/dex/classes.dex"

cp -f "${BIN_DIR}/sipen_android_arm64" "${TMP_BUILD}/aab/staging/lib/arm64-v8a/libsipen.so"
cp -f "${BIN_DIR}/sipen_android_arm7" "${TMP_BUILD}/aab/staging/lib/armeabi-v7a/libsipen.so"
cp -f "${BIN_DIR}/sipen_android_x86_64" "${TMP_BUILD}/aab/staging/lib/x86_64/libsipen.so"
cp -f "${BIN_DIR}/sipen_android_x86" "${TMP_BUILD}/aab/staging/lib/x86/libsipen.so"

# Kemas base.zip final
(
    cd "${TMP_BUILD}/aab/staging"
    zip -q -r "${TMP_BUILD}/aab/base_module.zip" .
)

# B4. Build AAB dengan bundletool
echo "==> Mengompilasi final Android App Bundle (.aab)..."
"${JAVA_BIN}/java" -jar "${BUNDLETOOL_JAR}" build-bundle \
    --modules="${TMP_BUILD}/aab/base_module.zip" \
    --output="${TMP_BUILD}/unsigned.aab"

# B5. Tanda tangani AAB dengan jarsigner atas nama 'KOU'
echo "==> Menandatangani .aab dengan jarsigner atas nama 'KOU'..."
"${JAVA_BIN}/jarsigner" \
    -keystore "${KEYSTORE}" \
    -storepass "sipendosa2026" \
    -keypass "sipendosa2026" \
    "${TMP_BUILD}/unsigned.aab" \
    "KOU"

cp -f "${TMP_BUILD}/unsigned.aab" "${DIST_DIR}/SiPenDosa-Android.aab"

# B6. Validasi integritas AAB dengan bundletool
echo "==> Memvalidasi berkas .aab dengan bundletool..."
"${JAVA_BIN}/java" -jar "${BUNDLETOOL_JAR}" validate --bundle="${DIST_DIR}/SiPenDosa-Android.aab"

# Bersihkan temporary build directory
rm -rf "${TMP_BUILD}"

echo ""
echo "========================================================================"
echo "🎉 SELURUH PAKET RESMI ANDROID BERHASIL DIBUAT & DITANDATANGANI (KOU):"
echo "  • Standalone Universal APK : ${DIST_DIR}/SiPenDosa-Android.apk"
echo "  • Google App Bundle (AAB)  : ${DIST_DIR}/SiPenDosa-Android.aab"
echo "  • Signing Keystore         : ${KEYSTORE} (Alias: KOU)"
echo "  • Arsitektur Didukung      : arm64-v8a, armeabi-v7a (arm7), x86_64, x86"
echo "  • Minimum Versi Android    : Android 5.0+ (API 21+) — Mendukung Semua Versi Android"
echo "=============================================================================="
ls -lh "${DIST_DIR}/SiPenDosa-Android.apk" "${DIST_DIR}/SiPenDosa-Android.aab"
echo "========================================================================"
