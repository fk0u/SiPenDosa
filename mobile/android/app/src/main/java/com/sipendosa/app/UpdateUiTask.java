package com.sipendosa.app;

public class UpdateUiTask implements Runnable {
    private final MainActivity activity;
    private final boolean isReady;

    public UpdateUiTask(MainActivity activity, boolean isReady) {
        this.activity = activity;
        this.isReady = isReady;
    }

    @Override
    public void run() {
        activity.onServerStatusChecked(isReady);
    }
}
