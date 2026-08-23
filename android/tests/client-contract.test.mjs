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
const strings = read("android/app/src/main/res/values/strings.xml");
const versionFile = read("android/VERSION").trim();

test("Sub2API Android cold start races every configured source without a preferred-source wait", () => {
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
  assert.match(updateManager, /Executors\.newFixedThreadPool\(SourceRegistry\.UPDATE_URLS\.length\)/);
  assert.match(updateManager, /CompletionService<UpdateManifest>/);
  assert.match(updateManager, /completion\.submit\(\(\) -> fetchManifest\(source\)\)/);
  assert.match(updateManager, /completion\.take\(\)\.get\(\)/);
  assert.match(updateManager, /cancelChecks\(checks\)/);
  assert.match(updateManager, /String base = SourceRegistry\.UPDATE_URLS\[source\]/);
  assert.match(activity, /Executors\.newFixedThreadPool\(SourceRegistry\.URLS\.length\)/);
  assert.match(activity, /CompletionService<Integer>/);
  assert.match(activity, /new ExecutorCompletionService<>\(probeExecutor\)/);
  assert.match(activity, /completion\.take\(\)\.get\(\)/);
  assert.match(activity, /probe\.cancel\(true\)/);
  assert.match(activity, /beginSourceResolution\(\)[\s\S]*resolveAndLoad\(-1, true\)/);
  assert.doesNotMatch(activity, /SourceRegistry\.probe\(preferred\)/);
  assert.doesNotMatch(appPreferences, /preferredSource|recentActiveSource|KEY_SOURCE_MODE/);
  assert.match(activity, /result\.healthy \? source : -\(source \+ 1\)/);
  assert.match(activity, /completion\.take\(\)\.get\(\)/);
  assert.match(activity, /if \(result >= 0\)[\s\S]*cancelProbes\(probes\)/);
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
  assert.doesNotMatch(settingsActivity, /putString\(AppPreferences\.KEY_SOURCE_MODE/);
  assert.match(settingsActivity, /UpdateManager\.check\(this, activeSource\(\),/);
  assert.match(manifest, /android:name="\.SettingsActivity"/);
});

test("Sub2API Android uses native WebView touch handling and permits pinch zoom", () => {
  assert.doesNotMatch(appPreferences, /TOUCH_VIRTUAL_MOUSE|KEY_TOUCH_MODE/);
  assert.doesNotMatch(activity, /touchModeButton|toggleTouchMode|TOUCH_CONTROL_BOOTSTRAP/);
  assert.match(activity, /setSupportZoom\(true\)/);
  assert.match(activity, /setBuiltInZoomControls\(true\)/);
  assert.match(activity, /setDisplayZoomControls\(false\)/);
  assert.doesNotMatch(activity, /window\.__sub2apiTouch|new PointerEvent/);
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

test("Sub2API Android release signing explicitly supports Android v1 and v2 installers", () => {
  assert.match(buildScript, /--v1-signing-enabled true/);
  assert.match(buildScript, /--v2-signing-enabled true/);
});

test("Sub2API Android release identity comes from its monotonic client version", () => {
  assert.match(gradle, /applicationId = "com\.sub2api\.app"/);
  assert.match(gradle, /providers\.gradleProperty\("sub2apiVersion"\)/);
  assert.match(buildScript, /ANDROID_DIR}\/VERSION/);
  assert.doesNotMatch(buildScript, /backend\/scripts\/resolve-version\.sh/);
  assert.match(buildScript, /apksigner" verify --verbose --print-certs/);
  assert.match(buildScript, /sub2api-\$\{client_version\}\.apk/);
  assert.match(versionFile, /^[0-9]+\.[0-9]+\.[0-9]+$/);
});

test("Sub2API Android silently keeps a healthy source and only races after a real failure", () => {
  assert.doesNotMatch(activity, /connectionProgress/);
  assert.doesNotMatch(activity, /showConnectingState/);
  assert.equal((activity.match(/webView\.stopLoading\(\)/g) || []).length, 1);
  assert.match(activity, /verifyCurrentSource\(\)/);
  assert.match(activity, /SourceRegistry\.probe\(current\)/);
  assert.match(activity, /if \(result\.healthy\)[\s\S]*finishResolution\(generation\)/);
  assert.match(activity, /resolveAndLoad\(current, false\)/);
  assert.match(activity, /raceHealthySources\(\)/);
  assert.match(activity, /if \(coldStart\)[\s\S]*showConnectionFailure\(\)/);
  assert.match(activity, /if \(!coldStart\)[\s\S]*return;/);
  assert.doesNotMatch(activity, /switchSourceWithCountdown|switchCountdownActive|pendingSwitchSource/);
  assert.doesNotMatch(strings, /switching_server_countdown|秒后切换到服务器/);
});

test("Sub2API Android coalesces network callbacks and ignores stale WebView failures", () => {
  assert.match(activity, /pendingNetworkReevaluation = true/);
  assert.match(activity, /finishResolution\(int generation\)/);
  assert.match(activity, /resolutionGeneration/);
  assert.match(activity, /if \(generation != resolutionGeneration \|\| destroyed\)/);
  assert.match(activity, /isCurrentMainFrameRequest\(request\)/);
  assert.match(activity, /request\.isForMainFrame\(\)\s*&& isSelectedSourceUrl\(request\.getUrl\(\)\.toString\(\)\)/);
  assert.match(activity, /failedGeneration == navigationGeneration/);
  assert.match(activity, /navigationGeneration\+\+/);
  assert.match(activity, /pendingFailedSource = selectedSource/);
  assert.match(activity, /pendingFailedNavigationGeneration = failedGeneration/);
  assert.match(activity, /if \(!hadPendingFailure && currentPageFailed && current == selectedSource\)[\s\S]*loadSource\(current\)/);
  assert.match(activity, /loadSource\(winner\)/);
  assert.doesNotMatch(activity, /Toast\.makeText\([\s\S]{0,160}server|switching_server_countdown/);
});
