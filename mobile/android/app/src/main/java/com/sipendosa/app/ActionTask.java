package com.sipendosa.app;

import java.io.File;

public class ActionTask implements Runnable {
    public static final int ACTION_PROGRESS = 1;
    public static final int ACTION_SUCCESS = 2;
    public static final int ACTION_FAILED = 3;
    public static final int ACTION_START_DOWNLOAD = 4;
    public static final int ACTION_DO_DOWNLOAD = 5;
    public static final int ACTION_DO_PAIR_CODE = 6;
    public static final int ACTION_PAIR_CODE_RESULT = 7;

    private final MainActivity activity;
    private final int action;
    private int progress;
    private String text;
    private File file;

    public ActionTask(MainActivity activity, int action) {
        this.activity = activity;
        this.action = action;
    }

    public static ActionTask progress(MainActivity act, int progress) {
        ActionTask t = new ActionTask(act, ACTION_PROGRESS);
        t.progress = progress;
        return t;
    }

    public static ActionTask success(MainActivity act, File file) {
        ActionTask t = new ActionTask(act, ACTION_SUCCESS);
        t.file = file;
        return t;
    }

    public static ActionTask failed(MainActivity act, String error) {
        ActionTask t = new ActionTask(act, ACTION_FAILED);
        t.text = error;
        return t;
    }

    public static ActionTask startDownload(MainActivity act, String url, String ver) {
        ActionTask t = new ActionTask(act, ACTION_START_DOWNLOAD);
        t.text = url;
        return t;
    }

    public static ActionTask doDownload(MainActivity act, String url) {
        ActionTask t = new ActionTask(act, ACTION_DO_DOWNLOAD);
        t.text = url;
        return t;
    }

    public static ActionTask doPairCode(MainActivity act, String phone) {
        ActionTask t = new ActionTask(act, ACTION_DO_PAIR_CODE);
        t.text = phone;
        return t;
    }

    public static ActionTask pairCodeResult(MainActivity act, String result) {
        ActionTask t = new ActionTask(act, ACTION_PAIR_CODE_RESULT);
        t.text = result;
        return t;
    }

    @Override
    public void run() {
        switch (action) {
            case ACTION_PROGRESS:
                activity.onDownloadProgress(progress);
                break;
            case ACTION_SUCCESS:
                activity.onDownloadSuccess(file);
                break;
            case ACTION_FAILED:
                activity.onDownloadError(text);
                break;
            case ACTION_START_DOWNLOAD:
                activity.startApkDownloadAndInstall(text, "");
                break;
            case ACTION_DO_DOWNLOAD:
                activity.downloadApkInternal(text);
                break;
            case ACTION_DO_PAIR_CODE:
                activity.requestPairingCodeInternal(text);
                break;
            case ACTION_PAIR_CODE_RESULT:
                activity.onPairCodeResult(text);
                break;
        }
    }
}
