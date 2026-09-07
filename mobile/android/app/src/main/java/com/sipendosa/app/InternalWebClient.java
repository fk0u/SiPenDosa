package com.sipendosa.app;

import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.webkit.WebView;
import android.webkit.WebViewClient;

public class InternalWebClient extends WebViewClient {
    private static final String SERVER_URL = "http://127.0.0.1:8473";

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
}
