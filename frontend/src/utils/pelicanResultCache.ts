export interface PelicanTestResult {
  has_html: boolean
  html?: string
  downgraded: boolean
  reason: string
  response_model?: string
}

export type PelicanUserRating = 'accurate' | 'inaccurate'

export interface PelicanCachedResult {
  account: {
    id: number
    name: string
    type?: string
  }
  status: 'success' | 'downgraded' | 'failed'
  result?: PelicanTestResult
  responseModel?: string
  reason?: string
  error?: string
  elapsedMs?: number
  testedAt: number
  prompt?: string
  userRating?: PelicanUserRating
}

interface StoredPelicanResult extends Omit<PelicanCachedResult, 'account'> {
  accountId: number
  accountName: string
  accountType?: string
}

interface StoredAccountHistory {
  accountId: number
  results: StoredPelicanResult[]
}

const CACHE_STORAGE_KEY = 'sub2api:account-pelican-results:v1'
const MAX_CACHE_ENTRIES = 100
const MAX_HISTORY_PER_ACCOUNT = 20

const history = new Map<number, PelicanCachedResult[]>()

function toCachedResult(stored: StoredPelicanResult): PelicanCachedResult {
  return {
    account: {
      id: stored.accountId,
      name: stored.accountName || String(stored.accountId),
      type: stored.accountType
    },
    status: stored.status,
    result: stored.result,
    responseModel: stored.responseModel,
    reason: stored.reason,
    error: stored.error,
    elapsedMs: stored.elapsedMs,
    testedAt: stored.testedAt,
    prompt: stored.prompt,
    userRating: stored.userRating === 'accurate' || stored.userRating === 'inaccurate' ? stored.userRating : undefined
  }
}

function toStoredResult(item: PelicanCachedResult): StoredPelicanResult {
  return {
    accountId: item.account.id,
    accountName: item.account.name,
    accountType: item.account.type,
    status: item.status,
    result: item.result,
    responseModel: item.responseModel,
    reason: item.reason,
    error: item.error,
    elapsedMs: item.elapsedMs,
    testedAt: item.testedAt,
    prompt: item.prompt,
    userRating: item.userRating
  }
}

function loadHistory(rawResults: unknown[]): void {
  if (rawResults.length === 0) return
  const first = rawResults[0] as StoredPelicanResult & StoredAccountHistory

  if (Array.isArray(first?.results)) {
    for (const account of rawResults as StoredAccountHistory[]) {
      if (!account || typeof account.accountId !== 'number' || !Array.isArray(account.results)) continue
      history.set(
        account.accountId,
        account.results
          .filter(item => item && typeof item.accountId === 'number' && typeof item.testedAt === 'number')
          .map(toCachedResult)
      )
    }
    return
  }

  for (const stored of rawResults as StoredPelicanResult[]) {
    if (!stored || typeof stored.accountId !== 'number' || typeof stored.testedAt !== 'number') continue
    history.set(stored.accountId, [toCachedResult(stored)])
  }
}

function loadFromStorage(): void {
  try {
    const raw = localStorage.getItem(CACHE_STORAGE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw) as (StoredPelicanResult | StoredAccountHistory)[]
    if (!Array.isArray(parsed)) return

    loadHistory(parsed)
  } catch {
    // Ignore corrupted local cache.
  }
}

function persistToStorage(): void {
  try {
    const toStore: StoredAccountHistory[] = Array.from(history.entries())
      .sort((a, b) => (b[1][0]?.testedAt || 0) - (a[1][0]?.testedAt || 0))
      .slice(0, MAX_CACHE_ENTRIES)
      .map(([accountId, results]) => ({
        accountId,
        results: results.map(toStoredResult)
      }))
    localStorage.setItem(CACHE_STORAGE_KEY, JSON.stringify(toStore))
  } catch {
    // Ignore storage failures such as private browsing quotas.
  }
}

function trimToStorageLimit(): void {
  const entries = Array.from(history.entries()).sort(
    (a, b) => (b[1][0]?.testedAt || 0) - (a[1][0]?.testedAt || 0)
  )
  for (const [id] of entries.slice(MAX_CACHE_ENTRIES)) {
    history.delete(id)
  }
}

export function getCachedPelicanResult(accountId: number): PelicanCachedResult | undefined {
  return history.get(accountId)?.[0]
}

export function getCachedPelicanHistory(accountId: number): PelicanCachedResult[] {
  return [...(history.get(accountId) || [])]
}

export function getAllCachedPelicanResults(): PelicanCachedResult[] {
  return Array.from(history.values())
    .flat()
    .sort((a, b) => b.testedAt - a.testedAt)
}

export function setCachedPelicanResult(item: PelicanCachedResult): void {
  const results = history.get(item.account.id) || []
  const existingIndex = results.findIndex(entry => entry.testedAt === item.testedAt)
  if (existingIndex >= 0) {
    results.splice(existingIndex, 1)
  }
  results.unshift(item)
  results.sort((a, b) => b.testedAt - a.testedAt)
  history.set(item.account.id, results.slice(0, MAX_HISTORY_PER_ACCOUNT))
  trimToStorageLimit()
  persistToStorage()
}

export function setCachedPelicanUserRating(
  accountId: number,
  testedAt: number,
  userRating: PelicanUserRating
): PelicanCachedResult | undefined {
  const entry = history.get(accountId)?.find(item => item.testedAt === testedAt)
  if (!entry) return undefined
  entry.userRating = entry.userRating === userRating ? undefined : userRating
  persistToStorage()
  return { ...entry }
}

export function clearPelicanResultCache(): void {
  history.clear()
  try {
    localStorage.removeItem(CACHE_STORAGE_KEY)
  } catch {
    // Ignore storage failures.
  }
}

export function reloadPelicanResultCacheFromStorage(): void {
  history.clear()
  loadFromStorage()
}

reloadPelicanResultCacheFromStorage()
