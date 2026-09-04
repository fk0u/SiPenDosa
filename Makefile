.PHONY: build run test clean install-service installer

APP_NAME = sipen
BUILD_DIR = bin
GO = go

build:
	@echo "==> Membangun binary $(APP_NAME) (CGO_ENABLED=0)..."
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(APP_NAME).exe ./cmd/sipen
	@echo "==> Berhasil dikompilasi: $(APP_NAME).exe"

installer: build
	@echo "==> Menyiapkan binary untuk installer..."
	cp $(APP_NAME).exe cmd/installer/sipen.exe
	@echo "==> Mengompilasi SiPenDosa-Setup.exe standalone installer..."
	$(GO) build -ldflags="-s -w" -o SiPenDosa-Setup.exe ./cmd/installer
	@echo "==> Installer berhasil dibangun: SiPenDosa-Setup.exe"

run:
	$(GO) run ./cmd/sipen

test:
	@echo "==> Menjalankan pengujian unit..."
	$(GO) test -v ./internal/...

clean:
	@echo "==> Membersihkan file temporary dan binary..."
	rm -f $(APP_NAME) $(APP_NAME).exe SiPenDosa-Setup.exe cmd/installer/sipen.exe

install-service:
	@echo "==> Memasang service systemd ke /etc/systemd/system/$(APP_NAME).service..."
	sudo cp $(APP_NAME).service /etc/systemd/system/$(APP_NAME).service
	sudo systemctl daemon-reload
	sudo systemctl enable $(APP_NAME)
	sudo systemctl restart $(APP_NAME)
	@echo "==> SiPenDosa service berhasil dipasang dan dijalankan!"
	sudo systemctl status $(APP_NAME)
