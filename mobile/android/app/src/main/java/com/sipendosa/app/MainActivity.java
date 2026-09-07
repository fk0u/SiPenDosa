package com.sipendosa.app;

import android.app.Activity;
import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.content.Intent;
import android.graphics.Color;
import android.graphics.Typeface;
import android.net.Uri;
import android.net.wifi.WifiManager;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.format.Formatter;
import android.util.TypedValue;
import android.view.View;
import android.view.ViewGroup;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.widget.Button;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;

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
                URL url = new URL(SERVER_URL + "/api/health");
                HttpURLConnection conn = (HttpURLConnection) url.openConnection();
                conn.setConnectTimeout(1000);
                conn.setReadTimeout(1000);
                conn.setRequestMethod("GET");
                int code = conn.getResponseCode();
                if (code == 200 || code == 302 || code == 401) {
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
}
