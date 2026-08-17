<template>
  <div class="space-y-3 border-t border-gray-200 pt-3 dark:border-dark-600">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-1.5 text-xs font-medium text-gray-700 dark:text-gray-300">
        <Icon name="clock" size="sm" class="text-primary-500" :stroke-width="2" />
        {{ t('admin.scheduledTests.visualBuilder') }}
      </div>
      <button
        type="button"
        data-testid="cron-builder-help"
        class="inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-dark-600 dark:hover:text-gray-200"
        :aria-expanded="showHelp"
        @click="showHelp = !showHelp"
      >
        <Icon name="questionCircle" size="xs" :stroke-width="2" />
        {{ t('admin.scheduledTests.builderHelp') }}
      </button>
    </div>

    <div class="flex flex-wrap gap-1" role="group" :aria-label="t('admin.scheduledTests.visualBuilder')">
      <button
        v-for="item in modes"
        :key="item.value"
        type="button"
        :data-testid="`cron-mode-${item.value}`"
        :class="[
          'h-8 rounded-md px-2.5 text-xs font-medium transition-colors',
          mode === item.value
            ? 'bg-primary-500 text-white'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500'
        ]"
        @click="setMode(item.value)"
      >
        {{ item.label }}
      </button>
    </div>

    <div v-if="mode !== 'custom'" class="flex flex-wrap items-end gap-3">
      <label v-if="mode === 'interval'" class="min-w-32 text-xs text-gray-600 dark:text-gray-400">
        <span class="mb-1 block">{{ t('admin.scheduledTests.intervalMinutes') }}</span>
        <select v-model.number="intervalMinutes" data-testid="cron-interval" class="input h-9 py-1.5 text-sm">
          <option v-for="value in intervalOptions" :key="value" :value="value">
            {{ t('admin.scheduledTests.minuteCount', { count: value }) }}
          </option>
        </select>
      </label>

      <label v-if="mode === 'weekly'" class="min-w-32 text-xs text-gray-600 dark:text-gray-400">
        <span class="mb-1 block">{{ t('admin.scheduledTests.weekday') }}</span>
        <select v-model.number="weekday" data-testid="cron-weekday" class="input h-9 py-1.5 text-sm">
          <option v-for="item in weekdayOptions" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>
      </label>

      <label v-if="mode === 'daily' || mode === 'weekly'" class="min-w-24 text-xs text-gray-600 dark:text-gray-400">
        <span class="mb-1 block">{{ t('admin.scheduledTests.hour') }}</span>
        <select v-model.number="hour" data-testid="cron-hour" class="input h-9 py-1.5 text-sm">
          <option v-for="value in hourOptions" :key="value" :value="value">{{ pad(value) }}</option>
        </select>
      </label>

      <label v-if="mode === 'hourly' || mode === 'daily' || mode === 'weekly'" class="min-w-24 text-xs text-gray-600 dark:text-gray-400">
        <span class="mb-1 block">{{ t('admin.scheduledTests.minute') }}</span>
        <select v-model.number="minute" data-testid="cron-minute" class="input h-9 py-1.5 text-sm">
          <option v-for="value in minuteOptions" :key="value" :value="value">{{ pad(value) }}</option>
        </select>
      </label>
    </div>

    <div class="flex min-h-8 items-center gap-2 text-xs text-gray-600 dark:text-gray-400">
      <Icon name="calendar" size="sm" class="shrink-0 text-gray-400" />
      <span>{{ preview }}</span>
      <code v-if="mode !== 'custom'" class="ml-auto shrink-0 font-mono text-[11px] text-primary-600 dark:text-primary-400">
        {{ modelValue }}
      </code>
    </div>

    <div
      v-if="showHelp"
      class="border-l-2 border-primary-400 bg-gray-50 px-3 py-2 text-xs leading-5 text-gray-600 dark:bg-dark-700/50 dark:text-gray-300"
    >
      <p>{{ t('admin.scheduledTests.builderHelpBody') }}</p>
      <p class="mt-1 font-mono text-[11px] text-gray-500 dark:text-gray-400">
        {{ t('admin.scheduledTests.cronFieldOrder') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'

type ScheduleMode = 'interval' | 'hourly' | 'daily' | 'weekly' | 'custom'

const modelValue = defineModel<string>({ required: true })
const { t } = useI18n()

const mode = ref<ScheduleMode>('interval')
const intervalMinutes = ref(30)
const minute = ref(0)
const hour = ref(9)
const weekday = ref(1)
const showHelp = ref(false)

const intervalOptions = [5, 10, 15, 20, 30]
const minuteOptions = Array.from({ length: 60 }, (_, index) => index)
const hourOptions = Array.from({ length: 24 }, (_, index) => index)

const modes = computed(() => [
  { value: 'interval' as const, label: t('admin.scheduledTests.modeInterval') },
  { value: 'hourly' as const, label: t('admin.scheduledTests.modeHourly') },
  { value: 'daily' as const, label: t('admin.scheduledTests.modeDaily') },
  { value: 'weekly' as const, label: t('admin.scheduledTests.modeWeekly') },
  { value: 'custom' as const, label: t('admin.scheduledTests.modeCustom') }
])

const weekdayOptions = computed(() => [
  { value: 1, label: t('admin.scheduledTests.weekdays.monday') },
  { value: 2, label: t('admin.scheduledTests.weekdays.tuesday') },
  { value: 3, label: t('admin.scheduledTests.weekdays.wednesday') },
  { value: 4, label: t('admin.scheduledTests.weekdays.thursday') },
  { value: 5, label: t('admin.scheduledTests.weekdays.friday') },
  { value: 6, label: t('admin.scheduledTests.weekdays.saturday') },
  { value: 0, label: t('admin.scheduledTests.weekdays.sunday') }
])

const pad = (value: number) => String(value).padStart(2, '0')

const applyVisualSchedule = () => {
  switch (mode.value) {
    case 'interval':
      modelValue.value = `*/${intervalMinutes.value} * * * *`
      break
    case 'hourly':
      modelValue.value = `${minute.value} * * * *`
      break
    case 'daily':
      modelValue.value = `${minute.value} ${hour.value} * * *`
      break
    case 'weekly':
      modelValue.value = `${minute.value} ${hour.value} * * ${weekday.value}`
      break
  }
}

const setMode = (nextMode: ScheduleMode) => {
  mode.value = nextMode
  if (nextMode !== 'custom') applyVisualSchedule()
}

const parseExpression = (expression: string) => {
  const value = expression.trim()
  let match = value.match(/^\*\/(5|10|15|20|30) \* \* \* \*$/)
  if (match) {
    mode.value = 'interval'
    intervalMinutes.value = Number(match[1])
    return
  }
  match = value.match(/^([0-5]?\d) \* \* \* \*$/)
  if (match) {
    mode.value = 'hourly'
    minute.value = Number(match[1])
    return
  }
  match = value.match(/^([0-5]?\d) ([01]?\d|2[0-3]) \* \* \*$/)
  if (match) {
    mode.value = 'daily'
    minute.value = Number(match[1])
    hour.value = Number(match[2])
    return
  }
  match = value.match(/^([0-5]?\d) ([01]?\d|2[0-3]) \* \* ([0-6])$/)
  if (match) {
    mode.value = 'weekly'
    minute.value = Number(match[1])
    hour.value = Number(match[2])
    weekday.value = Number(match[3])
    return
  }
  mode.value = 'custom'
}

const preview = computed(() => {
  switch (mode.value) {
    case 'interval':
      return t('admin.scheduledTests.previewInterval', { count: intervalMinutes.value })
    case 'hourly':
      return t('admin.scheduledTests.previewHourly', { minute: pad(minute.value) })
    case 'daily':
      return t('admin.scheduledTests.previewDaily', { time: `${pad(hour.value)}:${pad(minute.value)}` })
    case 'weekly':
      return t('admin.scheduledTests.previewWeekly', {
        weekday: weekdayOptions.value.find((item) => item.value === weekday.value)?.label || '',
        time: `${pad(hour.value)}:${pad(minute.value)}`
      })
    default:
      return t('admin.scheduledTests.previewCustom')
  }
})

watch(modelValue, (value) => parseExpression(value || ''), { immediate: true })
watch([intervalMinutes, minute, hour, weekday], () => {
  if (mode.value !== 'custom') applyVisualSchedule()
})
</script>
