import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SchedulingRulesModal from '../SchedulingRulesModal.vue'

const api = vi.hoisted(() => ({
  getSuperPriority: vi.fn(),
  deactivate: vi.fn(),
  updateSuperPriority: vi.fn(),
  getProbeSettings: vi.fn(),
  updateProbeSettings: vi.fn()
}))

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/admin', () => ({
  adminAPI: {
    superPriority: { get: api.getSuperPriority, deactivate: api.deactivate, update: api.updateSuperPriority },
    accounts: { getUpstreamBillingProbeSettings: api.getProbeSettings, updateUpstreamBillingProbeSettings: api.updateProbeSettings }
  }
}))

const flush = async () => await Promise.resolve()

describe('SchedulingRulesModal', () => {
  it('saves the selected lowest-cost rule, disables the legacy overlay, and updates the probe interval', async () => {
    api.getSuperPriority.mockResolvedValue({ mode: 'super_priority', base_strategy: 'default', failure_threshold: 2, check_interval: '@every 1m', test_model_id: '', test_prompt: '', activated_at: '', demoted_at: '', is_active: true })
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
    await wrapper.get('[data-testid="scheduling-rule-lowest-cost"]').trigger('click')
    await wrapper.get('[data-testid="scheduling-rule-liveness-interval"]').setValue(3)
    await wrapper.get('[data-testid="scheduling-rule-liveness-threshold"]').setValue(4)
    await wrapper.get('[data-testid="scheduling-rule-interval"]').setValue(5)
    await wrapper.get('[data-testid="scheduling-rule-notify-on-change-only"]').setValue(true)
    await wrapper.get('[data-testid="scheduling-rule-save"]').trigger('click')
    await flush()
    await flush()

    expect(api.deactivate).toHaveBeenCalledOnce()
    expect(api.updateSuperPriority).toHaveBeenCalledWith(expect.objectContaining({
      base_strategy: 'lowest_cost',
      check_interval: '@every 3m',
      failure_threshold: 4
    }))
    expect(api.updateProbeSettings).toHaveBeenCalledWith({ enabled: true, interval_minutes: 5, notify_on_change_only: true })
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })
})
