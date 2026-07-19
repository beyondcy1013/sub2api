import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const SelectStub = {
  inheritAttrs: false,
  props: ['modelValue', 'options', 'disabled', 'placeholder'],
  template: '<select></select>'
}

const mountBar = (refreshingUsage = false) => mount(AccountBulkActionsBar, {
  props: {
    selectedIds: [],
    refreshingUsage
  } as any,
  global: { stubs: { Select: SelectStub } }
})

describe('AccountBulkActionsBar bulk usage refresh', () => {
  it('shows bulk usage refresh immediately before bulk update and emits the action', async () => {
    const wrapper = mountBar()
    const actionButtons = wrapper.findAll('[data-test="bulk-primary-action"]')

    expect(actionButtons.map(button => button.text())).toEqual([
      'admin.accounts.bulkActions.refreshUsage',
      'admin.accounts.bulkEdit.submit'
    ])

    await actionButtons[0].trigger('click')
    expect(wrapper.emitted('refresh-usage')).toHaveLength(1)
  })

  it('disables both filtered-scope actions while usage refresh is running', () => {
    const wrapper = mountBar(true)
    const actionButtons = wrapper.findAll('[data-test="bulk-primary-action"]')

    expect(actionButtons[0].text()).toBe('admin.accounts.bulkActions.refreshingUsage')
    expect(actionButtons.every(button => button.attributes('disabled') !== undefined)).toBe(true)
  })
})
