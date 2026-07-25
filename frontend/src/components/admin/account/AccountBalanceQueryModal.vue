<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.balanceQuery.title')"
    @close="emit('close')"
  >
    <div v-if="account" class="space-y-5">
      <div class="border-b border-gray-200 pb-3 dark:border-dark-600">
        <div class="flex min-w-0 items-baseline gap-2">
          <span class="truncate font-medium text-gray-900 dark:text-white">{{ account.name }}</span>
          <span class="shrink-0 font-mono text-xs text-gray-500">#{{ account.id }}</span>
        </div>
        <div v-if="baseURL" class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">
          {{ baseURL }}
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="input-label">{{ t('admin.accounts.balanceQuery.scheme') }}</span>
          <select
            v-model="scheme"
            data-testid="balance-query-scheme"
            class="input mt-1"
            :disabled="querying"
          >
            <option v-for="option in schemeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label class="block">
          <span class="input-label">{{ t('admin.accounts.balanceQuery.apiURL') }}</span>
          <input
            v-model.trim="apiURL"
            data-testid="balance-query-api-url"
            type="text"
            class="input mt-1 font-mono text-sm"
            :placeholder="t('admin.accounts.balanceQuery.apiURLPlaceholder')"
            :disabled="querying"
          />
        </label>
      </div>

      <div v-if="querying" class="border-l-4 border-primary-500 bg-primary-50 px-3 py-3 text-sm text-primary-700 dark:bg-primary-950/30 dark:text-primary-300">
        {{ t('admin.accounts.balanceQuery.querying') }}
      </div>

      <div v-else-if="result?.success" class="border-l-4 border-emerald-500 bg-emerald-50 px-3 py-3 dark:bg-emerald-950/30">
        <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <div class="text-2xl font-semibold text-emerald-700 dark:text-emerald-300">
            {{ result.balance }}
            <span v-if="result.unit" class="text-sm font-medium">{{ result.unit }}</span>
            <span v-if="result.unlimited" class="ml-2 text-sm font-medium">{{ t('admin.accounts.balanceQuery.unlimited') }}</span>
          </div>
          <div class="text-xs text-emerald-700 dark:text-emerald-300">
            {{ schemeLabel(result.scheme) }}
          </div>
        </div>
        <div v-if="result.api_url" class="mt-2 break-all font-mono text-xs text-emerald-700 dark:text-emerald-300">
          {{ result.api_url }}
        </div>
        <div class="mt-1 text-xs text-emerald-600 dark:text-emerald-400">
          {{ formatQueriedAt(result.queried_at) }}
        </div>
      </div>

      <div v-else-if="result" class="border-l-4 border-red-500 bg-red-50 px-3 py-3 dark:bg-red-950/30">
        <div class="text-sm font-medium text-red-700 dark:text-red-300">
          {{ t('admin.accounts.balanceQuery.notDetected') }}
        </div>
        <div v-if="result.attempts.length" class="mt-2 space-y-1">
          <div
            v-for="attempt in result.attempts"
            :key="`${attempt.scheme}:${attempt.api_url}`"
            class="flex min-w-0 items-center justify-between gap-3 text-xs text-red-600 dark:text-red-400"
          >
            <span class="truncate">{{ schemeLabel(attempt.scheme) }}</span>
            <span class="shrink-0 font-mono">{{ attempt.http_status || attempt.error || '-' }}</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="querying" @click="emit('close')">
        {{ t('common.close') }}
      </button>
      <button
        type="button"
        data-testid="balance-query-submit"
        class="btn btn-primary"
        :disabled="!canQuery"
        @click="runQuery(true)"
      >
        {{ querying ? t('common.processing') : t('admin.accounts.balanceQuery.query') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type {
  Account,
  AccountBalanceQueryConfig,
  AccountBalanceQueryResult,
  AccountBalanceQueryScheme,
} from '@/types'

const props = defineProps<{
  show: boolean
  account: Account | null
}>()
const emit = defineEmits<{
  close: []
  updated: [account: Account]
}>()
const { t } = useI18n()
const appStore = useAppStore()

const scheme = ref<AccountBalanceQueryScheme>('auto')
const apiURL = ref('')
const persistedScheme = ref<AccountBalanceQueryScheme>('auto')
const persistedAPIURL = ref('')
const querying = ref(false)
const result = ref<AccountBalanceQueryResult | null>(null)

const schemeOptions = computed(() => [
  { value: 'auto' as const, label: t('admin.accounts.balanceQuery.schemes.auto') },
  { value: 'sub2api' as const, label: t('admin.accounts.balanceQuery.schemes.sub2api') },
  { value: 'newapi' as const, label: t('admin.accounts.balanceQuery.schemes.newapi') },
  { value: 'openai_compatible' as const, label: t('admin.accounts.balanceQuery.schemes.openaiCompatible') },
  { value: 'cpa' as const, label: t('admin.accounts.balanceQuery.schemes.cpa') },
  { value: 'custom' as const, label: t('admin.accounts.balanceQuery.schemes.custom') },
])

const baseURL = computed(() => {
  const value = props.account?.credentials?.base_url
  return typeof value === 'string' ? value : ''
})

const canQuery = computed(() => Boolean(
  props.account &&
  !querying.value &&
  !(scheme.value === 'custom' && apiURL.value.trim() === '') &&
  !(scheme.value === 'auto' && apiURL.value.trim() !== '')
))

const schemeLabel = (value?: AccountBalanceQueryScheme) => {
  return schemeOptions.value.find(option => option.value === value)?.label ?? value ?? '-'
}

const formatQueriedAt = (value: string) => {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

const currentAccountConfig = (): AccountBalanceQueryConfig => {
  const value = props.account?.extra?.balance_query
  if (!value || typeof value !== 'object') return { scheme: 'auto' }
  return value as AccountBalanceQueryConfig
}

const reset = () => {
  const config = currentAccountConfig()
  scheme.value = config.scheme || 'auto'
  apiURL.value = config.api_url || ''
  persistedScheme.value = scheme.value
  persistedAPIURL.value = apiURL.value
  result.value = null
}

const emitConfigUpdate = (config: AccountBalanceQueryConfig) => {
  if (!props.account) return
  emit('updated', {
    ...props.account,
    extra: {
      ...(props.account.extra || {}),
      balance_query: config,
    },
  })
}

const saveConfigIfChanged = async () => {
  if (!props.account) return currentAccountConfig()
  const normalizedAPIURL = apiURL.value.trim()
  if (scheme.value === persistedScheme.value && normalizedAPIURL === persistedAPIURL.value) {
    return currentAccountConfig()
  }
  const config = await adminAPI.accounts.updateAccountBalanceQueryConfig(props.account.id, {
    scheme: scheme.value,
    api_url: normalizedAPIURL,
  })
  scheme.value = config.scheme
  apiURL.value = config.api_url || ''
  persistedScheme.value = scheme.value
  persistedAPIURL.value = apiURL.value
  emitConfigUpdate(config)
  return config
}

const runQuery = async (saveChanges: boolean) => {
  const target = props.account
  if (!target || querying.value || (saveChanges && !canQuery.value)) return
  querying.value = true
  result.value = null
  try {
    const savedConfig = saveChanges ? await saveConfigIfChanged() : currentAccountConfig()
    const response = await adminAPI.accounts.queryAccountBalance(target.id)
    if (props.account?.id !== target.id) return
    result.value = response
    if (!response.success || !response.scheme) {
      appStore.showError(t('admin.accounts.balanceQuery.notDetected'))
      return
    }

    const detectedFallback = response.scheme !== savedConfig.scheme
    const nextConfig: AccountBalanceQueryConfig = {
      ...savedConfig,
      scheme: response.scheme,
      api_url: detectedFallback ? '' : (savedConfig.api_url || ''),
      detected_api_url: response.api_url,
      last_result: {
        balance: response.balance,
        unit: response.unit || '',
        unlimited: response.unlimited,
        queried_at: response.queried_at,
      },
    }
    scheme.value = nextConfig.scheme
    apiURL.value = nextConfig.api_url || ''
    persistedScheme.value = scheme.value
    persistedAPIURL.value = apiURL.value
    emitConfigUpdate(nextConfig)
    appStore.showSuccess(t('admin.accounts.balanceQuery.success'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.accounts.balanceQuery.failed')))
  } finally {
    querying.value = false
  }
}

watch(
  () => [props.show, props.account?.id] as const,
  ([show]) => {
    if (!show) return
    reset()
    void runQuery(false)
  },
  { immediate: true },
)
</script>
