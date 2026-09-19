import { describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ScheduledTestsPanel from '../ScheduledTestsPanel.vue'
import { adminAPI } from '@/api/admin'

const { listAll, listByAccount, batchCreate, listAccounts, getAllGroups } = vi.hoisted(() => ({
  listAll: vi.fn(),
  listByAccount: vi.fn(),
  batchCreate: vi.fn(),
  listAccounts: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params: Record<string, unknown> = {}) =>
        Object.entries(params).reduce(
          (message, [name, value]) => message.replace(`{${name}}`, String(value)),
          key
        )
    })
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    scheduledTests: {
      listAll,
      listByAccount,
      batchCreate
    },
    accounts: {
      list: listAccounts
    },
    groups: {
      getAll: getAllGroups
    }
  },
  default: {
    scheduledTests: {},
    accounts: {},
    groups: {}
  }
}))

describe('ScheduledTestsPanel global mode', () => {
  it('loads all plans and submits the selected account IDs as one batch', async () => {
    const plans = [{ id: 1, account_id: 11, account_name: 'Account A', model_id: 'gpt-5.4', cron_expression: '0 4 * * *' }]
    listAll.mockResolvedValue(plans as any)
    listAccounts.mockResolvedValue({
      items: [
        { id: 11, name: 'Account A', type: 'api_key', group_ids: [2] },
        { id: 12, name: 'Account B', type: 'api_key', group_ids: [3] }
      ]
    } as any)
    getAllGroups.mockResolvedValue([
      { id: 2, name: 'Group 2' },
      { id: 3, name: 'Group 3' }
    ] as any)
    batchCreate.mockResolvedValue({ created: 2, failed: 0 })

    const wrapper = mount(ScheduledTestsPanel, {
      props: { show: false, accountId: null, modelOptions: [] },
      global: {
        plugins: [createPinia()],
        stubs: {
          BaseDialog: { template: '<div><slot /></div>' },
          CronScheduleBuilder: true,
          Select: true,
          Input: true,
          Toggle: true,
          ConfirmDialog: true,
          Icon: true
        }
      }
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(listAll).toHaveBeenCalled()

    // Open the global batch-create form to reveal account checkboxes.
    const addButton = wrapper.findAll('button').find(button => button.text().includes('admin.scheduledTests.batchPlan'))
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')
    await flushPromises()

    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    expect(checkboxes).toHaveLength(2)
    await checkboxes[0].setValue(true)
    await checkboxes[1].setValue(true)

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('common.save'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(batchCreate).toHaveBeenCalledWith(expect.objectContaining({
      account_ids: [11, 12],
      cron_expression: '*/30 * * * *'
    }))
  })
})
