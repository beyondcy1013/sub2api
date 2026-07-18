import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AccountsView from '../AccountsView.vue'

const { listAccounts, listWithEtag, getBatchTodayStats, getAllProxies, getAllGroups } = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: true, interval_minutes: 30 }),
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn(), showInfo: vi.fn() })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token' })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const mountView = () => mount(AccountsView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
      DataTable: true,
      Pagination: true,
      ConfirmDialog: true,
      AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
      AccountTableFilters: true,
      AccountBulkActionsBar: true,
      AccountActionMenu: true,
      ImportDataModal: true,
      EnhancedImportDataModal: {
        props: ['show'],
        template: '<div data-test="enhanced-import-modal" :data-show="String(show)" />'
      },
      ReAuthAccountModal: true,
      AccountTestModal: true,
      AccountStatsModal: true,
      StickySessionReassignModal: true,
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
      AccountUsageCell: true,
      UpstreamBillingRateCell: true,
      Icon: true
    }
  }
})

describe('admin AccountsView enhanced import menu', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  it('places enhanced import directly below import and opens its modal', async () => {
    const wrapper = mountView()
    await flushPromises()

    const moreButton = wrapper.findAll('button').find(button =>
      button.text().includes('admin.accounts.moreActions')
    )
    expect(moreButton).toBeTruthy()
    await moreButton!.trigger('click')

    const menuItems = wrapper.findAll('.account-tools-menu-item')
    const labels = menuItems.map(item => item.text())
    const importIndex = labels.findIndex(label => label.includes('admin.accounts.dataImport'))
    const enhancedIndex = labels.findIndex(label => label.includes('admin.accounts.enhancedImport'))
    expect(enhancedIndex).toBe(importIndex + 1)

    await menuItems[enhancedIndex]!.trigger('click')
    expect(wrapper.find('[data-test="enhanced-import-modal"]').attributes('data-show')).toBe('true')
  })
})
