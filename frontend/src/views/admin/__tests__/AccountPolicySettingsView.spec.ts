import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { Account } from '@/types'

import AccountPolicySettingsView from '../AccountPolicySettingsView.vue'

const {
  list,
  getPolicySettings,
  updatePolicySettings,
  recoverState,
  setSchedulable,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  list: vi.fn(),
  getPolicySettings: vi.fn(),
  updatePolicySettings: vi.fn(),
  recoverState: vi.fn(),
  setSchedulable: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: { list, getPolicySettings, updatePolicySettings, recoverState, setSchedulable }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const account = {
  id: 42,
  name: 'relay-account',
  platform: 'openai',
  type: 'apikey',
  status: 'error',
  schedulable: true,
  error_message: 'upstream unavailable',
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null
} as Account

const fullPolicy = () => ({
  account_id: 42,
  relay_failure_budget: {
    supported: true,
    enabled: true,
    window_minutes: 10,
    failure_threshold_percent: 30,
    min_requests: 10,
    consecutive_failures: 5,
    cooldown_minutes: 2
  },
  quota: { supported: true, total_limit: 20, daily_limit: 5, weekly_limit: 15 },
  scheduling_rate: { rate_multiplier: 0.8, sync_mode: 'auto_overwrite' as const }
})

function mountView() {
  return mount(AccountPolicySettingsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        Pagination: true
      }
    }
  })
}

describe('AccountPolicySettingsView', () => {
  beforeEach(() => {
    list.mockReset()
    getPolicySettings.mockReset()
    updatePolicySettings.mockReset()
    recoverState.mockReset()
    setSchedulable.mockReset()
    showError.mockReset()
    showSuccess.mockReset()

    list.mockResolvedValue({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
    getPolicySettings.mockResolvedValue(fullPolicy())
    updatePolicySettings.mockImplementation(async () => fullPolicy())
    recoverState.mockResolvedValue({ ...account, status: 'active', error_message: null })
    setSchedulable.mockImplementation(async (_id: number, value: boolean) => ({ ...account, schedulable: value }))
  })

  it('loads the first account and saves all supported policy sections together', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(list).toHaveBeenCalledWith(1, 20, { search: undefined, lite: '1' })
    expect(getPolicySettings).toHaveBeenCalledWith(42)
    expect((wrapper.get('[data-testid="account-policy-relay-window"]').element as HTMLInputElement).value).toBe('10')
    expect((wrapper.get('[data-testid="account-policy-quota-total"]').element as HTMLInputElement).value).toBe('20')

    await wrapper.get('[data-testid="account-policy-relay-window"]').setValue(25)
    await wrapper.get('[data-testid="account-policy-quota-total"]').setValue(100)
    await wrapper.get('[data-testid="account-policy-rate-manual"]').trigger('click')
    await wrapper.get('[data-testid="account-policy-rate"]').setValue(0.4)
    await wrapper.get('[data-testid="account-policy-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(updatePolicySettings).toHaveBeenCalledWith(42, {
      relay_failure_budget: {
        enabled: true,
        window_minutes: 25,
        failure_threshold_percent: 30,
        min_requests: 10,
        consecutive_failures: 5,
        cooldown_minutes: 2
      },
      quota: { total_limit: 100, daily_limit: 5, weekly_limit: 15 },
      scheduling_rate: { rate_multiplier: 0.4, sync_mode: 'manual_lock' }
    })
  })

  it('keeps state recovery and scheduling on their dedicated APIs', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="account-policy-recover"]').trigger('click')
    await flushPromises()
    expect(recoverState).toHaveBeenCalledWith(42)

    await wrapper.get('[data-testid="account-policy-schedulable"]').trigger('click')
    await flushPromises()
    expect(setSchedulable).toHaveBeenCalledWith(42, false)
  })

  it('omits unsupported policy sections from the save request', async () => {
    getPolicySettings.mockResolvedValue({
      ...fullPolicy(),
      relay_failure_budget: { ...fullPolicy().relay_failure_budget, supported: false },
      quota: { ...fullPolicy().quota, supported: false }
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="account-policy-relay-enabled"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-policy-quota-total"]').exists()).toBe(false)
    await wrapper.get('[data-testid="account-policy-form"]').trigger('submit.prevent')
    await flushPromises()

    expect(updatePolicySettings).toHaveBeenCalledWith(42, {
      scheduling_rate: { rate_multiplier: 0.8, sync_mode: 'auto_overwrite' }
    })
  })
})
