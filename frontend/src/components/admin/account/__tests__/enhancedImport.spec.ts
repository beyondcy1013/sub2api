import { describe, expect, it } from 'vitest'
import {
  EnhancedImportError,
  mergeEnhancedImportPayloads,
  parseEnhancedImportText,
  parseEnhancedImportSource
} from '../enhancedImport'

describe('enhanced account import normalization', () => {
  it('converts a CLIProxyAPI codex auth file into an OpenAI OAuth account', () => {
    const payload = parseEnhancedImportSource(
      JSON.stringify({
        type: 'codex',
        email: 'codex@example.com',
        account_id: 'acct-123',
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        id_token: 'id-token',
        expired: '2026-08-01T00:00:00Z'
      }),
      'codex-auth.json'
    )

    expect(payload.proxies).toEqual([])
    expect(payload.accounts).toEqual([
      expect.objectContaining({
        name: 'codex@example.com',
        platform: 'openai',
        type: 'oauth',
        concurrency: 4,
        priority: 1,
        credentials: expect.objectContaining({
          access_token: 'access-token',
          refresh_token: 'refresh-token',
          id_token: 'id-token',
          chatgpt_account_id: 'acct-123',
          expires_at: '2026-08-01T00:00:00Z'
        })
      })
    ])
    expect(payload.accounts[0]?.credentials).not.toHaveProperty('type')
    expect(payload.accounts[0]?.credentials).not.toHaveProperty('account_id')
  })

  it('converts CLIProxyAPI claude and antigravity auth objects from a JSON array', () => {
    const payload = parseEnhancedImportSource(
      JSON.stringify([
        {
          type: 'claude',
          email: 'claude@example.com',
          access_token: 'claude-access',
          refresh_token: 'claude-refresh',
          expired: '2026-08-02T00:00:00Z'
        },
        {
          type: 'antigravity',
          email: 'gravity@example.com',
          access_token: 'gravity-access',
          refresh_token: 'gravity-refresh',
          project_id: 'gravity-project',
          expired: '2026-08-03T00:00:00Z'
        }
      ]),
      'oauth-accounts.json'
    )

    expect(payload.accounts).toHaveLength(2)
    expect(payload.accounts[0]).toEqual(expect.objectContaining({
      name: 'claude@example.com',
      platform: 'anthropic',
      type: 'oauth'
    }))
    expect(payload.accounts[0]?.credentials).toEqual(expect.objectContaining({
      access_token: 'claude-access',
      refresh_token: 'claude-refresh',
      expires_at: '2026-08-02T00:00:00Z'
    }))
    expect(payload.accounts[1]).toEqual(expect.objectContaining({
      name: 'gravity@example.com',
      platform: 'antigravity',
      type: 'oauth'
    }))
    expect(payload.accounts[1]?.credentials).toEqual(expect.objectContaining({
      project_id: 'gravity-project',
      expires_at: '2026-08-03T00:00:00Z'
    }))
  })

  it('flattens the nested OAuth token from a CLIProxyAPI Gemini auth file', () => {
    const payload = parseEnhancedImportSource(
      JSON.stringify({
        type: 'gemini',
        email: 'gemini@example.com',
        project_id: 'gemini-project',
        token: {
          access_token: 'gemini-access',
          refresh_token: 'gemini-refresh',
          token_type: 'Bearer',
          scope: 'scope-a scope-b',
          expiry: '2026-08-04T00:00:00Z'
        }
      }),
      'gemini-auth.json'
    )

    expect(payload.accounts[0]).toEqual(expect.objectContaining({
      name: 'gemini@example.com',
      platform: 'gemini',
      type: 'oauth',
      credentials: expect.objectContaining({
        access_token: 'gemini-access',
        refresh_token: 'gemini-refresh',
        token_type: 'Bearer',
        scope: 'scope-a scope-b',
        project_id: 'gemini-project',
        oauth_type: 'code_assist',
        expires_at: '2026-08-04T00:00:00Z'
      })
    }))
  })

  it('accepts native sub2api export JSON text unchanged', () => {
    const nativePayload = {
      type: 'sub2api-data',
      version: 1,
      exported_at: '2026-07-19T00:00:00Z',
      proxies: [{ proxy_key: 'proxy-1' }],
      accounts: [{ name: 'native-account' }]
    }

    expect(parseEnhancedImportSource(JSON.stringify(nativePayload), 'pasted JSON')).toEqual(nativePayload)
  })

  it('merges native exports and CLIProxyAPI auth sources into one payload', () => {
    const merged = mergeEnhancedImportPayloads([
      parseEnhancedImportSource(JSON.stringify({
        exported_at: '2026-07-19T00:00:00Z',
        proxies: [],
        accounts: [{ name: 'native-account' }]
      }), 'native.json'),
      parseEnhancedImportSource(JSON.stringify({
        type: 'codex',
        email: 'codex@example.com',
        refresh_token: 'refresh-token'
      }), 'codex.json')
    ])

    expect(merged.accounts.map(account => account.name)).toEqual([
      'native-account',
      'codex@example.com'
    ])
  })

  it('reports invalid JSON, unsupported providers, and missing OAuth credentials', () => {
    expect(() => parseEnhancedImportSource('{', 'broken.json')).toThrowError(
      expect.objectContaining<Partial<EnhancedImportError>>({ code: 'invalid_json' })
    )
    expect(() => parseEnhancedImportSource('{"type":"kimi","access_token":"token"}', 'kimi.json')).toThrowError(
      expect.objectContaining<Partial<EnhancedImportError>>({ code: 'unsupported_provider' })
    )
    expect(() => parseEnhancedImportSource('{"type":"codex","email":"empty@example.com"}', 'empty.json')).toThrowError(
      expect.objectContaining<Partial<EnhancedImportError>>({ code: 'missing_credentials' })
    )
  })

  it('extracts complete native exports from mixed chat and Markdown text', () => {
    const payloads = parseEnhancedImportText([
      'image 1',
      JSON.stringify({
        exported_at: '2026-07-19T00:00:00Z',
        proxies: [],
        accounts: [{ name: 'one', extra: { note: 'brace } and [ inside string' } }]
      }),
      'image 2',
      '```json',
      JSON.stringify({
        exported_at: '2026-07-19T00:00:01Z',
        proxies: [],
        accounts: [{ name: 'two', extra: { escaped: '\"quoted\" \\ slash' } }]
      }),
      '```',
      'usage guide',
      JSON.stringify({
        exported_at: '2026-07-19T00:00:02Z',
        proxies: [],
        accounts: [{ name: 'three' }]
      })
    ].join('\n'), 'pasted JSON')

    expect(payloads).toHaveLength(3)
    expect(mergeEnhancedImportPayloads(payloads).accounts.map(account => account.name)).toEqual([
      'one',
      'two',
      'three'
    ])
  })

  it('extracts a CLI auth array beside a native export', () => {
    const payloads = parseEnhancedImportText([
      'account file',
      JSON.stringify([{
        type: 'codex',
        email: 'array@example.com',
        refresh_token: 'fake-refresh-token'
      }]),
      'native export',
      JSON.stringify({ proxies: [], accounts: [{ name: 'native' }] })
    ].join('\n'), 'pasted JSON')

    expect(mergeEnhancedImportPayloads(payloads).accounts).toEqual([
      expect.objectContaining({ name: 'array@example.com', concurrency: 4 }),
      expect.objectContaining({ name: 'native' })
    ])
  })

  it('keeps pure JSON text as one source', () => {
    const payloads = parseEnhancedImportText(
      JSON.stringify({ proxies: [], accounts: [{ name: 'pure-json' }] }),
      'pasted JSON'
    )

    expect(payloads).toHaveLength(1)
    expect(payloads[0]?.accounts[0]?.name).toBe('pure-json')
  })

  it('rejects mixed text without a complete JSON value', () => {
    for (const text of ['usage guide without JSON', 'image 1\n{\"proxies\":[],\"accounts\":[']) {
      expect(() => parseEnhancedImportText(text, 'pasted JSON')).toThrowError(
        expect.objectContaining<Partial<EnhancedImportError>>({ code: 'invalid_json' })
      )
    }
  })

  it('identifies the failing segment when a later extracted JSON is unsupported', () => {
    const mixed = [
      JSON.stringify({ proxies: [], accounts: [{ name: 'valid' }] }),
      'image 2',
      JSON.stringify({ type: 'unsupported-provider', access_token: 'fake-token' })
    ].join('\n')

    expect(() => parseEnhancedImportText(mixed, 'pasted JSON')).toThrowError(
      expect.objectContaining<Partial<EnhancedImportError>>({
        code: 'unsupported_provider',
        sourceName: 'pasted JSON #2'
      })
    )
  })
})
