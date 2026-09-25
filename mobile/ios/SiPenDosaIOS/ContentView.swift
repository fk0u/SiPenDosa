// ==============================================================================
// SiPenDosa — iOS Client & Local Server Companion (SwiftUI)
// Dilengkapi Background Execution, Live WebSocket Hub, dan Local Push Alert
// ==============================================================================

import SwiftUI
import WebKit

struct ContentView: View {
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var syncManager = BackgroundSyncManager.shared

    @State private var serverUrlString: String = {
        UserDefaults.standard.string(forKey: "sipendosa_server_url") ?? "http://localhost:8473"
    }()
    @State private var isShowingSettings: Bool = false
    @State private var webViewKey: UUID = UUID()

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 8/255, green: 9/255, blue: 13/255)
                    .ignoresSafeArea()

                VStack(spacing: 0) {
                    // Top Live Telemetry Bar (Latency, Engine State, Pending Queue)
                    telemetryHeader

                    // Webview Interface Container
                    WebViewContainer(urlString: serverUrlString)
                        .id(webViewKey)
                        .ignoresSafeArea(.keyboard, edges: .bottom)
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .principal) {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(syncManager.isEngineOnline ? Color(red: 16/255, green: 185/255, blue: 129/255) : Color(red: 225/255, green: 29/255, blue: 72/255))
                            .frame(width: 8, height: 8)
                        Text("⚡ SiPenDosa")
                            .font(.system(size: 15, weight: .bold))
                            .foregroundColor(.white)
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {
                        isShowingSettings = true
                    }) {
                        Image(systemName: "slider.horizontal.3")
                            .foregroundColor(Color(red: 245/255, green: 158/255, blue: 11/255))
                    }
                }
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: {
                        webViewKey = UUID()
                        syncManager.performBackgroundSync { _ in }
                    }) {
                        Image(systemName: "arrow.clockwise")
                            .foregroundColor(.white)
                    }
                }
            }
            .sheet(isPresented: $isShowingSettings) {
                ServerConfigView(currentUrl: $serverUrlString, isPresented: $isShowingSettings) {
                    UserDefaults.standard.set(serverUrlString, forKey: "sipendosa_server_url")
                    webViewKey = UUID()
                    syncManager.connectWebSocket(serverUrl: serverUrlString)
                    syncManager.performBackgroundSync { _ in }
                }
            }
        }
        .preferredColorScheme(.dark)
        .onAppear {
            syncManager.connectWebSocket(serverUrl: serverUrlString)
            syncManager.performBackgroundSync { _ in }
        }
        .onChange(of: scenePhase) { newPhase in
            switch newPhase {
            case .background:
                #if canImport(UIKit)
                syncManager.beginExtendedBackgroundTask()
                #endif
                #if canImport(BackgroundTasks)
                syncManager.scheduleNextBackgroundRefresh()
                #endif
            case .active:
                #if canImport(UIKit)
                syncManager.endExtendedBackgroundTask()
                #endif
                syncManager.connectWebSocket(serverUrl: serverUrlString)
                syncManager.performBackgroundSync { _ in }
            default:
                break
            }
        }
    }

    private var telemetryHeader: some View {
        HStack(spacing: 12) {
            // Engine status
            HStack(spacing: 4) {
                Image(systemName: syncManager.isEngineOnline ? "bolt.horizontal.fill" : "bolt.slash.fill")
                    .font(.system(size: 10))
                    .foregroundColor(syncManager.isEngineOnline ? Color(red: 16/255, green: 185/255, blue: 129/255) : Color(red: 225/255, green: 29/255, blue: 72/255))
                Text(syncManager.isEngineOnline ? "Engine Online" : "Connecting...")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(.white)
            }

            Spacer()

            // Latency ping
            if syncManager.lastPingMs >= 0 {
                HStack(spacing: 3) {
                    Image(systemName: "waveform")
                        .font(.system(size: 9))
                        .foregroundColor(.gray)
                    Text("\(syncManager.lastPingMs)ms")
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(.gray)
                }
            }

            // WhatsApp indicator
            HStack(spacing: 4) {
                Circle()
                    .fill(syncManager.waStatus == "connected" ? Color.green : Color.orange)
                    .frame(width: 6, height: 6)
                Text(syncManager.waStatus == "connected" ? "WA Aktif" : "WA Standby")
                    .font(.system(size: 10))
                    .foregroundColor(.gray)
            }

            // Pending Queue
            if syncManager.pendingCount > 0 {
                Text("\(syncManager.pendingCount) Antrian")
                    .font(.system(size: 9, weight: .bold))
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color(red: 225/255, green: 29/255, blue: 72/255).opacity(0.8))
                    .foregroundColor(.white)
                    .cornerRadius(8)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 6)
        .background(Color(red: 13/255, green: 16/255, blue: 23/255))
    }
}

struct WebViewContainer: UIViewRepresentable {
    let urlString: String

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.isOpaque = false
        webView.backgroundColor = UIColor(red: 8/255, green: 9/255, blue: 13/255, alpha: 1)
        webView.scrollView.backgroundColor = UIColor(red: 8/255, green: 9/255, blue: 13/255, alpha: 1)

        if let url = URL(string: urlString) {
            let request = URLRequest(url: url)
            webView.load(request)
        }
        return webView
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {
        // Updated if needed
    }
}

struct ServerConfigView: View {
    @Binding var currentUrl: String
    @Binding var isPresented: Bool
    var onSave: () -> Void

    @State private var editUrl: String = ""

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Alamat Host Server SiPenDosa")) {
                    TextField("http://localhost:8473", text: $editUrl)
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                        .keyboardType(.URL)
                    
                    Text("Gunakan 'http://localhost:8473' jika berjalan di perangkat yang sama, atau masukkan IP Wi-Fi (misal: 'http://192.168.1.5:8473') jika server di-host di laptop atau HP Android.")
                        .font(.caption)
                        .foregroundColor(.gray)
                }

                Section(header: Text("Fitur Latar Belakang & Realtime")) {
                    HStack {
                        Image(systemName: "bell.badge.fill")
                            .foregroundColor(Color(red: 245/255, green: 158/255, blue: 11/255))
                        Text("Notifikasi Pengiriman Realtime")
                        Spacer()
                        Text("Aktif")
                            .foregroundColor(.green)
                            .font(.caption)
                    }
                    HStack {
                        Image(systemName: "arrow.triangle.2.circlepath")
                            .foregroundColor(Color(red: 16/255, green: 185/255, blue: 129/255))
                        Text("BGTaskScheduler Refresh")
                        Spacer()
                        Text("Otomatis 15m")
                            .foregroundColor(.gray)
                            .font(.caption)
                    }
                    HStack {
                        Image(systemName: "antenna.radiowaves.left.and.right")
                            .foregroundColor(Color(red: 225/255, green: 29/255, blue: 72/255))
                        Text("WebSocket Telemetry Live")
                        Spacer()
                        Text("Port 8473")
                            .foregroundColor(.gray)
                            .font(.caption)
                    }
                }

                Section {
                    Button(action: {
                        if !editUrl.isEmpty {
                            currentUrl = editUrl
                            onSave()
                            isPresented = false
                        }
                    }) {
                        Text("Simpan & Hubungkan")
                            .frame(maxWidth: .infinity, alignment: .center)
                            .foregroundColor(Color(red: 225/255, green: 29/255, blue: 72/255))
                            .fontWeight(.bold)
                    }
                }
            }
            .navigationTitle("Pengaturan Server")
            .navigationBarItems(trailing: Button("Tutup") {
                isPresented = false
            })
            .onAppear {
                editUrl = currentUrl
            }
        }
        .preferredColorScheme(.dark)
    }
}
