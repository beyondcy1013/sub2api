import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SchedulingRateModal from '../SchedulingRateModal.vue'
import type { Account } from '@/types'

const { showInfoMock } = vi.hoisted(() => ({ showInfoMock: vi.fn() }))

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showInfo: showInfoMock })
}))

const account: Account = {
  id: 1,
  name: 'rate-account',
  platform: 'openai',
  type: 'apikey',
  proxy_id: null,
  concurrency: 1,
  priority: 1,
  rate_multiplier: 0.8,
  scheduling_rate_multiplier: 0.8,
  scheduling_rate_known: true,
  scheduling_rate_source: 'manual',
  scheduling_rate_sync_mode: 'manual_lock',
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-07-22T00:00:00Z',
  updated_at: '2026-07-22T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null
}

describe('SchedulingRateModal', () => {
  beforeEach(() => {
    showInfoMock.mockReset()
  })

  it('locks the rate and shows a toast after the manual multiplier changes', async () => {
    const wrapper = mount(SchedulingRateModal, {
      props: { show: true, account, upstreamRate: 0.35, upstreamKnown: true, conflict: true },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })

    expect(wrapper.get('[data-testid="scheduling-rate-conflict"]').exists()).toBe(true)
    await wrapper.get('[data-testid="scheduling-rate-manual"]').setValue('0.45')

    expect(wrapper.get<HTMLInputElement>('[data-testid="scheduling-rate-auto-overwrite"]').element.checked).toBe(false)
    expect(showInfoMock).toHaveBeenCalledOnce()
    expect(showInfoMock).toHaveBeenCalledWith('admin.accounts.schedulingRate.manualEditLocks')

    await wrapper.get('[data-testid="scheduling-rate-save"]').trigger('click')
    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({
      sync_mode: 'manual_lock',
      rate_multiplier: 0.45
    })
  })

  it('disables the manual multiplier while automatic overwrite is selected', async () => {
    const autoAccount: Account = {
      ...account,
      scheduling_rate_source: 'upstream',
      scheduling_rate_sync_mode: 'auto_overwrite'
    }
    const wrapper = mount(SchedulingRateModal, {
      props: { show: true, account: autoAccount },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })

    const manualInput = wrapper.get<HTMLInputElement>('[data-testid="scheduling-rate-manual"]')
    expect(manualInput.element.disabled).toBe(true)
    expect(manualInput.classes()).toContain('disabled:bg-gray-100')

    await wrapper.get('[data-testid="scheduling-rate-auto-overwrite"]').setValue(false)
    expect(manualInput.element.disabled).toBe(false)
  })

  it('copies the detected upstream rate into the manual rate on request', async () => {
    const wrapper = mount(SchedulingRateModal, {
      props: { show: true, account, upstreamRate: 0.35, upstreamKnown: true, conflict: true },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })

    await wrapper.get('[data-testid="scheduling-rate-copy-upstream"]').trigger('click')
    await wrapper.get('[data-testid="scheduling-rate-save"]').trigger('click')

    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({ sync_mode: 'manual_lock', rate_multiplier: 0.35 })
    expect(showInfoMock).toHaveBeenCalledWith('admin.accounts.schedulingRate.manualEditLocks')
  })
})
