<script setup lang="ts">
defineProps<{
  colorClass: string
  tooltip?: string
  current: string | number
  max: string | number
  suffix?: string
}>()
</script>

<template>
  <span
    :class="['inline-flex items-center gap-0.5 text-[10px] font-medium leading-tight font-mono', colorClass]"
    :title="tooltip"
  >
    <slot />
    <span>{{ current }}</span>
    <span class="text-gray-400 dark:text-gray-500">/</span>
    <span>{{ max }}</span>
    <span v-if="suffix" class="text-[9px] opacity-60 ml-0.5">{{ suffix }}</span>
  </span>
</template>