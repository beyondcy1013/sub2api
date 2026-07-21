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
    expect(wrapper.get('[data-testid="scheduling-rate-source"]').text()).toContain('manual')
    await wrapper.get('[data-testid="scheduling-rate-edit"]').trigger('click')
    expect(wrapper.emitted('manage')).toHaveLength(1)
  })

  it('makes an upstream-following unknown rate explicit', () => {
    const wrapper = mount(SchedulingRateCell, {
      props: { account: account({ scheduling_rate_known: false, scheduling_rate_multiplier: undefined, scheduling_rate_source: 'upstream' }) }
    })

    expect(wrapper.get('[data-testid="scheduling-rate-value"]').text()).toContain('?')
    expect(wrapper.get('[data-testid="scheduling-rate-source"]').text()).toContain('upstream')
  })
})
