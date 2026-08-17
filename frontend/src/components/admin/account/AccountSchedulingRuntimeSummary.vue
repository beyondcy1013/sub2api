<template>
  <div
    data-testid="account-scheduling-runtime"
    class="relative flex h-9 w-full min-w-0 items-center gap-2 overflow-hidden border-l-2 px-2 sm:w-[19rem]"
    :class="toneClass"
    role="group"
    :aria-label="t('admin.accounts.schedulingRules.runtimeLabel')"
  >
    <Icon
      :name="busy ? 'refresh' : 'clock'"
      size="sm"
      class="shrink-0"
      :class="{ 'animate-spin': busy }"
    />
    <div class="min-w-0 flex-1">
      <div class="flex min-w-0 items-center gap-1.5 text-xs leading-4">
        <span class="shrink-0 font-semibold text-gray-700 dark:text-gray-200">
          {{ t('admin.accounts.schedulingRules.runtimeLabel') }}
        </span>
        <span
          data-testid="account-scheduling-runtime-state"
          class="truncate font-medium tabular-nums text-gray-700 dark:text-gray-200"
          :title="runtimeStateText"
        >
          {{ runtimeStateText }}
        </span>
      </div>
      <div
        data-testid="account-scheduling-runtime-result"
        class="truncate text-[11px] leading-4 text-gray-500 dark:text-gray-400"
        :title="latestResultText"
      >
        {{ latestResultText }}
      </div>
    </div>
    <div
      v-if="runtimeProgressPercent !== null"
      data-testid="account-scheduling-runtime-progress"
      class="absolute inset-x-0 bottom-0 h-0.5 bg-gray-200/70 dark:bg-dark-600/70"
      role="progressbar"
      :aria-label="t('admin.accounts.schedulingRules.runtimeProgress')"
      aria-valuemin="0"
      aria-valuemax="100"
      :aria-valuenow="runtimeProgressPercent"
    >
      <div
        class="h-full bg-current opacity-70 transition-[width] duration-500"
        :style="{ width: `${runtimeProgressPercent}%` }"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import Icon from '@/components/icons/Icon.vue'
import type { SchedulingLivenessRuntimeStatus } from '@/api/admin/superPriority'
import { getSchedulingRuntimeCountdown } from '@/utils/schedulingRuntimeCountdown'

const emit = defineEmits<{
  (event: 'upstream-billing-completed'): void
}>()
const { t, locale } = useI18n()
const runtime = ref<SchedulingLivenessRuntimeStatus | null>(null)
const checkInterval = ref<string | undefined>()
const loading = ref(true)
const unavailable = ref(false)
const nowMs = ref(Date.now())
let clockTimer: number | undefined
let pollTimer: number | undefined
let requestInFlight = false
let mounted = false
let upstreamRuntimeInitialized = false
let upstreamLastFinishedAt: string | undefined

const busy = computed(() => loading.value || runtime.value?.running === true)

const runtimeCountdown = computed(() =>
  getSchedulingRuntimeCountdown(runtime.value?.next_run_at, nowMs.value, checkInterval.value)
)

const runtimeStateText = computed(() => {
  if (loading.value) {
    return t('admin.accounts.schedulingRules.runtimeLoading')
  }
  if (unavailable.value && !runtime.value) {
    return t('admin.accounts.schedulingRules.runtimeUnavailable')
  }
  if (runtime.value?.running) {
    return t('admin.accounts.schedulingRules.runtimeRunning')
  }
  if (!runtime.value?.enabled) {
    return t('admin.accounts.schedulingRules.runtimeDisabled')
  }
  const nextRunAt = runtime.value.next_run_at
  if (!nextRunAt) {
    return t('admin.accounts.schedulingRules.runtimeWaiting')
  }
  if (!runtimeCountdown.value) return t('admin.accounts.schedulingRules.runtimeUnavailable')
  const key = runtimeCountdown.value.isDue
    ? 'admin.accounts.schedulingRules.runtimeDueCountdown'
    : 'admin.accounts.schedulingRules.runtimeCountdown'
  return t(key, { duration: runtimeCountdown.value.duration })
})

const runtimeProgressPercent = computed(() => {
  if (loading.value || unavailable.value || runtime.value?.running) return null
  return runtimeCountdown.value?.progressPercent ?? null
})

const latestResultText = computed(() => {
  if (loading.value) {
    return t('admin.accounts.schedulingRules.runtimeNoResult')
  }
  if (unavailable.value && !runtime.value) {
    return t('admin.accounts.schedulingRules.runtimeUnavailable')
  }
  const lastRun = runtime.value?.last_run
  if (!lastRun) {
    return t('admin.accounts.schedulingRules.runtimeNoResult')
  }
  if (lastRun.error) {
    return t('admin.accounts.schedulingRules.runtimeLastError', { error: lastRun.error })
  }
  const finishedAt = new Date(lastRun.finished_at)
  const time = Number.isNaN(finishedAt.getTime())
    ? '-'
    : new Intl.DateTimeFormat(locale.value || undefined, {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      }).format(finishedAt)
  return t('admin.accounts.schedulingRules.runtimeLastResult', {
    time,
    succeeded: lastRun.result.succeeded,
    failed: lastRun.result.failed,
    skipped: lastRun.result.skipped ?? 0
  })
})

const toneClass = computed(() => {
  if (unavailable.value && !runtime.value) {
    return 'border-rose-400 text-rose-600 dark:border-rose-500 dark:text-rose-300'
  }
  if (busy.value) {
    return 'border-primary-500 text-primary-600 dark:border-primary-400 dark:text-primary-300'
  }
  if ((runtime.value?.last_run?.result.failed ?? 0) > 0 || runtime.value?.last_run?.error) {
    return 'border-amber-400 text-amber-600 dark:border-amber-500 dark:text-amber-300'
  }
  if (runtime.value?.enabled) {
    return 'border-emerald-400 text-emerald-600 dark:border-emerald-500 dark:text-emerald-300'
  }
  return 'border-gray-300 text-gray-500 dark:border-dark-500 dark:text-gray-400'
})

const refresh = async () => {
  if (requestInFlight) return
  requestInFlight = true
  try {
    const [settings, schedulingRuntime] = await Promise.all([
      adminAPI.superPriority.get(),
      adminAPI.superPriority.getRuntime().catch(() => null)
    ])
    if (!mounted) return
    runtime.value = schedulingRuntime?.liveness ?? settings.liveness_runtime ?? null
    checkInterval.value = settings.check_interval
    if (schedulingRuntime) {
      const lastRun = schedulingRuntime.upstream_billing.last_run
      const finishedAt = lastRun?.finished_at
      if (upstreamRuntimeInitialized && finishedAt && finishedAt !== upstreamLastFinishedAt && (lastRun.result.checked > 0 || lastRun.result.skipped > 0)) {
        emit('upstream-billing-completed')
      }
      upstreamLastFinishedAt = finishedAt
      upstreamRuntimeInitialized = true
    }
    unavailable.value = false
    nowMs.value = Date.now()
  } catch {
    if (mounted && !runtime.value) unavailable.value = true
  } finally {
    if (mounted) loading.value = false
    requestInFlight = false
  }
}

onMounted(() => {
  mounted = true
  void refresh()
  clockTimer = window.setInterval(() => {
    nowMs.value = Date.now()
  }, 1000)
  pollTimer = window.setInterval(() => {
    void refresh()
  }, 5000)
})

onUnmounted(() => {
  mounted = false
  if (clockTimer !== undefined) window.clearInterval(clockTimer)
  if (pollTimer !== undefined) window.clearInterval(pollTimer)
})

defineExpose({ refresh })
</script>
