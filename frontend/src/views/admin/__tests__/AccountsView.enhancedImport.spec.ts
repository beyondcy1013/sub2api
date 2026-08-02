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
        props: ['show', 'operation'],
        template: '<div data-test="enhanced-import-modal" :data-show="String(show)" :data-operation="operation" />'
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
      TrashBinModal: true,
      Teleport: true,
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

  it('places enhanced import below import and clear-import directly below enhanced import', async () => {
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
    const enhancedIndex = labels.findIndex(label => label === 'admin.accounts.enhancedImport')
    const clearIndex = labels.findIndex(label => label.includes('admin.accounts.enhancedImportClearButton'))
    expect(enhancedIndex).toBe(importIndex + 1)
    expect(clearIndex).toBe(enhancedIndex + 1)

    await menuItems[enhancedIndex]!.trigger('click')
    const modal = wrapper.find('[data-test="enhanced-import-modal"]')
    expect(modal.attributes('data-show')).toBe('true')
    expect(modal.attributes('data-operation')).toBe('import')
  })

  it('opens the enhanced parser in clear mode from the new menu item', async () => {
    const wrapper = mountView()
    await flushPromises()

    const moreButton = wrapper.findAll('button').find(button =>
      button.text().includes('admin.accounts.moreActions')
    )
    await moreButton!.trigger('click')

    const clearItem = wrapper.findAll('.account-tools-menu-item').find(item =>
      item.text().includes('admin.accounts.enhancedImportClearButton')
    )
    expect(clearItem).toBeTruthy()
    await clearItem!.trigger('click')

    const modal = wrapper.find('[data-test="enhanced-import-modal"]')
    expect(modal.attributes('data-show')).toBe('true')
    expect(modal.attributes('data-operation')).toBe('clear')
  })

  it('positions the teleported tools menu below its trigger on desktop', async () => {
    const wrapper = mountView()
    await flushPromises()

    const moreButton = wrapper.findAll('button').find(button =>
      button.text().includes('admin.accounts.moreActions')
    )
    expect(moreButton).toBeTruthy()
    vi.spyOn(moreButton!.element, 'getBoundingClientRect').mockReturnValue({
      top: 40,
      right: 1000,
      bottom: 72,
      left: 900,
      width: 100,
      height: 32,
      x: 900,
      y: 40,
      toJSON: () => ({})
    } as DOMRect)
    await moreButton!.trigger('click')

    const dropdown = wrapper.find('[data-test="account-tools-dropdown"]')
    expect(dropdown.exists()).toBe(true)
    expect(dropdown.classes()).toContain('fixed')
    expect(dropdown.attributes('style')).toContain('top: 80px')
    expect(dropdown.attributes('style')).toContain('left: 680px')
    expect(dropdown.attributes('style')).toContain('width: 320px')
  })
})
