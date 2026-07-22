<template>
  <div class="inline-flex h-6 min-w-[8rem] items-center gap-1 whitespace-nowrap">
    <span
      data-testid="scheduling-rate-value"
      class="font-mono text-sm font-medium"
      :class="account.scheduling_rate_optimal ? 'text-amber-500 dark:text-amber-300' : 'text-gray-800 dark:text-gray-200'"
      :title="account.scheduling_rate_optimal ? t('admin.accounts.schedulingRate.optimalHint') : undefined"
    >
      {{ displayRate }}
    </span>
    <span data-testid="scheduling-rate-source" class="text-[10px] text-gray-500 dark:text-gray-400">
      {{ sourceLabel }}
    </span>
    <span
      v-if="livenessLabel"
      data-testid="scheduling-liveness-status"
      class="text-[10px]"
      :class="livenessClass"
    >
      {{ livenessLabel }}
    </span>
    <span
      v-if="account.scheduling_rate_optimal"
      data-testid="scheduling-rate-optimal"
      class="text-[10px] font-medium text-amber-500 dark:text-amber-300"
      :title="t('admin.accounts.schedulingRate.optimalHint')"
    >
      {{ t('admin.accounts.schedulingRate.optimal') }}
    </span>
    <button
      type="button"
      data-testid="scheduling-rate-edit"
      class="inline-flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
      :aria-label="t('admin.accounts.schedulingRate.edit')"
      :title="t('admin.accounts.schedulingRate.edit')"
      @click="$emit('manage')"
    >
      <Icon name="edit" size="xs" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Account } from '@/types'

const props = defineProps<{ account: Account }>()
defineEmits<{ (event: 'manage'): void }>()
const { t } = useI18n()
const displayRate = computed(() => {
  const rate = props.account.scheduling_rate_multiplier ?? props.account.rate_multiplier ?? 1
  return `${Number(Number(rate).toPrecision(6))}x`
})
const sourceLabel = computed(() => {
  const auto = props.account.scheduling_rate_sync_mode
    ? props.account.scheduling_rate_sync_mode === 'auto_overwrite'
    : props.account.scheduling_rate_source !== 'manual'
  return auto
    ? t('admin.accounts.schedulingRate.autoOverwrite')
    : t('admin.accounts.schedulingRate.manualLock')
})
const livenessLabel = computed(() => {
  const status = props.account.scheduling_liveness_status
  return status && status !== 'unknown'
    ? t(`admin.accounts.schedulingRate.liveness.${status}`)
    : ''
})
const livenessClass = computed(() => {
  if (props.account.scheduling_liveness_status === 'dead') return 'text-red-600 dark:text-red-400'
  if (props.account.scheduling_liveness_status === 'suspect') return 'text-amber-600 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
})
</script>
