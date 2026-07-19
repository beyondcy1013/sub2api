import { describe, expect, it, vi } from 'vitest'

import { refreshAccountUsageInBatches } from '../batchAccountUsageRefresh'

describe('refreshAccountUsageInBatches', () => {
  it('returns an empty result without starting a worker for an empty scope', async () => {
    const query = vi.fn()

    const result = await refreshAccountUsageInBatches([], query)

    expect(query).not.toHaveBeenCalled()
    expect(result).toEqual({ successful: [], failed: [] })
  })

  it('runs at most four active usage queries concurrently and preserves input order', async () => {
    let active = 0
    let maxActive = 0
    const query = vi.fn(async (accountId: number) => {
      active += 1
      maxActive = Math.max(maxActive, active)
      await new Promise(resolve => setTimeout(resolve, 5))
      active -= 1
      return { accountId }
    })

    const result = await refreshAccountUsageInBatches(
      [1, 2, 3, 4, 5, 6, 7, 8, 9],
      query
    )

    expect(maxActive).toBe(4)
    expect(result.successful.map(item => item.accountId)).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9])
    expect(result.failed).toEqual([])
  })

  it('continues after individual failures and reports each failed account', async () => {
    const result = await refreshAccountUsageInBatches([1, 2, 3], async accountId => {
      if (accountId === 2) throw new Error('upstream unavailable')
      return { accountId }
    })

    expect(result.successful.map(item => item.accountId)).toEqual([1, 3])
    expect(result.failed).toHaveLength(1)
    expect(result.failed[0].accountId).toBe(2)
  })
})
