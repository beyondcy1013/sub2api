package com.sub2api.app;

import android.app.Activity;
import android.content.Context;
import android.content.SharedPreferences;

final class AppPreferences {
    static final String FILE = "app_settings";
    static final String KEY_THEME_MODE = "theme.mode";
    static final String KEY_AUTO_UPDATE = "update.auto_check";
    static final String KEY_ACTIVE_SOURCE = "data_source.active_index";
    static final String KEY_ACTIVE_SOURCE_AT = "data_source.active_at";

    static final String THEME_SYSTEM = "system";
    static final String THEME_LIGHT = "light";
    static final String THEME_DARK = "dark";

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

    static boolean autoUpdate(Context context) {
        return get(context).getBoolean(KEY_AUTO_UPDATE, true);
    }

}
