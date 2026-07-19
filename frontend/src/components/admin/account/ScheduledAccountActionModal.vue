<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.scheduledAction.title')"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div v-if="account" class="border-b border-gray-200 pb-3 dark:border-dark-600">
        <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.scheduledAction.account') }}</div>
        <div class="mt-1 font-medium text-gray-900 dark:text-white">
          {{ account.name }}
          <span class="font-mono text-xs text-gray-500">#{{ account.id }}</span>
        </div>
      </div>

      <div v-if="loading" class="py-3 text-center text-sm text-gray-500">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="currentTask" class="border-l-4 border-amber-500 bg-amber-50 px-3 py-2 dark:bg-amber-950/30">
        <div class="text-xs font-medium text-amber-700 dark:text-amber-300">
          {{ t('admin.accounts.scheduledAction.currentTask') }}
        </div>
        <div class="mt-1 text-sm text-amber-900 dark:text-amber-100">
          {{ actionLabel(currentTask.action) }} · {{ formatDateTime(currentTask.execute_at) }}
        </div>
        <div v-if="currentTask.last_error" class="mt-1 text-xs text-red-600 dark:text-red-300">
          {{ t('admin.accounts.scheduledAction.lastError') }}: {{ currentTask.last_error }}
        </div>
        <button
          type="button"
          class="mt-2 text-sm font-medium text-amber-700 hover:text-amber-900 disabled:opacity-50 dark:text-amber-300 dark:hover:text-amber-100"
          :disabled="canceling"
          @click="cancelTask"
        >
          {{ canceling ? t('common.processing') : t('admin.accounts.scheduledAction.cancelTask') }}
        </button>
      </div>

      <div>
        <div class="input-label">{{ t('admin.accounts.scheduledAction.action') }}</div>
        <div class="mt-1 flex rounded-md border border-gray-200 p-1 dark:border-dark-600">
          <button
            type="button"
            class="min-w-0 flex-1 px-2 py-2 text-sm"
            :class="action === 'enable_and_recover' ? activeModeClass : inactiveModeClass"
            @click="action = 'enable_and_recover'"
          >
            {{ t('admin.accounts.scheduledAction.enableAndRecover') }}
          </button>
          <button
            type="button"
            class="min-w-0 flex-1 px-2 py-2 text-sm"
            :class="action === 'pause' ? activeModeClass : inactiveModeClass"
            @click="action = 'pause'"
          >
            {{ t('admin.accounts.scheduledAction.pause') }}
          </button>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <label class="block">
          <span class="input-label">{{ t('admin.accounts.scheduledAction.hours') }}</span>
          <input
            v-model.number="hours"
            data-testid="scheduled-action-hours"
            type="number"
            min="0"
            max="8760"
            step="1"
            class="input"
          />
        </label>
        <label class="block">
          <span class="input-label">{{ t('admin.accounts.scheduledAction.minutes') }}</span>
          <input
            v-model.number="minutes"
            data-testid="scheduled-action-minutes"
            type="number"
            min="0"
            max="59"
            step="1"
            class="input"
          />
        </label>
      </div>

      <div v-if="!validDelay" class="text-sm text-red-600 dark:text-red-400">
        {{ t('admin.accounts.scheduledAction.minimumDelay') }}
      </div>
      <div v-else class="text-sm text-gray-600 dark:text-gray-300">
        <div>{{ actionSummary }}</div>
        <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.scheduledAction.targetTime') }}: {{ targetTime }}
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="submitting" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button type="button" class="btn btn-primary" :disabled="!canSubmit" @click="save">
        {{ submitting ? t('common.processing') : t('admin.accounts.scheduledAction.save') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { ScheduledAccountAction, ScheduledAccountActionType } from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'
import type { Account } from '@/types'

const props = defineProps<{
  show: boolean
  account: Account | null
  initialAction: ScheduledAccountActionType
}>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const canceling = ref(false)
const action = ref<ScheduledAccountActionType>('pause')
const hours = ref(0)
const minutes = ref(30)
const currentTask = ref<ScheduledAccountAction | null>(null)
const clock = ref(Date.now())

const activeModeClass = 'bg-primary-600 text-white shadow-sm'
const inactiveModeClass = 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'

const normalizedHours = computed(() => Number.isInteger(hours.value) ? hours.value : -1)
const normalizedMinutes = computed(() => Number.isInteger(minutes.value) ? minutes.value : -1)
const totalMinutes = computed(() => normalizedHours.value * 60 + normalizedMinutes.value)
const validDelay = computed(() =>
  normalizedHours.value >= 0 &&
  normalizedHours.value <= 8760 &&
  normalizedMinutes.value >= 0 &&
  normalizedMinutes.value <= 59 &&
  totalMinutes.value >= 1 &&
  totalMinutes.value <= 365 * 24 * 60
)
const canSubmit = computed(() => Boolean(props.account && validDelay.value && !submitting.value))
const targetTime = computed(() => new Date(clock.value + totalMinutes.value * 60_000).toLocaleString())
const actionSummary = computed(() => t(
  action.value === 'enable_and_recover'
    ? 'admin.accounts.scheduledAction.enableSummary'
    : 'admin.accounts.scheduledAction.pauseSummary',
  { hours: normalizedHours.value, minutes: normalizedMinutes.value },
))

const actionLabel = (value: ScheduledAccountActionType) => t(
  value === 'enable_and_recover'
    ? 'admin.accounts.scheduledAction.enableAndRecover'
    : 'admin.accounts.scheduledAction.pause',
)

const formatDateTime = (value: string) => new Date(value).toLocaleString()

const load = async () => {
  if (!props.account) return
  action.value = props.initialAction
  hours.value = 0
  minutes.value = 30
  clock.value = Date.now()
  loading.value = true
  try {
    currentTask.value = await adminAPI.accounts.getScheduledAction(props.account.id)
  } catch (error: any) {
    currentTask.value = null
    appStore.showError(error?.message || t('admin.accounts.scheduledAction.loadFailed'))
  } finally {
    loading.value = false
  }
}

const save = async () => {
  if (!props.account || !canSubmit.value) return
  submitting.value = true
  try {
    currentTask.value = await adminAPI.accounts.scheduleAction(props.account.id, {
      action: action.value,
      hours: normalizedHours.value,
      minutes: normalizedMinutes.value,
    })
    appStore.showSuccess(t('admin.accounts.scheduledAction.saved'))
    emit('saved')
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.scheduledAction.saveFailed'))
  } finally {
    submitting.value = false
  }
}

const cancelTask = async () => {
  if (!props.account || !currentTask.value || canceling.value) return
  canceling.value = true
  try {
    await adminAPI.accounts.cancelScheduledAction(props.account.id)
    currentTask.value = null
    appStore.showSuccess(t('admin.accounts.scheduledAction.canceled'))
    emit('saved')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.scheduledAction.cancelFailed'))
  } finally {
    canceling.value = false
  }
}

watch(
  () => [props.show, props.account?.id, props.initialAction] as const,
  ([show]) => {
    if (show) void load()
  },
  { immediate: true },
)
</script>
