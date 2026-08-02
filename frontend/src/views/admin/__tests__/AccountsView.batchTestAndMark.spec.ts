import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const { list, listWithEtag, batchTestAndMark, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  listWithEtag: vi.fn(),
  batchTestAndMark: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list,
      listWithEtag,
      batchTestAndMark,
      getBatchTodayStats: vi.fn().mockResolvedValue({ stats: {} }),
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: true, interval_minutes: 30 })
    },
    proxies: { getAll: vi.fn().mockResolvedValue([]) },
    groups: { getAll: vi.fn().mockResolvedValue([]) }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, showInfo: vi.fn() })
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ token: 'test-token' }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const BulkBarStub = {
  props: ['selectedIds', 'testingSelected'],
  emits: ['test-and-mark'],
  template: '<button data-test="batch-test" @click="$emit(\'test-and-mark\')">test</button>'
}

const account = {
  id: 7,
  name: 'selected-account',
  platform: 'openai',
  type: 'oauth',
  status: 'active',
  schedulable: true,
  created_at: '2026-07-30T00:00:00Z',
  updated_at: '2026-07-30T00:00:00Z'
}

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: { props: ['data'], template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-select" :row="row" /></div></div>' },
        AccountBulkActionsBar: BulkBarStub,
        AccountTableActions: { template: '<div><slot name="after" /></div>' },
        Pagination: true,
        ConfirmDialog: true,
        AccountTableFilters: true,
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
        TrashBinModal: true,
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

describe('AccountsView batch test and mark', () => {
  beforeEach(() => {
    localStorage.clear()
    list.mockReset().mockResolvedValue({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
    listWithEtag.mockReset().mockResolvedValue({ notModified: true, etag: null, data: null })
    batchTestAndMark.mockReset().mockResolvedValue({ success: 0, failed: 1, results: [] })
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('tests selected accounts, reports failures, clears selection and reloads statuses', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('input[type="checkbox"]').trigger('change')
    await wrapper.get('[data-test="batch-test"]').trigger('click')
    await flushPromises()

    expect(batchTestAndMark).toHaveBeenCalledWith([7])
    expect(showError).toHaveBeenCalledWith('admin.accounts.bulkActions.testAndMarkPartial')
    expect(list).toHaveBeenCalledTimes(2)
  })
})
