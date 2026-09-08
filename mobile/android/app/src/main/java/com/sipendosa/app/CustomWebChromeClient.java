package com.sipendosa.app;

import android.app.Activity;
import android.app.AlertDialog;
import android.util.Log;
import android.webkit.ConsoleMessage;
import android.webkit.JsResult;
import android.webkit.WebChromeClient;
import android.webkit.WebView;

public class CustomWebChromeClient extends WebChromeClient {
    private final Activity activity;

    public CustomWebChromeClient(Activity activity) {
        this.activity = activity;
    }

    @Override
    public boolean onConsoleMessage(ConsoleMessage consoleMessage) {
        Log.d("SiPenDosaWeb", "[" + consoleMessage.messageLevel() + "] " +
                consoleMessage.message() + " (" + consoleMessage.sourceId() + ":" + consoleMessage.lineNumber() + ")");
        return true;
    }

    @Override
    public boolean onJsAlert(WebView view, String url, String message, final JsResult result) {
        JsDialogClickListener listener = new JsDialogClickListener(result, true);
        new AlertDialog.Builder(activity)
                .setTitle("SiPenDosa")
                .setMessage(message)
                .setPositiveButton(android.R.string.ok, listener)
                .setOnCancelListener(listener)
                .create()
                .show();
        return true;
    }

    @Override
    public boolean onJsConfirm(WebView view, String url, String message, final JsResult result) {
        new AlertDialog.Builder(activity)
                .setTitle("Konfirmasi")
                .setMessage(message)
                .setPositiveButton(android.R.string.ok, new JsDialogClickListener(result, true))
                .setNegativeButton(android.R.string.cancel, new JsDialogClickListener(result, false))
                .setOnCancelListener(new JsDialogClickListener(result, false))
                .create()
                .show();
        return true;
    }
}
