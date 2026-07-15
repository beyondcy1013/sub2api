<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  used: number
  limit: number
  label?: string // text prefix, e.g. "D" / "W"; omitted shows icon
}>()

const { t } = useI18n()

const textClass = computed(() => {
  if (props.used >= props.limit) {
    return 'text-red-600 dark:text-red-400'
  }
  if (props.used >= props.limit * 0.8) {
    return 'text-amber-600 dark:text-amber-400'
  }
  return 'text-emerald-600 dark:text-emerald-400'
})

const tooltip = computed(() => {
  if (props.used >= props.limit) {
    return t('admin.accounts.capacity.quota.exceeded')
  }
  return t('admin.accounts.capacity.quota.normal')
})

const fmt = (v: number) => v.toFixed(2)
</script>

<template>
  <span
    :class="['inline-flex items-center gap-0.5 text-[10px] font-medium leading-tight font-mono', textClass]"
    :title="tooltip"
  >
    <span v-if="label" class="font-semibold opacity-70 mr-0.5">{{ label }}</span>
    <span>${{ fmt(used) }}</span>
    <span class="text-gray-400 dark:text-gray-500">/</span>
    <span>${{ fmt(limit) }}</span>
  </span>
</template>