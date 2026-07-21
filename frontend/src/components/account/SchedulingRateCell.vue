<template>
  <div class="inline-flex h-6 min-w-[8rem] items-center gap-1 whitespace-nowrap">
    <span data-testid="scheduling-rate-value" class="font-mono text-sm font-medium text-gray-800 dark:text-gray-200">
      {{ displayRate }}
    </span>
    <span data-testid="scheduling-rate-source" class="text-[10px] text-gray-500 dark:text-gray-400">
      {{ sourceLabel }}
    </span>
    <span v-if="!known" class="text-[10px] text-amber-600 dark:text-amber-400">{{ t('admin.accounts.schedulingRate.unknown') }}</span>
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
const known = computed(() => props.account.scheduling_rate_known !== false && typeof props.account.scheduling_rate_multiplier === 'number')
const displayRate = computed(() => known.value ? `${Number(Number(props.account.scheduling_rate_multiplier).toPrecision(6))}x` : '?')
const sourceLabel = computed(() => props.account.scheduling_rate_source === 'upstream'
  ? t('admin.accounts.schedulingRate.upstream')
  : t('admin.accounts.schedulingRate.manual'))
</script>
