import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SchedulingRateModal from '../SchedulingRateModal.vue'
import type { Account } from '@/types'

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
  it('offers follow-upstream and manual overwrite choices when rates conflict', async () => {
    const wrapper = mount(SchedulingRateModal, {
      props: { show: true, account, upstreamRate: 0.35, upstreamKnown: true, conflict: true },
      global: { stubs: { BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' } } }
    })

    expect(wrapper.get('[data-testid="scheduling-rate-conflict"]').exists()).toBe(true)
    await wrapper.get('[data-testid="scheduling-rate-source-upstream"]').setValue(true)
    await wrapper.get('[data-testid="scheduling-rate-save"]').trigger('click')
    expect(wrapper.emitted('save')?.[0]?.[0]).toEqual({ source: 'upstream' })
  })
})

