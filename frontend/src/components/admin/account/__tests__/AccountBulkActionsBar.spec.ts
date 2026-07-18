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

describe('AccountBulkActionsBar', () => {
  it('提供所有页全选按钮并发出事件', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: { selectedIds: [1] }
    })

    const button = wrapper
      .findAll('button')
      .find(node => node.text() === 'admin.accounts.bulkActions.selectAllPages')

    expect(button).toBeTruthy()
    await button!.trigger('click')
    expect(wrapper.emitted('select-all-pages')).toHaveLength(1)
  })
})
