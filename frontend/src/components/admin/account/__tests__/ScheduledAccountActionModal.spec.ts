import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account } from '@/types'

const { getScheduledActionMock, scheduleActionMock, cancelScheduledActionMock } = vi.hoisted(() => ({
  getScheduledActionMock: vi.fn(),
  scheduleActionMock: vi.fn(),
  cancelScheduledActionMock: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getScheduledAction: getScheduledActionMock,
      scheduleAction: scheduleActionMock,
      cancelScheduledAction: cancelScheduledActionMock,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

import ScheduledAccountActionModal from '../ScheduledAccountActionModal.vue'

const BaseDialogStub = defineComponent({
  props: { show: Boolean, title: String },
  template: '<div v-if="show"><h2>{{ title }}</h2><slot /><slot name="footer" /></div>',
})

const account = {
  id: 20,
  name: 'scheduled-account',
  platform: 'openai',
  type: 'oauth',
  status: 'active',
  schedulable: true,
  concurrency: 4,
} as Account

function mountModal(action: 'enable_and_recover' | 'pause' = 'pause') {
  return mount(ScheduledAccountActionModal, {
    props: { show: true, account, initialAction: action },
    global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
  })
}

describe('ScheduledAccountActionModal', () => {
  beforeEach(() => {
    getScheduledActionMock.mockReset().mockResolvedValue(null)
    scheduleActionMock.mockReset().mockResolvedValue({ id: 1, account_id: 20, action: 'pause' })
    cancelScheduledActionMock.mockReset().mockResolvedValue(undefined)
  })

  it('uses the selected menu action and shows its execution summary', async () => {
    const wrapper = mountModal('enable_and_recover')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.scheduledAction.enableAndRecover')
    expect(wrapper.text()).toContain('admin.accounts.scheduledAction.targetTime')
  })

  it('converts hours and minutes into the API request', async () => {
    const wrapper = mountModal('pause')
    await flushPromises()

    await wrapper.get('[data-testid="scheduled-action-hours"]').setValue('2')
    await wrapper.get('[data-testid="scheduled-action-minutes"]').setValue('15')
    const save = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.scheduledAction.save'))
    expect(save).toBeDefined()
    await save!.trigger('click')
    await flushPromises()

    expect(scheduleActionMock).toHaveBeenCalledWith(20, {
      action: 'pause',
      hours: 2,
      minutes: 15,
    })
  })

  it('rejects zero hours and zero minutes', async () => {
    const wrapper = mountModal('pause')
    await flushPromises()

    await wrapper.get('[data-testid="scheduled-action-hours"]').setValue('0')
    await wrapper.get('[data-testid="scheduled-action-minutes"]').setValue('0')
    const save = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.scheduledAction.save'))
    expect(save?.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('admin.accounts.scheduledAction.minimumDelay')
  })

  it('shows and cancels an existing pending action', async () => {
    getScheduledActionMock.mockResolvedValue({
      id: 9,
      account_id: 20,
      action: 'pause',
      execute_at: '2026-07-20T15:00:00Z',
      status: 'pending',
      attempts: 0,
      last_error: null,
    })
    const wrapper = mountModal('pause')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.scheduledAction.currentTask')
    const cancel = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.scheduledAction.cancelTask'))
    expect(cancel).toBeDefined()
    await cancel!.trigger('click')
    await flushPromises()

    expect(cancelScheduledActionMock).toHaveBeenCalledWith(20)
    expect(wrapper.emitted('saved')).toBeTruthy()
  })
})
