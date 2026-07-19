import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import EnhancedImportDataModal from '../EnhancedImportDataModal.vue'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, showWarning: vi.fn() })
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

const mountModal = () => mount(EnhancedImportDataModal, {
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
    vi.mocked(adminAPI.proxies.getAll).mockReset()
    vi.mocked(adminAPI.groups.getAll).mockReset()
    vi.mocked(adminAPI.proxies.getAll).mockResolvedValue([
      { id: 11, name: 'proxy-one' },
      { id: 22, name: 'proxy-two' }
    ] as never)
    vi.mocked(adminAPI.groups.getAll).mockResolvedValue([
      { id: 31, name: 'group-one', platform: 'openai' },
      { id: 32, name: 'group-two', platform: 'anthropic' }
    ] as never)
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
    await flushPromises()

    expect((wrapper.get('[data-test="import-apply-default-proxy"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-test="import-apply-default-groups"]').element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.get('[data-test="import-default-proxy"]').attributes('data-value')).toBe('22')
    expect(wrapper.get('[data-test="import-default-groups"]').attributes('data-value')).toBe('31')

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
      apply_proxy_settings: true,
      default_proxy_id: 22,
      apply_group_settings: true,
      default_group_ids: [31],
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
      apply_proxy_settings: true,
      default_proxy_id: 22,
      apply_group_settings: true,
      default_group_ids: [31],
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

  it('requires a selected file in file mode', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('reports malformed pasted JSON', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()

    await wrapper.find('[data-test="enhanced-import-mode-text"]').trigger('click')
    await wrapper.find('textarea').setValue('{')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.enhancedImportInvalidJson')
    expect(adminAPI.accounts.importData).not.toHaveBeenCalled()
  })

  it('opens the hidden file picker from the choose-file button', async () => {
    const wrapper = mountModal()
    const input = wrapper.find('input[type="file"]')
    const click = vi.spyOn(input.element as HTMLInputElement, 'click')

    await wrapper.findAll('button.btn-secondary')[0]!.trigger('click')

    expect(click).toHaveBeenCalledTimes(1)
  })

  it('imports multiple JSON files dropped on the file target', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()
    const dropTarget = wrapper.find('.border-dashed')
    const files = [
      makeJsonFile('codex.json', {
        type: 'codex',
        email: 'codex@example.com',
        refresh_token: 'codex-refresh'
      }),
      makeJsonFile('claude.json', {
        type: 'claude',
        email: 'claude@example.com',
        access_token: 'claude-access'
      })
    ]

    await dropTarget.trigger('dragenter')
    expect(dropTarget.classes()).toContain('border-primary-400')

    await dropTarget.trigger('dragleave')
    expect(dropTarget.classes()).not.toContain('border-primary-400')

    await dropTarget.trigger('drop', { dataTransfer: { files } })
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.objectContaining({
        accounts: [
          expect.objectContaining({ platform: 'openai', type: 'oauth' }),
          expect.objectContaining({ platform: 'anthropic', type: 'oauth' })
        ]
      }),
      apply_proxy_settings: true,
      default_proxy_id: 22,
      apply_group_settings: true,
      default_group_ids: [31],
      skip_default_group_bind: true
    })
  })

  it('can disable applying the selected proxy and groups', async () => {
    const { adminAPI } = await import('@/api/admin')
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.get('[data-test="import-apply-default-proxy"]').setValue(false)
    await wrapper.get('[data-test="import-apply-default-groups"]').setValue(false)
    await wrapper.get('[data-test="enhanced-import-mode-text"]').trigger('click')
    await wrapper.find('textarea').setValue(JSON.stringify({
      exported_at: '2026-07-19T00:00:00Z',
      proxies: [],
      accounts: [{ name: 'native-account' }]
    }))
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: expect.any(Object),
      apply_proxy_settings: false,
      apply_group_settings: false,
      skip_default_group_bind: true
    })
  })

  it('keeps partial import results refreshable when the modal closes', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockResolvedValueOnce({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 1
    })
    const wrapper = mountModal()

    await wrapper.find('[data-test="enhanced-import-mode-text"]').trigger('click')
    await wrapper.find('textarea').setValue(JSON.stringify({
      exported_at: '2026-07-19T00:00:00Z',
      proxies: [],
      accounts: [{ name: 'created' }, { name: 'failed' }]
    }))
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportCompletedWithErrors')
    expect(wrapper.emitted('imported')).toBeUndefined()

    await wrapper.find('button.btn-secondary').trigger('click')
    expect(wrapper.emitted('imported')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('surfaces an unexpected import API error', async () => {
    const { adminAPI } = await import('@/api/admin')
    vi.mocked(adminAPI.accounts.importData).mockRejectedValueOnce(new Error('import unavailable'))
    const wrapper = mountModal()

    await wrapper.find('[data-test="enhanced-import-mode-text"]').trigger('click')
    await wrapper.find('textarea').setValue(JSON.stringify({
      exported_at: '2026-07-19T00:00:00Z',
      proxies: [],
      accounts: [{ name: 'native-account' }]
    }))
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('import unavailable')
  })
})
