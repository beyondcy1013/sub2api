<template>
  <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.mode') }}
            </label>
            <Select v-model="form.mode" :options="testModeOptions" />
            <p
              v-if="form.mode === 'pelican'"
              class="mt-0.5 text-xs text-amber-600 dark:text-amber-400"
            >
              {{ t('admin.scheduledTests.modePelicanHint') }}
            </p>
          </div>
          <div>
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.model') }}
            </label>
            <AccountTestModelPicker v-model="form.model_id" :options="modelOptions" />
          </div>
          <div class="sm:col-span-2">
            <label class="mb-1 flex items-center gap-1 text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.cronExpression') }}
              <HelpTooltip>
                <template #trigger>
                  <span class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-gray-400/70 text-[10px] font-semibold text-gray-400 transition-colors hover:border-primary-500 hover:text-primary-600 dark:border-gray-500 dark:text-gray-500 dark:hover:border-primary-400 dark:hover:text-primary-400">
                    ?
                  </span>
                </template>
                <div class="space-y-1.5">
                  <p class="font-medium">{{ t('admin.scheduledTests.cronTooltipTitle') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipMeaning') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleEvery30Min') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleHourly') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleDaily') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipExampleWeekly') }}</p>
                  <p>{{ t('admin.scheduledTests.cronTooltipRange') }}</p>
                </div>
              </HelpTooltip>
            </label>
            <Input
              v-model="form.cron_expression"
              :placeholder="'*/30 * * * *'"
              :hint="t('admin.scheduledTests.cronHelp')"
            />
            <CronScheduleBuilder v-model="form.cron_expression" />
          </div>
          <div>
            <label class="mb-1 flex items-center gap-1 text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.maxResults') }}
              <HelpTooltip>
                <template #trigger>
                  <span class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-gray-400/70 text-[10px] font-semibold text-gray-400 transition-colors hover:border-primary-500 hover:text-primary-600 dark:border-gray-500 dark:text-gray-500 dark:hover:border-primary-400 dark:hover:text-primary-400">
                    ?
                  </span>
                </template>
                <div class="space-y-1.5">
                  <p class="font-medium">{{ t('admin.scheduledTests.maxResultsTooltipTitle') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipMeaning') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipBody') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipExample') }}</p>
                  <p>{{ t('admin.scheduledTests.maxResultsTooltipRange') }}</p>
                </div>
              </HelpTooltip>
            </label>
            <Input
              v-model="form.max_results"
              type="number"
              placeholder="100"
            />
          </div>
          <div class="flex items-end">
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <Toggle v-model="form.enabled" />
              {{ t('admin.scheduledTests.enabled') }}
            </label>
          </div>
          <div class="flex items-end">
            <div>
              <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <Toggle v-model="form.auto_recover" />
                {{ t('admin.scheduledTests.autoRecover') }}
              </label>
              <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
                {{ t('admin.scheduledTests.autoRecoverHelp') }}
              </p>
            </div>
          </div>
          <div class="flex items-end">
            <div>
              <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <Toggle v-model="form.auto_recover_schedulable" />
                {{ t('admin.scheduledTests.autoRecoverSchedulable') }}
              </label>
              <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
                {{ t('admin.scheduledTests.autoRecoverSchedulableHelp') }}
              </p>
            </div>
          </div>
          <template v-if="form.mode === 'pelican'">
            <DetectionRulePicker v-model="form.rule_ids" :rules="rules" class="sm:col-span-2" />
            <div v-if="form.rule_ids.length > 1">
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.ruleStrategy') }}</label>
              <Select
                v-model="form.rule_strategy"
                :options="[
                  { label: t('admin.scheduledTests.strategyAll'), value: 'all' },
                  { label: t('admin.scheduledTests.strategyRandom'), value: 'random' },
                  { label: t('admin.scheduledTests.strategySequential'), value: 'sequential' }
                ]"
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.downgradeAction') }}</label>
              <Select
                v-model="form.downgrade_action"
                :options="[
                  { label: t('admin.scheduledTests.actionMark'), value: 'mark' },
                  { label: t('admin.scheduledTests.actionPause'), value: 'pause' }
                ]"
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.scheduledTests.downgradeThreshold') }}
              </label>
              <Input v-model="form.downgrade_threshold" type="number" :min="1" />
              <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
                {{ t('admin.scheduledTests.downgradeThresholdHelp') }}
              </p>
            </div>
          </template>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import Toggle from '@/components/common/Toggle.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import CronScheduleBuilder from './CronScheduleBuilder.vue'
import AccountTestModelPicker from './AccountTestModelPicker.vue'
import DetectionRulePicker from './DetectionRulePicker.vue'
import type { DegradationRule, DetectionModelOption, ScheduledTestMode } from '@/types'
export interface ScheduledDetectionForm {
 model_id: string; mode: ScheduledTestMode; cron_expression: string; max_results: string
 enabled: boolean; auto_recover: boolean; auto_recover_schedulable: boolean; auto_pause_on_downgrade: boolean
 rule_ids: number[]; rule_strategy: 'all' | 'random' | 'sequential'; downgrade_action: 'mark' | 'pause'; downgrade_threshold: string
}
const props = defineProps<{
  modelValue: ScheduledDetectionForm
  modelOptions: DetectionModelOption[]
  rules: DegradationRule[]
}>()
// The parent owns the draft; both create and edit render the same fields.
const form = computed(() => props.modelValue)
const { t } = useI18n()
const testModeOptions = computed(() => [
 {label: t('admin.scheduledTests.modeDefault'), value:'default'},
 {label: t('admin.scheduledTests.modePelican'), value:'pelican'}
])
</script>
