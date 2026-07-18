import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import VersionBadge from '../VersionBadge.vue'

const { fetchVersion } = vi.hoisted(() => ({
  fetchVersion: vi.fn()
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAdmin: true }),
  useAppStore: () => ({
    versionLoading: false,
    currentVersion: '',
    latestVersion: '',
    hasUpdate: false,
    buildType: 'source',
    releaseInfo: null,
    fetchVersion,
    clearVersionCache: vi.fn()
  })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api/admin/system', () => ({
  performUpdate: vi.fn(),
  restartService: vi.fn(),
  getRollbackVersions: vi.fn(),
  rollback: vi.fn()
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: false, copyToClipboard: vi.fn() })
}))

describe('VersionBadge', () => {
  it('only displays the installed version without checking or offering updates', () => {
    const wrapper = shallowMount(VersionBadge, {
      props: { version: '1.2.3' }
    })

    expect(wrapper.text()).toContain('v1.2.3')
    expect(wrapper.find('button').exists()).toBe(false)
    expect(fetchVersion).not.toHaveBeenCalled()
  })
})
