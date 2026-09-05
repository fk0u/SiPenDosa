package com.sipendosa.app

import android.annotation.SuppressLint
import android.content.Intent
import android.os.Bundle
import android.webkit.WebChromeClient
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity

/**
 * MainActivity — Native WebView Container untuk SiPenDosa Dashboard
 * Memuat dashboard lokal di http://127.0.0.1:8473 secara responsif
 */
class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private val localServerUrl = "http://127.0.0.1:8473"

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // 1. Jalankan Foreground Server Service
        val serviceIntent = Intent(this, SiPenDosaServerService::class.java)
        startForegroundService(serviceIntent)

        // 2. Siapkan WebView
        webView = WebView(this)
        setContentView(webView)

        val settings: WebSettings = webView.settings
        settings.javaScriptEnabled = true
        settings.domStorageEnabled = true
        settings.databaseEnabled = true
        settings.useWideViewPort = true
        settings.loadWithOverviewMode = true
        settings.setSupportZoom(false)

        webView.webViewClient = object : WebViewClient() {
            override fun onReceivedError(
                view: WebView?,
                errorCode: Int,
                description: String?,
                failingUrl: String?
            ) {
                Toast.makeText(
                    this@MainActivity,
                    "Menghubungkan ke server lokal SiPenDosa...",
                    Toast.LENGTH_SHORT
                ).show()
                // Coba muat ulang setelah 2 detik
                view?.postDelayed({ view.loadUrl(localServerUrl) }, 2000)
            }
        }

        webView.webChromeClient = WebChromeClient()
        webView.loadUrl(localServerUrl)
    }

    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            super.onBackPressed()
        }
    }
}
