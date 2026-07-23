import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportRoutingOptions from '../ImportRoutingOptions.vue'

const STORAGE_KEY = 'sub2api.import.routing-options'
const showError = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: { getAll: vi.fn() },
    groups: { getAll: vi.fn() }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const mountOptions = () => mount(ImportRoutingOptions, {
  global: {
    stubs: {
      ProxySelector: {
        props: ['modelValue'],
        emits: ['update:modelValue'],
        template: `
          <button
            data-test="import-default-proxy"
            :data-value="modelValue"
            @click="$emit('update:modelValue', 11)"
          />
        `
      },
      GroupSelector: {
        props: ['modelValue'],
        emits: ['update:modelValue'],
        template: `
          <button
            data-test="import-default-groups"
            :data-value="modelValue.join(',')"
            @click="$emit('update:modelValue', [32])"
          />
        `
      }
    }
  }
})

describe('ImportRoutingOptions remembered defaults', () => {
  beforeEach(async () => {
    localStorage.clear()
    showError.mockReset()
    const { adminAPI } = await import('@/api/admin')
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
  })

  it('saves the current routing choices and restores them on the next mount', async () => {
    const first = mountOptions()
    await flushPromises()

    await first.get('[data-test="import-default-proxy"]').trigger('click')
    await first.get('[data-test="import-default-groups"]').trigger('click')
    await first.get('[data-test="import-auto-save-routing"]').setValue(true)

    expect(await (first.vm as any).getRequestOptions()).toEqual({
      apply_proxy_settings: true,
      default_proxy_id: 11,
      apply_group_settings: true,
      default_group_ids: [32]
    })
    first.unmount()

    const second = mountOptions()
    await flushPromises()

    expect((second.get('[data-test="import-auto-save-routing"]').element as HTMLInputElement).checked).toBe(true)
    expect(second.get('[data-test="import-default-proxy"]').attributes('data-value')).toBe('11')
    expect(second.get('[data-test="import-default-groups"]').attributes('data-value')).toBe('32')
  })

  it('falls back to current candidates when remembered ids no longer exist', async () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      auto_save: true,
      apply_proxy_settings: true,
      default_proxy_id: 99,
      apply_group_settings: true,
      default_group_ids: [98]
    }))

    const wrapper = mountOptions()
    await flushPromises()

    expect(wrapper.get('[data-test="import-default-proxy"]').attributes('data-value')).toBe('22')
    expect(wrapper.get('[data-test="import-default-groups"]').attributes('data-value')).toBe('31')
  })

  it('keeps the last saved defaults when automatic saving is turned off', async () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      auto_save: true,
      apply_proxy_settings: true,
      default_proxy_id: 22,
      apply_group_settings: true,
      default_group_ids: [31]
    }))

    const wrapper = mountOptions()
    await flushPromises()
    await wrapper.get('[data-test="import-auto-save-routing"]').setValue(false)
    await wrapper.get('[data-test="import-default-proxy"]').trigger('click')
    await wrapper.get('[data-test="import-default-groups"]').trigger('click')
    await (wrapper.vm as any).getRequestOptions()

    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')).toEqual({
      auto_save: false,
      apply_proxy_settings: true,
      default_proxy_id: 22,
      apply_group_settings: true,
      default_group_ids: [31]
    })
  })
})
