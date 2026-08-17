import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'
import accountsViewSource from '../AccountsView.vue?raw'

const {
  deleteAccount,
  getAllGroups,
  getAllProxies,
  getBatchTodayStats,
  getById,
  listAccounts,
  listWithEtag,
  permanentDelete,
  restoreFromTrash
} = vi.hoisted(() => ({
  deleteAccount: vi.fn(),
  getAllGroups: vi.fn(),
  getAllProxies: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getById: vi.fn(),
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  permanentDelete: vi.fn(),
  restoreFromTrash: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getById,
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: false }),
      delete: deleteAccount,
      recycle: vi.fn(),
      restore: vi.fn(),
      listTrashed: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 }),
      restoreFromTrash,
      permanentDelete,
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
  props: ['data', 'rowClass'],
  template: `
    <div>
      <div
        v-for="row in data"
        :key="row.id"
        :data-test="'account-row-' + row.id"
        :class="typeof rowClass === 'function' ? rowClass(row) : rowClass"
      >
        <slot name="cell-name" :row="row" :value="row.name" />
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
}

const AccountTestModalStub = {
  props: ['show', 'account'],
  template: `
    <div v-if="show && account" data-test="account-test-modal">
      {{ account.name }}
    </div>
  `
}

const AccountTableActionsStub = {
  props: ['recycled', 'deleted'],
  emits: ['create', 'toggle-recycled', 'toggle-deleted'],
  template: `
    <div>
      <button data-test="open-create-account" @click="$emit('create')">create</button>
      <button data-test="toggle-recycled" @click="$emit('toggle-recycled')">toggle</button>
      <button data-test="toggle-deleted" @click="$emit('toggle-deleted')">deleted</button>
      <slot name="after" />
    </div>
  `
}

const AccountBulkActionsBarStub = {
  props: ['selectedIds', 'showDelete', 'showPermanentDelete', 'searchQuery'],
  emits: ['delete', 'permanent-delete', 'select-all-results', 'update:search-query'],
  template: `
    <div>
      <button data-test="select-all-results" @click="$emit('select-all-results')">select all results</button>
      <button v-if="showDelete && selectedIds.length" data-test="bulk-delete" @click="$emit('delete')">delete</button>
      <button v-if="showPermanentDelete && selectedIds.length" data-test="bulk-permanent-delete" @click="$emit('permanent-delete')">permanent delete</button>
      <input data-test="bulk-account-filter" :value="searchQuery" @input="$emit('update:search-query', $event.target.value)" />
    </div>
  `
}

const CreateAccountModalStub = {
  props: ['show'],
  emits: ['created', 'close'],
  template: `
    <button v-if="show" data-test="complete-create-account" @click="$emit('created')">
      complete create
    </button>
  `
}

const ImportDataModalStub = {
  props: ['show'],
  emits: ['imported', 'close'],
  template: `
    <button
      v-if="show"
      data-test="complete-data-import"
      @click="$emit('imported', [{ id: 99, name: 'brake-imported' }])"
    >
      complete import
    </button>
  `
}

const ConfirmDialogStub = {
  props: ['show'],
  emits: ['confirm', 'cancel'],
  template: '<button v-if="show" data-test="confirm-delete" @click="$emit(\'confirm\')">confirm</button>'
}

const AccountActionMenuStub = {
  props: ['show', 'account'],
  emits: ['delete', 'close'],
  template: `
    <button
      v-if="show && account && account.extra?.deleted !== true"
      data-test="action-menu-delete"
      @click="$emit('delete', account); $emit('close')"
    >common.delete</button>
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
        Pagination: true,
        ConfirmDialog: ConfirmDialogStub,
        AccountTableActions: AccountTableActionsStub,
        AccountTableFilters: true,
        AccountBulkActionsBar: AccountBulkActionsBarStub,
        AccountActionMenu: AccountActionMenuStub,
        ImportDataModal: ImportDataModalStub,
        ReAuthAccountModal: true,
        AccountTestModal: AccountTestModalStub,
        AccountStatsModal: true,
        StickySessionReassignModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        TrashBinModal: true,
        CreateAccountModal: CreateAccountModalStub,
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
    vi.stubGlobal('confirm', vi.fn(() => true))
    for (const mock of [
      deleteAccount,
      getAllGroups,
      getAllProxies,
      getBatchTodayStats,
      getById,
      listAccounts,
      listWithEtag,
      permanentDelete,
      restoreFromTrash
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
    getById.mockResolvedValue(account)
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    deleteAccount.mockResolvedValue({ message: 'deleted' })
    permanentDelete.mockResolvedValue({ message: 'permanently deleted' })
  })

  it('moves an account from staging into recoverable deleted staging after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="toggle-recycled"]').trigger('click')
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-42"]')
    expect(row.text()).toContain('admin.accounts.restore')

    expect(row.text()).not.toContain('common.delete')
    const moreButton = row.findAll('button').find(button => button.text().includes('common.more'))
    expect(moreButton).toBeDefined()
    await moreButton!.trigger('click')
    await wrapper.get('[data-test="action-menu-delete"]').trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(deleteAccount).toHaveBeenCalledOnce()
    expect(deleteAccount).toHaveBeenCalledWith(42)
  })

  it('loads recoverable deleted staging and keeps manual management actions available', async () => {
    listAccounts
      .mockResolvedValueOnce({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({
        items: [{ ...account, extra: { deleted: true } }],
        total: 1,
        page: 1,
        page_size: 20,
        pages: 1
      })
    restoreFromTrash.mockResolvedValue({ message: 'restored' })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="toggle-deleted"]').trigger('click')
    await flushPromises()

    expect(listAccounts).toHaveBeenLastCalledWith(
      1,
      100,
      expect.objectContaining({ deleted: '1', recycled: '' }),
      expect.objectContaining({ signal: expect.anything() })
    )
    const row = wrapper.get('[data-test="account-row-42"]')
    expect(row.findAll('button').map(button => button.text())).toEqual([
      'common.edit',
      'admin.accounts.testConnection',
      'admin.accounts.restoreDeleted',
      'common.more'
    ])

    await row.get('[data-test="account-restore-deleted-action"]').trigger('click')
    await flushPromises()
    expect(restoreFromTrash).toHaveBeenCalledOnce()
    expect(restoreFromTrash).toHaveBeenCalledWith(42)
  })

  it('does not expose irreversible deletion from deleted staging', async () => {
    listAccounts
      .mockResolvedValueOnce({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [{ ...account, extra: { deleted: true } }], total: 1, page: 1, page_size: 20, pages: 1 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="toggle-deleted"]').trigger('click')
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-42"]')
    await row.get('[data-test="account-more-action"]').trigger('click')

    expect(wrapper.find('[data-test="action-menu-delete"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="bulk-delete"]').exists()).toBe(false)
    expect(permanentDelete).not.toHaveBeenCalled()
    expect(deleteAccount).not.toHaveBeenCalled()
  })

  it('permanently deletes all selected deleted-staging results after confirmation', async () => {
    listAccounts
      .mockResolvedValueOnce({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [{ ...account, id: 42, extra: { deleted: true } }], total: 2, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [{ ...account, id: 42 }, { ...account, id: 43 }], total: 2, page: 1, page_size: 1000, pages: 1 })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="toggle-deleted"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="select-all-results"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="bulk-delete"]').exists()).toBe(false)
    await wrapper.get('[data-test="bulk-permanent-delete"]').trigger('click')
    await flushPromises()

    expect(permanentDelete).toHaveBeenCalledTimes(2)
    expect(permanentDelete).toHaveBeenCalledWith(42)
    expect(permanentDelete).toHaveBeenCalledWith(43)
    expect(deleteAccount).not.toHaveBeenCalled()
  })

  it('binds the compact bulk filter to the account search query', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="bulk-account-filter"]').setValue('notes needle')
    await flushPromises()

    expect(listAccounts).toHaveBeenLastCalledWith(
      1,
      100,
      expect.objectContaining({ search: 'notes needle' }),
      expect.objectContaining({ signal: expect.anything() })
    )
  })
  it('keeps edit, test connection and more available in staging mode alongside restore', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="toggle-recycled"]').trigger('click')
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-42"]')
    // Staging mode keeps the shared edit/test/more actions and swaps recycle for restore.
    expect(row.findAll('button').map(button => button.text())).toEqual([
      'common.edit',
      'admin.accounts.testConnection',
      'admin.accounts.restore',
      'common.more'
    ])
    expect(row.text()).not.toContain('admin.accounts.recycle')
  })

  it('shows edit, test connection, recycle and more in that order, and opens the test modal', async () => {
    const wrapper = mountView()
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-42"]')
    expect(row.findAll('button').map(button => button.text())).toEqual([
      'common.edit',
      'admin.accounts.testConnection',
      'admin.accounts.recycle',
      'common.more'
    ])
    expect(row.findAll('button').every(button => !button.classes().includes('w-6'))).toBe(true)
    expect(row.findAll('button').every(button => button.classes().includes('px-2'))).toBe(true)

    await row.findAll('button')[1].trigger('click')

    expect(wrapper.get('[data-test="account-test-modal"]').text()).toContain(account.name)
  })

  it('shows an overlong account name on at most two lines without growing the table row', async () => {
    listAccounts.mockResolvedValueOnce({
      items: [{ ...account, name: 'account-name-that-is-far-too-long-for-the-name-column' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const name = wrapper.get('[data-test="account-name-value"]')
    const nameCell = wrapper.get('[data-test="account-name-cell"]')
    expect(nameCell.classes()).toEqual(expect.arrayContaining([
      'line-clamp-2',
      'whitespace-normal',
      'account-name-flow',
      'leading-4',
    ]))
    expect(name.classes()).not.toContain('flex-1')
    expect(name.classes()).not.toContain('line-clamp-2')

    const outerCell = nameCell.element.parentElement
    expect(outerCell?.classList.contains('h-8')).toBe(true)
    expect(outerCell?.classList.contains('w-[212px]')).toBe(true)
    expect(outerCell?.classList.contains('max-w-[212px]')).toBe(true)
    expect(outerCell?.classList.contains('overflow-hidden')).toBe(true)
  })

  it('keeps the name prefix before a long supplemental email in one end-clamped text flow', async () => {
    listAccounts.mockResolvedValueOnce({
      items: [{
        ...account,
        name: '平台失联',
        credentials: { email: 'tariqnatalino08048+fm3p11pl@outlook.com' }
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const nameCell = wrapper.get('[data-test="account-name-cell"]')
    expect(nameCell.text()).toBe('平台失联 · tariqnatalino08048+fm3p11pl@outlook.com')
    expect(nameCell.element.firstElementChild?.getAttribute('data-test')).toBe('account-name-value')
    expect(nameCell.classes()).toEqual(expect.arrayContaining(['line-clamp-2', 'account-name-flow']))
    expect(nameCell.find('[data-test="account-display-email"]').classes()).not.toContain('truncate')
  })

  it('does not append an email that is already the suffix of the account name', async () => {
    listAccounts.mockResolvedValueOnce({
      items: [{
        ...account,
        name: '收不到邮件talker-missing1t@icloud.com',
        credentials: { email: ' TALKER-MISSING1T@ICLOUD.COM ' }
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    const nameCell = wrapper.get('[data-test="account-name-cell"]')
    expect(nameCell.text()).toBe('收不到邮件talker-missing1t@icloud.com')
    expect(nameCell.find('[data-test="account-display-email"]').exists()).toBe(false)
  })

  it('temporarily highlights only the account that appears after a successful create', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-27T09:59:59Z'))
    const createdAccount = {
      ...account,
      id: 99,
      name: 'just-created-account',
      created_at: '2026-07-27T10:00:00Z',
      updated_at: '2026-07-27T10:00:00Z'
    }
    listAccounts
      .mockResolvedValueOnce({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [account, createdAccount], total: 2, page: 1, page_size: 20, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-account"]').trigger('click')
    await wrapper.get('[data-test="complete-create-account"]').trigger('click')

    await vi.waitFor(() => {
      expect(wrapper.findAll('[data-test^="account-row-"]').map(row => row.attributes('data-test'))).toEqual([
        'account-row-99',
        'account-row-42'
      ])
    })
    expect(wrapper.get('[data-test="account-row-42"]').classes()).not.toContain('recently-created-account-row')
    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row')
    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row-red')

    vi.advanceTimersByTime(10_000)
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row')
    expect(wrapper.get('[data-test="account-row-99"]').classes()).not.toContain('recently-created-account-row-red')
    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row-orange')
    expect(wrapper.findAll('[data-test^="account-row-"]').map(row => row.attributes('data-test'))).toEqual([
      'account-row-99',
      'account-row-42'
    ])

    vi.advanceTimersByTime(10_000)
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-test="account-row-99"]').classes()).not.toContain('recently-created-account-row')
    expect(wrapper.findAll('[data-test^="account-row-"]').map(row => row.attributes('data-test'))).toEqual([
      'account-row-42',
      'account-row-99'
    ])
    wrapper.unmount()
    vi.useRealTimers()
  })

  it('loads the newest matching account and pins it when the active sort omits it from page one', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-27T10:00:00Z'))
    const createdAccount = {
      ...account,
      id: 99,
      name: 'z-last-by-name',
      created_at: '2026-07-27T10:00:01Z',
      updated_at: '2026-07-27T10:00:01Z'
    }
    const pageByName = { items: [account], total: 2, page: 1, page_size: 20, pages: 1 }
    listAccounts
      .mockResolvedValueOnce(pageByName)
      .mockResolvedValueOnce(pageByName)
      .mockResolvedValueOnce({ items: [createdAccount, account], total: 2, page: 1, page_size: 20, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-account"]').trigger('click')
    await wrapper.get('[data-test="complete-create-account"]').trigger('click')
    await flushPromises()

    expect(listAccounts).toHaveBeenLastCalledWith(
      1,
      100,
      expect.objectContaining({ sort_by: 'created_at', sort_order: 'desc' })
    )
    expect(wrapper.findAll('[data-test^="account-row-"]').map(row => row.attributes('data-test'))).toEqual([
      'account-row-99',
      'account-row-42'
    ])
    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row')

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('temporarily pins and highlights an account added through data import', async () => {
    vi.useFakeTimers()
    const importedAccount = {
      ...account,
      id: 99,
      name: 'brake-imported',
      created_at: '2026-07-28T09:30:00+08:00',
      updated_at: '2026-07-28T09:30:00+08:00'
    }
    listAccounts
      .mockResolvedValueOnce({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({ items: [account, importedAccount], total: 2, page: 1, page_size: 20, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[title="admin.accounts.moreActions"]').trigger('click')
    await wrapper.vm.$nextTick()
    const importButton = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('[data-test="account-tools-dropdown"] button')
    ).find(button => button.textContent?.includes('admin.accounts.dataImport'))
    expect(importButton).toBeDefined()
    importButton!.click()
    await wrapper.vm.$nextTick()
    await wrapper.get('[data-test="complete-data-import"]').trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[data-test^="account-row-"]').map(row => row.attributes('data-test'))).toEqual([
      'account-row-99',
      'account-row-42'
    ])
    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row')

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('uses imported account identity instead of highlighting a newly appeared name-sort row', async () => {
    vi.useFakeTimers()
    const nameSortFirstAccount = {
      ...account,
      id: 88,
      name: 'a.r.n-existing',
      created_at: '2026-07-28T09:31:00+08:00'
    }
    const importedAccount = {
      ...account,
      id: 99,
      name: 'brake-imported',
      created_at: '2026-07-28T09:30:00+08:00'
    }
    getById.mockResolvedValueOnce(importedAccount)
    listAccounts
      .mockResolvedValueOnce({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
      .mockResolvedValueOnce({
        items: [nameSortFirstAccount, account, importedAccount],
        total: 3,
        page: 1,
        page_size: 20,
        pages: 1
      })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[title="admin.accounts.moreActions"]').trigger('click')
    await wrapper.vm.$nextTick()
    const importButton = Array.from(
      document.body.querySelectorAll<HTMLButtonElement>('[data-test="account-tools-dropdown"] button')
    ).find(button => button.textContent?.includes('admin.accounts.dataImport'))
    importButton!.click()
    await wrapper.vm.$nextTick()
    await wrapper.get('[data-test="complete-data-import"]').trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[data-test^="account-row-"]').map(row => row.attributes('data-test'))).toEqual([
      'account-row-99',
      'account-row-88',
      'account-row-42'
    ])
    expect(wrapper.get('[data-test="account-row-99"]').classes()).toContain('recently-created-account-row')
    expect(wrapper.get('[data-test="account-row-88"]').classes()).not.toContain('recently-created-account-row')

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('keeps dark-mode highlight selectors global through scoped-style compilation', () => {
    expect(accountsViewSource).toContain(':global(.dark tr.recently-created-account-row-red > td)')
    expect(accountsViewSource).toContain(':global(.dark tr.recently-created-account-row-orange > td)')
    expect(accountsViewSource).toContain(':global(.dark div.recently-created-account-row-red)')
    expect(accountsViewSource).toContain(':global(.dark div.recently-created-account-row-orange)')
    expect(accountsViewSource).toContain('background-color: rgb(254 226 226) !important')
    expect(accountsViewSource).toContain('background-color: rgb(255 237 213) !important')
    expect(accountsViewSource).toContain('background-color: rgb(127 29 29 / 0.36) !important')
    expect(accountsViewSource).toContain('background-color: rgb(124 45 18 / 0.36) !important')
    expect(accountsViewSource).not.toContain(':global(.dark) :deep(')
  })
})
