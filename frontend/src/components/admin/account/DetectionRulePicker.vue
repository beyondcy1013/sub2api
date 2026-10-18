<template>
  <div class="space-y-2">
    <div class="flex flex-wrap items-center justify-between gap-2 text-xs">
      <span class="font-medium">{{ t('admin.scheduledTests.ruleSelection') }} ({{ model.length }})</span>
      <div class="flex gap-3">
        <button type="button" :disabled="disabled" class="text-primary-600" @click="model = rules.filter(r => r.enabled).map(r => r.id)">{{ t('admin.scheduledTests.ruleSelectAll') }}</button>
        <button type="button" :disabled="disabled" @click="model = []">{{ t('admin.scheduledTests.ruleSelectNone') }}</button>
      </div>
    </div>
    <div class="grid max-h-48 gap-2 overflow-y-auto rounded-lg border border-gray-200 p-2 sm:grid-cols-2 dark:border-dark-600">
      <label v-for="rule in rules" :key="rule.id" class="flex items-start gap-2 text-xs" :class="{ 'opacity-50': !rule.enabled }">
        <input type="checkbox" :checked="model.includes(rule.id)" :disabled="disabled || !rule.enabled" @change="toggle(rule.id)" />
        <span class="min-w-0 break-words">{{ rule.name }} <span v-if="!rule.enabled">({{ t('admin.scheduledTests.ruleDisabled') }})</span></span>
      </label>
      <span v-for="id in missingIds" :key="id" class="text-xs text-amber-600">#{{ id }} — {{ t('admin.accounts.detection.missingRule') }} <button type="button" :disabled="disabled" @click="toggle(id)">×</button></span>
    </div>
    <p v-if="model.length === 0" class="text-xs text-gray-500">{{ emptyHint || t('admin.scheduledTests.ruleSelectionEmpty') }}</p>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DegradationRule } from '@/types'
const model = defineModel<number[]>({ required: true })
const props = defineProps<{ rules: DegradationRule[]; disabled?: boolean; emptyHint?: string }>()
const { t } = useI18n()
const missingIds = computed(() => model.value.filter(id => !props.rules.some(r => r.id === id)))
const toggle = (id: number) => { model.value = model.value.includes(id) ? model.value.filter(v => v !== id) : [...model.value, id] }
</script>
