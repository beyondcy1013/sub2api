<template>
  <BaseDialog
    :show="show"
    :title="accountStates.length > 1 ? t('admin.accounts.pelicanBatchTitle') : t('admin.accounts.pelicanAction')"
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
            <div class="mt-1 flex flex-wrap items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
              <span>
                {{ t('admin.accounts.pelicanAccountsCount', { selected: accountStates.length, total: candidateAccounts.length || accountStates.length }) }}
              </span>
              <button
                type="button"
                @click="showAccountPicker = !showAccountPicker"
                class="inline-flex items-center gap-1 font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
              >
                <Icon :name="showAccountPicker ? 'chevronUp' : 'chevronDown'" size="xs" />
                <span>{{ t('admin.accounts.pelicanSelectAccounts') }} ({{ selectedAccountIds.size }})</span>
              </button>
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
              :disabled="accountStates.length === 0"
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

        <!-- Account Picker Panel -->
        <div v-if="showAccountPicker" class="mt-3 rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800">
          <div class="flex flex-wrap items-center justify-between gap-2 pb-2 border-b border-gray-100 dark:border-dark-700">
            <div class="relative flex-1 min-w-[200px] max-w-xs">
              <input
                v-model="accountSearchQuery"
                type="search"
                :placeholder="t('admin.accounts.pelicanSearchAccount')"
                class="w-full rounded-md border border-gray-300 bg-gray-50 py-1 pl-7 pr-2 text-xs text-gray-800 focus:border-primary-500 focus:bg-white focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200"
              />
              <Icon name="search" size="xs" class="absolute left-2 top-2 text-gray-400" />
            </div>
            <div class="flex items-center gap-2 text-xs">
              <button
                type="button"
                @click="selectAllAccounts"
                class="text-primary-600 hover:text-primary-700 dark:text-primary-400 font-medium"
              >
                {{ t('admin.accounts.pelicanSelectAll') }}
              </button>
              <span class="text-gray-300 dark:text-dark-600">|</span>
              <button
                type="button"
                @click="clearAllAccounts"
                class="text-gray-500 hover:text-gray-700 dark:text-gray-400 font-medium"
              >
                {{ t('admin.accounts.pelicanClearSelection') }}
              </button>
            </div>
          </div>
          <div class="mt-2 grid max-h-40 grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2 overflow-y-auto pr-1">
            <label
              v-for="acc in filteredCandidateAccounts"
              :key="acc.id"
              class="flex items-center gap-2 rounded p-1.5 text-xs hover:bg-gray-50 cursor-pointer dark:hover:bg-dark-700/60"
            >
              <input
                type="checkbox"
                :checked="selectedAccountIds.has(acc.id)"
                @change="toggleAccountSelection(acc)"
                class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="truncate font-medium text-gray-800 dark:text-gray-200" :title="acc.name">{{ acc.name }}</span>
              <span class="text-[10px] text-gray-400 uppercase">({{ acc.type }})</span>
            </label>
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
          <span>完成: {{ completedCount }} / {{ accountStates.length }}</span>
          <span class="text-green-600 font-medium">未降智: {{ normalCount }}</span>
          <span class="text-amber-600 font-medium">疑似降智: {{ downgradedCount }}</span>
          <span v-if="failedCount > 0" class="text-red-600 font-medium">失败: {{ failedCount }}</span>
        </div>
        <div class="h-2 w-48 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
          <div
            class="h-full bg-primary-600 transition-all duration-300"
            :style="{ width: `${accountStates.length ? (completedCount / accountStates.length) * 100 : 0}%` }"
          />
        </div>
      </div>

      <!-- Empty account state hint -->
      <div
        v-if="accountStates.length === 0"
        class="flex h-64 flex-col items-center justify-center rounded-xl border border-dashed border-gray-300 p-8 text-center text-gray-500 dark:border-dark-700 dark:text-gray-400"
      >
        <Icon name="search" size="lg" class="mb-2 text-gray-400" />
        <p class="text-sm font-medium">{{ t('admin.accounts.pelicanNoAccountsSelected') }}</p>
        <button
          type="button"
          @click="showAccountPicker = true"
          class="mt-3 rounded-lg bg-primary-600 px-4 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-primary-500"
        >
          {{ t('admin.accounts.pelicanSelectAccounts') }}
        </button>
      </div>

      <!-- Results Grid (Dynamic columns & expanded heights based on account count) -->
      <div
        v-else
        class="grid max-h-[72vh] gap-4 overflow-y-auto p-1"
        :class="gridColsClass"
      >
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
                {{ isRunning ? t('admin.accounts.pelicanStatusQueued') : t('admin.accounts.pelicanStatusReady') }}
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
            <!-- Animation iframe preview with enlarged responsive height -->
            <div
              v-if="item.result?.has_html && item.activeTab === 'preview'"
              class="group/preview relative overflow-hidden rounded-lg border border-gray-200 bg-white shadow-inner dark:border-dark-600"
            >
              <iframe
                :srcdoc="pelicanPreviewDocument(item.result.html || '')"
                sandbox="allow-scripts"
                class="w-full border-0 transition-all duration-200"
                :class="previewHeightClass"
                loading="lazy"
                title="Animation Preview"
              />
              <!-- Enlarge / Lightbox Button in corner -->
              <button
                type="button"
                @click="openLightbox(item)"
                class="absolute right-2 top-2 flex items-center gap-1 rounded-lg bg-black/50 px-2 py-1 text-[11px] text-white opacity-80 backdrop-blur-sm transition hover:bg-black/80 hover:opacity-100 group-hover/preview:opacity-100"
                :title="t('admin.accounts.pelicanZoomIn')"
              >
                <Icon name="search" size="xs" />
                <span>{{ t('admin.accounts.pelicanZoomIn') }}</span>
              </button>
            </div>

            <!-- HTML Source view -->
            <div v-else-if="item.result?.has_html && item.activeTab === 'source'" class="relative">
              <pre
                class="overflow-y-auto rounded-lg bg-gray-900 p-2.5 font-mono text-[11px] text-gray-200"
                :class="previewHeightClass"
              ><code>{{ item.result.html }}</code></pre>
            </div>

            <!-- Streaming / Output preview while running -->
            <div
              v-else-if="item.status === 'running'"
              class="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 p-4 text-center text-xs text-gray-400 dark:border-dark-700"
              :class="previewHeightClass"
            >
              <Icon name="refresh" size="lg" class="mb-2 animate-spin text-primary-500" />
              <div class="line-clamp-4 font-mono text-[11px] text-gray-500 dark:text-gray-400">
                {{ item.streamingContent || t('admin.accounts.connectingToApi') }}
              </div>
            </div>

            <!-- Downgraded or failed message without HTML -->
            <div
              v-else-if="item.status === 'downgraded' || item.status === 'failed'"
              class="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 p-4 text-center text-xs text-gray-500 dark:border-dark-700"
              :class="previewHeightClass"
            >
              <Icon :name="item.status === 'downgraded' ? 'exclamationTriangle' : 'xCircle'" size="md" :class="item.status === 'downgraded' ? 'text-amber-500' : 'text-red-500'" class="mb-1" />
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ item.reason || item.error || t('admin.accounts.pelicanNoHtml') }}</span>
              <p v-if="item.streamingContent" class="mt-1 line-clamp-4 font-mono text-[11px] text-gray-400">
                {{ item.streamingContent }}
              </p>
            </div>

            <!-- Idle / Queued placeholder -->
            <div
              v-else
              class="flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 p-4 text-xs text-gray-400 dark:border-dark-700"
              :class="previewHeightClass"
            >
              <template v-if="isRunning">
                <Icon name="clock" size="md" class="mb-1 text-gray-400" />
                <span>{{ t('admin.accounts.pelicanStatusQueued') }}</span>
              </template>
              <template v-else>
                <button
                  type="button"
                  @click="runSingleTest(item)"
                  class="flex flex-col items-center gap-1 text-primary-600 hover:text-primary-700 transition"
                >
                  <Icon name="play" size="md" />
                  <span>{{ t('admin.accounts.pelicanStatusReady') }}</span>
                </button>
              </template>
            </div>
          </div>

          <!-- Card Footer Actions -->
          <div class="mt-auto flex items-center justify-between border-t border-gray-100 pt-2 text-xs dark:border-dark-700">
            <div class="flex items-center gap-1.5">
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
                <button
                  type="button"
                  @click="openLightbox(item)"
                  class="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                  :title="t('admin.accounts.pelicanZoomIn')"
                >
                  <Icon name="search" size="xs" />
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

    <!-- Lightbox Modal for Large Screen Preview -->
    <BaseDialog
      :show="lightboxItem !== null"
      :title="lightboxItem ? `${lightboxItem.account.name} - ${t('admin.accounts.pelicanAction')}` : ''"
      width="extra-wide"
      @close="closeLightbox"
    >
      <div v-if="lightboxItem" class="space-y-3">
        <div class="flex items-center justify-between border-b border-gray-100 pb-2 dark:border-dark-700">
          <div class="flex items-center gap-2">
            <span
              :class="[
                'inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-semibold',
                lightboxItem.status === 'success' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' :
                lightboxItem.status === 'downgraded' ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' :
                'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
              ]"
            >
              <Icon :name="lightboxItem.status === 'success' ? 'checkCircle' : 'exclamationTriangle'" size="xs" />
              {{ lightboxItem.status === 'success' ? t('admin.accounts.pelicanStatusSuccess') : t('admin.accounts.pelicanStatusDowngraded') }}
            </span>
            <span v-if="lightboxItem.responseModel" class="text-xs text-gray-500">
              ({{ lightboxItem.responseModel }})
            </span>
          </div>
          <div class="flex items-center gap-2">
            <button
              type="button"
              @click="lightboxTab = lightboxTab === 'preview' ? 'source' : 'preview'"
              class="rounded-lg border border-gray-200 px-3 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-700"
            >
              {{ lightboxTab === 'preview' ? t('admin.accounts.pelicanSourceTab') : t('admin.accounts.pelicanPreviewTab') }}
            </button>
            <button
              type="button"
              @click="copyToClipboard(lightboxItem.result?.html || '', t('admin.accounts.outputCopied'))"
              class="flex items-center gap-1 rounded-lg border border-gray-200 px-3 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-700"
            >
              <Icon name="copy" size="xs" />
              <span>{{ t('admin.accounts.copyOutput') }}</span>
            </button>
          </div>
        </div>

        <!-- Lightbox Content -->
        <div v-if="lightboxTab === 'preview'" class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-inner dark:border-dark-600">
          <iframe
            :srcdoc="pelicanPreviewDocument(lightboxItem.result?.html || '')"
            sandbox="allow-scripts"
            class="h-[65vh] w-full border-0"
            loading="lazy"
            title="Pelican Animation Full Preview"
          />
        </div>
        <div v-else class="relative">
          <pre class="h-[65vh] overflow-y-auto rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-200"><code>{{ lightboxItem.result?.html }}</code></pre>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button
            type="button"
            @click="closeLightbox"
            class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
          >
            {{ t('common.close') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <template #footer>
      <div class="flex items-center justify-between gap-3">
        <div class="text-xs text-gray-400">
          {{ t('admin.accounts.pelicanAccountsCount', { selected: accountStates.length, total: candidateAccounts.length || accountStates.length }) }}
        </div>
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

const props = withDefaults(
  defineProps<{
    show: boolean
    accounts: Account[]
    allAccounts?: Account[]
    autoStart?: boolean
  }>(),
  {
    allAccounts: () => [],
    autoStart: true
  }
)

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

// 默认并发数调整为 1
const concurrency = ref(1)
const isRunning = ref(false)
const showPromptEdit = ref(false)
const showAccountPicker = ref(false)
const accountSearchQuery = ref('')
const customPrompt = ref(t('admin.accounts.pelicanPromptDefault'))
const accountStates = ref<PelicanAccountState[]>([])
const selectedAccountIds = ref<Set<number>>(new Set())

// Lightbox preview state
const lightboxItem = ref<PelicanAccountState | null>(null)
const lightboxTab = ref<'preview' | 'source'>('preview')

const openLightbox = (item: PelicanAccountState) => {
  lightboxItem.value = item
  lightboxTab.value = 'preview'
}

const closeLightbox = () => {
  lightboxItem.value = null
}

// 可选候选账号（结合传入的全部账号与已有选中账号）
const candidateAccounts = computed<Account[]>(() => {
  const map = new Map<number, Account>()
  for (const acc of props.allAccounts) {
    map.set(acc.id, acc)
  }
  for (const acc of props.accounts) {
    map.set(acc.id, acc)
  }
  return Array.from(map.values())
})

const filteredCandidateAccounts = computed(() => {
  const q = accountSearchQuery.value.trim().toLowerCase()
  if (!q) return candidateAccounts.value
  return candidateAccounts.value.filter(
    acc => acc.name.toLowerCase().includes(q) || String(acc.id).includes(q) || acc.type.toLowerCase().includes(q)
  )
})

const toggleAccountSelection = (acc: Account) => {
  if (selectedAccountIds.value.has(acc.id)) {
    selectedAccountIds.value.delete(acc.id)
    accountStates.value = accountStates.value.filter(s => s.account.id !== acc.id)
  } else {
    selectedAccountIds.value.add(acc.id)
    accountStates.value.push({
      account: acc,
      status: 'idle',
      streamingContent: '',
      activeTab: 'preview'
    })
  }
}

const selectAllAccounts = () => {
  for (const acc of filteredCandidateAccounts.value) {
    selectedAccountIds.value.add(acc.id)
  }
  syncAccountStatesFromSelected()
}

const clearAllAccounts = () => {
  selectedAccountIds.value.clear()
  accountStates.value = []
}

const syncAccountStatesFromSelected = () => {
  const currentStatesMap = new Map<number, PelicanAccountState>()
  for (const state of accountStates.value) {
    currentStatesMap.set(state.account.id, state)
  }

  const newStates: PelicanAccountState[] = []
  for (const acc of candidateAccounts.value) {
    if (selectedAccountIds.value.has(acc.id)) {
      if (currentStatesMap.has(acc.id)) {
        newStates.push(currentStatesMap.get(acc.id)!)
      } else {
        newStates.push({
          account: acc,
          status: 'idle',
          streamingContent: '',
          activeTab: 'preview'
        })
      }
    }
  }
  accountStates.value = newStates
}

// 动态网格与高度控制：根据账号数量自适应放大
const gridColsClass = computed(() => {
  const count = accountStates.value.length
  if (count <= 1) return 'grid-cols-1'
  if (count === 2) return 'grid-cols-1 md:grid-cols-2'
  return 'grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3'
})

const previewHeightClass = computed(() => {
  const count = accountStates.value.length
  if (count <= 1) return 'h-[480px]'
  if (count === 2) return 'h-[380px]'
  return 'h-72'
})

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
    if (show) {
      customPrompt.value = t('admin.accounts.pelicanPromptDefault')
      const targetAccounts = accounts && accounts.length > 0 ? accounts : props.allAccounts.slice(0, 1)
      selectedAccountIds.value = new Set(targetAccounts.map(a => a.id))
      accountStates.value = targetAccounts.map(account => ({
        account,
        status: 'idle',
        streamingContent: '',
        activeTab: 'preview'
      }))

      // 打开弹窗后自动启动测智，免去卡在“排队中/待测试”的等待困扰
      if (props.autoStart && targetAccounts.length > 0) {
        setTimeout(() => {
          if (!isRunning.value && accountStates.value.length > 0) {
            startAllTests()
          }
        }, 50)
      }
    } else {
      stopAllTests()
      closeLightbox()
      showAccountPicker.value = false
    }
  },
  { immediate: true }
)

const handleClose = () => {
  stopAllTests()
  closeLightbox()
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

  // 将所有未成功的账号重置为就绪/排队
  for (const item of accountStates.value) {
    if (item.status !== 'success') {
      item.status = 'idle'
    }
  }

  const queue = [...accountStates.value.filter(s => s.status !== 'success')]
  const runQueue = queue.length > 0 ? queue : [...accountStates.value]
  const limit = Math.max(1, Math.min(10, concurrency.value || 1))

  const workers = Array.from({ length: limit }, async () => {
    while (runQueue.length > 0 && !signal.aborted) {
      const item = runQueue.shift()
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
  closeLightbox()
})
</script>
