package com.sipendosa.app;

import android.content.DialogInterface;
import android.webkit.JsResult;

public class JsDialogClickListener implements DialogInterface.OnClickListener, DialogInterface.OnCancelListener {
    private final JsResult result;
    private final boolean isConfirm;

    public JsDialogClickListener(JsResult result, boolean isConfirm) {
        this.result = result;
        this.isConfirm = isConfirm;
    }

    @Override
    public void onClick(DialogInterface dialog, int which) {
        if (isConfirm) {
            result.confirm();
        } else {
            result.cancel();
        }
    }

    @Override
    public void onCancel(DialogInterface dialog) {
        result.cancel();
    }
}
