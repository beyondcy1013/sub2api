import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QualityManagementDialog from '../QualityManagementDialog.vue'

const api = vi.hoisted(() => ({ list: vi.fn(), rules: vi.fn(), settings: vi.fn(), records: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list: api.list, getRequestQualityProbeSettings: api.settings, listDetectionRecords: api.records }, degradationRules: { list: api.rules } } }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))

const mountDialog = (props = {}) => mount(QualityManagementDialog, { props: { show: true, ...props }, global: { plugins: [createPinia()], stubs: { BaseDialog: { template: '<div><slot /></div>' }, Select: true } } })

describe('manual quality management', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.list.mockResolvedValue({ items: [{ id: 1, name: 'A', status: 'active', platform: 'openai', extra: { pelican_downgraded: true } }], total: 1 })
    api.rules.mockResolvedValue([]); api.settings.mockResolvedValue({ enabled: false, rule_ids: [] }); api.records.mockResolvedValue([])
  })

  it('shows only manual and history tabs', async () => {
    const wrapper = mountDialog(); await flushPromises()
    expect(wrapper.findAll('[role="tab"]')).toHaveLength(2)
    expect(wrapper.get('[data-test="detection-tab-records"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.text()).toContain('A #1')
    wrapper.unmount()
  })

  it('opens manual account selection without automatic policy controls', async () => {
    const wrapper = mountDialog({ initialTab: 'manual' }); await flushPromises()
    expect(wrapper.get('[data-test="detection-tab-manual"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.text()).not.toContain('automaticDetection.title')
    wrapper.unmount()
  })
})
