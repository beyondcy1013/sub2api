import { mount, flushPromises } from '@vue/test-utils'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import AutomaticDetectionDialog from '../AutomaticDetectionDialog.vue'
const api = vi.hoisted(() => ({ list: vi.fn(), settings: vi.fn(), rules: vi.fn(), save: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list: api.list, getRequestQualityProbeSettings: api.settings, updateRequestQualityProbeSettings: api.save }, degradationRules: { list: api.rules } } }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))
const stubs = {
  BaseDialog: { template: '<div v-if="show"><slot /></div>', props: ['show'] },
  AutomaticDetectionPolicies: { name: 'AutomaticDetectionPolicies', props: ['accounts', 'hasStandard'], template: '<div />' },
  RequestQualityProbeForm: { template: '<div />' },
  ScheduledTestsPanel: { name: 'ScheduledTestsPanel', props: ['accountId', 'rulesOnly', 'modelOptions'], template: '<div />' },
}
describe('independent automatic detection dialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.list.mockResolvedValueOnce({ items: [{ id: 1, name: 'A' }], total: 2 }).mockResolvedValue({ items: [{ id: 2, name: 'B' }], total: 2 })
    api.settings.mockResolvedValue({ rule_ids: [1] }); api.rules.mockResolvedValue([])
  })
  it('loads all account pages independently and exposes automatic modes', async () => {
    const w = mount(AutomaticDetectionDialog, { props: { show: true, accounts: [] }, global: { stubs } })
    await flushPromises()
    expect(api.list).toHaveBeenCalledTimes(2)
    expect(w.findComponent({ name: 'AutomaticDetectionPolicies' }).props('accounts').map((a: any) => a.id)).toEqual([1, 2])
    expect(w.findComponent({ name: 'AutomaticDetectionPolicies' }).props('hasStandard')).toBe(true)
    expect(w.findAll('[role="tab"]')).toHaveLength(4)
    await w.get('[data-test="automatic-tab-rules"]').trigger('click')
    expect(w.findComponent({ name: 'ScheduledTestsPanel' }).props('rulesOnly')).toBe(true)
    w.unmount()
  })
  it('resets to the requested mode on each opening with per-account context', async () => {
    const w = mount(AutomaticDetectionDialog, { props: { show: false, accounts: [] }, global: { stubs } })
    await w.setProps({ show: true, initialMode: 'scheduled', initialAccountId: 42 })
    await flushPromises()
    expect(w.findComponent({ name: 'ScheduledTestsPanel' }).props('accountId')).toBe(42)
    await w.setProps({ show: false })
    await w.setProps({ show: true, initialMode: 'policies' })
    await flushPromises()
    expect(w.get('[data-test="automatic-tab-policies"]').attributes('aria-selected')).toBe('true')
    w.unmount()
  })
})
