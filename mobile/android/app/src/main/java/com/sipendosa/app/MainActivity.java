package com.sipendosa.app;

import android.app.Activity;
import android.app.ProgressDialog;
import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
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
import android.view.View;
import android.view.ViewGroup;
import android.webkit.JavascriptInterface;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.widget.Button;
import android.widget.LinearLayout;
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
    private TextView tvStatus;
    private TextView tvIp;
    private Button btnCopy;
    private Button btnBrowser;
    private Button btnTerminal;
    private Handler handler;
    private boolean isServerReady = false;
    private String lanIp = "127.0.0.1";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        if (android.os.Build.VERSION.SDK_INT >= 33) {
            if (checkSelfPermission("android.permission.POST_NOTIFICATIONS") != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                requestPermissions(new String[]{"android.permission.POST_NOTIFICATIONS"}, 101);
            }
        }

        Intent serviceIntent = new Intent(this, SiPenDosaServerService.class);
        startService(serviceIntent);

        handler = new Handler(Looper.getMainLooper());
        lanIp = getLanIpAddress();

        LinearLayout rootLayout = new LinearLayout(this);
        rootLayout.setOrientation(LinearLayout.VERTICAL);
        rootLayout.setBackgroundColor(Color.parseColor("#08090d"));
        rootLayout.setLayoutParams(new ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));

        LinearLayout topBar = new LinearLayout(this);
        topBar.setOrientation(LinearLayout.VERTICAL);
        topBar.setBackgroundColor(Color.parseColor("#11131a"));
        topBar.setPadding(dp(16), dp(16), dp(16), dp(12));
        topBar.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
        ));

        TextView tvTitle = new TextView(this);
        tvTitle.setText("⚡ SiPenDosa Terminal Console");
        tvTitle.setTextColor(Color.parseColor("#f59e0b"));
        tvTitle.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16);
        tvTitle.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        topBar.addView(tvTitle);

        tvStatus = new TextView(this);
        tvStatus.setText("Status: Memulai server background...");
        tvStatus.setTextColor(Color.parseColor("#e11d48"));
        tvStatus.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
        tvStatus.setTypeface(Typeface.MONOSPACE);
        tvStatus.setPadding(0, dp(4), 0, 0);
        topBar.addView(tvStatus);

        tvIp = new TextView(this);
        tvIp.setText("Akses LAN: http://" + lanIp + ":8473");
        tvIp.setTextColor(Color.parseColor("#94a3b8"));
        tvIp.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        tvIp.setTypeface(Typeface.MONOSPACE);
        tvIp.setPadding(0, dp(2), 0, dp(8));
        topBar.addView(tvIp);

        LinearLayout btnRow = new LinearLayout(this);
        btnRow.setOrientation(LinearLayout.HORIZONTAL);

        btnCopy = createButton("Salin IP", Color.parseColor("#1e293b"), Color.parseColor("#e2e8f0"));
        btnCopy.setOnClickListener(this);
        btnRow.addView(btnCopy);

        btnBrowser = createButton("Buka Browser", Color.parseColor("#e11d48"), Color.WHITE);
        btnBrowser.setOnClickListener(this);
        btnRow.addView(btnBrowser);

        btnTerminal = createButton("Konsol Shell", Color.parseColor("#f59e0b"), Color.parseColor("#08090d"));
        btnTerminal.setOnClickListener(this);
        btnRow.addView(btnTerminal);

        topBar.addView(btnRow);
        rootLayout.addView(topBar);

        webView = new WebView(this);
        webView.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                0,
                1.0f
        ));
        webView.setBackgroundColor(Color.parseColor("#08090d"));

        WebSettings ws = webView.getSettings();
        ws.setJavaScriptEnabled(true);
        ws.setDomStorageEnabled(true);
        ws.setUseWideViewPort(true);
        ws.setLoadWithOverviewMode(true);

        webView.setWebViewClient(new InternalWebClient());
        webView.addJavascriptInterface(new AndroidBridge(this), "AndroidBridge");
        rootLayout.addView(webView);

        setContentView(rootLayout);

        Thread checkThread = new Thread(this);
        checkThread.start();
    }

    private Button createButton(String text, int bgColor, int textColor) {
        Button btn = new Button(this);
        btn.setText(text);
        btn.setTextColor(textColor);
        btn.setBackgroundColor(bgColor);
        btn.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        btn.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(0, dp(36), 1.0f);
        params.setMargins(dp(2), 0, dp(2), 0);
        btn.setLayoutParams(params);
        return btn;
    }

    @Override
    public void onClick(View v) {
        if (v == btnCopy) {
            ClipboardManager clipboard = (ClipboardManager) getSystemService(Context.CLIPBOARD_SERVICE);
            ClipData clip = ClipData.newPlainText("SiPenDosa", "http://" + lanIp + ":8473");
            if (clipboard != null) {
                clipboard.setPrimaryClip(clip);
                Toast.makeText(this, "Tautan server disalin!", Toast.LENGTH_SHORT).show();
            }
        } else if (v == btnBrowser) {
            Intent browserIntent = new Intent(Intent.ACTION_VIEW, Uri.parse(SERVER_URL));
            startActivity(browserIntent);
        } else if (v == btnTerminal) {
            if (webView != null) {
                webView.loadUrl(SERVER_URL + "/terminal");
            }
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
                Thread.sleep(800);
            } catch (InterruptedException ignored) {}
        }

        handler.post(new UpdateUiTask(this, isServerReady));
    }

    public void onServerStatusChecked(boolean ready) {
        if (ready) {
            tvStatus.setText("Status: ● ONLINE (Port 8473)");
            tvStatus.setTextColor(Color.parseColor("#10b981"));
            webView.loadUrl(SERVER_URL);
        } else {
            tvStatus.setText("Status: Menginisialisasi daemon...");
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

    private ProgressDialog progressDialog;

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
            conn.setRequestProperty("User-Agent", "SiPenDosa-Android/1.1.0");
            conn.connect();

            int status = conn.getResponseCode();
            if (status == HttpURLConnection.HTTP_MOVED_TEMP || status == HttpURLConnection.HTTP_MOVED_PERM || status == 307 || status == 308) {
                String newUrl = conn.getHeaderField("Location");
                conn.disconnect();
                url = new URL(newUrl);
                conn = (HttpURLConnection) url.openConnection();
                conn.setRequestProperty("User-Agent", "SiPenDosa-Android/1.1.0");
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
}
