<template>
  <BaseDialog :show="show" :title="t('admin.accounts.detection.title')" width="extra-wide" @close="emit('close')">
    <div class="space-y-4">
      <div role="tablist" class="flex flex-wrap gap-2 border-b border-gray-200 pb-3 dark:border-dark-600">
        <button v-for="item in tabs" :key="item" type="button" role="tab" :aria-selected="tab === item" :data-test="`detection-tab-${item}`" class="btn" :class="tab === item ? 'btn-primary' : 'btn-secondary'" @click="tab = item">{{ t(`admin.accounts.detection.tabs.${item}`) }}</button>
      </div>
      <p v-if="loadError" role="alert" class="text-sm text-red-600">{{ loadError }}</p>
      <div v-if="tab === 'manual'" class="space-y-4">
        <div class="flex flex-wrap gap-3">
          <button type="button" class="btn btn-primary" :disabled="loading || selectedIds.length === 0" @click="launchManual">{{ t('admin.accounts.detection.openTest') }}</button>
        </div>
        <p class="text-xs text-gray-500">{{ t('admin.accounts.detection.manualHint') }}</p>
        <DetectionAccountPicker v-model="selectedIds" :accounts="manualAccounts" :disabled="loading" />
      </div>
      <div v-if="tab === 'records'" class="space-y-3">
        <h4 class="text-sm font-medium">{{ t('admin.accounts.detection.currentAccounts') }}</h4>
        <p class="text-xs text-gray-500">{{ t('admin.accounts.detection.currentAccountsHint') }}</p>
        <input v-model="accountSearch" type="search" class="input w-full" :placeholder="t('admin.scheduledTests.searchAccounts')" />
        <div class="max-h-64 overflow-auto">
          <table class="w-full whitespace-nowrap text-left text-xs">
            <thead><tr><th class="p-2">{{ t('admin.accounts.columns.name') }}</th><th class="p-2">{{ t('admin.accounts.columns.downgraded') }}</th><th class="p-2">{{ t('admin.accounts.detection.capabilityState') }}</th></tr></thead>
            <tbody><tr v-for="account in statusAccounts" :key="account.id">
              <td class="p-2">{{ account.name }} #{{ account.id }}</td>
              <td class="p-2">{{ t(!settingsKnown ? 'admin.accounts.detection.qualityUnknown' : !settings.enabled ? 'admin.accounts.detection.qualityDisabled' : accountQualityEnabled(settings, account) ? 'common.enabled' : 'common.disabled') }}</td>
              <td class="p-2" :class="account.extra?.pelican_downgraded === true ? 'text-red-600' : ''">{{ t(account.extra?.pelican_downgraded === true ? 'admin.accounts.pelicanDowngraded' : account.extra?.pelican_downgraded === false ? 'admin.accounts.pelicanNormal' : 'admin.accounts.detection.capability.unknown') }}</td>
            </tr></tbody>
          </table>
        </div>
        <h4 class="text-sm font-medium">{{ t('admin.accounts.detection.history') }}</h4>
        <div class="flex flex-wrap gap-2">
          <Select v-model="recordAccount" :options="accounts.map(a=>({label:`${a.name} #${a.id}`,value:a.id}))" :placeholder="t('admin.accounts.detection.allAccounts')" clearable searchable class="min-w-48" />
          <Select v-model="recordSource" :options="['manual','scheduled','request'].map(source=>({label:t(`admin.accounts.detection.sources.${source}`),value:source}))" :placeholder="t('admin.accounts.detection.allSources')" clearable class="min-w-40" />
          <input v-model="recordModel" class="input" :placeholder="t('admin.scheduledTests.model')" @keyup.enter="loadRecords()" />
          <button type="button" class="btn btn-secondary" :disabled="loadingRecords" @click="refreshStatus">{{ t('common.refresh') }}</button>
        </div>
        <p v-if="recordsError" role="alert" class="text-sm text-red-600">{{ recordsError }}</p>
        <p v-if="records.length === 0 && !loadingRecords && !recordsError" class="text-sm text-gray-500">{{ t('admin.scheduledTests.noResults') }}</p>
        <div v-for="record in records" :key="record.id" class="space-y-2 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="flex flex-wrap justify-between gap-2 text-sm"><span>{{ record.account_name }} #{{ record.account_id }} · {{ t('admin.usage.qualityEvidence.column') }} #{{ record.id }} · {{ t(`admin.accounts.detection.sources.${record.source}`) }}</span><span class="text-xs text-gray-500">{{ formatDateTime(record.finished_at) }}</span></div>
          <DetectionResultSummary :status="record.result.status" :capability="record.mode === 'pelican' || !!record.rule_fingerprint" :downgraded="record.result.downgraded" :reason="record.result.downgrade_reason" :error="record.result.error_message" :model="record.model_id" :latency-ms="record.result.latency_ms" :outcomes="record.result.rule_results" />
          <details v-if="record.result.response_text"><summary class="cursor-pointer text-xs">{{ t('admin.scheduledTests.responseText') }}</summary>
            <iframe v-if="record.mode === 'pelican' && hasPreview(record.result.response_text)" sandbox="allow-scripts" :srcdoc="previewDocument(record.result.response_text)" :title="record.account_name" class="mt-2 h-80 w-full bg-white" />
            <pre class="mt-2 whitespace-pre-wrap break-words text-xs">{{ record.result.response_text }}</pre>
          </details>
        </div>
        <button v-if="records.length && hasMore" type="button" class="btn btn-secondary" :disabled="loadingRecords" @click="loadRecords(true)">{{ t('admin.accounts.detection.moreRecords') }}</button>
      </div>
    </div>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import DetectionAccountPicker from '@/components/admin/account/DetectionAccountPicker.vue'
import DetectionResultSummary from '@/components/admin/account/DetectionResultSummary.vue'
import { defaultRequestQualitySettings, normalizeRequestQualitySettings, loadDetectionAccounts } from '@/utils/accountDetection'
import { formatDateTime } from '@/utils/format'
import { extractPelicanPreviewHtml, pelicanPreviewDocument } from '@/utils/pelicanPreviewDocument'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'
import { accountQualityEnabled } from './settings'
import type { Account, AccountDetectionRecord, DegradationRule } from '@/types'
const props = defineProps<{ show: boolean; initialTab?: 'manual'|'records' }>()
const emit = defineEmits<{ (e:'close'):void; (e:'test',accounts:Account[]):void; (e:'settings-saved'):void }>()
const { t }=useI18n()
const app=useAppStore()
const tabs=['manual','records'] as const
const tab=ref<typeof tabs[number]>('records')
const accountSearch=ref('')
const accounts=ref<Account[]>([]), rules=ref<DegradationRule[]>([])
const selectedIds=ref<number[]>([])
const settings=reactive(defaultRequestQualitySettings())
const loading=ref(false),loadError=ref(''),settingsKnown=ref(false),recordsError=ref('')
const manualAccounts=computed(()=>accounts.value)
const statusAccounts=computed(()=>accounts.value.filter(a=>`${a.name} ${a.id}`.toLowerCase().includes(accountSearch.value.trim().toLowerCase())))
const records=ref<AccountDetectionRecord[]>([]),recordAccount=ref<number|null>(null),recordSource=ref<string|null>(null),recordModel=ref(''),loadingRecords=ref(false),hasMore=ref(false)
let generation=0
let openGeneration=0
const loadRecords=async (more=false)=>{
  const current=++generation
  loadingRecords.value=true;recordsError.value=''
  if(!more)records.value=[]
  try {
    const data=await adminAPI.accounts.listDetectionRecords({account_id:recordAccount.value??undefined,source:recordSource.value??undefined,model_id:recordModel.value.trim()||undefined,before_id:more?records.value[records.value.length-1]?.id:undefined})
    if(current!==generation)return
    records.value=more?[...records.value,...data]:data;hasMore.value=data.length===100
  } catch(error) { if(current===generation){recordsError.value=extractApiErrorMessage(error);app.showError(recordsError.value)} }
  finally { if(current===generation)loadingRecords.value=false }
}
watch([recordAccount,recordSource],()=>{if(props.show&&tab.value==='records')void loadRecords()})
watch(tab,value=>{if(props.show&&value==='records')void loadRecords()})
watch(()=>props.show,async visible=>{
  const current=++openGeneration
  if(!visible){generation++;loadingRecords.value=false;return}
  const initial=props.initialTab??'records'
  const nextTab=initial==='manual'?'manual':'records'
  accountSearch.value='';settingsKnown.value=false;accounts.value=[];rules.value=[]
  if(tab.value===nextTab&&nextTab==='records')void loadRecords()
  tab.value=nextTab;loading.value=true;loadError.value=''
  try {
    const data=await Promise.allSettled([loadDetectionAccounts(),adminAPI.degradationRules.list(),adminAPI.accounts.getRequestQualityProbeSettings()])
    if(current!==openGeneration)return
    if(data[0].status==='fulfilled')accounts.value=data[0].value
    if(data[1].status==='fulfilled')rules.value=data[1].value
    if(data[2].status==='fulfilled'){Object.assign(settings,normalizeRequestQualitySettings(data[2].value));settingsKnown.value=true}
    loadError.value=data.filter(item=>item.status==='rejected').map(item=>String((item as PromiseRejectedResult).reason?.message??(item as PromiseRejectedResult).reason)).join('; ')
  }finally{if(current===openGeneration)loading.value=false}
},{immediate:true})
const refreshStatus=async()=>{
  const current=openGeneration
  try {
    const [pool,config]=await Promise.all([loadDetectionAccounts(),adminAPI.accounts.getRequestQualityProbeSettings()])
    if(current!==openGeneration)return
    accounts.value=pool;Object.assign(settings,normalizeRequestQualitySettings(config));settingsKnown.value=true
    await loadRecords()
  }catch(error){app.showError(extractApiErrorMessage(error))}
}
const launchManual=()=>{
 const selected=manualAccounts.value.filter(a=>selectedIds.value.includes(a.id))
 if (selected.length) {
   emit('test', selected)
 }
}
const hasPreview=(text:string)=>!!extractPelicanPreviewHtml(text)
const previewDocument=(text:string)=>pelicanPreviewDocument(extractPelicanPreviewHtml(text))
</script>
