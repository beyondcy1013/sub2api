import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearPelicanResultCache,
  getCachedPelicanHistory,
  getCachedPelicanResult,
  setCachedPelicanResult,
  reloadPelicanResultCacheFromStorage
} from '../pelicanResultCache'

describe('pelicanResultCache', () => {
  beforeEach(() => {
    localStorage.clear()
    clearPelicanResultCache()
  })

  it('stores and persists successful pelican results', () => {
    setCachedPelicanResult({
      account: { id: 7, name: 'Cached Account', type: 'oauth' },
      status: 'success',
      result: {
        has_html: true,
        html: '<svg></svg>',
        downgraded: false,
        reason: 'ok',
        response_model: 'gpt-6-astra'
      },
      responseModel: 'gpt-6-astra',
      reason: 'ok',
      testedAt: 1234
    })

    expect(getCachedPelicanResult(7)?.account.name).toBe('Cached Account')
    expect(JSON.parse(localStorage.getItem('sub2api:account-pelican-results:v1') || '[]')).toHaveLength(1)
  })

  it('keeps the newest 100 entries', () => {
    for (let id = 1; id <= 101; id++) {
      setCachedPelicanResult({
        account: { id, name: `Account ${id}` },
        status: id === 1 ? 'failed' : 'success',
        testedAt: id
      })
    }

    expect(getCachedPelicanResult(1)).toBeUndefined()
    expect(getCachedPelicanResult(2)).toBeDefined()
    expect(getCachedPelicanResult(101)).toBeDefined()
  })

  it('keeps multiple historical results per account and orders them newest first', () => {
    setCachedPelicanResult({
      account: { id: 8, name: 'History Account' },
      status: 'success',
      responseModel: 'old-model',
      testedAt: 1000
    })
    setCachedPelicanResult({
      account: { id: 8, name: 'History Account' },
      status: 'downgraded',
      responseModel: 'new-model',
      reason: 'downgraded now',
      testedAt: 2000
    })

    const all = getCachedPelicanHistory(8)
    expect(all).toHaveLength(2)
    expect(all[0].responseModel).toBe('new-model')
    expect(all[1].responseModel).toBe('old-model')
    expect(getCachedPelicanResult(8)?.responseModel).toBe('new-model')
  })

  it('reads the previous single-result cache format', () => {
    localStorage.setItem(
      'sub2api:account-pelican-results:v1',
      JSON.stringify([
        {
          accountId: 9,
          accountName: 'Legacy Account',
          status: 'success',
          testedAt: 3000
        }
      ])
    )
    reloadPelicanResultCacheFromStorage()

    expect(getCachedPelicanHistory(9)[0]?.account.name).toBe('Legacy Account')
  })
})
