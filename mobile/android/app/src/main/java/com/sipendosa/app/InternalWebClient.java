package com.sipendosa.app;

import android.content.Context;
import android.content.Intent;
import android.graphics.Bitmap;
import android.net.Uri;
import android.webkit.WebView;
import android.webkit.WebViewClient;

public class InternalWebClient extends WebViewClient {
    private static final String SERVER_URL = "http://127.0.0.1:8473";
    private final MainActivity activity;

    public InternalWebClient(MainActivity activity) {
        this.activity = activity;
    }

    @Override
    public boolean shouldOverrideUrlLoading(WebView view, String url) {
        if (url.startsWith(SERVER_URL) || url.startsWith("http://localhost:8473")) {
            view.loadUrl(url);
            return true;
        }
        Context ctx = view.getContext();
        Intent intent = new Intent(Intent.ACTION_VIEW, Uri.parse(url));
        ctx.startActivity(intent);
        return true;
    }

    @Override
    public void onPageStarted(WebView view, String url, Bitmap favicon) {
        super.onPageStarted(view, url, favicon);
        if (activity != null) {
            activity.onPageStarted(url);
        }
    }

    @Override
    public void onPageFinished(WebView view, String url) {
        super.onPageFinished(view, url);
        if (activity != null) {
            activity.onPageNavigated(url);
        }
    }
}
