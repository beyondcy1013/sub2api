<template>
  <div v-if="entries.length" class="space-y-2">
    <p>{{ t('admin.accounts.detection.accountOverrides') }}</p>
    <p class="text-xs text-gray-500">{{ t('admin.accounts.detection.accountOverridesHint') }}</p>
    <div v-for="[id, enabled] in entries" :key="id" class="flex flex-wrap items-center gap-2 text-xs">
      <span>{{ accounts.find(account => account.id === Number(id))?.name }} #{{ id }}</span>
      <button type="button" class="btn btn-secondary" role="switch" :aria-checked="enabled" @click="model = { ...model, [id]: !enabled }">{{ t(enabled ? 'common.enabled' : 'common.disabled') }}</button>
      <button type="button" class="text-primary-600" @click="reset(id)">{{ t('admin.accounts.detection.followScope') }}</button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
defineProps<{ accounts: Account[] }>()
const model = defineModel<Record<string, boolean> | undefined>()
const entries = computed(() => Object.entries(model.value ?? {}))
const { t } = useI18n()
function reset(id: string) {
  const next = { ...model.value }
  delete next[id]
  model.value = next
}
</script>
