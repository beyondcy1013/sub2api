<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.stickySessions.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="border-l-4 border-amber-500 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">
        {{ t('admin.accounts.stickySessions.userLimitWarning') }}
      </div>

      <div v-if="account" class="flex items-center justify-between border-b border-gray-200 pb-3 dark:border-dark-600">
        <div>
          <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.accounts.stickySessions.target') }}</div>
          <div class="font-medium text-gray-900 dark:text-white">{{ account.name }} <span class="font-mono text-xs text-gray-500">#{{ account.id }}</span></div>
        </div>
        <div class="text-right text-xs text-gray-500 dark:text-gray-400">
          {{ account.current_concurrency ?? 0 }} / {{ account.concurrency }}
        </div>
      </div>

      <div v-if="loading" class="py-8 text-center text-sm text-gray-500">
        {{ t('common.loading') }}
      </div>
      <div v-else-if="groups.length === 0" class="py-8 text-center text-sm text-gray-500">
        {{ t('admin.accounts.stickySessions.empty') }}
      </div>
      <template v-else>
        <div class="grid gap-4 md:grid-cols-3">
          <label class="block">
            <span class="input-label">{{ t('admin.accounts.stickySessions.group') }}</span>
            <select v-model.number="selectedGroupID" class="input" @change="resetSource">
              <option v-for="group in groups" :key="group.group_id" :value="group.group_id">
                {{ group.group_name }} ({{ group.total }})
              </option>
            </select>
          </label>

          <label class="block">
            <span class="input-label">{{ t('admin.accounts.stickySessions.activeWindow') }}</span>
            <select
              v-model.number="activeWithinSeconds"
              data-testid="sticky-active-window"
              class="input"
              @change="resetSource"
            >
              <option v-for="option in activityWindowOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </label>

          <label class="block">
            <span class="input-label">{{ t('admin.accounts.stickySessions.source') }}</span>
            <select v-model.number="selectedSourceID" class="input" @change="resetMoveCount">
              <option v-for="source in availableSources" :key="source.account_id" :value="source.account_id">
                {{ source.account_name }} ({{ recentCount(source) }} / {{ source.count }})
              </option>
            </select>
          </label>
        </div>

        <div v-if="availableSources.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.stickySessions.noRecent') }}
        </div>

        <div v-if="selectedSource" class="grid grid-cols-3 gap-3 border-y border-gray-200 py-3 text-sm dark:border-dark-600">
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.accounts.stickySessions.recentBound') }}</div>
            <div class="font-semibold text-gray-900 dark:text-white">{{ selectedRecentCount }} / {{ selectedSource.count }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.accounts.stickySessions.sourceLoad') }}</div>
            <div class="font-semibold text-gray-900 dark:text-white">{{ selectedSource.current_concurrency }} / {{ selectedSource.concurrency }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500">{{ t('admin.accounts.stickySessions.targetHeadroom') }}</div>
            <div class="font-semibold text-gray-900 dark:text-white">{{ targetHeadroom }}</div>
          </div>
        </div>

        <label class="block">
          <span class="input-label">{{ t('admin.accounts.stickySessions.count') }}</span>
          <input
            v-model.number="moveCount"
            type="number"
            min="1"
            :max="Math.min(100, selectedRecentCount || 1)"
            class="input"
          />
          <span class="input-hint">{{ t('admin.accounts.stickySessions.countHint') }}</span>
        </label>

        <div v-if="recentCandidates.length" class="rounded-lg border border-gray-200 dark:border-dark-600">
          <div class="border-b border-gray-200 px-3 py-2 text-xs font-medium text-gray-600 dark:border-dark-600 dark:text-gray-300">
            {{ t('admin.accounts.stickySessions.recentSessions') }}
          </div>
          <div class="grid max-h-36 grid-cols-2 gap-x-4 gap-y-1 overflow-y-auto px-3 py-2 text-xs sm:grid-cols-3">
            <div v-for="(session, index) in recentCandidates" :key="`${session.session_suffix}-${index}`" class="flex items-center justify-between gap-2">
              <span class="font-mono text-gray-700 dark:text-gray-200">…{{ session.session_suffix }}</span>
              <span class="whitespace-nowrap text-gray-400">{{ formatActiveAgo(session.active_ago_seconds) }}</span>
            </div>
          </div>
        </div>

        <div v-if="selectedGroup?.protected_response_bindings" class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.stickySessions.protected', { count: selectedGroup.protected_response_bindings }) }}
        </div>
      </template>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="submitting" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button type="button" class="btn btn-primary" :disabled="!canSubmit || submitting" @click="submit">
        {{ submitting ? t('common.processing') : t('admin.accounts.stickySessions.submit') }}
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
import type { Account } from '@/types'
import type { StickySessionGroupSummary, StickySessionSource } from '@/api/admin/accounts'

const props = defineProps<{ show: boolean; account: Account | null }>()
const emit = defineEmits<{ close: []; reassigned: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const groups = ref<StickySessionGroupSummary[]>([])
const selectedGroupID = ref<number | null>(null)
const selectedSourceID = ref<number | null>(null)
const moveCount = ref(1)
const activeWithinSeconds = ref(300)

const activityWindowOptions = computed(() => [
  { value: 60, label: t('admin.accounts.stickySessions.window1m') },
  { value: 300, label: t('admin.accounts.stickySessions.window5m') },
  { value: 900, label: t('admin.accounts.stickySessions.window15m') },
  { value: 3600, label: t('admin.accounts.stickySessions.window60m') },
])

const selectedGroup = computed(() => groups.value.find(group => group.group_id === selectedGroupID.value) ?? null)
const recentCount = (source: StickySessionSource) => source.recent_counts?.[String(activeWithinSeconds.value)] ?? 0
const availableSources = computed(() =>
  (selectedGroup.value?.sources ?? []).filter(source => source.account_id !== props.account?.id && recentCount(source) > 0)
)
const selectedSource = computed<StickySessionSource | null>(() =>
  availableSources.value.find(source => source.account_id === selectedSourceID.value) ?? null
)
const targetHeadroom = computed(() => Math.max(0, (props.account?.concurrency ?? 0) - (props.account?.current_concurrency ?? 0)))
const selectedRecentCount = computed(() => selectedSource.value ? recentCount(selectedSource.value) : 0)
const recentCandidates = computed(() =>
  (selectedSource.value?.recent_sessions ?? [])
    .filter(session => session.active_ago_seconds <= activeWithinSeconds.value)
    .slice(0, 100)
)
const canSubmit = computed(() => {
  const source = selectedSource.value
  return Boolean(source && moveCount.value >= 1 && moveCount.value <= Math.min(100, selectedRecentCount.value))
})

const formatActiveAgo = (seconds: number) => {
  if (seconds < 5) return t('admin.accounts.stickySessions.justNow')
  if (seconds < 60) return t('admin.accounts.stickySessions.secondsAgo', { count: seconds })
  return t('admin.accounts.stickySessions.minutesAgo', { count: Math.max(1, Math.floor(seconds / 60)) })
}

const resetMoveCount = () => {
  const sourceCount = selectedRecentCount.value || 1
  moveCount.value = Math.max(1, Math.min(100, sourceCount, Math.max(1, targetHeadroom.value)))
}

const resetSource = () => {
  selectedSourceID.value = availableSources.value[0]?.account_id ?? null
  resetMoveCount()
}

const load = async () => {
  if (!props.account) return
  loading.value = true
  try {
    const result = await adminAPI.accounts.getStickySessionSummary(props.account.id)
    groups.value = (result.groups ?? []).filter(group =>
      group.sources.some(source => source.account_id !== props.account?.id && source.count > 0)
    )
    selectedGroupID.value = groups.value[0]?.group_id ?? null
    resetSource()
  } catch (error: any) {
    groups.value = []
    appStore.showError(error?.message || t('admin.accounts.stickySessions.loadFailed'))
  } finally {
    loading.value = false
  }
}

const submit = async () => {
  if (!props.account || !selectedGroup.value || !selectedSource.value || !canSubmit.value) return
  if (!window.confirm(t('admin.accounts.stickySessions.confirm', { count: moveCount.value, target: props.account.name }))) return
  submitting.value = true
  try {
    const result = await adminAPI.accounts.reassignStickySessions(props.account.id, {
      group_id: selectedGroup.value.group_id,
      source_account_id: selectedSource.value.account_id,
      count: moveCount.value,
      active_within_seconds: activeWithinSeconds.value,
    })
    appStore.showSuccess(t('admin.accounts.stickySessions.success', { count: result.moved }))
    emit('reassigned')
    await load()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.stickySessions.failed'))
  } finally {
    submitting.value = false
  }
}

watch(
  () => [props.show, props.account?.id] as const,
  ([show]) => {
    if (show) load()
  },
  { immediate: true }
)
</script>
