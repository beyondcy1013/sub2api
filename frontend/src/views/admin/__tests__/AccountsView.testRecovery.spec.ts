import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AccountsView from '../AccountsView.vue'

const {
  getAllGroups,
  getAllProxies,
  getBatchTodayStats,
  listAccounts,
  listWithEtag,
  recoverState,
  setSchedulable,
  showError,
  showSuccess,
  toggleStatus
} = vi.hoisted(() => ({
  getAllGroups: vi.fn(),
  getAllProxies: vi.fn(),
  getBatchTodayStats: vi.fn(),
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  recoverState: vi.fn(),
  setSchedulable: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  toggleStatus: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: false }),
      recoverState,
      setSchedulable,
      toggleStatus
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, showInfo: vi.fn() })
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

const account = {
  id: 42,
  name: 'paused-account',
  platform: 'openai',
  type: 'apikey',
  status: 'error',
  schedulable: false,
  concurrency: 10,
  priority: 0,
  error_message: 'previous failure',
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z'
}

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-test="'account-row-' + row.id">
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
}

const AccountTestModalStub = {
  props: ['show', 'account'],
  emits: ['close', 'test-succeeded'],
  template: `
    <button
      v-if="show && account"
      data-test="complete-account-test"
      @click="$emit('test-succeeded', account)"
    >complete</button>
  `
}

const ConfirmDialogStub = {
  props: ['show', 'title', 'message'],
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-test="confirm-dialog">
      <span>{{ title }}</span>
      <span>{{ message }}</span>
      <button data-test="confirm-recovery" @click="$emit('confirm')">confirm</button>
      <button data-test="cancel-recovery" @click="$emit('cancel')">cancel</button>
    </div>
  `
}

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="table" /></div>' },
        DataTable: DataTableStub,
        Pagination: true,
        ConfirmDialog: ConfirmDialogStub,
        AccountTableActions: true,
        AccountTableFilters: true,
        AccountBulkActionsBar: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        EnhancedImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: AccountTestModalStub,
        AccountStatsModal: true,
        StickySessionReassignModal: true,
        ScheduledAccountActionModal: true,
        AccountBalanceQueryModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        TrashBinModal: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        BulkEditAccountModal: true,
        SchedulingRulesModal: true,
        SchedulingRateModal: true,
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
}

describe('admin AccountsView successful test recovery prompt', () => {
  beforeEach(() => {
    localStorage.clear()
    for (const mock of [
      getAllGroups,
      getAllProxies,
      getBatchTodayStats,
      listAccounts,
      listWithEtag,
      recoverState,
      setSchedulable,
      showError,
      showSuccess,
      toggleStatus
    ]) {
      mock.mockReset()
    }
    listAccounts.mockResolvedValue({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  it('prompts after a successful test and restores both state and scheduling', async () => {
    recoverState.mockResolvedValue({ ...account, status: 'active', error_message: null })
    setSchedulable.mockResolvedValue({ ...account, status: 'active', schedulable: true, error_message: null })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="account-test-action"]').trigger('click')
    await wrapper.get('[data-test="complete-account-test"]').trigger('click')

    expect(wrapper.get('[data-test="confirm-dialog"]').text()).toContain('admin.accounts.testRecoveryTitle')
    await wrapper.get('[data-test="confirm-recovery"]').trigger('click')
    await flushPromises()

    expect(recoverState).toHaveBeenCalledWith(account.id)
    expect(setSchedulable).toHaveBeenCalledWith(account.id, true)
    expect(toggleStatus).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.recoverStateSuccess')
    wrapper.unmount()
  })

  it('activates an inactive account after recovery when the recovery API leaves it inactive', async () => {
    const inactive = { ...account, status: 'inactive', schedulable: true }
    listAccounts.mockResolvedValue({ items: [inactive], total: 1, page: 1, page_size: 20, pages: 1 })
    recoverState.mockResolvedValue(inactive)
    toggleStatus.mockResolvedValue({ ...inactive, status: 'active' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="account-test-action"]').trigger('click')
    await wrapper.get('[data-test="complete-account-test"]').trigger('click')
    await wrapper.get('[data-test="confirm-recovery"]').trigger('click')
    await flushPromises()

    expect(toggleStatus).toHaveBeenCalledWith(account.id, 'active')
    expect(setSchedulable).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not prompt for an already active and schedulable account', async () => {
    listAccounts.mockResolvedValue({
      items: [{ ...account, status: 'active', schedulable: true }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="account-test-action"]').trigger('click')
    await wrapper.get('[data-test="complete-account-test"]').trigger('click')

    expect(wrapper.find('[data-test="confirm-dialog"]').exists()).toBe(false)
    expect(recoverState).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
