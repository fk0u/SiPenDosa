.PHONY: build run test clean install-service installer-windows build-linux build-windows build-darwin build-all package-deb package-linux package-macos package-all

APP_NAME = sipen
DIST_DIR = dist
BIN_DIR = $(DIST_DIR)/bin
GO = go
LDFLAGS = -s -w

# 1. Build Lokal (Native OS)
build:
	@echo "==> Membangun binary $(APP_NAME) untuk platform saat ini (CGO_ENABLED=0)..."
	CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS)" -o $(APP_NAME) ./cmd/sipen
	@echo "==> Berhasil dikompilasi: ./$(APP_NAME)"

run:
	$(GO) run ./cmd/sipen

test:
	@echo "==> Menjalankan seluruh pengujian unit..."
	$(GO) test -v ./internal/...

# 2. Cross-Compilation Lintas OS
build-linux:
	@echo "==> Mengompilasi binary Linux (amd64 & arm64)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)_linux_amd64 ./cmd/sipen
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)_linux_arm64 ./cmd/sipen
	@echo "==> Selesai: $(BIN_DIR)/$(APP_NAME)_linux_amd64 & arm64"

build-windows:
	@echo "==> Mengompilasi binary Windows (amd64)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)_windows_amd64.exe ./cmd/sipen
	@echo "==> Selesai: $(BIN_DIR)/$(APP_NAME)_windows_amd64.exe"

build-darwin:
	@echo "==> Mengompilasi binary macOS (arm64 Apple Silicon & amd64 Intel)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)_darwin_arm64 ./cmd/sipen
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)_darwin_amd64 ./cmd/sipen
	@echo "==> Selesai: $(BIN_DIR)/$(APP_NAME)_darwin_arm64 & amd64"

build-all: build-linux build-windows build-darwin
	@echo "==> Seluruh binary lintas OS berhasil dikompilasi di $(BIN_DIR)!"

# 3. Windows Standalone Setup Executable (Self-Contained)
installer-windows: build-windows
	@echo "==> Menyiapkan binary embedded untuk Windows Installer..."
	cp -f $(BIN_DIR)/$(APP_NAME)_windows_amd64.exe cmd/installer/sipen.exe
	@echo "==> Mengompilasi SiPenDosa-Setup.exe standalone installer..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/SiPenDosa-Setup.exe ./cmd/installer
	cp -f $(DIST_DIR)/SiPenDosa-Setup.exe ./SiPenDosa-Setup.exe
	@echo "==> Windows Installer siap: $(DIST_DIR)/SiPenDosa-Setup.exe (dan ./SiPenDosa-Setup.exe)"

# 4. Packaging Debian / Ubuntu (.deb)
package-deb: build-linux
	@chmod +x scripts/build-deb.sh
	@echo "==> Membangun paket Debian/Ubuntu (.deb) untuk amd64..."
	@bash scripts/build-deb.sh amd64
	@echo "==> Membangun paket Debian/Ubuntu (.deb) untuk arm64..."
	@bash scripts/build-deb.sh arm64

# 5. Packaging Universal Linux Tarball (Multi-Distro RHEL/CentOS/Ubuntu/Debian)
package-linux: build-linux
	@echo "==> Mengemas Universal Linux distribution tarball..."
	@mkdir -p $(DIST_DIR)/tmp_linux_amd64
	@cp -f $(BIN_DIR)/$(APP_NAME)_linux_amd64 $(DIST_DIR)/tmp_linux_amd64/$(APP_NAME)
	@cp -f install-linux.sh $(DIST_DIR)/tmp_linux_amd64/
	@cp -f sipen.service $(DIST_DIR)/tmp_linux_amd64/
	@cp -f .env.example $(DIST_DIR)/tmp_linux_amd64/
	@chmod +x $(DIST_DIR)/tmp_linux_amd64/install-linux.sh
	@tar -czf $(DIST_DIR)/sipendosa_linux_amd64.tar.gz -C $(DIST_DIR)/tmp_linux_amd64 .
	@rm -rf $(DIST_DIR)/tmp_linux_amd64
	@echo "==> Selesai: $(DIST_DIR)/sipendosa_linux_amd64.tar.gz"

# 6. Packaging macOS (App Bundle, PKG Installer, DMG Image, dan Tarball)
app-macos:
	@bash scripts/build-macos-app.sh

pkg: app-macos
	@bash scripts/build-pkg.sh

dmg: app-macos
	@bash scripts/build-dmg.sh

package-macos: build-darwin app-macos pkg dmg
	@echo "==> Mengemas macOS distribution package..."
	@mkdir -p $(DIST_DIR)/tmp_macos
	@cp -f $(BIN_DIR)/$(APP_NAME)_darwin_arm64 $(DIST_DIR)/tmp_macos/$(APP_NAME)
	@cp -f install-macos.sh $(DIST_DIR)/tmp_macos/
	@cp -f .env.example $(DIST_DIR)/tmp_macos/
	@chmod +x $(DIST_DIR)/tmp_macos/install-macos.sh
	@tar -czf $(DIST_DIR)/sipendosa_macos_universal.tar.gz -C $(DIST_DIR)/tmp_macos .
	@rm -rf $(DIST_DIR)/tmp_macos
	@echo "==> Selesai paket macOS (App, PKG, DMG, Tarball) di $(DIST_DIR)/"

# 7. Packaging Android (Universal APK, Google App Bundle AAB & Termux Script)
build-android:
	@echo "==> Mengompilasi binary Android untuk seluruh arsitektur (arm64, arm7, x86_64, x86)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)_android_arm64 ./cmd/$(APP_NAME)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)_android_arm7 ./cmd/$(APP_NAME)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)_android_x86_64 ./cmd/$(APP_NAME)
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME)_android_x86 ./cmd/$(APP_NAME)
	@echo "==> Selesai kompilasi biner Android multi-arch."

package-apk:
	@bash scripts/build-android.sh

package-aab:
	@bash scripts/build-android.sh

package-android: package-apk

# 8. Package All Distributions
package-all: build-all installer-windows package-deb package-linux package-macos package-android
	@echo ""
	@echo "╔════════════════════════════════════════════════════════════════════════╗"
	@echo "║   ✓ SELURUH PAKET DISTRIBUSI STANDALONE SIPENDOSA BERHASIL DIBUAT!     ║"
	@echo "╠════════════════════════════════════════════════════════════════════════╣"
	@echo "║  • Android Standalone APK   : $(DIST_DIR)/SiPenDosa-Android.apk"
	@echo "║  • Google App Bundle (AAB)  : $(DIST_DIR)/SiPenDosa-Android.aab"
	@echo "║  • Apple Disk Image (.dmg)  : $(DIST_DIR)/SiPenDosa-1.0.0.dmg"
	@echo "║  • Apple Installer (.pkg)   : $(DIST_DIR)/SiPenDosa-1.0.0-Installer.pkg"
	@echo "║  • Apple Native (.app)      : $(DIST_DIR)/SiPenDosa.app"
	@echo "║  • Windows Standalone Setup : $(DIST_DIR)/SiPenDosa-Setup.exe"
	@echo "║  • Debian/Ubuntu (.deb)     : $(DIST_DIR)/sipendosa_1.0.0_amd64.deb"
	@echo "║  • Linux Universal Tarball  : $(DIST_DIR)/sipendosa_linux_amd64.tar.gz"
	@echo "║  • macOS Standalone Package : $(DIST_DIR)/sipendosa_macos_universal.tar.gz"
	@echo "╚════════════════════════════════════════════════════════════════════════╝"

# 8. Utilitas Service Systemd Lokal
install-service:
	@echo "==> Memasang service systemd ke /etc/systemd/system/$(APP_NAME).service..."
	sudo cp $(APP_NAME).service /etc/systemd/system/$(APP_NAME).service
	sudo systemctl daemon-reload
	sudo systemctl enable $(APP_NAME)
	sudo systemctl restart $(APP_NAME)
	@echo "==> SiPenDosa service berhasil dipasang dan dijalankan!"
	sudo systemctl status $(APP_NAME)

clean:
	@echo "==> Membersihkan direktori build dan temporary..."
	rm -rf $(APP_NAME) $(APP_NAME).exe SiPenDosa-Setup.exe cmd/installer/sipen.exe $(DIST_DIR)
