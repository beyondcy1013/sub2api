<template>
  <section class="space-y-4" data-test="automatic-policies">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h3 class="text-sm font-semibold">{{ t('automaticDetection.title') }}</h3>
      <button type="button" class="btn btn-secondary" :disabled="loading || saving" @click="refreshRuntime">{{ t('automaticDetection.refresh') }}</button>
    </div>
    <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    <p class="text-xs text-gray-500">{{ t('automaticDetection.hint') }}</p>
    <p v-if="plansError" role="alert" class="text-sm text-red-600">{{ plansError }}</p>
    <div v-if="state" class="space-y-4">
      <p class="text-xs text-gray-500">{{ t('automaticDetection.tableHint') }}</p>
      <div class="max-w-full overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="w-full min-w-[900px] text-left text-xs" data-test="periodic-task-table">
          <thead class="bg-gray-50 dark:bg-dark-800"><tr>
            <th v-for="column in ['taskType', 'status', 'scope', 'cycle', 'lastExecution', 'nextExecution', 'actions']" :key="column" class="whitespace-nowrap p-2">{{ t(`automaticDetection.${column}`) }}</th>
          </tr></thead>
          <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
            <tr v-for="row in taskRows" :key="row.id" :data-test="`task-row-${row.id}`">
              <td class="p-2"><button type="button" class="text-left text-primary-600 hover:underline" @click="editTask(row.editor, row.id)">{{ row.name }}</button></td>
              <td class="whitespace-nowrap p-2">{{ row.enabled ? t('automaticDetection.enabled') : t('automaticDetection.notEnabled') }}</td>
              <td class="max-w-xs p-2 break-words">{{ row.scope }}</td>
              <td class="whitespace-nowrap p-2">{{ row.cycle }}</td>
              <td class="whitespace-nowrap p-2">{{ row.last ? formatDateTime(row.last) : '—' }}</td>
              <td class="whitespace-nowrap p-2">{{ row.enabled && row.next ? formatDateTime(row.next) : '—' }}</td>
              <td class="p-2"><button type="button" class="btn btn-secondary whitespace-nowrap" :data-test="`edit-task-${row.id}`" @click="editTask(row.editor, row.id)">{{ t('automaticDetection.edit') }}</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-if="editor" class="flex items-center justify-between gap-2"><h3 class="text-sm font-semibold">{{ t(`automaticDetection.${editor}`) }}</h3><button type="button" class="btn btn-secondary" @click="editor = null">{{ t('automaticDetection.closeEditor') }}</button></div>
      <div class="space-y-1 text-xs" aria-live="polite">
        <p>{{ state.runtime.running ? t('automaticDetection.running') : config.policies.some(p => p.enabled) ? t('automaticDetection.idle') : t('automaticDetection.notEnabled') }} · {{ t('automaticDetection.next') }}: {{ state.runtime.next_run_at ? formatDateTime(state.runtime.next_run_at) : '—' }}</p>
        <p v-if="state.runtime.last_run">{{ t('automaticDetection.summary', { ...state.runtime.last_run }) }} · {{ formatDateTime(state.runtime.last_run.finished_at) }}</p>
        <p v-else>{{ t('automaticDetection.none') }}</p>
        <p v-if="state.runtime.last_run?.error" class="text-red-600">{{ state.runtime.last_run.error }}</p>
      </div>
      <fieldset v-for="policy in config.policies" v-show="editor === policy.kind" :key="policy.kind" :data-test="`policy-${policy.kind}`" :disabled="saving" class="min-w-0 space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <legend class="px-1 text-sm font-semibold">{{ t(`automaticDetection.${policy.kind}`) }}</legend>
        <label class="flex items-center gap-2 text-sm"><input v-model="policy.enabled" type="checkbox" data-test="policy-enabled" />{{ t('automaticDetection.enabled') }}</label>
        <p v-if="policy.kind === 'quality'" class="text-xs text-gray-500">{{ t('automaticDetection.standard') }}</p>
        <p v-if="policy.kind === 'quality' && !hasStandard" class="text-xs text-amber-600">{{ t('automaticDetection.noStandard') }}</p>
        <div class="flex flex-wrap items-center gap-3 text-sm">
          <span>{{ t('automaticDetection.scope') }}</span>
          <label v-for="scope in scopes" :key="scope" class="flex items-center gap-1"><input v-model="policy.account_scope" type="radio" :name="`${policy.kind}-scope`" :value="scope" :data-test="`scope-${scope}`" />{{ t(`automaticDetection.${scope}`) }}</label>
        </div>
        <DetectionAccountPicker v-if="policy.account_scope === 'selected'" v-model="policy.account_ids" :accounts="policyAccounts(policy)" :disabled="saving || accountsLoading" />
        <div class="flex flex-wrap gap-3 text-xs">
          <span>{{ t('automaticDetection.platforms') }}</span>
          <label v-for="platform in policy.kind === 'quality' ? ['openai'] : platforms" :key="platform" class="flex items-center gap-1"><input v-model="policy.platforms" type="checkbox" :value="platform" />{{ platform }}</label>
        </div>
        <div class="space-y-2">
          <p class="text-xs font-medium">{{ t('automaticDetection.states') }}</p>
          <div class="flex flex-wrap gap-3 text-xs"><label v-for="status in statuses" :key="status" class="flex items-center gap-1"><input v-model="policy.statuses" type="checkbox" :value="status" :data-test="`state-${status}`" />{{ t(`automaticDetection.${status}`) }}</label></div>
        </div>
        <label class="flex items-start gap-2 text-xs"><input v-model="policy.include_paused" type="checkbox" data-test="include-paused" />{{ t('automaticDetection.includePaused') }}</label>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="space-y-1 text-xs"><span>{{ t('automaticDetection.interval') }}</span><input v-model.number="policy.interval_minutes" type="number" min="1" max="1440" class="input w-full" data-test="interval" /></label>
          <label class="space-y-1 text-xs"><span>{{ t('automaticDetection.model') }}</span><input v-model.trim="policy.model_id" type="text" class="input w-full" :list="`policy-models-${policy.kind}`" /><datalist :id="`policy-models-${policy.kind}`"><option v-for="model in modelOptions" :key="String(model.value)" :value="model.value">{{ model.label }}</option></datalist></label>
        </div>
        <div class="space-y-2 text-xs">
          <label class="flex items-center gap-2"><input v-model="policy.recover" type="checkbox" data-test="recover" />{{ t('automaticDetection.recover') }}</label>
          <label class="flex items-center gap-2"><input v-model="policy.activate" type="checkbox" data-test="activate" />{{ t('automaticDetection.activate') }}</label>
          <label class="flex items-center gap-2"><input v-model="policy.resume" type="checkbox" data-test="resume" />{{ t('automaticDetection.resume') }}</label>
          <template v-if="policy.kind === 'quality'">
            <label class="flex items-center gap-2"><input v-model="policy.pause_on_downgrade" type="checkbox" />{{ t('automaticDetection.pause') }}</label>
            <label v-if="policy.pause_on_downgrade" class="flex items-center gap-2">{{ t('automaticDetection.threshold') }}<input v-model.number="policy.failure_threshold" type="number" min="1" max="10" class="input w-20" /></label>
          </template>
        </div>
      </fieldset>
      <button v-show="editor === 'connection' || editor === 'quality'" type="button" class="btn btn-primary" data-test="save-policies" :disabled="saving || accountsLoading || !valid" @click="save">{{ t('automaticDetection.save') }}</button>
      <section v-show="editor === 'billing'" class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600" data-test="billing-policy">
        <h3 class="text-sm font-semibold">{{ t('automaticDetection.billing') }}</h3>
        <p class="text-xs text-gray-500">{{ t('automaticDetection.billingHint') }}</p>
        <template v-if="billing">
          <p v-if="!billing.account_scope" class="text-xs text-amber-600">{{ t('automaticDetection.legacyBilling') }}</p>
          <fieldset :disabled="savingBilling" class="min-w-0 space-y-3 text-sm">
            <label class="flex items-center gap-2"><input v-model="billing.enabled" type="checkbox" />{{ t('automaticDetection.enabled') }}</label>
            <div class="flex flex-wrap gap-3"><span>{{ t('automaticDetection.scope') }}</span><label v-for="scope in scopes" :key="scope" class="flex items-center gap-1"><input v-model="billingScope" type="radio" name="billing-scope" :value="scope" />{{ t(`automaticDetection.${scope}`) }}</label></div>
            <DetectionAccountPicker v-if="billingScope === 'selected'" v-model="billingIDs" :accounts="accounts.filter(a => a.type === 'apikey')" :disabled="savingBilling || accountsLoading" />
            <label class="flex items-center gap-2 text-xs">{{ t('automaticDetection.interval') }}<input v-model.number="billing.interval_minutes" type="number" min="5" max="1440" class="input w-24" /></label>
          </fieldset>
          <p v-if="billingRuntime" class="text-xs">{{ t('automaticDetection.billingRuntime') }}: {{ billingRuntime.running ? t('automaticDetection.running') : billingRuntime.enabled ? t('automaticDetection.idle') : t('automaticDetection.notEnabled') }} · {{ t('automaticDetection.next') }}: {{ billingRuntime.next_run_at ? formatDateTime(billingRuntime.next_run_at) : '—' }}</p>
          <p v-if="billingRuntime?.last_run" class="text-xs">{{ t('automaticDetection.summary', { ...billingRuntime.last_run.result, recovered: 0 }) }}</p>
          <button type="button" class="btn btn-primary" data-test="save-billing" :disabled="savingBilling || accountsLoading || !validBilling" @click="saveBilling">{{ t('automaticDetection.billingSave') }}</button>
        </template>
      </section>
      <details v-if="state.runtime.results?.length" class="space-y-2">
        <summary class="cursor-pointer text-sm font-medium">{{ t('automaticDetection.results') }}</summary>
        <div class="max-h-72 overflow-auto">
          <table class="w-full text-left text-xs"><thead><tr><th class="p-2">{{ t('automaticDetection.account') }}</th><th class="p-2">{{ t('automaticDetection.status') }}</th><th class="p-2">{{ t('automaticDetection.finished') }}</th></tr></thead><tbody>
            <tr v-for="(result, index) in state.runtime.results" :key="index"><td class="p-2">{{ result.account_name }} #{{ result.account_id }} · {{ t(`automaticDetection.${result.kind}`) }}</td><td class="p-2"><span :class="result.status === 'failed' ? 'text-red-600' : ''">{{ t(`automaticDetection.${result.status}`) }}</span><p class="max-w-sm break-words text-gray-500">{{ result.message }}</p></td><td class="whitespace-nowrap p-2">{{ formatDateTime(result.finished_at) }}</td></tr>
          </tbody></table>
        </div>
      </details>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import superPriorityAPI from '@/api/admin/superPriority'
import type { SchedulingLivenessRuntimeStatus } from '@/api/admin/superPriority'
import type { Account, SelectOption, UpstreamBillingProbeSettings, ScheduledTestPlan } from '@/types'
import DetectionAccountPicker from '@/components/admin/account/DetectionAccountPicker.vue'
import { formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import { automaticDetectionAPI, validDetectionPolicy } from './automaticDetection'
import type { AutomaticDetectionState, DetectionPolicy } from './automaticDetection'
import messages from './automaticMessages'
const props = defineProps<{ accounts: Account[]; accountsLoading?: boolean; hasStandard: boolean; modelOptions?: SelectOption[] }>()
const emit = defineEmits<{ (e: 'saved'): void; (e: 'schedules', planId: number): void }>()
const { t } = useI18n({ useScope: 'local', messages })
const app = useAppStore()
const scopes = ['all', 'selected'] as const
const statuses = ['active', 'inactive', 'error', 'paused', 'rate_limited', 'downgraded']
const platforms = ['openai', 'anthropic', 'gemini', 'antigravity', 'grok']
const state = ref<AutomaticDetectionState | null>(null)
const config = ref<AutomaticDetectionState['config']>({ revision: '', policies: [] })
const billing = ref<UpstreamBillingProbeSettings | null>(null)
const billingScope = ref<'all' | 'selected'>('selected'), billingIDs = ref<number[]>([])
const billingRuntime = ref<SchedulingLivenessRuntimeStatus | null>(null)
const loading = ref(true), saving = ref(false), savingBilling = ref(false), error = ref('')
const editor = ref<'connection' | 'quality' | 'billing' | null>(null)
const plans = ref<ScheduledTestPlan[]>([]), plansError = ref('')
const savedBilling = ref<UpstreamBillingProbeSettings | null>(null)
type Editor = 'connection' | 'quality' | 'billing' | 'scheduled'
interface TaskRow { id: string; editor: Editor; name: string; enabled: boolean; scope: string; cycle: string; last?: string | null; next?: string | null }
function editTask(kind: Editor, id: string) {
  if (kind === 'scheduled') emit('schedules', Number(id.replace('plan-', '')))
  else editor.value = kind
}
const scopeLabel = (scope: string | undefined, ids: number[]) => scope === 'all' ? t('automaticDetection.all') : scope === 'selected' ? `${t('automaticDetection.selected')} (${ids.length})` : t('automaticDetection.legacyScope')
const taskRows = computed(() => {
  const rows: TaskRow[] = (state.value?.config.policies ?? []).map(p => ({
    id: p.kind, editor: p.kind as Editor, name: t(`automaticDetection.${p.kind}`), enabled: p.enabled,
    scope: scopeLabel(p.account_scope, p.account_ids), cycle: t('automaticDetection.everyMinutes', { minutes: p.interval_minutes }),
    last: state.value?.runtime.results?.filter(r => r.kind === p.kind).map(r => r.finished_at).sort().slice(-1)[0],
    next: state.value?.runtime.next_run_at,
  }))
  if (savedBilling.value) rows.push({
    id: 'billing', editor: 'billing', name: t('automaticDetection.billing'), enabled: savedBilling.value.enabled,
    scope: scopeLabel(savedBilling.value.account_scope, savedBilling.value.account_ids ?? []),
    cycle: t('automaticDetection.everyMinutes', { minutes: savedBilling.value.interval_minutes }),
    last: billingRuntime.value?.last_run?.finished_at, next: billingRuntime.value?.next_run_at,
  })
  for (const plan of plans.value) rows.push({
    id: `plan-${plan.id}`, editor: 'scheduled', name: `${t('automaticDetection.scheduled')} #${plan.id} · ${plan.model_id}`,
    enabled: plan.enabled, scope: `${plan.account_name || props.accounts.find(a => a.id === plan.account_id)?.name || ''} #${plan.account_id}`,
    cycle: plan.cron_expression, last: plan.last_run_at ?? undefined, next: plan.next_run_at,
  })
  return rows
})
async function loadPlans() {
  try { plans.value = await adminAPI.scheduledTests.listAll(); plansError.value = '' }
  catch (e) { plansError.value = extractApiErrorMessage(e) }
}
let legacyIDsInitialized = false
watch([() => props.accounts, () => props.accountsLoading], () => {
  if (billing.value && !billing.value.account_scope && !legacyIDsInitialized && !props.accountsLoading) {
    billingIDs.value = props.accounts.filter(a => a.extra?.upstream_billing_probe_enabled === true).map(a => a.id)
    legacyIDsInitialized = true
  }
})
const valid = computed(() => config.value.policies.length === 2 && config.value.policies.every(validDetectionPolicy) && (!config.value.policies.some(p => p.enabled && p.kind === 'quality') || props.hasStandard))
const validBilling = computed(() => !!billing.value && Number.isInteger(billing.value.interval_minutes) && billing.value.interval_minutes >= 5 && billing.value.interval_minutes <= 1440 && (!billing.value.enabled || billingScope.value === 'all' || billingIDs.value.length > 0))
const policyAccounts = (p: DetectionPolicy) => props.accounts.filter(a => p.kind !== 'quality' || a.platform === 'openai')
async function load() {
  loading.value = true; error.value = ''
  try {
    const [policies, rates] = await Promise.all([automaticDetectionAPI.get(), adminAPI.accounts.getUpstreamBillingProbeSettings()])
    state.value = policies; config.value = JSON.parse(JSON.stringify(policies.config)); billing.value = rates; savedBilling.value = { ...rates }
    billingScope.value = rates.account_scope ?? 'selected'
    billingIDs.value = rates.account_ids ?? props.accounts.filter(a => a.extra?.upstream_billing_probe_enabled === true).map(a => a.id)
    legacyIDsInitialized = !props.accountsLoading
    await Promise.all([loadBillingRuntime(), loadPlans()])
  } catch (e) { error.value = extractApiErrorMessage(e) }
  finally { loading.value = false }
}
async function loadBillingRuntime() {
  try { const value = await superPriorityAPI.getRuntime(); billingRuntime.value = value.upstream_billing ?? null } catch { billingRuntime.value = null }
}
async function refreshRuntime() {
  loading.value = true
  try { const [policies, rates] = await Promise.all([automaticDetectionAPI.get(), adminAPI.accounts.getUpstreamBillingProbeSettings()]); state.value = policies; savedBilling.value = rates; await Promise.all([loadBillingRuntime(), loadPlans()]); error.value = '' } catch (e) { error.value = extractApiErrorMessage(e) } finally { loading.value = false }
}
async function save() {
  saving.value = true; error.value = ''
  try { const saved = await automaticDetectionAPI.save(config.value); state.value = saved; config.value = JSON.parse(JSON.stringify(saved.config)); app.showSuccess(t('automaticDetection.saved')); emit('saved') }
  catch (e) { error.value = extractApiErrorMessage(e) } finally { saving.value = false }
}
async function saveBilling() {
  if (!billing.value) return
  savingBilling.value = true; error.value = ''
  try { billing.value = await adminAPI.accounts.updateUpstreamBillingProbeSettings({ ...billing.value, account_scope: billingScope.value, account_ids: [...billingIDs.value] }); savedBilling.value = { ...billing.value }; app.showSuccess(t('automaticDetection.billingSaved')); emit('saved') }
  catch (e) { error.value = extractApiErrorMessage(e) } finally { savingBilling.value = false }
}
defineExpose({ refreshRuntime })
onMounted(load)
</script>
