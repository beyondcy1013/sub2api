import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'
import accountsViewSource from '../AccountsView.vue?raw'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getDegradationCounts,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getDegradationCounts: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getDegradationCounts,
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: true, interval_minutes: 30 }),
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: {
      getAll: getAllProxies
    },
    groups: {
      getAll: getAllGroups
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] })
    }
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
  useAuthStore: () => ({
    token: 'test-token'
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, any>) => {
        if (key === 'admin.accounts.columns.degradationCount') return '降智次数'
        if (key === 'admin.accounts.degradationRanges.24h') return '24小时'
        if (key === 'admin.accounts.degradationRanges.7d') return '7天'
        if (key === 'admin.accounts.degradationRanges.30d') return '30天'
        if (key === 'admin.accounts.degradationRanges.all') return '全部'
        if (key === 'admin.accounts.degradationCountTooltip') return `近 ${params?.range} 检测到 ${params?.count} 次降智`
        if (key === 'admin.accounts.degradationCountAllTooltip') return `历史累计检测到 ${params?.count} 次降智`
        return key
      }
    })
  }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div data-test="data-table">
      <template v-for="column in columns" :key="column.key">
        <div data-test="column-key" :data-column-key="column.key"></div>
        <div v-if="column.key === 'degradation_count'" data-test="degradation-count-header">
          <slot :name="'header-' + column.key" :column="column" />
        </div>
      </template>
      <div v-for="row in data" :key="row.id" :data-test="'row-' + row.id">
        <div data-test="cell-degradation-count">
          <slot name="cell-degradation_count" :row="row" />
        </div>
      </div>
    </div>
  `
}

describe('AccountsView degradation_count column', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    getBatchTodayStats.mockResolvedValue({ stats: {}, total_stats: {} })
    getDegradationCounts.mockResolvedValue({ '83': 38, '297': 0 })
    listAccounts.mockResolvedValue({
      items: [
        { id: 83, name: 'Account 83', platform: 'openai', type: 'apikey', status: 'active', schedulable: true },
        { id: 297, name: 'Account 297', platform: 'openai', type: 'apikey', status: 'active', schedulable: true }
      ],
      total: 2,
      page: 1,
      page_size: 100
    })
    listWithEtag.mockResolvedValue({
      status: 200,
      etag: 'etag-1',
      data: {
        items: [
          { id: 83, name: 'Account 83', platform: 'openai', type: 'apikey', status: 'active', schedulable: true },
          { id: 297, name: 'Account 297', platform: 'openai', type: 'apikey', status: 'active', schedulable: true }
        ],
        total: 2,
        page: 1,
        page_size: 100
      }
    })
  })

  it('declares degradation_count right after quality_status with 130px width', () => {
    expect(accountsViewSource).toContain("{ key: 'degradation_count', label: t('admin.accounts.columns.degradationCount'), sortable: true, width: '130px' }")
    const qualityIndex = accountsViewSource.indexOf("{ key: 'quality_status'")
    const degradationIndex = accountsViewSource.indexOf("{ key: 'degradation_count'")
    const usageIndex = accountsViewSource.indexOf("{ key: 'usage'")
    expect(degradationIndex).toBeGreaterThan(qualityIndex)
    expect(usageIndex).toBeGreaterThan(degradationIndex)
  })

  it('renders dropdown in header and fetches degradation counts with default 7d range', async () => {
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: DataTableStub,
          AccountTableFilters: { template: '<div></div>' },
          AccountTableActions: { template: '<div><slot name="before" /><slot name="after" /><slot name="afterCreate" /></div>' },
          AccountBulkActionsBar: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: true,
          AccountTestModal: true,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          TrashBinModal: true,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })
    await flushPromises()

    expect(getDegradationCounts).toHaveBeenCalledWith([83, 297], '7d')

    const select = wrapper.find('[data-test="degradation-range-select"]')
    expect(select.exists()).toBe(true)
    const selectEl = select.element as HTMLSelectElement
    expect(selectEl.value).toBe('7d')

    // Cell checks
    const cells = wrapper.findAll('[data-test="cell-degradation-count"]')
    expect(cells.length).toBe(2)
    expect(cells[0].text()).toContain('38 次')
    expect(cells[1].text()).toContain('0')

    // Changing dropdown
    await select.setValue('24h')
    await select.trigger('change')
    await flushPromises()

    expect(getDegradationCounts).toHaveBeenCalledWith([83, 297], '24h')
    expect(localStorage.getItem('account-degradation-count-range')).toBe('24h')
  })
})
