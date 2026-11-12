<template>
  <fieldset :disabled="disabled" class="space-y-4 text-sm text-gray-700 disabled:opacity-60 dark:text-gray-300">
    <label class="flex items-start gap-2">
      <input v-model="settings.enabled" data-testid="scheduling-rule-quality-probe-enabled" type="checkbox" />
      <span>{{ t('admin.accounts.schedulingRules.requestQualityProbe') }}<span class="block text-xs text-gray-500">{{ t('admin.accounts.schedulingRules.requestQualityProbeHint') }}</span></span>
    </label>
    <label class="block">{{ t('admin.accounts.detection.statusRecentMinutes') }}
      <input v-model.number="settings.status_recent_minutes" data-testid="quality-status-recent-minutes" type="number" min="1" max="1440" class="input w-full" />
      <span class="block text-xs text-gray-500">{{ t('admin.accounts.detection.statusRecentMinutesHint') }}</span>
    </label>
    <AccountQualityOverrides v-model="settings.account_overrides" :accounts="accounts" />
    <div v-if="settings.enabled" class="space-y-4">
      <label class="block">{{ t('admin.accounts.detection.modelScope') }}
        <Select v-model="settings.model_scope" :options="[{label:t('admin.accounts.detection.selectedModels'),value:'selected'},{label:t('admin.accounts.detection.allModels'),value:'all'}]" />
      </label>
      <label v-if="settings.model_scope !== 'all'" class="block">{{ t('admin.accounts.schedulingRules.requestQualityProbeModels') }}
        <input v-model="modelText" data-testid="scheduling-rule-quality-probe-models" type="text" class="input w-full" placeholder="gpt-6*, claude-3-7*, claude*" />
        <span class="block text-xs text-gray-500">{{ t('admin.accounts.detection.emptyModels') }}</span>
      </label>
      <label class="block">{{ t('admin.accounts.schedulingRules.requestQualityProbeAccountScope') }}
        <Select v-model="settings.account_scope" :options="scopeOptions" />
      </label>
      <DetectionAccountPicker v-if="settings.account_scope === 'selected'" v-model="settings.account_ids" :accounts="accounts" />
      <label class="block">{{ t('admin.accounts.detection.probeQuestion') }}
        <span data-test="automatic-quality-question" class="block py-1">{{ selectedRuleName }}</span>
        <span class="block text-xs text-gray-500">{{ t('admin.accounts.detection.singleProbeHint') }}</span>
      </label>
      <button type="button" class="text-primary-600" @click="emit('rules')">{{ t('admin.accounts.detection.editRules') }}</button>
      <label class="block">{{ t('admin.accounts.detection.timeout') }}
        <input v-model.number="settings.timeout_seconds" type="number" min="5" max="120" class="input w-full" />
      </label>
      <label class="block">{{ t('admin.accounts.detection.reuseSeconds') }}
        <input v-model.number="settings.reuse_seconds" data-testid="quality-reuse-seconds" type="number" min="0" max="3600" step="30" class="input w-full" />
        <span class="block text-xs text-gray-500">{{ t('admin.accounts.detection.reuseSecondsHint') }}</span>
      </label>
      <p class="text-xs text-gray-500">{{ t('admin.accounts.detection.parallelHint') }}</p>
      <p v-if="settings.model_scope !== 'all' && !settings.models.length || settings.account_scope === 'selected' && !settings.account_ids.length && !Object.values(settings.account_overrides ?? {}).some(Boolean)" role="alert" class="text-xs text-amber-600">{{ t('admin.accounts.detection.emptyScope') }}</p>
    </div>
  </fieldset>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountQualityOverrides from '@/modules/qualitycontrol/AccountQualityOverrides.vue'
import Select from '@/components/common/Select.vue'
import DetectionAccountPicker from './DetectionAccountPicker.vue'
import type { Account, DegradationRule, RequestQualityProbeSettings } from '@/types'
const props = defineProps<{ modelValue: RequestQualityProbeSettings; accounts: Account[]; rules: DegradationRule[]; disabled?: boolean }>()
const emit = defineEmits<{ (e: 'rules'): void }>()
const { t } = useI18n()
const settings = computed(() => props.modelValue)
const modelText = computed({get:() => settings.value.models.join(', '),set:(value:string) => { settings.value.models=value.split(/[,，]/).map(v=>v.trim()).filter(Boolean) }})
const selectedRuleName = computed(() => {
  const id = settings.value.rule_ids?.[0]
  if (!id) return t('admin.accounts.accountTesting.noAutoQuestion')
  const rule = props.rules.find(item => item.id === id)
  return rule ? `${rule.name}${rule.enabled ? '' : ` — ${t('admin.accounts.detection.missingRule')}`}` : `#${id} — ${t('admin.accounts.detection.missingRule')}`
})
const scopeOptions = computed(() => [
 {label:t('admin.accounts.schedulingRules.requestQualityProbeScopeTroubled'),value:'error_or_downgraded'},
 {label:t('admin.accounts.schedulingRules.requestQualityProbeScopeAll'),value:'all'},
 {label:t('admin.accounts.schedulingRules.requestQualityProbeScopeSelected'),value:'selected'}
])
</script>
