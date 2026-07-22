<template>
  <BaseDialog :show="show" :title="t('admin.accounts.schedulingRate.title')" width="normal" @close="$emit('close')">
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.accounts.schedulingRate.account', { name: account?.name || '-' }) }}
      </p>
      <div v-if="conflict" data-testid="scheduling-rate-conflict" class="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
        <p>{{ t('admin.accounts.schedulingRate.conflict', { manual: formatRate(account?.rate_multiplier), upstream: formatRate(upstreamRate) }) }}</p>
        <button
          v-if="upstreamKnown"
          type="button"
          data-testid="scheduling-rate-copy-upstream"
          class="mt-2 text-xs font-medium text-amber-900 underline dark:text-amber-100"
          @click="copyUpstreamToManual"
        >
          {{ t('admin.accounts.schedulingRate.copyUpstream') }}
        </button>
      </div>
      <label class="block rounded border border-gray-200 p-3 dark:border-dark-600">
        <span class="block text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.accounts.schedulingRate.manual') }}</span>
        <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRate.manualHint') }}</span>
        <input v-model.number="manualRate" data-testid="scheduling-rate-manual" type="number" min="0" step="0.001" class="mt-2 w-32 rounded border border-gray-300 px-2 py-1 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
      </label>
      <label class="flex cursor-pointer items-start gap-2 rounded border border-gray-200 p-3 dark:border-dark-600">
        <input v-model="autoOverwrite" data-testid="scheduling-rate-auto-overwrite" type="checkbox" class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
        <span>
          <span class="block text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.accounts.schedulingRate.autoOverwrite') }}</span>
          <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRate.autoOverwriteHint') }}</span>
        </span>
      </label>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
      <button type="button" data-testid="scheduling-rate-save" class="btn btn-primary" :disabled="saving || !Number.isFinite(manualRate) || manualRate < 0" @click="save">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { Account, UpdateSchedulingRateRequest } from '@/types'

const props = withDefaults(defineProps<{
  show: boolean
  account: Account | null
  upstreamRate?: number
  upstreamKnown?: boolean
  conflict?: boolean
  saving?: boolean
}>(), { upstreamKnown: false, conflict: false, saving: false })
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'save', payload: UpdateSchedulingRateRequest): void
}>()
const { t } = useI18n()
const autoOverwrite = ref(true)
const manualRate = ref(1)
watch(() => [props.show, props.account?.id], () => {
  autoOverwrite.value = props.account?.scheduling_rate_sync_mode
    ? props.account.scheduling_rate_sync_mode === 'auto_overwrite'
    : props.account?.scheduling_rate_source !== 'manual'
  manualRate.value = props.account?.rate_multiplier ?? 1
}, { immediate: true })
const formatRate = (value?: number) => typeof value === 'number' && Number.isFinite(value) ? `${Number(value.toPrecision(6))}x` : '?'
const copyUpstreamToManual = () => {
  if (!props.upstreamKnown || typeof props.upstreamRate !== 'number') return
  manualRate.value = props.upstreamRate
}
const save = () => {
  emit('save', {
    sync_mode: autoOverwrite.value ? 'auto_overwrite' : 'manual_lock',
    rate_multiplier: manualRate.value
  })
}
</script>
