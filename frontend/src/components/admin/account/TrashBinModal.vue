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
          class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600"
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate font-medium text-gray-900 dark:text-gray-100">{{ acc.name }}</span>
              <span class="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                {{ acc.platform }} / {{ acc.type }}
              </span>
            </div>
            <div class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
              ID: {{ acc.id }}
              <span v-if="acc.deleted_at"> · {{ t('admin.accounts.deletedAt') }}: {{ formatTime(acc.deleted_at) }}</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1.5">
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
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { Icon } from '@/components/icons'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; restored: [] }>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const trashAccounts = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchQuery = ref('')

const showPermDeleteDialog = ref(false)
const permDeleteAcc = ref<any>(null)

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

async function handleRestore(acc: any) {
  try {
    await adminAPI.accounts.restoreFromTrash(acc.id)
    appStore.showSuccess(t('admin.accounts.restoreFromTrashSuccess'))
    await loadTrash()
    emit('restored')
  } catch (e: any) {
    appStore.showError(e?.response?.data?.message || t('admin.accounts.restoreFromTrashFailed'))
  }
}

function handlePermanentDelete(acc: any) {
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

function formatTime(ts: string): string {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString()
  } catch {
    return ts
  }
}

watch(() => props.show, (val) => {
  if (val) {
    page.value = 1
    searchQuery.value = ''
    loadTrash()
  }
})
</script>
