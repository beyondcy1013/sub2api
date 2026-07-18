import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import EnhancedImportDataModal from '../EnhancedImportDataModal.vue'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, showWarning: vi.fn() })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { accounts: { importData: vi.fn() } }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const mountModal = () => mount(EnhancedImportDataModal, {
  props: { show: true },
  global: {
    stubs: {
      BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
    }
  }
})

const makeJsonFile = (name: string, value: unknown) => {
  const content = JSON.stringify(value)
  const file = new File([content], name, { type: 'application/json' })
  Object.defineProperty(file, 'text', { value: () => Promise.resolve(content) })
  return file
}

describe('EnhancedImportDataModal', () => {
  beforeEach(async () => {
    showError.mockReset()
    showSuccess.mockReset()
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockReset()
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0
    })
  })

  it('imports pasted native sub2api JSON text', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    await wrapper.find('[data-test="enhanced-import-mode-text"]').trigger('click')
    await wrapper.find('textarea').setValue(JSON.stringify({
      exported_at: '2026-07-19T00:00:00Z',
      proxies: [],
      accounts: [{ name: 'native-account' }]
    }))
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({ accounts: [{ name: 'native-account' }] }),
      skip_default_group_bind: true
    })
    expect(wrapper.emitted('imported')).toHaveLength(1)
  })

  it('imports selected CLIProxyAPI JSON files', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [makeJsonFile('codex.json', {
        type: 'codex',
        email: 'codex@example.com',
        refresh_token: 'refresh-token'
      })]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({
        accounts: [expect.objectContaining({
          name: 'codex@example.com',
          platform: 'openai',
          type: 'oauth'
        })]
      }),
      skip_default_group_bind: true
    })
  })

  it('rejects unsupported pasted CLIProxyAPI provider JSON', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    await wrapper.find('[data-test="enhanced-import-mode-text"]').trigger('click')
    await wrapper.find('textarea').setValue('{"type":"kimi","access_token":"token"}')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.enhancedImportUnsupportedProvider')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })
})
