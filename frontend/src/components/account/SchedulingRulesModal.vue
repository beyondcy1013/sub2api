<template>
  <BaseDialog :show="show" :title="t('admin.accounts.schedulingRules.title')" width="normal" @close="$emit('close')">
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
      <div data-testid="scheduling-rule-strategy-group" class="space-y-3" role="group" :aria-label="t('admin.accounts.schedulingRules.strategy')">
        <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
          {{ t('admin.accounts.schedulingRules.strategy') }}
        </div>
        <div class="grid grid-cols-2 gap-2">
          <button
            type="button"
            data-testid="scheduling-rule-default"
            :aria-pressed="strategy === 'default'"
            class="h-10 border px-3 text-sm font-medium"
            :class="strategy === 'default' ? 'border-primary-600 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-300 text-gray-700 dark:border-dark-600 dark:text-gray-200'"
            @click="strategy = 'default'"
          >
            {{ t('admin.accounts.schedulingRules.default') }}
          </button>
          <button
            type="button"
            data-testid="scheduling-rule-lowest-cost"
            :aria-pressed="strategy === 'lowest_cost'"
            class="h-10 border px-3 text-sm font-medium"
            :class="strategy === 'lowest_cost' ? 'border-primary-600 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-300 text-gray-700 dark:border-dark-600 dark:text-gray-200'"
            @click="strategy = 'lowest_cost'"
          >
            {{ t('admin.accounts.schedulingRules.lowestCost') }}
          </button>
        </div>
      </div>

      <div class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
          {{ t('admin.accounts.schedulingRules.liveness') }}
        </div>
        <label class="flex items-center justify-between gap-3 text-sm text-gray-700 dark:text-gray-300">
          <span>{{ t('admin.accounts.schedulingRules.livenessInterval') }}</span>
          <input v-model.number="livenessIntervalMinutes" data-testid="scheduling-rule-liveness-interval" type="number" min="1" max="1440" step="1" class="h-9 w-24 rounded border border-gray-300 px-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
        </label>
        <label class="flex items-center justify-between gap-3 text-sm text-gray-700 dark:text-gray-300">
          <span>{{ t('admin.accounts.schedulingRules.livenessThreshold') }}</span>
          <input v-model.number="livenessFailureThreshold" data-testid="scheduling-rule-liveness-threshold" type="number" min="1" max="10" step="1" class="h-9 w-24 rounded border border-gray-300 px-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
        </label>
        <label class="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="livenessIncludeUnschedulable" data-testid="scheduling-rule-liveness-include-unschedulable" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <span>
            <span class="block font-medium">{{ t('admin.accounts.schedulingRules.livenessIncludeUnschedulable') }}</span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.livenessIncludeUnschedulableHint') }}</span>
          </span>
        </label>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRules.livenessHint') }}</p>
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
    </div>

    <template #footer>
      <button type="button" data-testid="scheduling-rule-refresh" class="btn btn-secondary mr-auto gap-2" :disabled="loading || saving || refreshing" @click="refreshNow">
        <Icon name="refresh" size="sm" :class="{ 'animate-spin': refreshing }" />
        {{ refreshing ? t('admin.accounts.schedulingRules.refreshing') : t('admin.accounts.schedulingRules.immediateRefresh') }}
      </button>
      <button type="button" class="btn btn-secondary" :disabled="saving || refreshing" @click="$emit('close')">{{ t('common.cancel') }}</button>
      <button type="button" data-testid="scheduling-rule-save" class="btn btn-primary" :disabled="saving || refreshing || !validInterval" @click="save">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
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
import type { SchedulingLivenessRuntimeStatus, SchedulingRulesRefreshResult, SuperPrioritySettings } from '@/api/admin/superPriority'
import { getSchedulingRuntimeCountdown } from '@/utils/schedulingRuntimeCountdown'

type SchedulingStrategy = 'default' | 'lowest_cost'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'saved'): void
  (event: 'refreshed', result: SchedulingRulesRefreshResult): void
  (event: 'refresh-error', error: unknown): void
  (event: 'error', error: unknown): void
}>()
const { t, locale } = useI18n()
const loading = ref(false)
const saving = ref(false)
const refreshing = ref(false)
const strategy = ref<SchedulingStrategy>('default')
const probeEnabled = ref(true)
const intervalMinutes = ref(30)
const notifyOnChangeOnly = ref(false)
const livenessIntervalMinutes = ref(1)
const livenessFailureThreshold = ref(2)
const livenessIncludeUnschedulable = ref(false)
const currentSettings = ref<SuperPrioritySettings | null>(null)
const livenessRuntime = ref<SchedulingLivenessRuntimeStatus | null>(null)
const nowMs = ref(Date.now())
let runtimeTimer: number | undefined
let runtimeTicks = 0
const validInterval = computed(() =>
  (!probeEnabled.value || (Number.isInteger(intervalMinutes.value) && intervalMinutes.value >= 5 && intervalMinutes.value <= 1440)) &&
  Number.isInteger(livenessIntervalMinutes.value) && livenessIntervalMinutes.value >= 1 && livenessIntervalMinutes.value <= 1440 &&
  Number.isInteger(livenessFailureThreshold.value) && livenessFailureThreshold.value >= 1 && livenessFailureThreshold.value <= 10
)

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
  } catch {
    // Keep the last known status; the primary load/save paths still surface errors.
  }
}

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
    livenessIntervalMinutes.value = intervalFromExpression(settings.check_interval)
    livenessFailureThreshold.value = settings.failure_threshold
    livenessIncludeUnschedulable.value = settings.liveness_include_unschedulable === true
    livenessRuntime.value = settings.liveness_runtime ?? null
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
      liveness_include_unschedulable: livenessIncludeUnschedulable.value,
      test_model_id: current.test_model_id,
      test_prompt: current.test_prompt
    })
    await adminAPI.accounts.updateUpstreamBillingProbeSettings({
      enabled: probeEnabled.value,
      interval_minutes: intervalMinutes.value,
      notify_on_change_only: notifyOnChangeOnly.value
    })
    emit('saved')
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
