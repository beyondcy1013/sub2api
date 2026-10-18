import assert from 'node:assert/strict'
import { chromium } from 'playwright'

// Uses the deployed assets with an isolated, fully mocked management API. No
// credentials, production settings, account state or upstream requests are used.
const origins = process.argv.slice(2)
if (!origins.length) origins.push('http://127.0.0.1:18381', 'http://127.0.0.1:18382')
const browser = await chromium.launch({ headless: true, executablePath: process.env.CHROMIUM_EXECUTABLE_PATH || '/home/codes/third_party/browser-tools/bin/chromium', args: ['--no-sandbox'] })
const user = { id: 1, email: 'browser-fixture@example.invalid', role: 'admin', status: 'active', balance: 0, concurrency: 10 }
const policy = kind => ({ kind, enabled: false, account_scope: 'all', account_ids: [], platforms: [], statuses: ['error'], include_paused: false, interval_minutes: 5, model_id: '', recover: false, activate: false, resume: false, pause_on_downgrade: false, failure_threshold: 2 })
try {
 for (const origin of origins) {
  for (const [width, locale] of [[1440, 'zh'], [390, 'zh'], [320, 'en']]) {
   const context = await browser.newContext({ viewport: { width, height: 900 } })
   await context.addInitScript(({ user, locale }) => { localStorage.setItem('auth_token', 'isolated-browser-fixture'); localStorage.setItem('auth_user', JSON.stringify(user)); localStorage.setItem('token_expires_at', String(Date.now() + 3600000)); localStorage.setItem('sub2api_locale', locale); localStorage.setItem(`admin_guide_${user.id}_${user.role}_v4_interactive`, 'true') }, { user, locale })
   let state = { config: { revision: '', policies: [policy('connection'), policy('quality')] }, runtime: { running: false, next_run_at: null, last_run: null, history: [], results: [] } }
   let savedPolicy, savedBilling
   const errors = []
   const page = await context.newPage()
   page.on('pageerror', e => errors.push(String(e)))
   await page.route('**/api/**', async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname
    let data = {}
    if (path.endsWith('/auth/me')) data = user
    else if (path.endsWith('/setup/status')) data = { setup_required: false }
    else if (path.endsWith('/settings/public')) data = { site_name: 'Detection acceptance', registration_enabled: false, deployment_profile: origin.endsWith('18382') ? 'free' : 'main' }
    else if (path.endsWith('/accounts/automatic-detection')) {
     if (request.method() === 'PUT') { savedPolicy = request.postDataJSON(); state = { ...state, config: { ...savedPolicy, revision: 'saved' } } }
     data = state
    } else if (path.endsWith('/accounts/upstream-billing-probe/settings')) {
     if (request.method() === 'PUT') savedBilling = request.postDataJSON()
     data = savedBilling ?? { enabled: false, interval_minutes: 30 }
    } else if (path.endsWith('/accounts/request-quality-probe/settings')) data = { enabled: false, rule_ids: [], account_scope: 'all', account_ids: [], model_scope: 'all', models: [], timeout_seconds: 35, status_recent_minutes: 5 }
    else if (path.endsWith('/scheduling-rules/runtime')) data = { liveness: { enabled: false, running: false }, upstream_billing: { enabled: false, running: false } }
    else if (/\/accounts$/.test(path)) data = { items: [{ id: 1, name: 'Fixture account', platform: 'openai', type: 'apikey', status: 'error', schedulable: false, credentials: {}, extra: {}, concurrency: 10, priority: 0, rate_multiplier: 1, group_ids: [], groups: [], created_at: '2026-10-18T00:00:00Z', updated_at: '2026-10-18T00:00:00Z' }], total: 1, page: 1, page_size: 100, pages: 1 }
    else if (/degradation-rules|detection-records|quality-summaries|\/groups\/all$|\/proxies\/all$|\/models$|\/scheduled-test-plans$/.test(path)) data = []
    else if (/\/groups$|\/proxies$/.test(path)) data = { items: [], total: 0, pages: 0 }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'success', data }) })
   })
   await page.goto(`${origin}/admin/accounts`, { waitUntil: 'networkidle' })
   if (width < 1024) { await page.locator('[data-test="account-functions-trigger"]').click(); await page.locator('[data-test="functions-automatic-detection"]').click() }
   else await page.locator('[data-test="automatic-detection-entry"]').click()
   const panel = page.locator('[data-test="automatic-policies"]')
   await panel.locator('[data-test="policy-connection"]').waitFor()
   const connection = panel.locator('[data-test="policy-connection"]')
   assert.ok((await connection.textContent()).includes(locale === 'zh' ? '状态检测与恢复' : 'Health detection and recovery'))
   await connection.locator('[data-test="policy-enabled"]').check()
   await connection.locator('[data-test="scope-selected"]').check()
   assert.equal(await panel.locator('[data-test="save-policies"]').isDisabled(), true)
   await connection.locator('.grid input[type="checkbox"]').first().check()
   await connection.locator('[data-test="include-paused"]').check()
   await connection.locator('[data-test="recover"]').check()
   await connection.locator('[data-test="resume"]').check()
   await Promise.all([page.waitForResponse(r => r.url().endsWith('/automatic-detection') && r.request().method() === 'PUT'), panel.locator('[data-test="save-policies"]').click()])
   assert.deepEqual(savedPolicy.policies[0].account_ids, [1]); assert.equal(savedPolicy.policies[0].account_scope, 'selected'); assert.equal(savedPolicy.policies[0].resume, true)
   await connection.locator('[data-test="scope-all"]').check()
   await Promise.all([page.waitForResponse(r => r.url().endsWith('/automatic-detection') && r.request().method() === 'PUT'), panel.locator('[data-test="save-policies"]').click()])
   assert.equal(savedPolicy.policies[0].account_scope, 'all')
   await panel.locator('[data-test="policy-quality"] [data-test="policy-enabled"]').check()
   assert.equal(await panel.locator('[data-test="save-policies"]').isDisabled(), true)
   await panel.locator('[data-test="policy-quality"] [data-test="policy-enabled"]').uncheck()
   await panel.locator('[data-test="billing-policy"] input[value="all"]').check()
   await Promise.all([page.waitForResponse(r => r.url().endsWith('/upstream-billing-probe/settings') && r.request().method() === 'PUT'), panel.locator('[data-test="save-billing"]').click()])
   assert.equal(savedBilling.account_scope, 'all')
   const box = await panel.boundingBox(); assert.ok(box.x >= 0 && box.x + box.width <= width + 1, `panel exceeds viewport at ${width}`)
   await page.locator('[data-test="automatic-tab-request"]').click()
   await page.locator('[data-testid="scheduling-rule-quality-probe-enabled"]').waitFor()
   await page.locator('[data-test="automatic-tab-scheduled"]').click()
   await page.locator('[data-test="automatic-tab-rules"]').click()
   await page.locator('[data-test="automatic-tab-policies"]').click()
   assert.equal(await connection.locator('[data-test="recover"]').isChecked(), true)
   await page.getByRole('button', { name: 'Close modal', exact: true }).click()
   if (width < 1024) { await page.locator('[data-test="account-functions-trigger"]').click(); await page.locator('[data-test="functions-account-detection"]').click() }
   else await page.locator('[data-test="account-detection-entry"]').click()
   assert.equal(await page.locator('[data-test="detection-tab-automatic"]').count(), 0)
   assert.equal(await page.getByRole('dialog').getByRole('tab').count(), 2)
   assert.deepEqual(errors, [])
   console.log(JSON.stringify({ origin, width, locale, scopeSelection: 'passed', recoveryOptions: 'passed', missingStandard: 'blocked', billingScope: 'passed', geometry: 'passed', api: 'mocked' }))
   await context.close()
  }
 }
} finally { await browser.close() }
