import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SuperPrioritySettingsModal from '../SuperPrioritySettingsModal.vue'

const { getSettings, updateSettings, activate, deactivate } = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  activate: vi.fn(),
  deactivate: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    superPriority: {
      get: getSettings,
      update: updateSettings,
      activate,
      deactivate,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

function mountModal() {
  return mount(SuperPrioritySettingsModal, {
    props: { show: false },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div><slot /><slot name="footer" /></div>',
        },
        Icon: true,
      },
    },
  })
}

describe('SuperPrioritySettingsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getSettings.mockResolvedValue({
      mode: 'normal',
      base_strategy: 'default',
      failure_threshold: 2,
      check_interval: '@every 1m',
      test_model_id: '',
      test_prompt: '',
      activated_at: '',
      demoted_at: '',
      is_active: false,
    })
    updateSettings.mockResolvedValue({ message: 'updated', restart_required: true })
    activate.mockResolvedValue({ message: 'activated' })
    deactivate.mockResolvedValue({ message: 'deactivated' })
  })

  it('loads and saves the selected base scheduling strategy', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const strategy = wrapper.get('select')
    expect((strategy.element as HTMLSelectElement).value).toBe('default')
    await strategy.setValue('lowest_cost')

    const save = wrapper.findAll('button').find((button) => button.text() === 'common.save')
    expect(save).toBeTruthy()
    await save!.trigger('click')
    await flushPromises()

    expect(updateSettings).toHaveBeenCalledWith({
      base_strategy: 'lowest_cost',
      failure_threshold: 2,
      check_interval: '@every 1m',
      test_model_id: '',
      test_prompt: '',
    })
  })

  it('activates only the overlay and does not submit paused-account controls', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const activateButton = wrapper.findAll('button').find((button) => button.text() === 'common.activate')
    expect(activateButton).toBeTruthy()
    await activateButton!.trigger('click')
    await flushPromises()

    expect(activate).toHaveBeenCalledWith()
    expect(wrapper.find('[data-test="paused-account-list"]').exists()).toBe(false)
  })
})
