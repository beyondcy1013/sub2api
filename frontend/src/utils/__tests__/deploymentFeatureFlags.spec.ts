import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { PublicSettings } from '@/types'

describe('deployment feature flags', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it.each([
    ['balance check', FeatureFlags.balanceCheck, 'balance_check_enabled'],
    ['sticky reassignment', FeatureFlags.stickySessionReassignment, 'sticky_session_reassignment_enabled'],
  ] as const)('%s is opt-in', (_name, flag, key) => {
    const store = useAppStore()

    store.cachedPublicSettings = null
    expect(isFeatureFlagEnabled(flag)).toBe(false)

    store.cachedPublicSettings = { [key]: false } as PublicSettings
    expect(isFeatureFlagEnabled(flag)).toBe(false)

    store.cachedPublicSettings = { [key]: true } as PublicSettings
    expect(isFeatureFlagEnabled(flag)).toBe(true)
  })
})
