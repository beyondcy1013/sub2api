import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearPelicanResultCache,
  getAllCachedPelicanResults,
  getCachedPelicanHistory,
  getCachedPelicanResult,
  setCachedPelicanResult,
  setCachedPelicanUserRating,
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
      testedAt: 1234,
      output: 'Raw response with HTML',
      prompt: 'Create a pelican animation'
    })

    reloadPelicanResultCacheFromStorage()
    expect(getCachedPelicanResult(7)?.output).toBe('Raw response with HTML')
    expect(getCachedPelicanResult(7)?.account.name).toBe('Cached Account')
    expect(getCachedPelicanResult(7)?.prompt).toBe('Create a pelican animation')
    expect(getAllCachedPelicanResults().map(item => item.account.id)).toEqual([7])
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

  it('stores and toggles a user rating for an exact cached run', () => {
    setCachedPelicanResult({
      account: { id: 12, name: 'Rated Account' },
      status: 'success',
      prompt: 'Create a pelican',
      testedAt: 5000
    })

    expect(setCachedPelicanUserRating(12, 5000, 'accurate')?.userRating).toBe('accurate')
    expect(getCachedPelicanResult(12)?.userRating).toBe('accurate')
    expect(setCachedPelicanUserRating(12, 5000, 'inaccurate')?.userRating).toBe('inaccurate')
    expect(getCachedPelicanResult(12)?.userRating).toBe('inaccurate')
    expect(setCachedPelicanUserRating(12, 5000, 'inaccurate')?.userRating).toBeUndefined()
    expect(getCachedPelicanResult(12)?.userRating).toBeUndefined()
  })

  it('读取旧缓存时以能力判定为准，已完成且未降智的条目不再显示为失败', () => {
    localStorage.setItem('sub2api:account-pelican-results:v1', JSON.stringify([
      {
        accountId: 308,
        results: [
          { accountId: 308, accountName: 'WorkBuddy2apiRs18371', accountType: 'apikey', testMode: 'pelican', status: 'failed',
            result: { has_html: true, html: '<svg></svg>', downgraded: false, reason: '测智通过（未降智，已生成完整动画）', response_model: 'deepseek-v4.1-flash' },
            responseModel: 'deepseek-v4.1-flash', reason: '测智通过（未降智，已生成完整动画）', testedAt: 2000 },
          { accountId: 308, accountName: 'WorkBuddy2apiRs18371', accountType: 'apikey', testMode: 'pelican', status: 'failed',
            result: { has_html: true, html: '<svg></svg>', downgraded: true, reason: '疑似降智', response_model: 'deepseek-v4.1-flash' },
            responseModel: 'deepseek-v4.1-flash', reason: '疑似降智', testedAt: 1000 },
          { accountId: 308, accountName: 'WorkBuddy2apiRs18371', accountType: 'apikey', testMode: 'pelican', status: 'failed',
            result: { has_html: false, downgraded: false, reason: 'connection refused', response_model: '' },
            reason: 'connection refused', testedAt: 500 }
        ]
      }
    ]))
    reloadPelicanResultCacheFromStorage()

    const [passed, downgraded, broken] = getCachedPelicanHistory(308)
    expect(passed.status).toBe('success')
    expect(downgraded.status).toBe('downgraded')
    expect(broken.status).toBe('failed')
  })
})
