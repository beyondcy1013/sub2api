import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  deleteAccount,
  getAllGroups,
  getAllProxies,
  getBatchTodayStats,
  listAccounts,
  listWithEtag
} = vi.hoisted(() => ({
  deleteAccount: vi.fn(),
  getAllGroups: vi.fn(),
  getAllProxies: vi.fn(),
  getBatchTodayStats: vi.fn(),
  listAccounts: vi.fn(),
  listWithEtag: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      delete: deleteAccount,
      recycle: vi.fn(),
      restore: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
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

const account = {
  id: 42,
  name: 'recycled-account',
  platform: 'openai',
  type: 'apikey',
  status: 'inactive',
  schedulable: false,
  concurrency: 1,
  priority: 0,
  error_message: null,
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

const AccountTableActionsStub = {
  props: ['recycled'],
  emits: ['toggle-recycled'],
  template: `
    <div>
      <button data-test="toggle-recycled" @click="$emit('toggle-recycled')">toggle</button>
      <slot name="after" />
    </div>
  `
}

const ConfirmDialogStub = {
  props: ['show'],
  emits: ['confirm', 'cancel'],
  template: '<button v-if="show" data-test="confirm-delete" @click="$emit(\'confirm\')">confirm</button>'
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
        Pagination: true,
        ConfirmDialog: ConfirmDialogStub,
        AccountTableActions: AccountTableActionsStub,
        AccountTableFilters: true,
        AccountBulkActionsBar: true,
        AccountActionMenu: true,
        ImportDataModal: true,
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
        Icon: true
      }
    }
  })
}

describe('admin AccountsView recycle-bin deletion', () => {
  beforeEach(() => {
    localStorage.clear()
    for (const mock of [
      deleteAccount,
      getAllGroups,
      getAllProxies,
      getBatchTodayStats,
      listAccounts,
      listWithEtag
    ]) {
      mock.mockReset()
    }

    listAccounts.mockResolvedValue({
      items: [account],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    deleteAccount.mockResolvedValue({ message: 'deleted' })
  })

  it('permanently deletes an account directly from the recycle bin after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="toggle-recycled"]').trigger('click')
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-42"]')
    expect(row.text()).toContain('admin.accounts.restore')

    const deleteButton = row.findAll('button').find(button => button.text().includes('common.delete'))
    expect(deleteButton).toBeDefined()

    await deleteButton!.trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(deleteAccount).toHaveBeenCalledOnce()
    expect(deleteAccount).toHaveBeenCalledWith(42)
  })
})
