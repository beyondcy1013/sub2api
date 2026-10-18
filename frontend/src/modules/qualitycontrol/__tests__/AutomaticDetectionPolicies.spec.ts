import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AutomaticDetectionPolicies from '../AutomaticDetectionPolicies.vue'
import type { AutomaticDetectionState, DetectionPolicy } from '../automaticDetection'
const api = vi.hoisted(() => ({ get: vi.fn(), save: vi.fn(), billing: vi.fn(), saveBilling: vi.fn(), runtime: vi.fn() }))
vi.mock('../automaticDetection', async () => ({ ...await vi.importActual('../automaticDetection'), automaticDetectionAPI: { get: api.get, save: api.save } }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { getUpstreamBillingProbeSettings: api.billing, updateUpstreamBillingProbeSettings: api.saveBilling } } }))
vi.mock('@/api/admin/superPriority', () => ({ default: { getRuntime: api.runtime } }))
const policy = (kind: 'connection' | 'quality'): DetectionPolicy => ({ kind, enabled: false, account_scope: 'all', account_ids: [], platforms: [], statuses: ['error'], include_paused: false, interval_minutes: 5, model_id: '', recover: false, activate: false, resume: false, pause_on_downgrade: false, failure_threshold: 2 })
const value = (): AutomaticDetectionState => ({ config: { revision: '', policies: [policy('connection'), policy('quality')] }, runtime: { running: false, history: [], results: [] } })
const accounts = [{ id: 1, name: 'A', status: 'error', platform: 'openai', type: 'apikey', extra: {} }, { id: 2, name: 'B', status: 'error', platform: 'anthropic', type: 'apikey', extra: {} }] as any
const mountPanel = (props = {}) => mount(AutomaticDetectionPolicies, { props: { accounts, hasStandard: true, ...props }, global: { plugins: [createPinia(), createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } })] } })
describe('automatic detection policies', () => {
  beforeEach(() => {
    vi.clearAllMocks(); api.get.mockResolvedValue(value()); api.billing.mockResolvedValue({ enabled: false, interval_minutes: 30 }); api.runtime.mockResolvedValue({ upstream_billing: { enabled: false, running: false } }); api.save.mockImplementation(async config => ({ ...value(), config })); api.saveBilling.mockImplementation(async config => config)
  })
  it('requires explicit selection and submits only the selected account IDs', async () => {
    const w = mountPanel(); await flushPromises()
    const p = w.get('[data-test="policy-connection"]')
    await p.get('[data-test="policy-enabled"]').setValue(true)
    await p.get('[data-test="scope-selected"]').setValue()
    expect(w.get('[data-test="save-policies"]').attributes('disabled')).toBeDefined()
    await p.get('.grid input[type="checkbox"]').setValue(true)
    await w.get('[data-test="save-policies"]').trigger('click'); await flushPromises()
    expect(api.save.mock.calls[0][0].policies[0]).toMatchObject({ account_scope: 'selected', account_ids: [1] })
    w.unmount()
  })
  it('persists state and recovery choices and supports all accounts without enumerating IDs', async () => {
    const w = mountPanel(); await flushPromises()
    const p = w.get('[data-test="policy-connection"]')
    await p.get('[data-test="policy-enabled"]').setValue(true)
    await p.get('[data-test="state-inactive"]').setValue(true)
    await p.get('[data-test="include-paused"]').setValue(true)
    await p.get('[data-test="recover"]').setValue(true)
    await p.get('[data-test="activate"]').setValue(true)
    await p.get('[data-test="resume"]').setValue(true)
    await w.get('[data-test="save-policies"]').trigger('click'); await flushPromises()
    expect(api.save.mock.calls[0][0].policies[0]).toMatchObject({ account_scope: 'all', account_ids: [], statuses: ['error', 'inactive'], include_paused: true, recover: true, activate: true, resume: true })
    w.unmount()
  })
  it('does not allow periodic quality tests without a designated standard question', async () => {
    const w = mountPanel({ hasStandard: false }); await flushPromises()
    await w.get('[data-test="policy-quality"] [data-test="policy-enabled"]').setValue(true)
    expect(w.text()).toContain('尚未指定标准试题')
    expect(w.get('[data-test="save-policies"]').attributes('disabled')).toBeDefined()
    expect(api.save).not.toHaveBeenCalled(); w.unmount()
  })
  it('refreshes runtime without overwriting unsaved policy edits', async () => {
    const w = mountPanel(); await flushPromises()
    await w.get('[data-test="policy-connection"] [data-test="interval"]').setValue(15)
    await w.findAll('button').find(b => b.text() === '刷新运行情况')!.trigger('click'); await flushPromises()
    expect((w.get('[data-test="policy-connection"] [data-test="interval"]').element as HTMLInputElement).value).toBe('15')
    expect(api.save).not.toHaveBeenCalled(); w.unmount()
  })
  it('saves a dynamic all-account scope for rate probes separately', async () => {
    const w = mountPanel(); await flushPromises()
    await w.get('[data-test="billing-policy"] input[value="all"]').setValue()
    await w.get('[data-test="save-billing"]').trigger('click'); await flushPromises()
    expect(api.saveBilling).toHaveBeenCalledWith(expect.objectContaining({ account_scope: 'all', account_ids: [] }))
    expect(api.save).not.toHaveBeenCalled(); w.unmount()
  })
  it('shows errors and never sends default settings when loading failed', async () => {
    api.get.mockRejectedValue(new Error('read failed')); const w = mountPanel(); await flushPromises()
    expect(w.get('[role="alert"]').text()).toContain('read failed'); expect(w.find('[data-test="save-policies"]').exists()).toBe(false); expect(api.save).not.toHaveBeenCalled(); w.unmount()
  })
})
