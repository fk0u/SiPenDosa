package com.sipendosa.app;

import android.webkit.JavascriptInterface;

public class AndroidBridge {
    private final MainActivity activity;

    public AndroidBridge(MainActivity activity) {
        this.activity = activity;
    }

    @JavascriptInterface
    public boolean isAndroidApp() {
        return true;
    }

    @JavascriptInterface
    public String getAppVersion() {
        return "1.1.1";
    }

    @JavascriptInterface
    public void downloadAndInstallUpdate(final String apkUrl, final String versionName) {
        activity.runOnUiThread(ActionTask.startDownload(activity, apkUrl, versionName));
    }
}
