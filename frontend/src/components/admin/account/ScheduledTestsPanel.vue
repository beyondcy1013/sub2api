<template>
  <component :is="embedded ? 'div' : BaseDialog"
    :show="show"
    :title="t('admin.scheduledTests.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <!-- Add Plan Button -->
      <div class="flex items-center justify-between">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduledTests.title') }}
        </p>
        <div class="flex items-center gap-2">
          <button
            @click="toggleRulesLibrary"
            class="btn btn-secondary flex items-center gap-1.5 text-sm"
          >
            <Icon name="brain" size="sm" :stroke-width="2" />
            {{ t('admin.scheduledTests.ruleLibrary') }}
          </button>
          <button
            v-if="!rulesOnly && !isGlobalMode"
            @click="showAddForm = !showAddForm"
            class="btn btn-primary flex items-center gap-1.5 text-sm"
          >
            <Icon name="plus" size="sm" :stroke-width="2" />
            {{ t('admin.scheduledTests.addPlan') }}
          </button>
          <button
            v-else-if="!rulesOnly"
            @click="showAddForm = !showAddForm"
            class="btn btn-primary flex items-center gap-1.5 text-sm"
          >
            <Icon name="plus" size="sm" :stroke-width="2" />
            {{ t('admin.scheduledTests.batchPlan') }}
          </button>
        </div>
      </div>

      <!-- Rules Library -->
      <div
        v-if="showRulesLibrary"
        class="rounded-xl border border-amber-200 bg-amber-50/50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
      >
        <div class="mb-3 flex items-center justify-between">
          <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.scheduledTests.ruleLibrary') }}
          </div>
          <button
            v-if="!ruleEditing"
            @click="startRuleCreate"
            class="flex items-center gap-1 rounded-lg bg-primary-500 px-2.5 py-1 text-xs font-medium text-white hover:bg-primary-600"
          >
            <Icon name="plus" size="sm" :stroke-width="2" />
            {{ t('admin.scheduledTests.ruleAdd') }}
          </button>
        </div>

        <!-- Rule edit form -->
        <div v-if="ruleEditing" class="mb-3 rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.ruleName') }}</label>
              <Input v-model="ruleForm.name" :placeholder="t('admin.scheduledTests.ruleName')" />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.ruleEvaluator') }}</label>
              <Select v-model="ruleForm.evaluator" :options="evaluatorOptions" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.rulePrompt') }}</label>
              <textarea
                v-model="ruleForm.prompt"
                rows="4"
                class="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-xs dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200"
              ></textarea>
            </div>
            <div v-if="ruleForm.evaluator === 'numeric_answer'">
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.ruleExpectedAnswer') }}</label>
              <Input v-model="ruleForm.expected_answer" type="number" />
            </div>
            <div v-if="ruleForm.evaluator === 'contains_keywords'" class="sm:col-span-2">
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.scheduledTests.ruleKeywords') }}</label>
              <Input v-model="ruleForm.keywords" :placeholder="t('admin.scheduledTests.ruleKeywordsPlaceholder')" />
              <Select
                v-model="ruleForm.keyword_match"
                class="mt-2"
                :options="[
                  { label: t('admin.scheduledTests.ruleKeywordsAll'), value: 'all' },
                  { label: t('admin.scheduledTests.ruleKeywordsAny'), value: 'any' }
                ]"
              />
            </div>
            <div class="flex items-end">
              <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <Toggle v-model="ruleForm.enabled" />
                {{ t('admin.scheduledTests.enabled') }}
              </label>
            </div>
          </div>
          <div class="mt-3 flex justify-end gap-2">
            <button class="rounded-lg bg-gray-100 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300" @click="ruleEditing = false">
              {{ t('common.cancel') }}
            </button>
            <button class="rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-primary-600" :disabled="ruleSaving" @click="handleRuleSave">
              {{ t('common.save') }}
            </button>
          </div>
        </div>

        <!-- Rules list -->
        <div class="space-y-2">
          <div
            v-for="rule in degradationRules"
            :key="rule.id"
            class="flex items-center justify-between rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ rule.name }}</span>
                <span class="rounded-full bg-gray-100 px-1.5 py-0.5 text-[10px] uppercase text-gray-500 dark:bg-dark-700 dark:text-gray-400">{{ rule.evaluator }}</span>
                <span
                  :class="[
                    'text-[10px] font-medium',
                    rule.enabled ? 'text-green-600 dark:text-green-400' : 'text-gray-400'
                  ]"
                >
                  {{ rule.enabled ? t('admin.scheduledTests.ruleEnabled') : t('admin.scheduledTests.ruleDisabled') }}
                </span>
              </div>
              <p class="mt-0.5 truncate text-xs text-gray-400" :title="rule.prompt">{{ rule.prompt }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-1">
              <button class="rounded p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-500 dark:hover:bg-blue-900/20" :title="t('admin.scheduledTests.ruleEdit')" @click="startRuleEdit(rule)">
                <Icon name="edit" size="sm" :stroke-width="2" />
              </button>
              <button class="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20" :title="t('admin.scheduledTests.ruleDelete')" @click="handleRuleDelete(rule)">
                <Icon name="trash" size="sm" :stroke-width="2" />
              </button>
            </div>
          </div>
          <p v-if="!degradationRules.length && !rulesLoading" class="py-2 text-center text-xs text-gray-400">
            {{ t('admin.scheduledTests.ruleEmpty') }}
          </p>
        </div>
      </div>

      <template v-if="!rulesOnly">
      <!-- Add Plan Form -->
      <div
        v-if="showAddForm"
        class="rounded-xl border border-primary-200 bg-primary-50/50 p-4 dark:border-primary-800 dark:bg-primary-900/20"
      >
        <div class="mb-3 text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.scheduledTests.addPlan') }}
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div v-if="isGlobalMode" class="sm:col-span-2">
            <div class="mb-1 flex flex-wrap items-center justify-between gap-2">
              <label class="text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.scheduledTests.selectAccounts') }} ({{ selectedAccountIds.size }})
              </label>
              <div class="flex items-center gap-2">
                <Select
                  v-model="selectedGroupId"
                  :options="groupOptions"
                  :placeholder="t('admin.scheduledTests.allGroups')"
                  class="w-40"
                  clearable
                />
                <button type="button" class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="selectVisibleAccounts">
                  {{ t('admin.scheduledTests.selectAllVisible') }}
                </button>
                <button type="button" class="text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400" @click="clearAccountSelection">
                  {{ t('admin.scheduledTests.clearSelection') }}
                </button>
              </div>
            </div>
            <Input
              v-model="accountSearch"
              type="search"
              :placeholder="t('admin.scheduledTests.searchAccounts')"
              class="mb-2"
            />
            <p
              v-if="newPlan.mode === 'pelican'"
              class="mb-2 text-xs text-amber-600 dark:text-amber-400"
            >
              {{ t('admin.scheduledTests.modePelicanHint') }}
            </p>
            <div class="grid max-h-44 grid-cols-1 gap-1.5 overflow-y-auto rounded-lg border border-gray-200 p-2 sm:grid-cols-2 md:grid-cols-3 dark:border-dark-600">
              <label
                v-for="account in filteredCandidateAccounts"
                :key="account.id"
                class="flex min-w-0 cursor-pointer items-center gap-2 rounded p-1 text-xs hover:bg-gray-50 dark:hover:bg-dark-700/60"
                :title="account.name"
              >
                <input
                  type="checkbox"
                  :checked="selectedAccountIds.has(account.id)"
                  class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
                  @change="toggleAccountSelection(account.id)"
                />
                <span class="truncate font-medium text-gray-800 dark:text-gray-200">{{ account.name }}</span>
                <span class="shrink-0 text-[10px] uppercase text-gray-400">{{ account.type }}</span>
              </label>
            </div>
          </div>
          <ScheduledDetectionPlanForm :model-value="newPlan" :model-options="modeModelOptions" :rules="degradationRules" class="sm:col-span-2" />
        </div>
        <div class="mt-3 flex justify-end gap-2">
          <button
            @click="showAddForm = false; resetNewPlan()"
            class="rounded-lg bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            @click="handleCreate"
            :disabled="createDisabled"
            class="flex items-center gap-1.5 rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Icon v-if="creating" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            {{ t('common.save') }}
          </button>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="md" class="animate-spin text-gray-400" :stroke-width="2" />
        <span class="ml-2 text-sm text-gray-500">{{ t('common.loading') }}...</span>
      </div>

      <!-- Empty State -->
      <div
        v-else-if="plans.length === 0"
        class="rounded-xl border border-dashed border-gray-300 py-10 text-center dark:border-dark-600"
      >
        <Icon name="calendar" size="lg" class="mx-auto mb-2 text-gray-400" :stroke-width="1.5" />
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.scheduledTests.noPlans') }}
        </p>
      </div>

      <!-- Plans List -->
      <div v-else class="space-y-3">
        <div
          v-for="plan in plans"
          :key="plan.id"
          class="rounded-xl border border-gray-200 bg-white transition-all dark:border-dark-600 dark:bg-dark-800"
        >
          <!-- Plan Header -->
          <div
            class="flex cursor-pointer items-center justify-between px-4 py-3"
            @click="toggleExpand(plan.id)"
          >
            <div class="flex flex-1 items-center gap-4">
              <!-- Model -->
              <div class="min-w-0">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">
                  <span v-if="plan.account_name" class="mr-2">{{ plan.account_name }}</span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ plan.model_id }}</span>
                </div>
                <div v-if="isGlobalMode && plan.account_platform" class="mt-0.5 text-xs text-gray-400">
                  {{ plan.account_platform }}
                </div>
                <div class="mt-0.5 font-mono text-xs text-gray-500 dark:text-gray-400">
                  {{ plan.cron_expression }}
                </div>
              </div>

              <!-- Enabled Toggle -->
              <div class="flex items-center gap-1.5" @click.stop>
                <Toggle
                  :model-value="plan.enabled"
                  @update:model-value="(val: boolean) => handleToggleEnabled(plan, val)"
                />
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ plan.enabled ? t('admin.scheduledTests.enabled') : '' }}
                </span>
              </div>

              <!-- Auto Recover Badge -->
              <span
                v-if="plan.auto_recover"
                class="inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400"
              >
                {{ t('admin.scheduledTests.autoRecover') }}
              </span>

              <!-- Pelican Mode Badge -->
              <span
                v-if="plan.mode === 'pelican'"
                class="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/20 dark:text-amber-400"
              >
                {{ t('admin.scheduledTests.modePelican') }}
              </span>
            </div>

            <div class="flex items-center gap-3">
              <!-- Last Run -->
              <div v-if="plan.last_run_at" class="hidden text-right text-xs text-gray-500 dark:text-gray-400 sm:block">
                <div>{{ t('admin.scheduledTests.lastRun') }}</div>
                <div>{{ formatDateTime(plan.last_run_at) }}</div>
              </div>

              <!-- Next Run -->
              <div v-if="plan.next_run_at" class="hidden text-right text-xs text-gray-500 dark:text-gray-400 sm:block">
                <div>{{ t('admin.scheduledTests.nextRun') }}</div>
                <div>{{ formatDateTime(plan.next_run_at) }}</div>
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-1" @click.stop>
                <button
                  @click="startEdit(plan)"
                  class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-blue-50 hover:text-blue-500 dark:hover:bg-blue-900/20"
                  :title="t('admin.scheduledTests.editPlan')"
                >
                  <Icon name="edit" size="sm" :stroke-width="2" />
                </button>
                <button
                  @click="confirmDeletePlan(plan)"
                  class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
                  :title="t('admin.scheduledTests.deletePlan')"
                >
                  <Icon name="trash" size="sm" :stroke-width="2" />
                </button>
              </div>

              <!-- Expand indicator -->
              <Icon
                name="chevronDown"
                size="sm"
                :class="[
                  'text-gray-400 transition-transform duration-200',
                  expandedPlanId === plan.id ? 'rotate-180' : ''
                ]"
              />
            </div>
          </div>

          <!-- Edit Form -->
          <div
            v-if="editingPlanId === plan.id"
            class="border-t border-blue-100 bg-blue-50/50 px-4 py-3 dark:border-blue-900 dark:bg-blue-900/10"
            @click.stop
          >
            <div class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.editPlan') }}
            </div>
            <ScheduledDetectionPlanForm :model-value="editForm" :model-options="editModelOptions" :rules="degradationRules" />
            <div class="mt-3 flex justify-end gap-2">
              <button
                @click="cancelEdit"
                class="rounded-lg bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
              >
                {{ t('common.cancel') }}
              </button>
              <button
                @click="handleEdit"
                :disabled="!editForm.model_id || !editForm.cron_expression || updating"
                class="flex items-center gap-1.5 rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Icon v-if="updating" name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
                {{ t('common.save') }}
              </button>
            </div>
          </div>

          <!-- Expanded Results Section -->
          <div
            v-if="expandedPlanId === plan.id"
            class="border-t border-gray-100 px-4 py-3 dark:border-dark-700"
          >
            <div class="mb-2 text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.scheduledTests.results') }}
            </div>

            <!-- Results Loading -->
            <div v-if="loadingResults" class="flex items-center justify-center py-4">
              <Icon name="refresh" size="sm" class="animate-spin text-gray-400" :stroke-width="2" />
              <span class="ml-2 text-xs text-gray-500">{{ t('common.loading') }}...</span>
            </div>

            <!-- No Results -->
            <div
              v-else-if="results.length === 0"
              class="py-4 text-center text-xs text-gray-500 dark:text-gray-400"
            >
              {{ t('admin.scheduledTests.noResults') }}
            </div>

            <!-- Results List -->
            <div v-else class="max-h-64 space-y-2 overflow-y-auto">
              <div
                v-for="result in results"
                :key="result.id"
                class="rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900"
              >
                <DetectionResultSummary :status="result.status" :capability="plan.mode === 'pelican'" :downgraded="result.downgraded" :reason="result.downgrade_reason" :error="result.error_message" :model="result.response_model" :latency-ms="result.latency_ms" :outcomes="ruleOutcomes(result)" />
                <span class="text-xs text-gray-400">{{ formatDateTime(result.started_at) }}</span>

                <!-- Response / Error (collapsible) -->
                <div v-if="result.error_message" class="mt-2">
                  <div
                    class="cursor-pointer text-xs font-medium text-red-600 dark:text-red-400"
                    @click="toggleResultDetail(result.id)"
                  >
                    {{ t('admin.scheduledTests.errorMessage') }}
                    <Icon
                      name="chevronDown"
                      size="sm"
                      :class="[
                        'inline transition-transform duration-200',
                        expandedResultIds.has(result.id) ? 'rotate-180' : ''
                      ]"
                    />
                  </div>
                  <pre
                    v-if="expandedResultIds.has(result.id)"
                    class="mt-1 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-red-50 p-2 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300"
                  >{{ result.error_message }}</pre>
                </div>
                <div v-else-if="result.response_text" class="mt-2">
                  <div
                    class="cursor-pointer text-xs font-medium text-gray-600 dark:text-gray-400"
                    @click="toggleResultDetail(result.id)"
                  >
                    {{ t('admin.scheduledTests.responseText') }}
                    <Icon
                      name="chevronDown"
                      size="sm"
                      :class="[
                        'inline transition-transform duration-200',
                        expandedResultIds.has(result.id) ? 'rotate-180' : ''
                      ]"
                    />
                  </div>
                  <pre
                    v-if="expandedResultIds.has(result.id)"
                    class="mt-1 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-gray-100 p-2 text-xs text-gray-700 dark:bg-dark-800 dark:text-gray-300"
                  >{{ result.response_text }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      </template>
    </div>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      :show="showDeleteConfirm"
      :title="t('admin.scheduledTests.deletePlan')"
      :message="t('admin.scheduledTests.confirmDelete')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteConfirm = false"
    />
  </component>
</template>

<script setup lang="ts">
import { computed, ref, reactive, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import Toggle from '@/components/common/Toggle.vue'
import ScheduledDetectionPlanForm from './ScheduledDetectionPlanForm.vue'
import DetectionResultSummary from './DetectionResultSummary.vue'
import { loadDetectionAccounts } from '@/utils/accountDetection'
import { Icon } from '@/components/icons'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import type { Account, DegradationRule, DegradationEvaluator, RuleOutcome, ScheduledTestMode, ScheduledTestPlan, ScheduledTestResult } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

// Default model used by the pelican downgrade probe when no model is set
// (mirrors service.DefaultPelicanModel in the backend).
const PELICAN_DEFAULT_MODEL = 'gpt-6-astra'

const props = defineProps<{
  show: boolean
  accountId: number | null
  modelOptions: SelectOption[]
  embedded?: boolean
  rulesOnly?: boolean
  initialPlanId?: number
}>()

const isGlobalMode = computed(() => props.accountId === null)

const emit = defineEmits<{
  (e: 'close'): void
}>()

// State
const loading = ref(false)
const creating = ref(false)
const loadingResults = ref(false)
const plans = ref<ScheduledTestPlan[]>([])
const results = ref<ScheduledTestResult[]>([])
const expandedPlanId = ref<number | null>(null)
let resultsRequestId = 0
watch(expandedPlanId, () => {
  resultsRequestId++
  results.value = []
  loadingResults.value = false
}, { flush: 'sync' })
onBeforeUnmount(() => { resultsRequestId++ })
const expandedResultIds = reactive(new Set<number>())
const showAddForm = ref(false)
const showDeleteConfirm = ref(false)
const deletingPlan = ref<ScheduledTestPlan | null>(null)
const editingPlanId = ref<number | null>(null)
const updating = ref(false)
const editForm = reactive({
  model_id: '' as string,
  cron_expression: '' as string,
  max_results: '100' as string,
  enabled: true,
  auto_recover: false,
  auto_recover_schedulable: false,
  mode: 'default' as ScheduledTestMode,
  auto_pause_on_downgrade: false,
  rule_ids: [] as number[],
  rule_strategy: 'all' as 'all' | 'random' | 'sequential',
  downgrade_action: 'mark' as 'mark' | 'pause',
  downgrade_threshold: '1' as string
})

const newPlan = reactive({
  model_id: '' as string,
  cron_expression: '*/30 * * * *' as string,
  max_results: '100' as string,
  enabled: true,
  auto_recover: false,
  auto_recover_schedulable: false,
  mode: 'default' as ScheduledTestMode,
  auto_pause_on_downgrade: false,
  rule_ids: [] as number[],
  rule_strategy: 'all' as 'all' | 'random' | 'sequential',
  downgrade_action: 'pause' as 'mark' | 'pause',
  downgrade_threshold: '1' as string
})

// --- Degradation rule library ---
const showRulesLibrary = ref(props.rulesOnly === true)
const degradationRules = ref<DegradationRule[]>([])
const rulesLoading = ref(false)
const ruleEditing = ref(false)
const ruleSaving = ref(false)
const ruleForm = reactive({
  id: null as number | null,
  name: '',
  enabled: true,
  evaluator: 'numeric_answer' as DegradationEvaluator,
  prompt: '',
  expected_answer: '21' as string,
  keywords: '' as string,
  keyword_match: 'all' as string
})

const evaluatorOptions = [
  { label: t('admin.scheduledTests.evaluatorNumeric'), value: 'numeric_answer' },
  { label: t('admin.scheduledTests.evaluatorKeywords'), value: 'contains_keywords' },
  { label: t('admin.scheduledTests.evaluatorRefusal'), value: 'refusal_only' },
  { label: t('admin.scheduledTests.evaluatorHTML'), value: 'html_document' }
]

const toggleRulesLibrary = async () => {
  showRulesLibrary.value = !showRulesLibrary.value
  if (showRulesLibrary.value) {
    await loadRules()
  }
}

const loadRules = async () => {
  rulesLoading.value = true
  try {
    degradationRules.value = await adminAPI.degradationRules.list()
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to load rules')
  } finally {
    rulesLoading.value = false
  }
}

const startRuleCreate = () => {
  ruleForm.id = null
  ruleForm.name = ''
  ruleForm.enabled = true
  ruleForm.evaluator = 'numeric_answer'
  ruleForm.prompt = ''
  ruleForm.expected_answer = '21'
  ruleForm.keywords = ''
  ruleForm.keyword_match = 'all'
  ruleEditing.value = true
}

const startRuleEdit = (rule: DegradationRule) => {
  ruleForm.id = rule.id
  ruleForm.name = rule.name
  ruleForm.enabled = rule.enabled
  ruleForm.evaluator = rule.evaluator
  ruleForm.prompt = rule.prompt
  ruleForm.expected_answer = String(rule.params.expected_answer ?? 21)
  ruleForm.keywords = (rule.params.keywords || []).join(', ')
  ruleForm.keyword_match = rule.params.keyword_match || 'all'
  ruleEditing.value = true
}

const handleRuleSave = async () => {
  if (!ruleForm.name.trim() || !ruleForm.prompt.trim() || ruleSaving.value) return
  ruleSaving.value = true
  try {
    const params: import('@/types').DegradationParams = {}
    if (ruleForm.evaluator === 'numeric_answer') {
      params.expected_answer = Number(ruleForm.expected_answer) || 21
    }
    if (ruleForm.evaluator === 'contains_keywords') {
      params.keywords = ruleForm.keywords.split(/[,，]/).map(k => k.trim()).filter(Boolean)
      params.keyword_match = ruleForm.keyword_match
    }
    const payload = {
      name: ruleForm.name.trim(),
      enabled: ruleForm.enabled,
      evaluator: ruleForm.evaluator,
      prompt: ruleForm.prompt.trim(),
      params
    }
    if (ruleForm.id === null) {
      await adminAPI.degradationRules.create(payload)
    } else {
      await adminAPI.degradationRules.update(ruleForm.id, payload)
    }
    appStore.showSuccess(t('admin.scheduledTests.ruleSaveSuccess'))
    ruleEditing.value = false
    await loadRules()
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to save rule')
  } finally {
    ruleSaving.value = false
  }
}

const handleRuleDelete = async (rule: DegradationRule) => {
  if (!window.confirm(t('admin.scheduledTests.ruleDeleteConfirm', { name: rule.name }))) return
  try {
    await adminAPI.degradationRules.delete(rule.id)
    appStore.showSuccess(t('admin.scheduledTests.ruleDeleteSuccess'))
    await loadRules()
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to delete rule')
  }
}

const ruleOutcomes = (result: ScheduledTestResult): RuleOutcome[] => {
  return Array.isArray(result.rule_results) ? result.rule_results : []
}

// Ensure the pelican default model is selectable even when it is not part of
// the account's model catalog.
const withPelicanDefaultModel = (options: SelectOption[]) => {
  if (!options.some(option => option.value === PELICAN_DEFAULT_MODEL)) {
    return [...options, { label: `${PELICAN_DEFAULT_MODEL} (${t('admin.scheduledTests.pelicanDefaultModel')})`, value: PELICAN_DEFAULT_MODEL }]
  }
  return options
}
const modeModelOptions = computed(() =>
  newPlan.mode === 'pelican' ? withPelicanDefaultModel(props.modelOptions) : props.modelOptions
)
const editModelOptions = computed(() =>
  editForm.mode === 'pelican' ? withPelicanDefaultModel(props.modelOptions) : props.modelOptions
)

// Switching to pelican mode defaults the model, the rule selection and the
// handling action; switching back clears them.
watch(
  () => newPlan.mode,
  (mode) => {
    if (mode === 'pelican') {
      if (!newPlan.model_id) newPlan.model_id = PELICAN_DEFAULT_MODEL
      if (newPlan.rule_ids.length === 0 && degradationRules.value.length > 0) {
        newPlan.rule_ids = degradationRules.value.filter(r => r.enabled).map(r => r.id)
      }
      newPlan.downgrade_action = 'pause'
    } else {
      newPlan.rule_ids = []
      newPlan.downgrade_action = 'mark'
    }
  }
)

const candidateAccounts = ref<Account[]>([])
const accountsLoading = ref(false)
const accountsLoaded = ref(false)
const accountSearch = ref('')
const selectedGroupId = ref<number | null>(null)
const selectedAccountIds = ref(new Set<number>())
const groups = ref<import('@/types').AdminGroup[]>([])
const groupOptions = computed(() => [
  ...groups.value.map(group => ({ label: group.name, value: group.id }))
])
const filteredCandidateAccounts = computed(() => {
  const query = accountSearch.value.trim().toLowerCase()
  return candidateAccounts.value.filter(account => {
    if (newPlan.mode === 'pelican' && account.platform !== 'openai') {
      return false
    }
    if (selectedGroupId.value !== null && !(account.group_ids || []).includes(selectedGroupId.value)) {
      return false
    }
    if (!query) return true
    return account.name.toLowerCase().includes(query) || account.type.toLowerCase().includes(query)
  })
})
const createDisabled = computed(() => {
  if (!newPlan.cron_expression || creating.value) return true
  return isGlobalMode.value
    ? selectedAccountIds.value.size === 0
    : !newPlan.model_id
})

const resetNewPlan = () => {
  newPlan.model_id = ''
  newPlan.cron_expression = '*/30 * * * *'
  newPlan.max_results = '100'
  newPlan.enabled = true
  newPlan.auto_recover = false
  newPlan.auto_recover_schedulable = false
  newPlan.mode = 'default'
  newPlan.auto_pause_on_downgrade = false
  newPlan.rule_ids = []
  newPlan.rule_strategy = 'all'
  newPlan.downgrade_action = 'mark'
  newPlan.downgrade_threshold = '1'
}


const loadPlans = async () => {
  loading.value = true
  try {
    plans.value = isGlobalMode.value
      ? await adminAPI.scheduledTests.listAll()
      : await adminAPI.scheduledTests.listByAccount(props.accountId!)
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to load plans')
  } finally {
    loading.value = false
  }
}

const loadGlobalFormData = async () => {
  if (accountsLoaded.value) return
  accountsLoading.value = true
  try {
    const [accountPage, groupList] = await Promise.all([
      loadDetectionAccounts(),
      adminAPI.groups.getAll()
    ])
    candidateAccounts.value = accountPage
    groups.value = groupList
    accountsLoaded.value = true
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to load accounts')
  } finally {
    accountsLoading.value = false
  }
}

const toggleAccountSelection = (accountID: number) => {
  if (selectedAccountIds.value.has(accountID)) {
    selectedAccountIds.value.delete(accountID)
  } else {
    selectedAccountIds.value.add(accountID)
  }
  selectedAccountIds.value = new Set(selectedAccountIds.value)
}

const selectVisibleAccounts = () => {
  for (const account of filteredCandidateAccounts.value) {
    selectedAccountIds.value.add(account.id)
  }
  selectedAccountIds.value = new Set(selectedAccountIds.value)
}

const clearAccountSelection = () => {
  selectedAccountIds.value = new Set()
}

const handleCreate = async () => {
  if (!newPlan.cron_expression || creating.value) return
  if (isGlobalMode.value && selectedAccountIds.value.size === 0) return
  if (!isGlobalMode.value && (!props.accountId || !newPlan.model_id)) return
  creating.value = true
  try {
    const maxResults = Number(newPlan.max_results) || 100
  if (isGlobalMode.value) {
      const result = await adminAPI.scheduledTests.batchCreate({
        account_ids: Array.from(selectedAccountIds.value),
        model_id: newPlan.model_id,
        cron_expression: newPlan.cron_expression,
        enabled: newPlan.enabled,
        max_results: maxResults,
        auto_recover: newPlan.auto_recover,
        auto_recover_schedulable: newPlan.auto_recover_schedulable,
        mode: newPlan.mode,
        auto_pause_on_downgrade: newPlan.auto_pause_on_downgrade,
        rule_ids: newPlan.rule_ids,
        rule_strategy: newPlan.rule_strategy,
        downgrade_action: newPlan.downgrade_action,
        downgrade_threshold: Number(newPlan.downgrade_threshold) || 1
      })
      if (result.failed > 0) {
        appStore.showError(t('admin.scheduledTests.batchPartial', { created: result.created, failed: result.failed }))
      } else {
        appStore.showSuccess(t('admin.scheduledTests.batchSuccess', { count: result.created }))
      }
    } else {
      await adminAPI.scheduledTests.create({
        account_id: props.accountId!,
        model_id: newPlan.model_id,
        cron_expression: newPlan.cron_expression,
        enabled: newPlan.enabled,
        max_results: maxResults,
        auto_recover: newPlan.auto_recover,
        auto_recover_schedulable: newPlan.auto_recover_schedulable,
        mode: newPlan.mode,
        auto_pause_on_downgrade: newPlan.auto_pause_on_downgrade,
        rule_ids: newPlan.rule_ids,
        rule_strategy: newPlan.rule_strategy,
        downgrade_action: newPlan.downgrade_action,
        downgrade_threshold: Number(newPlan.downgrade_threshold) || 1
      })
      appStore.showSuccess(t('admin.scheduledTests.createSuccess'))
    }
    showAddForm.value = false
    resetNewPlan()
    selectedAccountIds.value = new Set()
    await loadPlans()
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to create plan')
  } finally {
    creating.value = false
  }
}

const handleToggleEnabled = async (plan: ScheduledTestPlan, enabled: boolean) => {
  try {
    const updated = await adminAPI.scheduledTests.update(plan.id, { enabled })
    const index = plans.value.findIndex((p) => p.id === plan.id)
    if (index !== -1) {
      plans.value[index] = updated
    }
    appStore.showSuccess(t('admin.scheduledTests.updateSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to update plan')
  }
}

const startEdit = (plan: ScheduledTestPlan) => {
  editingPlanId.value = plan.id
  editForm.model_id = plan.model_id
  editForm.cron_expression = plan.cron_expression
  editForm.max_results = String(plan.max_results)
  editForm.enabled = plan.enabled
  editForm.auto_recover = plan.auto_recover
  editForm.auto_recover_schedulable = plan.auto_recover_schedulable
  editForm.mode = plan.mode === 'pelican' ? 'pelican' : 'default'
  editForm.auto_pause_on_downgrade = plan.auto_pause_on_downgrade
  editForm.rule_ids = [...(plan.rule_ids || [])]
  editForm.rule_strategy = plan.rule_strategy || 'all'
  editForm.downgrade_action = plan.downgrade_action || (plan.auto_pause_on_downgrade ? 'pause' : 'mark')
  editForm.downgrade_threshold = String(plan.downgrade_threshold || 1)
}

const cancelEdit = () => {
  editingPlanId.value = null
}

const handleEdit = async () => {
  if (!editingPlanId.value || !editForm.model_id || !editForm.cron_expression) return
  updating.value = true
  try {
    const updated = await adminAPI.scheduledTests.update(editingPlanId.value, {
      model_id: editForm.model_id,
      cron_expression: editForm.cron_expression,
      max_results: Number(editForm.max_results) || 100,
      enabled: editForm.enabled,
      auto_recover: editForm.auto_recover,
      auto_recover_schedulable: editForm.auto_recover_schedulable,
      mode: editForm.mode,
      auto_pause_on_downgrade: editForm.auto_pause_on_downgrade,
      rule_ids: editForm.rule_ids,
      rule_strategy: editForm.rule_strategy,
      downgrade_action: editForm.downgrade_action,
      downgrade_threshold: Number(editForm.downgrade_threshold) || 1
    })
    const index = plans.value.findIndex((p) => p.id === editingPlanId.value)
    if (index !== -1) {
      plans.value[index] = updated
    }
    appStore.showSuccess(t('admin.scheduledTests.updateSuccess'))
    editingPlanId.value = null
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to update plan')
  } finally {
    updating.value = false
  }
}

const confirmDeletePlan = (plan: ScheduledTestPlan) => {
  deletingPlan.value = plan
  showDeleteConfirm.value = true
}

const handleDelete = async () => {
  if (!deletingPlan.value) return
  try {
    await adminAPI.scheduledTests.delete(deletingPlan.value.id)
    appStore.showSuccess(t('admin.scheduledTests.deleteSuccess'))
    plans.value = plans.value.filter((p) => p.id !== deletingPlan.value!.id)
    if (expandedPlanId.value === deletingPlan.value.id) {
      expandedPlanId.value = null
      results.value = []
    }
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to delete plan')
  } finally {
    showDeleteConfirm.value = false
    deletingPlan.value = null
  }
}

const toggleExpand = async (planId: number) => {
  if (expandedPlanId.value === planId) {
    expandedPlanId.value = null
    results.value = []
    expandedResultIds.clear()
    return
  }

  expandedPlanId.value = planId
  expandedResultIds.clear()
  loadingResults.value = true
  const requestId = resultsRequestId
  try {
    const data = await adminAPI.scheduledTests.listResults(planId, 20)
    if (requestId !== resultsRequestId) return
    results.value = data
  } catch (error: any) {
    if (requestId !== resultsRequestId) return
    appStore.showError(error?.message || 'Failed to load results')
    results.value = []
  } finally {
    if (requestId === resultsRequestId) loadingResults.value = false
  }
}

const toggleResultDetail = (resultId: number) => {
  if (expandedResultIds.has(resultId)) {
    expandedResultIds.delete(resultId)
  } else {
    expandedResultIds.add(resultId)
  }
}
// Load plans when dialog opens
watch(
  () => props.show,
  async (visible) => {
    if (!visible) {
      plans.value = []
      results.value = []
      expandedPlanId.value = null
      expandedResultIds.clear()
      showAddForm.value = false
      showDeleteConfirm.value = false
      accountSearch.value = ''
      selectedGroupId.value = null
      selectedAccountIds.value = new Set()
      return
    }
    if (visible && isGlobalMode.value) {
      await Promise.all([...(props.rulesOnly ? [] : [loadPlans()]), loadRules()])
      if (!props.rulesOnly) await loadGlobalFormData()
    } else if (visible && props.accountId) {
      await Promise.all([...(props.rulesOnly ? [] : [loadPlans()]), loadRules()])
    }
    if (!props.rulesOnly) {
      const initialPlan = plans.value.find(plan => plan.id === props.initialPlanId)
      if (initialPlan) startEdit(initialPlan)
    }
  },
  { immediate: true }
)

</script>
