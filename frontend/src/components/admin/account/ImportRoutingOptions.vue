<template>
  <section class="space-y-3 border-t border-gray-200 pt-4 dark:border-dark-700">
    <h4 class="text-sm font-medium text-gray-900 dark:text-white">
      {{ t('admin.accounts.dataImportRoutingTitle') }}
    </h4>

    <div class="space-y-4">
      <div>
        <label class="flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
          <input
            v-model="applyProxySettings"
            data-test="import-apply-default-proxy"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
          />
          <span>{{ t('admin.accounts.dataImportApplyProxySettings') }}</span>
        </label>
        <div v-if="applyProxySettings" class="mt-2 pl-6">
          <label class="input-label">{{ t('admin.accounts.dataImportDefaultProxy') }}</label>
          <ProxySelector v-model="defaultProxyId" :proxies="proxies" />
        </div>
      </div>

      <div>
        <label class="flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
          <input
            v-model="applyGroupSettings"
            data-test="import-apply-default-groups"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
          />
          <span>{{ t('admin.accounts.dataImportApplyGroupSettings') }}</span>
        </label>
        <div v-if="applyGroupSettings" class="mt-2 pl-6">
          <label class="input-label">{{ t('admin.accounts.dataImportDefaultGroups') }}</label>
          <GroupSelector v-model="defaultGroupIds" :groups="groups" />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, Proxy } from '@/types'
import GroupSelector from '@/components/common/GroupSelector.vue'
import ProxySelector from '@/components/common/ProxySelector.vue'

interface ImportRoutingRequestOptions {
  apply_proxy_settings: boolean
  default_proxy_id?: number | null
  apply_group_settings: boolean
  default_group_ids?: number[]
}

const { t } = useI18n()
const appStore = useAppStore()
const proxies = ref<Proxy[]>([])
const groups = ref<AdminGroup[]>([])
const applyProxySettings = ref(true)
const applyGroupSettings = ref(true)
const defaultProxyId = ref<number | null>(null)
const defaultGroupIds = ref<number[]>([])

let loadPromise: Promise<void> | null = null

const loadCandidates = async () => {
  const [proxyResult, groupResult] = await Promise.allSettled([
    adminAPI.proxies.getAll(),
    adminAPI.groups.getAll()
  ])

  if (proxyResult.status === 'fulfilled') {
    proxies.value = proxyResult.value
    defaultProxyId.value = proxyResult.value.at(-1)?.id ?? null
  } else {
    applyProxySettings.value = false
  }
  if (groupResult.status === 'fulfilled') {
    groups.value = groupResult.value
    const defaultGroup = groupResult.value[0]
    defaultGroupIds.value = defaultGroup ? [defaultGroup.id] : []
  } else {
    applyGroupSettings.value = false
  }
  if (proxyResult.status === 'rejected' || groupResult.status === 'rejected') {
    appStore.showError(t('admin.accounts.dataImportRoutingLoadFailed'))
  }
}

onMounted(() => {
  loadPromise = loadCandidates()
})

const getRequestOptions = async (): Promise<ImportRoutingRequestOptions> => {
  await loadPromise
  return {
    apply_proxy_settings: applyProxySettings.value,
    ...(applyProxySettings.value ? { default_proxy_id: defaultProxyId.value } : {}),
    apply_group_settings: applyGroupSettings.value,
    ...(applyGroupSettings.value ? { default_group_ids: [...defaultGroupIds.value] } : {})
  }
}

defineExpose({ getRequestOptions })
</script>
