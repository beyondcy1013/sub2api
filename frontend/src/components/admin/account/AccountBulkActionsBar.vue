<template>
  <div class="mb-4 flex flex-col lg:flex-row lg:flex-wrap lg:items-center lg:justify-between gap-3 rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20">
    <div class="hidden lg:flex flex-wrap items-center gap-2">
      <span v-if="allResultsSelected" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selectedAll', { count: selectedIds.length }) }}
      </span>
      <span v-else-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkEdit.title') }}
      </span>
      <template v-if="selectedIds.length > 0">
        <button
          @click="$emit('select-page')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
        </button>
      </template>
      <template v-if="!allResultsSelected && totalResults > selectedIds.length">
        <span v-if="selectedIds.length > 0" class="text-gray-300 dark:text-primary-800">•</span>
        <button
          :disabled="selectingAll"
          @click="$emit('select-all-results')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 disabled:cursor-not-allowed disabled:opacity-60 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{
            selectingAll
              ? t('admin.accounts.bulkActions.selectingAll')
              : t('admin.accounts.bulkActions.selectAllResults', { count: totalResults })
          }}
        </button>
      </template>
      <template v-if="selectedIds.length > 0">
        <span class="text-gray-300 dark:text-primary-800">•</span>
        <button
          @click="$emit('clear')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </template>
      <input
        data-test="bulk-account-filter"
        type="search"
        class="input h-8 w-52 text-sm"
        :value="searchQuery"
        :placeholder="t('admin.accounts.bulkActions.filterPlaceholder')"
        @input="$emit('update:search-query', ($event.target as HTMLInputElement).value)"
      />
      <Select
        v-model="quickProxyId"
        data-test="quick-proxy-select"
        class="w-40"
        :options="proxyOptions"
        :placeholder="quickUpdating === 'proxy' ? t('admin.accounts.bulkActions.updatingProxy') : t('admin.accounts.bulkActions.selectProxy')"
        :disabled="selectedIds.length === 0 || quickUpdating !== null"
        @change="handleQuickProxyChange"
      />
      <Select
        v-model="quickGroupId"
        data-test="quick-group-select"
        class="w-40"
        :options="groupOptions"
        :placeholder="quickUpdating === 'group' ? t('admin.accounts.bulkActions.updatingGroup') : t('admin.accounts.bulkActions.selectGroup')"
        :disabled="selectedIds.length === 0 || quickUpdating !== null"
        @change="handleQuickGroupChange"
      />
    </div>
    <div class="hidden lg:flex flex-wrap gap-2">
      <template v-if="selectedIds.length > 0">
        <button
          data-test="batch-test-and-mark"
          class="btn btn-secondary btn-sm"
          :disabled="testingSelected"
          @click="$emit('test-and-mark')"
        >
          {{ testingSelected ? t('admin.accounts.bulkActions.testingAndMarking') : t('admin.accounts.bulkActions.testAndMark') }}
        </button>
        <button
          data-test="batch-pelican-test"
          class="btn btn-secondary btn-sm"
          @click="$emit('pelican-test')"
        >
          {{ t('admin.accounts.pelicanAction') }}
        </button>
        <button v-if="showDelete" data-test="bulk-delete" @click="$emit('delete')" class="btn btn-danger btn-sm">{{ t('admin.accounts.bulkActions.delete') }}</button>
        <button
          v-if="showPermanentDelete"
          data-test="bulk-permanent-delete"
          class="btn btn-danger btn-sm"
          :disabled="permanentDeleting"
          @click="$emit('permanent-delete')"
        >
          {{ permanentDeleting ? t('admin.accounts.bulkActions.permanentlyDeleting') : t('admin.accounts.bulkActions.permanentDelete') }}
        </button>
        <button @click="$emit('reset-status')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
        <button @click="$emit('refresh-token')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
        <button @click="$emit('probe-upstream-billing')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}</button>
        <button @click="$emit('toggle-schedulable', true)" class="btn btn-success btn-sm">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
        <button @click="$emit('toggle-schedulable', false)" class="btn btn-warning btn-sm">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
        <button @click="$emit('edit-selected')" class="btn btn-primary btn-sm">{{ t('admin.accounts.bulkActions.edit') }}</button>
      </template>
      <button
        data-test="bulk-primary-action"
        class="btn btn-secondary btn-sm"
        :disabled="refreshingUsage"
        @click="$emit('refresh-usage')"
      >
        {{
          refreshingUsage
            ? t('admin.accounts.bulkActions.refreshingUsage')
            : t('admin.accounts.bulkActions.refreshUsage')
        }}
      </button>
      <button
        data-test="bulk-primary-action"
        class="btn btn-primary btn-sm"
        :disabled="refreshingUsage"
        @click="$emit('edit-filtered')"
      >
        {{ t('admin.accounts.bulkEdit.submit') }}
      </button>
    </div>

    <div class="flex items-center justify-between gap-2 lg:hidden">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <span
          class="truncate text-sm font-medium text-primary-900 dark:text-primary-100"
          :title="mobileSelectionText"
        >
          {{ mobileSelectionText }}
        </span>
        <button
          v-if="selectedIds.length > 0"
          type="button"
          class="shrink-0 text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
          @click="$emit('clear')"
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </div>
      <div class="relative shrink-0" data-test="mobile-bulk-actions-root">
        <button
          type="button"
          data-test="mobile-bulk-actions-trigger"
          class="btn btn-secondary gap-1.5 px-2.5"
          :aria-expanded="mobileMenuOpen"
          @click="mobileMenuOpen = !mobileMenuOpen"
        >
          <Icon name="more" size="sm" />
          <span>{{ t('admin.accounts.bulkActions.mobileMenu') }}</span>
          <Icon :name="mobileMenuOpen ? 'chevronUp' : 'chevronDown'" size="xs" />
        </button>
        <div
          v-if="mobileMenuOpen"
          data-test="mobile-bulk-actions-panel"
          class="absolute right-0 z-50 mt-2 w-[min(calc(100vw-2rem),16rem)] rounded-xl border border-gray-200 bg-white p-3 shadow-xl dark:border-dark-700 dark:bg-dark-800"
        >
          <input
            data-test="bulk-account-filter"
            type="search"
            class="input h-9 w-full text-sm"
            :value="searchQuery"
            :placeholder="t('admin.accounts.bulkActions.filterPlaceholder')"
            @input="$emit('update:search-query', ($event.target as HTMLInputElement).value)"
          />

          <div class="mt-3 space-y-1.5">
            <button
              type="button"
              class="mobile-bulk-action"
              :disabled="selectingAll || !canSelectAllResults"
              @click="$emit('select-all-results')"
            >
              {{
                selectingAll
                  ? t('admin.accounts.bulkActions.selectingAll')
                  : t('admin.accounts.bulkActions.selectAllResults', { count: totalResults })
              }}
            </button>
            <button
              type="button"
              class="mobile-bulk-action"
              :disabled="selectedIds.length === 0"
              @click="$emit('select-page')"
            >
              {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
            </button>
          </div>

          <div class="mt-3 space-y-2">
            <Select
              v-model="quickProxyId"
              data-test="quick-proxy-select"
              :options="proxyOptions"
              :placeholder="quickUpdating === 'proxy' ? t('admin.accounts.bulkActions.updatingProxy') : t('admin.accounts.bulkActions.selectProxy')"
              :disabled="selectedIds.length === 0 || quickUpdating !== null"
              @change="handleQuickProxyChange"
            />
            <Select
              v-model="quickGroupId"
              data-test="quick-group-select"
              :options="groupOptions"
              :placeholder="quickUpdating === 'group' ? t('admin.accounts.bulkActions.updatingGroup') : t('admin.accounts.bulkActions.selectGroup')"
              :disabled="selectedIds.length === 0 || quickUpdating !== null"
              @change="handleQuickGroupChange"
            />
          </div>

          <div class="mt-3 border-t border-gray-100 pt-3 dark:border-dark-700">
            <div class="space-y-1.5">
              <template v-if="selectedIds.length > 0">
                <button
                  type="button"
                  data-test="batch-test-and-mark"
                  class="mobile-bulk-action"
                  :disabled="testingSelected"
                  @click="$emit('test-and-mark')"
                >
                  {{ testingSelected ? t('admin.accounts.bulkActions.testingAndMarking') : t('admin.accounts.bulkActions.testAndMark') }}
                </button>
                <button
                  type="button"
                  data-test="batch-pelican-test"
                  class="mobile-bulk-action"
                  @click="$emit('pelican-test')"
                >
                  {{ t('admin.accounts.pelicanAction') }}
                </button>
                <button
                  v-if="showDelete"
                  type="button"
                  data-test="bulk-delete"
                  class="mobile-bulk-action mobile-bulk-action-danger"
                  @click="$emit('delete')"
                >
                  {{ t('admin.accounts.bulkActions.delete') }}
                </button>
                <button
                  v-if="showPermanentDelete"
                  type="button"
                  data-test="bulk-permanent-delete"
                  class="mobile-bulk-action mobile-bulk-action-danger"
                  :disabled="permanentDeleting"
                  @click="$emit('permanent-delete')"
                >
                  {{ permanentDeleting ? t('admin.accounts.bulkActions.permanentlyDeleting') : t('admin.accounts.bulkActions.permanentDelete') }}
                </button>
                <button type="button" class="mobile-bulk-action" @click="$emit('reset-status')">
                  {{ t('admin.accounts.bulkActions.resetStatus') }}
                </button>
                <button type="button" class="mobile-bulk-action" @click="$emit('refresh-token')">
                  {{ t('admin.accounts.bulkActions.refreshToken') }}
                </button>
                <button type="button" class="mobile-bulk-action" @click="$emit('probe-upstream-billing')">
                  {{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}
                </button>
                <button type="button" class="mobile-bulk-action" @click="$emit('toggle-schedulable', true)">
                  {{ t('admin.accounts.bulkActions.enableScheduling') }}
                </button>
                <button type="button" class="mobile-bulk-action" @click="$emit('toggle-schedulable', false)">
                  {{ t('admin.accounts.bulkActions.disableScheduling') }}
                </button>
                <button type="button" class="mobile-bulk-action mobile-bulk-action-primary" @click="$emit('edit-selected')">
                  {{ t('admin.accounts.bulkActions.edit') }}
                </button>
              </template>
              <button
                type="button"
                data-test="bulk-primary-action"
                class="mobile-bulk-action"
                :disabled="refreshingUsage"
                @click="$emit('refresh-usage')"
              >
                {{ refreshingUsage ? t('admin.accounts.bulkActions.refreshingUsage') : t('admin.accounts.bulkActions.refreshUsage') }}
              </button>
              <button
                type="button"
                data-test="bulk-primary-action"
                class="mobile-bulk-action mobile-bulk-action-primary"
                @click="$emit('edit-filtered')"
              >
                {{ t('admin.accounts.bulkEdit.submit') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import type { AdminGroup, Proxy as ProxyConfig } from '@/types'

const props = withDefaults(defineProps<{
  selectedIds: number[]
  totalResults?: number
  selectingAll?: boolean
  allResultsSelected?: boolean
  quickUpdating?: 'proxy' | 'group' | null
  refreshingUsage?: boolean
  testingSelected?: boolean
  showDelete?: boolean
  showPermanentDelete?: boolean
  permanentDeleting?: boolean
  searchQuery?: string
  proxies?: ProxyConfig[]
  groups?: AdminGroup[]
}>(), {
  totalResults: 0,
  selectingAll: false,
  allResultsSelected: false,
  quickUpdating: null,
  refreshingUsage: false,
  testingSelected: false,
  showDelete: true,
  showPermanentDelete: false,
  permanentDeleting: false,
  searchQuery: '',
  proxies: () => [],
  groups: () => []
})
const emit = defineEmits<{
  delete: []
  'permanent-delete': []
  'update:search-query': [value: string]
  'edit-selected': []
  'edit-filtered': []
  clear: []
  'select-page': []
  'select-all-results': []
  'quick-set-proxy': [proxyId: number]
  'quick-set-group': [groupId: number]
  'toggle-schedulable': [enabled: boolean]
  'reset-status': []
  'refresh-token': []
  'probe-upstream-billing': []
  'refresh-usage': []
  'test-and-mark': []
  'pelican-test': []
}>()

const { t } = useI18n()
const quickProxyId = ref<number | null>(null)
const quickGroupId = ref<number | null>(null)
const mobileMenuOpen = ref(false)
const proxyOptions = computed(() => [
  { value: 0, label: t('admin.accounts.noProxy') },
  ...props.proxies.map(proxy => ({ value: proxy.id, label: proxy.name }))
])
const groupOptions = computed(() => [
  { value: 0, label: t('admin.accounts.bulkActions.noGroup') },
  ...props.groups.map(group => ({ value: group.id, label: group.name }))
])
const canSelectAllResults = computed(() => !props.allResultsSelected && props.totalResults > props.selectedIds.length)
const mobileSelectionText = computed(() => {
  if (props.allResultsSelected) {
    return t('admin.accounts.bulkActions.selectedAll', { count: props.selectedIds.length })
  }
  if (props.selectedIds.length > 0) {
    return t('admin.accounts.bulkActions.selected', { count: props.selectedIds.length })
  }
  return t('admin.accounts.bulkEdit.title')
})

const handleQuickProxyChange = (value: string | number | boolean | null) => {
  if (typeof value !== 'number') return
  emit('quick-set-proxy', value)
  nextTick(() => { quickProxyId.value = null })
}

const handleQuickGroupChange = (value: string | number | boolean | null) => {
  if (typeof value !== 'number') return
  emit('quick-set-group', value)
  nextTick(() => { quickGroupId.value = null })
}
</script>

<style scoped>
.mobile-bulk-action {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: flex-start;
  border-radius: 0.5rem;
  padding: 0.375rem 0.5rem;
  text-align: left;
  font-size: 0.8125rem;
  line-height: 1.25rem;
  color: rgb(55 65 81);
}

.mobile-bulk-action:hover:not(:disabled) {
  background-color: rgb(243 244 246);
}

.mobile-bulk-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.mobile-bulk-action-primary {
  background-color: rgb(2 132 199);
  font-weight: 500;
  color: white;
}

.mobile-bulk-action-primary:hover:not(:disabled) {
  background-color: rgb(14 165 233);
}

.mobile-bulk-action-danger {
  color: rgb(220 38 38);
}

.mobile-bulk-action-danger:hover:not(:disabled) {
  background-color: rgb(254 242 242);
}

</style>
