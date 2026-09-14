<template>
  <BaseDialog
    :show="show"
    :title="accounts.length > 1 ? t('admin.accounts.pelicanBatchTitle') : t('admin.accounts.pelicanAction')"
    width="extra-wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Description & Config Bar -->
      <div class="rounded-xl border border-gray-200 bg-gray-50/50 p-4 dark:border-dark-700 dark:bg-dark-800/40">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="max-w-2xl">
            <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ t('admin.accounts.pelicanBatchDesc') }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.pelicanAccountsCount', { selected: accounts.length, total: accounts.length }) }}
            </div>
          </div>
          <div class="flex items-center gap-3">
            <div class="flex items-center gap-2">
              <label class="text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.accounts.pelicanConcurrency') }}:
              </label>
              <select
                v-model.number="concurrency"
                :disabled="isRunning"
                class="rounded-lg border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
              >
                <option :value="1">1</option>
                <option :value="2">2</option>
                <option :value="3">3</option>
                <option :value="5">5</option>
                <option :value="10">10</option>
              </select>
            </div>
            <button
              v-if="!isRunning"
              @click="startAllTests"
              :disabled="accounts.length === 0"
              class="flex items-center gap-1.5 rounded-lg bg-primary-600 px-4 py-1.5 text-xs font-medium text-white shadow-sm transition hover:bg-primary-500 disabled:opacity-50"
            >
              <Icon name="play" size="sm" />
              <span>{{ t('admin.accounts.pelicanStartBatch') }}</span>
            </button>
            <button
              v-else
              @click="stopAllTests"
              class="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-1.5 text-xs font-medium text-white shadow-sm transition hover:bg-red-500"
            >
              <Icon name="x" size="sm" />
              <span>{{ t('admin.accounts.pelicanStopBatch') }}</span>
            </button>
          </div>
        </div>

        <!-- Customizable Prompt toggle / view -->
        <div class="mt-3">
          <button
            type="button"
            @click="showPromptEdit = !showPromptEdit"
            class="flex items-center gap-1 text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
          >
            <Icon :name="showPromptEdit ? 'chevronUp' : 'chevronDown'" size="xs" />
            <span>{{ t('admin.accounts.pelicanPromptLabel') }}</span>
          </button>
          <div v-if="showPromptEdit" class="mt-2">
            <TextArea
              v-model="customPrompt"
              :disabled="isRunning"
              rows="2"
              class="text-xs"
              :placeholder="t('admin.accounts.pelicanPromptPlaceholder')"
            />
          </div>
        </div>
      </div>

      <!-- Overall progress bar when testing -->
      <div v-if="isRunning || completedCount > 0" class="flex items-center justify-between text-xs text-gray-500">
        <div class="flex items-center gap-3">
          <span>完成: {{ completedCount }} / {{ accounts.length }}</span>
          <span class="text-green-600 font-medium">未降智: {{ normalCount }}</span>
          <span class="text-amber-600 font-medium">疑似降智: {{ downgradedCount }}</span>
          <span v-if="failedCount > 0" class="text-red-600 font-medium">失败: {{ failedCount }}</span>
        </div>
        <div class="h-2 w-48 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
          <div
            class="h-full bg-primary-600 transition-all duration-300"
            :style="{ width: `${accounts.length ? (completedCount / accounts.length) * 100 : 0}%` }"
          />
        </div>
      </div>

      <!-- Results Grid -->
      <div class="grid max-h-[60vh] grid-cols-1 gap-4 overflow-y-auto p-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="item in accountStates"
          :key="item.account.id"
          class="flex flex-col rounded-xl border bg-white p-3.5 shadow-sm transition-all dark:bg-dark-800"
          :class="[
            item.status === 'success' ? 'border-green-300 dark:border-green-800/60' :
            item.status === 'downgraded' ? 'border-amber-300 bg-amber-50/20 dark:border-amber-800/60 dark:bg-amber-950/10' :
            item.status === 'failed' ? 'border-red-300 bg-red-50/20 dark:border-red-800/60 dark:bg-red-950/10' :
            item.status === 'running' ? 'border-primary-300 ring-1 ring-primary-500/20 dark:border-primary-800/60' :
            'border-gray-200 dark:border-dark-700'
          ]"
        >
          <!-- Card Header -->
          <div class="flex items-start justify-between gap-2 border-b border-gray-100 pb-2 dark:border-dark-700">
            <div class="min-w-0">
              <div class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100" :title="item.account.name">
                {{ item.account.name }}
              </div>
              <div class="flex items-center gap-1.5 text-[11px] text-gray-400">
                <span class="uppercase">{{ item.account.type }}</span>
                <span>•</span>
                <span>ID: {{ item.account.id }}</span>
                <span v-if="item.responseModel" class="rounded bg-gray-100 px-1 text-[10px] text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ item.responseModel }}
                </span>
              </div>
            </div>

            <!-- Status Badge -->
            <div class="shrink-0">
              <span
                v-if="item.status === 'idle'"
                class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-400"
              >
                {{ t('admin.accounts.pelicanStatusQueued') }}
              </span>
              <span
                v-else-if="item.status === 'running'"
                class="inline-flex items-center gap-1 rounded-full bg-blue-100 px-2 py-0.5 text-[11px] font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
              >
                <Icon name="refresh" size="xs" class="animate-spin" />
                {{ t('admin.accounts.pelicanStatusRunning') }}
              </span>
              <span
                v-else-if="item.status === 'success'"
                class="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-[11px] font-medium text-green-700 dark:bg-green-900/30 dark:text-green-300"
              >
                <Icon name="checkCircle" size="xs" />
                {{ t('admin.accounts.pelicanStatusSuccess') }}
              </span>
              <span
                v-else-if="item.status === 'downgraded'"
                class="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                :title="item.reason"
              >
                <Icon name="exclamationTriangle" size="xs" />
                {{ t('admin.accounts.pelicanStatusDowngraded') }}
              </span>
              <span
                v-else-if="item.status === 'failed'"
                class="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-[11px] font-medium text-red-700 dark:bg-red-900/30 dark:text-red-300"
                :title="item.error"
              >
                <Icon name="xCircle" size="xs" />
                {{ t('admin.accounts.pelicanStatusFailed') }}
              </span>
            </div>
          </div>

          <!-- Card Content / Preview -->
          <div class="my-2 flex flex-1 flex-col justify-center">
            <!-- Animation iframe preview -->
            <div v-if="item.result?.has_html && item.activeTab === 'preview'" class="relative overflow-hidden rounded-lg border border-gray-200 bg-white shadow-inner dark:border-dark-600">
              <iframe
                :srcdoc="pelicanPreviewDocument(item.result.html || '')"
                sandbox="allow-scripts"
                class="h-44 w-full border-0"
                loading="lazy"
                title="Animation Preview"
              />
            </div>

            <!-- HTML Source view -->
            <div v-else-if="item.result?.has_html && item.activeTab === 'source'" class="relative">
              <pre class="h-44 overflow-y-auto rounded-lg bg-gray-900 p-2 font-mono text-[10px] text-gray-200"><code>{{ item.result.html }}</code></pre>
            </div>

            <!-- Streaming / Output preview while running -->
            <div v-else-if="item.status === 'running'" class="flex h-44 flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 p-3 text-center text-xs text-gray-400 dark:border-dark-700">
              <Icon name="refresh" size="lg" class="mb-2 animate-spin text-primary-500" />
              <div class="line-clamp-3 font-mono text-[11px] text-gray-500 dark:text-gray-400">
                {{ item.streamingContent || t('admin.accounts.connectingToApi') }}
              </div>
            </div>

            <!-- Downgraded or failed message without HTML -->
            <div v-else-if="item.status === 'downgraded' || item.status === 'failed'" class="flex h-44 flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 p-3 text-center text-xs text-gray-500 dark:border-dark-700">
              <Icon :name="item.status === 'downgraded' ? 'exclamationTriangle' : 'xCircle'" size="md" :class="item.status === 'downgraded' ? 'text-amber-500' : 'text-red-500'" class="mb-1" />
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ item.reason || item.error || t('admin.accounts.pelicanNoHtml') }}</span>
              <p v-if="item.streamingContent" class="mt-1 line-clamp-3 font-mono text-[10px] text-gray-400">
                {{ item.streamingContent }}
              </p>
            </div>

            <!-- Idle placeholder -->
            <div v-else class="flex h-44 items-center justify-center rounded-lg border border-dashed border-gray-200 text-xs text-gray-400 dark:border-dark-700">
              <span>{{ t('admin.accounts.pelicanStatusQueued') }}</span>
            </div>
          </div>

          <!-- Card Footer Actions -->
          <div class="mt-auto flex items-center justify-between border-t border-gray-100 pt-2 text-xs dark:border-dark-700">
            <div class="flex items-center gap-1">
              <template v-if="item.result?.has_html">
                <button
                  type="button"
                  @click="item.activeTab = item.activeTab === 'preview' ? 'source' : 'preview'"
                  class="rounded px-2 py-0.5 text-[11px] font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                >
                  {{ item.activeTab === 'preview' ? t('admin.accounts.pelicanSourceTab') : t('admin.accounts.pelicanPreviewTab') }}
                </button>
                <button
                  type="button"
                  @click="copyToClipboard(item.result.html || '', t('admin.accounts.outputCopied'))"
                  class="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                  :title="t('admin.accounts.copyOutput')"
                >
                  <Icon name="copy" size="xs" />
                </button>
              </template>
              <span v-if="item.elapsedMs" class="ml-1 text-[10px] text-gray-400">
                {{ (item.elapsedMs / 1000).toFixed(1) }}s
              </span>
            </div>

            <button
              v-if="item.status !== 'running'"
              type="button"
              @click="runSingleTest(item)"
              class="flex items-center gap-1 rounded px-2 py-0.5 text-[11px] font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-950/30"
            >
              <Icon name="refresh" size="xs" />
              <span>{{ t('admin.accounts.pelicanRetry') }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-3">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { buildApiUrl } from '@/api/client'
import { ADMIN_UI_REQUEST_HEADER } from '@/api/adminUIRequest'
import { pelicanPreviewDocument } from '@/utils/pelicanPreviewDocument'
import type { Account } from '@/types'

const props = defineProps<{
  show: boolean
  accounts: Account[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

interface PelicanAccountState {
  account: Account
  status: 'idle' | 'running' | 'success' | 'downgraded' | 'failed'
  streamingContent: string
  responseModel?: string
  reason?: string
  error?: string
  elapsedMs?: number
  activeTab: 'preview' | 'source'
  result?: {
    has_html: boolean
    html?: string
    downgraded: boolean
    reason: string
    response_model?: string
  }
}

const concurrency = ref(3)
const isRunning = ref(false)
const showPromptEdit = ref(false)
const customPrompt = ref(t('admin.accounts.pelicanPromptDefault'))
const accountStates = ref<PelicanAccountState[]>([])

let activeControllers: AbortController[] = []
let globalAbort: AbortController | null = null

const completedCount = computed(() =>
  accountStates.value.filter(s => s.status === 'success' || s.status === 'downgraded' || s.status === 'failed').length
)
const normalCount = computed(() =>
  accountStates.value.filter(s => s.status === 'success').length
)
const downgradedCount = computed(() =>
  accountStates.value.filter(s => s.status === 'downgraded').length
)
const failedCount = computed(() =>
  accountStates.value.filter(s => s.status === 'failed').length
)

watch(
  () => [props.show, props.accounts] as const,
  ([show, accounts]) => {
    if (show && accounts) {
      customPrompt.value = t('admin.accounts.pelicanPromptDefault')
      accountStates.value = accounts.map(account => ({
        account,
        status: 'idle',
        streamingContent: '',
        activeTab: 'preview'
      }))
    } else {
      stopAllTests()
    }
  },
  { immediate: true }
)

const handleClose = () => {
  stopAllTests()
  emit('close')
}

const stopAllTests = () => {
  isRunning.value = false
  if (globalAbort) {
    globalAbort.abort()
    globalAbort = null
  }
  for (const ac of activeControllers) {
    ac.abort()
  }
  activeControllers = []
  for (const item of accountStates.value) {
    if (item.status === 'running') {
      item.status = 'idle'
    }
  }
}

async function testSingleAccount(item: PelicanAccountState, signal: AbortSignal) {
  item.status = 'running'
  item.streamingContent = ''
  item.error = undefined
  item.reason = undefined
  item.result = undefined
  const startAt = Date.now()

  try {
    const requestBody = {
      model_id: '',
      prompt: customPrompt.value.trim() || t('admin.accounts.pelicanPromptDefault'),
      mode: 'pelican'
    }
    const url = buildApiUrl(`/admin/accounts/${item.account.id}/test`)
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
        'Content-Type': 'application/json',
        [ADMIN_UI_REQUEST_HEADER]: '1'
      },
      body: JSON.stringify(requestBody),
      signal
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const jsonStr = line.slice(6).trim()
          if (jsonStr) {
            try {
              const event = JSON.parse(jsonStr)
              if (event.type === 'test_start' && event.model) {
                item.responseModel = event.model
              } else if (event.type === 'content' && event.text) {
                item.streamingContent += event.text
              } else if (event.type === 'pelican_result' && event.data) {
                item.result = event.data
                item.responseModel = event.data.response_model || item.responseModel
                item.reason = event.data.reason
                if (event.data.downgraded) {
                  item.status = 'downgraded'
                } else {
                  item.status = 'success'
                }
              } else if (event.type === 'error') {
                item.error = event.error || 'Test failed'
                if (item.status !== 'downgraded' && item.status !== 'success') {
                  item.status = 'failed'
                }
              }
            } catch (e) {
              console.error('Failed to parse SSE event in pelican batch test:', e)
            }
          }
        }
      }
    }

    if (item.status === 'running') {
      if (item.result) {
        item.status = item.result.downgraded ? 'downgraded' : 'success'
      } else {
        item.status = 'failed'
        item.error = 'No test result returned'
      }
    }
  } catch (err: unknown) {
    if (signal.aborted) {
      item.status = 'idle'
      return
    }
    item.status = 'failed'
    item.error = err instanceof Error ? err.message : 'Unknown error'
  } finally {
    item.elapsedMs = Date.now() - startAt
  }
}

async function runSingleTest(item: PelicanAccountState) {
  const ac = new AbortController()
  activeControllers.push(ac)
  try {
    await testSingleAccount(item, ac.signal)
  } finally {
    activeControllers = activeControllers.filter(c => c !== ac)
  }
}

async function startAllTests() {
  if (isRunning.value) return
  isRunning.value = true
  globalAbort = new AbortController()
  const signal = globalAbort.signal

  const queue = [...accountStates.value]
  const limit = Math.max(1, Math.min(10, concurrency.value || 3))

  const workers = Array.from({ length: limit }, async () => {
    while (queue.length > 0 && !signal.aborted) {
      const item = queue.shift()
      if (!item) break
      await testSingleAccount(item, signal)
    }
  })

  try {
    await Promise.all(workers)
  } finally {
    isRunning.value = false
    globalAbort = null
  }
}

onUnmounted(() => {
  stopAllTests()
})
</script>
