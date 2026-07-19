import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportDataModal from '../ImportDataModal.vue'

const showError = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess: vi.fn(), showWarning: vi.fn() })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: { importData: vi.fn() },
    proxies: { getAll: vi.fn() },
    groups: { getAll: vi.fn() }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const mountModal = () => mount(ImportDataModal, {
  props: { show: true },
  global: {
    stubs: {
      BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
      ProxySelector: {
        props: ['modelValue'],
        template: '<div data-test="import-default-proxy" :data-value="modelValue"></div>'
      },
      GroupSelector: {
        props: ['modelValue'],
        template: '<div data-test="import-default-groups" :data-value="modelValue.join(\',\')"></div>'
      }
    }
  }
})

const makeDataFile = () => {
  const content = JSON.stringify({
    type: 'sub2api-data',
    version: 1,
    exported_at: '2026-07-19T00:00:00Z',
    proxies: [],
    accounts: []
  })
  const file = new File([content], 'accounts.json', { type: 'application/json' })
  Object.defineProperty(file, 'text', { value: () => Promise.resolve(content) })
  return file
}

describe('ImportDataModal routing defaults', () => {
  beforeEach(async () => {
    showError.mockReset()
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockReset()
    vi.mocked(adminAPI.proxies.getAll).mockResolvedValue([
      { id: 11, name: 'proxy-one' },
      { id: 22, name: 'proxy-two' }
    ] as never)
    vi.mocked(adminAPI.groups.getAll).mockResolvedValue([
      { id: 31, name: 'group-one', platform: 'openai' }
    ] as never)
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })
  })

  it('applies the last proxy and first group by default', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()
    await flushPromises()

    expect((wrapper.get('[data-test="import-apply-default-proxy"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-test="import-apply-default-groups"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.get('[data-test="import-default-proxy"]').attributes('data-value')).toBe('22')
    expect(wrapper.get('[data-test="import-default-groups"]').attributes('data-value')).toBe('31')

    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [makeDataFile()]
    })
    await input.trigger('change')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.any(Object),
      apply_proxy_settings: true,
      default_proxy_id: 22,
      apply_group_settings: true,
      default_group_ids: [31],
      skip_default_group_bind: true
    })
  })
})
