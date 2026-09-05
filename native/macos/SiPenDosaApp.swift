// ==============================================================================
// SiPenDosa — Native macOS Menu Bar Controller & Companion App
// Ditulis dalam Swift & terhubung ke C++ High-Performance Bridge & Go Core
// ==============================================================================

import Cocoa
import Foundation
import UserNotifications

class AppDelegate: NSObject, NSApplicationDelegate, NSMenuDelegate {
    private var statusItem: NSStatusItem!
    private var statusMenu: NSMenu!
    private var timer: Timer?
    private var backendProcess: Process?
    private var backendPid: Int32 = 0
    private var isEngineOnline = false
    private var dataDir: URL!
    private var logFileURL: URL!
    private var envFileURL: URL!

    // Menu Items
    private var titleMenuItem: NSMenuItem!
    private var statusMenuItem: NSMenuItem!
    private var latencyMenuItem: NSMenuItem!
    private var openDashboardItem: NSMenuItem!
    private var scanQRItem: NSMenuItem!
    private var restartEngineItem: NSMenuItem!
    private var stopEngineItem: NSMenuItem!
    private var viewLogsItem: NSMenuItem!
    private var openFolderItem: NSMenuItem!

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Hindari muncul icon di dock jika diinginkan berjalan sebagai menu bar background agent
        // namun izinkan aktivasi window browser
        NSApp.setActivationPolicy(.accessory)

        setupPaths()
        setupNotificationPermissions()
        setupMenuBar()
        startBackendIfNeeded()

        // Timer pengecekan status via C++ Bridge setiap 2 detik
        timer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { [weak self] _ in
            self?.checkBackendStatus()
        }
        checkBackendStatus()
    }

    func applicationWillTerminate(_ notification: Notification) {
        timer?.invalidate()
        if let proc = backendProcess, proc.isRunning {
            proc.terminate()
        }
    }

    private func setupPaths() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        dataDir = home.appendingPathComponent(".local/share/sipendosa")
        try? FileManager.default.createDirectory(at: dataDir, withIntermediateDirectories: true)
        logFileURL = dataDir.appendingPathComponent("sipen.log")
        envFileURL = dataDir.appendingPathComponent(".env")

        // Buat .env default jika belum ada
        if !FileManager.default.fileExists(atPath: envFileURL.path) {
            let secret = UUID().uuidString.replacingOccurrences(of: "-", with: "") + UUID().uuidString.replacingOccurrences(of: "-", with: "")
            let defaultEnv = """
            PORT=8473
            HOST=0.0.0.0
            APP_ENV=production
            SESSION_SECRET=\(secret)

            DB_PATH=data/sipen.db
            WA_SESSION_PATH=session/whatsapp.db

            DEFAULT_TIMEZONE=Asia/Makassar
            SEND_WINDOW_START=08:00
            SEND_WINDOW_END=16:00

            RATE_LIMIT_MIN_SEC=5
            RATE_LIMIT_MAX_SEC=15
            MAX_RETRIES=3

            GLOBAL_DRY_RUN=false
            """
            try? defaultEnv.write(to: envFileURL, atomically: true, encoding: .utf8)
        }
    }

    private func setupNotificationPermissions() {
        if #available(macOS 10.14, *) {
            UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { _, _ in }
        }
    }

    private func showNativeNotification(title: String, body: String) {
        if #available(macOS 10.14, *) {
            let content = UNMutableNotificationContent()
            content.title = title
            content.body = body
            content.sound = UNNotificationSound.default
            let request = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
            UNUserNotificationCenter.current().add(request, withCompletionHandler: nil)
        }
    }

    private func setupMenuBar() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem.button {
            button.title = "⚡ SiPenDosa"
            button.toolTip = "SiPenDosa — Sistem Pengingat Dosen Saatnya"
        }

        statusMenu = NSMenu()
        statusMenu.delegate = self

        // 1. Header Identitas
        titleMenuItem = NSMenuItem(title: "⚡ SiPenDosa v1.0.0", action: nil, keyEquivalent: "")
        titleMenuItem.isEnabled = false
        statusMenu.addItem(titleMenuItem)

        // 2. Status Engine & Latency
        statusMenuItem = NSMenuItem(title: "● Status: Memeriksa koneksi...", action: nil, keyEquivalent: "")
        statusMenuItem.isEnabled = false
        statusMenu.addItem(statusMenuItem)

        latencyMenuItem = NSMenuItem(title: "⏱ Latency: -- | RAM: --", action: nil, keyEquivalent: "")
        latencyMenuItem.isEnabled = false
        statusMenu.addItem(latencyMenuItem)

        statusMenu.addItem(NSMenuItem.separator())

        // 3. Aksi Cepat Web Dashboard & WhatsApp
        openDashboardItem = NSMenuItem(title: "🌐 Buka Web Dashboard", action: #selector(openDashboard), keyEquivalent: "d")
        openDashboardItem.target = self
        statusMenu.addItem(openDashboardItem)

        scanQRItem = NSMenuItem(title: "📱 Tautkan WhatsApp (Scan QR)", action: #selector(openScanQR), keyEquivalent: "q")
        scanQRItem.target = self
        statusMenu.addItem(scanQRItem)

        statusMenu.addItem(NSMenuItem.separator())

        // 4. Manajemen Engine
        restartEngineItem = NSMenuItem(title: "🔄 Mulai Ulang Engine", action: #selector(restartEngine), keyEquivalent: "r")
        restartEngineItem.target = self
        statusMenu.addItem(restartEngineItem)

        stopEngineItem = NSMenuItem(title: "⏹ Hentikan Engine", action: #selector(stopEngine), keyEquivalent: "s")
        stopEngineItem.target = self
        statusMenu.addItem(stopEngineItem)

        statusMenu.addItem(NSMenuItem.separator())

        // 5. Utilitas & Log
        viewLogsItem = NSMenuItem(title: "📄 Buka File Log (Console)", action: #selector(viewLogs), keyEquivalent: "l")
        viewLogsItem.target = self
        statusMenu.addItem(viewLogsItem)

        openFolderItem = NSMenuItem(title: "📁 Buka Folder Konfigurasi (.env)", action: #selector(openDataFolder), keyEquivalent: "o")
        openFolderItem.target = self
        statusMenu.addItem(openFolderItem)

        statusMenu.addItem(NSMenuItem.separator())

        // 6. Tentang & Keluar
        let aboutItem = NSMenuItem(title: "ℹ️ Tentang SiPenDosa", action: #selector(showAbout), keyEquivalent: "a")
        aboutItem.target = self
        statusMenu.addItem(aboutItem)

        let quitItem = NSMenuItem(title: "Keluar (Quit)", action: #selector(quitApp), keyEquivalent: "w")
        quitItem.target = self
        statusMenu.addItem(quitItem)

        statusItem.menu = statusMenu
    }

    private func startBackendIfNeeded() {
        // Periksa apakah port 8473 sudah dijawab oleh backend yang ada
        let latency = sipen_check_socket_latency_us("127.0.0.1", 8473, 300)
        if latency >= 0 {
            // Sudah berjalan (misal via launchagent atau terminal)
            isEngineOnline = true
            updateMenuUI(online: true, latencyUs: latency, rssBytes: -1)
            return
        }

        // Cari lokasi binary sipen di dalam bundle: Contents/Resources/sipen
        guard let sipenBinURL = Bundle.main.url(forResource: "sipen", withExtension: nil) else {
            // Coba fallback di .local/share/sipendosa/sipen
            let fallbackBin = dataDir.appendingPathComponent("sipen")
            if FileManager.default.fileExists(atPath: fallbackBin.path) {
                launchBinary(at: fallbackBin)
            }
            return
        }

        launchBinary(at: sipenBinURL)
    }

    private func launchBinary(at binURL: URL) {
        let proc = Process()
        proc.executableURL = binURL
        proc.currentDirectoryURL = dataDir

        // Buat log file stream
        if !FileManager.default.fileExists(atPath: logFileURL.path) {
            FileManager.default.createFile(atPath: logFileURL.path, contents: nil)
        }

        if let logHandle = try? FileHandle(forWritingTo: logFileURL) {
            logHandle.seekToEndOfFile()
            proc.standardOutput = logHandle
            proc.standardError = logHandle
        }

        do {
            try proc.run()
            backendProcess = proc
            backendPid = proc.processIdentifier
        } catch {
            print("Gagal menjalankan backend binary: \(error)")
        }
    }

    private func checkBackendStatus() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self = self else { return }

            // Panggil C++ High-Performance Socket Latency Bridge
            let latencyUs = sipen_check_socket_latency_us("127.0.0.1", 8473, 500)
            let online = (latencyUs >= 0)

            var rss: Int64 = -1
            if self.backendPid > 0 {
                rss = sipen_get_process_memory_rss(self.backendPid)
            }

            DispatchQueue.main.async {
                let statusChanged = (online != self.isEngineOnline)
                self.isEngineOnline = online
                self.updateMenuUI(online: online, latencyUs: latencyUs, rssBytes: rss)

                if statusChanged && online {
                    self.showNativeNotification(
                        title: "⚡ SiPenDosa Aktif!",
                        body: "Asisten pengingat dosen siap beroperasi di http://localhost:8473"
                    )
                }
            }
        }
    }

    private func updateMenuUI(online: Bool, latencyUs: Int64, rssBytes: Int64) {
        if online {
            statusItem.button?.title = "⚡ SiPenDosa"
            statusMenuItem.title = "● Status: Aktif & Terhubung (Online)"
            let ms = Double(latencyUs) / 1000.0
            var ramStr = "--"
            if rssBytes > 0 {
                let mb = Double(rssBytes) / (1024.0 * 1024.0)
                ramStr = String(format: "%.1f MB", mb)
            }
            latencyMenuItem.title = String(format: "⏱ Latency: %.2f ms | RAM: %@", ms, ramStr)
            stopEngineItem.isEnabled = true
        } else {
            statusItem.button?.title = "⚡ SiPenDosa (Offline)"
            statusMenuItem.title = "○ Status: Offline / Memulai..."
            latencyMenuItem.title = "⏱ Latency: Tidak terjangkau | RAM: --"
            stopEngineItem.isEnabled = false
        }
    }

    // MARK: - Actions

    @objc private func openDashboard() {
        if let url = URL(string: "http://localhost:8473") {
            NSWorkspace.shared.open(url)
        }
    }

    @objc private func openScanQR() {
        if let url = URL(string: "http://localhost:8473#scan-qr") {
            NSWorkspace.shared.open(url)
        }
    }

    @objc private func restartEngine() {
        stopEngine()
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
            self?.startBackendIfNeeded()
            self?.checkBackendStatus()
        }
    }

    @objc private func stopEngine() {
        if let proc = backendProcess, proc.isRunning {
            proc.terminate()
            backendProcess = nil
            backendPid = 0
        } else if backendPid > 0 {
            sipen_terminate_pid(backendPid, false)
            backendPid = 0
        }
        checkBackendStatus()
    }

    @objc private func viewLogs() {
        if FileManager.default.fileExists(atPath: logFileURL.path) {
            NSWorkspace.shared.open(logFileURL)
        } else {
            let alert = NSAlert()
            alert.messageText = "Log Belum Tersedia"
            alert.informativeText = "File log di \(logFileURL.path) belum dibuat. Jalankan engine terlebih dahulu."
            alert.alertStyle = .informational
            alert.runModal()
        }
    }

    @objc private func openDataFolder() {
        NSWorkspace.shared.open(dataDir)
    }

    @objc private func showAbout() {
        let alert = NSAlert()
        alert.messageText = "SiPenDosa — Native macOS Edition"
        alert.informativeText = """
        Sistem Pengingat Dosen Saatnya (SiPenDosa)
        Versi 1.0.0 (Universal Apple Architecture)

        "Asisten yang rela 'berdosa' demi mengingatkan dosen agar mahasiswa tidak perlu merasa sungkan."

        ⚡ Native Engine: Go Core + C++17 Socket Bridge + Swift AppKit
        🎨 Persona: Scarlet Rose & Burnished Gold (Tanpa kompromi)
        """
        alert.alertStyle = .informational
        alert.addButton(withTitle: "Buka Dashboard")
        alert.addButton(withTitle: "Tutup")
        let res = alert.runModal()
        if res == .alertFirstButtonReturn {
            openDashboard()
        }
    }

    @objc private func quitApp() {
        stopEngine()
        NSApp.terminate(nil)
    }
}

// Entrypoint
let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
