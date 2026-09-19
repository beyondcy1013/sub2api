import { vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it } from 'vitest'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

function makeAccount(overrides: Partial<Account>): Account {
  return {
    id: 1,
    name: 'test-account',
    platform: 'openai',
    type: 'oauth',
    proxy_id: null,
    concurrency: 3,
    priority: 50,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

const position = { top: 0, left: 0 }
const bodyText = () => document.body.textContent ?? ''

describe('AccountActionMenu state protection', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('shows disable label for a protected OpenAI OAuth parent and hides it for API keys', () => {
    const wrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount({ extra: { state_protection_enabled: true } }),
        position
      },
      attachTo: document.body,
      global: { plugins: [createPinia()] }
    })
    expect(bodyText()).toContain('admin.accounts.stateProtection.disable')
    wrapper.unmount()

    const apiKeyWrapper = mount(AccountActionMenu, {
      props: {
        show: true,
        account: makeAccount({ type: 'apikey' }),
        position
      },
      attachTo: document.body,
      global: { plugins: [createPinia()] }
    })
    expect(bodyText()).not.toContain('admin.accounts.stateProtection.disable')
    apiKeyWrapper.unmount()
  })
})
