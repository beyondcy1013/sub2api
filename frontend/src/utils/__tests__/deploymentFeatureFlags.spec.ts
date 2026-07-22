import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { PublicSettings } from '@/types'

describe('deployment feature flags', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('balance check is opt-in', () => {
    const store = useAppStore()

    store.cachedPublicSettings = null
    expect(isFeatureFlagEnabled(FeatureFlags.balanceCheck)).toBe(false)

    store.cachedPublicSettings = { balance_check_enabled: false } as PublicSettings
    expect(isFeatureFlagEnabled(FeatureFlags.balanceCheck)).toBe(false)

    store.cachedPublicSettings = { balance_check_enabled: true } as PublicSettings
    expect(isFeatureFlagEnabled(FeatureFlags.balanceCheck)).toBe(true)
  })

  it('sticky reassignment is enabled until the backend explicitly disables it', () => {
    const store = useAppStore()

    store.cachedPublicSettings = null
    expect(isFeatureFlagEnabled(FeatureFlags.stickySessionReassignment)).toBe(true)

    store.cachedPublicSettings = { sticky_session_reassignment_enabled: false } as PublicSettings
    expect(isFeatureFlagEnabled(FeatureFlags.stickySessionReassignment)).toBe(false)

    store.cachedPublicSettings = { sticky_session_reassignment_enabled: true } as PublicSettings
    expect(isFeatureFlagEnabled(FeatureFlags.stickySessionReassignment)).toBe(true)
  })
})
