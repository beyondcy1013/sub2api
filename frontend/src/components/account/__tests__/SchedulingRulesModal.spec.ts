import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SchedulingRulesModal from '../SchedulingRulesModal.vue'

const api = vi.hoisted(() => ({
  getSuperPriority: vi.fn(),
  deactivate: vi.fn(),
  updateSuperPriority: vi.fn(),
  getProbeSettings: vi.fn(),
  updateProbeSettings: vi.fn(),
  getQualityProbeSettings: vi.fn(),
  updateQualityProbeSettings: vi.fn(),
  refreshSchedulingRules: vi.fn()
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key
  })
}))
vi.mock('@/api/admin', () => ({
  adminAPI: {
    superPriority: {
      get: api.getSuperPriority,
      deactivate: api.deactivate,
      update: api.updateSuperPriority,
      refresh: api.refreshSchedulingRules
    },
    degradationRules: {list: vi.fn().mockResolvedValue([])},
    accounts: {
      list: vi.fn().mockResolvedValue({items:[{id:10,name:"A",platform:"openai"},{id:20,name:"B",platform:"openai"}],total:2}),
      getUpstreamBillingProbeSettings: api.getProbeSettings,
      updateUpstreamBillingProbeSettings: api.updateProbeSettings,
      getRequestQualityProbeSettings: api.getQualityProbeSettings,
      updateRequestQualityProbeSettings: api.updateQualityProbeSettings
    }
  }
}))

const flush = async () => await Promise.resolve()

describe('SchedulingRulesModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getQualityProbeSettings.mockResolvedValue({
      enabled: false,
      models: ['gpt-5.1', 'gpt-5.2'],
      recent_minutes: 10,
      account_scope: 'error_or_downgraded',
      account_ids: []
    })
    api.updateQualityProbeSettings.mockResolvedValue({
      enabled: false,
      models: ['gpt-5.1', 'gpt-5.2'],
      recent_minutes: 10,
      account_scope: 'error_or_downgraded',
      account_ids: []
    })
  })
  it('shows algorithm help beside the dialog title and reveals it on hover', async () => {
    const wrapper = mount(SchedulingRulesModal, {
      attachTo: document.body,
      props: { show: false },
      global: {
        stubs: {
          BaseDialog: {
            props: ['title'],
            template: '<div><div>{{ title }}<slot name="title-actions" /></div><slot /><slot name="footer" /></div>'
          }
        }
      }
    })

    expect(wrapper.get('[data-testid="scheduling-rules-help"]').text()).toBe('admin.accounts.schedulingRules.help')
    await wrapper.getComponent({ name: 'HelpTooltip' }).get('.group').trigger('mouseenter')
    await flush()

    const tooltip = document.body.querySelector('[role="tooltip"]')
    expect(tooltip).not.toBeNull()
    expect((tooltip as HTMLElement).style.display).not.toBe('none')
    expect(tooltip?.textContent).toContain('admin.accounts.schedulingRules.helpLowestCost')
    expect(tooltip?.textContent).toContain('admin.accounts.schedulingRules.helpLiveness')

    wrapper.unmount()
  })

  it('saves the selected lowest-cost rule, disables the legacy overlay, and updates the probe interval', async () => {
    api.getSuperPriority.mockResolvedValue({ mode: 'super_priority', base_strategy: 'default', failure_threshold: 2, check_interval: '@every 1m', liveness_enabled: true, test_model_id: '', test_prompt: '', activated_at: '', demoted_at: '', is_active: true })
    api.getProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 30, notify_on_change_only: false })
    api.deactivate.mockResolvedValue({ message: 'ok' })
    api.updateSuperPriority.mockResolvedValue({ message: 'ok', restart_required: false })
    api.updateProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 5 })
    const wrapper = mount(SchedulingRulesModal, {
      props: { show: true },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })

    await flush()
    await flush()
    expect(wrapper.get('[data-testid="scheduling-rule-strategy-group"]').attributes('aria-label')).toBe('admin.accounts.schedulingRules.strategy')
    expect(wrapper.find('[data-testid="scheduling-rule-liveness-include-unschedulable"]').exists()).toBe(false)
    await wrapper.get('[data-testid="scheduling-rule-lowest-cost"]').trigger('click')
    await wrapper.get('[data-testid="scheduling-rule-liveness-interval"]').setValue(3)
    await wrapper.get('[data-testid="scheduling-rule-liveness-threshold"]').setValue(4)
    expect(wrapper.find('[data-testid="scheduling-rule-liveness-abnormal-only"]').exists()).toBe(false)
    await wrapper.get('[data-testid="scheduling-rule-interval"]').setValue(5)
    await wrapper.get('[data-testid="scheduling-rule-notify-on-change-only"]').setValue(true)
    expect(wrapper.find('[data-testid="scheduling-rule-model-mismatch-on-error"]').exists()).toBe(true)
    await wrapper.get('[data-testid="scheduling-rule-model-mismatch-on-error"]').setValue(true)
    await wrapper.get('[data-testid="scheduling-rule-save"]').trigger('click')
    await flush()
    await flush()

    expect(api.deactivate).toHaveBeenCalledOnce()
    expect(api.updateSuperPriority).toHaveBeenCalledWith(expect.objectContaining({
      base_strategy: 'lowest_cost',
      check_interval: '@every 3m',
      liveness_enabled: true,
      failure_threshold: 4,
      liveness_include_unschedulable: false,
      model_mismatch_on_error: true
    }))
    expect(api.updateProbeSettings).toHaveBeenCalledWith({ enabled: true, interval_minutes: 5, notify_on_change_only: true })
    expect(wrapper.emitted('saved')).toHaveLength(1)
    wrapper.unmount()
  })

  it('runs an immediate scheduling refresh without closing the dialog', async () => {
    api.getSuperPriority.mockResolvedValue({ mode: 'normal', base_strategy: 'lowest_cost', failure_threshold: 2, check_interval: '@every 1m', test_model_id: '', test_prompt: '', activated_at: '', demoted_at: '', is_active: false })
    api.getProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 30, notify_on_change_only: false })
    const result = {
      liveness: { checked: 3, succeeded: 2, failed: 1 },
      upstream_billing: { checked: 2, succeeded: 2, failed: 0 }
    }
    api.refreshSchedulingRules.mockResolvedValue(result)
    const wrapper = mount(SchedulingRulesModal, {
      props: { show: true },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })

    await flush()
    await flush()
    await wrapper.get('[data-testid="scheduling-rule-refresh"]').trigger('click')
    await flush()
    await flush()

    expect(api.refreshSchedulingRules).toHaveBeenCalledOnce()
    expect(wrapper.emitted('refreshed')).toEqual([[result]])
    expect(wrapper.emitted('close')).toBeUndefined()
    wrapper.unmount()
  })

  it('shows the server countdown and latest liveness result in the upper-right header', async () => {
    api.getSuperPriority.mockResolvedValue({
      mode: 'normal', base_strategy: 'lowest_cost', failure_threshold: 2, check_interval: '@every 1m',
      liveness_include_unschedulable: false, test_model_id: '', test_prompt: '', activated_at: '', demoted_at: '', is_active: false,
      liveness_runtime: {
        enabled: true,
        running: false,
        next_run_at: new Date(Date.now() + 90_000).toISOString(),
        last_run: {
          trigger: 'scheduled',
          started_at: '2026-07-27T12:00:00Z',
          finished_at: '2026-07-27T12:00:05Z',
          result: { checked: 4, succeeded: 2, failed: 1, skipped: 1 }
        }
      }
    })
    api.getProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 30, notify_on_change_only: false })
    const wrapper = mount(SchedulingRulesModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot name="header-actions" /><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await flush()
    await flush()

    const runtime = wrapper.get('[data-testid="scheduling-rules-runtime"]')
    expect(runtime.text()).toContain('admin.accounts.schedulingRules.runtimeCountdown')
    expect(runtime.text()).toContain('"duration":"00:01:30"')
    expect(runtime.text()).toContain('admin.accounts.schedulingRules.runtimeLastResult')
    expect(wrapper.find('[data-testid="scheduling-rule-liveness-include-unschedulable"]').exists()).toBe(false)

    wrapper.unmount()
  })

  it('opens the authoritative quality settings without overwriting them when saving scheduling rules', async () => {
    const wrapper = mount(SchedulingRulesModal, {
      props: { show: true },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })
    await flushPromises()
    expect(wrapper.find('[data-testid="scheduling-rule-quality-probe-enabled"]').exists()).toBe(false)
    await wrapper.get('[data-testid="scheduling-rule-quality-settings"]').trigger('click')
    expect(wrapper.emitted('detection')).toHaveLength(1)
    await wrapper.get('[data-testid="scheduling-rule-save"]').trigger('click')
    await flushPromises()
    expect(api.updateQualityProbeSettings).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
