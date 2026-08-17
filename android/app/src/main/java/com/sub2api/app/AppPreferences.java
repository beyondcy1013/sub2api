package com.sub2api.app;

import android.app.Activity;
import android.content.Context;
import android.content.SharedPreferences;

final class AppPreferences {
    static final String FILE = "app_settings";
    static final String KEY_THEME_MODE = "theme.mode";
    static final String KEY_SOURCE_MODE = "data_source.mode";
    static final String KEY_AUTO_UPDATE = "update.auto_check";
    static final String KEY_TOUCH_MODE = "touch.mode";
    static final String KEY_ACTIVE_SOURCE = "data_source.active_index";
    static final String KEY_ACTIVE_SOURCE_AT = "data_source.active_at";

    static final long LOGIN_ORIGIN_RETENTION_MS = 30L * 24 * 60 * 60 * 1000;

    static final String THEME_SYSTEM = "system";
    static final String THEME_LIGHT = "light";
    static final String THEME_DARK = "dark";
    static final String SOURCE_AUTO = "auto";
    static final String TOUCH_VIRTUAL_MOUSE = "virtual_mouse";
    static final String TOUCH_DIRECT = "direct";

    private AppPreferences() {}

    static SharedPreferences get(Context context) {
        return context.getSharedPreferences(FILE, Context.MODE_PRIVATE);
    }

    static void applyTheme(Activity activity) {
        switch (themeMode(activity)) {
            case THEME_LIGHT:
                activity.setTheme(R.style.Theme_Sub2Api_Light);
                break;
            case THEME_DARK:
                activity.setTheme(R.style.Theme_Sub2Api_Dark);
                break;
            default:
                activity.setTheme(R.style.Theme_Sub2Api);
                break;
        }
    }

    static String themeMode(Context context) {
        String value = get(context).getString(KEY_THEME_MODE, THEME_SYSTEM);
        if (THEME_LIGHT.equals(value) || THEME_DARK.equals(value)) {
            return value;
        }
        return THEME_SYSTEM;
    }

    static String sourceMode(Context context) {
        String value = get(context).getString(KEY_SOURCE_MODE, SOURCE_AUTO);
        if (SOURCE_AUTO.equals(value)) {
            return value;
        }
        try {
            int index = Integer.parseInt(value);
            return SourceRegistry.isValidIndex(index) ? value : SOURCE_AUTO;
        } catch (NumberFormatException ignored) {
            return SOURCE_AUTO;
        }
    }

    static int preferredSource(Context context) {
        String value = sourceMode(context);
        return SOURCE_AUTO.equals(value) ? -1 : Integer.parseInt(value);
    }

    static int recentActiveSource(Context context) {
        SharedPreferences preferences = get(context);
        int source = preferences.getInt(KEY_ACTIVE_SOURCE, -1);
        long activeAt = preferences.getLong(KEY_ACTIVE_SOURCE_AT, 0);
        long age = System.currentTimeMillis() - activeAt;
        if (!SourceRegistry.isValidIndex(source) || age < 0 || age > LOGIN_ORIGIN_RETENTION_MS) {
            return -1;
        }
        return source;
    }

    static boolean autoUpdate(Context context) {
        return get(context).getBoolean(KEY_AUTO_UPDATE, true);
    }

    static String touchMode(Context context) {
        String value = get(context).getString(KEY_TOUCH_MODE, TOUCH_VIRTUAL_MOUSE);
        return TOUCH_DIRECT.equals(value) ? value : TOUCH_VIRTUAL_MOUSE;
    }

    static boolean virtualMouseEnabled(Context context) {
        return TOUCH_VIRTUAL_MOUSE.equals(touchMode(context));
    }
}
