import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SchedulingRateCell from '../SchedulingRateCell.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const account = (overrides: Partial<Account> = {}): Account => ({
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
  session_window_status: null,
  ...overrides
})

describe('SchedulingRateCell', () => {
  it('shows the effective multiplier, source, and edit action', async () => {
    const wrapper = mount(SchedulingRateCell, { props: { account: account() } })

    expect(wrapper.get('[data-testid="scheduling-rate-value"]').text()).toContain('0.8x')
    expect(wrapper.get('[data-testid="scheduling-rate-source"]').text()).toContain('manualLock')
    await wrapper.get('[data-testid="scheduling-rate-edit"]').trigger('click')
    expect(wrapper.emitted('manage')).toHaveLength(1)
  })

  it('always shows the persisted scheduling multiplier in automatic mode', () => {
    const wrapper = mount(SchedulingRateCell, {
      props: { account: account({ scheduling_rate_sync_mode: 'auto_overwrite', scheduling_rate_multiplier: 0.8 }) }
    })

    expect(wrapper.get('[data-testid="scheduling-rate-value"]').text()).toContain('0.8x')
    expect(wrapper.get('[data-testid="scheduling-rate-source"]').text()).toContain('autoOverwrite')
  })

  it('highlights an available live lowest-rate account in gold', () => {
    const wrapper = mount(SchedulingRateCell, {
      props: { account: account({ scheduling_rate_optimal: true, scheduling_liveness_status: 'alive' }) }
    })

    expect(wrapper.get('[data-testid="scheduling-rate-value"]').classes()).toContain('text-amber-500')
    expect(wrapper.get('[data-testid="scheduling-rate-optimal"]').text()).toContain('optimal')
  })

  it('keeps a non-optimal rate neutral', () => {
    const wrapper = mount(SchedulingRateCell, {
      props: { account: account({ scheduling_rate_optimal: false, scheduling_liveness_status: 'alive' }) }
    })

    expect(wrapper.get('[data-testid="scheduling-rate-value"]').classes()).toContain('text-gray-800')
    expect(wrapper.find('[data-testid="scheduling-rate-optimal"]').exists()).toBe(false)
  })
})
