import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AccountBalanceQueryModal from '../AccountBalanceQueryModal.vue'
import type { Account } from '@/types'

const api = vi.hoisted(() => ({
  queryAccountBalance: vi.fn(),
  updateAccountBalanceQueryConfig: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { accounts: api },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const account = {
  id: 7,
  name: 'relay-account',
  platform: 'openai',
  type: 'apikey',
  credentials: { base_url: 'https://relay.example/v1' },
  extra: {
    balance_query: {
      scheme: 'newapi',
      api_url: '',
    },
  },
} as unknown as Account

describe('AccountBalanceQueryModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.queryAccountBalance.mockResolvedValue({
      account_id: account.id,
      success: true,
      scheme: 'newapi',
      api_url: 'https://relay.example/api/usage/token/',
      balance: 6250000,
      unit: 'quota',
      queried_at: '2026-07-25T00:00:00Z',
      attempts: [],
    })
    api.updateAccountBalanceQueryConfig.mockResolvedValue({ scheme: 'custom', api_url: '/balance' })
  })

  it('打开后立即使用已记忆方案查询并显示余额', async () => {
    const wrapper = mount(AccountBalanceQueryModal, {
      props: { show: true, account },
      attachTo: document.body,
      global: { stubs: { Teleport: true } },
    })
    await flushPromises()

    expect(api.queryAccountBalance).toHaveBeenCalledWith(account.id)
    expect(wrapper.text()).toContain('6250000')
    expect(wrapper.text()).toContain('quota')
    expect((wrapper.emitted('updated')?.[0]?.[0] as Account).extra?.balance_query).toMatchObject({
      scheme: 'newapi',
      detected_api_url: 'https://relay.example/api/usage/token/',
    })
    wrapper.unmount()
  })

  it('同一账号的探测结果回写后不会重复自动查询', async () => {
    const wrapper = mount(AccountBalanceQueryModal, {
      props: { show: true, account },
      attachTo: document.body,
      global: { stubs: { Teleport: true } },
    })
    await flushPromises()

    expect(api.queryAccountBalance).toHaveBeenCalledTimes(1)

    await wrapper.setProps({
      account: {
        ...account,
        extra: {
          ...account.extra,
          balance_query: {
            scheme: 'newapi',
            detected_api_url: 'https://relay.example/api/usage/token/',
          },
        },
      } as unknown as Account,
    })
    await flushPromises()

    expect(api.queryAccountBalance).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('查询前保存自定义 API 配置', async () => {
    const wrapper = mount(AccountBalanceQueryModal, {
      props: { show: true, account },
      attachTo: document.body,
      global: { stubs: { Teleport: true } },
    })
    await flushPromises()
    api.queryAccountBalance.mockClear()

    await wrapper.get('[data-testid="balance-query-scheme"]').setValue('custom')
    await wrapper.get('[data-testid="balance-query-api-url"]').setValue('/balance')
    await wrapper.get('[data-testid="balance-query-submit"]').trigger('click')
    await flushPromises()

    expect(api.updateAccountBalanceQueryConfig).toHaveBeenCalledWith(account.id, {
      scheme: 'custom',
      api_url: '/balance',
    })
    expect(api.queryAccountBalance).toHaveBeenCalledWith(account.id)
    wrapper.unmount()
  })

  it('可配置 signIn 浏览器站点作为通用回退', async () => {
    api.updateAccountBalanceQueryConfig.mockResolvedValue({
      scheme: 'signin',
      api_url: '',
      sign_in_site_id: '32b00162-427c-41d9-8325-faa4dcc0f3a3',
    })
    const wrapper = mount(AccountBalanceQueryModal, {
      props: { show: true, account },
      attachTo: document.body,
      global: { stubs: { Teleport: true } },
    })
    await flushPromises()
    api.queryAccountBalance.mockClear()

    await wrapper.get('[data-testid="balance-query-scheme"]').setValue('signin')
    await wrapper.get('[data-testid="balance-query-signin-site-id"]').setValue('32b00162-427c-41d9-8325-faa4dcc0f3a3')
    await wrapper.get('[data-testid="balance-query-submit"]').trigger('click')
    await flushPromises()

    expect(api.updateAccountBalanceQueryConfig).toHaveBeenCalledWith(account.id, {
      scheme: 'signin',
      api_url: '',
      sign_in_site_id: '32b00162-427c-41d9-8325-faa4dcc0f3a3',
    })
    expect(api.queryAccountBalance).toHaveBeenCalledWith(account.id)
    wrapper.unmount()
  })
})
