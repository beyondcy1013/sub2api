<template>
  <span
    class="text-xs font-medium cursor-default"
    :class="!settings ? 'text-gray-400 dark:text-dark-500' : enabled ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-dark-400'"
    :title="t('admin.accounts.detection.modifyInCenterHint')"
  >
    {{ !settings ? '-' : enabled ? t('admin.accounts.detection.enabled') : t('admin.accounts.detection.disabled') }}
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { loadQualitySettings } from './api'
import { normalizeRequestQualitySettings } from '@/utils/accountDetection'
import { accountQualityEnabled } from './settings'
import type { Account, RequestQualityProbeSettings } from '@/types'

const props = defineProps<{ account: Account; refreshToken?: number }>()
const { t } = useI18n()
const settings = ref<RequestQualityProbeSettings | null>(null)
const enabled = computed(() => !!settings.value && accountQualityEnabled(settings.value, props.account))

let generation = 0
watch(() => props.refreshToken, async () => {
  const current = ++generation
  try {
    const value = await loadQualitySettings()
    if (current === generation) settings.value = normalizeRequestQualitySettings(value)
  } catch {
    if (current === generation) settings.value = null
  }
}, { immediate: true })
</script>
