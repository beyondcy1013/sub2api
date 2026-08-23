package com.sub2api.app;

import android.app.Activity;
import android.content.SharedPreferences;
import android.graphics.Color;
import android.os.Bundle;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.HorizontalScrollView;
import android.widget.LinearLayout;
import android.widget.RadioButton;
import android.widget.RadioGroup;
import android.widget.ScrollView;
import android.widget.Switch;
import android.widget.TextView;
import android.widget.Toast;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public final class SettingsActivity extends Activity {
    private static final String[] TAB_KEYS = {"theme", "data-source", "update"};
    private static final int[] TAB_LABELS = {
        R.string.settings_tab_theme,
        R.string.settings_tab_data_source,
        R.string.settings_tab_update,
    };

    private final ExecutorService probeExecutor = Executors.newFixedThreadPool(
        SourceRegistry.URLS.length
    );
    private LinearLayout tabRow;
    private FrameLayout content;
    private String activeTab = TAB_KEYS[0];

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        AppPreferences.applyTheme(this);
        super.onCreate(savedInstanceState);
        setContentView(createScreen());
        showTab(activeTab);
    }

    private View createScreen() {
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setBackgroundColor(color(android.R.attr.windowBackground, Color.WHITE));

        LinearLayout appBar = new LinearLayout(this);
        appBar.setGravity(Gravity.CENTER_VERTICAL);
        appBar.setPadding(dp(4), 0, dp(12), 0);
        Button back = new Button(this);
        back.setText("‹");
        back.setTextSize(28);
        back.setContentDescription(getString(R.string.back));
        back.setBackgroundColor(Color.TRANSPARENT);
        back.setOnClickListener(view -> finish());
        appBar.addView(back, new LinearLayout.LayoutParams(dp(48), dp(48)));
        TextView title = text(getString(R.string.settings), 20, true);
        appBar.addView(title, new LinearLayout.LayoutParams(0, dp(48), 1));
        root.addView(appBar);

        HorizontalScrollView tabScroll = new HorizontalScrollView(this);
        tabScroll.setHorizontalScrollBarEnabled(false);
        tabRow = new LinearLayout(this);
        tabRow.setOrientation(LinearLayout.HORIZONTAL);
        for (int index = 0; index < TAB_KEYS.length; index++) {
            String key = TAB_KEYS[index];
            Button button = new Button(this);
            button.setTag(key);
            button.setText(TAB_LABELS[index]);
            button.setAllCaps(false);
            button.setMinWidth(dp(88));
            button.setOnClickListener(view -> showTab((String) view.getTag()));
            tabRow.addView(button, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                dp(44)
            ));
        }
        tabScroll.addView(tabRow);
        root.addView(tabScroll, new LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            dp(44)
        ));

        content = new FrameLayout(this);
        root.addView(content, new LinearLayout.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            0,
            1
        ));
        return root;
    }

    private void showTab(String key) {
        activeTab = key;
        for (int index = 0; index < tabRow.getChildCount(); index++) {
            View child = tabRow.getChildAt(index);
            child.setSelected(key.equals(child.getTag()));
            child.setAlpha(child.isSelected() ? 1f : 0.62f);
        }
        content.removeAllViews();
        View panel;
        switch (key) {
            case "data-source":
                panel = dataSourcePanel();
                break;
            case "update":
                panel = updatePanel();
                break;
            default:
                panel = themePanel();
                break;
        }
        content.addView(panel);
    }

    private View themePanel() {
        LinearLayout panel = panel();
        addHeading(panel, R.string.settings_theme_mode);
        RadioGroup group = new RadioGroup(this);
        String[] values = {
            AppPreferences.THEME_SYSTEM,
            AppPreferences.THEME_LIGHT,
            AppPreferences.THEME_DARK,
        };
        int[] labels = {R.string.theme_system, R.string.theme_light, R.string.theme_dark};
        String current = AppPreferences.themeMode(this);
        for (int index = 0; index < values.length; index++) {
            RadioButton option = new RadioButton(this);
            option.setId(View.generateViewId());
            option.setTag(values[index]);
            option.setText(labels[index]);
            option.setChecked(values[index].equals(current));
            group.addView(option, rowParams());
        }
        group.setOnCheckedChangeListener((view, checkedId) -> {
            View selected = view.findViewById(checkedId);
            if (selected == null) {
                return;
            }
            AppPreferences.get(this).edit()
                .putString(AppPreferences.KEY_THEME_MODE, (String) selected.getTag())
                .apply();
            recreate();
        });
        panel.addView(group);
        return scroll(panel);
    }

    private View dataSourcePanel() {
        LinearLayout panel = panel();
        addHeading(panel, R.string.settings_source_strategy);
        panel.addView(text(getString(R.string.source_auto), 14, false), rowParams());

        addHeading(panel, R.string.settings_source_status);
        int active = AppPreferences.get(this).getInt(AppPreferences.KEY_ACTIVE_SOURCE, -1);
        long activeAt = AppPreferences.get(this).getLong(AppPreferences.KEY_ACTIVE_SOURCE_AT, 0);
        TextView activeStatus = text(
            SourceRegistry.isValidIndex(active)
                ? getString(R.string.source_active_value, SourceRegistry.LABELS[active], formatTime(activeAt))
                : getString(R.string.source_active_unknown),
            14,
            false
        );
        panel.addView(activeStatus, rowParams());

        TextView[] statuses = new TextView[SourceRegistry.URLS.length];
        for (int index = 0; index < statuses.length; index++) {
            statuses[index] = text(
                getString(R.string.source_not_tested, SourceRegistry.LABELS[index]),
                14,
                false
            );
            panel.addView(statuses[index], rowParams());
        }
        Button test = command(R.string.source_test_all);
        test.setOnClickListener(view -> testSources(test, statuses));
        panel.addView(test, rowParams());
        return scroll(panel);
    }

    private View updatePanel() {
        LinearLayout panel = panel();
        addHeading(panel, R.string.settings_update_automatic);
        Switch automatic = new Switch(this);
        automatic.setText(R.string.update_check_on_start);
        automatic.setChecked(AppPreferences.autoUpdate(this));
        automatic.setOnCheckedChangeListener((view, checked) ->
            AppPreferences.get(this).edit()
                .putBoolean(AppPreferences.KEY_AUTO_UPDATE, checked)
                .apply()
        );
        panel.addView(automatic, rowParams());

        addHeading(panel, R.string.settings_update_current);
        TextView status = text(
            getString(R.string.update_current_version, UpdateManager.currentVersionName(this)),
            14,
            false
        );
        panel.addView(status, rowParams());
        Button check = command(R.string.update_check_now);
        check.setOnClickListener(view -> checkUpdate(check, status, panel));
        panel.addView(check, rowParams());
        return scroll(panel);
    }

    private void testSources(Button test, TextView[] statuses) {
        test.setEnabled(false);
        test.setText(R.string.source_testing);
        final int[] remaining = {statuses.length};
        for (int index = 0; index < statuses.length; index++) {
            int source = index;
            statuses[index].setText(getString(R.string.source_testing_value, SourceRegistry.LABELS[index]));
            probeExecutor.execute(() -> {
                SourceRegistry.ProbeResult result = SourceRegistry.probe(source);
                runOnUiThread(() -> {
                    statuses[source].setText(result.healthy
                        ? getString(R.string.source_healthy, SourceRegistry.LABELS[source], result.latencyMs)
                        : getString(R.string.source_unavailable, SourceRegistry.LABELS[source]));
                    remaining[0]--;
                    if (remaining[0] == 0) {
                        test.setEnabled(true);
                        test.setText(R.string.source_test_all);
                    }
                });
            });
        }
    }

    private void checkUpdate(Button check, TextView status, LinearLayout panel) {
        check.setEnabled(false);
        status.setText(R.string.update_checking);
        UpdateManager.check(this, activeSource(), new UpdateManager.Callback() {
            @Override
            public void onSuccess(UpdateManager.UpdateManifest manifest) {
                runOnUiThread(() -> {
                    check.setEnabled(true);
                    if (!UpdateManager.isNewer(SettingsActivity.this, manifest)) {
                        status.setText(R.string.update_already_latest);
                        return;
                    }
                    status.setText(getString(R.string.update_available, manifest.version));
                    Button download = command(R.string.update_download_install);
                    download.setOnClickListener(view -> {
                        UpdateManager.enqueue(SettingsActivity.this, manifest);
                        download.setEnabled(false);
                        download.setText(R.string.update_downloading);
                        Toast.makeText(SettingsActivity.this, R.string.update_download_started, Toast.LENGTH_LONG).show();
                    });
                    panel.addView(download, rowParams());
                });
            }

            @Override
            public void onError(String message) {
                runOnUiThread(() -> {
                    check.setEnabled(true);
                    status.setText(getString(R.string.update_check_error, message));
                });
            }
        });
    }

    private int activeSource() {
        int active = AppPreferences.get(this).getInt(AppPreferences.KEY_ACTIVE_SOURCE, -1);
        return SourceRegistry.isValidIndex(active) ? active : -1;
    }

    private LinearLayout panel() {
        LinearLayout panel = new LinearLayout(this);
        panel.setOrientation(LinearLayout.VERTICAL);
        panel.setPadding(dp(16), dp(12), dp(16), dp(24));
        return panel;
    }

    private ScrollView scroll(View child) {
        ScrollView scroll = new ScrollView(this);
        scroll.addView(child);
        return scroll;
    }

    private void addHeading(LinearLayout panel, int label) {
        TextView heading = text(getString(label), 15, true);
        LinearLayout.LayoutParams params = rowParams();
        params.topMargin = dp(10);
        panel.addView(heading, params);
    }

    private TextView text(String value, int size, boolean bold) {
        TextView view = new TextView(this);
        view.setText(value);
        view.setTextSize(size);
        view.setTextColor(color(android.R.attr.textColorPrimary, Color.BLACK));
        view.setGravity(Gravity.CENTER_VERTICAL);
        if (bold) {
            view.setTypeface(view.getTypeface(), android.graphics.Typeface.BOLD);
        }
        return view;
    }

    private Button command(int label) {
        Button button = new Button(this);
        button.setText(label);
        button.setAllCaps(false);
        return button;
    }

    private LinearLayout.LayoutParams rowParams() {
        return new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(48));
    }

    private int color(int attribute, int fallback) {
        android.content.res.TypedArray values = obtainStyledAttributes(new int[]{attribute});
        try {
            return values.getColor(0, fallback);
        } finally {
            values.recycle();
        }
    }

    private String formatTime(long timestamp) {
        if (timestamp <= 0) {
            return getString(R.string.never);
        }
        return android.text.format.DateFormat.getDateFormat(this).format(timestamp)
            + " "
            + android.text.format.DateFormat.getTimeFormat(this).format(timestamp);
    }

    private int dp(int value) {
        return Math.round(value * getResources().getDisplayMetrics().density);
    }

    @Override
    protected void onDestroy() {
        probeExecutor.shutdownNow();
        super.onDestroy();
    }
}
