// ==============================================================================
// SiPenDosa — iOS Background Sync & Realtime Dispatch Engine
// Mendukung BGTaskScheduler (BGAppRefreshTask), Extended Background Execution,
// dan Real-time WebSocket Hub Listener
// ==============================================================================

import Foundation
import UserNotifications
#if canImport(BackgroundTasks)
import BackgroundTasks
#endif
#if canImport(UIKit)
import UIKit
#endif

public final class BackgroundSyncManager: NSObject, ObservableObject {
    public static let shared = BackgroundSyncManager()

    public static let refreshTaskId = "com.sipendosa.ios.refresh"
    public static let syncTaskId = "com.sipendosa.ios.sync"

    @Published public var isEngineOnline: Bool = false
    @Published public var lastPingMs: Int64 = -1
    @Published public var waStatus: String = "disconnected"
    @Published public var pendingCount: Int = 0
    @Published public var lastSyncDate: Date? = nil

    private var webSocketTask: URLSessionWebSocketTask?
    private var urlSession: URLSession!
    #if canImport(UIKit)
    private var backgroundTaskIdentifier: UIBackgroundTaskIdentifier = .invalid
    #endif
    private var heartbeatTimer: Timer?

    private override init() {
        super.init()
        let config = URLSessionConfiguration.default
        config.waitsForConnectivity = true
        config.timeoutIntervalForRequest = 15.0
        self.urlSession = URLSession(configuration: config)
        requestNotificationPermission()
    }

    // MARK: - Notification Permissions
    public func requestNotificationPermission() {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { granted, error in
            if granted {
                NSLog("[SiPenDosa-iOS] Izin notifikasi lokal diberikan.")
            } else if let err = error {
                NSLog("[SiPenDosa-iOS] Gagal meminta izin notifikasi: %@", err.localizedDescription)
            }
        }
    }

    public func showNotification(title: String, body: String, category: String = "general") {
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default
        content.categoryIdentifier = category

        let trigger = UNTimeIntervalNotificationTrigger(timeInterval: 1, repeats: false)
        let request = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: trigger)
        UNUserNotificationCenter.current().add(request) { err in
            if let err = err {
                NSLog("[SiPenDosa-iOS] Error presenting notification: %@", err.localizedDescription)
            }
        }
    }

    // MARK: - Background Tasks Registration (iOS)
    #if canImport(BackgroundTasks)
    public func registerBackgroundTasks() {
        BGTaskScheduler.shared.register(forTaskWithIdentifier: Self.refreshTaskId, using: nil) { task in
            guard let appRefreshTask = task as? BGAppRefreshTask else { return }
            self.handleAppRefresh(task: appRefreshTask)
        }

        BGTaskScheduler.shared.register(forTaskWithIdentifier: Self.syncTaskId, using: nil) { task in
            guard let processingTask = task as? BGProcessingTask else { return }
            self.handleProcessingSync(task: processingTask)
        }
        NSLog("[SiPenDosa-iOS] BGTaskScheduler registered: %@, %@", Self.refreshTaskId, Self.syncTaskId)
    }

    public func scheduleNextBackgroundRefresh() {
        let request = BGAppRefreshTaskRequest(identifier: Self.refreshTaskId)
        // Jadwalkan paling cepat 15 menit ke depan (standar efisiensi baterai iOS)
        request.earliestBeginDate = Date(timeIntervalSinceNow: 15 * 60)
        do {
            try BGTaskScheduler.shared.submit(request)
            NSLog("[SiPenDosa-iOS] BGAppRefreshTask dijadwalkan untuk 15 menit ke depan.")
        } catch {
            NSLog("[SiPenDosa-iOS] Gagal menjadwalkan BGAppRefreshTask: %@", error.localizedDescription)
        }
    }

    private func handleAppRefresh(task: BGAppRefreshTask) {
        scheduleNextBackgroundRefresh()

        let queue = OperationQueue()
        queue.maxConcurrentOperationCount = 1

        task.expirationHandler = {
            queue.cancelAllOperations()
            task.setTaskCompleted(success: false)
        }

        let operation = BlockOperation {
            let semaphore = DispatchSemaphore(value: 0)
            self.performBackgroundSync { success in
                task.setTaskCompleted(success: success)
                semaphore.signal()
            }
            _ = semaphore.wait(timeout: .now() + 25.0)
        }

        queue.addOperation(operation)
    }

    private func handleProcessingSync(task: BGProcessingTask) {
        task.expirationHandler = {
            task.setTaskCompleted(success: false)
        }

        performBackgroundSync { success in
            task.setTaskCompleted(success: success)
        }
    }
    #endif

    // MARK: - Extended Background Execution on App Minimization
    #if canImport(UIKit)
    public func beginExtendedBackgroundTask() {
        endExtendedBackgroundTask()
        backgroundTaskIdentifier = UIApplication.shared.beginBackgroundTask(withName: "SiPenDosaExtendedSync") {
            NSLog("[SiPenDosa-iOS] Background execution time expired.")
            self.endExtendedBackgroundTask()
        }

        // Lakukan sinkronisasi cepat saat transisi ke background
        performBackgroundSync { _ in
            // Biarkan task hidup sesaat jika diperlukan, lalu tutup
            DispatchQueue.main.asyncAfter(deadline: .now() + 5.0) {
                self.endExtendedBackgroundTask()
            }
        }
    }

    public func endExtendedBackgroundTask() {
        if backgroundTaskIdentifier != .invalid {
            UIApplication.shared.endBackgroundTask(backgroundTaskIdentifier)
            backgroundTaskIdentifier = .invalid
        }
    }
    #endif

    // MARK: - Session Authentication
    // Server mewajibkan autentikasi pada /api/*. Sesi login diambil dari
    // cookie "sipen_session" yang di-set WebView setelah user login.
    private func sessionToken(for serverUrl: String) -> String? {
        guard let base = URL(string: serverUrl),
              let cookies = HTTPCookieStorage.shared.cookies(for: base) else {
            return nil
        }
        return cookies.first(where: { $0.name == "sipen_session" })?.value
    }

    private func authenticatedRequest(_ url: URL, serverUrl: String) -> URLRequest {
        var req = URLRequest(url: url)
        req.timeoutInterval = 8.0
        if let token = sessionToken(for: serverUrl) {
            req.setValue("sipen_session=\(token)", forHTTPHeaderField: "Cookie")
        }
        return req
    }

    // MARK: - Core Background Sync Operation
    public func performBackgroundSync(completion: @escaping (Bool) -> Void) {
        let serverUrl = UserDefaults.standard.string(forKey: "sipendosa_server_url") ?? "http://localhost:8473"
        guard let url = URL(string: "\(serverUrl)/api/stats") else {
            completion(false)
            return
        }

        let req = authenticatedRequest(url, serverUrl: serverUrl)

        let startTime = DispatchTime.now()
        let task = urlSession.dataTask(with: req) { [weak self] data, response, error in
            guard let self = self else {
                completion(false)
                return
            }

            if let error = error {
                NSLog("[SiPenDosa-iOS] Background sync failed: %@", error.localizedDescription)
                DispatchQueue.main.async {
                    self.isEngineOnline = false
                }
                completion(false)
                return
            }

            let endTime = DispatchTime.now()
            let latencyMs = Int64((endTime.uptimeNanoseconds - startTime.uptimeNanoseconds) / 1_000_000)

            DispatchQueue.main.async {
                self.isEngineOnline = true
                self.lastPingMs = latencyMs
                self.lastSyncDate = Date()
            }

            // Periksa jika ada pesan tertunda di antrian
            self.checkPendingQueue(serverUrl: serverUrl) { hasPending in
                completion(true)
            }
        }
        task.resume()
    }

    private func checkPendingQueue(serverUrl: String, completion: @escaping (Bool) -> Void) {
        guard let url = URL(string: "\(serverUrl)/api/queue") else {
            completion(false)
            return
        }

        let req = authenticatedRequest(url, serverUrl: serverUrl)

        let task = urlSession.dataTask(with: req) { data, _, _ in
            guard let data = data else {
                completion(false)
                return
            }

            if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
               let items = json["data"] as? [[String: Any]] {
                let pendingCount = items.filter { ($0["status"] as? String) == "pending" }.count
                DispatchQueue.main.async {
                    self.pendingCount = pendingCount
                }
            }
            completion(true)
        }
        task.resume()
    }

    // MARK: - Live WebSocket Hub Listener (Realtime Updates)
    public func connectWebSocket(serverUrl: String) {
        disconnectWebSocket()

        guard let httpUrl = URL(string: serverUrl) else { return }
        let wsScheme = (httpUrl.scheme == "https") ? "wss" : "ws"
        var components = URLComponents()
        components.scheme = wsScheme
        components.host = httpUrl.host ?? "localhost"
        components.port = httpUrl.port ?? 8473
        components.path = "/api/ws"
        // iOS URLSessionWebSocketTask tidak bisa set header custom —
        // token sesi dikirim sebagai query param.
        if let token = sessionToken(for: serverUrl) {
            components.queryItems = [URLQueryItem(name: "token", value: token)]
        }
        guard let wsUrl = components.url else {
            return
        }

        webSocketTask = urlSession.webSocketTask(with: wsUrl)
        webSocketTask?.resume()
        listenWebSocketMessages()

        startHeartbeat()
        NSLog("[SiPenDosa-iOS] Terhubung ke WebSocket Realtime: %@", wsUrl.absoluteString)
    }

    public func disconnectWebSocket() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = nil
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
    }

    private func startHeartbeat() {
        heartbeatTimer?.invalidate()
        heartbeatTimer = Timer.scheduledTimer(withTimeInterval: 20.0, repeats: true) { [weak self] _ in
            self?.webSocketTask?.sendPing { error in
                if let error = error {
                    NSLog("[SiPenDosa-iOS] WebSocket ping failed: %@", error.localizedDescription)
                }
            }
        }
    }

    private func listenWebSocketMessages() {
        webSocketTask?.receive { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .success(let message):
                switch message {
                case .string(let text):
                    self.handleIncomingWebSocketMessage(text)
                case .data(let data):
                    if let text = String(data: data, encoding: .utf8) {
                        self.handleIncomingWebSocketMessage(text)
                    }
                @unknown default:
                    break
                }
                self.listenWebSocketMessages()
            case .failure(let error):
                NSLog("[SiPenDosa-iOS] WebSocket receive error: %@", error.localizedDescription)
                DispatchQueue.main.asyncAfter(deadline: .now() + 5.0) {
                    let savedUrl = UserDefaults.standard.string(forKey: "sipendosa_server_url") ?? "http://localhost:8473"
                    self.connectWebSocket(serverUrl: savedUrl)
                }
            }
        }
    }

    private func handleIncomingWebSocketMessage(_ raw: String) {
        guard let data = raw.data(using: .utf8),
              let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let event = json["event"] as? String else {
            return
        }

        DispatchQueue.main.async {
            switch event {
            case "wa_status":
                if let payload = json["payload"] as? [String: Any],
                   let state = payload["state"] as? String {
                    self.waStatus = state
                }
            case "queue_update":
                if let payload = json["payload"] as? [String: Any],
                   let action = payload["action"] as? String {
                    if action == "sent" {
                        self.showNotification(
                            title: "⚡ SiPenDosa Terkirim!",
                            body: "Pesan pengingat dosen berhasil dikirim ke WhatsApp secara realtime.",
                            category: "sent"
                        )
                    }
                }
            default:
                break
            }
        }
    }
}
