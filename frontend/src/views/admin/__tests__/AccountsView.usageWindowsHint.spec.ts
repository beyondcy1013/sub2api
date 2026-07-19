import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
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
    proxies: {
      getAll: getAllProxies
    },
    groups: {
      getAll: getAllGroups
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
      t: (key: string) => key
    })
  }
})

// Render the per-column header slots so we can assert the usage-window header hint.
const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div data-test="data-table">
      <div data-test="row-order">{{ data.map(row => row.name).join(',') }}</div>
      <template v-for="column in columns" :key="column.key">
        <div v-if="column.key === 'usage'" data-test="usage-header">
          <slot :name="'header-' + column.key" :column="column" />
        </div>
        <div v-if="column.key === 'upstream_billing_rate'" data-test="upstream-billing-header">
          <slot :name="'header-' + column.key" :column="column" />
        </div>
      </template>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-usage" :row="row" />
      </div>
    </div>
  `
}

// Expose the content passed to HelpTooltip without dealing with its <Teleport>.
const HelpTooltipStub = {
  props: ['content', 'widthClass'],
  template: '<span data-test="usage-windows-hint">{{ content }}</span>'
}

const AccountUsageCellStub = {
  props: ['account'],
  emits: ['usage-loaded'],
  template: `
    <button
      :data-test="'emit-usage-' + account.id"
      @click="$emit('usage-loaded', account.extra.test_usage)"
    >usage</button>
  `
}

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: DataTableStub,
        HelpTooltip: HelpTooltipStub,
        Pagination: true,
        ConfirmDialog: true,
        AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
        AccountTableFilters: { template: '<div></div>' },
        AccountBulkActionsBar: true,
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

describe('admin AccountsView usage windows hint', () => {
  beforeEach(() => {
    localStorage.clear()

    listAccounts.mockReset()
    listWithEtag.mockReset()
    getBatchTodayStats.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()

    listAccounts.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    listWithEtag.mockResolvedValue({
      notModified: true,
      etag: null,
      data: null
    })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  it('renders an explanatory tooltip next to the usage windows column header', async () => {
    const wrapper = mountView()
    await flushPromises()

    const header = wrapper.find('[data-test="usage-header"]')
    expect(header.exists()).toBe(true)
    // Column label is still shown alongside the help icon.
    expect(header.text()).toContain('admin.accounts.columns.usageWindows')

    const hint = wrapper.find('[data-test="usage-windows-hint"]')
    expect(hint.exists()).toBe(true)
    expect(hint.text()).toBe('admin.accounts.usageWindowsHint')
  })

  it('renders the upstream billing trust warning next to the declared-rate column', async () => {
    const wrapper = mountView()
    await flushPromises()

    const header = wrapper.find('[data-test="upstream-billing-header"]')
    expect(header.exists()).toBe(true)
    expect(header.text()).toContain('admin.accounts.columns.upstreamBillingRate')
    expect(wrapper.findAll('[data-test="usage-windows-hint"]').some(node =>
      node.text() === 'admin.accounts.upstreamBilling.trustWarning'
    )).toBe(true)
    const columns = wrapper.getComponent(DataTableStub).props('columns') as Array<{ key: string; sortable: boolean }>
    expect(columns.find(column => column.key === 'upstream_billing_rate')?.sortable).toBe(true)
  })

  it('sorts the current page by each 5h and 7d utilization or reset metric', async () => {
    listAccounts.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'lower-usage-later-reset',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          extra: {
            test_usage: {
              five_hour: { utilization: 11, resets_at: '2026-07-19T10:00:00Z' },
              seven_day: { utilization: 12, resets_at: '2026-07-26T10:00:00Z' }
            }
          }
        },
        {
          id: 2,
          name: 'higher-usage-sooner-reset',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          extra: {
            test_usage: {
              five_hour: { utilization: 80, resets_at: '2026-07-19T09:00:00Z' },
              seven_day: { utilization: 93, resets_at: '2026-07-25T11:00:00Z' }
            }
          }
        },
        {
          id: 3,
          name: 'idle-now',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          extra: {
            test_usage: {
              five_hour: { utilization: 0, resets_at: '2026-07-27T10:00:00Z' },
              seven_day: { utilization: 0, resets_at: '2026-07-28T10:00:00Z' }
            }
          }
        },
        {
          id: 4,
          name: 'usage-not-loaded',
          platform: 'openai',
          type: 'oauth',
          status: 'inactive',
          extra: {}
        }
      ],
      total: 4,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="emit-usage-1"]').trigger('click')
    await wrapper.get('[data-test="emit-usage-2"]').trigger('click')
    await wrapper.get('[data-test="emit-usage-3"]').trigger('click')

    await wrapper.get('[data-test="usage-window-sort-trigger"]').trigger('click')
    const sortItems = [
      ['five_hour_utilization', 'admin.accounts.usageWindow.sort.fiveHourUtilization'],
      ['five_hour_reset', 'admin.accounts.usageWindow.sort.fiveHourReset'],
      ['seven_day_utilization', 'admin.accounts.usageWindow.sort.sevenDayUtilization'],
      ['seven_day_reset', 'admin.accounts.usageWindow.sort.sevenDayReset']
    ] as const
    for (const [metric, label] of sortItems) {
      const item = wrapper.get(`[data-test="usage-window-sort-${metric}"]`)
      expect(item.text()).toContain(label)
    }

    await wrapper.get('[data-test="usage-window-sort-seven_day_utilization"]').trigger('click')
    expect(wrapper.get('[data-test="row-order"]').text()).toBe(
      'higher-usage-sooner-reset,lower-usage-later-reset,idle-now,usage-not-loaded'
    )

    await wrapper.get('[data-test="usage-window-sort-trigger"]').trigger('click')
    await wrapper.get('[data-test="usage-window-sort-seven_day_reset"]').trigger('click')
    expect(wrapper.get('[data-test="row-order"]').text()).toBe(
      'lower-usage-later-reset,higher-usage-sooner-reset,idle-now,usage-not-loaded'
    )

    await wrapper.get('[data-test="usage-window-sort-trigger"]').trigger('click')
    await wrapper.get('[data-test="usage-window-sort-five_hour_utilization"]').trigger('click')
    expect(wrapper.get('[data-test="row-order"]').text()).toBe(
      'higher-usage-sooner-reset,lower-usage-later-reset,idle-now,usage-not-loaded'
    )

    await wrapper.get('[data-test="usage-window-sort-trigger"]').trigger('click')
    await wrapper.get('[data-test="usage-window-sort-five_hour_reset"]').trigger('click')
    expect(wrapper.get('[data-test="row-order"]').text()).toBe(
      'lower-usage-later-reset,higher-usage-sooner-reset,idle-now,usage-not-loaded'
    )

    await wrapper.get('[data-test="usage-window-sort-trigger"]').trigger('click')
    await wrapper.get('[data-test="usage-window-sort-five_hour_reset"]').trigger('click')
    expect(wrapper.get('[data-test="row-order"]').text()).toBe(
      'idle-now,higher-usage-sooner-reset,lower-usage-later-reset,usage-not-loaded'
    )

    await wrapper.get('[data-test="usage-window-sort-trigger"]').trigger('click')
    await wrapper.get('[data-test="usage-window-sort-five_hour_reset"]').trigger('click')
    expect(wrapper.get('[data-test="row-order"]').text()).toBe(
      'lower-usage-later-reset,higher-usage-sooner-reset,idle-now,usage-not-loaded'
    )
  })
})
