import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountQualityToggle from '../AccountQualityToggle.vue'
import AccountQualityOverrides from '../AccountQualityOverrides.vue'
import { accountQualityEnabled } from '../settings'
import { defaultRequestQualitySettings } from '@/utils/accountDetection'
import type { Account } from '@/types'
const api = vi.hoisted(() => ({ get: vi.fn(), save: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { getRequestQualityProbeSettings: api.get, updateRequestQualityProbeSettings: api.save } } }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))
const account = { id: 42, status: 'active', name: 'A' } as Account
const mountToggle = (id = 42) => mount(AccountQualityToggle, { props: { account: { ...account, id } }, global: { plugins: [createPinia()] } })
describe('account quality participation', () => {
  beforeEach(() => { vi.clearAllMocks(); api.get.mockResolvedValue(defaultRequestQualitySettings()); api.save.mockImplementation(async value => value) })
  it('displays enabled or disabled status with modify hint on hover', async () => {
    const wrapper = mountToggle()
    await flushPromises()
    expect(wrapper.text()).toBe('admin.accounts.detection.enabled')
    expect(wrapper.get('span').attributes('title')).toBe('admin.accounts.detection.modifyInCenterHint')

    api.get.mockResolvedValue({ ...defaultRequestQualitySettings(), account_overrides: { 42: false } })
    const disabledWrapper = mountToggle()
    await flushPromises()
    expect(disabledWrapper.text()).toBe('admin.accounts.detection.disabled')
    expect(disabledWrapper.get('span').attributes('title')).toBe('admin.accounts.detection.modifyInCenterHint')
    wrapper.unmount()
    disabledWrapper.unmount()
  })
  it('does not pretend a failed settings read is a disabled account', async () => {
    api.get.mockRejectedValue(new Error('offline'))
    const wrapper = mountToggle()
    await flushPromises()
    expect(wrapper.text()).toBe('-')
    expect(wrapper.get('span').attributes('title')).toBe('admin.accounts.detection.modifyInCenterHint')
    wrapper.unmount()
  })
  it('uses overrides before dynamic or selected account scope', () => {
    const settings = { ...defaultRequestQualitySettings(), account_scope: 'selected' as const, account_ids: [], account_overrides: { 42: true } }
    expect(accountQualityEnabled(settings, account)).toBe(true)
    settings.account_overrides[42] = false; settings.account_ids = [42] as never[]
    expect(accountQualityEnabled(settings, account)).toBe(false)
    expect(accountQualityEnabled({ ...settings, account_scope: 'error_or_downgraded', account_overrides: {} }, { ...account, status: 'error' })).toBe(true)
  })
  it('shows the same override in global settings and permits resetting it', async () => {
    const wrapper = mount(AccountQualityOverrides, { props: { accounts: [account], modelValue: { 42: false } } })
    expect(wrapper.text()).toContain('A #42')
    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([{42: true}])
    await wrapper.findAll('button')[1]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[1]).toEqual([{}])
  })
})
