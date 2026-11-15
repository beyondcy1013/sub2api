<template>
  <BaseDialog :show="show" :title="t('automaticDetection.title')" width="extra-wide" @close="emit('close')">
    <div role="tablist" class="mb-4 flex flex-wrap gap-2">
      <button v-for="item in modes" :key="item" type="button" role="tab" :data-test="`automatic-tab-${item}`" :aria-selected="mode === item" class="btn" :class="mode === item ? 'btn-primary' : 'btn-secondary'" @click="mode = item">{{ item === 'policies' ? t('automaticDetection.policyTab') : t(`admin.accounts.detection.tabs.${item}`) }}</button>
    </div>
    <p v-if="error" role="alert" class="mb-3 text-sm text-red-600">{{ error }}</p>
    <AutomaticDetectionPolicies
      ref="policiesPanel"
      v-show="mode === 'policies'"
      :accounts="pool"
      :accounts-loading="loading || !!error"
      :has-standard="standardReady"
      :model-options="modelOptions"
      @saved="emit('saved')"
      @schedules="editSchedule"
    />
    <div v-show="mode === 'request'" class="space-y-4">
      <RequestQualityProbeForm :model-value="settings" :accounts="pool" :rules="rules" :disabled="loading || saving || !!error" @rules="mode = 'rules'" />
      <button type="button" class="btn btn-primary" :disabled="loading || saving || !!error || !validRequestQualitySettings(settings)" @click="saveSettings">{{ t('common.save') }}</button>
    </div>
    <ScheduledTestsPanel
      v-if="mode === 'scheduled' || mode === 'rules'"
      :key="`${mode}-${editingScheduleId ?? 'all'}`"
      :show="show"
      :account-id="mode === 'scheduled' && !editingScheduleId ? initialAccountId ?? null : null"
      :initial-plan-id="editingScheduleId"
      :model-options="modelOptions ?? []"
      embedded
      :rules-only="mode === 'rules'"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { reactive, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { Account, SelectOption } from '@/types'
import AutomaticDetectionPolicies from './AutomaticDetectionPolicies.vue'
import RequestQualityProbeForm from '@/components/admin/account/RequestQualityProbeForm.vue'
import { defaultRequestQualitySettings, normalizeRequestQualitySettings, validRequestQualitySettings, loadDetectionAccounts } from '@/utils/accountDetection'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { DegradationRule } from '@/types'
import ScheduledTestsPanel from '@/components/admin/account/ScheduledTestsPanel.vue'

const emit = defineEmits<{ (event: 'close'): void; (event: 'saved'): void }>()
const { t } = useI18n()
const props = defineProps<{
  show: boolean
  accounts: Account[]
  accountsLoading?: boolean
  hasStandard?: boolean
  modelOptions?: SelectOption[]
  initialMode?: 'policies' | 'request' | 'scheduled' | 'rules'
  initialAccountId?: number | null
}>()
const modes = ['policies', 'request', 'scheduled', 'rules'] as const
const mode = ref<typeof modes[number]>('policies')
const policiesPanel = ref<InstanceType<typeof AutomaticDetectionPolicies> | null>(null)
const editingScheduleId = ref<number | undefined>()
function editSchedule(id: number) { editingScheduleId.value = id; mode.value = 'scheduled' }
const pool = ref<Account[]>([])
const rules = ref<DegradationRule[]>([])
const settings = reactive(defaultRequestQualitySettings())
const standardReady = ref(false)
const loading = ref(false), saving = ref(false), error = ref('')
let generation = 0
async function refreshRules() {
  rules.value = await adminAPI.degradationRules.list()
  const updated = await adminAPI.accounts.getRequestQualityProbeSettings()
  standardReady.value = Boolean(updated.rule_ids?.length)
}
watch(mode, async (value, previous) => {
  if (value === 'policies' && previous === 'scheduled') await policiesPanel.value?.refreshRuntime?.()
  if (previous === 'rules' && props.show) {
    try { await refreshRules() } catch (e) { error.value = extractApiErrorMessage(e) }
  }
})
watch(() => props.show, async visible => {
  const current = ++generation
  if (!visible) return
  editingScheduleId.value = undefined
  mode.value = props.initialMode ?? 'policies'
  loading.value = true; error.value = ''; pool.value = []; standardReady.value = false
  try {
    const [accounts, config, library] = await Promise.all([loadDetectionAccounts(), adminAPI.accounts.getRequestQualityProbeSettings(), adminAPI.degradationRules.list()])
    if (current !== generation) return
    pool.value = accounts; rules.value = library
    Object.assign(settings, normalizeRequestQualitySettings(config))
    standardReady.value = Boolean(config.rule_ids?.length)
  } catch (e) { if (current === generation) error.value = extractApiErrorMessage(e) }
  finally { if (current === generation) loading.value = false }
}, { immediate: true })
async function saveSettings() {
  saving.value = true; error.value = ''
  try {
    Object.assign(settings, await adminAPI.accounts.updateRequestQualityProbeSettings({ ...settings }))
    emit('saved')
  } catch (e) { error.value = extractApiErrorMessage(e) }
  finally { saving.value = false }
}
</script>
