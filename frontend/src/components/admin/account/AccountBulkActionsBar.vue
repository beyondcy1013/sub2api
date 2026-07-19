<template>
  <div class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20">
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkEdit.title') }}
      </span>
      <button
        @click="$emit('select-page')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
      </button>
      <span class="text-gray-300 dark:text-primary-800">•</span>
      <button
        @click="$emit('select-all-pages')"
        :disabled="selectingAllPages"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 disabled:cursor-wait disabled:opacity-60 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{
          selectingAllPages
            ? t('admin.accounts.bulkActions.selectingAllPages')
            : t('admin.accounts.bulkActions.selectAllPages')
        }}
      </button>
      <template v-if="selectedIds.length > 0">
        <span class="text-gray-300 dark:text-primary-800">•</span>
        <button
          @click="$emit('clear')"
          class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </template>
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
    <div class="flex flex-wrap gap-2">
      <template v-if="selectedIds.length > 0">
        <button @click="$emit('delete')" class="btn btn-danger btn-sm">{{ t('admin.accounts.bulkActions.delete') }}</button>
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
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import type { AdminGroup, Proxy as ProxyConfig } from '@/types'

const props = withDefaults(defineProps<{
  selectedIds: number[]
  selectingAllPages?: boolean
  quickUpdating?: 'proxy' | 'group' | null
  refreshingUsage?: boolean
  proxies?: ProxyConfig[]
  groups?: AdminGroup[]
}>(), {
  selectingAllPages: false,
  quickUpdating: null,
  refreshingUsage: false,
  proxies: () => [],
  groups: () => []
})
const emit = defineEmits<{
  delete: []
  'edit-selected': []
  'edit-filtered': []
  clear: []
  'select-page': []
  'select-all-pages': []
  'quick-set-proxy': [proxyId: number]
  'quick-set-group': [groupId: number]
  'toggle-schedulable': [enabled: boolean]
  'reset-status': []
  'refresh-token': []
  'probe-upstream-billing': []
  'refresh-usage': []
}>()

const { t } = useI18n()
const quickProxyId = ref<number | null>(null)
const quickGroupId = ref<number | null>(null)
const proxyOptions = computed(() => [
  { value: 0, label: t('admin.accounts.noProxy') },
  ...props.proxies.map(proxy => ({ value: proxy.id, label: proxy.name }))
])
const groupOptions = computed(() => [
  { value: 0, label: t('admin.accounts.bulkActions.noGroup') },
  ...props.groups.map(group => ({ value: group.id, label: group.name }))
])

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
