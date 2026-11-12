<template>
  <div
    class="rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20"
    :class="[
      bare ? 'flex min-w-0 flex-wrap items-center gap-2 max-lg:w-full max-lg:basis-full' : 'mb-4 flex flex-col gap-3 lg:flex-row lg:flex-wrap lg:items-center lg:justify-between',
      selectedIds.length === 0 ? 'max-lg:hidden' : ''
    ]"
  >
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
          {{ t('admin.accounts.accountTesting.title') }}
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
        <button
          :disabled="refreshingBalance"
          @click="$emit('refresh-balance')"
          class="btn btn-secondary btn-sm"
        >
          {{ refreshingBalance ? t('admin.accounts.bulkActions.refreshingBalance') : t('admin.accounts.bulkActions.refreshBalance') }}
        </button>
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

    <div
      v-if="selectedIds.length > 0"
      class="flex items-center justify-between gap-2 lg:hidden"
      :class="bare ? 'min-w-0 flex-1' : 'w-full'"
      data-test="mobile-bulk-bar"
    >
      <span
        class="min-w-0 truncate text-sm font-medium text-primary-900 dark:text-primary-100"
        :title="mobileSelectionText"
      >
        {{ mobileSelectionText }}
      </span>
      <div class="flex shrink-0 items-center gap-2" data-test="mobile-bulk-actions-root">
        <button
          type="button"
          data-test="mobile-bulk-actions-trigger"
          class="btn btn-secondary min-h-[44px] gap-1.5 px-2.5"
          :aria-expanded="mobileSheetOpen"
          @click="mobileSheetOpen = true"
        >
          <Icon name="more" size="sm" />
          <span>{{ t('admin.accounts.bulkActions.title') }}</span>
          <Icon name="chevronUp" size="xs" />
        </button>
        <button
          type="button"
          data-test="mobile-bulk-clear"
          class="btn btn-ghost min-h-[44px] px-2.5"
          @click="$emit('clear')"
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </div>
    </div>

    <MobileActionSheet v-model="mobileSheetOpen" :title="t('admin.accounts.bulkActions.title')">
      <div class="space-y-4" data-test="mobile-bulk-actions-sheet">
        <div class="text-sm font-medium text-primary-900 dark:text-primary-100">
          {{ mobileSelectionText }}
        </div>

        <input
          data-test="bulk-account-filter"
          type="search"
          class="input h-11 w-full text-sm"
          :value="searchQuery"
          :placeholder="t('admin.accounts.bulkActions.filterPlaceholder')"
          @input="$emit('update:search-query', ($event.target as HTMLInputElement).value)"
        />

        <div>
          <div class="mobile-sheet-section">{{ t('admin.accounts.bulkActions.selectionScope') }}</div>
          <div class="mt-2 space-y-1.5">
            <button
              type="button"
              class="mobile-sheet-action"
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
              class="mobile-sheet-action"
              @click="$emit('select-page')"
            >
              {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
            </button>
          </div>
        </div>

        <div>
          <div class="mobile-sheet-section">{{ t('admin.accounts.bulkActions.quickSettings') }}</div>
          <div class="mt-2 space-y-2">
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
        </div>

        <div>
          <div class="mobile-sheet-section">{{ t('admin.accounts.bulkActions.selectedActions') }}</div>
          <div class="mt-2 space-y-1.5">
            <template v-if="selectedIds.length > 0">
              <button
                type="button"
                data-test="batch-test-and-mark"
                class="mobile-sheet-action"
                :disabled="testingSelected"
                @click="closeAndRun(() => $emit('test-and-mark'))"
              >
                {{ testingSelected ? t('admin.accounts.bulkActions.testingAndMarking') : t('admin.accounts.bulkActions.testAndMark') }}
              </button>
              <button
                type="button"
                data-test="batch-pelican-test"
                class="mobile-sheet-action"
                @click="closeAndRun(() => $emit('pelican-test'))"
              >
                {{ t('admin.accounts.accountTesting.title') }}
              </button>
              <button type="button" class="mobile-sheet-action" @click="closeAndRun(() => $emit('reset-status'))">
                {{ t('admin.accounts.bulkActions.resetStatus') }}
              </button>
              <button type="button" class="mobile-sheet-action" @click="closeAndRun(() => $emit('refresh-token'))">
                {{ t('admin.accounts.bulkActions.refreshToken') }}
              </button>
              <button type="button" class="mobile-sheet-action" @click="closeAndRun(() => $emit('probe-upstream-billing'))">
                {{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}
              </button>
              <button
                type="button"
                class="mobile-sheet-action"
                :disabled="refreshingBalance"
                @click="closeAndRun(() => $emit('refresh-balance'))"
              >
                {{ refreshingBalance ? t('admin.accounts.bulkActions.refreshingBalance') : t('admin.accounts.bulkActions.refreshBalance') }}
              </button>
              <button type="button" class="mobile-sheet-action" @click="closeAndRun(() => $emit('toggle-schedulable', true))">
                {{ t('admin.accounts.bulkActions.enableScheduling') }}
              </button>
              <button type="button" class="mobile-sheet-action" @click="closeAndRun(() => $emit('toggle-schedulable', false))">
                {{ t('admin.accounts.bulkActions.disableScheduling') }}
              </button>
              <button type="button" class="mobile-sheet-action mobile-sheet-action-primary" @click="closeAndRun(() => $emit('edit-selected'))">
                {{ t('admin.accounts.bulkActions.edit') }}
              </button>
            </template>
            <button
              type="button"
              data-test="bulk-primary-action"
              class="mobile-sheet-action"
              :disabled="refreshingUsage"
              @click="closeAndRun(() => $emit('refresh-usage'))"
            >
              {{ refreshingUsage ? t('admin.accounts.bulkActions.refreshingUsage') : t('admin.accounts.bulkActions.refreshUsage') }}
            </button>
            <button
              type="button"
              data-test="bulk-primary-action"
              class="mobile-sheet-action mobile-sheet-action-primary"
              @click="closeAndRun(() => $emit('edit-filtered'))"
            >
              {{ t('admin.accounts.bulkEdit.submit') }}
            </button>
          </div>
        </div>

        <div v-if="selectedIds.length > 0 && (showDelete || showPermanentDelete)">
          <div class="mobile-sheet-section text-red-500 dark:text-red-400">
            {{ t('admin.accounts.bulkActions.dangerActions') }}
          </div>
          <div class="mt-2 space-y-1.5">
            <button
              v-if="showDelete"
              type="button"
              data-test="bulk-delete"
              class="mobile-sheet-action mobile-sheet-action-danger"
              @click="closeAndRun(() => $emit('delete'))"
            >
              {{ t('admin.accounts.bulkActions.delete') }}
            </button>
            <button
              v-if="showPermanentDelete"
              type="button"
              data-test="bulk-permanent-delete"
              class="mobile-sheet-action mobile-sheet-action-danger"
              :disabled="permanentDeleting"
              @click="closeAndRun(() => $emit('permanent-delete'))"
            >
              {{ permanentDeleting ? t('admin.accounts.bulkActions.permanentlyDeleting') : t('admin.accounts.bulkActions.permanentDelete') }}
            </button>
          </div>
        </div>
      </div>
    </MobileActionSheet>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import MobileActionSheet from '@/components/common/MobileActionSheet.vue'
import type { AdminGroup, Proxy as ProxyConfig } from '@/types'

const props = withDefaults(defineProps<{
  selectedIds: number[]
  totalResults?: number
  selectingAll?: boolean
  allResultsSelected?: boolean
  quickUpdating?: 'proxy' | 'group' | null
  refreshingUsage?: boolean
  refreshingBalance?: boolean
  testingSelected?: boolean
  showDelete?: boolean
  showPermanentDelete?: boolean
  permanentDeleting?: boolean
  searchQuery?: string
  proxies?: ProxyConfig[]
  groups?: AdminGroup[]
  /** Render as a borderless inline control that can share the toolbar row (Android). */
  bare?: boolean
}>(), {
  totalResults: 0,
  selectingAll: false,
  allResultsSelected: false,
  quickUpdating: null,
  refreshingUsage: false,
  refreshingBalance: false,
  testingSelected: false,
  showDelete: true,
  showPermanentDelete: false,
  permanentDeleting: false,
  searchQuery: '',
  proxies: () => [],
  groups: () => [],
  bare: false
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
  'refresh-balance': []
  'test-and-mark': []
  'pelican-test': []
}>()

const { t } = useI18n()
const quickProxyId = ref<number | null>(null)
const quickGroupId = ref<number | null>(null)
const mobileSheetOpen = ref(false)

// Close first, then emit after the DOM flush so the sheet's focus restoration
// cannot steal focus back from any dialog the parent opens for the action.
const closeAndRun = async (action: () => void) => {
  mobileSheetOpen.value = false
  await nextTick()
  action()
}

// Nothing selected -> the bar disappears; an open sheet must not linger.
watch(() => props.selectedIds.length, count => {
  if (count === 0) mobileSheetOpen.value = false
})

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
