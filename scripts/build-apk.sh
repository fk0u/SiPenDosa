#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

echo "========================================================================"
echo "🤖 MEMBANGUN NATIVE ANDROID APK MANDIRI UNIVERSAL (MULTI-ABI COMPATIBLE)"
echo "========================================================================"

JAVA_BIN="/usr/local/opt/openjdk/bin"
TOOLS_DIR="/Users/ghani/.android-tools"
BUILD_TOOLS="${TOOLS_DIR}/android-13"
ANDROID_JAR="${TOOLS_DIR}/android-33.jar"

export PATH="${JAVA_BIN}:${PATH}"

mkdir -p dist/bin
mkdir -p mobile/android/app/src/main/res/drawable

# 1. Kompilasi binary Go untuk multi-arsitektur Android (ARM64, ARMv7, x86_64)
echo "==> Mengompilasi Go binary untuk Android ARM64 (64-bit)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/bin/sipen_android_arm64 ./cmd/sipen

echo "==> Mengompilasi Go binary untuk Android ARMv7 (32-bit ponsel budget/lama)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o dist/bin/sipen_android_armv7 ./cmd/sipen

echo "==> Mengompilasi Go binary untuk Android x86_64 (Emulator / Chromebook)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/bin/sipen_android_amd64 ./cmd/sipen

# 2. Salin ikon aplikasi
if [ -f "web/static/img/icon-192.png" ]; then
    cp web/static/img/icon-192.png mobile/android/app/src/main/res/drawable/icon.png
fi

# 3. Kompilasi Java sources ke .class
echo "==> Mengompilasi Java classes dengan OpenJDK javac..."
TMP_BUILD="/tmp/sipen-apk-build"
rm -rf "$TMP_BUILD"
mkdir -p "${TMP_BUILD}/classes"
mkdir -p "${TMP_BUILD}/dex"
mkdir -p "${TMP_BUILD}/apk_staging"
mkdir -p "${TMP_BUILD}/apk_staging/assets"
mkdir -p "${TMP_BUILD}/apk_staging/lib/arm64-v8a"
mkdir -p "${TMP_BUILD}/apk_staging/lib/armeabi-v7a"
mkdir -p "${TMP_BUILD}/apk_staging/lib/x86_64"

"${JAVA_BIN}/javac" --release 8 \
    -cp "${ANDROID_JAR}" \
    -d "${TMP_BUILD}/classes" \
    mobile/android/app/src/main/java/com/sipendosa/app/*.java

# 4. Konversi .class ke classes.dex dengan D8
echo "==> Mengonversi bytecode ke Dalvik Executable (classes.dex) dengan D8..."
"${BUILD_TOOLS}/d8" \
    --lib "${ANDROID_JAR}" \
    --output "${TMP_BUILD}/dex" \
    "${TMP_BUILD}/classes/com/sipendosa/app"/*.class

# 5. Paketkan resources dan AndroidManifest.xml dengan AAPT
echo "==> Mengemas resources dan binary XML dengan AAPT..."
"${BUILD_TOOLS}/aapt" package -f \
    -M mobile/android/app/src/main/AndroidManifest.xml \
    -S mobile/android/app/src/main/res \
    -I "${ANDROID_JAR}" \
    -F "${TMP_BUILD}/unaligned.apk"

# 6. Masukkan DEX, Native Binaries multi-arsitektur, dan Assets ke dalam APK
echo "==> Menyisipkan classes.dex, assets/sipen, dan native libraries (Universal)..."
cd "${TMP_BUILD}"
cp "${TMP_BUILD}/dex/classes.dex" ./classes.dex
"${BUILD_TOOLS}/aapt" add unaligned.apk classes.dex

mkdir -p assets
cp "${ROOT_DIR}/dist/bin/sipen_android_arm64" assets/sipen
"${BUILD_TOOLS}/aapt" add unaligned.apk assets/sipen

# Tambahkan native libraries untuk semua arsitektur
mkdir -p lib/arm64-v8a
cp "${ROOT_DIR}/dist/bin/sipen_android_arm64" lib/arm64-v8a/libsipen.so
"${BUILD_TOOLS}/aapt" add unaligned.apk lib/arm64-v8a/libsipen.so

mkdir -p lib/armeabi-v7a
cp "${ROOT_DIR}/dist/bin/sipen_android_armv7" lib/armeabi-v7a/libsipen.so
"${BUILD_TOOLS}/aapt" add unaligned.apk lib/armeabi-v7a/libsipen.so

mkdir -p lib/x86_64
cp "${ROOT_DIR}/dist/bin/sipen_android_amd64" lib/x86_64/libsipen.so
"${BUILD_TOOLS}/aapt" add unaligned.apk lib/x86_64/libsipen.so

cd "${ROOT_DIR}"

# 7. ZipAlign APK (4-byte alignment untuk performa & validitas sistem Android)
echo "==> Menyelaraskan ZIP archive dengan zipalign..."
"${BUILD_TOOLS}/zipalign" -p -f 4 "${TMP_BUILD}/unaligned.apk" "${TMP_BUILD}/aligned.apk"

# 8. Pastikan Keystore signing permanen ada
KEYSTORE="${TOOLS_DIR}/sipendosa.keystore"
if [ ! -f "$KEYSTORE" ]; then
    echo "==> Men-generate signing keystore permanen..."
    "${JAVA_BIN}/keytool" -genkeypair \
        -keystore "$KEYSTORE" \
        -storepass "sipendosa2026" \
        -keypass "sipendosa2026" \
        -alias "sipendosa" \
        -keyalg RSA \
        -keysize 2048 \
        -validity 36500 \
        -dname "CN=SiPenDosa, OU=Mobile, O=SiPenDosa Project, C=ID"
fi

# 9. Tanda tangani APK dengan APKSIGNER (V1 + V2 + V3 Signature Scheme)
echo "==> Menandatangani APK dengan APKSIGNER (Scheme v1, v2, v3)..."
"${BUILD_TOOLS}/apksigner" sign \
    --ks "$KEYSTORE" \
    --ks-pass "pass:sipendosa2026" \
    --key-pass "pass:sipendosa2026" \
    --ks-key-alias "sipendosa" \
    --v1-signing-enabled true \
    --v2-signing-enabled true \
    --v3-signing-enabled true \
    --out "dist/SiPenDosa-Android.apk" \
    "${TMP_BUILD}/aligned.apk"

# 10. Verifikasi APK
echo "==> Memverifikasi validitas tanda tangan APK..."
"${BUILD_TOOLS}/apksigner" verify --verbose "dist/SiPenDosa-Android.apk"

echo ""
echo "========================================================================"
echo "✓ APK ANDROID UNIVERSAL (API 21+, ARM64+ARMv7+x86_64) BERHASIL DIBUAT:"
ls -lh dist/SiPenDosa-Android.apk
"${BUILD_TOOLS}/aapt" dump badging dist/SiPenDosa-Android.apk | head -n 12
echo "========================================================================"
