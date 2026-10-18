<template>
  <div class="space-y-1 text-xs" data-test="detection-result-summary">
    <div class="flex flex-wrap items-center gap-3">
      <span :class="status === 'success' ? 'text-green-600 dark:text-green-400' : 'text-gray-600 dark:text-gray-300'">{{ t(`admin.accounts.detection.connection.${connectionState}`) }}</span>
      <span v-if="capability" :class="status !== 'success' ? 'text-gray-500' : downgraded ? 'text-amber-600 dark:text-amber-400' : 'text-green-600 dark:text-green-400'">{{ t(`admin.accounts.detection.capability.${capabilityState}`) }}</span>
      <span v-if="model" class="break-all">{{ model }}</span>
      <span v-if="latencyMs != null">{{ latencyMs }} ms</span>
    </div>
    <p v-if="error" class="whitespace-pre-wrap break-words text-red-600 dark:text-red-400">{{ error }}</p>
    <p v-if="reason" class="whitespace-pre-wrap break-words">{{ reason }}</p>
    <div v-for="outcome in outcomes" :key="outcome.rule_id" class="whitespace-pre-wrap break-words">
      {{ outcome.rule_name }}: {{ outcome.passed ? '✓' : '✗' }} {{ outcome.reason }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RuleOutcome } from '@/types'
const props = defineProps<{ status: string; capability?: boolean; downgraded?: boolean; reason?: string; error?: string; model?: string; latencyMs?: number; outcomes?: RuleOutcome[] | null }>()
const { t } = useI18n()
const connectionState = computed(() => props.status === 'success' ? 'success' : props.status === 'running' || props.status === 'connecting' ? 'running' : props.status === 'cancelled' ? 'cancelled' : props.status === 'skipped' ? 'skipped' : 'failed')
const capabilityState = computed(() => props.status !== 'success' ? 'unknown' : props.downgraded ? 'failed' : 'passed')
</script>
