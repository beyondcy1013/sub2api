import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

const SelectStub = {
  props: ['modelValue', 'options', 'disabled', 'placeholder'],
  emits: ['update:modelValue', 'change'],
  template: `
    <select
      v-bind="$attrs"
      :disabled="disabled"
      :value="modelValue ?? ''"
      @change="
        $emit('update:modelValue', Number($event.target.value));
        $emit('change', Number($event.target.value), null)
      "
    >
      <option value="">{{ placeholder }}</option>
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `
}

function mountBar(selectedIds: number[], testingSelected = false, totalResults = 45, overrides: Record<string, unknown> = {}) {
  return mount(AccountBulkActionsBar, {
    props: {
      selectedIds,
      totalResults,
      selectingAll: false,
      allResultsSelected: false,
      testingSelected,
      proxies: [{ id: 9, name: 'proxy-9' }],
      groups: [{ id: 5, name: 'group-5' }],
      ...overrides
    } as any,
    global: { stubs: { Select: SelectStub } }
  })
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('AccountBulkActionsBar', () => {
  it('零选择时仍提供所有结果全选按钮并发出事件', async () => {
    const wrapper = mountBar([])

    const button = wrapper
      .findAll('button')
      .find(node => node.text().includes('admin.accounts.bulkActions.selectAllResults'))

    expect(button).toBeTruthy()
    await button!.trigger('click')
    expect(wrapper.emitted('select-all-results')).toHaveLength(1)
  })

  it('零选择时显示代理和群组下拉框但保持禁用', () => {
    const wrapper = mountBar([])

    expect(wrapper.get('[data-test="quick-proxy-select"]').attributes()).toHaveProperty('disabled')
    expect(wrapper.get('[data-test="quick-group-select"]').attributes()).toHaveProperty('disabled')
  })

  it('在选择控制旁提供账号过滤框并发出查询更新', async () => {
    const wrapper = mountBar([1], false, 45, { searchQuery: 'before' })
    const input = wrapper.get('[data-test="bulk-account-filter"]')

    expect((input.element as HTMLInputElement).value).toBe('before')
    await input.setValue('account notes')

    expect(wrapper.emitted('update:search-query')).toEqual([['account notes']])
  })

  it('仅在启用已删除视图操作时提供彻底删除按钮', async () => {
    const normal = mountBar([1, 2])
    expect(normal.find('[data-test="bulk-permanent-delete"]').exists()).toBe(false)

    const deleted = mountBar([1, 2], false, 45, { showDelete: false, showPermanentDelete: true })
    const button = deleted.get('[data-test="bulk-permanent-delete"]')
    await button.trigger('click')

    expect(deleted.find('[data-test="bulk-delete"]').exists()).toBe(false)
    expect(deleted.emitted('permanent-delete')).toEqual([[]])
  })

  it('彻底删除运行时禁用按钮并显示进度文案', () => {
    const wrapper = mountBar([1], false, 45, { showPermanentDelete: true, permanentDeleting: true })
    const button = wrapper.get('[data-test="bulk-permanent-delete"]')

    expect(button.attributes()).toHaveProperty('disabled')
    expect(button.text()).toBe('admin.accounts.bulkActions.permanentlyDeleting')
  })

  it('选择账号后可直接选择代理和群组', async () => {
    const wrapper = mountBar([1, 2])

    expect(wrapper.get('[data-test="quick-proxy-select"]').text()).toContain('admin.accounts.noProxy')
    expect(wrapper.get('[data-test="quick-group-select"]').text()).toContain('admin.accounts.bulkActions.noGroup')

    await wrapper.get('[data-test="quick-proxy-select"]').setValue('9')
    await wrapper.get('[data-test="quick-group-select"]').setValue('5')

    expect(wrapper.emitted('quick-set-proxy')).toEqual([[9]])
    expect(wrapper.emitted('quick-set-group')).toEqual([[5]])
  })

  it('preserves the upstream billing probe action from v0.1.166', async () => {
    const wrapper = mountBar([1])

    const button = wrapper.findAll('button').find(item =>
      item.text().includes('admin.accounts.bulkActions.probeUpstreamBilling')
    )

    expect(button).toBeDefined()
    await button!.trigger('click')
    expect(wrapper.emitted('probe-upstream-billing')).toHaveLength(1)
  })

  it('选择账号后提供批量测试并标记按钮', async () => {
    const wrapper = mountBar([1, 2])

    const button = wrapper.get('[data-test="batch-test-and-mark"]')
    await button.trigger('click')

    expect(wrapper.emitted('test-and-mark')).toEqual([[]])
  })

  it('批量测试运行时禁用按钮并显示进度文案', () => {
    const wrapper = mountBar([1, 2], true)

    const button = wrapper.get('[data-test="batch-test-and-mark"]')
    expect(button.attributes()).toHaveProperty('disabled')
    expect(button.text()).toBe('admin.accounts.bulkActions.testingAndMarking')
  })
})
