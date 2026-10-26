// ==============================================================================
// SiPenDosa — Native macOS Menu Bar Controller & Companion App
// Ditulis dalam Swift & terhubung ke C++ High-Performance Bridge & Go Core
// ==============================================================================

import Cocoa
import Foundation
import UserNotifications
import WebKit

class AppDelegate: NSObject, NSApplicationDelegate, NSMenuDelegate {
    private var statusItem: NSStatusItem!
    private var statusMenu: NSMenu!
    private var timer: Timer?
    private var backendProcess: Process?
    private var backendPid: Int32 = 0
    private var isEngineOnline = false
    private var desktopWindow: NSWindow?
    private var launchedBackendOurselves = false
    private var dataDir: URL!
    private var logFileURL: URL!
    private var envFileURL: URL!
    private var configuredPort: Int = 8473

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

        // Mode desktop: bila engine dimulai oleh aplikasi ini, tampilkan
        // jendela desktop begitu engine siap melayani.
        if launchedBackendOurselves {
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { [weak self] in
                self?.openDesktopApp()
            }
        }
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

        loadPortFromEnv()
    }

    private func loadPortFromEnv() {
        guard let content = try? String(contentsOf: envFileURL, encoding: .utf8) else { return }
        for line in content.components(separatedBy: .newlines) {
            let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.hasPrefix("PORT=") {
                let val = trimmed.replacingOccurrences(of: "PORT=", with: "").trimmingCharacters(in: .whitespacesAndNewlines)
                if let p = Int(val), p > 0 && p < 65536 {
                    self.configuredPort = p
                }
            }
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
        titleMenuItem = NSMenuItem(title: "⚡ SiPenDosa v1.4.1 (Universal)", action: nil, keyEquivalent: "")
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

        // 3. Aksi Cepat: Aplikasi Desktop, Web Dashboard & WhatsApp
        let desktopAppItem = NSMenuItem(title: "🖥 Buka Aplikasi Desktop", action: #selector(openDesktopApp), keyEquivalent: "a")
        desktopAppItem.target = self
        statusMenu.addItem(desktopAppItem)

        openDashboardItem = NSMenuItem(title: "🌐 Buka Web Dashboard (Browser)", action: #selector(openDashboard), keyEquivalent: "d")
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

    private let expectedVersion = "1.4.1"

    private func checkRunningBackendVersion() -> String? {
        guard let url = URL(string: "http://127.0.0.1:\(configuredPort)/api/system/version") else { return nil }
        var request = URLRequest(url: url)
        request.timeoutInterval = 0.8
        var verStr: String?
        let sem = DispatchSemaphore(value: 0)
        let task = URLSession.shared.dataTask(with: request) { data, _, _ in
            defer { sem.signal() }
            guard let data = data,
                  let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let v = json["version"] as? String else {
                return
            }
            verStr = v
        }
        task.resume()
        _ = sem.wait(timeout: .now() + 1.0)
        return verStr
    }

    private func syncBundledBinaryToDisk() {
        guard let bundledURL = Bundle.main.url(forResource: "sipen", withExtension: nil) else { return }
        
        let target1 = dataDir.appendingPathComponent("sipen")
        try? FileManager.default.removeItem(at: target1)
        try? FileManager.default.copyItem(at: bundledURL, to: target1)
        try? FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: target1.path)

        let home = FileManager.default.homeDirectoryForCurrentUser
        let appSupportSipen = home.appendingPathComponent("Applications/SiPenDosa/sipen")
        if FileManager.default.fileExists(atPath: appSupportSipen.deletingLastPathComponent().path) {
            try? FileManager.default.removeItem(at: appSupportSipen)
            try? FileManager.default.copyItem(at: bundledURL, to: appSupportSipen)
            try? FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: appSupportSipen.path)
        }
    }

    private func killOldBackendProcesses() {
        if let proc = backendProcess, proc.isRunning {
            proc.terminate()
            backendProcess = nil
            backendPid = 0
        }

        // 1. Matikan proses yang mendengarkan di port yang dikonfigurasi (mis. 8473)
        let lsofTask = Process()
        lsofTask.launchPath = "/bin/sh"
        lsofTask.arguments = ["-c", "/usr/sbin/lsof -ti :\(configuredPort) | xargs kill -9 2>/dev/null || true"]
        try? lsofTask.run()
        lsofTask.waitUntilExit()

        // 2. Matikan proses dengan nama biner sipen
        let killallTask = Process()
        killallTask.launchPath = "/bin/sh"
        killallTask.arguments = ["-c", "killall -9 sipen 2>/dev/null || true"]
        try? killallTask.run()
        killallTask.waitUntilExit()
    }

    private func startBackendIfNeeded() {
        // Sinkronisasi biner terbaru dari App Bundle ke lokasi disk
        syncBundledBinaryToDisk()

        // Periksa apakah port sudah dijawab oleh backend yang ada
        let latency = sipen_check_socket_latency_us("127.0.0.1", Int32(configuredPort), 300)
        if latency >= 0 {
            // Ada proses yang merespons di port ini. Periksa versinya!
            if let runningVer = checkRunningBackendVersion() {
                if runningVer == expectedVersion {
                    // Backend sudah berjalan dengan versi yang tepat
                    isEngineOnline = true
                    updateMenuUI(online: true, latencyUs: latency, rssBytes: -1)
                    return
                }
                print("[SiPenDosa] Engine versi lama terdeteksi (v\(runningVer) != v\(expectedVersion)). Mematikan engine lama...")
                showNativeNotification(
                    title: "⚡ Memperbarui SiPenDosa ke v\(expectedVersion)",
                    body: "Engine versi lama (v\(runningVer)) dimatikan dan digantikan dengan engine terbaru."
                )
            } else {
                print("[SiPenDosa] Port \(configuredPort) terbuka tapi tidak merespons versi. Mematikan proses lama...")
            }

            // Hentikan backend lama yang out-of-date atau zombie
            killOldBackendProcesses()
            Thread.sleep(forTimeInterval: 0.6)
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
        proc.arguments = ["web"]
        var env = ProcessInfo.processInfo.environment
        env["SIPEN_UI"] = "web"
        proc.environment = env
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
            launchedBackendOurselves = true
        } catch {
            print("Gagal menjalankan backend binary: \(error)")
        }
    }

    // MARK: - Jendela Aplikasi Desktop (WKWebView native)
    @objc private func openDesktopApp() {
        if let win = desktopWindow {
            win.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }

        let win = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1280, height: 820),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        win.title = "SiPenDosa — Sistem Pengingat Dosen Saatnya"
        win.titlebarAppearsTransparent = false
        win.center()

        let config = WKWebViewConfiguration()
        config.preferences.setValue(true, forKey: "developerExtrasEnabled")
        let webView = WKWebView(frame: .zero, configuration: config)
        if let url = URL(string: "http://localhost:\(configuredPort)") {
            webView.load(URLRequest(url: url))
        }
        win.contentView = webView
        desktopWindow = win
        win.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func checkBackendStatus() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self = self else { return }

            // Panggil C++ High-Performance Socket Latency Bridge
            let latencyUs = sipen_check_socket_latency_us("127.0.0.1", Int32(self.configuredPort), 500)
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
                        body: "Asisten pengingat dosen siap beroperasi di http://localhost:\(self.configuredPort)"
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
        if let url = URL(string: "http://localhost:\(configuredPort)") {
            NSWorkspace.shared.open(url)
        }
    }

    @objc private func openScanQR() {
        if let url = URL(string: "http://localhost:\(configuredPort)#scan-qr") {
            NSWorkspace.shared.open(url)
        }
    }

    @objc private func restartEngine() {
        stopEngine()
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.8) { [weak self] in
            self?.startBackendIfNeeded()
            self?.checkBackendStatus()
        }
    }

    @objc private func stopEngine() {
        killOldBackendProcesses()
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
        🎨 Persona: Material Design 3 — Rose Seed Tonal Palette
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
