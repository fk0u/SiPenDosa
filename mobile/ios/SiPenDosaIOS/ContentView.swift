// ==============================================================================
// SiPenDosa — iOS Client & Local Server Companion (SwiftUI)
// ==============================================================================

import SwiftUI
import WebKit

struct ContentView: View {
    @State private var serverUrlString: String = "http://localhost:8473"
    @State private var inputHost: String = ""
    @State private var isShowingSettings: Bool = false
    @State private var webViewKey: UUID = UUID()

    var body: some View {
        NavigationView {
            ZStack {
                Color(red: 9/255, green: 10/255, blue: 15/255)
                    .ignoresSafeArea()

                WebViewContainer(urlString: serverUrlString)
                    .id(webViewKey)
                    .ignoresSafeArea(.keyboard)
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .principal) {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(Color(red: 225/255, green: 29/255, blue: 72/255))
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
                        Image(systemName: "network")
                            .foregroundColor(Color(red: 245/255, green: 158/255, blue: 11/255))
                    }
                }
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: {
                        webViewKey = UUID()
                    }) {
                        Image(systemName: "arrow.clockwise")
                            .foregroundColor(.white)
                    }
                }
            }
            .sheet(isPresented: $isShowingSettings) {
                ServerConfigView(currentUrl: $serverUrlString, isPresented: $isShowingSettings) {
                    webViewKey = UUID()
                }
            }
        }
        .preferredColorScheme(.dark)
    }
}

struct WebViewContainer: UIViewRepresentable {
    let urlString: String

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.isOpaque = false
        webView.backgroundColor = UIColor(red: 9/255, green: 10/255, blue: 15/255, alpha: 1)
        webView.scrollView.backgroundColor = UIColor(red: 9/255, green: 10/255, blue: 15/255, alpha: 1)

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
