package com.sipendosa.app;

import android.app.Activity;
import android.app.AlertDialog;
import android.app.ProgressDialog;
import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.content.Intent;
import android.graphics.Color;
import android.graphics.Typeface;
import android.graphics.drawable.GradientDrawable;
import android.net.Uri;
import android.net.wifi.WifiManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.Settings;
import android.text.InputType;
import android.text.format.Formatter;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.HapticFeedbackConstants;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowManager;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.widget.Button;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.ImageView;
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.TextView;
import android.widget.Toast;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;

public class MainActivity extends Activity implements View.OnClickListener, Runnable {
    public static final String SERVER_URL = "http://127.0.0.1:8473";

    private static final String[] TAB_ROUTES = {
            "/",
            "/contacts",
            "/schedules",
            "/queue",
            "/settings"
    };

    // Theme Colors (Crimson Scarlet, Burnished Gold, Emerald, Obsidian Dark)
    private static final int COLOR_BG = Color.parseColor("#08090d");
    private static final int COLOR_SURFACE = Color.parseColor("#0d1017");
    private static final int COLOR_CRIMSON = Color.parseColor("#e11d48");
    private static final int COLOR_GOLD = Color.parseColor("#f59e0b");
    private static final int COLOR_EMERALD = Color.parseColor("#10b981");
    private static final int COLOR_TEXT_MUTED = Color.parseColor("#64748b");
    private static final int COLOR_TEXT_ACTIVE = Color.parseColor("#ffffff");

    private FrameLayout rootContainer;
    private LinearLayout mainContentLayout;
    private LinearLayout topNavBar;
    private LinearLayout bottomNavBar;
    private WebView webView;
    private ProgressBar pageProgressBar;
    private LinearLayout splashOverlay;
    private TextView tvSplashStatus;

    // Top Bar Views
    private View daemonStatusDot;
    private TextView tvDaemonStatus;
    private Button btnLan;
    private Button btnWaPair;
    private Button btnRefresh;

    // Bottom Nav Tabs (5 items)
    private final LinearLayout[] navTabViews = new LinearLayout[5];
    private final TextView[] navIconViews = new TextView[5];
    private final TextView[] navLabelViews = new TextView[5];
    private final View[] navIndicatorViews = new View[5];
    private int currentTab = 0;

    private Handler handler;
    private boolean isServerReady = false;
    private String lanIp = "127.0.0.1";
    private ProgressDialog progressDialog;
    private ProgressDialog pairProgressDialog;
    private EditText phoneInput;
    private String lastPairCode = "";
    private long backPressedTime = 0;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // 1. Edge-to-Edge System Dark Bars
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            Window window = getWindow();
            window.addFlags(WindowManager.LayoutParams.FLAG_DRAWS_SYSTEM_BAR_BACKGROUNDS);
            window.setStatusBarColor(COLOR_BG);
            window.setNavigationBarColor(COLOR_SURFACE);
        }

        // 2. Notification Permission for Android 13+
        if (Build.VERSION.SDK_INT >= 33) {
            if (checkSelfPermission("android.permission.POST_NOTIFICATIONS") != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                requestPermissions(new String[]{"android.permission.POST_NOTIFICATIONS"}, 101);
            }
        }

        // 3. Start Background Server Daemon Service
        Intent serviceIntent = new Intent(this, SiPenDosaServerService.class);
        startService(serviceIntent);

        handler = new Handler(Looper.getMainLooper());
        lanIp = getLanIpAddress();

        // 4. Root Container
        rootContainer = new FrameLayout(this);
        rootContainer.setBackgroundColor(COLOR_BG);
        rootContainer.setLayoutParams(new ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));

        // 5. Main Content Vertical Layout
        mainContentLayout = new LinearLayout(this);
        mainContentLayout.setOrientation(LinearLayout.VERTICAL);
        mainContentLayout.setLayoutParams(new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));

        // Top Navigation Bar
        buildTopNavBar();
        mainContentLayout.addView(topNavBar);

        // Page Progress Bar (Horizontal line under top bar)
        pageProgressBar = new ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal);
        pageProgressBar.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                dp(2)
        ));
        pageProgressBar.setProgressDrawable(createProgressDrawable());
        pageProgressBar.setVisibility(View.GONE);
        mainContentLayout.addView(pageProgressBar);

        // WebView Container
        FrameLayout webContainer = new FrameLayout(this);
        LinearLayout.LayoutParams webContainerParams = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                0,
                1.0f
        );
        webContainer.setLayoutParams(webContainerParams);

        webView = new WebView(this);
        webView.setLayoutParams(new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));
        webView.setBackgroundColor(COLOR_BG);
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

        webView.setWebViewClient(new InternalWebClient(this));
        webView.setWebChromeClient(new CustomWebChromeClient(this));
        webView.addJavascriptInterface(new AndroidBridge(this), "AndroidBridge");
        webContainer.addView(webView);
        mainContentLayout.addView(webContainer);

        // Bottom Navigation Bar
        buildBottomNavBar();
        mainContentLayout.addView(bottomNavBar);

        rootContainer.addView(mainContentLayout);

        // Sleek Splash Overlay
        buildSplashOverlay();
        rootContainer.addView(splashOverlay);

        setContentView(rootContainer);

        // Start Background Health Checker Thread
        Thread checkThread = new Thread(this);
        checkThread.start();
    }

    private void buildTopNavBar() {
        topNavBar = new LinearLayout(this);
        topNavBar.setOrientation(LinearLayout.HORIZONTAL);
        topNavBar.setGravity(Gravity.CENTER_VERTICAL);
        topNavBar.setBackgroundColor(COLOR_SURFACE);
        topNavBar.setPadding(dp(14), dp(8), dp(14), dp(8));
        topNavBar.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                dp(48)
        ));

        // Brand & Status Pill
        LinearLayout brandLayout = new LinearLayout(this);
        brandLayout.setOrientation(LinearLayout.HORIZONTAL);
        brandLayout.setGravity(Gravity.CENTER_VERTICAL);
        LinearLayout.LayoutParams brandParams = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1.0f);
        brandLayout.setLayoutParams(brandParams);

        TextView tvLogo = new TextView(this);
        tvLogo.setText("SiPenDosa");
        tvLogo.setTextColor(Color.WHITE);
        tvLogo.setTextSize(TypedValue.COMPLEX_UNIT_SP, 15);
        tvLogo.setTypeface(Typeface.SANS_SERIF, Typeface.BOLD);
        brandLayout.addView(tvLogo);

        // Live Daemon Status Indicator
        LinearLayout statusPill = new LinearLayout(this);
        statusPill.setOrientation(LinearLayout.HORIZONTAL);
        statusPill.setGravity(Gravity.CENTER_VERTICAL);
        statusPill.setPadding(dp(6), dp(2), dp(6), dp(2));
        GradientDrawable pillBg = new GradientDrawable();
        pillBg.setCornerRadius(dp(10));
        pillBg.setColor(Color.parseColor("#141923"));
        pillBg.setStroke(dp(1), Color.parseColor("#222c3d"));
        statusPill.setBackground(pillBg);
        LinearLayout.LayoutParams pillParams = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT
        );
        pillParams.setMargins(dp(8), 0, 0, 0);
        statusPill.setLayoutParams(pillParams);

        daemonStatusDot = new View(this);
        daemonStatusDot.setLayoutParams(new LinearLayout.LayoutParams(dp(7), dp(7)));
        GradientDrawable dotBg = new GradientDrawable();
        dotBg.setShape(GradientDrawable.OVAL);
        dotBg.setColor(COLOR_GOLD);
        daemonStatusDot.setBackground(dotBg);
        statusPill.addView(daemonStatusDot);

        tvDaemonStatus = new TextView(this);
        tvDaemonStatus.setText("STARTING");
        tvDaemonStatus.setTextColor(COLOR_GOLD);
        tvDaemonStatus.setTextSize(TypedValue.COMPLEX_UNIT_SP, 9);
        tvDaemonStatus.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        tvDaemonStatus.setPadding(dp(4), 0, 0, 0);
        statusPill.addView(tvDaemonStatus);
        brandLayout.addView(statusPill);

        topNavBar.addView(brandLayout);

        // Action Buttons: WhatsApp Pairing, LAN Share, Reload
        btnWaPair = createTopButton("📱 WA", COLOR_EMERALD);
        btnWaPair.setOnClickListener(this);
        topNavBar.addView(btnWaPair);

        btnLan = createTopButton("⚡ LAN", COLOR_GOLD);
        btnLan.setOnClickListener(this);
        topNavBar.addView(btnLan);

        btnRefresh = createTopButton("🔄", Color.parseColor("#94a3b8"));
        btnRefresh.setOnClickListener(this);
        topNavBar.addView(btnRefresh);
    }

    private Button createTopButton(String text, int textColor) {
        Button btn = new Button(this);
        btn.setText(text);
        btn.setTextColor(textColor);
        btn.setTextSize(TypedValue.COMPLEX_UNIT_SP, 10);
        btn.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);

        GradientDrawable btnBg = new GradientDrawable();
        btnBg.setColor(Color.parseColor("#161b26"));
        btnBg.setCornerRadius(dp(8));
        btnBg.setStroke(dp(1), Color.parseColor("#222c3d"));
        btn.setBackground(btnBg);

        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                dp(30)
        );
        lp.setMargins(dp(4), 0, 0, 0);
        btn.setLayoutParams(lp);
        btn.setPadding(dp(8), 0, dp(8), 0);
        return btn;
    }

    private void buildBottomNavBar() {
        bottomNavBar = new LinearLayout(this);
        bottomNavBar.setOrientation(LinearLayout.HORIZONTAL);
        bottomNavBar.setBackgroundColor(COLOR_SURFACE);
        bottomNavBar.setLayoutParams(new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                dp(58)
        ));

        String[][] tabs = {
                {"🏠", "Beranda"},
                {"👥", "Dosen"},
                {"📅", "Jadwal"},
                {"📨", "Antrean"},
                {"⚙️", "Setelan"}
        };

        for (int i = 0; i < 5; i++) {
            LinearLayout tab = new LinearLayout(this);
            tab.setOrientation(LinearLayout.VERTICAL);
            tab.setGravity(Gravity.CENTER);
            LinearLayout.LayoutParams tabParams = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 1.0f);
            tab.setLayoutParams(tabParams);
            tab.setClickable(true);
            tab.setTag(Integer.valueOf(i));
            tab.setOnClickListener(this);

            // Active indicator pill (top of tab)
            View indicator = new View(this);
            LinearLayout.LayoutParams indParams = new LinearLayout.LayoutParams(dp(24), dp(3));
            indParams.setMargins(0, 0, 0, dp(4));
            indicator.setLayoutParams(indParams);
            GradientDrawable indBg = new GradientDrawable();
            indBg.setCornerRadius(dp(2));
            indBg.setColor(i == 0 ? COLOR_CRIMSON : Color.TRANSPARENT);
            indicator.setBackground(indBg);

            // Icon
            TextView tvIcon = new TextView(this);
            tvIcon.setText(tabs[i][0]);
            tvIcon.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16);
            tvIcon.setGravity(Gravity.CENTER);

            // Label
            TextView tvLabel = new TextView(this);
            tvLabel.setText(tabs[i][1]);
            tvLabel.setTextSize(TypedValue.COMPLEX_UNIT_SP, 10);
            tvLabel.setTypeface(Typeface.SANS_SERIF, Typeface.BOLD);
            tvLabel.setTextColor(i == 0 ? COLOR_TEXT_ACTIVE : COLOR_TEXT_MUTED);
            tvLabel.setGravity(Gravity.CENTER);
            tvLabel.setPadding(0, dp(1), 0, 0);

            tab.addView(indicator);
            tab.addView(tvIcon);
            tab.addView(tvLabel);

            navTabViews[i] = tab;
            navIndicatorViews[i] = indicator;
            navIconViews[i] = tvIcon;
            navLabelViews[i] = tvLabel;

            bottomNavBar.addView(tab);
        }
    }

    private void selectTab(int index, boolean animate) {
        currentTab = index;
        for (int i = 0; i < 5; i++) {
            boolean isActive = (i == index);
            GradientDrawable indBg = new GradientDrawable();
            indBg.setCornerRadius(dp(2));
            indBg.setColor(isActive ? COLOR_CRIMSON : Color.TRANSPARENT);
            navIndicatorViews[i].setBackground(indBg);
            navLabelViews[i].setTextColor(isActive ? COLOR_TEXT_ACTIVE : COLOR_TEXT_MUTED);
            navIconViews[i].setAlpha(isActive ? 1.0f : 0.6f);
        }
    }

    private void loadRoute(String path) {
        if (webView != null) {
            webView.loadUrl(SERVER_URL + path);
        }
    }

    public void onPageStarted(String url) {
        if (pageProgressBar != null) {
            pageProgressBar.setVisibility(View.VISIBLE);
            pageProgressBar.setProgress(25);
        }
    }

    public void onPageNavigated(String url) {
        if (pageProgressBar != null) {
            pageProgressBar.setVisibility(View.GONE);
        }

        if (url == null) return;

        int newTab = 0;
        if (url.contains("/contacts")) {
            newTab = 1;
        } else if (url.contains("/schedules")) {
            newTab = 2;
        } else if (url.contains("/queue")) {
            newTab = 3;
        } else if (url.contains("/settings") || url.contains("/changelog") || url.contains("/logs")) {
            newTab = 4;
        }

        if (newTab != currentTab) {
            selectTab(newTab, false);
        }
    }

    private GradientDrawable createProgressDrawable() {
        GradientDrawable gd = new GradientDrawable();
        gd.setColor(COLOR_CRIMSON);
        return gd;
    }

    private void buildSplashOverlay() {
        splashOverlay = new LinearLayout(this);
        splashOverlay.setOrientation(LinearLayout.VERTICAL);
        splashOverlay.setGravity(Gravity.CENTER);
        splashOverlay.setBackgroundColor(COLOR_BG);
        splashOverlay.setLayoutParams(new FrameLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT
        ));

        ImageView logoView = new ImageView(this);
        int iconRes = getResources().getIdentifier("icon", "drawable", getPackageName());
        if (iconRes != 0) {
            logoView.setImageResource(iconRes);
        }
        LinearLayout.LayoutParams logoParams = new LinearLayout.LayoutParams(dp(88), dp(88));
        logoParams.setMargins(0, 0, 0, dp(20));
        logoView.setLayoutParams(logoParams);
        splashOverlay.addView(logoView);

        TextView tvTitle = new TextView(this);
        tvTitle.setText("SiPenDosa");
        tvTitle.setTextColor(Color.WHITE);
        tvTitle.setTextSize(TypedValue.COMPLEX_UNIT_SP, 26);
        tvTitle.setTypeface(Typeface.SANS_SERIF, Typeface.BOLD);
        tvTitle.setGravity(Gravity.CENTER);
        splashOverlay.addView(tvTitle);

        TextView tvSub = new TextView(this);
        tvSub.setText("Academic Assistant • v1.1.1");
        tvSub.setTextColor(COLOR_GOLD);
        tvSub.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        tvSub.setTypeface(Typeface.MONOSPACE, Typeface.BOLD);
        tvSub.setGravity(Gravity.CENTER);
        tvSub.setPadding(0, dp(4), 0, dp(24));
        splashOverlay.addView(tvSub);

        ProgressBar splashProgress = new ProgressBar(this);
        LinearLayout.LayoutParams progParams = new LinearLayout.LayoutParams(dp(36), dp(36));
        progParams.setMargins(0, 0, 0, dp(16));
        splashProgress.setLayoutParams(progParams);
        splashOverlay.addView(splashProgress);

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
        v.performHapticFeedback(HapticFeedbackConstants.VIRTUAL_KEY);
        if (v == btnLan) {
            showLanDialog();
        } else if (v == btnWaPair) {
            showWaPairDialog();
        } else if (v == btnRefresh) {
            if (webView != null) {
                webView.reload();
            }
        } else if (v.getTag() instanceof Integer) {
            int tabIndex = ((Integer) v.getTag()).intValue();
            if (tabIndex >= 0 && tabIndex < TAB_ROUTES.length) {
                selectTab(tabIndex, true);
                loadRoute(TAB_ROUTES[tabIndex]);
            }
        }
    }

    private void showLanDialog() {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("⚡ Akses Jaringan Lokal (Wi-Fi / LAN)");
        builder.setMessage("SiPenDosa berjalan sebagai server mandiri di Android:\n\n" +
                "• Alamat Lokal : " + SERVER_URL + "\n" +
                "• Alamat Jaringan : http://" + lanIp + ":8473\n\n" +
                "Perangkat laptop/PC di satu Wi-Fi dapat langsung membuka alamat jaringan di atas.");

        builder.setPositiveButton("Salin URL", new LanDialogClickListener(this, LanDialogClickListener.ACTION_LAN_COPY));
        builder.setNeutralButton("Buka Browser", new LanDialogClickListener(this, LanDialogClickListener.ACTION_LAN_BROWSER));
        builder.setNegativeButton("Tutup", null);
        builder.show();
    }

    private void showWaPairDialog() {
        String[] options = {"📷 Buka Scanner QR Code", "🔢 Tautkan Kode Nomor Telepon", "🔄 Cek Status Koneksi"};
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("📱 WhatsApp Connection Manager");
        builder.setItems(options, new LanDialogClickListener(this, LanDialogClickListener.ACTION_WA_MENU));
        builder.setNegativeButton("Tutup", null);
        builder.show();
    }

    public void handleDialogAction(int action, int which) {
        if (action == LanDialogClickListener.ACTION_LAN_COPY) {
            ClipboardManager clipboard = (ClipboardManager) getSystemService(Context.CLIPBOARD_SERVICE);
            ClipData clip = ClipData.newPlainText("SiPenDosa", "http://" + lanIp + ":8473");
            if (clipboard != null) {
                clipboard.setPrimaryClip(clip);
                Toast.makeText(this, "URL disalin ke clipboard!", Toast.LENGTH_SHORT).show();
            }
        } else if (action == LanDialogClickListener.ACTION_LAN_BROWSER) {
            Intent browserIntent = new Intent(Intent.ACTION_VIEW, Uri.parse(SERVER_URL));
            startActivity(browserIntent);
        } else if (action == LanDialogClickListener.ACTION_WA_MENU) {
            if (which == 0) {
                webView.loadUrl(SERVER_URL + "/");
                Toast.makeText(this, "Silakan klik tombol Scan QR di layar", Toast.LENGTH_SHORT).show();
            } else if (which == 1) {
                promptPhoneNumberPairing();
            } else if (which == 2) {
                webView.loadUrl(SERVER_URL + "/");
            }
        } else if (action == LanDialogClickListener.ACTION_PROMPT_PAIR) {
            if (phoneInput != null) {
                String phone = phoneInput.getText().toString().trim();
                if (phone.isEmpty()) {
                    Toast.makeText(this, "Nomor tidak boleh kosong", Toast.LENGTH_SHORT).show();
                    return;
                }
                requestPairingCode(phone);
            }
        } else if (action == LanDialogClickListener.ACTION_COPY_PAIR_CODE) {
            ClipboardManager clipboard = (ClipboardManager) getSystemService(Context.CLIPBOARD_SERVICE);
            ClipData clip = ClipData.newPlainText("Pairing Code", lastPairCode);
            if (clipboard != null) {
                clipboard.setPrimaryClip(clip);
                Toast.makeText(this, "Kode berhasil disalin!", Toast.LENGTH_SHORT).show();
            }
        }
    }

    private void promptPhoneNumberPairing() {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("🔢 Tautkan via Nomor Telepon");
        builder.setMessage("Masukkan nomor WhatsApp yang akan digunakan (contoh: 6281234567890):");

        phoneInput = new EditText(this);
        phoneInput.setInputType(InputType.TYPE_CLASS_PHONE);
        phoneInput.setHint("628xxxxxxxxxx");
        phoneInput.setTextColor(Color.WHITE);
        phoneInput.setHintTextColor(Color.GRAY);
        phoneInput.setPadding(dp(16), dp(12), dp(16), dp(12));
        builder.setView(phoneInput);

        builder.setPositiveButton("Minta Kode", new LanDialogClickListener(this, LanDialogClickListener.ACTION_PROMPT_PAIR));
        builder.setNegativeButton("Batal", null);
        builder.show();
    }

    private void requestPairingCode(String phone) {
        pairProgressDialog = new ProgressDialog(this);
        pairProgressDialog.setMessage("Meminta 8-digit kode pairing WhatsApp...");
        pairProgressDialog.show();

        new Thread(ActionTask.doPairCode(this, phone)).start();
    }

    public void requestPairingCodeInternal(String phone) {
        String pairCode = null;
        try {
            URL url = new URL(SERVER_URL + "/api/whatsapp/pair-phone");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setDoOutput(true);
            conn.setRequestProperty("Content-Type", "application/x-www-form-urlencoded");
            String postData = "phone=" + Uri.encode(phone);
            OutputStream os = conn.getOutputStream();
            os.write(postData.getBytes());
            os.flush();
            os.close();

            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream()));
            StringBuilder sb = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) {
                sb.append(line);
            }
            reader.close();
            pairCode = sb.toString();
        } catch (Exception e) {
            pairCode = "ERR: " + e.getMessage();
        }

        handler.post(ActionTask.pairCodeResult(this, pairCode));
    }

    public void onPairCodeResult(String result) {
        if (pairProgressDialog != null && pairProgressDialog.isShowing()) {
            pairProgressDialog.dismiss();
        }

        if (result != null && result.startsWith("ERR:")) {
            Toast.makeText(this, "Gagal meminta kode: " + result.substring(4), Toast.LENGTH_LONG).show();
            return;
        }

        lastPairCode = (result != null) ? result : "";
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("🔑 Kode Pairing WhatsApp Anda");
        builder.setMessage("Buka WhatsApp di ponsel Anda -> Perangkat Tertaut -> Tautkan dengan nomor telepon -> Masukkan kode berikut:\n\n" +
                "👉  " + lastPairCode + "  👈\n\n" +
                "Kode ini berlaku selama beberapa menit.");

        builder.setPositiveButton("Salin Kode", new LanDialogClickListener(this, LanDialogClickListener.ACTION_COPY_PAIR_CODE));
        builder.setNegativeButton("Tutup", null);
        builder.show();
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
            if (daemonStatusDot != null) {
                GradientDrawable dotBg = new GradientDrawable();
                dotBg.setShape(GradientDrawable.OVAL);
                dotBg.setColor(COLOR_EMERALD);
                daemonStatusDot.setBackground(dotBg);
            }
            if (tvDaemonStatus != null) {
                tvDaemonStatus.setText("ONLINE");
                tvDaemonStatus.setTextColor(COLOR_EMERALD);
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
    public void startApkDownloadAndInstall(String apkUrl, String versionName) {
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
            return;
        }

        if (System.currentTimeMillis() - backPressedTime < 2000) {
            super.onBackPressed();
        } else {
            backPressedTime = System.currentTimeMillis();
            Toast.makeText(this, "Tekan sekali lagi untuk keluar", Toast.LENGTH_SHORT).show();
        }
    }
}
