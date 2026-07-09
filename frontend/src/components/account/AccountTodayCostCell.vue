<template>
  <div>
    <!-- Loading state -->
    <div v-if="props.loading && !props.stats" class="h-4 w-14 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>

    <!-- Error state -->
    <div v-else-if="props.error && !props.stats" class="text-xs text-red-500">{{ props.error }}</div>

    <!-- Cost data -->
    <div v-else-if="props.stats" class="space-y-0.5 text-xs">
      <div class="flex items-center gap-1">
        <span class="text-gray-500 dark:text-gray-400">{{ t('usage.accountBilled') }}:</span>
        <span class="font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(props.stats.cost) }}</span>
      </div>
      <div v-if="props.stats.user_cost != null" class="flex items-center gap-1">
        <span class="text-gray-500 dark:text-gray-400">{{ t('usage.userBilled') }}:</span>
        <span class="font-medium text-gray-700 dark:text-gray-300">{{ formatCurrency(props.stats.user_cost) }}</span>
      </div>
    </div>

    <!-- No data -->
    <div v-else class="text-xs text-gray-400">-</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { WindowStats } from '@/types'
import { formatCurrency } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    stats?: WindowStats | null
    loading?: boolean
    error?: string | null
  }>(),
  {
    stats: null,
    loading: false,
    error: null
  }
)

const { t } = useI18n()
</script>
