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
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 pb-3 dark:border-dark-700">
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
          <div class="flex items-center gap-2">
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
              v-if="!isRunning"
              type="button"
              data-test="pelican-mark-downgraded-batch"
              :disabled="accountStates.length === 0 || isBatchMarking"
              @click="markSelectedAccounts(true)"
              class="flex items-center gap-1.5 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-medium text-amber-700 shadow-sm transition hover:bg-amber-100 disabled:opacity-50 dark:border-amber-700/60 dark:bg-amber-950/30 dark:text-amber-300 dark:hover:bg-amber-900/40"
              :title="t('admin.accounts.pelicanMarkDowngraded')"
            >
              <Icon name="exclamationTriangle" size="xs" />
              <span>{{ isBatchMarking ? t('admin.accounts.pelicanMarkingStatus') : t('admin.accounts.pelicanMarkDowngraded') }}</span>
            </button>
            <button
              v-if="!isRunning"
              type="button"
              data-test="pelican-mark-normal-batch"
              :disabled="accountStates.length === 0 || isBatchMarking"
              @click="markSelectedAccounts(false)"
              class="flex items-center gap-1.5 rounded-lg border border-green-300 bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700 shadow-sm transition hover:bg-green-100 disabled:opacity-50 dark:border-green-700/60 dark:bg-green-950/30 dark:text-green-300 dark:hover:bg-green-900/40"
              :title="t('admin.accounts.pelicanMarkNormal')"
            >
              <Icon name="checkCircle" size="xs" />
              <span>{{ isBatchMarking ? t('admin.accounts.pelicanMarkingStatus') : t('admin.accounts.pelicanMarkNormal') }}</span>
            </button>
            <button
              type="button"
              data-test="pelican-history-toggle"
              class="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 transition hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700"
              :title="t('admin.accounts.pelicanHistory')"
              @click="toggleHistoryPanel"
            >
              <span>{{ t('admin.accounts.pelicanHistory') }} ({{ allCachedPelicanResults.length }})</span>
            </button>
            <button
              type="button"
              class="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 transition hover:bg-gray-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700"
              :title="t('admin.accounts.pelicanClearCache')"
              @click="handleClearCache"
            >
              <span>{{ t('admin.accounts.pelicanClearCache') }}</span>
            </button>
            <button
              v-if="isRunning"
              @click="stopAllTests"
              class="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-1.5 text-xs font-medium text-white shadow-sm transition hover:bg-red-500"
            >
              <Icon name="x" size="sm" />
              <span>{{ t('admin.accounts.pelicanStopBatch') }}</span>
            </button>
          </div>
        </div>

        <!-- Controls: Model Selection, Prompt Scenario & Reasoning Effort -->
        <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-12 items-end">
          <!-- Big Model Selector (single dropdown) -->
          <div class="sm:col-span-2 md:col-span-6 space-y-1">
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.pelicanModel') }}
            </label>
            <select
              v-model="selectedModel"
              :disabled="isRunning"
              class="w-full rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
            >
              <option v-for="m in MODEL_PRESETS" :key="m.value" :value="m.value">
                {{ m.label }}
              </option>
            </select>
          </div>

          <!-- Prompt Scenario Dropdown -->
          <div class="sm:col-span-1 md:col-span-3">
            <select
              v-model="selectedPromptId"
              :aria-label="t('admin.accounts.pelicanPromptScenario')"
              :disabled="isRunning"
              class="w-full rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
            >
              <option v-for="scenario in promptScenarios" :key="scenario.id" :value="scenario.id">
                {{ pelicanPromptOptionLabel(scenario.prompt) }}
              </option>
            </select>
          </div>

          <!-- Reasoning Effort -->
          <div class="sm:col-span-1 md:col-span-3 space-y-1">
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.pelicanEffort') }}
            </label>
            <select
              v-model="reasoningEffort"
              :disabled="isRunning"
              class="w-full rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
            >
              <option value="low">{{ t('admin.accounts.pelicanEffortLow') }}</option>
              <option value="medium">{{ t('admin.accounts.pelicanEffortMedium') }}</option>
              <option value="high">{{ t('admin.accounts.pelicanEffortHigh') }}</option>
              <option value="xhigh">{{ t('admin.accounts.pelicanEffortXHigh') }}</option>
              <option value="max">{{ t('admin.accounts.pelicanEffortMax') }}</option>
            </select>
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

        <!-- Customizable Prompt editor (always visible for immediate preview) -->
        <div class="mt-3">
          <div class="pt-3 border-t border-gray-100 dark:border-dark-700">
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div class="text-xs font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.accounts.pelicanPromptLabel') }}
              </div>
              <div class="flex items-center gap-2">
                <button
                  type="button"
                  :disabled="isRunning || promptScenarios.length >= 20"
                  class="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-medium text-gray-700 transition hover:bg-gray-50 disabled:opacity-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700"
                  @click="addPromptScenario"
                >
                  <Icon name="plus" size="xs" />
                  <span>{{ t('admin.accounts.pelicanPromptAdd') }}</span>
                </button>
                <button
                  type="button"
                  :disabled="isRunning || promptScenarios.length <= 1"
                  class="inline-flex items-center gap-1 rounded-lg border border-red-200 px-2.5 py-1.5 text-xs font-medium text-red-600 transition hover:bg-red-50 disabled:opacity-50 dark:border-red-900/60 dark:text-red-400 dark:hover:bg-red-950/30"
                  @click="deletePromptScenario"
                >
                  <Icon name="trash" size="xs" />
                  <span>{{ t('admin.accounts.pelicanPromptDelete') }}</span>
                </button>
                <button
                  type="button"
                  data-test="pelican-prompt-save"
                  :disabled="isRunning || !promptDirty"
                  class="inline-flex items-center gap-1 rounded-lg bg-primary-600 px-2.5 py-1.5 text-xs font-medium text-white transition hover:bg-primary-500 disabled:opacity-50"
                  @click="savePromptScenario"
                >
                  <Icon name="checkCircle" size="xs" />
                  <span>{{ t('common.save') }}</span>
                </button>
              </div>
            </div>
            <TextArea
              v-model="customPrompt"
              :disabled="isRunning"
              rows="3"
              class="text-xs"
              :placeholder="t('admin.accounts.pelicanPromptPlaceholder')"
            />
          </div>
        </div>
      </div>

      <!-- Global cached history browser -->
      <div
        v-if="showHistoryPanel"
        data-test="pelican-history-panel"
        class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800/60"
      >
        <div class="flex items-center justify-between gap-2 border-b border-gray-100 pb-2 dark:border-dark-700">
          <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
            {{ t('admin.accounts.pelicanHistory') }}
          </div>
          <button
            type="button"
            class="rounded p-1 text-gray-400 transition hover:text-gray-600 dark:hover:text-gray-200"
            :title="t('common.close')"
            @click="showHistoryPanel = false"
          >
            <Icon name="x" size="xs" />
          </button>
        </div>

        <div
          v-if="allCachedPelicanResults.length === 0"
          class="flex h-24 items-center justify-center text-xs text-gray-500 dark:text-gray-400"
        >
          {{ t('admin.accounts.pelicanNoHistory') }}
        </div>
        <div v-else class="mt-2 grid gap-3 lg:grid-cols-[minmax(420px,1fr)_minmax(300px,420px)]">
          <div class="min-w-0">
            <div class="grid grid-cols-[minmax(72px,84px)_minmax(78px,92px)_minmax(58px,64px)_minmax(66px,74px)_minmax(112px,124px)] gap-2 border-b border-gray-100 px-1 pb-1 text-[10px] font-medium uppercase tracking-wide text-gray-400 dark:border-dark-700">
              <span>{{ t('admin.accounts.columns.name') }}</span>
              <span>{{ t('admin.accounts.pelicanModel') }}</span>
              <span>{{ t('admin.accounts.pelicanResultColumn') }}</span>
              <span>{{ t('admin.accounts.pelicanPromptColumn') }}</span>
              <span>{{ t('admin.accounts.pelicanUserRating') }}</span>
            </div>
            <div class="max-h-[calc(16rem+34px)] overflow-y-auto">
              <button
                v-for="entry in allCachedPelicanResults"
                :key="`${entry.account.id}-${entry.testedAt}`"
                type="button"
                data-test="pelican-global-history-entry"
                class="grid w-full grid-cols-[minmax(72px,84px)_minmax(78px,92px)_minmax(58px,64px)_minmax(66px,74px)_minmax(112px,124px)] items-center gap-2 border-b border-gray-100 px-1 py-1.5 text-left text-[11px] transition last:border-b-0 dark:border-dark-700"
                :class="isHistoryPreview(entry) ? 'bg-primary-50/60 dark:bg-primary-950/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700/60'"
                @click="historyPreview = entry"
              >
                <span class="min-w-0">
                  <span class="block truncate font-medium text-gray-800 dark:text-gray-200" :title="entry.account.name">
                    {{ entry.account.name }}
                  </span>
                  <span class="block truncate text-[10px] text-gray-400">
                    {{ formatTestedAt(entry.testedAt) }}
                  </span>
                </span>
                <span class="truncate text-gray-700 dark:text-gray-300">
                  {{ entry.responseModel || '-' }}
                </span>
                <span class="flex min-w-0 items-center gap-1">
                  <span class="inline-block size-1.5 shrink-0 rounded-full" :class="historyStatusDotClass(entry)" />
                  <span class="truncate" :class="historyResultTextClass(entry)" :title="historyResultText(entry)">
                    {{ historyResultText(entry) }}
                  </span>
                </span>
                <span class="truncate text-gray-500 dark:text-gray-400" :title="entry.prompt || t('admin.accounts.pelicanPromptMissing')">
                  {{ pelicanPromptOptionLabel(entry.prompt || '', 10) || t('admin.accounts.pelicanPromptMissing') }}
                </span>
                <span class="truncate rounded-full px-1.5 py-0.5 text-[10px] font-medium" :class="historyStatusClass(entry)">
                  {{ historyStatusLabel(entry) }}
                </span>
              </button>
            </div>
          </div>

          <div v-if="historyPreview" class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
                  {{ historyPreview.account.name }}
                </div>
                <div class="mt-0.5 flex flex-wrap items-center gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
                  <span class="uppercase">{{ historyPreview.account.type || '-' }}</span>
                  <span>•</span>
                  <span>ID: {{ historyPreview.account.id }}</span>
                  <span v-if="historyPreview.responseModel">• {{ historyPreview.responseModel }}</span>
                  <span>• {{ formatTestedAt(historyPreview.testedAt) }}</span>
                </div>
              </div>
              <span
                class="inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium"
                :class="historyStatusClass(historyPreview)"
              >
                {{ historyStatusLabel(historyPreview) }}
              </span>
            </div>
            <p
              v-if="historyPreview.reason || historyPreview.error"
              class="mt-2 rounded-md bg-gray-50 p-2 text-xs text-gray-600 dark:bg-dark-900/50 dark:text-gray-300"
            >
              {{ historyPreview.reason || historyPreview.error }}
            </p>
            <div class="mt-2 rounded-md bg-gray-50 p-2 text-xs dark:bg-dark-900/50">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="font-medium text-gray-600 dark:text-gray-300">
                  {{ t('admin.accounts.pelicanUserRating') }}
                </div>
                <div class="flex items-center gap-1.5">
                  <button
                    v-for="option in (['accurate', 'inaccurate'] as const)"
                    :key="option"
                    type="button"
                    :data-test="`pelican-rating-${option}`"
                    class="rounded-md border px-2 py-1 text-[11px] font-medium transition"
                    :class="historyPreview.userRating === option ? ratingButtonActiveClass(option) : ratingButtonIdleClass(option)"
                    @click="rateHistoryResult(historyPreview, option)"
                  >
                    {{ option === 'accurate' ? t('admin.accounts.pelicanRatingAccurate') : t('admin.accounts.pelicanRatingInaccurate') }}
                  </button>
                </div>
              </div>
            </div>
            <div class="mt-2 space-y-1">
              <div class="text-xs font-medium">{{ t('admin.accounts.pelicanPromptColumn') }}</div>
              <pre data-test="pelican-history-prompt" class="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded bg-gray-50 p-2 text-xs dark:bg-dark-900/50">{{ historyPreview.prompt || t('admin.accounts.pelicanPromptMissing') }}</pre>
            </div>
            <div class="mt-2 space-y-1">
              <div class="text-xs font-medium">{{ t('admin.accounts.pelicanResultColumn') }} / {{ t('admin.accounts.pelicanSourceTab') }}</div>
              <pre data-test="pelican-history-output" class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-gray-900 p-2 text-xs text-gray-200">{{ historyPreview.output || historyPreview.result?.html || historyPreview.error || historyPreview.reason || t('admin.accounts.pelicanHistoryContentMissing') }}</pre>
            </div>
            <div
              v-if="historyPreview.result?.has_html"
              class="mt-2 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600"
            >
              <iframe
                :srcdoc="pelicanPreviewDocument(historyPreview.result.html || '')"
                sandbox="allow-scripts"
                class="h-[42vh] w-full border-0"
                loading="lazy"
                title="Pelican History Preview"
              />
            </div>
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
                <span v-if="item.testedAt" class="text-[10px]">
                  {{ t('admin.accounts.pelicanCachedAt', { time: formatTestedAt(item.testedAt) }) }}
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

            <!-- History Toggle -->
            <button
              type="button"
              class="ml-1 flex shrink-0 items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 transition hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600"
              :title="t('admin.accounts.pelicanHistory')"
              @click="toggleHistory(item.account.id)"
            >
              <Icon name="clock" size="xs" />
              <span>{{ historyCount(item.account.id) }}</span>
              <Icon :name="expandedHistoryAccountIds.has(item.account.id) ? 'chevronUp' : 'chevronDown'" size="xs" />
            </button>
          </div>

          <!-- Historical Results -->
          <div
            v-if="expandedHistoryAccountIds.has(item.account.id) && getCachedPelicanHistory(item.account.id).length > 0"
            class="mt-2 max-h-36 overflow-y-auto rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-900/40"
          >
            <button
              v-for="(entry, index) in getCachedPelicanHistory(item.account.id)"
              :key="`${item.account.id}-${entry.testedAt}`"
              type="button"
              class="mb-1 flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-[11px] transition last:mb-0"
              :class="index === 0 ? 'bg-white shadow-sm dark:bg-dark-800' : 'hover:bg-white dark:hover:bg-dark-800'"
              @click="selectHistoryResult(item, index)"
            >
              <span class="flex min-w-0 items-center gap-2">
                <span
                  class="inline-block size-1.5 shrink-0 rounded-full"
                  :class="entry.status === 'success' ? 'bg-green-500' : entry.status === 'downgraded' ? 'bg-amber-500' : 'bg-red-500'"
                />
                <span class="truncate text-gray-700 dark:text-gray-300" :title="entry.reason || entry.error">
                  {{ entry.responseModel || entry.reason || entry.error || t('admin.accounts.pelicanAction') }}
                </span>
              </span>
              <span class="shrink-0 text-[10px] text-gray-400">{{ formatTestedAt(entry.testedAt) }}</span>
            </button>
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

            <div class="flex items-center gap-1">
              <!-- Manual mark buttons: available whenever not running -->
              <template v-if="item.status !== 'running'">
                <button
                  type="button"
                  :disabled="manualMarkingIds.has(item.account.id)"
                  @click="manualMarkItem(item, true)"
                  class="flex items-center gap-0.5 rounded px-1.5 py-0.5 text-[11px] font-medium transition disabled:opacity-50"
                  :class="item.status === 'downgraded' || item.account.extra?.pelican_downgraded === true
                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300 font-semibold'
                    : 'text-amber-600 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-950/30'"
                  :title="t('admin.accounts.pelicanMarkDowngraded')"
                >
                  <Icon name="exclamationTriangle" size="xs" />
                  <span>{{ manualMarkingIds.has(item.account.id) ? t('admin.accounts.pelicanMarkingStatus') : t('admin.accounts.pelicanMarkDowngraded') }}</span>
                </button>
                <button
                  type="button"
                  :disabled="manualMarkingIds.has(item.account.id)"
                  @click="manualMarkItem(item, false)"
                  class="flex items-center gap-0.5 rounded px-1.5 py-0.5 text-[11px] font-medium transition disabled:opacity-50"
                  :class="item.status === 'success' || (item.status !== 'downgraded' && item.account.extra?.pelican_downgraded === false)
                    ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300 font-semibold'
                    : 'text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-950/30'"
                  :title="t('admin.accounts.pelicanMarkNormal')"
                >
                  <Icon name="checkCircle" size="xs" />
                  <span>{{ manualMarkingIds.has(item.account.id) ? t('admin.accounts.pelicanMarkingStatus') : t('admin.accounts.pelicanMarkNormal') }}</span>
                </button>
              </template>
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
        <div class="flex items-center gap-2">
          <button
            @click="handleClose"
            class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
          >
            {{ t('common.close') }}
          </button>
          <button
            v-if="!isRunning"
            @click="startAllTests"
            :disabled="accountStates.length === 0"
            class="flex items-center gap-1.5 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-primary-500 disabled:opacity-50"
          >
            <Icon name="play" size="sm" />
            <span>{{ t('admin.accounts.pelicanStartBatch') }}</span>
          </button>
          <button
            v-if="!isRunning"
            type="button"
            :disabled="accountStates.length === 0 || isBatchMarking"
            @click="markSelectedAccounts(true)"
            class="flex items-center gap-1.5 rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-sm font-medium text-amber-700 shadow-sm transition hover:bg-amber-100 disabled:opacity-50 dark:border-amber-700/60 dark:bg-amber-950/30 dark:text-amber-300 dark:hover:bg-amber-900/40"
            :title="t('admin.accounts.pelicanMarkDowngraded')"
          >
            <Icon name="exclamationTriangle" size="xs" />
            <span>{{ isBatchMarking ? t('admin.accounts.pelicanMarkingStatus') : t('admin.accounts.pelicanMarkDowngraded') }}</span>
          </button>
          <button
            v-if="!isRunning"
            type="button"
            :disabled="accountStates.length === 0 || isBatchMarking"
            @click="markSelectedAccounts(false)"
            class="flex items-center gap-1.5 rounded-lg border border-green-300 bg-green-50 px-3 py-2 text-sm font-medium text-green-700 shadow-sm transition hover:bg-green-100 disabled:opacity-50 dark:border-green-700/60 dark:bg-green-950/30 dark:text-green-300 dark:hover:bg-green-900/40"
            :title="t('admin.accounts.pelicanMarkNormal')"
          >
            <Icon name="checkCircle" size="xs" />
            <span>{{ isBatchMarking ? t('admin.accounts.pelicanMarkingStatus') : t('admin.accounts.pelicanMarkNormal') }}</span>
          </button>
          <button
            v-else
            @click="stopAllTests"
            class="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-red-500"
          >
            <Icon name="x" size="sm" />
            <span>{{ t('admin.accounts.pelicanStopBatch') }}</span>
          </button>
        </div>
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
import {
  pelicanPromptOptionLabel,
  loadPelicanPromptScenarios,
  savePelicanPromptScenarios
} from '@/utils/pelicanPromptScenarios'
import {
  clearPelicanResultCache,
  getAllCachedPelicanResults,
  getCachedPelicanHistory,
  getCachedPelicanResult,
  setCachedPelicanResult,
  setCachedPelicanUserRating,
  type PelicanCachedResult,
  type PelicanUserRating
} from '@/utils/pelicanResultCache'
import { adminAPI } from '@/api'
import type { Account } from '@/types'

const DEFAULT_PELICAN_MODEL = 'gpt-6-astra'

const MODEL_PRESETS = [
  { label: 'gpt-6-astra (默认)', value: 'gpt-6-astra' },
  { label: 'gpt-5.6-sol', value: 'gpt-5.6-sol' },
  { label: 'gpt-5.6-terra', value: 'gpt-5.6-terra' },
  { label: 'gpt-5.6-luna', value: 'gpt-5.6-luna' },
  { label: 'gpt-5.5', value: 'gpt-5.5' },
  { label: 'gpt-5.4', value: 'gpt-5.4' },
  { label: 'gpt-5.4-mini', value: 'gpt-5.4-mini' }
]

const props = withDefaults(
  defineProps<{
    show: boolean
    accounts: Account[]
    allAccounts?: Account[]
    autoStart?: boolean
  }>(),
  {
    allAccounts: () => [],
    autoStart: false
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'updated', payload: { id: number; downgraded: boolean }): void
  (e: 'running-change', running: boolean): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

interface PelicanAccountState {
  account: Account
  status: 'idle' | 'running' | 'success' | 'downgraded' | 'failed'
  streamingContent: string
  prompt?: string
  responseModel?: string
  reason?: string
  error?: string
  elapsedMs?: number
  testedAt?: number
  activeTab: 'preview' | 'source'
  result?: {
    has_html: boolean
    html?: string
    downgraded: boolean
    reason: string
    response_model?: string
  }
}

// 默认模型选择与思考程度；点击开始后所有选中账号同时测试
const selectedModel = ref(DEFAULT_PELICAN_MODEL)
const reasoningEffort = ref('low')

const isRunning = ref(false)
const showAccountPicker = ref(false)
const accountSearchQuery = ref('')
const promptState = loadPelicanPromptScenarios(
  t('admin.accounts.pelicanPromptScenarioDefault'),
  t('admin.accounts.pelicanPromptDefault')
)
const promptScenarios = ref(promptState.scenarios)
const selectedPromptId = ref(promptState.selectedId)
const customPrompt = ref(promptState.scenarios.find(item => item.id === selectedPromptId.value)?.prompt || '')

const selectedPromptScenario = computed(() =>
  promptScenarios.value.find(item => item.id === selectedPromptId.value)
)
const promptDirty = computed(() =>
  customPrompt.value !== (selectedPromptScenario.value?.prompt || '')
)

const updateSelectedScenario = (mutator: (scenario: (typeof promptScenarios.value)[number]) => void) => {
  const scenario = promptScenarios.value.find(item => item.id === selectedPromptId.value)
  if (!scenario) return
  mutator(scenario)
  savePelicanPromptScenarios({
    scenarios: promptScenarios.value,
    selectedId: selectedPromptId.value
  })
}

watch(selectedPromptId, value => {
  const scenario = promptScenarios.value.find(item => item.id === value)
  if (!scenario) return
  customPrompt.value = scenario.prompt
})

const savePromptScenario = () => {
  if (isRunning.value || !promptDirty.value) return
  const prompt = customPrompt.value
  updateSelectedScenario(scenario => {
    scenario.prompt = prompt
    scenario.name = pelicanPromptOptionLabel(prompt, 40)
  })
}

const addPromptScenario = () => {
  const id = `prompt-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  const prompt = t('admin.accounts.pelicanPromptDefault')
  promptScenarios.value.push({
    id,
    name: pelicanPromptOptionLabel(prompt, 40),
    prompt
  })
  selectedPromptId.value = id
  customPrompt.value = prompt
  savePelicanPromptScenarios({ scenarios: promptScenarios.value, selectedId: id })
}

const deletePromptScenario = () => {
  if (promptScenarios.value.length <= 1) return
  const index = promptScenarios.value.findIndex(item => item.id === selectedPromptId.value)
  if (index < 0) return
  promptScenarios.value.splice(index, 1)
  const nextScenario = promptScenarios.value[Math.min(index, promptScenarios.value.length - 1)]
  selectedPromptId.value = nextScenario.id
  customPrompt.value = nextScenario.prompt
}
const accountStates = ref<PelicanAccountState[]>([])
const selectedAccountIds = ref<Set<number>>(new Set())
const expandedHistoryAccountIds = ref<Set<number>>(new Set())
const showHistoryPanel = ref(false)
const historyPreview = ref<PelicanCachedResult | null>(null)

const allCachedPelicanResults = computed(() => getAllCachedPelicanResults())

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

const toCachedResult = (item: PelicanAccountState): PelicanCachedResult => ({
  account: {
    id: item.account.id,
    name: item.account.name,
    type: item.account.type
  },
  status: item.status === 'idle' || item.status === 'running' ? 'failed' : item.status,
  result: item.result ? { ...item.result } : undefined,
  prompt: item.prompt,
  output: item.streamingContent,
  responseModel: item.responseModel,
  reason: item.reason,
  error: item.error,
  elapsedMs: item.elapsedMs,
  testedAt: item.testedAt ?? Date.now()
})

const applyCachedResult = (item: PelicanAccountState, cached: PelicanCachedResult) => {
  item.status = cached.status
  item.prompt = cached.prompt
  item.streamingContent = cached.output || cached.result?.html || ''
  item.result = cached.result ? { ...cached.result } : undefined
  item.responseModel = cached.responseModel
  item.reason = cached.reason
  item.error = cached.error
  item.elapsedMs = cached.elapsedMs
  item.testedAt = cached.testedAt
}

const hydrateFromCache = (item: PelicanAccountState): boolean => {
  const cached = getCachedPelicanResult(item.account.id)
  if (!cached) return false
  applyCachedResult(item, cached)
  return true
}

const historyCount = (accountId: number) => getCachedPelicanHistory(accountId).length

const toggleHistory = (accountId: number) => {
  const next = new Set(expandedHistoryAccountIds.value)
  if (next.has(accountId)) {
    next.delete(accountId)
  } else {
    next.add(accountId)
  }
  expandedHistoryAccountIds.value = next
}

const toggleHistoryPanel = () => {
  showHistoryPanel.value = !showHistoryPanel.value
  if (showHistoryPanel.value && !historyPreview.value) {
    historyPreview.value = allCachedPelicanResults.value[0] || null
  }
}

const historyStatusLabel = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') return t('admin.accounts.pelicanStatusSuccess')
  if (entry.status === 'downgraded') return t('admin.accounts.pelicanStatusDowngraded')
  return t('admin.accounts.pelicanStatusFailed')
}

const historyStatusClass = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') {
    return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  }
  if (entry.status === 'downgraded') {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  }
  return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
}

const historyStatusDotClass = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') return 'bg-green-500'
  if (entry.status === 'downgraded') return 'bg-amber-500'
  return 'bg-red-500'
}

const historyResultText = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') return t('admin.accounts.pelicanStatusSuccess')
  if (entry.status === 'downgraded') return t('admin.accounts.pelicanStatusDowngraded')
  return entry.error || entry.reason || t('admin.accounts.pelicanStatusFailed')
}

const historyResultTextClass = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') return 'text-green-600 dark:text-green-400'
  if (entry.status === 'downgraded') return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

const isHistoryPreview = (entry: PelicanCachedResult) =>
  historyPreview.value?.account.id === entry.account.id &&
  historyPreview.value?.testedAt === entry.testedAt

const ratingButtonActiveClass = (rating: PelicanUserRating) => (
  rating === 'accurate'
    ? 'border-green-300 bg-green-100 text-green-700 dark:border-green-700 dark:bg-green-900/40 dark:text-green-300'
    : 'border-red-300 bg-red-100 text-red-700 dark:border-red-700 dark:bg-red-900/40 dark:text-red-300'
)

const ratingButtonIdleClass = (rating: PelicanUserRating) => (
  rating === 'accurate'
    ? 'border-gray-300 text-gray-600 hover:bg-green-50 hover:text-green-700 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-green-950/30'
    : 'border-gray-300 text-gray-600 hover:bg-red-50 hover:text-red-700 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-red-950/30'
)

const rateHistoryResult = (entry: PelicanCachedResult, rating: PelicanUserRating) => {
  const updated = setCachedPelicanUserRating(entry.account.id, entry.testedAt, rating)
  if (updated && isHistoryPreview(entry)) {
    historyPreview.value = updated
  }
}

const selectHistoryResult = (item: PelicanAccountState, index: number) => {
  const entry = getCachedPelicanHistory(item.account.id)[index]
  if (!entry || item.status === 'running') return
  applyCachedResult(item, entry)
  historyPreview.value = entry
  showHistoryPanel.value = true
}

const formatTestedAt = (testedAt: number) => new Date(testedAt).toLocaleString()

const handleClearCache = () => {
  clearPelicanResultCache()
  expandedHistoryAccountIds.value = new Set()
  showHistoryPanel.value = false
  historyPreview.value = null
  for (const item of accountStates.value) {
    item.status = 'idle'
    item.streamingContent = ''
    item.result = undefined
    item.responseModel = undefined
    item.reason = undefined
    item.error = undefined
    item.elapsedMs = undefined
    item.testedAt = undefined
  }
}

const toggleAccountSelection = (acc: Account) => {
  if (selectedAccountIds.value.has(acc.id)) {
    selectedAccountIds.value.delete(acc.id)
    accountStates.value = accountStates.value.filter(s => s.account.id !== acc.id)
  } else {
    selectedAccountIds.value.add(acc.id)
    const item: PelicanAccountState = {
      account: acc,
      status: 'idle',
      streamingContent: '',
      activeTab: 'preview'
    }
    hydrateFromCache(item)
    accountStates.value.push(item)
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
        const item: PelicanAccountState = {
          account: acc,
          status: 'idle',
          streamingContent: '',
          activeTab: 'preview'
        }
        hydrateFromCache(item)
        newStates.push(item)
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

// Track which accounts are being manually marked (id → promise in flight)
const manualMarkingIds = ref<Set<number>>(new Set())

const manualMarkItem = async (item: PelicanAccountState, markAsDowngraded: boolean) => {
  if (manualMarkingIds.value.has(item.account.id)) return
  const next = new Set(manualMarkingIds.value)
  next.add(item.account.id)
  manualMarkingIds.value = next
  try {
    await adminAPI.accounts.updateDowngradedFlag(item.account.id, markAsDowngraded)
    emit('updated', { id: item.account.id, downgraded: markAsDowngraded })
    item.status = markAsDowngraded ? 'downgraded' : 'success'
    item.account.extra = {
      ...(item.account.extra || {}),
      pelican_downgraded: markAsDowngraded
    }
    // Keep result in sync so the cached entry also reflects the override
    if (item.result) {
      item.result = { ...item.result, downgraded: markAsDowngraded }
    }
    if (item.testedAt) {
      setCachedPelicanResult(toCachedResult(item))
    }
  } catch (error) {
    console.error('Failed to manually set pelican downgrade state:', error)
  } finally {
    const done = new Set(manualMarkingIds.value)
    done.delete(item.account.id)
    manualMarkingIds.value = done
  }
}

const isBatchMarking = ref(false)

const markSelectedAccounts = async (markAsDowngraded: boolean) => {
  if (isBatchMarking.value || accountStates.value.length === 0) return
  isBatchMarking.value = true
  try {
    for (const item of accountStates.value) {
      await manualMarkItem(item, markAsDowngraded)
    }
  } finally {
    isBatchMarking.value = false
  }
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
      if (!selectedModel.value) {
        selectedModel.value = DEFAULT_PELICAN_MODEL
      }
      if (!reasoningEffort.value) {
        reasoningEffort.value = 'low'
      }
      const targetAccounts = accounts && accounts.length > 0 ? accounts : props.allAccounts.slice(0, 1)

      if (isRunning.value) {
        const existingIds = new Set(accountStates.value.map(item => item.account.id))
        for (const account of targetAccounts) {
          if (existingIds.has(account.id)) continue
          const item: PelicanAccountState = {
            account,
            status: 'idle',
            streamingContent: '',
            activeTab: 'preview'
          }
          hydrateFromCache(item)
          accountStates.value.push(item)
          selectedAccountIds.value.add(account.id)
        }
      } else {
        selectedAccountIds.value = new Set(targetAccounts.map(a => a.id))
        accountStates.value = targetAccounts.map(account => {
          const item: PelicanAccountState = {
            account,
            status: 'idle',
            streamingContent: '',
            activeTab: 'preview'
          }
          hydrateFromCache(item)
          return item
        })
      }

      // 默认不在打开弹窗时直接开始，需在弹出窗口中选择好大模型后点击“开始测智”手动开始
      if (props.autoStart && targetAccounts.length > 0) {
        setTimeout(() => {
          if (!isRunning.value && accountStates.value.length > 0) {
            startAllTests()
          }
        }, 50)
      }
    } else {
      selectedAccountIds.value = new Set(Array.from(selectedAccountIds.value).filter(id =>
        candidateAccounts.value.some(acc => acc.id === id)
      ))
      if (!isRunning.value) {
        syncAccountStatesFromSelected()
      }
      closeLightbox()
      showAccountPicker.value = false
    }
  },
  { immediate: true }
)

const handleClose = () => {
  closeLightbox()
  showAccountPicker.value = false
  emit('close')
}

async function testSingleAccount(item: PelicanAccountState, signal: AbortSignal) {
  const testModel = selectedModel.value.trim() || DEFAULT_PELICAN_MODEL
  const testPrompt = customPrompt.value.trim() || t('admin.accounts.pelicanPromptDefault')
  item.prompt = testPrompt
  item.status = 'running'
  item.streamingContent = ''
  item.error = undefined
  item.reason = undefined
  item.result = undefined
  item.responseModel = testModel
  item.testedAt = undefined
  item.elapsedMs = undefined
  const startAt = Date.now()

  try {
    const requestBody: {
      model_id: string
      prompt: string
      mode: string
      reasoning_effort?: string
    } = {
      model_id: testModel,
      prompt: testPrompt,
      mode: 'pelican',
      reasoning_effort: reasoningEffort.value || 'low'
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
    if (item.status !== 'idle') {
      item.testedAt = Date.now()
      setCachedPelicanResult({
        ...toCachedResult(item),
        prompt: testPrompt
      })
      try {
        await adminAPI.accounts.updateDowngradedFlag(
          item.account.id,
          item.status === 'downgraded'
        )
        emit('updated', { id: item.account.id, downgraded: item.status === 'downgraded' })
        item.account.extra = {
          ...(item.account.extra || {}),
          pelican_downgraded: item.status === 'downgraded'
        }
      } catch (error) {
        console.error('Failed to persist pelican downgrade state:', error)
      }
    }
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
  emit('running-change', true)
  globalAbort = new AbortController()
  const signal = globalAbort.signal

  // 将所有未成功的账号重置为就绪/排队
  for (const item of accountStates.value) {
    if (item.status !== 'idle' && item.status !== 'running') {
      item.status = 'idle'
    }
  }

  const runQueue = [...accountStates.value]
  const workers = accountStates.value.map(async () => {
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
    emit('running-change', false)
  }
}

onUnmounted(() => {
  stopAllTests()
  closeLightbox()
})
</script>
