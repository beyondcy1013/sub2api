import { describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ScheduledTestsPanel from '../ScheduledTestsPanel.vue'
import { adminAPI } from '@/api/admin'

const { listAll, listByAccount, batchCreate, listAccounts, getAllGroups, update, listResults, listRules, createRule, updateRule, deleteRule } = vi.hoisted(() => ({
  listAll: vi.fn(),
  listByAccount: vi.fn(),
  batchCreate: vi.fn(),
  listAccounts: vi.fn(),
  getAllGroups: vi.fn(),
  update: vi.fn(),
  listResults: vi.fn(),
  listRules: vi.fn(),
  createRule: vi.fn(),
  updateRule: vi.fn(),
  deleteRule: vi.fn()
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
      batchCreate,
      update,
      listResults
    },
    accounts: {
      list: listAccounts
    },
    groups: {
      getAll: getAllGroups
    },
    degradationRules: {
      list: listRules,
      create: createRule,
      update: updateRule,
      delete: deleteRule
    }
  },
  default: {
    scheduledTests: {},
    accounts: {},
    groups: {},
    degradationRules: {}
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
    listRules.mockResolvedValue([])
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

// Interactive Select/Toggle stubs so v-model works in pelican-mode tests.
const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: `<select class="test-select" :value="modelValue" @change="$emit('update:modelValue', $event.target.value)">
    <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option>
  </select>`
}
const ToggleStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: `<input type="checkbox" class="test-toggle" :checked="modelValue" @change="$emit('update:modelValue', $event.target.checked)" />`
}

const mountPanel = (props: any = {}) => mount(ScheduledTestsPanel, {
  props: { show: false, accountId: null, modelOptions: [{ label: 'gpt-5.4', value: 'gpt-5.4' }], ...props },
  global: {
    plugins: [createPinia()],
    stubs: {
      BaseDialog: { template: '<div><slot /></div>' },
      CronScheduleBuilder: true,
      Select: SelectStub,
      Input: true,
      Toggle: ToggleStub,
      ConfirmDialog: true,
      Icon: true
    }
  }
})

describe('ScheduledTestsPanel pelican mode', () => {
  it('filters candidates to OpenAI accounts and submits pelican payload with default model', async () => {
    listAll.mockResolvedValue([] as any)
    listRules.mockResolvedValue([] as any)
    listAccounts.mockResolvedValue({
      items: [
        { id: 11, name: 'OpenAI A', type: 'oauth', platform: 'openai', group_ids: [] },
        { id: 12, name: 'Claude B', type: 'oauth', platform: 'anthropic', group_ids: [] }
      ]
    } as any)
    listRules.mockResolvedValue([])
    getAllGroups.mockResolvedValue([] as any)
    batchCreate.mockResolvedValue({ created: 1, failed: 0 })

    const wrapper = mountPanel()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const addButton = wrapper.findAll('button').find(button => button.text().includes('admin.scheduledTests.batchPlan'))
    await addButton!.trigger('click')
    await flushPromises()

    // Default mode shows both candidates (toggles render as .test-toggle checkboxes).
    expect(wrapper.findAll('input[type="checkbox"]:not(.test-toggle)')).toHaveLength(2)

    // Switch the mode select to pelican (the group filter select comes first
    // in the DOM, so find the select offering the pelican option).
    const selects = wrapper.findAll('select.test-select')
    const modeSelect = selects.find(select =>
      select.findAll('option').some(option => option.element.value === 'pelican')
    )
    expect(modeSelect).toBeTruthy()
    expect((modeSelect!.element as HTMLSelectElement).value).toBe('default')
    await modeSelect!.setValue('pelican')
    await flushPromises()

    // Pelican defaults the model and the handling action to pause.
    const selects2 = wrapper.findAll('select.test-select')
    const actionSelect = selects2.find(select =>
      select.findAll('option').some(option => option.element.value === 'pause')
    )
    expect(actionSelect).toBeTruthy()
    expect((actionSelect!.element as HTMLSelectElement).value).toBe('pause')

    // Only the OpenAI candidate remains selectable.
    const checkboxes = wrapper.findAll('input[type="checkbox"]:not(.test-toggle)')
    expect(checkboxes).toHaveLength(1)
    await checkboxes[0].setValue(true)

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('common.save'))
    await saveButton!.trigger('click')
    await flushPromises()

    expect(batchCreate).toHaveBeenCalledWith(expect.objectContaining({
      account_ids: [11],
      mode: 'pelican',
      downgrade_action: 'pause',
      model_id: 'gpt-6-astra'
    }))
  })

  it('renders the pelican plan badge and the suspected-downgrade result details', async () => {
    listAll.mockResolvedValue([
      {
        id: 1,
        account_id: 11,
        account_name: 'Account A',
        model_id: 'gpt-6-astra',
        cron_expression: '0 4 * * *',
        enabled: true,
        mode: 'pelican',
        auto_pause_on_downgrade: true,
        auto_recover: false,
        auto_recover_schedulable: false
      }
    ] as any)
    listResults.mockResolvedValue([
      {
        id: 5,
        plan_id: 1,
        status: 'success',
        response_text: '',
        error_message: '',
        downgraded: true,
        downgrade_reason: '模型被上游降级为: gpt-4o-mini',
        response_model: 'gpt-4o-mini',
        latency_ms: 1200,
        started_at: '2026-10-11T00:00:00Z',
        finished_at: '2026-10-11T00:00:01Z',
        created_at: '2026-10-11T00:00:01Z'
      }
    ] as any)
    listRules.mockResolvedValue([])
    getAllGroups.mockResolvedValue([] as any)

    const wrapper = mountPanel()
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.scheduledTests.modePelican')

    // Expand the plan to load results.
    await wrapper.find('.cursor-pointer').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.detection.capability.failed')
    expect(wrapper.text()).toContain('模型被上游降级为: gpt-4o-mini')
    expect(wrapper.text()).toContain('gpt-4o-mini')
  })
})

describe('ScheduledTestsPanel task-table entry', () => {
  it('opens the requested plan with its saved schedule and type', async () => {
    listAll.mockResolvedValue([{ id: 19, account_id: 11, model_id: 'gpt-5.4', cron_expression: '*/15 * * * *', enabled: true, max_results: 50, mode: 'pelican', rule_ids: [], auto_recover: true }])
    listRules.mockResolvedValue([])
    listAccounts.mockResolvedValue({ items: [], total: 0 })
    getAllGroups.mockResolvedValue([])
    const wrapper = mountPanel({ initialPlanId: 19 })
    await wrapper.setProps({ show: true })
    await flushPromises()
    const form = wrapper.findComponent({ name: 'ScheduledDetectionPlanForm' })
    expect(form.exists()).toBe(true)
    expect(form.props('modelValue')).toMatchObject({ model_id: 'gpt-5.4', cron_expression: '*/15 * * * *', mode: 'pelican', auto_recover: true })
    wrapper.unmount()
  })
})
