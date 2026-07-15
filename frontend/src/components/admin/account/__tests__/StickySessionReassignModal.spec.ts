import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account } from '@/types'

const { getStickySessionSummaryMock, reassignStickySessionsMock } = vi.hoisted(() => ({
  getStickySessionSummaryMock: vi.fn(),
  reassignStickySessionsMock: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getStickySessionSummary: getStickySessionSummaryMock,
      reassignStickySessions: reassignStickySessionsMock,
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

import StickySessionReassignModal from '../StickySessionReassignModal.vue'
import zhAccounts from '@/i18n/locales/zh/admin/accounts'

const BaseDialogStub = defineComponent({
  props: { show: Boolean },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

const target = {
  id: 20,
  name: 'idle-target',
  platform: 'openai',
  type: 'apikey',
  status: 'active',
  schedulable: true,
  concurrency: 4,
  current_concurrency: 1,
  group_ids: [2],
} as Account

describe('StickySessionReassignModal', () => {
  beforeEach(() => {
    getStickySessionSummaryMock.mockReset().mockResolvedValue({
      target_account_id: 20,
      groups: [{
        group_id: 2,
        group_name: 'main',
        total: 8,
        protected_response_bindings: 1,
        sources: [
          {
            account_id: 12,
            account_name: 'busy-source',
            count: 8,
            current_concurrency: 4,
            concurrency: 4,
            recent_counts: { '60': 1, '300': 3, '900': 8, '3600': 8 },
            recent_sessions: [
              { session_suffix: 'aaaaaaaa', active_ago_seconds: 10 },
              { session_suffix: 'bbbbbbbb', active_ago_seconds: 600 },
            ],
          },
        ],
      }],
    })
    reassignStickySessionsMock.mockReset().mockResolvedValue({ moved: 3, remaining_source_bindings: 5 })
  })

  it('shows the for-user concurrency explanation', async () => {
    const wrapper = mount(StickySessionReassignModal, {
      props: { show: true, account: target },
      global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.stickySessions.userLimitWarning')
  })

  it('explains that sticky account skew can starve local user slots', () => {
    const warning = zhAccounts.accounts.stickySessions.userLimitWarning

    expect(warning).toContain('sub2api')
    expect(warning).toContain('等待账号槽位')
    expect(warning).toContain('可缓解')
    expect(warning).not.toContain('迁移账号绑定无法修复')
    expect(warning).not.toContain('用户管理')
  })

  it('defaults the move count to target account headroom', async () => {
    const wrapper = mount(StickySessionReassignModal, {
      props: { show: true, account: target },
      global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
    })
    await flushPromises()

    expect((wrapper.get('input[type="number"]').element as HTMLInputElement).value).toBe('3')
  })

  it('defaults to the last five minutes and hides older session candidates', async () => {
    const wrapper = mount(StickySessionReassignModal, {
      props: { show: true, account: target },
      global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
    })
    await flushPromises()

    expect(wrapper.get<HTMLSelectElement>('[data-testid="sticky-active-window"]').element.value).toBe('300')
    expect(wrapper.text()).toContain('…aaaaaaaa')
    expect(wrapper.text()).not.toContain('…bbbbbbbb')
  })

  it('sends the selected activity window to the backend', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mount(StickySessionReassignModal, {
      props: { show: true, account: target },
      global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
    })
    await flushPromises()

    const submitButton = wrapper.findAll('button').find(button =>
      button.text().includes('admin.accounts.stickySessions.submit')
    )
    expect(submitButton).toBeDefined()
    await submitButton!.trigger('click')
    await flushPromises()

    expect(reassignStickySessionsMock).toHaveBeenCalledWith(20, {
      group_id: 2,
      source_account_id: 12,
      count: 3,
      active_within_seconds: 300,
    })
  })
})
