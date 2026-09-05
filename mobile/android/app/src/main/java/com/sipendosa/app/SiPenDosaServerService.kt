package com.sipendosa.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.net.wifi.WifiManager
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import android.text.format.Formatter
import androidx.core.app.NotificationCompat

/**
 * SiPenDosaServerService — Android Foreground Service dengan PowerManager Wakelock
 * Menjaga proses server tetap hidup 24/7 di smartphone Android tanpa terputus
 */
class SiPenDosaServerService : Service() {

    private var wakeLock: PowerManager.WakeLock? = null
    private var serverProcess: Process? = null
    private val channelId = "sipendosa_server_channel"
    private val notificationId = 8473

    override fun onCreate() {
        super.onCreate()
        acquireWakeLock()
        createNotificationChannel()
        startForeground(notificationId, buildNotification("Memulai server SiPenDosa..."))
        startNativeServer()
    }

    private fun acquireWakeLock() {
        val powerManager = getSystemService(Context.POWER_SERVICE) as PowerManager
        wakeLock = powerManager.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "SiPenDosa::ServerWakeLock"
        ).apply {
            acquire(24 * 60 * 60 * 1000L) // 24 jam wakelock
        }
    }

    private fun getLocalWifiIp(): String {
        return try {
            val wifiManager = applicationContext.getSystemService(Context.WIFI_SERVICE) as WifiManager
            Formatter.formatIpAddress(wifiManager.connectionInfo.ipAddress)
        } catch (e: Exception) {
            "127.0.0.1"
        }
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                channelId,
                "SiPenDosa Server Daemon",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Layanan pengingat dosen WhatsApp yang berjalan di latar belakang"
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildNotification(statusText: String): Notification {
        val intent = Intent(this, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            this, 0, intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val ip = getLocalWifiIp()

        return NotificationCompat.Builder(this, channelId)
            .setContentTitle("⚡ SiPenDosa Server Aktif")
            .setContentText("Akses LAN: http://$ip:8473 • $statusText")
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }

    private fun startNativeServer() {
        Thread {
            try {
                val binPath = applicationInfo.nativeLibraryDir + "/libsipen.so"
                val appDir = filesDir.absolutePath

                val pb = ProcessBuilder(binPath)
                pb.directory(filesDir)
                pb.environment()["PORT"] = "8473"
                pb.environment()["HOST"] = "0.0.0.0"
                pb.environment()["APP_ENV"] = "production"

                serverProcess = pb.start()
                updateNotification("Melayani di port 8473 (Online)")
                serverProcess?.waitFor()
            } catch (e: Exception) {
                updateNotification("Status: Menunggu eksekusi server...")
            }
        }.start()
    }

    private fun updateNotification(text: String) {
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(notificationId, buildNotification(text))
    }

    override fun onDestroy() {
        super.onDestroy()
        serverProcess?.destroy()
        wakeLock?.let {
            if (it.isHeld) it.release()
        }
    }

    override fun onBind(intent: Intent?): IBinder? = null
}
