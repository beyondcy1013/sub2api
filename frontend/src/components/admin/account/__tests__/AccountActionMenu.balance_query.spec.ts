import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: {} }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

function account(type: 'apikey' | 'oauth'): Account {
  return {
    id: 1,
    name: 'relay-account',
    platform: 'openai',
    type,
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
  }
}

describe('AccountActionMenu balance query', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('shows the action for API key accounts and emits query-balance', async () => {
    const target = account('apikey')
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account: target, position: { top: 100, left: 100 } },
      attachTo: document.body,
    })

    const button = Array.from(document.body.querySelectorAll('button'))
      .find(item => item.textContent?.includes('admin.accounts.balanceQuery.action'))
    expect(button).toBeDefined()
    button!.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('query-balance')?.[0]).toEqual([target])
    wrapper.unmount()
  })

  it('hides the action for OAuth accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      props: { show: true, account: account('oauth'), position: { top: 100, left: 100 } },
      attachTo: document.body,
    })

    expect(document.body.textContent).not.toContain('admin.accounts.balanceQuery.action')
    wrapper.unmount()
  })
})
