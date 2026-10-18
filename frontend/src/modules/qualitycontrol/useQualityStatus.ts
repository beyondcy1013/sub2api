import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import { loadQualitySettings } from './api'
import { adminAPI } from '@/api/admin'
import type { AccountQualitySummary } from '@/types'

// One bounded batch per table page, shared by every status cell.
export function useQualityStatus(accounts: Ref<{ id: number }[]>, revision: Ref<number>) {
  const summaries = ref<Record<number, AccountQualitySummary>>({})
  const minutes = ref(5)
  const now = ref(Date.now())
  const state = ref<'loading' | 'ready' | 'error'>('loading')
  let generation = 0
  async function refresh() {
    const current = ++generation
    const ids = accounts.value.map(account => account.id)
    if (!ids.length) { summaries.value = {}; state.value = 'ready'; return }
    state.value = 'loading'
    try {
      const [settings, records] = await Promise.all([
        loadQualitySettings(), adminAPI.accounts.getQualityStatus(ids),
      ])
      if (current !== generation) return
      minutes.value = settings.status_recent_minutes || 5
      summaries.value = Object.fromEntries(records.map(record => [record.account_id, record]))
      now.value = Date.now()
      state.value = 'ready'
    } catch { if (current === generation) state.value = 'error' }
  }
  watch([accounts, revision], () => { void refresh() }, { immediate: true })
  let timer: ReturnType<typeof setInterval> | undefined
  onMounted(() => { timer = setInterval(() => {
    now.value = Date.now()
    if (!document.hidden) void refresh()
  }, 30000) })
  onUnmounted(() => { generation++; if (timer) clearInterval(timer) })
  return { summaries, minutes, now, state }
}
