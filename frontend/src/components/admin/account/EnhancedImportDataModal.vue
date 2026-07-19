<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.enhancedImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="enhanced-import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <p class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.enhancedImportHint') }}
      </p>

      <div
        class="grid grid-cols-2 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-800"
        role="tablist"
      >
        <button
          type="button"
          role="tab"
          data-test="enhanced-import-mode-file"
          class="flex min-h-9 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors"
          :class="sourceMode === 'file'
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200'"
          :aria-selected="sourceMode === 'file'"
          @click="sourceMode = 'file'"
        >
          <Icon name="upload" size="sm" />
          {{ t('admin.accounts.enhancedImportFileMode') }}
        </button>
        <button
          type="button"
          role="tab"
          data-test="enhanced-import-mode-text"
          class="flex min-h-9 items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors"
          :class="sourceMode === 'text'
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-200'"
          :aria-selected="sourceMode === 'text'"
          @click="sourceMode = 'text'"
        >
          <Icon name="document" size="sm" />
          {{ t('admin.accounts.enhancedImportTextMode') }}
        </button>
      </div>

      <div v-if="sourceMode === 'file'">
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed px-4 py-3 transition-colors"
          :class="dragActive
            ? 'border-primary-400 bg-primary-50/70 dark:border-primary-500 dark:bg-primary-900/20'
            : 'border-gray-300 bg-gray-50 dark:border-dark-600 dark:bg-dark-800'"
          @dragenter.prevent="handleDragEnter"
          @dragover.prevent
          @dragleave.prevent="handleDragLeave"
          @drop.prevent="handleDrop"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200" :title="fileListTitle">
              {{ selectedFilesLabel || t('admin.accounts.dataImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">JSON (.json)</div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="openFilePicker">
            {{ t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json"
          multiple
          @change="handleFileChange"
        />
      </div>

      <div v-else class="space-y-3">
        <TextArea
          v-model="jsonText"
          :label="t('admin.accounts.enhancedImportTextLabel')"
          :placeholder="t('admin.accounts.enhancedImportTextPlaceholder')"
          :rows="12"
        />
        <div
          data-test="enhanced-import-usage-guide"
          class="rounded-lg border border-cyan-200 bg-cyan-50/70 px-3 py-2.5 text-xs text-cyan-900 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-100"
        >
          <div class="font-medium">{{ t('admin.accounts.enhancedImportUsageGuideTitle') }}</div>
          <div class="mt-1 leading-5">{{ t('admin.accounts.enhancedImportUsageGuide') }}</div>
        </div>
        <div
          v-if="jsonText.trim()"
          data-test="enhanced-import-extraction-summary"
          class="text-xs text-gray-500 dark:text-dark-400"
        >
          {{ extractionSummaryText }}
        </div>
      </div>

      <ImportRoutingOptions ref="routingOptionsRef" />

      <div
        v-if="result"
        class="space-y-2 rounded-lg border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>
        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div class="mt-2 max-h-48 overflow-auto rounded-md bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800">
            <div v-for="(item, index) in errorItems" :key="index" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} - {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="enhanced-import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import Icon from '@/components/icons/Icon.vue'
import ImportRoutingOptions from './ImportRoutingOptions.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminDataImportResult, AdminDataPayload } from '@/types'
import {
  EnhancedImportError,
  mergeEnhancedImportPayloads,
  parseEnhancedImportText,
  parseEnhancedImportSource
} from './enhancedImport'

interface Props {
  show: boolean
}

interface Emits {
  (event: 'close'): void
  (event: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()
const appStore = useAppStore()

const sourceMode = ref<'file' | 'text'>('file')
const importing = ref(false)
const files = ref<File[]>([])
const jsonText = ref('')
const dragDepth = ref(0)
const hasCreatedData = ref(false)
const result = ref<AdminDataImportResult | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const routingOptionsRef = ref<InstanceType<typeof ImportRoutingOptions> | null>(null)

const dragActive = computed(() => dragDepth.value > 0)
const fileListTitle = computed(() => files.value.map(file => file.name).join(', '))
const selectedFilesLabel = computed(() => {
  if (files.value.length === 0) return ''
  if (files.value.length === 1) return files.value[0]?.name || ''
  return t('admin.accounts.selectedCount', { count: files.value.length })
})
const errorItems = computed(() => result.value?.errors || [])
const extractionSummaryText = computed(() => {
  try {
    const payloads = parseEnhancedImportText(jsonText.value, 'pasted JSON')
    return t('admin.accounts.enhancedImportExtractionSummary', {
      sources: payloads.length,
      accounts: payloads.reduce((sum, payload) => sum + payload.accounts.length, 0)
    })
  } catch {
    return t('admin.accounts.enhancedImportExtractionPending')
  }
})

watch(
  () => props.show,
  open => {
    if (!open) return
    sourceMode.value = 'file'
    files.value = []
    jsonText.value = ''
    dragDepth.value = 0
    hasCreatedData.value = false
    result.value = null
    if (fileInput.value) fileInput.value.value = ''
  }
)

const openFilePicker = () => fileInput.value?.click()

const isJsonFile = (file: File) =>
  file.name.toLowerCase().endsWith('.json') || file.type === 'application/json'

const setSelectedFiles = (sourceFiles: FileList | File[] | null | undefined) => {
  if (importing.value) return
  const incoming = Array.from(sourceFiles || [])
  const picked = incoming.filter(isJsonFile)
  if (!picked.length) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }
  files.value = picked
  result.value = null
}

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  setSelectedFiles(target.files)
  target.value = ''
}

const handleDragEnter = () => {
  if (!importing.value) dragDepth.value += 1
}

const handleDragLeave = () => {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
}

const handleDrop = (event: DragEvent) => {
  dragDepth.value = 0
  if (!importing.value) setSelectedFiles(event.dataTransfer?.files)
}

const readFileAsText = async (file: File): Promise<string> => {
  if (typeof file.text === 'function') return file.text()
  if (typeof file.arrayBuffer === 'function') {
    return new TextDecoder().decode(await file.arrayBuffer())
  }
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(file)
  })
}

const handleClose = () => {
  if (importing.value) return
  if (hasCreatedData.value) {
    hasCreatedData.value = false
    emit('imported')
  }
  emit('close')
}

const showParseError = (error: EnhancedImportError) => {
  const key = {
    invalid_json: 'admin.accounts.enhancedImportInvalidJson',
    unsupported_format: 'admin.accounts.enhancedImportUnsupportedFormat',
    unsupported_provider: 'admin.accounts.enhancedImportUnsupportedProvider',
    missing_credentials: 'admin.accounts.enhancedImportMissingCredentials'
  }[error.code]
  appStore.showError(t(key, { source: error.sourceName, provider: error.provider || '-' }))
}

const readPayloads = async (): Promise<AdminDataPayload[] | null> => {
  if (sourceMode.value === 'text') {
    if (!jsonText.value.trim()) {
      appStore.showError(t('admin.accounts.enhancedImportEnterText'))
      return null
    }
    return parseEnhancedImportText(jsonText.value, 'pasted JSON')
  }
  if (files.value.length === 0) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return null
  }

  const payloads: AdminDataPayload[] = []
  for (const file of files.value) {
    payloads.push(parseEnhancedImportSource(await readFileAsText(file), file.name))
  }
  return payloads
}

const handleImport = async () => {
  if (importing.value) return
  importing.value = true
  try {
    const payloads = await readPayloads()
    if (!payloads) return
    const routingOptions = await routingOptionsRef.value?.getRequestOptions()

    const response = await adminAPI.accounts.importData({
      data: mergeEnhancedImportPayloads(payloads),
      ...routingOptions,
      skip_default_group_bind: true
    })
    result.value = response

    const messageParams: Record<string, unknown> = {
      account_created: response.account_created,
      account_failed: response.account_failed,
      proxy_created: response.proxy_created,
      proxy_reused: response.proxy_reused,
      proxy_failed: response.proxy_failed
    }
    if (response.account_failed > 0 || response.proxy_failed > 0) {
      hasCreatedData.value = response.account_created > 0 || response.proxy_created > 0
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', messageParams))
      return
    }

    appStore.showSuccess(t('admin.accounts.dataImportSuccess', messageParams))
    emit('imported')
  } catch (error: unknown) {
    if (error instanceof EnhancedImportError) {
      showParseError(error)
    } else {
      appStore.showError(error instanceof Error ? error.message : t('admin.accounts.dataImportFailed'))
    }
  } finally {
    importing.value = false
  }
}
</script>
