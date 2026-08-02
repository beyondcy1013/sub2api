import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, put }
}))

import { getPolicySettings, updatePolicySettings } from '@/api/admin/accounts'

describe('admin account policy settings API', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('uses the dedicated per-account policy endpoints', async () => {
    const policy = {
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
      quota: { supported: true, total_limit: 0, daily_limit: 0, weekly_limit: 0 },
      scheduling_rate: { rate_multiplier: 0.8, sync_mode: 'manual_lock' }
    }
    const update = {
      relay_failure_budget: {
        enabled: true,
        window_minutes: 20,
        failure_threshold_percent: 25,
        min_requests: 12,
        consecutive_failures: 4,
        cooldown_minutes: 3
      },
      quota: { total_limit: 100, daily_limit: 10, weekly_limit: 50 },
      scheduling_rate: { rate_multiplier: 0.5, sync_mode: 'manual_lock' as const }
    }
    get.mockResolvedValueOnce({ data: policy })
    put.mockResolvedValueOnce({ data: policy })

    await expect(getPolicySettings(42)).resolves.toEqual(policy)
    await expect(updatePolicySettings(42, update)).resolves.toEqual(policy)

    expect(get).toHaveBeenCalledWith('/admin/accounts/42/policy-settings')
    expect(put).toHaveBeenCalledWith('/admin/accounts/42/policy-settings', update)
  })
})
