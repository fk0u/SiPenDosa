package com.sipendosa.app;

import android.app.Activity;
import android.app.AlertDialog;
import android.app.ProgressDialog;
import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.content.DialogInterface;
import android.content.Intent;
import android.graphics.Color;
import android.graphics.Typeface;
import android.net.Uri;
import android.net.wifi.WifiManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.Settings;
import android.text.format.Formatter;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowManager;
import android.webkit.JsResult;
import android.webkit.WebChromeClient;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.TextView;
import android.widget.Toast;

import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;

public class MainActivity extends Activity implements View.OnClickListener, Runnable {
    private static final String SERVER_URL = "http://127.0.0.1:8473";
    private WebView webView;
    private FrameLayout rootContainer;
    private LinearLayout splashOverlay;
    private TextView tvSplashStatus;
    private ProgressBar splashProgress;
    private Button btnLanInfo;

    private Handler handler;
    private boolean isServerReady = false;
    private String lanIp = "127.0.0.1";
    private ProgressDialog progressDialog;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // 1. Android Immersive Dark Bar Styling
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            Window window = getWindow();
            window.addFlags(WindowManager.LayoutParams.FLAG_DRAWS_SYSTEM_BAR_BACKGROUNDS);
            window.setStatusBarColor(Color.parseColor("#08090d"));
            window.setNavigationBarColor(Color.parseColor("#090a0f"));
        }

        // 2. Notifikasi Permission untuk Android 13+
        if (Build.VERSION.SDK_INT >= 33) {
            if (checkSelfPermission("android.permission.POST_NOTIFICATIONS") != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                requestPermissions(new String[]{"android.permission.POST_NOTIFICATIONS"}, 101);
            }
        }

        // 3. Start Background Daemon Service
        Intent serviceIntent = new Intent(this, SiPenDosaServerService.class);
        startService(serviceIntent);

        handler = new Handler(Looper.getMainLooper());
        lanIp = getLanIpAddress();

        // 4. Root Container
        rootContainer = new FrameLayout(this);
        rootContainer.setBackgroundColor(Color.parseColor("#08090d"));
        rootContainer.setLayoutParams(new ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));

        // 5. Native-like Fullscreen WebView
        webView = new WebView(this);
        webView.setLayoutParams(new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));
        webView.setBackgroundColor(Color.parseColor("#08090d"));
        webView.setOverScrollMode(View.OVER_SCROLL_NEVER);

        WebSettings ws = webView.getSettings();
        ws.setJavaScriptEnabled(true);
        ws.setDomStorageEnabled(true);
        ws.setDatabaseEnabled(true);
        ws.setUseWideViewPort(true);
        ws.setLoadWithOverviewMode(true);
        ws.setSupportZoom(false);
        ws.setBuiltInZoomControls(false);
        ws.setDisplayZoomControls(false);
        ws.setCacheMode(WebSettings.LOAD_DEFAULT);
        ws.setAllowFileAccess(true);
        ws.setAllowContentAccess(true);

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.KITKAT) {
            WebView.setWebContentsDebuggingEnabled(false);
        }

        webView.setWebViewClient(new InternalWebClient());
        webView.setWebChromeClient(new CustomWebChromeClient(this));
        webView.addJavascriptInterface(new AndroidBridge(this), "AndroidBridge");
        rootContainer.addView(webView);

        // 6. Discreet Floating Action Pill (Top-Right: LAN & Info)
        btnLanInfo = new Button(this);
        btnLanInfo.setText("⚡ IP");
        btnLanInfo.setTextColor(Color.parseColor("#f59e0b"));
        btnLanInfo.setBackgroundColor(Color.parseColor("#1e293b"));
        btnLanInfo.setTextSize(TypedValue.COMPLEX_UNIT_SP, 10);
        btnLanInfo.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        FrameLayout.LayoutParams lanParams = new FrameLayout.LayoutParams(dp(44), dp(28));
        lanParams.gravity = Gravity.TOP | Gravity.END;
        lanParams.setMargins(0, dp(10), dp(12), 0);
        btnLanInfo.setLayoutParams(lanParams);
        btnLanInfo.setOnClickListener(this);
        btnLanInfo.setAlpha(0.65f);
        rootContainer.addView(btnLanInfo);

        // 7. Sleek Native Splash Loading Overlay
        buildSplashOverlay();
        rootContainer.addView(splashOverlay);

        setContentView(rootContainer);

        // 8. Start Background Health Checker Thread
        Thread checkThread = new Thread(this);
        checkThread.start();
    }

    private void buildSplashOverlay() {
        splashOverlay = new LinearLayout(this);
        splashOverlay.setOrientation(LinearLayout.VERTICAL);
        splashOverlay.setGravity(Gravity.CENTER);
        splashOverlay.setBackgroundColor(Color.parseColor("#08090d"));
        splashOverlay.setLayoutParams(new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));

        // Brand Icon / Avatar
        ImageView logoView = new ImageView(this);
        int iconRes = getResources().getIdentifier("icon", "drawable", getPackageName());
        if (iconRes != 0) {
            logoView.setImageResource(iconRes);
        }
        LinearLayout.LayoutParams logoParams = new LinearLayout.LayoutParams(dp(88), dp(88));
        logoParams.setMargins(0, 0, 0, dp(20));
        logoView.setLayoutParams(logoParams);
        splashOverlay.addView(logoView);

        // App Title
        TextView tvTitle = new TextView(this);
        tvTitle.setText("SiPenDosa");
        tvTitle.setTextColor(Color.WHITE);
        tvTitle.setTextSize(TypedValue.COMPLEX_UNIT_SP, 26);
        tvTitle.setTypeface(Typeface.SANS_SERIF, Typeface.BOLD);
        tvTitle.setGravity(Gravity.CENTER);
        splashOverlay.addView(tvTitle);

        // Subtitle
        TextView tvSub = new TextView(this);
        tvSub.setText("Academic Assistant • v1.1.1");
        tvSub.setTextColor(Color.parseColor("#f59e0b"));
        tvSub.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        tvSub.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        tvSub.setGravity(Gravity.CENTER);
        tvSub.setPadding(0, dp(4), 0, dp(24));
        splashOverlay.addView(tvSub);

        // Spinner
        splashProgress = new ProgressBar(this);
        LinearLayout.LayoutParams progParams = new LinearLayout.LayoutParams(dp(36), dp(36));
        progParams.setMargins(0, 0, 0, dp(16));
        splashProgress.setLayoutParams(progParams);
        splashOverlay.addView(splashProgress);

        // Status Text
        tvSplashStatus = new TextView(this);
        tvSplashStatus.setText("Menghubungkan layanan pengingat...");
        tvSplashStatus.setTextColor(Color.parseColor("#94a3b8"));
        tvSplashStatus.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        tvSplashStatus.setTypeface(Typeface.SANS_SERIF);
        tvSplashStatus.setGravity(Gravity.CENTER);
        splashOverlay.addView(tvSplashStatus);
    }

    @Override
    public void onClick(View v) {
        if (v == btnLanInfo) {
            showLanDialog();
        }
    }

    private void showLanDialog() {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("⚡ Akses SiPenDosa Jaringan (LAN)");
        builder.setMessage("Aplikasi ini berjalan sebagai server mandiri.\n\n" +
                "• URL Lokal: " + SERVER_URL + "\n" +
                "• URL Jaringan (Wi-Fi): http://" + lanIp + ":8473\n\n" +
                "Perangkat lain di satu jaringan dapat membuka alamat di atas untuk mengakses dashboard.");

        builder.setPositiveButton("Salin URL", new LanDialogClickListener(this, 1));
        builder.setNeutralButton("Buka Browser", new LanDialogClickListener(this, 2));
        builder.setNegativeButton("Tutup", null);
        builder.show();
    }

    public void handleLanDialogClick(int whichAction) {
        if (whichAction == 1) {
            ClipboardManager clipboard = (ClipboardManager) getSystemService(Context.CLIPBOARD_SERVICE);
            ClipData clip = ClipData.newPlainText("SiPenDosa", "http://" + lanIp + ":8473");
            if (clipboard != null) {
                clipboard.setPrimaryClip(clip);
                Toast.makeText(this, "URL disalin ke clipboard!", Toast.LENGTH_SHORT).show();
            }
        } else if (whichAction == 2) {
            Intent browserIntent = new Intent(Intent.ACTION_VIEW, Uri.parse(SERVER_URL));
            startActivity(browserIntent);
        }
    }

    @Override
    public void run() {
        int attempts = 0;
        while (attempts < 60 && !isServerReady) {
            attempts++;
            try {
                URL url = new URL(SERVER_URL + "/healthz");
                HttpURLConnection conn = (HttpURLConnection) url.openConnection();
                conn.setConnectTimeout(1000);
                conn.setReadTimeout(1000);
                conn.setRequestMethod("GET");
                int code = conn.getResponseCode();
                if (code >= 200 && code < 500) {
                    isServerReady = true;
                    break;
                }
            } catch (Exception ignored) {}

            try {
                Thread.sleep(600);
            } catch (InterruptedException ignored) {}
        }

        handler.post(new UpdateUiTask(this, isServerReady));
    }

    public void onServerStatusChecked(boolean ready) {
        if (ready) {
            webView.loadUrl(SERVER_URL);
            if (splashOverlay != null) {
                splashOverlay.setVisibility(View.GONE);
            }
        } else {
            if (tvSplashStatus != null) {
                tvSplashStatus.setText("Memulai daemon server...");
            }
        }
    }

    // =========================================================================
    // AUTOMATIC UPDATE ENGINE (1-KLIK INSTALASI TANPA RIBET)
    // =========================================================================
    public void startApkDownloadAndInstall(final String apkUrl, final String versionName) {
        if (apkUrl == null || apkUrl.isEmpty()) {
            Toast.makeText(this, "URL pembaruan tidak valid", Toast.LENGTH_SHORT).show();
            return;
        }

        progressDialog = new ProgressDialog(this);
        progressDialog.setTitle("Pembaruan SiPenDosa");
        progressDialog.setMessage("Mengunduh versi " + (versionName != null && !versionName.isEmpty() ? versionName : "terbaru") + "...");
        progressDialog.setProgressStyle(ProgressDialog.STYLE_HORIZONTAL);
        progressDialog.setMax(100);
        progressDialog.setProgress(0);
        progressDialog.setCancelable(false);
        progressDialog.show();

        new Thread(ActionTask.doDownload(this, apkUrl)).start();
    }

    public void onDownloadProgress(int progress) {
        if (progressDialog != null && progressDialog.isShowing()) {
            progressDialog.setProgress(progress);
        }
    }

    public void onDownloadSuccess(File apkFile) {
        if (progressDialog != null && progressDialog.isShowing()) {
            progressDialog.dismiss();
        }
        Toast.makeText(this, "✓ Unduhan selesai! Membuka instalasi...", Toast.LENGTH_SHORT).show();
        installApk(apkFile);
    }

    public void onDownloadError(String error) {
        if (progressDialog != null && progressDialog.isShowing()) {
            progressDialog.dismiss();
        }
        Toast.makeText(this, "Gagal mengunduh pembaruan: " + error, Toast.LENGTH_LONG).show();
    }

    public void downloadApkInternal(String targetUrl) {
        File apkFile = null;
        try {
            File baseDir = getExternalFilesDir(null);
            if (baseDir == null) {
                baseDir = getCacheDir();
            }
            apkFile = new File(baseDir, "SiPenDosa-Update.apk");
            if (apkFile.exists()) {
                apkFile.delete();
            }

            URL url = new URL(targetUrl);
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setInstanceFollowRedirects(true);
            conn.setRequestProperty("User-Agent", "SiPenDosa-Android/1.1.1");
            conn.connect();

            int status = conn.getResponseCode();
            if (status == HttpURLConnection.HTTP_MOVED_TEMP || status == HttpURLConnection.HTTP_MOVED_PERM || status == 307 || status == 308) {
                String newUrl = conn.getHeaderField("Location");
                conn.disconnect();
                url = new URL(newUrl);
                conn = (HttpURLConnection) url.openConnection();
                conn.setRequestProperty("User-Agent", "SiPenDosa-Android/1.1.1");
                conn.connect();
            }

            final int fileLength = conn.getContentLength();
            InputStream is = conn.getInputStream();
            FileOutputStream fos = new FileOutputStream(apkFile);

            byte[] buffer = new byte[8192];
            long total = 0;
            int count;
            while ((count = is.read(buffer)) != -1) {
                total += count;
                fos.write(buffer, 0, count);
                if (fileLength > 0) {
                    final int progress = (int) (total * 100 / fileLength);
                    handler.post(ActionTask.progress(this, progress));
                }
            }
            fos.flush();
            fos.close();
            is.close();
            conn.disconnect();

            handler.post(ActionTask.success(this, apkFile));
        } catch (Exception e) {
            handler.post(ActionTask.failed(this, e.getMessage()));
        }
    }

    public void installApk(File apkFile) {
        if (apkFile == null || !apkFile.exists()) {
            Toast.makeText(this, "Berkas APK pembaruan tidak ditemukan", Toast.LENGTH_SHORT).show();
            return;
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            if (!getPackageManager().canRequestPackageInstalls()) {
                Toast.makeText(this, "Mohon izinkan pemasangan aplikasi dari sumber ini untuk melanjutkan", Toast.LENGTH_LONG).show();
                Intent permIntent = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:" + getPackageName()));
                startActivity(permIntent);
                return;
            }
        }

        try {
            Intent intent = new Intent(Intent.ACTION_VIEW);
            intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            Uri apkUri;
            if (Build.VERSION.SDK_INT >= 24) {
                apkUri = Uri.parse("content://" + ApkFileProvider.AUTHORITY + "/" + apkFile.getName());
                intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
            } else {
                apkUri = Uri.fromFile(apkFile);
            }
            intent.setDataAndType(apkUri, "application/vnd.android.package-archive");
            startActivity(intent);
        } catch (Exception e) {
            Toast.makeText(this, "Gagal membuka installer: " + e.getMessage(), Toast.LENGTH_LONG).show();
        }
    }

    private String getLanIpAddress() {
        try {
            WifiManager wm = (WifiManager) getApplicationContext().getSystemService(Context.WIFI_SERVICE);
            if (wm != null) {
                int ip = wm.getConnectionInfo().getIpAddress();
                return Formatter.formatIpAddress(ip);
            }
        } catch (Exception ignored) {}
        return "127.0.0.1";
    }

    private int dp(int value) {
        return (int) TypedValue.applyDimension(
                TypedValue.COMPLEX_UNIT_DIP,
                value,
                getResources().getDisplayMetrics()
        );
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
        } else {
            super.onBackPressed();
        }
    }
}
