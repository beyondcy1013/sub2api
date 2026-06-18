package server

import (
	"net/http"
	"strings"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const balanceCheckSettingsHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>余额检测设置</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f5f7fb;
      --panel: #ffffff;
      --text: #1f2937;
      --muted: #64748b;
      --line: #d8dee8;
      --accent: #0f766e;
      --accent-strong: #0b5f59;
      --danger: #b42318;
      --ok: #067647;
      --shadow: 0 18px 50px rgba(15, 23, 42, .16);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: var(--bg);
      color: var(--text);
    }
    .shell {
      width: min(980px, calc(100vw - 28px));
      margin: 28px auto;
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 14px;
    }
    h1 {
      margin: 0;
      font-size: 22px;
      line-height: 1.25;
      font-weight: 700;
      letter-spacing: 0;
    }
    .links {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      justify-content: flex-end;
    }
    a, button {
      min-height: 36px;
      border-radius: 6px;
      border: 1px solid var(--line);
      background: #fff;
      color: var(--text);
      padding: 8px 12px;
      font-size: 14px;
      line-height: 18px;
      text-decoration: none;
      cursor: pointer;
    }
    button.primary {
      border-color: var(--accent);
      background: var(--accent);
      color: #fff;
      font-weight: 600;
    }
    button.primary:hover { background: var(--accent-strong); }
    button:disabled {
      opacity: .62;
      cursor: not-allowed;
    }
    .dialog {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
      box-shadow: var(--shadow);
      overflow: hidden;
    }
    .dialog-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 16px 18px;
      border-bottom: 1px solid var(--line);
      background: #fbfcfe;
    }
    .status {
      min-height: 24px;
      font-size: 13px;
      color: var(--muted);
      text-align: right;
    }
    .status.ok { color: var(--ok); }
    .status.err { color: var(--danger); }
    form {
      padding: 18px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 14px;
    }
    .field {
      display: flex;
      flex-direction: column;
      gap: 6px;
      min-width: 0;
    }
    .field.full { grid-column: 1 / -1; }
    label {
      font-size: 13px;
      color: #334155;
      font-weight: 600;
    }
    input {
      width: 100%;
      min-height: 38px;
      border: 1px solid var(--line);
      border-radius: 6px;
      padding: 8px 10px;
      font-size: 14px;
      color: var(--text);
      background: #fff;
    }
    input:focus {
      outline: 2px solid rgba(15, 118, 110, .18);
      border-color: var(--accent);
    }
    .toggle-row {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
      margin-bottom: 16px;
    }
    .toggle {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      border: 1px solid var(--line);
      border-radius: 8px;
      padding: 12px;
      background: #fff;
    }
    .toggle span {
      font-size: 14px;
      font-weight: 600;
      color: #334155;
    }
    .toggle input {
      width: 42px;
      min-height: 22px;
      accent-color: var(--accent);
    }
    .actions {
      display: flex;
      justify-content: flex-end;
      gap: 8px;
      padding-top: 16px;
    }
    .meta {
      margin-top: 12px;
      color: var(--muted);
      font-size: 12px;
      overflow-wrap: anywhere;
    }
    @media (max-width: 720px) {
      .shell { width: min(100vw - 18px, 980px); margin: 12px auto; }
      .topbar, .dialog-head { align-items: flex-start; flex-direction: column; }
      .links, .actions { width: 100%; justify-content: stretch; }
      .links a, .links button, .actions button { flex: 1; }
      .grid, .toggle-row { grid-template-columns: 1fr; }
      .status { text-align: left; }
    }
  </style>
</head>
<body>
  <main class="shell">
    <div class="topbar">
      <h1>余额检测设置</h1>
      <div class="links">
        <a href="/admin/settings">返回后台</a>
        <button id="reloadBtn" type="button">重新读取</button>
      </div>
    </div>
    <section class="dialog" aria-label="余额检测设置窗口">
      <div class="dialog-head">
        <div>
          <strong>余额检测规则</strong>
          <div class="meta">保存写入 data/config.yaml，重启服务后余额检测任务生效。</div>
        </div>
        <div id="status" class="status">正在加载...</div>
      </div>
      <form id="settingsForm">
        <div class="toggle-row">
          <label class="toggle"><span>启用余额检测</span><input id="enabled" type="checkbox"></label>
          <label class="toggle"><span>仅检测 5 小时额度账号</span><input id="requireQuotaHourlyLimit" type="checkbox"></label>
        </div>
        <div class="grid">
          <div class="field">
            <label for="interval">检测间隔</label>
            <input id="interval" name="interval" autocomplete="off" placeholder="@every 5m" required>
          </div>
          <div class="field">
            <label for="pauseDurationHours">暂停时长（小时）</label>
            <input id="pauseDurationHours" name="pauseDurationHours" type="number" min="0.1" step="0.1" required>
          </div>
          <div class="field full">
            <label for="balanceUrl">余额查询地址</label>
            <input id="balanceUrl" name="balanceUrl" type="url" autocomplete="off" required>
          </div>
          <div class="field">
            <label for="requestTimeoutSeconds">请求超时（秒）</label>
            <input id="requestTimeoutSeconds" name="requestTimeoutSeconds" type="number" min="1" step="1" required>
          </div>
          <div class="field">
            <label for="maxConcurrentChecks">最大并发检测数</label>
            <input id="maxConcurrentChecks" name="maxConcurrentChecks" type="number" min="1" step="1" required>
          </div>
          <div class="field">
            <label for="stopWhenCurrentBelow">余额低于此值停止</label>
            <input id="stopWhenCurrentBelow" name="stopWhenCurrentBelow" type="number" min="0" step="0.01" required>
          </div>
          <div class="field">
            <label for="resumeWhenCurrentAbove">余额恢复到此值自动恢复</label>
            <input id="resumeWhenCurrentAbove" name="resumeWhenCurrentAbove" type="number" min="0" step="0.01" required>
          </div>
          <div class="field">
            <label for="minDecrease">余额下降阈值</label>
            <input id="minDecrease" name="minDecrease" type="number" min="0" step="0.01" required>
          </div>
          <div class="field">
            <label for="pauseWhenCurrentBelow">余额低于此值暂停</label>
            <input id="pauseWhenCurrentBelow" name="pauseWhenCurrentBelow" type="number" min="0" step="0.01" required>
          </div>
          <div class="field">
            <label for="pauseWhenDropPercent">余额下降百分比阈值</label>
            <input id="pauseWhenDropPercent" name="pauseWhenDropPercent" type="number" min="0" step="0.01" required>
          </div>
        </div>
        <div class="actions">
          <button id="saveBtn" class="primary" type="submit">保存设置</button>
        </div>
        <div id="configPath" class="meta"></div>
      </form>
    </section>
  </main>
  <script nonce="__LOCAL_CSP_NONCE__">
    const apiPath = '/api/v1/admin/settings/balance-check';
    const fields = {
      enabled: document.getElementById('enabled'),
      interval: document.getElementById('interval'),
      balance_url: document.getElementById('balanceUrl'),
      request_timeout_seconds: document.getElementById('requestTimeoutSeconds'),
      max_concurrent_checks: document.getElementById('maxConcurrentChecks'),
      stop_when_current_below: document.getElementById('stopWhenCurrentBelow'),
      resume_when_current_above: document.getElementById('resumeWhenCurrentAbove'),
      pause_duration_hours: document.getElementById('pauseDurationHours'),
      min_decrease: document.getElementById('minDecrease'),
      pause_when_current_below: document.getElementById('pauseWhenCurrentBelow'),
      pause_when_drop_percent: document.getElementById('pauseWhenDropPercent'),
      require_quota_hourly_limit: document.getElementById('requireQuotaHourlyLimit')
    };
    const statusEl = document.getElementById('status');
    const pathEl = document.getElementById('configPath');
    const saveBtn = document.getElementById('saveBtn');
    const reloadBtn = document.getElementById('reloadBtn');
    const form = document.getElementById('settingsForm');

    function token() {
      return localStorage.getItem('auth_token') || localStorage.getItem('token') || '';
    }
    function headers() {
      const h = { 'Content-Type': 'application/json' };
      const t = token();
      if (t) h.Authorization = 'Bearer ' + t;
      return h;
    }
    function setStatus(message, kind) {
      statusEl.textContent = message;
      statusEl.className = 'status ' + (kind || '');
    }
    function setBusy(busy) {
      saveBtn.disabled = busy;
      reloadBtn.disabled = busy;
    }
    function fill(cfg) {
      fields.enabled.checked = Boolean(cfg.enabled);
      fields.interval.value = cfg.interval || '@every 5m';
      fields.balance_url.value = cfg.balance_url || '';
      fields.request_timeout_seconds.value = cfg.request_timeout_seconds || 30;
      fields.max_concurrent_checks.value = cfg.max_concurrent_checks || 1;
      fields.stop_when_current_below.value = cfg.stop_when_current_below ?? 0;
      fields.resume_when_current_above.value = cfg.resume_when_current_above ?? 0;
      fields.pause_duration_hours.value = cfg.pause_duration_hours || 5;
      fields.min_decrease.value = cfg.min_decrease ?? 5;
      fields.pause_when_current_below.value = cfg.pause_when_current_below ?? 0;
      fields.pause_when_drop_percent.value = cfg.pause_when_drop_percent ?? 0;
      fields.require_quota_hourly_limit.checked = Boolean(cfg.require_quota_hourly_limit);
    }
    function readForm() {
      return {
        enabled: fields.enabled.checked,
        interval: fields.interval.value.trim(),
        balance_url: fields.balance_url.value.trim(),
        request_timeout_seconds: Number(fields.request_timeout_seconds.value),
        max_concurrent_checks: Number(fields.max_concurrent_checks.value),
        stop_when_current_below: Number(fields.stop_when_current_below.value),
        resume_when_current_above: Number(fields.resume_when_current_above.value),
        pause_duration_hours: Number(fields.pause_duration_hours.value),
        min_decrease: Number(fields.min_decrease.value),
        pause_when_current_below: Number(fields.pause_when_current_below.value),
        pause_when_drop_percent: Number(fields.pause_when_drop_percent.value),
        require_quota_hourly_limit: fields.require_quota_hourly_limit.checked
      };
    }
    async function request(method, body) {
      const res = await fetch(apiPath, {
        method,
        headers: headers(),
        body: body ? JSON.stringify(body) : undefined
      });
      const payload = await res.json().catch(() => ({}));
      if (!res.ok || payload.code !== 0) {
        throw new Error(payload.message || ('HTTP ' + res.status));
      }
      return payload.data;
    }
    async function load() {
      setBusy(true);
      setStatus('正在加载...', '');
      try {
        const data = await request('GET');
        fill(data.config || {});
        pathEl.textContent = data.config_path ? '配置文件：' + data.config_path : '';
        setStatus('已读取当前配置', 'ok');
      } catch (err) {
        setStatus((err && err.message ? err.message : '加载失败') + '。请确认已登录管理员账号。', 'err');
      } finally {
        setBusy(false);
      }
    }
    async function save(event) {
      event.preventDefault();
      setBusy(true);
      setStatus('正在保存...', '');
      try {
        const data = await request('PUT', readForm());
        fill(data.config || {});
        pathEl.textContent = data.config_path ? '配置文件：' + data.config_path : '';
        setStatus('保存成功，重启服务后生效', 'ok');
      } catch (err) {
        setStatus(err && err.message ? err.message : '保存失败', 'err');
      } finally {
        setBusy(false);
      }
    }
    reloadBtn.addEventListener('click', load);
    form.addEventListener('submit', save);
    load();
  </script>
</body>
</html>`

func registerLocalPages(r *gin.Engine) {
	r.GET("/local/balance-check-settings", serveBalanceCheckSettingsPage)
}

func serveBalanceCheckSettingsPage(c *gin.Context) {
	html := strings.ReplaceAll(balanceCheckSettingsHTML, "__LOCAL_CSP_NONCE__", middleware2.GetNonceFromContext(c))
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
