import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getUpstreamBillingProbeSettings,
  getAccountUsage,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getUpstreamBillingProbeSettings: vi.fn(),
  getAccountUsage: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings,
      getUsage: getAccountUsage
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token' })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-select" :row="row" /><slot name="cell-usage" :row="row" /></div></div>'
}

const AccountBulkActionsBarStub = {
  props: ['selectedIds', 'refreshingUsage'],
  emits: ['refresh-usage'],
  template: '<button data-test="refresh-usage" @click="$emit(\'refresh-usage\')">refresh usage</button>'
}

const AccountUsageCellStub = {
  props: ['account', 'externalUsage'],
  template: '<span :data-test="`usage-${account.id}`">{{ externalUsage?.updated_at ?? \'-\' }}</span>'
}

const makeAccount = (id: number, platform: string, type: string) => ({
  id,
  name: `account-${id}`,
  platform,
  type,
  status: 'active',
  schedulable: true,
  created_at: '2026-07-19T00:00:00Z',
  updated_at: '2026-07-19T00:00:00Z'
})

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: DataTableStub,
        Pagination: true,
        ConfirmDialog: true,
        AccountTableActions: { template: '<div><slot name="after" /></div>' },
        AccountTableFilters: true,
        AccountBulkActionsBar: AccountBulkActionsBarStub,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: AccountUsageCellStub,
        Icon: true
      }
    }
  })
}

describe('AccountsView bulk usage refresh', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockReset()
    listWithEtag.mockReset()
    getBatchTodayStats.mockReset()
    getUpstreamBillingProbeSettings.mockReset()
    getAccountUsage.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()

    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getUpstreamBillingProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 30 })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    getAccountUsage.mockImplementation(async (id: number) => ({
      updated_at: `usage-${id}`,
      five_hour: null,
      seven_day: null,
      seven_day_sonnet: null
    }))
  })

  it('queries every eligible account in the filtered result with active force semantics', async () => {
    const openAI = makeAccount(1, 'openai', 'oauth')
    const apiKey = makeAccount(2, 'openai', 'apikey')
    const anthropic = makeAccount(3, 'anthropic', 'setup-token')
    listAccounts
      .mockResolvedValueOnce({ items: [openAI], total: 3, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [openAI, apiKey], total: 3, page: 1, page_size: 1000, pages: 2 })
      .mockResolvedValueOnce({ items: [anthropic], total: 3, page: 2, page_size: 1000, pages: 2 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="refresh-usage"]').trigger('click')
    await flushPromises()

    expect(getAccountUsage.mock.calls).toEqual([
      [1, 'active', true],
      [3, 'active', true]
    ])
    expect(wrapper.get('[data-test="usage-1"]').text()).toBe('usage-1')
  })

  it('queries only selected eligible accounts when a selection exists', async () => {
    const openAI = makeAccount(1, 'openai', 'oauth')
    const anthropic = makeAccount(3, 'anthropic', 'setup-token')
    listAccounts.mockResolvedValue({
      items: [openAI, anthropic],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.findAll('input[type="checkbox"]')[0].trigger('change')
    await wrapper.get('[data-test="refresh-usage"]').trigger('click')
    await flushPromises()

    expect(getAccountUsage.mock.calls).toEqual([[1, 'active', true]])
  })
})
