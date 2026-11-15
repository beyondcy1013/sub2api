<template>
  <BaseDialog :show="show" :title="t('admin.accounts.schedulingRules.title')" width="normal" @close="requestClose">
    <template #title-actions>
      <HelpTooltip placement="bottom" width-class="w-56 max-w-[calc(100vw-1rem)] sm:w-96">
        <template #trigger>
          <span
            data-testid="scheduling-rules-help"
            class="inline-flex cursor-help items-center text-xs font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
          >
            {{ t('admin.accounts.schedulingRules.help') }}
          </span>
        </template>
        <div class="space-y-2 text-left">
          <p class="font-semibold">{{ t('admin.accounts.schedulingRules.helpTitle') }}</p>
          <p>{{ t('admin.accounts.schedulingRules.helpEligibility') }}</p>
          <p>{{ t('admin.accounts.schedulingRules.helpDefault') }}</p>
          <p>{{ t('admin.accounts.schedulingRules.helpLowestCost') }}</p>
          <p>{{ t('admin.accounts.schedulingRules.helpLiveness') }}</p>
        </div>
      </HelpTooltip>
    </template>

    <template #header-actions>
      <div data-testid="scheduling-rules-runtime" class="min-w-0 max-w-32 text-right text-[10px] leading-4 sm:max-w-52 sm:text-[11px]">
        <div class="truncate font-medium text-gray-700 dark:text-gray-200">{{ runtimeStateText }}</div>
        <div class="truncate text-gray-500 dark:text-gray-400" :title="latestResultText">{{ latestResultText }}</div>
      </div>
    </template>

    <div v-if="loading" class="flex justify-center py-10">
      <Icon name="refresh" size="md" class="animate-spin text-gray-400" />
    </div>
    <div v-else class="space-y-5">
      <div data-testid="scheduling-rules-current-mode" class="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900/40">
        <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.currentMode') }}</span>
        <span data-testid="scheduling-rules-effective-mode" class="text-xs font-semibold text-gray-800 dark:text-gray-100">{{ strategyLabel(serverStrategy) }}</span>
        <span
          data-testid="scheduling-rule-mode-badge"
          class="rounded-full px-2 py-0.5 text-[10px] font-medium"
          :class="dirty ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-200' : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200'"
        >{{ dirty ? t('admin.accounts.schedulingRules.modeUnsaved') : t('admin.accounts.schedulingRules.modeActive') }}</span>
      </div>

      <div data-testid="scheduling-rule-strategy-group" class="space-y-3" role="radiogroup" :aria-label="t('admin.accounts.schedulingRules.strategy')">
        <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
          {{ t('admin.accounts.schedulingRules.strategy') }}
        </div>
        <div class="grid gap-2 sm:grid-cols-2">
          <button
            v-for="option in strategyOptions"
            :key="option.value"
            type="button"
            role="radio"
            :aria-checked="strategy === option.value"
            :data-testid="option.testId"
            class="rounded-lg border p-3 text-left transition"
            :class="strategy === option.value ? 'border-primary-600 bg-primary-50/70 ring-1 ring-primary-500/30 dark:border-primary-400 dark:bg-primary-900/30' : 'border-gray-300 hover:border-gray-400 dark:border-dark-600 dark:hover:border-dark-400'"
            @click="strategy = option.value"
          >
            <div class="flex items-start gap-2">
              <span
                class="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border"
                :class="strategy === option.value ? 'border-primary-600 bg-primary-600 dark:border-primary-400 dark:bg-primary-400' : 'border-gray-300 dark:border-dark-500'"
              >
                <span v-if="strategy === option.value" class="h-1.5 w-1.5 rounded-full bg-white dark:bg-dark-900" />
              </span>
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-1.5">
                  <span class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ option.label }}</span>
                  <span v-if="option.value === serverStrategy" class="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-medium text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200">{{ t('admin.accounts.schedulingRules.badgeEffective') }}</span>
                  <span v-if="option.value === strategy && strategy !== serverStrategy" class="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-200">{{ t('admin.accounts.schedulingRules.badgePending') }}</span>
                </div>
                <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">{{ option.description }}</p>
                <p class="mt-0.5 text-[11px] text-gray-400 dark:text-gray-500">{{ option.fit }}</p>
              </div>
            </div>
          </button>
        </div>
      </div>

      <div class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
          {{ t('admin.accounts.schedulingRules.liveness') }}
        </div>
        <label class="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="livenessEnabled" data-testid="scheduling-rule-liveness-enabled" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <span>
            <span class="block font-medium">{{ t('admin.accounts.schedulingRules.livenessEnabled') }}</span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.livenessEnabledHint') }}</span>
          </span>
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-gray-700 dark:text-gray-300">
          <span>{{ t('admin.accounts.schedulingRules.livenessInterval') }}</span>
          <input v-model.number="livenessIntervalMinutes" data-testid="scheduling-rule-liveness-interval" type="number" min="1" max="1440" step="1" class="h-9 w-24 rounded border border-gray-300 px-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-gray-700 dark:text-gray-300">
          <span>{{ t('admin.accounts.schedulingRules.livenessThreshold') }}</span>
          <input v-model.number="livenessFailureThreshold" data-testid="scheduling-rule-liveness-threshold" type="number" min="1" max="10" step="1" class="h-9 w-24 rounded border border-gray-300 px-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
        </label>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.livenessHint') }}</p>
      </div>

      <div data-testid="scheduling-rules-tasks" class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="flex items-center justify-between gap-3">
          <div class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.accounts.schedulingRules.tasksAndLogs') }}</div>
          <button type="button" class="btn btn-secondary h-7 px-2 text-xs" :disabled="runtimeLoading" @click="loadRuntime">
            <Icon name="refresh" size="xs" :class="{ 'animate-spin': runtimeLoading }" />
            {{ t('common.refresh') }}
          </button>
        </div>
        <div class="grid gap-2 sm:grid-cols-2">
          <div v-for="task in runtimeTasks" :key="task.key" class="border border-gray-200 p-2 text-xs dark:border-dark-600">
            <div class="flex items-center justify-between gap-2 font-medium text-gray-700 dark:text-gray-200">
              <span>{{ task.label }}</span><span :class="task.status.enabled ? 'text-emerald-600' : 'text-gray-400'">{{ task.status.enabled ? t('admin.accounts.schedulingRules.taskEnabled') : t('admin.accounts.schedulingRules.taskDisabled') }}</span>
            </div>
            <div class="mt-1 text-gray-500 dark:text-gray-400">{{ task.status.running ? t('admin.accounts.schedulingRules.taskRunning') : formatNextRun(task.status.next_run_at) }}</div>
            <div class="mt-1 text-gray-600 dark:text-gray-300">{{ formatLastRun(task.status.last_run) }}</div>
          </div>
        </div>
        <div v-if="runtimeHistory.length" data-testid="scheduling-rules-history" class="max-h-40 space-y-1 overflow-y-auto text-xs">
          <div v-for="(entry, index) in runtimeHistory" :key="`${entry.started_at}-${index}`" class="flex items-center justify-between gap-2 border-b border-gray-100 py-1 text-gray-600 dark:border-dark-700 dark:text-gray-300">
            <span>{{ entry.trigger }} · {{ formatHistoryTime(entry.finished_at) }}</span>
            <span>{{ entry.result.succeeded }}/{{ entry.result.failed }}/{{ entry.result.skipped || 0 }}</span>
          </div>
        </div>
        <p v-else class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.noHistory') }}</p>
      </div>

      <div class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-gray-200">
          <input v-model="probeEnabled" data-testid="scheduling-rule-probe-enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          {{ t('admin.accounts.schedulingRules.upstreamProbe') }}
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-gray-700 dark:text-gray-300">
          <span>{{ t('admin.accounts.schedulingRules.interval') }}</span>
          <input v-model.number="intervalMinutes" data-testid="scheduling-rule-interval" type="number" min="5" max="1440" step="1" :disabled="!probeEnabled" class="h-9 w-24 rounded border border-gray-300 px-2 text-sm disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
        </label>
        <label class="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="notifyOnChangeOnly" data-testid="scheduling-rule-notify-on-change-only" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <span>
            <span class="block font-medium">{{ t('admin.accounts.schedulingRules.notifyOnChangeOnly') }}</span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.notifyOnChangeOnlyHint') }}</span>
          </span>
        </label>
      </div>

      <div class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="modelMismatchOnError" data-testid="scheduling-rule-model-mismatch-on-error" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <span>
            <span class="block font-medium">{{ t('admin.accounts.schedulingRules.modelMismatchOnError') }}</span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.modelMismatchOnErrorHint') }}</span>
          </span>
        </label>
      </div>

      <div class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <button type="button" data-testid="scheduling-rule-quality-settings" class="btn btn-secondary" @click="emit('detection')">{{ t('admin.accounts.detection.qualitySettings') }}</button>
      </div>
    </div>

    <template #footer>
      <template v-if="pendingClose">
        <span class="mr-auto text-xs font-medium text-amber-600 dark:text-amber-400" data-testid="scheduling-rule-unsaved-prompt">{{ t('admin.accounts.schedulingRules.unsavedPrompt') }}</span>
        <button type="button" data-testid="scheduling-rule-keep-editing" class="btn btn-secondary" @click="pendingClose = false">{{ t('admin.accounts.schedulingRules.keepEditing') }}</button>
        <button type="button" data-testid="scheduling-rule-discard-close" class="btn btn-secondary text-red-600 dark:text-red-400" @click="discardAndClose">{{ t('admin.accounts.schedulingRules.discardAndClose') }}</button>
      </template>
      <template v-else>
        <button type="button" data-testid="scheduling-rule-refresh" class="btn btn-secondary mr-auto gap-2" :disabled="loading || saving || refreshing" @click="refreshNow">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': refreshing }" />
          {{ refreshing ? t('admin.accounts.schedulingRules.refreshing') : t('admin.accounts.schedulingRules.immediateRefresh') }}
        </button>
        <button type="button" data-testid="scheduling-rule-cancel" class="btn btn-secondary" :disabled="saving || refreshing" @click="requestClose">{{ t('common.cancel') }}</button>
        <button type="button" data-testid="scheduling-rule-save" class="btn btn-primary" :disabled="saving || refreshing || !validInterval || !dirty" @click="save">
          {{ saving ? t('common.saving') : t('admin.accounts.schedulingRules.saveAndSwitch', { mode: strategyLabel(strategy) }) }}
        </button>
      </template>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import type { SchedulingLivenessRuntimeStatus, SchedulingRulesRefreshResult, SchedulingRulesRuntimeStatus, SuperPrioritySettings } from '@/api/admin/superPriority'
import { getSchedulingRuntimeCountdown } from '@/utils/schedulingRuntimeCountdown'

type SchedulingStrategy = 'default' | 'lowest_cost'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'saved', strategy: SchedulingStrategy): void
  (event: 'detection'): void
  (event: 'refreshed', result: SchedulingRulesRefreshResult): void
  (event: 'refresh-error', error: unknown): void
  (event: 'error', error: unknown): void
}>()
const { t, locale } = useI18n()
const loading = ref(false)
const saving = ref(false)
const refreshing = ref(false)
const strategy = ref<SchedulingStrategy>('default')
const probeEnabled = ref(false)
const intervalMinutes = ref(30)
const notifyOnChangeOnly = ref(false)
const livenessIntervalMinutes = ref(1)
const livenessFailureThreshold = ref(2)
const livenessEnabled = ref(true)
const modelMismatchOnError = ref(false)
const currentSettings = ref<SuperPrioritySettings | null>(null)
const savedProbe = ref({ enabled: false, intervalMinutes: 30, notifyOnChangeOnly: false })
const pendingClose = ref(false)

const serverStrategy = computed<SchedulingStrategy>(() =>
  currentSettings.value?.base_strategy === 'lowest_cost' ? 'lowest_cost' : 'default'
)
const strategyLabel = (value: SchedulingStrategy) =>
  t(value === 'lowest_cost' ? 'admin.accounts.schedulingRules.lowestCost' : 'admin.accounts.schedulingRules.default')
const strategyOptions = computed(() => [
  {
    value: 'default' as SchedulingStrategy,
    testId: 'scheduling-rule-default',
    label: t('admin.accounts.schedulingRules.default'),
    description: t('admin.accounts.schedulingRules.defaultDesc'),
    fit: t('admin.accounts.schedulingRules.defaultFit')
  },
  {
    value: 'lowest_cost' as SchedulingStrategy,
    testId: 'scheduling-rule-lowest-cost',
    label: t('admin.accounts.schedulingRules.lowestCost'),
    description: t('admin.accounts.schedulingRules.lowestCostDesc'),
    fit: t('admin.accounts.schedulingRules.lowestCostFit')
  }
])
const livenessRuntime = ref<SchedulingLivenessRuntimeStatus | null>(null)
const schedulingRuntime = ref<SchedulingRulesRuntimeStatus | null>(null)
const runtimeLoading = ref(false)
const nowMs = ref(Date.now())
let runtimeTimer: number | undefined
let runtimeTicks = 0
const validInterval = computed(() =>
  (!probeEnabled.value || (Number.isInteger(intervalMinutes.value) && intervalMinutes.value >= 5 && intervalMinutes.value <= 1440)) &&
  Number.isInteger(livenessIntervalMinutes.value) && livenessIntervalMinutes.value >= 1 && livenessIntervalMinutes.value <= 1440 &&
  Number.isInteger(livenessFailureThreshold.value) && livenessFailureThreshold.value >= 1 && livenessFailureThreshold.value <= 10
)

// The dialog must show whether the saved mode is the one the server reports,
// so every field is compared against the last loaded/verified snapshot.
const dirty = computed(() => {
  const settings = currentSettings.value
  if (!settings) return false
  return (
    strategy.value !== serverStrategy.value ||
    livenessEnabled.value !== (settings.liveness_enabled !== false) ||
    livenessIntervalMinutes.value !== intervalFromExpression(settings.check_interval ?? '@every 1m') ||
    livenessFailureThreshold.value !== (settings.failure_threshold ?? 2) ||
    modelMismatchOnError.value !== (settings.model_mismatch_on_error === true) ||
    probeEnabled.value !== savedProbe.value.enabled ||
    intervalMinutes.value !== savedProbe.value.intervalMinutes ||
    notifyOnChangeOnly.value !== savedProbe.value.notifyOnChangeOnly
  )
})

const requestClose = () => {
  if (dirty.value) {
    pendingClose.value = true
    return
  }
  emit('close')
}

const discardAndClose = () => {
  pendingClose.value = false
  emit('close')
}

const runtimeCountdown = computed(() =>
  getSchedulingRuntimeCountdown(
    livenessRuntime.value?.next_run_at,
    nowMs.value,
    currentSettings.value?.check_interval ?? `@every ${livenessIntervalMinutes.value}m`
  )
)

const intervalFromExpression = (expression: string): number => {
  const match = /^@every\s+(\d+)m$/i.exec(expression.trim())
  return match ? Math.max(1, Math.min(1440, Number(match[1]))) : 1
}

const runtimeStateText = computed(() => {
  if (refreshing.value || livenessRuntime.value?.running) {
    return t('admin.accounts.schedulingRules.runtimeRunning')
  }
  if (!livenessRuntime.value?.enabled) {
    return t('admin.accounts.schedulingRules.runtimeDisabled')
  }
  const nextRunAt = livenessRuntime.value.next_run_at
  if (!nextRunAt) {
    return t('admin.accounts.schedulingRules.runtimeWaiting')
  }
  if (!runtimeCountdown.value) return t('admin.accounts.schedulingRules.runtimeUnavailable')
  return t(
    runtimeCountdown.value.isDue
      ? 'admin.accounts.schedulingRules.runtimeDueCountdown'
      : 'admin.accounts.schedulingRules.runtimeCountdown',
    { duration: runtimeCountdown.value.duration }
  )
})

const latestResultText = computed(() => {
  const lastRun = livenessRuntime.value?.last_run
  if (!lastRun) {
    return t('admin.accounts.schedulingRules.runtimeNoResult')
  }
  if (lastRun.error) {
    return t('admin.accounts.schedulingRules.runtimeLastError', { error: lastRun.error })
  }
  const finishedAt = new Date(lastRun.finished_at)
  const time = Number.isNaN(finishedAt.getTime())
    ? '-'
    : new Intl.DateTimeFormat(locale?.value || undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(finishedAt)
  return t('admin.accounts.schedulingRules.runtimeLastResult', {
    time,
    succeeded: lastRun.result.succeeded,
    failed: lastRun.result.failed,
    skipped: lastRun.result.skipped ?? 0
  })
})

const pollRuntimeStatus = async () => {
  try {
    const settings = await adminAPI.superPriority.get()
    livenessRuntime.value = settings.liveness_runtime ?? null
    await loadRuntime()
  } catch {
    // Keep the last known status; the primary load/save paths still surface errors.
  }
}

const loadRuntime = async () => {
  const getter = adminAPI.superPriority.getRuntime
  if (!getter) return
  runtimeLoading.value = true
  try { schedulingRuntime.value = await getter() } catch { /* keep last snapshot */ } finally { runtimeLoading.value = false }
}

const runtimeTasks = computed(() => [
  { key: 'liveness', label: t('admin.accounts.schedulingRules.liveness'), status: schedulingRuntime.value?.liveness ?? livenessRuntime.value ?? { enabled: false, running: false } },
  { key: 'upstream', label: t('admin.accounts.schedulingRules.upstreamProbe'), status: schedulingRuntime.value?.upstream_billing ?? { enabled: false, running: false } }
])
const runtimeHistory = computed(() => [
  ...(schedulingRuntime.value?.liveness?.history ?? []),
  ...(schedulingRuntime.value?.upstream_billing?.history ?? [])
].sort((a, b) => Date.parse(b.finished_at) - Date.parse(a.finished_at)).slice(0, 20))
const formatNextRun = (value?: string) => value ? `${t('admin.accounts.schedulingRules.nextRun')} ${new Date(value).toLocaleString(locale?.value)}` : t('admin.accounts.schedulingRules.taskWaiting')
const formatHistoryTime = (value: string) => new Date(value).toLocaleString(locale?.value, { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
const formatLastRun = (entry?: { finished_at: string; result: { succeeded: number; failed: number; skipped?: number }; error?: string }) => entry ? (entry.error || `${entry.result.succeeded}/${entry.result.failed}/${entry.result.skipped ?? 0}`) : t('admin.accounts.schedulingRules.runtimeNoResult')

const stopRuntimeTimer = () => {
  if (runtimeTimer !== undefined) {
    window.clearInterval(runtimeTimer)
    runtimeTimer = undefined
  }
}

const startRuntimeTimer = () => {
  stopRuntimeTimer()
  runtimeTicks = 0
  runtimeTimer = window.setInterval(() => {
    nowMs.value = Date.now()
    runtimeTicks += 1
    if (runtimeTicks % 5 === 0) void pollRuntimeStatus()
  }, 1000)
}

const load = async () => {
  loading.value = true
  try {
    const [settings, probe] = await Promise.all([
      adminAPI.superPriority.get(),
      adminAPI.accounts.getUpstreamBillingProbeSettings()
    ])
    currentSettings.value = settings
    strategy.value = settings.base_strategy
    probeEnabled.value = probe.enabled
    intervalMinutes.value = probe.interval_minutes
    notifyOnChangeOnly.value = probe.notify_on_change_only === true
    savedProbe.value = {
      enabled: probe.enabled,
      intervalMinutes: probe.interval_minutes,
      notifyOnChangeOnly: probe.notify_on_change_only === true
    }
    livenessIntervalMinutes.value = intervalFromExpression(settings.check_interval)
    livenessFailureThreshold.value = settings.failure_threshold
    livenessEnabled.value = settings.liveness_enabled !== false
    modelMismatchOnError.value = settings.model_mismatch_on_error === true
    livenessRuntime.value = settings.liveness_runtime ?? null
    await loadRuntime()
    nowMs.value = Date.now()
  } catch (error) {
    emit('error', error)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      void load()
      startRuntimeTimer()
    } else {
      stopRuntimeTimer()
    }
  },
  { immediate: true }
)

onUnmounted(stopRuntimeTimer)

const save = async () => {
  if (saving.value || !validInterval.value) return
  saving.value = true
  try {
    const current = currentSettings.value ?? await adminAPI.superPriority.get()
    // Retire any old overlay before saving the single supported scheduling rule.
    await adminAPI.superPriority.deactivate()
    await adminAPI.superPriority.update({
      base_strategy: strategy.value,
      failure_threshold: livenessFailureThreshold.value,
      check_interval: `@every ${livenessIntervalMinutes.value}m`,
      liveness_enabled: livenessEnabled.value,
      liveness_include_unschedulable: false,
      test_model_id: current.test_model_id,
      test_prompt: current.test_prompt,
      model_mismatch_on_error: modelMismatchOnError.value
    })
    await adminAPI.accounts.updateUpstreamBillingProbeSettings({
      ...await adminAPI.accounts.getUpstreamBillingProbeSettings(),
      enabled: probeEnabled.value,
      interval_minutes: intervalMinutes.value,
      notify_on_change_only: notifyOnChangeOnly.value
    })
    savedProbe.value = {
      enabled: probeEnabled.value,
      intervalMinutes: intervalMinutes.value,
      notifyOnChangeOnly: notifyOnChangeOnly.value
    }
    // Read the settings back so "switched" reflects the server, not the form.
    const fresh = await adminAPI.superPriority.get()
    currentSettings.value = fresh
    livenessRuntime.value = fresh.liveness_runtime ?? livenessRuntime.value
    if (fresh.base_strategy !== strategy.value) {
      strategy.value = fresh.base_strategy
      pendingClose.value = false
      emit('error', new Error(t('admin.accounts.schedulingRules.switchMismatch', { mode: strategyLabel(fresh.base_strategy) })))
      return
    }
    pendingClose.value = false
    emit('saved', strategy.value)
  } catch (error) {
    emit('error', error)
  } finally {
    saving.value = false
  }
}

const refreshNow = async () => {
  if (loading.value || saving.value || refreshing.value) return
  refreshing.value = true
  try {
    const result = await adminAPI.superPriority.refresh()
    emit('refreshed', result)
    await pollRuntimeStatus()
  } catch (error) {
    emit('refresh-error', error)
  } finally {
    refreshing.value = false
  }
}
</script>
