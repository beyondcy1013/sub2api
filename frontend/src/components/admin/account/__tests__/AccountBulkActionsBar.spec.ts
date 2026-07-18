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

function mountBar(selectedIds: number[]) {
  return mount(AccountBulkActionsBar, {
    props: {
      selectedIds,
      proxies: [{ id: 9, name: 'proxy-9' }],
      groups: [{ id: 5, name: 'group-5' }]
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
  it('零选择时仍提供所有页全选按钮并发出事件', async () => {
    const wrapper = mountBar([])

    const button = wrapper
      .findAll('button')
      .find(node => node.text() === 'admin.accounts.bulkActions.selectAllPages')

    expect(button).toBeTruthy()
    await button!.trigger('click')
    expect(wrapper.emitted('select-all-pages')).toHaveLength(1)
  })

  it('零选择时显示代理和群组下拉框但保持禁用', () => {
    const wrapper = mountBar([])

    expect(wrapper.get('[data-test="quick-proxy-select"]').attributes()).toHaveProperty('disabled')
    expect(wrapper.get('[data-test="quick-group-select"]').attributes()).toHaveProperty('disabled')
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
})
