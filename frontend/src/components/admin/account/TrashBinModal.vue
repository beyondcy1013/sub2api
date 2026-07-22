<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.trashBinTitle')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.trashBinHint') }}
      </div>

      <!-- Search + filters -->
      <div class="flex flex-wrap items-center gap-2">
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('admin.accounts.searchAccounts')"
          class="input flex-1 min-w-[200px]"
          @keyup.enter="loadTrash"
        />
        <button @click="loadTrash" class="btn btn-secondary px-3">
          <Icon name="search" size="sm" />
        </button>
        <button @click="loadTrash" class="btn btn-secondary px-3" :disabled="loading">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <LoadingSpinner />
      </div>

      <!-- Empty -->
      <div v-else-if="trashAccounts.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.trashBinEmpty') }}
      </div>

      <!-- List -->
      <div v-else class="space-y-2 max-h-[50vh] overflow-y-auto">
        <div
          v-for="acc in trashAccounts"
          :key="acc.id"
          class="grid gap-3 rounded-lg border border-gray-200 px-3 py-3 dark:border-dark-600 sm:grid-cols-[minmax(0,1fr)_auto]"
        >
          <div class="min-w-0 space-y-2">
            <div class="flex items-center gap-2">
              <span class="truncate font-medium text-gray-900 dark:text-gray-100" :title="acc.name">{{ acc.name || '-' }}</span>
              <span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
                {{ acc.platform }} / {{ acc.type }}
              </span>
            </div>

            <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-3">
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                ID: <span class="font-mono text-gray-700 dark:text-gray-300">{{ acc.id ?? '-' }}</span>
              </div>
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.columns.createdAt') }}:
                <span data-test="trash-created-at" class="text-gray-700 dark:text-gray-300">{{ displayTime(acc.created_at) }}</span>
              </div>
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.deletedAt') }}:
                <span data-test="trash-deleted-at" class="text-gray-700 dark:text-gray-300">{{ displayTime(acc.deleted_at) }}</span>
              </div>
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.columns.lastUsed') }}:
                <span class="text-gray-700 dark:text-gray-300">{{ displayTime(acc.last_used_at) }}</span>
              </div>
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.columns.status') }}:
                <span class="text-gray-700 dark:text-gray-300">{{ displayStatus(acc.status) }}</span>
              </div>
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.columns.schedulable') }}:
                <span class="text-gray-700 dark:text-gray-300">{{ t(acc.schedulable ? 'admin.accounts.schedulableEnabled' : 'admin.accounts.schedulableDisabled') }}</span>
              </div>
              <div class="min-w-0 text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.columns.capacity') }}:
                <span class="font-mono text-gray-700 dark:text-gray-300">{{ acc.concurrency ?? '-' }}</span>
              </div>
            </div>

            <div class="border-t border-gray-100 pt-2 dark:border-dark-700">
              <div class="mb-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.trashUsageSummary') }}
              </div>
              <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-4">
                <div class="text-gray-500 dark:text-gray-400">
                  {{ t('admin.accounts.stats.requests') }}:
                  <span data-test="trash-requests" class="font-mono text-gray-700 dark:text-gray-300">{{ displayRequests(acc) }}</span>
                </div>
                <div class="text-gray-500 dark:text-gray-400">
                  {{ t('admin.accounts.stats.tokens') }}:
                  <span data-test="trash-tokens" class="font-mono text-gray-700 dark:text-gray-300">{{ displayTokens(acc) }}</span>
                </div>
                <div class="text-gray-500 dark:text-gray-400">
                  {{ t('usage.accountBilled') }}:
                  <span data-test="trash-account-cost" class="font-mono text-emerald-600 dark:text-emerald-400">{{ displayCost(acc.usage_stats?.cost) }}</span>
                </div>
                <div class="text-gray-500 dark:text-gray-400">
                  {{ t('usage.userBilled') }}:
                  <span data-test="trash-user-cost" class="font-mono text-gray-700 dark:text-gray-300">{{ displayCost(acc.usage_stats?.user_cost) }}</span>
                </div>
              </div>
            </div>

            <div v-if="acc.notes" class="truncate text-xs text-gray-500 dark:text-gray-400" :title="acc.notes">
              {{ t('admin.accounts.columns.notes') }}: {{ acc.notes }}
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1.5 self-start sm:justify-end">
            <button
              @click="handleRestore(acc)"
              class="inline-flex h-7 items-center justify-center rounded border border-emerald-200 px-2 text-xs font-medium text-emerald-700 transition-colors hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-300 dark:hover:bg-emerald-900/20"
            >
              {{ t('admin.accounts.restoreFromTrash') }}
            </button>
            <button
              @click="handlePermanentDelete(acc)"
              class="inline-flex h-7 items-center justify-center rounded border border-red-200 px-2 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20"
            >
              {{ t('admin.accounts.permanentDelete') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="total > pageSize" class="flex items-center justify-between pt-2">
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ (page - 1) * pageSize + 1 }}-{{ Math.min(page * pageSize, total) }} / {{ total }}
        </span>
        <div class="flex gap-1">
          <button
            :disabled="page <= 1"
            @click="page--; loadTrash()"
            class="btn btn-secondary px-2 py-1 text-xs disabled:opacity-50"
          >
            <Icon name="chevronLeft" size="xs" />
          </button>
          <button
            :disabled="page * pageSize >= total"
            @click="page++; loadTrash()"
            class="btn btn-secondary px-2 py-1 text-xs disabled:opacity-50"
          >
            <Icon name="chevronRight" size="xs" />
          </button>
        </div>
      </div>
    </div>

    <!-- Permanent delete confirmation -->
    <ConfirmDialog
      :show="showPermDeleteDialog"
      :title="t('admin.accounts.permanentDelete')"
      :message="t('admin.accounts.permanentDeleteConfirm', { name: permDeleteAcc?.name })"
      :confirm-text="t('admin.accounts.permanentDelete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmPermanentDelete"
      @cancel="showPermDeleteDialog = false"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { TrashedAccount } from '@/types'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { formatCompactNumber, formatCurrency, formatDateTime, formatNumber } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { Icon } from '@/components/icons'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; restored: [] }>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const trashAccounts = ref<TrashedAccount[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchQuery = ref('')

const showPermDeleteDialog = ref(false)
const permDeleteAcc = ref<TrashedAccount | null>(null)

async function loadTrash() {
  loading.value = true
  try {
    const res = await adminAPI.accounts.listTrashed({
      page: page.value,
      page_size: pageSize.value,
      search: searchQuery.value || undefined,
    })
    trashAccounts.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    appStore.showError(e?.response?.data?.message || t('admin.accounts.trashBinLoadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleRestore(acc: TrashedAccount) {
  try {
    await adminAPI.accounts.restoreFromTrash(acc.id)
    appStore.showSuccess(t('admin.accounts.restoreFromTrashSuccess'))
    await loadTrash()
    emit('restored')
  } catch (e: any) {
    appStore.showError(e?.response?.data?.message || t('admin.accounts.restoreFromTrashFailed'))
  }
}

function handlePermanentDelete(acc: TrashedAccount) {
  permDeleteAcc.value = acc
  showPermDeleteDialog.value = true
}

async function confirmPermanentDelete() {
  if (!permDeleteAcc.value) return
  try {
    await adminAPI.accounts.permanentDelete(permDeleteAcc.value.id)
    appStore.showSuccess(t('admin.accounts.permanentDeleteSuccess'))
    showPermDeleteDialog.value = false
    permDeleteAcc.value = null
    await loadTrash()
  } catch (e: any) {
    appStore.showError(e?.response?.data?.message || t('admin.accounts.permanentDeleteFailed'))
  }
}

function handleClose() {
  emit('close')
}

function displayTime(value: string | null | undefined): string {
  return value ? formatDateTime(value) || '-' : '-'
}

function displayStatus(status: TrashedAccount['status']): string {
  return status ? t(`admin.accounts.status.${status}`) : '-'
}

function displayRequests(account: TrashedAccount): string {
  return account.usage_stats ? formatNumber(account.usage_stats.requests) : '-'
}

function displayTokens(account: TrashedAccount): string {
  return account.usage_stats ? formatCompactNumber(account.usage_stats.tokens) : '-'
}

function displayCost(value: number | null | undefined): string {
  return value == null ? '-' : formatCurrency(value)
}

watch(() => props.show, (val) => {
  if (val) {
    page.value = 1
    searchQuery.value = ''
    loadTrash()
  }
})
</script>
