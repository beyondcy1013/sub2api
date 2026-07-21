import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountTableActions from '../AccountTableActions.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('AccountTableActions scheduling rules', () => {
  it('exposes the scheduling-rules command in the account table toolbar', async () => {
    const wrapper = mount(AccountTableActions, {
      props: { loading: false, showFilters: false, recycled: false }
    })

    const button = wrapper.get('button:nth-of-type(2)')
    expect(button.text()).toContain('admin.accounts.schedulingRules.title')
    await button.trigger('click')

    expect(wrapper.emitted('scheduling-rules')).toHaveLength(1)
  })
})
