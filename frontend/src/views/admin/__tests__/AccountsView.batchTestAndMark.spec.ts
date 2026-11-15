import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const { list, listWithEtag, getById, batchTestAndMark, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  listWithEtag: vi.fn(),
  getById: vi.fn(),
  batchTestAndMark: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list,
      listWithEtag,
      getById,
      batchTestAndMark,
      getBatchTodayStats: vi.fn().mockResolvedValue({ stats: {} }),
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: true, interval_minutes: 30 })
    },
    proxies: { getAll: vi.fn().mockResolvedValue([]) },
    groups: { getAll: vi.fn().mockResolvedValue([]) },
    plugins: { list: vi.fn().mockResolvedValue([]) }
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
  props: ['selectedIds'],
  emits: ['pelican-test'],
  template: '<button data-test="batch-test" @click="$emit(\'pelican-test\')">test</button>'
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
        AccountTestModal: {
          name: 'AccountTestModal',
          props: ['show', 'accounts', 'initialMode'],
          template: '<div />'
        },
        AccountDetectionCenter: true,
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

// The old direct test-and-mark flow was replaced by the manual workbench.
describe('AccountsView unified batch test entry', () => {
  beforeEach(() => {
    localStorage.clear()
    list.mockReset().mockResolvedValue({ items: [account], total: 1, page: 1, page_size: 20, pages: 1 })
    listWithEtag.mockReset().mockResolvedValue({ notModified: true, etag: null, data: null })
    getById.mockReset().mockResolvedValue(account)
    batchTestAndMark.mockReset().mockResolvedValue({ success: 0, failed: 1, results: [] })
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('opens selected accounts in the workbench without testing or marking them automatically', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('input[type="checkbox"]').trigger('change')
    await wrapper.get('[data-test="batch-test"]').trigger('click')
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(7)
    expect(wrapper.findComponent({ name: 'AccountTestModal' }).props()).toMatchObject({
      show: true, accounts: [account], initialMode: 'default'
    })
    expect(batchTestAndMark).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
    expect(list).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('does not launch a batch test when no accounts are selected', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="batch-test"]').trigger('click')
    await flushPromises()

    expect(wrapper.findComponent({ name: 'AccountTestModal' }).props('show')).toBe(false)
    expect(getById).not.toHaveBeenCalled()
    expect(batchTestAndMark).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
