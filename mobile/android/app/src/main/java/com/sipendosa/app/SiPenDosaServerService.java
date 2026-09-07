package com.sipendosa.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.os.Build;
import android.os.IBinder;
import android.os.PowerManager;
import android.util.Log;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.ArrayList;
import java.util.List;

public class SiPenDosaServerService extends Service implements Runnable {
    private static final String TAG = "SiPenDosaService";
    private static final String CHANNEL_ID = "sipendosa_service_channel";
    private static final int NOTIFICATION_ID = 8473;

    private PowerManager.WakeLock wakeLock;
    private Process serverProcess;
    private boolean isRunning = false;
    private static final List<String> logBuffer = new ArrayList<>();

    @Override
    public void onCreate() {
        super.onCreate();
        createNotificationChannel();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (!isRunning) {
            startForegroundServiceNotification();
            acquireWakeLock();
            Thread serverThread = new Thread(this);
            serverThread.start();
            isRunning = true;
        }
        return START_STICKY;
    }

    private void acquireWakeLock() {
        try {
            PowerManager powerManager = (PowerManager) getSystemService(Context.POWER_SERVICE);
            if (powerManager != null) {
                wakeLock = powerManager.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "SiPenDosa:ServerWakeLock");
                wakeLock.acquire();
                Log.i(TAG, "WakeLock acquired successfully");
            }
        } catch (Exception e) {
            Log.e(TAG, "Failed to acquire WakeLock", e);
        }
    }

    private void createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            NotificationChannel channel = new NotificationChannel(
                    CHANNEL_ID,
                    "SiPenDosa Background Server",
                    NotificationManager.IMPORTANCE_LOW
            );
            channel.setDescription("Menjaga server pengingat SiPenDosa tetap aktif di latar belakang");
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
                .setContentText("Port 8473 • Engine Online • 24/7 Otomasi")
                .setSmallIcon(android.R.drawable.stat_notify_sync)
                .setContentIntent(pendingIntent)
                .setOngoing(true)
                .build();

        startForeground(NOTIFICATION_ID, notification);
    }

    @Override
    public void run() {
        try {
            File filesDir = getFilesDir();
            File binFile = null;

            // 1. Prioritaskan eksekusi dari nativeLibraryDir (resmi diizinkan oleh SELinux Android 10-14+)
            File nativeDir = new File(getApplicationInfo().nativeLibraryDir);
            File nativeBin = new File(nativeDir, "libsipen.so");
            if (nativeBin.exists() && nativeBin.canExecute()) {
                binFile = nativeBin;
                Log.i(TAG, "Using native library binary: " + binFile.getAbsolutePath());
            }

            // 2. Fallback jika nativeLibraryDir belum terisi
            if (binFile == null) {
                File fallbackBin = new File(filesDir, "sipen");
                extractAsset("sipen", fallbackBin);
                fallbackBin.setExecutable(true, false);
                fallbackBin.setReadable(true, false);
                binFile = fallbackBin;
                Log.i(TAG, "Using fallback asset binary: " + binFile.getAbsolutePath());
            }

            File dataDir = new File(filesDir, "data");
            File sessionDir = new File(filesDir, "session");
            dataDir.mkdirs();
            sessionDir.mkdirs();

            ProcessBuilder pb = new ProcessBuilder(binFile.getAbsolutePath());
            pb.directory(filesDir);
            pb.environment().put("PORT", "8473");
            pb.environment().put("HOST", "0.0.0.0");
            pb.environment().put("APP_ENV", "production");
            pb.environment().put("DB_PATH", new File(dataDir, "sipen.db").getAbsolutePath());
            pb.environment().put("WA_SESSION_PATH", new File(sessionDir, "whatsapp.db").getAbsolutePath());
            pb.environment().put("SESSION_SECRET", "sipendosa-android-service-session-key-32-chars");

            pb.redirectErrorStream(true);
            serverProcess = pb.start();
            Log.i(TAG, "Server process started: " + binFile.getAbsolutePath());

            BufferedReader reader = new BufferedReader(new InputStreamReader(serverProcess.getInputStream()));
            String line;
            while ((line = reader.readLine()) != null) {
                Log.d(TAG, "[SIPEN] " + line);
                synchronized (logBuffer) {
                    if (logBuffer.size() > 100) {
                        logBuffer.remove(0);
                    }
                    logBuffer.add(line);
                }
            }

            int exitCode = serverProcess.waitFor();
            Log.w(TAG, "Server process exited with code: " + exitCode);
        } catch (Exception e) {
            Log.e(TAG, "Error starting server process", e);
        }
    }

    private void extractAsset(String assetName, File dest) {
        try {
            if (dest.exists() && dest.length() > 0) {
                return;
            }
            InputStream in = getAssets().open(assetName);
            FileOutputStream out = new FileOutputStream(dest);
            byte[] buffer = new byte[8192];
            int read;
            while ((read = in.read(buffer)) != -1) {
                out.write(buffer, 0, read);
            }
            out.flush();
            out.close();
            in.close();
        } catch (Exception e) {
            Log.w(TAG, "Asset " + assetName + " not found or cannot extract: " + e.getMessage());
        }
    }

    public static List<String> getRecentLogs() {
        synchronized (logBuffer) {
            return new ArrayList<>(logBuffer);
        }
    }

    @Override
    public void onDestroy() {
        super.onDestroy();
        if (serverProcess != null) {
            try {
                serverProcess.destroy();
            } catch (Exception ignored) {}
        }
        if (wakeLock != null && wakeLock.isHeld()) {
            try {
                wakeLock.release();
            } catch (Exception ignored) {}
        }
        isRunning = false;
        Log.i(TAG, "Service destroyed");
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
