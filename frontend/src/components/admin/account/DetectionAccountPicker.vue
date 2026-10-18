<template>
  <div class="space-y-2">
    <input v-model="search" type="search" :placeholder="t('admin.scheduledTests.searchAccounts')" class="input w-full" />
    <div v-if="!single" class="flex gap-3 text-xs">
      <button type="button" :disabled="disabled" @click="model = [...new Set([...model, ...visible.map(a => a.id)])]">{{ t('admin.scheduledTests.selectAllVisible') }}</button>
      <button type="button" :disabled="disabled" @click="model = []">{{ t('admin.scheduledTests.clearSelection') }}</button>
      <span>{{ model.length }}</span>
    </div>
    <div class="grid max-h-48 gap-2 overflow-y-auto rounded-lg border border-gray-200 p-2 sm:grid-cols-2 dark:border-dark-600">
      <label v-for="account in visible" :key="account.id" class="flex min-w-0 items-start gap-2 text-xs">
        <input :type="single ? 'radio' : 'checkbox'" :name="single ? 'detection-account' : undefined" :checked="model.includes(account.id)" :disabled="disabled" @change="toggle(account.id)" />
        <span class="break-words">{{ account.name }} <span class="text-gray-500">#{{ account.id }} · {{ account.platform }}</span></span>
      </label>
      <span v-for="id in missingIds" :key="id" class="text-xs">#{{ id }} <button type="button" :disabled="disabled" @click="toggle(id)">×</button></span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
const props = defineProps<{ accounts: Account[]; disabled?: boolean; single?: boolean }>()
const model = defineModel<number[]>({ required: true })
const { t } = useI18n()
const search = ref('')
const visible = computed(() => props.accounts.filter(a => `${a.name} ${a.id} ${a.platform}`.toLowerCase().includes(search.value.toLowerCase().trim())))
const missingIds = computed(() => model.value.filter(id => !props.accounts.some(a => a.id === id)))
const toggle = (id: number) => { model.value = props.single ? [id] : model.value.includes(id) ? model.value.filter(v => v !== id) : [...model.value, id] }
</script>
