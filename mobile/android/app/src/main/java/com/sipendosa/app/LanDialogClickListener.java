package com.sipendosa.app;

import android.content.DialogInterface;

public class LanDialogClickListener implements DialogInterface.OnClickListener {
    public static final int ACTION_LAN_COPY = 1;
    public static final int ACTION_LAN_BROWSER = 2;
    public static final int ACTION_WA_MENU = 3;
    public static final int ACTION_PROMPT_PAIR = 4;
    public static final int ACTION_COPY_PAIR_CODE = 5;

    private final MainActivity activity;
    private final int action;

    public LanDialogClickListener(MainActivity activity, int action) {
        this.activity = activity;
        this.action = action;
    }

    @Override
    public void onClick(DialogInterface dialog, int which) {
        activity.handleDialogAction(action, which);
    }
}
