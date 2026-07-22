import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import TrashBinModal from '../TrashBinModal.vue'

const {
  listTrashed,
  restoreFromTrash,
  permanentDelete
} = vi.hoisted(() => ({
  listTrashed: vi.fn(),
  restoreFromTrash: vi.fn(),
  permanentDelete: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      listTrashed,
      restoreFromTrash,
      permanentDelete
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const BaseDialogStub = {
  props: ['show'],
  template: '<div v-if="show"><slot /></div>'
}

const ConfirmDialogStub = {
  props: ['show'],
  emits: ['confirm', 'cancel'],
  template: '<button v-if="show" data-test="confirm-perm-delete" @click="$emit(\'confirm\')">confirm</button>'
}

function mountModal() {
  return mount(TrashBinModal, {
    props: {
      show: false
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        LoadingSpinner: true,
        Icon: true
      }
    }
  })
}

describe('TrashBinModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listTrashed.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 })
  })

  it('loads trashed accounts when shown', async () => {
    const wrapper = mountModal()
    await flushPromises()
    expect(listTrashed).not.toHaveBeenCalled()

    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(listTrashed).toHaveBeenCalledTimes(1)
  })

  it('displays trash accounts and restore button', async () => {
    listTrashed.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'deleted-acc',
          platform: 'openai',
          type: 'apikey',
          status: 'inactive',
          schedulable: false,
          concurrency: 4,
          notes: 'archived note',
          created_at: '2025-12-01T00:00:00Z',
          last_used_at: '2025-12-31T12:00:00Z',
          deleted_at: '2026-01-01T00:00:00Z',
          usage_stats: {
            requests: 1250,
            tokens: 2450000,
            cost: 12.5,
            standard_cost: 10,
            user_cost: 18.75
          }
        }
      ],
      total: 1,
      page: 1,
      page_size: 20
    })

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.text()).toContain('deleted-acc')
    expect(wrapper.text()).toContain('archived note')
    expect(wrapper.get('[data-test="trash-created-at"]').text()).not.toBe('-')
    expect(wrapper.get('[data-test="trash-deleted-at"]').text()).not.toBe('-')
    expect(wrapper.get('[data-test="trash-requests"]').text()).toContain('1,250')
    expect(wrapper.get('[data-test="trash-tokens"]').text()).toContain('2.5M')
    expect(wrapper.get('[data-test="trash-account-cost"]').text()).toContain('12.50')
    expect(wrapper.get('[data-test="trash-user-cost"]').text()).toContain('18.75')
    expect(wrapper.text()).toContain('admin.accounts.restoreFromTrash')
    expect(wrapper.text()).toContain('admin.accounts.permanentDelete')
  })

  it('calls restoreFromTrash API when restore button clicked', async () => {
    restoreFromTrash.mockResolvedValue({ message: 'ok' })
    listTrashed.mockResolvedValue({
      items: [
        { id: 42, name: 'restore-me', platform: 'openai', type: 'apikey', deleted_at: '2026-01-01T00:00:00Z' }
      ],
      total: 1,
      page: 1,
      page_size: 20
    })

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const restoreBtn = wrapper.findAll('button').find(b => b.text().includes('admin.accounts.restoreFromTrash'))
    expect(restoreBtn).toBeDefined()
    await restoreBtn!.trigger('click')
    await flushPromises()

    expect(restoreFromTrash).toHaveBeenCalledWith(42)
    expect(wrapper.emitted('restored')).toBeTruthy()
  })

  it('shows empty state when no trashed accounts', async () => {
    listTrashed.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 })

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.trashBinEmpty')
  })

  it('permanently deletes after confirmation', async () => {
    permanentDelete.mockResolvedValue({ message: 'deleted' })
    listTrashed.mockResolvedValue({
      items: [
        { id: 99, name: 'perm-del', platform: 'openai', type: 'apikey', deleted_at: '2026-01-01T00:00:00Z' }
      ],
      total: 1,
      page: 1,
      page_size: 20
    })

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const permBtn = wrapper.findAll('button').find(b => b.text().includes('admin.accounts.permanentDelete'))
    await permBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="confirm-perm-delete"]').exists()).toBe(true)
    await wrapper.get('[data-test="confirm-perm-delete"]').trigger('click')
    await flushPromises()

    expect(permanentDelete).toHaveBeenCalledWith(99)
  })
})
