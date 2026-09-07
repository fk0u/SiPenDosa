package com.sipendosa.app;

import android.content.DialogInterface;

public class LanDialogClickListener implements DialogInterface.OnClickListener {
    private final MainActivity activity;
    private final int action;

    public LanDialogClickListener(MainActivity activity, int action) {
        this.activity = activity;
        this.action = action;
    }

    @Override
    public void onClick(DialogInterface dialog, int which) {
        activity.handleLanDialogClick(action);
    }
}
