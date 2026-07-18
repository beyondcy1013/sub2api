import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getUpstreamBillingProbeSettings,
  getAllProxies,
  getAllGroups,
  probeUpstreamBillingBatch,
  bulkUpdateAccounts
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getUpstreamBillingProbeSettings: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn(),
  probeUpstreamBillingBatch: vi.fn(),
  bulkUpdateAccounts: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings,
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      probeUpstreamBillingBatch,
      bulkUpdate: bulkUpdateAccounts,
      toggleSchedulable: vi.fn()
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
  template: `
    <div data-test="data-table">
      <span v-for="column in columns" :key="column.key" data-test="column-key">{{ column.key }}</span>
      <div v-for="row in data" :key="row.id">
        <div data-test="select-row"><slot name="cell-select" :row="row" /></div>
        <slot name="cell-name" :value="row.name" :row="row" />
        <slot name="cell-today_cost" :value="row.today_cost" :row="row" />
        <slot name="cell-created_at" :value="row.created_at" :row="row" />
      </div>
    </div>
  `
}

const AccountBulkActionsBarStub = {
  props: ['selectedIds', 'selectingAllPages', 'quickUpdating', 'proxies', 'groups'],
  emits: ['edit-selected', 'edit-filtered', 'probe-upstream-billing', 'select-all-pages', 'quick-set-proxy', 'quick-set-group'],
  template: `
    <div>
      <span data-test="selected-ids">{{ selectedIds.join(',') }}</span>
      <button data-test="select-all-pages" @click="$emit('select-all-pages')">select all pages</button>
      <button data-test="quick-set-proxy" @click="$emit('quick-set-proxy', 9)">proxy</button>
      <button data-test="quick-set-group" @click="$emit('quick-set-group', 5)">group</button>
      <button data-test="edit-selected" @click="$emit('edit-selected')">edit selected</button>
      <button data-test="edit-filtered" @click="$emit('edit-filtered')">edit filtered</button>
      <button data-test="probe-upstream-billing" @click="$emit('probe-upstream-billing')">probe</button>
    </div>
  `
}

const PaginationStub = {
  emits: ['update:page'],
  template: '<button data-test="next-page" @click="$emit(\'update:page\', 2)">next</button>'
}

const BulkEditAccountModalStub = {
  props: ['show', 'target'],
  template: '<div data-test="bulk-edit-modal" :data-show="String(show)" :data-target-mode="target?.mode ?? \'\'"></div>'
}

function mountView(extraStubs: Record<string, unknown> = {}) {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: DataTableStub,
        Pagination: true,
        ConfirmDialog: true,
        AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
        AccountTableFilters: { template: '<div></div>' },
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
        BulkEditAccountModal: BulkEditAccountModalStub,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: true,
        Icon: true,
        ...extraStubs
      }
    }
  })
}

describe('admin AccountsView bulk edit scope', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockReset()
    listWithEtag.mockReset()
    getBatchTodayStats.mockReset()
    getUpstreamBillingProbeSettings.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()
    probeUpstreamBillingBatch.mockReset()
    bulkUpdateAccounts.mockReset()

    listAccounts.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getUpstreamBillingProbeSettings.mockResolvedValue({ enabled: true, interval_minutes: 30 })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    probeUpstreamBillingBatch.mockResolvedValue([])
    bulkUpdateAccounts.mockResolvedValue({ success: 1, failed: 0, results: [] })
  })

  it('opens bulk edit in filtered-results mode from the bulk actions dropdown', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-show')).toBe('true')
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-target-mode')).toBe('filtered')
  })

  it('selects account IDs from every filtered page', async () => {
    const account = (id: number) => ({
      id,
      name: `account-${id}`,
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      created_at: '2026-07-13T00:00:00Z',
      updated_at: '2026-07-13T00:00:00Z'
    })
    listAccounts
      .mockResolvedValueOnce({ items: [account(7)], total: 2, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [account(7)], total: 2, page: 1, page_size: 1000, pages: 2 })
      .mockResolvedValueOnce({ items: [account(11)], total: 2, page: 2, page_size: 1000, pages: 2 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="select-all-pages"]').trigger('click')
    await flushPromises()

    expect(listAccounts).toHaveBeenNthCalledWith(
      2,
      1,
      1000,
      expect.objectContaining({ recycled: '' })
    )
    expect(listAccounts).toHaveBeenNthCalledWith(
      3,
      2,
      1000,
      expect.objectContaining({ recycled: '' })
    )
    expect(wrapper.get('[data-test="selected-ids"]').text()).toBe('7,11')
  })

  it('directly applies a proxy and group to selected accounts', async () => {
    const account = { id: 7, name: 'account-7', platform: 'openai', type: 'apikey', status: 'active', schedulable: true, created_at: '2026-07-13T00:00:00Z', updated_at: '2026-07-13T00:00:00Z' }
    listAccounts.mockResolvedValue({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="quick-set-proxy"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="quick-set-group"]').trigger('click')
    await flushPromises()

    expect(bulkUpdateAccounts).toHaveBeenNthCalledWith(1, [7], { proxy_id: 9 })
    expect(bulkUpdateAccounts).toHaveBeenNthCalledWith(2, [7], { group_ids: [5] })
  })

  it('renders the created_at column by default', async () => {
    listAccounts.mockResolvedValue({
      items: [{ id: 1, name: 'test-account', platform: 'anthropic', type: 'oauth', status: 'active', schedulable: true, created_at: '2026-03-07T10:00:00Z', updated_at: '2026-03-07T10:00:00Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    const wrapper = mountView()
    await flushPromises()

    const columnKeys = wrapper.findAll('[data-test="column-key"]').map(node => node.text())
    expect(columnKeys).toContain('created_at')
    const columns = wrapper.getComponent(DataTableStub).props('columns') as Array<{ key: string; label: string; sortable: boolean }>
    expect(columns.find(column => column.key === 'created_at')).toMatchObject({ label: 'admin.accounts.columns.createdAt', sortable: true })
  })

  it('renders account identifiers in plain text under account names', async () => {
    listAccounts.mockResolvedValue({
      items: [
        { id: 1, name: 'openai-main', platform: 'openai', type: 'oauth', credentials: { email: 'owner@example.com' }, extra: { chatgpt_account_id: 'acct-fallback' }, status: 'active', schedulable: true, created_at: '2026-03-07T10:00:00Z', updated_at: '2026-03-07T10:00:00Z' },
        { id: 2, name: 'vertex-main', platform: 'gemini', type: 'service_account', credentials: { client_email: 'svc@vertex-project.iam.gserviceaccount.com' }, status: 'active', schedulable: true, created_at: '2026-03-07T10:00:00Z', updated_at: '2026-03-07T10:00:00Z' }
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('owner@example.com')
    expect(wrapper.text()).toContain('svc@vertex-project.iam.gserviceaccount.com')
  })

  it('renders today cost by default from batched account stats', async () => {
    listAccounts.mockResolvedValue({
      items: [{ id: 7, name: 'cost-account', platform: 'openai', type: 'apikey', status: 'active', schedulable: true, created_at: '2026-03-07T10:00:00Z', updated_at: '2026-03-07T10:00:00Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getBatchTodayStats.mockResolvedValue({ stats: { '7': { requests: 3, tokens: 1200, cost: 1.23, standard_cost: 1.23, user_cost: 1.5 } } })
    const wrapper = mountView({
      AccountTodayCostCell: { props: ['stats'], template: '<span data-test="today-cost">{{ stats?.cost }}</span>' }
    })
    await flushPromises()
    await flushPromises()

    const columns = wrapper.getComponent(DataTableStub).props('columns') as Array<{ key: string; label: string; sortable: boolean; width?: string }>
    expect(columns.find(column => column.key === 'today_cost')).toMatchObject({ label: 'admin.accounts.columns.todayCost', sortable: false })
    expect(columns.find(column => column.key === 'status')).toMatchObject({ width: '80px' })
    expect(getBatchTodayStats).toHaveBeenCalledWith([7])
    expect(wrapper.get('[data-test="today-cost"]').text()).toBe('1.23')
  })

  it('passes the loaded global probe state to every upstream billing cell', async () => {
    listAccounts.mockResolvedValue({
      items: [{ id: 1, name: 'upstream', platform: 'openai', type: 'apikey', status: 'active', schedulable: true, created_at: '2026-07-13T00:00:00Z', updated_at: '2026-07-13T00:00:00Z' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getUpstreamBillingProbeSettings.mockResolvedValue({ enabled: false, interval_minutes: 30 })
    const wrapper = mountView({
      DataTable: { props: ['data'], template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-upstream_billing_rate" :row="row" /></div></div>' },
      UpstreamBillingRateCell: { props: ['globalProbeEnabled'], template: '<span data-test="upstream-billing-cell" :data-global-enabled="String(globalProbeEnabled)"></span>' }
    })
    await flushPromises()

    expect(getUpstreamBillingProbeSettings).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-test="upstream-billing-cell"]').attributes('data-global-enabled')).toBe('false')
  })

  it('submits selected account IDs from every page for backend eligibility checks', async () => {
    const account = (id: number) => ({ id, name: `account-${id}`, platform: 'openai', type: 'apikey', status: 'active', schedulable: true, created_at: '2026-07-13T00:00:00Z', updated_at: '2026-07-13T00:00:00Z' })
    listAccounts.mockResolvedValueOnce({ items: [account(7)], total: 2, page: 1, page_size: 1, pages: 2 }).mockResolvedValueOnce({ items: [account(11)], total: 2, page: 2, page_size: 1, pages: 2 })
    const wrapper = mountView({ Pagination: PaginationStub })
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="next-page"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="probe-upstream-billing"]').trigger('click')
    await flushPromises()

    expect(probeUpstreamBillingBatch).toHaveBeenCalledWith([7, 11])
  })

  it('reloads the server-sorted list after a batch probe changes a snapshot', async () => {
    localStorage.setItem('account-table-sort', JSON.stringify({ key: 'upstream_billing_rate', order: 'asc' }))
    const account = (id: number) => ({ id, name: `account-${id}`, platform: 'openai', type: 'apikey', status: 'active', schedulable: true, created_at: '2026-07-13T00:00:00Z', updated_at: '2026-07-13T00:00:00Z' })
    listAccounts.mockResolvedValueOnce({ items: [account(7)], total: 1, page: 1, page_size: 20, pages: 1 }).mockResolvedValueOnce({ items: [account(7)], total: 1, page: 1, page_size: 20, pages: 1 })
    probeUpstreamBillingBatch.mockResolvedValue([{ account_id: 7, snapshot: { status: 'ok', data: { effective_rate_multiplier: 0.5 }, last_attempt_at: '2026-07-13T00:00:00Z', next_probe_at: '2026-07-13T00:30:00Z' } }])
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="select-row"] input').trigger('change')
    await wrapper.get('[data-test="probe-upstream-billing"]').trigger('click')
    await flushPromises()

    expect(probeUpstreamBillingBatch).toHaveBeenCalledWith([7])
    expect(listAccounts).toHaveBeenCalledTimes(2)
  })
})
