import type { AdminDataAccount, AdminDataPayload, AccountPlatform } from '@/types'

export type EnhancedImportErrorCode =
  | 'invalid_json'
  | 'unsupported_format'
  | 'unsupported_provider'
  | 'missing_credentials'

export class EnhancedImportError extends Error {
  constructor(
    public readonly code: EnhancedImportErrorCode,
    public readonly sourceName: string,
    public readonly provider?: string
  ) {
    super(code)
    this.name = 'EnhancedImportError'
  }
}

const SUPPORTED_DATA_TYPES = new Set(['sub2api-data', 'sub2api-bundle'])
const SUPPORTED_DATA_VERSION = 1

const PROVIDER_CONFIG: Record<string, { platform: AccountPlatform; label: string }> = {
  codex: { platform: 'openai', label: 'Codex' },
  claude: { platform: 'anthropic', label: 'Claude' },
  gemini: { platform: 'gemini', label: 'Gemini' },
  antigravity: { platform: 'antigravity', label: 'Antigravity' }
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === 'object' && !Array.isArray(value)

const nonEmptyString = (value: unknown): string =>
  typeof value === 'string' ? value.trim() : ''

const isNativePayload = (value: unknown): value is Record<string, unknown> =>
  isRecord(value) && Array.isArray(value.proxies) && Array.isArray(value.accounts)

const validateNativePayload = (
  value: Record<string, unknown>,
  sourceName: string
): AdminDataPayload => {
  const dataType = nonEmptyString(value.type)
  if (dataType && !SUPPORTED_DATA_TYPES.has(dataType)) {
    throw new EnhancedImportError('unsupported_format', sourceName)
  }
  if (
    value.version !== undefined &&
    value.version !== 0 &&
    value.version !== SUPPORTED_DATA_VERSION
  ) {
    throw new EnhancedImportError('unsupported_format', sourceName)
  }
  return value as unknown as AdminDataPayload
}

const copyCredential = (
  credentials: Record<string, unknown>,
  source: Record<string, unknown>,
  key: string
) => {
  const value = source[key]
  if (value !== undefined && value !== null && value !== '') {
    credentials[key] = value
  }
}

const cliCredentials = (
  source: Record<string, unknown>,
  provider: string,
  sourceName: string
): Record<string, unknown> => {
  const token = isRecord(source.token) ? source.token : {}
  const credentials: Record<string, unknown> = {}
  const tokenSource = provider === 'gemini' ? token : source

  for (const key of [
    'access_token',
    'refresh_token',
    'id_token',
    'token_type',
    'scope',
    'email',
    'project_id',
    'client_id',
    'client_secret',
    'scopes',
    'universe_domain',
    'plan_type',
    'subscription_expires_at'
  ]) {
    copyCredential(credentials, tokenSource, key)
    if (credentials[key] === undefined) {
      copyCredential(credentials, source, key)
    }
  }

  const expiry =
    tokenSource.expires_at ??
    tokenSource.expiry ??
    tokenSource.expired ??
    source.expires_at ??
    source.expiry ??
    source.expired
  if (expiry !== undefined && expiry !== null && expiry !== '') {
    credentials.expires_at = expiry
  }

  if (provider === 'codex') {
    const accountID = nonEmptyString(source.account_id) || nonEmptyString(source.chatgpt_account_id)
    if (accountID) credentials.chatgpt_account_id = accountID
  }
  if (provider === 'gemini') {
    credentials.oauth_type = nonEmptyString(source.oauth_type) || 'code_assist'
  }

  if (!nonEmptyString(credentials.access_token) && !nonEmptyString(credentials.refresh_token)) {
    throw new EnhancedImportError('missing_credentials', sourceName, provider)
  }
  return credentials
}

const fallbackAccountName = (sourceName: string, providerLabel: string, index: number): string => {
  const fileBase = sourceName.replace(/\.json$/i, '').trim()
  if (fileBase && fileBase.toLowerCase() !== 'pasted json') {
    return index > 0 ? `${fileBase} #${index + 1}` : fileBase
  }
  return `${providerLabel} imported${index > 0 ? ` #${index + 1}` : ''}`
}

const convertCLIAuth = (
  value: Record<string, unknown>,
  sourceName: string,
  index: number
): AdminDataAccount => {
  const provider = nonEmptyString(value.type).toLowerCase()
  const config = PROVIDER_CONFIG[provider]
  if (!config) {
    throw new EnhancedImportError('unsupported_provider', sourceName, provider)
  }

  const email = nonEmptyString(value.email)
  return {
    name: email || fallbackAccountName(sourceName, config.label, index),
    platform: config.platform,
    type: 'oauth',
    credentials: cliCredentials(value, provider, sourceName),
    concurrency: 4,
    priority: 1
  }
}

const normalizeParsedValue = (value: unknown, sourceName: string): AdminDataPayload => {
  if (isNativePayload(value)) return validateNativePayload(value, sourceName)

  const authItems = Array.isArray(value) ? value : [value]
  if (authItems.length === 0 || authItems.some(item => !isRecord(item))) {
    throw new EnhancedImportError('unsupported_format', sourceName)
  }

  return {
    type: 'sub2api-data',
    version: 1,
    exported_at: new Date().toISOString(),
    proxies: [],
    accounts: authItems.map((item, index) => convertCLIAuth(item as Record<string, unknown>, sourceName, index))
  }
}

export const parseEnhancedImportSource = (
  text: string,
  sourceName: string
): AdminDataPayload => {
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    throw new EnhancedImportError('invalid_json', sourceName)
  }
  return normalizeParsedValue(parsed, sourceName)
}

const matchingCloser = (opener: string): string => opener === '{' ? '}' : ']'

const findCompleteJsonValueEnd = (text: string, start: number): number | null => {
  const stack: string[] = []
  let inString = false
  let escaped = false

  for (let index = start; index < text.length; index += 1) {
    const character = text[index]
    if (inString) {
      if (escaped) escaped = false
      else if (character === '\\') escaped = true
      else if (character === '"') inString = false
      continue
    }

    if (character === '"') {
      inString = true
    } else if (character === '{' || character === '[') {
      stack.push(character)
    } else if (character === '}' || character === ']') {
      const opener = stack.pop()
      if (!opener || matchingCloser(opener) !== character) return null
      if (stack.length === 0) return index
    }
  }
  return null
}

export const extractEnhancedImportJsonSources = (text: string): string[] => {
  const trimmed = text.trim()
  if (!trimmed) return []

  try {
    JSON.parse(trimmed)
    return [trimmed]
  } catch {
    // Mixed chat and Markdown input is handled by the depth-aware scanner below.
  }

  const sources: string[] = []
  for (let start = 0; start < text.length; start += 1) {
    const opener = text[start]
    if (opener !== '{' && opener !== '[') continue

    const end = findCompleteJsonValueEnd(text, start)
    if (end === null) return []
    const candidate = text.slice(start, end + 1)
    start = end
    try {
      JSON.parse(candidate)
      sources.push(candidate)
    } catch {
      // Ordinary prose may contain balanced braces. Skip that whole balanced
      // range so nested braces are not misidentified as independent JSON.
    }
  }
  return sources
}

export const parseEnhancedImportText = (
  text: string,
  sourceName = 'pasted JSON'
): AdminDataPayload[] => {
  const sources = extractEnhancedImportJsonSources(text)
  if (sources.length === 0) throw new EnhancedImportError('invalid_json', sourceName)

  const isWholeJson = sources.length === 1 && sources[0] === text.trim()
  return sources.map((source, index) =>
    parseEnhancedImportSource(source, isWholeJson ? sourceName : `${sourceName} #${index + 1}`)
  )
}

export const mergeEnhancedImportPayloads = (
  payloads: AdminDataPayload[]
): AdminDataPayload => {
  if (payloads.length === 1 && payloads[0]) return payloads[0]

  return {
    type: 'sub2api-data',
    version: 1,
    exported_at: new Date().toISOString(),
    proxies: payloads.flatMap(payload => payload.proxies),
    accounts: payloads.flatMap(payload => payload.accounts),
    skipped_shadows: payloads.reduce((sum, payload) => {
      const count = Number(payload.skipped_shadows || 0)
      return Number.isFinite(count) ? sum + count : sum
    }, 0)
  }
}
