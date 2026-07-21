<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.superPriority')"
    width="normal"
    @close="handleClose"
  >
    <div v-if="loading" class="flex items-center justify-center py-12">
      <Icon name="refresh" size="md" class="animate-spin text-gray-400" />
    </div>

    <div v-else-if="!settings" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loadFailed') }}
    </div>

    <div v-else class="space-y-5">
      <!-- 当前模式状态 -->
      <div
        class="flex items-center justify-between rounded-lg border p-4"
        :class="
          settings.is_active
            ? 'border-fuchsia-200 bg-fuchsia-50 dark:border-fuchsia-800 dark:bg-fuchsia-900/20'
            : 'border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-700'
        "
      >
        <div class="flex items-center gap-3">
          <Icon
            name="sparkles"
            size="md"
            :class="settings.is_active ? 'text-fuchsia-600 dark:text-fuchsia-400' : 'text-gray-400'"
          />
          <div>
            <div class="font-semibold text-gray-900 dark:text-white">
              {{ settings.is_active ? t('admin.accounts.superPriorityActive') : t('admin.accounts.superPriorityInactive') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              <template v-if="settings.is_active && settings.activated_at">
                {{ t('admin.accounts.superPriorityActivatedAt') }}: {{ formatTime(settings.activated_at) }}
              </template>
              <template v-else-if="settings.demoted_at">
                {{ t('admin.accounts.superPriorityDemotedAt') }}: {{ formatTime(settings.demoted_at) }}
              </template>
              <template v-else>{{ t('admin.accounts.superPriorityNeverActivated') }}</template>
            </div>
          </div>
        </div>
        <button
          type="button"
          :disabled="toggling"
          class="rounded-lg px-3 py-1.5 text-sm font-medium transition-colors disabled:opacity-50"
          :class="
            settings.is_active
              ? 'bg-gray-200 text-gray-700 hover:bg-gray-300 dark:bg-dark-600 dark:text-gray-200 dark:hover:bg-dark-500'
              : 'bg-fuchsia-600 text-white hover:bg-fuchsia-700'
          "
          @click="toggleMode"
        >
          {{ settings.is_active ? t('common.deactivate') : t('common.activate') }}
        </button>
      </div>

      <!-- 运行参数 -->
      <div class="space-y-3">
        <div class="text-sm font-semibold text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.superPriorityRuntime') }}
        </div>

        <div class="grid grid-cols-2 gap-3">
          <label class="block">
            <span class="mb-1 block text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.superPriorityBaseStrategy') }}
            </span>
            <select
              v-model="form.base_strategy"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            >
              <option value="default">{{ t('admin.accounts.superPriorityBaseStrategyDefault') }}</option>
              <option value="lowest_cost">{{ t('admin.accounts.superPriorityBaseStrategyLowestCost') }}</option>
            </select>
            <span class="mt-1 block text-[11px] text-gray-400">
              {{ t('admin.accounts.superPriorityBaseStrategyHint') }}
            </span>
          </label>

          <label class="block">
            <span class="mb-1 block text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.superPriorityThreshold') }}
            </span>
            <input
              v-model.number="form.failure_threshold"
              type="number"
              min="1"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            />
            <span class="mt-1 block text-[11px] text-gray-400">
              {{ t('admin.accounts.superPriorityThresholdHint') }}
            </span>
          </label>

          <label class="block">
            <span class="mb-1 block text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.superPriorityInterval') }}
            </span>
            <input
              v-model="form.check_interval"
              type="text"
              class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
            />
            <span class="mt-1 block text-[11px] text-gray-400">@every 1m</span>
          </label>
        </div>

        <label class="block">
          <span class="mb-1 block text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.superPriorityTestModel') }}
          </span>
          <input
            v-model="form.test_model_id"
            type="text"
            :placeholder="t('admin.accounts.superPriorityTestModelPlaceholder')"
            class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-700 dark:text-white"
          />
        </label>

        <button
          type="button"
          :disabled="savingParams"
          class="rounded-lg bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 disabled:opacity-50 dark:bg-dark-600 dark:text-gray-200 dark:hover:bg-dark-500"
          @click="saveParams"
        >
          {{ savingParams ? t('common.saving') : t('common.save') }}
        </button>
      </div>

    </div>

    <template #footer>
      <button
        type="button"
        class="rounded-lg px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-600"
        @click="handleClose"
      >
        {{ t('common.close') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { SuperPrioritySettings, SuperPriorityRuntimeParams } from '@/api/admin/superPriority'
import { formatTime } from '@/utils/format'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'changed'): void
}>()

const loading = ref(false)
const toggling = ref(false)
const savingParams = ref(false)
const settings = ref<SuperPrioritySettings | null>(null)

const form = reactive<SuperPriorityRuntimeParams>({
  base_strategy: 'default',
  failure_threshold: 2,
  check_interval: '@every 1m',
  test_model_id: '',
  test_prompt: '',
})

const load = async () => {
  loading.value = true
  try {
    const data = await adminAPI.superPriority.get()
    settings.value = data
    form.base_strategy = data.base_strategy
    form.failure_threshold = data.failure_threshold
    form.check_interval = data.check_interval
    form.test_model_id = data.test_model_id
    form.test_prompt = data.test_prompt
  } catch (err) {
    console.error('Failed to load super priority settings:', err)
  } finally {
    loading.value = false
  }
}

const toggleMode = async () => {
  if (!settings.value) return
  toggling.value = true
  try {
    if (settings.value.is_active) {
      await adminAPI.superPriority.deactivate()
    } else {
      await adminAPI.superPriority.activate()
    }
    await load()
    emit('changed')
  } catch (err) {
    console.error('Failed to toggle super priority mode:', err)
  } finally {
    toggling.value = false
  }
}

const saveParams = async () => {
  savingParams.value = true
  try {
    await adminAPI.superPriority.update({
      base_strategy: form.base_strategy,
      failure_threshold: form.failure_threshold,
      check_interval: form.check_interval,
      test_model_id: form.test_model_id,
      test_prompt: form.test_prompt,
    })
    await load()
  } catch (err) {
    console.error('Failed to save super priority params:', err)
  } finally {
    savingParams.value = false
  }
}

const handleClose = () => {
  emit('close')
}

watch(
  () => props.show,
  (val) => {
    if (val) load()
  },
)
</script>
