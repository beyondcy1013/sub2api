import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const root = new URL("../..", import.meta.url);
const read = (relative) => readFileSync(new URL(relative, root), "utf8");

const activity = read("android/app/src/main/java/com/sub2api/app/MainActivity.java");
const sourceRegistry = read("android/app/src/main/java/com/sub2api/app/SourceRegistry.java");
const appPreferences = read("android/app/src/main/java/com/sub2api/app/AppPreferences.java");
const settingsActivity = read("android/app/src/main/java/com/sub2api/app/SettingsActivity.java");
const updateManager = read("android/app/src/main/java/com/sub2api/app/UpdateManager.java");
const manifest = read("android/app/src/main/AndroidManifest.xml");
const networkPolicy = read("android/app/src/main/res/xml/network_security_config.xml");
const lightTheme = read("android/app/src/main/res/values/themes.xml");
const nightTheme = read("android/app/src/main/res/values-night/themes.xml");
const gradle = read("android/app/build.gradle.kts");
const gradleProperties = read("android/gradle.properties");
const buildScript = read("scripts/build-sub2api-android-apk.sh");
const versionFile = read("backend/cmd/server/VERSION").trim();

test("Sub2API Android client races LAN and public Sub2API sources", () => {
  for (const source of [
    "http://192.168.3.2:18381/",
    "http://fpsq.xyz:18381/",
  ]) {
    assert.match(sourceRegistry, new RegExp(`"${source.replaceAll(".", "\\.")}"`));
  }
  for (const source of [
    "http://192.168.3.2:11111/",
    "http://fpsq.xyz:11112/",
  ]) {
    assert.match(sourceRegistry, new RegExp(`"${source.replaceAll(".", "\\.")}"`));
  }
  assert.match(sourceRegistry, /PROBE_PATH = "health"/);
  assert.match(sourceRegistry, /UPDATE_MANIFEST_PATH = "api\/artifacts\/update\/android\/sub2api"/);
  assert.match(updateManager, /SourceRegistry\.UPDATE_URLS\.length/);
  assert.match(updateManager, /String base = SourceRegistry\.UPDATE_URLS\[source\]/);
  assert.match(activity, /Executors\.newFixedThreadPool\(SourceRegistry\.URLS\.length\)/);
  assert.match(activity, /CompletionService<Integer>/);
  assert.match(activity, /new ExecutorCompletionService<>\(probeExecutor\)/);
  assert.match(activity, /completion\.take\(\)\.get\(\)/);
  assert.match(activity, /probe\.cancel\(true\)/);
  assert.match(activity, /AppPreferences\.preferredSource\(this\)/);
  assert.match(activity, /AppPreferences\.recentActiveSource\(this\)/);
  assert.match(appPreferences, /LOGIN_ORIGIN_RETENTION_MS = 30L \* 24 \* 60 \* 60 \* 1000/);
  assert.match(appPreferences, /age > LOGIN_ORIGIN_RETENTION_MS/);
  assert.match(activity, /SourceRegistry\.probe\(preferred\)/);
  assert.match(activity, /retryAlternateSource\(\)/);
  assert.match(activity, /rejectedSources\[excludedSource\] = true/);
  assert.match(activity, /onPageFinished[\s\S]*isSelectedSourceUrl\(url\)/);
});

test("Sub2API WebView keeps navigation and cleartext access on known sources", () => {
  assert.match(activity, /isTrustedOrigin\(uri\)/);
  assert.match(activity, /shouldOverrideUrlLoading/);
  assert.match(activity, /Intent\.ACTION_VIEW/);
  assert.match(networkPolicy, /<base-config cleartextTrafficPermitted="false"/);
  assert.match(networkPolicy, />192\.168\.3\.2</);
  assert.match(networkPolicy, />fpsq\.xyz</);
  assert.doesNotMatch(manifest, /android:usesCleartextTraffic="true"/);
  assert.match(manifest, /android\.permission\.INTERNET/);
  assert.match(manifest, /android:networkSecurityConfig="@xml\/network_security_config"/);
  assert.match(activity, /settings\.setDomStorageEnabled\(true\)/);
  assert.match(activity, /CookieManager\.getInstance\(\)\.flush\(\)/);
});

test("Sub2API Android follows system light and dark mode", () => {
  assert.match(lightTheme, /parent="android:style\/Theme\.Material\.Light\.NoActionBar"/);
  assert.match(nightTheme, /parent="android:style\/Theme\.Material\.NoActionBar"/);
  assert.match(nightTheme, /<item name="android:windowLightStatusBar">false<\/item>/);
  assert.doesNotMatch(manifest, /android:configChanges="[^"]*uiMode/);
  assert.match(appPreferences, /THEME_SYSTEM = "system"/);
});

test("Sub2API Android settings expose theme, data-source, and update tabs", () => {
  assert.match(settingsActivity, /TAB_KEYS = \{"theme", "data-source", "update"\}/);
  assert.match(settingsActivity, /testSources\(test, statuses\)/);
  assert.match(settingsActivity, /SourceRegistry\.LABELS/);
  assert.match(settingsActivity, /UpdateManager\.check/);
  assert.match(manifest, /android:name="\.SettingsActivity"/);
});

test("Sub2API Android defaults to virtual-mouse touch mode and permits pinch zoom", () => {
  assert.match(appPreferences, /TOUCH_VIRTUAL_MOUSE = "virtual_mouse"/);
  assert.match(appPreferences, /TOUCH_DIRECT = "direct"/);
  assert.match(appPreferences, /KEY_TOUCH_MODE = "touch.mode"/);
  assert.match(appPreferences, /getString\(KEY_TOUCH_MODE, TOUCH_VIRTUAL_MOUSE\)/);
  assert.match(activity, /toggleTouchMode\(\)/);
  assert.match(activity, /applyTouchMode\(\)/);
  assert.match(activity, /setSupportZoom\(true\)/);
  assert.match(activity, /setBuiltInZoomControls\(true\)/);
  assert.match(activity, /setDisplayZoomControls\(false\)/);
  assert.match(activity, /window\.__sub2apiTouch/);
  assert.match(activity, /event\.touches\.length !== 1/);
  assert.match(activity, /new PointerEvent/);
  assert.match(activity, /user-scalable=yes/);

  const strings = read("android/app/src/main/res/values/strings.xml");
  assert.match(strings, /触控: 鼠标/);
  assert.match(strings, /触控: 直点/);
});

test("Sub2API Android automatic updates validate manifest, bytes, package, version, and signer", () => {
  assert.match(gradleProperties, /^android\.useAndroidX=true$/m);
  assert.match(updateManager, /Pattern\.compile\("\[0-9a-f\]\{64\}"\)/);
  assert.match(updateManager, /downloadUrl\.startsWith\("\/api\/artifacts\/download\/"\)/);
  assert.match(updateManager, /EXPECTED_SHA256/);
  assert.match(updateManager, /EXPECTED_SIZE/);
  assert.match(updateManager, /EXPECTED_VERSION_CODE/);
  assert.match(activity, /MessageDigest\.getInstance\("SHA-256"\)/);
  assert.match(activity, /copied != expectedSize/);
  assert.match(activity, /getPackageArchiveInfo/);
  assert.match(activity, /getPackageName\(\)\.equals\(archive\.packageName\)/);
  assert.match(activity, /signatureDigests\(archive\)\.equals\(signatureDigests\(installed\)\)/);
  assert.match(activity, /FileProvider\.getUriForFile/);
  assert.match(manifest, /android:name="androidx\.core\.content\.FileProvider"/);
  assert.match(manifest, /android:authorities="\$\{applicationId\}\.updateprovider"/);
  assert.match(activity, /showUpdateProgress\(downloadId\)/);
});

test("Sub2API Android release identity comes from backend version", () => {
  assert.match(gradle, /applicationId = "com\.sub2api\.app"/);
  assert.match(gradle, /providers\.gradleProperty\("sub2apiVersion"\)/);
  assert.match(buildScript, /backend\/scripts\/resolve-version\.sh/);
  assert.match(buildScript, /apksigner" verify --verbose --print-certs/);
  assert.match(buildScript, /sub2api-\$\{client_version\}\.apk/);
  assert.match(versionFile, /^[0-9]+\.[0-9]+\.[0-9]+$/);
});

test("Sub2API Android re-tests data sources in the background and switches via countdown toast", () => {
  // 连接阶段不再使用加载进度条或全屏连接面板
  assert.doesNotMatch(activity, /connectionProgress/);
  assert.doesNotMatch(activity, /showConnectingState/);
  // 只有销毁 WebView 时才允许 stopLoading；解析/重测期间不得打断当前页面
  assert.equal((activity.match(/webView\.stopLoading\(\)/g) || []).length, 1);
  assert.match(activity, /beginSourceReevaluation\(\)/);
  assert.match(activity, /raceHealthySources\(\)/);
  assert.match(activity, /switchSourceWithCountdown\(/);
  assert.match(activity, /switchCountdownActive/);
  assert.match(activity, /mainHandler\.postDelayed\(this, 1000\)/);

  const strings = read("android/app/src/main/res/values/strings.xml");
  assert.match(strings, /switching_server_countdown/);
  assert.match(strings, /%1\$d 秒后切换到服务器：%2\$s/);
});
