import { describe, expect, it } from 'vitest'

import {
  formatUsageWindowReset,
  formatUsageWindowUtilization
} from '../usageWindowDisplay'

describe('usage window table display', () => {
  const now = Date.parse('2026-07-19T04:00:00Z')
  const labels = { now: '现在', pending: '待刷新' }

  it('formats utilization as a compact percentage', () => {
    expect(formatUsageWindowUtilization(93)).toBe('93%')
    expect(formatUsageWindowUtilization(1001)).toBe('>999%')
    expect(formatUsageWindowUtilization(Number.NaN)).toBe('-')
  })

  it('formats a future reset as the remaining rolling-window duration', () => {
    expect(formatUsageWindowReset({
      utilization: 93,
      resetsAt: '2026-07-25T07:28:00Z',
      now,
      labels
    })).toBe('6d 3h')
  })

  it('shows an idle OpenAI rolling window as available now', () => {
    expect(formatUsageWindowReset({
      utilization: 0,
      resetsAt: '2026-07-25T07:28:00Z',
      now,
      labels,
      showNowWhenIdle: true
    })).toBe('现在')
  })

  it('distinguishes stale positive usage from an idle expired window', () => {
    expect(formatUsageWindowReset({
      utilization: 93,
      resetsAt: '2026-07-19T03:00:00Z',
      now,
      labels
    })).toBe('待刷新')
    expect(formatUsageWindowReset({
      utilization: 0,
      resetsAt: '2026-07-19T03:00:00Z',
      now,
      labels
    })).toBe('现在')
  })
})
