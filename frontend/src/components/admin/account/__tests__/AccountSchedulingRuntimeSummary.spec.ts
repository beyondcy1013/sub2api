import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AccountSchedulingRuntimeSummary from '../AccountSchedulingRuntimeSummary.vue'

const getSuperPriority = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin', () => ({
  adminAPI: {
    superPriority: { get: getSuperPriority }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key
  })
}))

describe('AccountSchedulingRuntimeSummary', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-27T12:00:00Z'))
    getSuperPriority.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows the live countdown and latest loop-test result', async () => {
    getSuperPriority.mockResolvedValue({
      check_interval: '@every 2m',
      liveness_runtime: {
        enabled: true,
        running: false,
        next_run_at: '2026-07-27T12:01:30Z',
        last_run: {
          trigger: 'scheduled',
          started_at: '2026-07-27T11:59:50Z',
          finished_at: '2026-07-27T11:59:55Z',
          result: { checked: 4, succeeded: 2, failed: 1, skipped: 1 }
        }
      }
    })

    const wrapper = mount(AccountSchedulingRuntimeSummary)
    await flushPromises()

    expect(wrapper.get('[data-testid="account-scheduling-runtime-state"]').text())
      .toContain('admin.accounts.schedulingRules.runtimeCountdown')
    expect(wrapper.get('[data-testid="account-scheduling-runtime-state"]').text())
      .toContain('"duration":"00:01:30"')
    expect(wrapper.get('[data-testid="account-scheduling-runtime-progress"]').attributes('aria-valuenow'))
      .toBe('25')
    expect(wrapper.get('[data-testid="account-scheduling-runtime-result"]').text())
      .toContain('"succeeded":2,"failed":1,"skipped":1')

    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.get('[data-testid="account-scheduling-runtime-state"]').text())
      .toContain('"duration":"00:01:29"')
    wrapper.unmount()
  })

  it('keeps an explicit zero countdown while a due probe is waiting to start', async () => {
    getSuperPriority.mockResolvedValue({
      check_interval: '@every 1m',
      liveness_runtime: {
        enabled: true,
        running: false,
        next_run_at: '2026-07-27T11:59:55Z'
      }
    })

    const wrapper = mount(AccountSchedulingRuntimeSummary)
    await flushPromises()

    expect(wrapper.get('[data-testid="account-scheduling-runtime-state"]').text())
      .toContain('admin.accounts.schedulingRules.runtimeDueCountdown')
    expect(wrapper.get('[data-testid="account-scheduling-runtime-state"]').text())
      .toContain('"duration":"00:00:00"')
    expect(wrapper.get('[data-testid="account-scheduling-runtime-progress"]').attributes('aria-valuenow'))
      .toBe('100')
    wrapper.unmount()
  })

  it('polls the server and exposes a running state without reloading the account table', async () => {
    getSuperPriority
      .mockResolvedValueOnce({
        liveness_runtime: {
          enabled: true,
          running: false,
          next_run_at: '2026-07-27T12:01:30Z'
        }
      })
      .mockResolvedValueOnce({
        liveness_runtime: {
          enabled: true,
          running: true,
          next_run_at: '2026-07-27T12:01:30Z'
        }
      })

    const wrapper = mount(AccountSchedulingRuntimeSummary)
    await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()

    expect(getSuperPriority).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-testid="account-scheduling-runtime-state"]').text())
      .toBe('admin.accounts.schedulingRules.runtimeRunning')
    wrapper.unmount()
  })
})
