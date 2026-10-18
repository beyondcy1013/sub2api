<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateTime } from '@/utils/format'
import type { AccountQualitySummary } from '@/types'
import { qualityVerdict } from '@/modules/qualityevidence/verdict'
const props = defineProps<{ summary?: AccountQualitySummary; minutes: number; now: number; state: 'loading' | 'ready' | 'error' }>()
const { t } = useI18n()
const recent = computed(() => {
  const timestamp = Date.parse(props.summary?.finished_at ?? '')
  return Number.isFinite(timestamp) && timestamp <= props.now && props.now - timestamp <= props.minutes * 60000
})
const resultKey = computed(() => {
  const s = props.summary
  if (!s) return 'noRecentTest'
  return { passed: 'recentPassed', downgraded: 'recentDowngraded', cancelled: 'testCancelled', skipped: 'testSkipped', failed: 'testFailed' }[qualityVerdict(s)]
})
const key = computed(() => props.state === 'loading' ? 'statusLoading' : props.state === 'error' ? 'statusError' : !recent.value ? 'noRecentTest' : resultKey.value)
const label = computed(() => t(`admin.accounts.detection.${key.value}`, { minutes: props.minutes }))
const tooltip = computed(() => {
  const lines = [label.value, t('admin.accounts.detection.recentStatusHint', { minutes: props.minutes })]
  const s = props.summary
  if (!s || props.state !== 'ready') return lines.join('\n')
  lines.push(`#${s.id} · ${formatDateTime(s.finished_at)} · ${s.model_id} · ${t(`admin.accounts.detection.sources.${s.source}`)}`)
  lines.push(t('admin.accounts.detection.lastTestResult') + ': ' + t(`admin.accounts.detection.${resultKey.value}`))
  if (s.questions.length) for (const question of s.questions) lines.push(`${question.name}: ${question.prompt || t('admin.accounts.detection.promptNotSaved')}\n${question.passed ? '✓' : '✗'} ${question.reason}`)
  else lines.push(t('admin.accounts.detection.legacyQuestion'))
  if (s.reason) lines.push(s.reason)
  if (s.response_summary) lines.push(t('admin.accounts.detection.responseSummary') + ': ' + s.response_summary)
  return lines.join('\n')
})
</script>

<template>
  <span tabindex="0" :title="tooltip" :aria-label="tooltip" :class="state === 'ready' && recent && summary?.status === 'success' ? summary.downgraded ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400' : 'text-gray-500'">{{ label }}</span>
</template>
