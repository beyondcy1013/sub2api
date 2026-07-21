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
      <div class="grid gap-2">
        <label class="flex cursor-pointer items-start gap-2 rounded border border-gray-200 p-3 dark:border-dark-600">
          <input v-model="source" data-testid="scheduling-rate-source-manual" type="radio" value="manual" class="mt-0.5" />
          <span class="flex-1">
            <span class="block text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.accounts.schedulingRate.manual') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.schedulingRate.manualHint') }}</span>
            <input v-model.number="manualRate" data-testid="scheduling-rate-manual" type="number" min="0" step="0.001" class="mt-2 w-32 rounded border border-gray-300 px-2 py-1 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white" />
          </span>
        </label>
        <label class="flex cursor-pointer items-start gap-2 rounded border border-gray-200 p-3 dark:border-dark-600">
          <input v-model="source" data-testid="scheduling-rate-source-upstream" type="radio" value="upstream" class="mt-0.5" />
          <span>
            <span class="block text-sm font-medium text-gray-800 dark:text-gray-200">{{ t('admin.accounts.schedulingRate.upstream') }}</span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
              {{ upstreamKnown ? t('admin.accounts.schedulingRate.upstreamHint', { rate: formatRate(upstreamRate) }) : t('admin.accounts.schedulingRate.upstreamUnknown') }}
            </span>
          </span>
        </label>
      </div>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
      <button type="button" data-testid="scheduling-rate-save" class="btn btn-primary" :disabled="saving || (source === 'manual' && (!Number.isFinite(manualRate) || manualRate < 0))" @click="save">
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { Account, SchedulingRateSource, UpdateSchedulingRateRequest } from '@/types'

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
const source = ref<SchedulingRateSource>('manual')
const manualRate = ref(1)
watch(() => [props.show, props.account?.id], () => {
  source.value = props.account?.scheduling_rate_source === 'upstream' ? 'upstream' : 'manual'
  manualRate.value = props.account?.rate_multiplier ?? 1
}, { immediate: true })
const formatRate = (value?: number) => typeof value === 'number' && Number.isFinite(value) ? `${Number(value.toPrecision(6))}x` : '?'
const copyUpstreamToManual = () => {
  if (!props.upstreamKnown || typeof props.upstreamRate !== 'number') return
  source.value = 'manual'
  manualRate.value = props.upstreamRate
}
const save = () => {
  if (source.value === 'upstream') emit('save', { source: 'upstream' })
  else emit('save', { source: 'manual', rate_multiplier: manualRate.value })
}
</script>
