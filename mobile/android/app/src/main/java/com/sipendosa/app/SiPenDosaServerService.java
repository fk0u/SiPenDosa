package com.sipendosa.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.net.wifi.WifiManager;
import android.os.Build;
import android.os.IBinder;
import android.os.PowerManager;
import android.util.Log;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.ArrayList;
import java.util.List;

/**
 * SiPenDosaServerService
 * Layanan latar belakang 24/7 untuk menjalankan core engine Go & whatsmeow
 * Dilengkapi dengan WakeLock, WifiLock (anti-deep-sleep), dan Auto-Respawn Watchdog
 * agar pengiriman pesan jadwal perkuliahan selalu tepat waktu secara real-time.
 */
public class SiPenDosaServerService extends Service implements Runnable {
    private static final String TAG = "SiPenDosaService";
    private static final String CHANNEL_ID = "sipendosa_service_channel";
    private static final int NOTIFICATION_ID = 8473;

    private PowerManager.WakeLock wakeLock;
    private WifiManager.WifiLock wifiLock;
    private Process serverProcess;
    private volatile boolean isRunning = false;
    private Thread serverThread;
    private static final List<String> logBuffer = new ArrayList<>();

    @Override
    public void onCreate() {
        super.onCreate();
        createNotificationChannel();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        startForegroundServiceNotification();
        acquireLocks();

        if (!isRunning || serverThread == null || !serverThread.isAlive()) {
            isRunning = true;
            serverThread = new Thread(this, "SiPenDosa-Engine-Thread");
            serverThread.start();
        }
        return START_STICKY;
    }

    private void acquireLocks() {
        // 1. Acquire Partial WakeLock agar CPU tidak ditidurkan oleh Doze Mode
        try {
            if (wakeLock == null) {
                PowerManager powerManager = (PowerManager) getSystemService(Context.POWER_SERVICE);
                if (powerManager != null) {
                    wakeLock = powerManager.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "SiPenDosa:ServerWakeLock");
                    wakeLock.setReferenceCounted(false);
                }
            }
            if (wakeLock != null && !wakeLock.isHeld()) {
                wakeLock.acquire();
                Log.i(TAG, "WakeLock berhasil diaktifkan.");
            }
        } catch (Exception e) {
            Log.e(TAG, "Gagal mengaktifkan WakeLock: " + e.getMessage(), e);
        }

        // 2. Acquire High-Performance WifiLock agar koneksi WhatsApp WebSocket tidak terputus saat layar mati
        try {
            if (wifiLock == null) {
                WifiManager wifiManager = (WifiManager) getApplicationContext().getSystemService(Context.WIFI_SERVICE);
                if (wifiManager != null) {
                    int lockMode = WifiManager.WIFI_MODE_FULL;
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB_MR1) {
                        lockMode = WifiManager.WIFI_MODE_FULL_HIGH_PERF;
                    }
                    wifiLock = wifiManager.createWifiLock(lockMode, "SiPenDosa:ServerWifiLock");
                    wifiLock.setReferenceCounted(false);
                }
            }
            if (wifiLock != null && !wifiLock.isHeld()) {
                wifiLock.acquire();
                Log.i(TAG, "WifiLock (High-Perf) berhasil diaktifkan.");
            }
        } catch (Exception e) {
            Log.e(TAG, "Gagal mengaktifkan WifiLock: " + e.getMessage(), e);
        }
    }

    private void releaseLocks() {
        try {
            if (wakeLock != null && wakeLock.isHeld()) {
                wakeLock.release();
                Log.i(TAG, "WakeLock dilepaskan.");
            }
        } catch (Exception ignored) {}

        try {
            if (wifiLock != null && wifiLock.isHeld()) {
                wifiLock.release();
                Log.i(TAG, "WifiLock dilepaskan.");
            }
        } catch (Exception ignored) {}
    }

    private void createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            NotificationChannel channel = new NotificationChannel(
                    CHANNEL_ID,
                    "SiPenDosa Background Server",
                    NotificationManager.IMPORTANCE_LOW
            );
            channel.setDescription("Menjaga server pengingat SiPenDosa tetap aktif di latar belakang 24/7");
            channel.setShowBadge(false);
            NotificationManager manager = (NotificationManager) getSystemService(Context.NOTIFICATION_SERVICE);
            if (manager != null) {
                manager.createNotificationChannel(channel);
            }
        }
    }

    private void startForegroundServiceNotification() {
        Intent notificationIntent = new Intent(this, MainActivity.class);
        PendingIntent pendingIntent = PendingIntent.getActivity(
                this, 0, notificationIntent,
                Build.VERSION.SDK_INT >= Build.VERSION_CODES.M ? PendingIntent.FLAG_IMMUTABLE : 0
        );

        Notification.Builder builder;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            builder = new Notification.Builder(this, CHANNEL_ID);
        } else {
            builder = new Notification.Builder(this);
        }

        Notification notification = builder
                .setContentTitle("SiPenDosa Server Active")
                .setContentText("Port 8473 • Engine Online • 24/7 Otomasi Realtime")
                .setSmallIcon(android.R.drawable.stat_notify_sync)
                .setContentIntent(pendingIntent)
                .setOngoing(true)
                .build();

        startForeground(NOTIFICATION_ID, notification);
    }

    @Override
    public void run() {
        Log.i(TAG, "Server daemon runner dimulai dengan fitur auto-respawn watchdog.");

        while (isRunning) {
            try {
                runServerProcessOnce();
            } catch (Exception e) {
                Log.e(TAG, "Exception saat menjalankan server process: " + e.getMessage(), e);
            }

            // Jika masih running tetapi server process berhenti (crash/killed),
            // lakukan auto-respawn setelah jeda singkat (watchdog).
            if (isRunning) {
                Log.w(TAG, "Server Go berhenti tidak terduga. Respawning dalam 3 detik...");
                try {
                    Thread.sleep(3000);
                } catch (InterruptedException e) {
                    Log.i(TAG, "Thread runner interrupted, keluar dari loop.");
                    break;
                }
            }
        }
        Log.i(TAG, "Server daemon runner selesai.");
    }

    private void runServerProcessOnce() {
        try {
            File filesDir = getFilesDir();
            File binFile = null;

            // 1. Cek apakah server sudah aktif di port 8473 (misal dari sesi sebelumnya)
            try {
                URL healthUrl = new URL("http://127.0.0.1:8473/healthz");
                HttpURLConnection testConn = (HttpURLConnection) healthUrl.openConnection();
                testConn.setConnectTimeout(600);
                testConn.setReadTimeout(600);
                if (testConn.getResponseCode() == 200) {
                    Log.i(TAG, "Server SiPenDosa sudah aktif di port 8473, memantau daemon...");
                    testConn.disconnect();
                    while (isRunning) {
                        Thread.sleep(4000);
                        try {
                            HttpURLConnection ping = (HttpURLConnection) new URL("http://127.0.0.1:8473/healthz").openConnection();
                            ping.setConnectTimeout(1000);
                            ping.setReadTimeout(1000);
                            int res = ping.getResponseCode();
                            ping.disconnect();
                            if (res < 200 || res >= 500) break;
                        } catch (Exception e) {
                            break;
                        }
                    }
                    if (!isRunning) return;
                }
            } catch (Exception ignored) {}

            // 2. Prioritaskan eksekusi dari nativeLibraryDir (resmi diizinkan oleh SELinux Android)
            File nativeDir = new File(getApplicationInfo().nativeLibraryDir);
            File nativeBin = new File(nativeDir, "libsipen.so");
            if (nativeBin.exists()) {
                nativeBin.setExecutable(true, false);
                binFile = nativeBin;
                Log.i(TAG, "Menggunakan native library binary: " + binFile.getAbsolutePath());
            }

            // 3. Fallback jika nativeLibraryDir belum terisi: ekstrak langsung dari APK archive
            if (binFile == null || !binFile.exists()) {
                binFile = extractBinaryFromApk(filesDir);
            }

            if (binFile == null || !binFile.exists()) {
                Log.e(TAG, "FATAL: Binary engine SiPenDosa (libsipen.so) tidak dapat ditemukan!");
                return;
            }

            File dataDir = new File(filesDir, "data");
            File sessionDir = new File(filesDir, "session");
            dataDir.mkdirs();
            sessionDir.mkdirs();

            // PENTING: Wajib menyertakan argumen "web" dan SIPEN_UI=web agar server HTTP port 8473 aktif!
            ProcessBuilder pb = new ProcessBuilder(binFile.getAbsolutePath(), "web");
            pb.directory(filesDir);
            pb.environment().put("SIPEN_UI", "web");
            pb.environment().put("PORT", "8473");
            pb.environment().put("HOST", "0.0.0.0");
            pb.environment().put("APP_ENV", "production");
            pb.environment().put("DB_PATH", new File(dataDir, "sipen.db").getAbsolutePath());
            pb.environment().put("WA_SESSION_PATH", new File(sessionDir, "whatsapp.db").getAbsolutePath());
            pb.environment().put("SESSION_SECRET", "sipendosa-android-service-session-key-32-chars");
            pb.environment().put("SSL_CERT_DIR", "/system/etc/security/cacerts:/apex/com.android.conscrypt/cacerts");

            pb.redirectErrorStream(true);
            serverProcess = pb.start();
            Log.i(TAG, "Server Go process berhasil dijalankan: " + binFile.getAbsolutePath());

            BufferedReader reader = new BufferedReader(new InputStreamReader(serverProcess.getInputStream()));
            String line;
            while ((line = reader.readLine()) != null) {
                Log.d(TAG, "[SIPEN] " + line);
                synchronized (logBuffer) {
                    if (logBuffer.size() > 150) {
                        logBuffer.remove(0);
                    }
                    logBuffer.add(line);
                }
            }

            int exitCode = serverProcess.waitFor();
            Log.w(TAG, "Server process keluar dengan kode: " + exitCode);
        } catch (Exception e) {
            Log.e(TAG, "Error saat menjalankan proses server Go: " + e.getMessage(), e);
        }
    }

    private File extractBinaryFromApk(File destDir) {
        try {
            String sourceApk = getApplicationInfo().sourceDir;
            java.util.zip.ZipFile zipFile = new java.util.zip.ZipFile(sourceApk);
            String[] abis = (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) ? Build.SUPPORTED_ABIS : new String[]{Build.CPU_ABI, Build.CPU_ABI2};

            java.util.zip.ZipEntry targetEntry = null;
            for (String abi : abis) {
                String entryPath = "lib/" + abi + "/libsipen.so";
                targetEntry = zipFile.getEntry(entryPath);
                if (targetEntry != null) {
                    Log.i(TAG, "Ditemukan binary untuk ABI: " + abi);
                    break;
                }
            }

            if (targetEntry == null) {
                java.util.Enumeration<? extends java.util.zip.ZipEntry> entries = zipFile.entries();
                while (entries.hasMoreElements()) {
                    java.util.zip.ZipEntry entry = entries.nextElement();
                    if (entry.getName().endsWith("libsipen.so")) {
                        targetEntry = entry;
                        break;
                    }
                }
            }

            if (targetEntry != null) {
                File dest = new File(destDir, "libsipen.so");
                InputStream in = zipFile.getInputStream(targetEntry);
                FileOutputStream out = new FileOutputStream(dest);
                byte[] buffer = new byte[8192];
                int read;
                while ((read = in.read(buffer)) != -1) {
                    out.write(buffer, 0, read);
                }
                out.flush();
                out.close();
                in.close();
                zipFile.close();
                dest.setExecutable(true, false);
                dest.setReadable(true, false);
                return dest;
            }
            zipFile.close();
        } catch (Exception e) {
            Log.e(TAG, "Gagal mengekstrak binary dari APK: " + e.getMessage(), e);
        }
        return null;
    }

    public static List<String> getRecentLogs() {
        synchronized (logBuffer) {
            return new ArrayList<>(logBuffer);
        }
    }

    @Override
    public void onDestroy() {
        isRunning = false;
        if (serverThread != null) {
            serverThread.interrupt();
        }
        if (serverProcess != null) {
            try {
                serverProcess.destroy();
            } catch (Exception ignored) {}
        }
        releaseLocks();
        super.onDestroy();
        Log.i(TAG, "SiPenDosaServerService dihentikan.");
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
