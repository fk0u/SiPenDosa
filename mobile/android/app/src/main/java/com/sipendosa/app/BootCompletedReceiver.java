package com.sipendosa.app;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.os.Build;
import android.util.Log;

/**
 * BootCompletedReceiver
 * Secara otomatis menyalakan background server SiPenDosa saat perangkat Android
 * selesai booting (reboot / nyala) atau aplikasi baru saja diperbarui.
 */
public class BootCompletedReceiver extends BroadcastReceiver {
    private static final String TAG = "SiPenDosaBoot";

    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null) return;

        String action = intent.getAction();
        Log.i(TAG, "Menerima broadcast: " + action);

        if (Intent.ACTION_BOOT_COMPLETED.equals(action) ||
            Intent.ACTION_MY_PACKAGE_REPLACED.equals(action) ||
            "android.intent.action.QUICKBOOT_POWERON".equals(action) ||
            "com.htc.intent.action.QUICKBOOT_POWERON".equals(action)) {

            Intent serviceIntent = new Intent(context, SiPenDosaServerService.class);
            try {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    context.startForegroundService(serviceIntent);
                } else {
                    context.startService(serviceIntent);
                }
                Log.i(TAG, "SiPenDosaServerService berhasil dipicu saat booting.");
            } catch (Exception e) {
                Log.e(TAG, "Gagal memicu SiPenDosaServerService saat booting: " + e.getMessage(), e);
            }
        }
    }
}
