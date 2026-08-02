<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <header>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accountPolicy.title') }}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accountPolicy.description') }}
        </p>
      </header>

      <div class="grid items-start gap-5 lg:grid-cols-[minmax(260px,320px)_minmax(0,1fr)]">
        <aside class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <form class="border-b border-gray-200 p-3 dark:border-dark-700" @submit.prevent="applySearch">
            <div class="flex gap-2">
              <div class="relative min-w-0 flex-1">
                <Icon
                  name="search"
                  size="sm"
                  class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
                />
                <input
                  v-model="searchInput"
                  class="input w-full pl-9"
                  :placeholder="t('admin.accountPolicy.searchPlaceholder')"
                  data-testid="account-policy-search"
                />
              </div>
              <button class="btn btn-secondary px-3" type="submit" :aria-label="t('common.search')">
                <Icon name="search" size="sm" />
              </button>
            </div>
            <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accountPolicy.accountCount', { count: pagination.total }) }}
            </p>
          </form>

          <div v-if="accountsLoading" class="flex h-48 items-center justify-center">
            <div class="h-6 w-6 animate-spin rounded-full border-2 border-gray-200 border-t-primary-600"></div>
          </div>
          <div v-else-if="accounts.length === 0" class="flex h-48 items-center justify-center px-4 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accountPolicy.empty') }}
          </div>
          <div v-else class="max-h-[calc(100vh-19rem)] min-h-48 overflow-y-auto">
            <button
              v-for="account in accounts"
              :key="account.id"
              type="button"
              class="block w-full border-b border-gray-100 px-4 py-3 text-left last:border-b-0 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary-500 dark:border-dark-700 dark:hover:bg-dark-700"
              :class="selectedAccount?.id === account.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
              :data-testid="`account-policy-account-${account.id}`"
              @click="selectAccount(account)"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="min-w-0 truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ account.name }}
                </span>
                <span class="flex-shrink-0 text-xs" :class="statusTextClass(account.status)">
                  {{ account.status }}
                </span>
              </div>
              <div class="mt-1 flex items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-400">
                <span class="truncate">{{ account.platform }} / {{ account.type }}</span>
                <span class="flex-shrink-0">
                  {{ account.schedulable ? t('admin.accountPolicy.state.enabled') : t('admin.accountPolicy.state.paused') }}
                </span>
              </div>
            </button>
          </div>

          <Pagination
            v-if="pagination.total > pagination.pageSize"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.pageSize"
            :show-page-size-selector="false"
            :show-jump="false"
            @update:page="changePage"
          />
        </aside>

        <main class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div v-if="!selectedAccount" class="flex min-h-96 items-center justify-center p-8 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accountPolicy.selectAccount') }}
          </div>
          <div v-else-if="policyLoading" class="flex min-h-96 items-center justify-center">
            <div class="h-7 w-7 animate-spin rounded-full border-2 border-gray-200 border-t-primary-600"></div>
          </div>
          <form v-else-if="policy" data-testid="account-policy-form" @submit.prevent="savePolicy">
            <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <div class="min-w-0">
                <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                  {{ selectedAccount.name }}
                </h2>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  #{{ selectedAccount.id }} · {{ selectedAccount.platform }} / {{ selectedAccount.type }}
                </p>
              </div>
              <button
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="policyLoading"
                :aria-label="t('common.refresh')"
                @click="loadPolicy(selectedAccount.id)"
              >
                <Icon name="refresh" size="sm" />
                <span>{{ t('common.refresh') }}</span>
              </button>
            </div>

            <section class="border-b border-gray-200 p-5 dark:border-dark-700">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.accountPolicy.state.title') }}
                </h3>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  data-testid="account-policy-recover"
                  :disabled="stateActionLoading"
                  @click="recoverAccountState"
                >
                  <Icon name="refresh" size="sm" />
                  <span>{{ t('admin.accountPolicy.state.recover') }}</span>
                </button>
              </div>

              <div class="mt-4 grid gap-4 sm:grid-cols-2">
                <div>
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountPolicy.state.status') }}</p>
                  <p class="mt-1 text-sm font-medium" :class="statusTextClass(selectedAccount.status)">
                    {{ selectedAccount.status }}
                  </p>
                </div>
                <div class="flex items-center justify-between gap-4 sm:block">
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountPolicy.state.scheduling') }}</p>
                  <div class="mt-1 flex items-center gap-3">
                    <Toggle
                      :model-value="selectedAccount.schedulable"
                      data-testid="account-policy-schedulable"
                      @update:model-value="updateSchedulable"
                    />
                    <span class="text-sm text-gray-700 dark:text-gray-300">
                      {{ selectedAccount.schedulable ? t('admin.accountPolicy.state.enabled') : t('admin.accountPolicy.state.paused') }}
                    </span>
                  </div>
                </div>
              </div>

              <div class="mt-4 space-y-1 text-xs text-gray-500 dark:text-gray-400">
                <p v-for="entry in runtimeStateEntries" :key="entry">{{ entry }}</p>
                <p v-if="runtimeStateEntries.length === 0">{{ t('admin.accountPolicy.state.noRuntimeBlock') }}</p>
              </div>
              <div v-if="selectedAccount.error_message" class="mt-3 border-l-2 border-red-400 pl-3">
                <p class="text-xs font-medium text-red-600 dark:text-red-400">
                  {{ t('admin.accountPolicy.state.errorMessage') }}
                </p>
                <p class="mt-1 break-words text-xs text-gray-600 dark:text-gray-300">
                  {{ selectedAccount.error_message }}
                </p>
              </div>
            </section>

            <section class="border-b border-gray-200 p-5 dark:border-dark-700">
              <div class="flex items-center justify-between gap-4">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.accountPolicy.relay.title') }}
                </h3>
                <Toggle
                  v-if="policy.relay_failure_budget.supported"
                  v-model="form.relay_failure_budget.enabled"
                  data-testid="account-policy-relay-enabled"
                />
              </div>
              <p v-if="!policy.relay_failure_budget.supported" class="mt-3 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.accountPolicy.relay.unavailable') }}
              </p>
              <div v-else class="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.relay.windowMinutes') }}</span>
                  <input v-model.number="form.relay_failure_budget.window_minutes" class="input w-full" type="number" min="1" max="1440" step="1" :disabled="!form.relay_failure_budget.enabled" data-testid="account-policy-relay-window" />
                </label>
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.relay.thresholdPercent') }}</span>
                  <input v-model.number="form.relay_failure_budget.failure_threshold_percent" class="input w-full" type="number" min="1" max="100" step="1" :disabled="!form.relay_failure_budget.enabled" />
                </label>
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.relay.minRequests') }}</span>
                  <input v-model.number="form.relay_failure_budget.min_requests" class="input w-full" type="number" min="1" max="10000" step="1" :disabled="!form.relay_failure_budget.enabled" />
                </label>
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.relay.consecutiveFailures') }}</span>
                  <input v-model.number="form.relay_failure_budget.consecutive_failures" class="input w-full" type="number" min="1" max="1000" step="1" :disabled="!form.relay_failure_budget.enabled" />
                </label>
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.relay.cooldownMinutes') }}</span>
                  <input v-model.number="form.relay_failure_budget.cooldown_minutes" class="input w-full" type="number" min="1" max="1440" step="1" :disabled="!form.relay_failure_budget.enabled" />
                </label>
              </div>
            </section>

            <section class="border-b border-gray-200 p-5 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.accountPolicy.quota.title') }}
              </h3>
              <p v-if="!policy.quota.supported" class="mt-3 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.accountPolicy.quota.unavailable') }}
              </p>
              <div v-else class="mt-4 grid gap-4 sm:grid-cols-3">
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.quota.total') }}</span>
                  <input v-model.number="form.quota.total_limit" class="input w-full" type="number" min="0" step="0.01" data-testid="account-policy-quota-total" />
                  <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.accountPolicy.quota.used', { value: selectedAccount.quota_used ?? 0 }) }}
                  </span>
                </label>
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.quota.daily') }}</span>
                  <input v-model.number="form.quota.daily_limit" class="input w-full" type="number" min="0" step="0.01" />
                  <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.accountPolicy.quota.used', { value: selectedAccount.quota_daily_used ?? 0 }) }}
                  </span>
                </label>
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.quota.weekly') }}</span>
                  <input v-model.number="form.quota.weekly_limit" class="input w-full" type="number" min="0" step="0.01" />
                  <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.accountPolicy.quota.used', { value: selectedAccount.quota_weekly_used ?? 0 }) }}
                  </span>
                </label>
                <p class="text-xs text-gray-500 dark:text-gray-400 sm:col-span-3">
                  {{ t('admin.accountPolicy.quota.unlimited') }}
                </p>
              </div>
            </section>

            <section class="p-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.accountPolicy.rate.title') }}
              </h3>
              <div class="mt-4 grid gap-4 md:grid-cols-[minmax(160px,220px)_minmax(0,1fr)]">
                <label class="block">
                  <span class="input-label">{{ t('admin.accountPolicy.rate.multiplier') }}</span>
                  <input
                    v-model.number="form.scheduling_rate.rate_multiplier"
                    class="input w-full"
                    :class="form.scheduling_rate.sync_mode === 'auto_overwrite' ? 'cursor-not-allowed opacity-60' : ''"
                    type="number"
                    min="0"
                    step="0.01"
                    :disabled="form.scheduling_rate.sync_mode === 'auto_overwrite'"
                    data-testid="account-policy-rate"
                  />
                </label>
                <div>
                  <div class="inline-flex max-w-full rounded-md border border-gray-300 p-0.5 dark:border-dark-600" role="group">
                    <button
                      type="button"
                      class="min-w-0 px-3 py-1.5 text-sm"
                      :class="form.scheduling_rate.sync_mode === 'auto_overwrite' ? 'bg-primary-600 text-white' : 'text-gray-700 dark:text-gray-300'"
                      data-testid="account-policy-rate-auto"
                      @click="form.scheduling_rate.sync_mode = 'auto_overwrite'"
                    >
                      {{ t('admin.accountPolicy.rate.autoOverwrite') }}
                    </button>
                    <button
                      type="button"
                      class="min-w-0 px-3 py-1.5 text-sm"
                      :class="form.scheduling_rate.sync_mode === 'manual_lock' ? 'bg-primary-600 text-white' : 'text-gray-700 dark:text-gray-300'"
                      data-testid="account-policy-rate-manual"
                      @click="form.scheduling_rate.sync_mode = 'manual_lock'"
                    >
                      {{ t('admin.accountPolicy.rate.manualLock') }}
                    </button>
                  </div>
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ form.scheduling_rate.sync_mode === 'auto_overwrite' ? t('admin.accountPolicy.rate.autoOverwriteHint') : t('admin.accountPolicy.rate.manualLockHint') }}
                  </p>
                </div>
              </div>
            </section>

            <div class="flex justify-end border-t border-gray-200 px-5 py-4 dark:border-dark-700">
              <button type="submit" class="btn btn-primary" :disabled="saving" data-testid="account-policy-save">
                <Icon v-if="!saving" name="check" size="sm" />
                <span>{{ t('admin.accountPolicy.save') }}</span>
              </button>
            </div>
          </form>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Toggle from '@/components/common/Toggle.vue'
import { adminAPI } from '@/api/admin'
import type {
  AccountPolicySettings,
  UpdateAccountPolicySettingsRequest
} from '@/api/admin/accounts'
import { useAppStore } from '@/stores'
import type { Account } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const accounts = ref<Account[]>([])
const selectedAccount = ref<Account | null>(null)
const policy = ref<AccountPolicySettings | null>(null)
const searchInput = ref('')
const activeSearch = ref('')
const accountsLoading = ref(false)
const policyLoading = ref(false)
const saving = ref(false)
const stateActionLoading = ref(false)
let policyRequest = 0

const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const form = reactive<Required<UpdateAccountPolicySettingsRequest>>({
  relay_failure_budget: {
    enabled: false,
    window_minutes: 10,
    failure_threshold_percent: 30,
    min_requests: 10,
    consecutive_failures: 5,
    cooldown_minutes: 2
  },
  quota: {
    total_limit: 0,
    daily_limit: 0,
    weekly_limit: 0
  },
  scheduling_rate: {
    rate_multiplier: 1,
    sync_mode: 'auto_overwrite'
  }
})

const runtimeStateEntries = computed(() => {
  const account = selectedAccount.value
  if (!account) return []
  const entries: string[] = []
  if (account.rate_limit_reset_at) {
    entries.push(t('admin.accountPolicy.state.rateLimitUntil', { time: formatTime(account.rate_limit_reset_at) }))
  }
  if (account.overload_until) {
    entries.push(t('admin.accountPolicy.state.overloadUntil', { time: formatTime(account.overload_until) }))
  }
  if (account.temp_unschedulable_until) {
    entries.push(t('admin.accountPolicy.state.temporaryUntil', { time: formatTime(account.temp_unschedulable_until) }))
  }
  if (account.quota_rate_limit) {
    entries.push(t('admin.accountPolicy.state.quotaLimited', {
      window: account.quota_rate_limit.window,
      utilization: account.quota_rate_limit.utilization,
      time: account.quota_rate_limit.reset_at ? formatTime(account.quota_rate_limit.reset_at) : '-'
    }))
  }
  return entries
})

function applyPolicy(next: AccountPolicySettings) {
  policy.value = next
  Object.assign(form.relay_failure_budget, {
    enabled: next.relay_failure_budget.enabled,
    window_minutes: next.relay_failure_budget.window_minutes,
    failure_threshold_percent: next.relay_failure_budget.failure_threshold_percent,
    min_requests: next.relay_failure_budget.min_requests,
    consecutive_failures: next.relay_failure_budget.consecutive_failures,
    cooldown_minutes: next.relay_failure_budget.cooldown_minutes
  })
  Object.assign(form.quota, {
    total_limit: next.quota.total_limit,
    daily_limit: next.quota.daily_limit,
    weekly_limit: next.quota.weekly_limit
  })
  Object.assign(form.scheduling_rate, next.scheduling_rate)
}

async function loadAccounts() {
  accountsLoading.value = true
  try {
    const result = await adminAPI.accounts.list(pagination.page, pagination.pageSize, {
      search: activeSearch.value || undefined,
      lite: '1'
    })
    accounts.value = result.items || []
    pagination.total = result.total || 0
    const current = accounts.value.find((item) => item.id === selectedAccount.value?.id)
    const next = current || accounts.value[0] || null
    if (next) {
      await selectAccount(next)
    } else {
      selectedAccount.value = null
      policy.value = null
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accountPolicy.loadFailed'))
  } finally {
    accountsLoading.value = false
  }
}

async function loadPolicy(accountID: number) {
  const request = ++policyRequest
  policyLoading.value = true
  try {
    const next = await adminAPI.accounts.getPolicySettings(accountID)
    if (request !== policyRequest || selectedAccount.value?.id !== accountID) return
    applyPolicy(next)
  } catch (error: any) {
    if (request === policyRequest) {
      policy.value = null
      appStore.showError(error?.message || t('admin.accountPolicy.loadFailed'))
    }
  } finally {
    if (request === policyRequest) policyLoading.value = false
  }
}

async function selectAccount(account: Account) {
  selectedAccount.value = account
  policy.value = null
  await loadPolicy(account.id)
}

function applySearch() {
  activeSearch.value = searchInput.value.trim()
  pagination.page = 1
  loadAccounts()
}

function changePage(page: number) {
  pagination.page = page
  loadAccounts()
}

function validatePolicy(): string | null {
  if (policy.value?.relay_failure_budget.supported && form.relay_failure_budget.enabled) {
    const relay = form.relay_failure_budget
    if (
      !validInteger(relay.window_minutes, 1, 1440) ||
      !validInteger(relay.failure_threshold_percent, 1, 100) ||
      !validInteger(relay.min_requests, 1, 10000) ||
      !validInteger(relay.consecutive_failures, 1, 1000) ||
      !validInteger(relay.cooldown_minutes, 1, 1440)
    ) return t('admin.accountPolicy.validation.relay')
  }
  if (policy.value?.quota.supported) {
    if (![form.quota.total_limit, form.quota.daily_limit, form.quota.weekly_limit].every(validNonNegative)) {
      return t('admin.accountPolicy.validation.quota')
    }
  }
  if (!validNonNegative(form.scheduling_rate.rate_multiplier)) {
    return t('admin.accountPolicy.validation.rate')
  }
  return null
}

async function savePolicy() {
  if (!selectedAccount.value || !policy.value) return
  const validationError = validatePolicy()
  if (validationError) {
    appStore.showError(validationError)
    return
  }
  const payload: UpdateAccountPolicySettingsRequest = {
    scheduling_rate: { ...form.scheduling_rate }
  }
  if (policy.value.relay_failure_budget.supported) {
    payload.relay_failure_budget = { ...form.relay_failure_budget }
  }
  if (policy.value.quota.supported) {
    payload.quota = { ...form.quota }
  }
  saving.value = true
  try {
    const updated = await adminAPI.accounts.updatePolicySettings(selectedAccount.value.id, payload)
    applyPolicy(updated)
    appStore.showSuccess(t('admin.accountPolicy.saved'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accountPolicy.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function recoverAccountState() {
  if (!selectedAccount.value || stateActionLoading.value) return
  stateActionLoading.value = true
  try {
    const updated = await adminAPI.accounts.recoverState(selectedAccount.value.id)
    replaceSelectedAccount(updated)
    appStore.showSuccess(t('admin.accountPolicy.state.recovered'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accountPolicy.state.recoverFailed'))
  } finally {
    stateActionLoading.value = false
  }
}

async function updateSchedulable(schedulable: boolean) {
  if (!selectedAccount.value || stateActionLoading.value) return
  stateActionLoading.value = true
  try {
    const updated = await adminAPI.accounts.setSchedulable(selectedAccount.value.id, schedulable)
    replaceSelectedAccount(updated)
    appStore.showSuccess(t('admin.accountPolicy.state.schedulingUpdated'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accountPolicy.state.schedulingFailed'))
  } finally {
    stateActionLoading.value = false
  }
}

function replaceSelectedAccount(updated: Account) {
  selectedAccount.value = updated
  const index = accounts.value.findIndex((account) => account.id === updated.id)
  if (index >= 0) accounts.value[index] = updated
}

function statusTextClass(status: Account['status']): string {
  if (status === 'active') return 'text-green-600 dark:text-green-400'
  if (status === 'error') return 'text-red-600 dark:text-red-400'
  return 'text-gray-500 dark:text-gray-400'
}

function validInteger(value: number, min: number, max: number): boolean {
  return Number.isInteger(Number(value)) && Number(value) >= min && Number(value) <= max
}

function validNonNegative(value: number): boolean {
  return Number.isFinite(Number(value)) && Number(value) >= 0
}

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

onMounted(loadAccounts)
</script>
