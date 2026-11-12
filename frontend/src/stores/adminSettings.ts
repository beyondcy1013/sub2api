import { defineStore } from 'pinia'
import { ref } from 'vue'
import { adminAPI } from '@/api'
import type { CustomMenuItem } from '@/types'

export function matchPattern(pattern: string, text: string): boolean {
  const p = pattern.trim().toLowerCase()
  const t = text.trim().toLowerCase()
  if (!p) return false
  if (p === '*' || p === t) return true
  const regexStr = '^' + p.split('*').map(s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('.*') + '$'
  try {
    return new RegExp(regexStr).test(t)
  } catch {
    return false
  }
}

export function isModelGloballyExcluded(patterns: string[], model: string): boolean {
  if (!patterns || patterns.length === 0 || !model) return false
  const m = model.trim().toLowerCase()
  const candidates = new Set<string>()
  const add = (v: string) => {
    v = v.trim().toLowerCase()
    if (!v) return
    candidates.add(v)
    if (v.startsWith('models/')) candidates.add(v.slice(7))
    if (v.endsWith('-thinking')) candidates.add(v.slice(0, -9))
  }
  add(m)
  const parts = m.split(/[/:]/)
  for (const part of parts) {
    add(part)
  }
  for (const pattern of patterns) {
    for (const candidate of candidates) {
      if (matchPattern(pattern, candidate)) return true
    }
  }
  return false
}

export const useAdminSettingsStore = defineStore('adminSettings', () => {
  const loaded = ref(false)
  const loading = ref(false)

  const readCachedBool = (key: string, defaultValue: boolean): boolean => {
    try {
      const raw = localStorage.getItem(key)
      if (raw === 'true') return true
      if (raw === 'false') return false
    } catch {
      // ignore localStorage failures
    }
    return defaultValue
  }

  const writeCachedBool = (key: string, value: boolean) => {
    try {
      localStorage.setItem(key, value ? 'true' : 'false')
    } catch {
      // ignore localStorage failures
    }
  }

  const readCachedString = (key: string, defaultValue: string): string => {
    try {
      const raw = localStorage.getItem(key)
      if (typeof raw === 'string' && raw.length > 0) return raw
    } catch {
      // ignore localStorage failures
    }
    return defaultValue
  }

  const writeCachedString = (key: string, value: string) => {
    try {
      localStorage.setItem(key, value)
    } catch {
      // ignore localStorage failures
    }
  }

  // Default open, but honor cached value to reduce UI flicker on first paint.
  const opsMonitoringEnabled = ref(readCachedBool('ops_monitoring_enabled_cached', true))
  const opsRealtimeMonitoringEnabled = ref(readCachedBool('ops_realtime_monitoring_enabled_cached', true))
  const opsQueryModeDefault = ref(readCachedString('ops_query_mode_default_cached', 'auto'))
  const paymentEnabled = ref(readCachedBool('payment_enabled_cached', false))
  const customMenuItems = ref<CustomMenuItem[]>([])
  const globalModelExcludePatterns = ref<string[]>([])

  async function fetch(force = false): Promise<void> {
    if (loaded.value && !force) return
    if (loading.value) return

    loading.value = true
    try {
      const [settings, paymentConfigResp] = await Promise.all([
        adminAPI.settings.getSettings(),
        adminAPI.payment.getConfig()
      ])
      opsMonitoringEnabled.value = settings.ops_monitoring_enabled ?? true
      writeCachedBool('ops_monitoring_enabled_cached', opsMonitoringEnabled.value)

      opsRealtimeMonitoringEnabled.value = settings.ops_realtime_monitoring_enabled ?? true
      writeCachedBool('ops_realtime_monitoring_enabled_cached', opsRealtimeMonitoringEnabled.value)

      opsQueryModeDefault.value = settings.ops_query_mode_default || 'auto'
      writeCachedString('ops_query_mode_default_cached', opsQueryModeDefault.value)

      customMenuItems.value = Array.isArray(settings.custom_menu_items) ? settings.custom_menu_items : []
      globalModelExcludePatterns.value = Array.isArray(settings.global_model_exclude_patterns)
        ? settings.global_model_exclude_patterns
        : []

      paymentEnabled.value = paymentConfigResp.data?.enabled ?? false
      writeCachedBool('payment_enabled_cached', paymentEnabled.value)

      loaded.value = true
    } catch (err) {
      // Keep cached/default value: do not "flip" the UI based on a transient fetch failure.
      console.error('[adminSettings] Failed to fetch settings:', err)
    } finally {
      loading.value = false
    }
  }

  function setOpsMonitoringEnabledLocal(value: boolean) {
    opsMonitoringEnabled.value = value
    writeCachedBool('ops_monitoring_enabled_cached', value)
    loaded.value = true
  }

  function setOpsRealtimeMonitoringEnabledLocal(value: boolean) {
    opsRealtimeMonitoringEnabled.value = value
    writeCachedBool('ops_realtime_monitoring_enabled_cached', value)
    loaded.value = true
  }

  function setPaymentEnabledLocal(value: boolean) {
    paymentEnabled.value = value
    writeCachedBool('payment_enabled_cached', value)
    loaded.value = true
  }

  function setOpsQueryModeDefaultLocal(value: string) {
    opsQueryModeDefault.value = value || 'auto'
    writeCachedString('ops_query_mode_default_cached', opsQueryModeDefault.value)
    loaded.value = true
  }

  // Keep UI consistent if we learn that ops is disabled via feature-gated 404s.
  // (event is dispatched from the axios interceptor)
  let eventHandlerCleanup: (() => void) | null = null

  function initializeEventListeners() {
    if (eventHandlerCleanup) return

    try {
      const handler = () => {
        setOpsMonitoringEnabledLocal(false)
      }
      window.addEventListener('ops-monitoring-disabled', handler)
      eventHandlerCleanup = () => {
        window.removeEventListener('ops-monitoring-disabled', handler)
      }
    } catch {
      // ignore window access failures (SSR)
    }
  }

  if (typeof window !== 'undefined') {
    initializeEventListeners()
  }

  function isModelExcluded(model: string): boolean {
    return isModelGloballyExcluded(globalModelExcludePatterns.value, model)
  }

  return {
    loaded,
    loading,
    opsMonitoringEnabled,
    opsRealtimeMonitoringEnabled,
    opsQueryModeDefault,
    paymentEnabled,
    customMenuItems,
    globalModelExcludePatterns,
    isModelGloballyExcluded: isModelExcluded,
    fetch,
    setOpsMonitoringEnabledLocal,
    setOpsRealtimeMonitoringEnabledLocal,
    setPaymentEnabledLocal,
    setOpsQueryModeDefaultLocal
  }
})
