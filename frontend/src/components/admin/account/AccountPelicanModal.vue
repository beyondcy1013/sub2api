<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.accountTesting.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <div class="space-y-2.5">
      <!-- Description & Config Bar -->
      <div class="rounded-xl border border-gray-200 bg-gray-50/50 p-3 dark:border-dark-700 dark:bg-dark-800/40">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 pb-2 dark:border-dark-700">
          <div class="max-w-2xl">
            <div class="text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ t('admin.accounts.accountTesting.description') }}
            </div>
            <p v-if="scenarioError" role="alert" class="mt-1 text-xs text-red-600 dark:text-red-400">
              {{ t('admin.accounts.accountTesting.syncError') }} {{ scenarioError }}
            </p>
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
            <div v-if="accountStates.length > 0" class="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs" data-test="top-selected-accounts">
              <div
                v-for="item in accountStates"
                :key="item.account.id"
                class="inline-flex flex-wrap items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-2.5 py-1 text-xs shadow-xs dark:border-dark-700 dark:bg-dark-800"
              >
                <span class="font-semibold text-gray-900 dark:text-gray-100" :title="item.account.name">{{ item.account.name }}</span>
                <span class="uppercase text-[11px] text-gray-400">{{ item.account.type }}</span>
                <span class="text-gray-300 dark:text-dark-600">•</span>
                <span class="text-[11px] text-gray-500 dark:text-gray-400">ID: {{ item.account.id }}</span>
                <span v-if="item.responseModel || item.selectedModel" class="rounded bg-gray-100 px-1 text-[10px] text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ item.responseModel || item.selectedModel }}
                </span>
                <span v-if="item.testedAt" class="text-[10px] text-gray-400">
                  {{ t('admin.accounts.pelicanCachedAt', { time: formatTestedAt(item.testedAt) }) }}
                </span>
              </div>
            </div>

          </div>
          <div class="flex items-center gap-2">
            <details class="relative" data-test="account-test-more">
              <summary class="cursor-pointer rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-dark-600">{{ t('admin.accounts.accountTesting.more') }}</summary>
              <div class="absolute right-0 z-10 mt-1 w-48 space-y-1 rounded-lg border bg-white p-2 shadow-lg dark:border-dark-600 dark:bg-dark-800">
                <button v-for="downgraded in [true, false]" :key="String(downgraded)" type="button"
                  :data-test="downgraded ? 'pelican-mark-downgraded-batch' : 'pelican-mark-normal-batch'"
                  :disabled="isRunning || !accountStates.length || isBatchMarking"
                  class="block w-full rounded px-2 py-2 text-left text-xs disabled:opacity-50"
                  @click="markSelectedAccounts(downgraded)">
                  {{ t(downgraded ? 'admin.accounts.pelicanMarkDowngraded' : 'admin.accounts.pelicanMarkNormal') }}
                </button>
              </div>
            </details>
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

        <div class="mt-2 space-y-2">
          <label v-if="supportsCompact" class="flex items-center gap-2 text-xs">
            <input v-model="compactTest" data-test="account-test-compact" type="checkbox" :disabled="isRunning" />
            {{ t('admin.accounts.openai.testModeCompact') }}
          </label>
          <TextArea v-if="hasImageModel" v-model="imagePrompt" :disabled="isRunning || compactTest" :rows="2" :label="t('admin.accounts.imagePromptLabel')" />
        </div>
        <!-- Controls: Model Selection & Reasoning Effort -->
        <div class="mt-2.5 grid grid-cols-1 gap-3 sm:grid-cols-12 items-start">
          <!-- Model preset plus free-form model id input. -->
          <div class="col-span-1 sm:col-span-8 space-y-1">
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.pelicanModel') }}
            </label>
            <AccountTestModelPicker v-model="selectedModel" :options="accountConfiguredModels" :datalist-options="modelDatalistOptions" :disabled="isRunning" free-input native list-id="pelican-model-presets" @update:model-value="rememberSelectedModel" />
            <p class="text-[11px] text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.pelicanModelOverridePlaceholder') }}
            </p>
          </div>

          <!-- Reasoning Effort -->
          <div class="col-span-1 sm:col-span-4 space-y-1" :class="{'opacity-40 pointer-events-none select-none': compactTest}">
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.pelicanEffort') }}
            </label>
            <select
              v-model="reasoningEffort"
              :disabled="isRunning || compactTest"
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
                :disabled="isRunning" @click="selectAllAccounts"
                class="text-primary-600 hover:text-primary-700 dark:text-primary-400 font-medium"
              >
                {{ t('admin.accounts.pelicanSelectAll') }}
              </button>
              <span class="text-gray-300 dark:text-dark-600">|</span>
              <button
                type="button"
                :disabled="isRunning" @click="clearAllAccounts"
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
                :disabled="isRunning" :checked="selectedAccountIds.has(acc.id)"
                @change="toggleAccountSelection(acc)"
                class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-700"
              />
              <span class="truncate font-medium text-gray-800 dark:text-gray-200" :title="acc.name">{{ acc.name }}</span>
              <span class="text-[10px] text-gray-400 uppercase">({{ acc.type }})</span>
            </label>
          </div>
        </div>

        <!-- Capability prompt editor -->
        <div class="mt-2.5" :class="{'opacity-40 pointer-events-none select-none': compactTest}">
          <div class="pt-2 border-t border-gray-100 dark:border-dark-700">
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div class="flex min-w-0 flex-1 items-center gap-2">
                <label for="account-test-question" class="shrink-0 text-xs font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.pelicanPromptLabel') }}
                </label>
                <select
                  id="account-test-question"
                  data-test="account-test-question"
                  v-model="selectedPromptId"
                  :aria-label="t('admin.accounts.pelicanPromptScenario')"
                  :disabled="isRunning || scenariosBusy || !promptScenarios.length || compactTest"
                  class="min-w-0 flex-1 max-w-md rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300"
                >
                  <option v-for="scenario in promptScenarios" :key="scenario.id" :value="scenario.id">
                    {{ scenario.prompt }}
                  </option>
                </select>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button
                  type="button"
                  :disabled="isRunning || scenariosBusy || compactTest"
                  class="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-medium text-gray-700 transition hover:bg-gray-50 disabled:opacity-50 dark:border-dark-600 dark:text-gray-300 dark:hover:bg-dark-700"
                  @click="addPromptScenario"
                >
                  <Icon name="plus" size="xs" />
                  <span>{{ t('admin.accounts.pelicanPromptAdd') }}</span>
                </button>
                <button
                  type="button"
                  data-test="pelican-prompt-delete"
                  :disabled="isRunning || scenariosBusy || !selectedPromptScenario || compactTest"
                  class="inline-flex items-center gap-1 rounded-lg border border-red-200 px-2.5 py-1.5 text-xs font-medium text-red-600 transition hover:bg-red-50 disabled:opacity-50 dark:border-red-900/60 dark:text-red-400 dark:hover:bg-red-950/30"
                  @click="openDeleteScenarioDialog"
                >
                  <Icon name="trash" size="xs" />
                  <span>{{ t('admin.accounts.pelicanPromptDelete') }}</span>
                </button>
                <button
                  type="button"
                  data-test="pelican-prompt-save"
                  :disabled="isRunning || scenariosBusy || isKeywordQuestion || (!promptDirty && !selectedPromptId.startsWith('prompt-')) || compactTest"
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
              :disabled="isRunning || scenariosBusy || isKeywordQuestion || compactTest"
              :rows="2"
              class="text-xs max-h-20"
              textarea-class="min-h-[44px] max-h-20 text-xs py-1.5 leading-relaxed"
              :placeholder="t('admin.accounts.pelicanPromptPlaceholder')"
            />
            <div class="mt-2 flex flex-wrap items-center gap-3">
              <label class="flex items-center gap-1.5 text-xs">
                <span class="text-gray-600 dark:text-gray-400">{{ t('admin.accounts.accountTesting.evaluator') }}:</span>
                <span v-if="isKeywordQuestion" class="font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.accountTesting.keywordQuestion') }}</span>
                <select v-else v-model="probeEvaluator" data-test="account-test-evaluator" :disabled="isRunning || scenariosBusy || compactTest" class="rounded-lg border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300">
                  <option v-for="value in (['auto', 'html', 'numeric', 'reply'] as const)" :key="value" :value="value">{{ t(`admin.accounts.probeEvaluator${{auto:'Auto',html:'Html',numeric:'Numeric',reply:'Reply'}[value]}`) }}</option>
                </select>
              </label>
              <label v-if="probeEvaluator === 'numeric'" class="flex items-center gap-1.5 text-xs">
                <span class="text-gray-600 dark:text-gray-400">{{ t('admin.accounts.probeExpectedAnswer') }}:</span>
                <input v-model="probeExpectedAnswer" data-test="account-test-expected-answer" type="number" step="1" :disabled="isRunning || compactTest" class="w-24 rounded-lg border border-gray-300 bg-white px-2 py-1 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300" />
              </label>
              <p v-if="validationError" role="alert" class="text-xs text-red-600">{{ validationError }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Results Header Bar: Results label, progress stats, history toggle & clear cache -->
      <div class="flex flex-wrap items-center justify-between gap-2 pt-1">
        <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500">
          <span class="text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('admin.accounts.pelicanResultColumn') }}</span>
          <template v-if="isRunning || completedCount > 0">
            <span>完成: {{ completedCount }} / {{ accountStates.length }}</span>
            <span class="text-green-600 font-medium">{{ t('admin.accounts.accountTesting.passed') }}: {{ normalCount }}</span>
            <span v-if="downgradedCount > 0" class="text-amber-600 font-medium">疑似降智: {{ downgradedCount }}</span>
            <span v-if="failedCount > 0" class="text-red-600 font-medium">失败: {{ failedCount }}</span>
          </template>
        </div>
        <div class="flex items-center gap-2">
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
            :disabled="isRunning" :title="t('admin.accounts.pelicanClearCache')"
            @click="handleClearCache"
          >
            <span>{{ t('admin.accounts.pelicanClearCache') }}</span>
          </button>
        </div>
      </div>

      <!-- Overall progress bar when testing -->
      <div v-if="isRunning || completedCount > 0" class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
        <div
          class="h-full bg-primary-600 transition-all duration-300"
          :style="{ width: `${accountStates.length ? (completedCount / accountStates.length) * 100 : 0}%` }"
        />
      </div>

      <!-- History scoped to selected account IDs, never account type/model -->
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

        <label class="mt-2 flex items-center gap-2 text-xs">
          {{ t('admin.accounts.pelicanHistoryAccount') }}
          <select v-model="historyAccountId" data-test="pelican-history-account" class="rounded border border-gray-200 bg-white p-1 dark:border-dark-600 dark:bg-dark-800">
            <option :value="null">{{ t('admin.accounts.pelicanHistorySelected') }}</option>
            <option v-for="item in accountStates" :key="item.account.id" :value="item.account.id">
              {{ item.account.name }} · ID: {{ item.account.id }}
            </option>
          </select>
        </label>
        <div
          v-if="visibleHistoryResults.length === 0"
          class="flex h-24 items-center justify-center text-xs text-gray-500 dark:text-gray-400"
        >
          {{ t('admin.accounts.pelicanNoHistory') }}
        </div>
        <div v-else class="mt-2 grid gap-3 lg:grid-cols-[minmax(420px,1fr)_minmax(300px,420px)]">
          <div class="min-w-0">
            <div class="hidden grid-cols-[minmax(72px,84px)_minmax(78px,92px)_minmax(58px,64px)_minmax(66px,74px)_minmax(112px,124px)] gap-2 border-b border-gray-100 px-1 pb-1 text-[10px] font-medium uppercase tracking-wide text-gray-400 lg:grid dark:border-dark-700">
              <span>{{ t('admin.accounts.columns.name') }}</span>
              <span>{{ t('admin.accounts.pelicanModel') }}</span>
              <span>{{ t('admin.accounts.pelicanResultColumn') }}</span>
              <span>{{ t('admin.accounts.pelicanPromptColumn') }}</span>
              <span>{{ t('admin.accounts.pelicanUserRating') }}</span>
            </div>
            <div class="max-h-[calc(16rem+34px)] overflow-y-auto">
              <button
                v-for="entry in visibleHistoryResults"
                :key="`${entry.account.id}-${entry.testedAt}`"
                type="button"
                data-test="pelican-global-history-entry"
                class="w-full rounded-md border-b border-gray-100 px-1 py-2 text-left text-[11px] transition last:border-b-0 lg:grid lg:grid-cols-[minmax(72px,84px)_minmax(78px,92px)_minmax(58px,64px)_minmax(66px,74px)_minmax(112px,124px)] lg:items-center lg:gap-2 dark:border-dark-700"
                :class="isHistoryPreview(entry) ? 'bg-primary-50/60 dark:bg-primary-950/20' : 'hover:bg-gray-50 dark:hover:bg-dark-700/60'"
                @click="historyPreview = entry; mobilePreviewSource = false"
              >
                <span class="min-w-0">
                  <span class="block truncate font-medium text-gray-800 dark:text-gray-200" :title="entry.account.name">
                    {{ entry.account.name }} · ID: {{ entry.account.id }}
                  </span>
                  <span class="block truncate text-[10px] text-gray-400">
                    {{ formatTestedAt(entry.testedAt) }}
                  </span>
                </span>
                <span class="whitespace-pre-wrap break-words text-gray-700 dark:text-gray-300">
                  {{ entry.responseModel || '-' }}
                </span>
                <span class="flex min-w-0 items-start gap-1">
                  <span class="mt-1 inline-block size-1.5 shrink-0 rounded-full" :class="historyStatusDotClass(entry)" />
                  <span class="whitespace-pre-wrap break-words" :class="historyResultTextClass(entry)" :title="historyResultText(entry)">
                    {{ historyResultText(entry) }}
                  </span>
                </span>
                <span class="whitespace-pre-wrap break-words text-gray-500 dark:text-gray-400" :title="entry.prompt || t('admin.accounts.pelicanPromptMissing')">
                  {{ entry.prompt || t('admin.accounts.pelicanPromptMissing') }}
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
                <div class="whitespace-pre-wrap break-words text-sm font-semibold text-gray-900 dark:text-gray-100">
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
            <div v-if="historyPreviewHtml" role="tablist" class="mt-2 flex gap-1 border-b border-gray-200 dark:border-dark-600">
              <button
                v-for="tab in (['preview', 'source'] as const)"
                :key="tab"
                type="button"
                role="tab"
                :data-test="`pelican-history-tab-${tab}`"
                :aria-selected="mobilePreviewSource === (tab === 'source')"
                :class="mobilePreviewSource === (tab === 'source') ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400'"
                class="border-b-2 px-3 py-2 text-xs font-medium"
                @click="mobilePreviewSource = tab === 'source'"
              >
                {{ t(tab === 'preview' ? 'admin.accounts.pelicanPreviewTab' : 'admin.accounts.pelicanSourceTab') }}
              </button>
            </div>
            <div
              v-if="historyPreviewHtml"
              v-show="!mobilePreviewSource"
              class="mt-2 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-600"
            >
              <iframe
                :key="`${historyPreview.account.id}:${historyPreview.testedAt}`"
                data-test="pelican-history-preview"
                :srcdoc="pelicanPreviewDocument(historyPreviewHtml)"
                sandbox="allow-scripts"
                class="h-[56vh] min-h-[320px] w-full border-0"
                title="Pelican History Preview"
              />
            </div>
            <div class="mt-2 space-y-1">
              <div class="text-xs font-medium">{{ t('admin.accounts.pelicanPromptColumn') }}</div>
              <pre data-test="pelican-history-prompt" class="whitespace-pre-wrap break-words rounded bg-gray-50 p-2 text-xs dark:bg-dark-900/50">{{ historyPreview.prompt || t('admin.accounts.pelicanPromptMissing') }}</pre>
            </div>
            <details
              :key="`source:${historyPreview.account.id}:${historyPreview.testedAt}`"
              data-test="pelican-history-source"
              :open="!historyPreviewHtml || mobilePreviewSource"
              v-show="!historyPreviewHtml || mobilePreviewSource"
              class="mt-2 space-y-1"
            >
              <summary class="cursor-pointer text-xs font-medium">{{ t('admin.accounts.pelicanResultColumn') }} / {{ t('admin.accounts.pelicanSourceTab') }}</summary>
              <pre data-test="pelican-history-output" class="whitespace-pre-wrap break-words rounded bg-gray-900 p-2 text-xs text-gray-200">{{ historyPreview.output || historyPreview.result?.html || historyPreview.error || historyPreview.reason || t('admin.accounts.pelicanHistoryContentMissing') }}</pre>
            </details>
          </div>
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
        class="grid max-h-[58vh] gap-3.5 overflow-y-auto p-1"
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
            <div v-if="accountStates.length > 1" class="min-w-0">
              <div class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100" :title="item.account.name">
                {{ item.account.name }}
              </div>
              <div class="flex items-center gap-1.5 text-[11px] text-gray-400">
                <span class="uppercase">{{ item.account.type }}</span>
                <span>•</span>
                <span>ID: {{ item.account.id }}</span>
                <span v-if="item.responseModel || item.selectedModel" class="rounded bg-gray-100 px-1 text-[10px] text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ item.responseModel || item.selectedModel }}
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
                {{ item.cancelled ? t('admin.accounts.testStopped') : isRunning ? t('admin.accounts.pelicanStatusQueued') : t('admin.accounts.pelicanStatusReady') }}
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
                {{ item.testMode === 'default' || item.testMode === 'compact' ? t('admin.accounts.detection.connection.success') : t('admin.accounts.pelicanStatusSuccess') }}
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

            <DetectionResultSummary v-if="item.status !== 'idle' || item.cancelled" :status="item.cancelled ? 'cancelled' : item.status === 'downgraded' ? 'success' : item.status" :capability="isCapabilityResult(item)" :downgraded="item.status === 'downgraded'" :reason="item.reason" :error="item.error" :model="item.responseModel" :latency-ms="item.elapsedMs" :outcomes="item.ruleOutcomes" class="p-3" />
          <!-- Historical Results -->
          <div
            v-if="expandedHistoryAccountIds.has(item.account.id) && getCachedPelicanHistory(item.account.id).length > 0"
            class="mt-2 rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-900/40"
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
                <span class="whitespace-pre-wrap break-words text-gray-700 dark:text-gray-300" :title="entry.reason || entry.error">
                  {{ entry.responseModel || entry.reason || entry.error || t('admin.accounts.pelicanAction') }}
                </span>
              </span>
              <span class="shrink-0 text-[10px] text-gray-400">{{ formatTestedAt(entry.testedAt) }}</span>
            </button>
          </div>

          <div v-if="item.status !== 'idle' || item.cancelled" role="tablist" :aria-label="t('admin.accounts.pelicanResultColumn')" class="mt-2 flex shrink-0 gap-1 border-b border-gray-200 dark:border-dark-600">
            <button
              v-for="tab in resultTabs(item)"
              :key="tab"
              type="button"
              role="tab"
              :data-test="`pelican-card-tab-${tab}`"
              :aria-selected="item.activeTab === tab"
              :class="item.activeTab === tab ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'"
              class="border-b-2 px-3 py-2 text-xs font-medium"
              @click="item.activeTab = tab"
            >
              {{ resultTabLabel(tab) }}
            </button>
          </div>

          <!-- Card Content / Preview -->
          <div class="my-2 flex flex-1 flex-col justify-center">
            <div v-if="item.activeTab === 'details'" class="space-y-2 rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-900/50">
              <div>{{ t('admin.accounts.accountTesting.type') }}: {{ t(isCapabilityResult(item) ? 'admin.accounts.accountTesting.capability' : 'admin.accounts.accountTesting.connection') }}</div>
              <div>{{ t('admin.accounts.pelicanModel') }}: {{ item.responseModel || item.selectedModel || '-' }}</div>
              <div>{{ t('admin.accounts.pelicanPromptColumn') }}:</div>
              <pre class="whitespace-pre-wrap break-words">{{ item.prompt || '-' }}</pre>
              <pre class="whitespace-pre-wrap break-words">{{ item.logs?.join('\n') || '-' }}</pre>
            </div>
            <pre v-else-if="item.activeTab === 'output'" data-test="account-test-output" class="whitespace-pre-wrap break-words rounded-lg bg-gray-900 p-3 font-mono text-xs text-gray-200">{{ item.streamingContent || item.reason || item.error || (item.cancelled ? t('admin.accounts.testStopped') : (item.status === 'running' ? '' : t('admin.accounts.detection.connection.success'))) }}</pre>
            <div v-else-if="item.images?.length && item.activeTab === 'preview'" class="grid gap-2">
              <a v-for="url in item.images" :key="url" :href="url" target="_blank" rel="noopener noreferrer"><img :src="url" :alt="t('admin.accounts.imagePromptLabel')" class="max-h-96 rounded-lg object-contain" /></a>
            </div>
            <!-- Animation iframe preview with enlarged responsive height -->
            <div
              v-else-if="hasItemHtml(item) && item.activeTab === 'preview'"
              class="group/preview relative overflow-hidden rounded-lg border border-gray-200 bg-white shadow-inner dark:border-dark-600"
            >
              <iframe
                :srcdoc="pelicanPreviewDocument(getItemHtml(item))"
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
            <div v-else-if="hasItemHtml(item) && item.activeTab === 'source'" class="relative">
              <pre
                data-test="pelican-card-source"
                class="whitespace-pre-wrap break-words rounded-lg bg-gray-900 p-2.5 font-mono text-[11px] text-gray-200"
              ><code>{{ getItemHtml(item) }}</code></pre>
            </div>

            <!-- Streaming / Output preview while running -->
            <div
              v-else-if="item.status === 'running'"
              class="flex flex-col rounded-lg border border-dashed border-gray-200 p-4 text-left text-xs text-gray-400 dark:border-dark-700"
            >
              <Icon name="refresh" size="lg" class="mb-2 animate-spin text-primary-500" />
              <div data-test="pelican-streaming-output" class="w-full whitespace-pre-wrap break-words font-mono text-[11px] text-gray-500 dark:text-gray-400">
                {{ item.streamingContent }}
              </div>
            </div>

            <!-- Downgraded or failed message without HTML -->
            <div
              v-else-if="item.status === 'downgraded' || item.status === 'failed'"
              class="flex flex-col rounded-lg border border-dashed border-gray-200 p-4 text-left text-xs text-gray-500 dark:border-dark-700"
            >
              <Icon :name="item.status === 'downgraded' ? 'exclamationTriangle' : 'xCircle'" size="md" :class="item.status === 'downgraded' ? 'text-amber-500' : 'text-red-500'" class="mb-1" />
              <span class="whitespace-pre-wrap break-words font-medium text-gray-700 dark:text-gray-300">{{ item.reason || item.error || t('admin.accounts.pelicanNoHtml') }}</span>
              <p v-if="item.streamingContent" data-test="pelican-streaming-output" class="mt-1 w-full whitespace-pre-wrap break-words font-mono text-[11px] text-gray-400">
                {{ item.streamingContent }}
              </p>
            </div>

            <pre v-else-if="item.status === 'success'" data-test="account-test-output" class="whitespace-pre-wrap break-words rounded-lg bg-gray-900 p-3 text-xs text-gray-200">{{ item.streamingContent || item.reason || t('admin.accounts.detection.connection.success') }}</pre>

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
          <div class="mt-auto flex flex-wrap items-start justify-between gap-x-2 gap-y-2 border-t border-gray-100 pt-2 text-xs dark:border-dark-700 lg:flex-nowrap lg:items-center lg:justify-between">
            <div class="flex items-center gap-1.5">
              <template v-if="item.streamingContent || item.result?.has_html">
                <button
                  type="button"
                  @click="copyToClipboard(item.result?.html || item.streamingContent || '', t('admin.accounts.outputCopied'))"
                  class="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                  :title="t('admin.accounts.copyOutput')"
                >
                  <Icon name="copy" size="xs" />
                </button>
                <button
                  type="button"
                  v-if="item.result?.has_html" @click="openLightbox(item)"
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

            <div class="flex flex-wrap items-center gap-1 lg:flex-nowrap">
              <button v-if="accountStates.length > 1 && item.status === 'success' && (item.account.status !== 'active' || !item.account.schedulable)" type="button" class="rounded px-2 py-1 text-xs text-primary-600" @click="emit('test-succeeded', item.account)">{{ t('admin.accounts.recoverState') }}</button>
              <button v-if="accountStates.length > 1 && item.status === 'failed' && item.account.status !== 'error'" type="button" class="rounded px-2 py-1 text-xs text-red-600" @click="emit('test-failed', item.account, item.error || 'Test failed')">{{ t('admin.accounts.markAsFailed') }}</button>
              <!-- Manual mark buttons: available whenever not running -->
              <template v-if="item.status !== 'running' && isCapabilityResult(item)">
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
                v-if="item.status === 'running'"
                type="button"
                :data-test="`pelican-stop-${item.account.id}`"
                @click="stopSingleTest(item)"
                class="rounded px-2 py-0.5 text-[11px] font-medium text-red-600 hover:bg-red-50"
              >{{ t('admin.accounts.pelicanStop') }}</button>
              <button
                v-else
                type="button"
                :data-test="`pelican-test-${item.account.id}`"
                @click="runSingleTest(item)"
                class="flex items-center gap-1 rounded px-2 py-0.5 text-[11px] font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-950/30"
              >
                <Icon name="refresh" size="xs" />
                <span>{{ t('admin.accounts.pelicanTestSingle') }}</span>
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
              v-for="tab in (['preview', 'source'] as const)"
              :key="tab"
              role="tab"
              :aria-selected="lightboxTab === tab"
              :data-test="`pelican-lightbox-tab-${tab}`"
              @click="lightboxTab = tab"
              class="rounded-lg border border-gray-200 px-3 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-700"
            >
              {{ t(tab === 'preview' ? 'admin.accounts.pelicanPreviewTab' : 'admin.accounts.pelicanSourceTab') }}
            </button>
            <button
              type="button"
              @click="copyToClipboard(getItemHtml(lightboxItem), t('admin.accounts.outputCopied'))"
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
            :srcdoc="pelicanPreviewDocument(getItemHtml(lightboxItem))"
            sandbox="allow-scripts"
            class="h-[65vh] w-full border-0"
            loading="lazy"
            title="Pelican Animation Full Preview"
          />
        </div>
        <div v-else class="relative">
          <pre class="whitespace-pre-wrap break-words rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-200"><code>{{ getItemHtml(lightboxItem) }}</code></pre>
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
            :disabled="accountStates.length === 0 || scenariosBusy"
            class="flex items-center gap-1.5 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-primary-500 disabled:opacity-50"
          >
            <Icon name="play" size="sm" />
            <span>{{ t('admin.accounts.accountTesting.start', { count: accountStates.length }) }}</span>
          </button>
          <button
            v-if="isRunning"
            @click="stopAllTests"
            class="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-red-500"
          >
            <Icon name="x" size="sm" />
            <span>{{ t('admin.accounts.pelicanStopBatch') }}</span>
          </button>
        </div>
      </div>
    </template>
    <ConfirmDialog
      :show="showDeleteScenarioDialog"
      :title="t('admin.accounts.pelicanPromptDelete')"
      :message="t('admin.accounts.pelicanPromptDeleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDeletePromptScenario"
      @cancel="showDeleteScenarioDialog = false"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import AccountTestModelPicker from './AccountTestModelPicker.vue'
import DetectionResultSummary from './DetectionResultSummary.vue'
import type { DegradationRule, RuleOutcome } from '@/types'
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { buildApiUrl } from '@/api/client'
import { ADMIN_UI_REQUEST_HEADER } from '@/api/adminUIRequest'
import { extractPelicanPreviewHtml, pelicanPreviewDocument } from '@/utils/pelicanPreviewDocument'
import {
  pelicanPromptOptionLabel,
  loadPelicanPromptScenarios,
  savePelicanPromptScenarios
} from '@/utils/pelicanPromptScenarios'
import { loadPelicanModelHistory, rememberPelicanModel } from '@/utils/pelicanModelHistory'
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
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Account } from '@/types'

interface PelicanModel {
  id: string
  display_name?: string
}

const props = withDefaults(
  defineProps<{
    show: boolean
    accounts: Account[]
    allAccounts?: Account[]
    initialMode?: 'default' | 'pelican'
    autoStart?: boolean
  }>(),
  {
    allAccounts: () => [],
    initialMode: 'default',
    autoStart: false
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'test-succeeded', account: Account): void
  (e: 'test-failed', account: Account, error: string): void
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
  activeTab: 'preview' | 'source' | 'output' | 'details'
  testMode?: 'default' | 'compact' | 'pelican'
  images?: string[]
  logs?: string[]
  cancelled?: boolean
  selectedModel?: string
  availableModels?: PelicanModel[]
  modelsLoading?: boolean
  ruleOutcomes?: RuleOutcome[] | null
  result?: {
    has_html: boolean
    html?: string
    downgraded: boolean
    reason: string
    response_model?: string
  }
}

type PelicanStateWithModels = PelicanAccountState & {
  selectedModel: string
  availableModels: PelicanModel[]
  modelsLoading: boolean
}

// 全局值仅在非空时批量覆盖；空值表示每个账号使用自身目录中的默认模型。
const selectedModel = ref('')
const pelicanModelHistory = ref(loadPelicanModelHistory())
const compactTest = ref(false)
const probeEvaluator = ref('auto')
const probeExpectedAnswer = ref('')
const validationError = ref('')
const imagePrompt = ref(t('admin.accounts.imagePromptDefault'))
const supportsCompact = computed(() => accountStates.value.length > 0 && accountStates.value.every(item => item.account.platform === 'openai'))
const isImageModel = (item: PelicanAccountState) => {
  const model = (selectedModel.value.trim() || item.selectedModel || '').toLowerCase()
  return (item.account.platform === 'openai' && model.startsWith('gpt-image-'))
    || ((item.account.platform === 'gemini' || (item.account.platform === 'antigravity' && item.account.type === 'apikey')) && model.startsWith('gemini-') && model.includes('-image'))
}
const hasImageModel = computed(() => accountStates.value.some(isImageModel))
const isCapabilityResult = (item: PelicanAccountState) => item.testMode ? item.testMode === 'pelican' : testMode.value === 'pelican'
const getItemHtml = (item: PelicanAccountState | null | undefined): string => {
  if (!item) return ''
  return item.result?.html?.trim() || extractPelicanPreviewHtml(item.streamingContent || '')
}
const hasItemHtml = (item: PelicanAccountState | null | undefined): boolean => {
  return !!getItemHtml(item)
}
const resultTabs = (item: PelicanAccountState): PelicanAccountState['activeTab'][] => hasItemHtml(item)
  ? ['preview', 'source', 'output', 'details'] : item.images?.length ? ['preview', 'output', 'details'] : ['output', 'details']
const resultTabLabel = (tab: PelicanAccountState['activeTab']) => t({preview:'admin.accounts.pelicanPreviewTab',source:'admin.accounts.pelicanSourceTab',output:'admin.accounts.accountTesting.output',details:'admin.accounts.accountTesting.details'}[tab])
const REASONING_EFFORT_STORAGE_KEY = 'sub2api:account-pelican-reasoning-effort:v1'
const reasoningEffort = ref(localStorage.getItem(REASONING_EFFORT_STORAGE_KEY) || 'low')
watch(reasoningEffort, val => {
  if (val) localStorage.setItem(REASONING_EFFORT_STORAGE_KEY, val)
})
const detectionRules = ref<DegradationRule[]>([])
const scenariosBusy = ref(false)
const scenarioError = ref('')
const scenarioFromRule = (rule: DegradationRule) => ({
  id: `rule-${rule.id}`, name: rule.name, prompt: rule.prompt,
  evaluator: rule.params?.probe_evaluator || (rule.evaluator === 'html_document' ? 'html' as const
    : rule.evaluator === 'numeric_answer' ? 'numeric' as const : 'reply' as const),
  expectedAnswer: rule.evaluator === 'numeric_answer' ? rule.params?.expected_answer ?? 0 : undefined
})
const scenarioRequest = (scenario: (typeof promptScenarios.value)[number]) => {
  const evaluator = scenario.evaluator || 'auto'
  const html = evaluator === 'html' || (evaluator === 'auto' && /html|svg|鹈鹕.*(绘制|动画)/i.test(scenario.prompt))
  const name = scenario.name?.trim() || pelicanPromptOptionLabel(scenario.prompt, 40) || t('admin.accounts.pelicanPromptScenarioDefault')
  return {
    name, prompt: scenario.prompt, enabled: false,
    evaluator: evaluator === 'numeric' ? 'numeric_answer' as const : html ? 'html_document' as const : 'refusal_only' as const,
    params: { probe_evaluator: evaluator, ...(evaluator === 'numeric' ? { expected_answer: scenario.expectedAnswer } : {}) }
  }
}
const persistScenarios = () => savePelicanPromptScenarios({ scenarios: promptScenarios.value, selectedId: selectedPromptId.value })
const loadDegradationRules = async () => {
  if (scenariosBusy.value || isRunning.value) return
  scenariosBusy.value = true
  scenarioError.value = ''
  try {
    const rules = await adminAPI.degradationRules.list()
    // Old browser-only questions are uploaded once. Never re-import a cached
    // server ID: a question deleted on another device must stay deleted.
    const local = promptScenarios.value.filter(item => !item.id.startsWith('rule-'))
    const migrated = new Map<string, string>()
    for (const item of local) {
      if (!item.prompt.trim() || (item.id === 'default' && item.prompt === t('admin.accounts.pelicanPromptDefault') && (!item.evaluator || item.evaluator === 'auto'))) continue
      const req = scenarioRequest(item)
      let rule = rules.find(rule => rule.prompt.trim() === req.prompt.trim() && rule.evaluator === req.evaluator
        && (req.evaluator !== 'numeric_answer' || (rule.params?.expected_answer ?? 0) === req.params.expected_answer))
      if (!rule) {
        try {
          rule = await adminAPI.degradationRules.create(req)
          rules.push(rule)
        } catch (createError) {
          console.error('Failed to migrate local scenario:', item, createError)
          continue
        }
      }
      migrated.set(item.id, `rule-${rule.id}`)
    }
    detectionRules.value = rules
    const previous = selectedPromptId.value
    const dirty = promptDirty.value
    promptScenarios.value = [...rules.map(scenarioFromRule), ...local.filter(item => !migrated.has(item.id))]
    if (!promptScenarios.value.length) {
      promptScenarios.value = [{ id: 'default', name: t('admin.accounts.pelicanPromptScenarioDefault'), prompt: t('admin.accounts.pelicanPromptDefault') }]
    }
    selectedPromptId.value = migrated.get(previous) || (promptScenarios.value.some(item => item.id === previous) ? previous : (promptScenarios.value[0]?.id || ''))
    if (!dirty && previous === selectedPromptId.value) applySelectedScenario()
    persistScenarios()
  } catch (error) { scenarioError.value = extractApiErrorMessage(error) }
  finally { scenariosBusy.value = false }
}
const accountModelOptions = ref(new Map<number, PelicanModel[]>())
const accountModelsLoading = ref(new Set<number>())

const accountConfiguredModels = computed<PelicanModel[]>(() => {
  const merged = new Map<string, PelicanModel>()
  const states = accountStates.value.length > 0
    ? accountStates.value
    : candidateAccounts.value.map(acc => ({ account: acc } as PelicanAccountState))

  for (const item of states) {
    const models = (item as PelicanStateWithModels).availableModels?.length
      ? (item as PelicanStateWithModels).availableModels
      : accountModelOptions.value.get(item.account.id) || []
    for (const m of models) {
      if (m.id && !merged.has(m.id)) {
        merged.set(m.id, m)
      }
    }
  }
  return [...merged.values()].sort((a, b) => a.id.localeCompare(b.id))
})

const modelDatalistOptions = computed<PelicanModel[]>(() => {
  const merged = new Map<string, PelicanModel>()
  for (const model of accountConfiguredModels.value) {
    merged.set(model.id, model)
  }
  for (const id of pelicanModelHistory.value) {
    if (!merged.has(id)) {
      merged.set(id, { id })
    }
  }
  return [...merged.values()].sort((a, b) => a.id.localeCompare(b.id))
})

const rememberSelectedModel = () => {
  const model = selectedModel.value.trim()
  if (!model) return
  pelicanModelHistory.value = rememberPelicanModel(model)
}

const ensureAccountModels = async (item: PelicanAccountState) => {
  const state = item as PelicanStateWithModels
  if (state.availableModels?.length || state.modelsLoading) return
  const accountID = state.account.id
  const loading = new Set(accountModelsLoading.value)
  loading.add(accountID)
  accountModelsLoading.value = loading
  state.modelsLoading = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(accountID, { local: true })
    const normalized = (models || [])
      .map(model => ({
        id: String(model?.id || '').trim(),
        display_name: String(model?.display_name || '').trim()
      }))
      .filter(model => model.id.length > 0)
    const options = new Map(accountModelOptions.value)
    options.set(accountID, normalized)
    accountModelOptions.value = options
    state.availableModels = normalized
    // The backend catalog is authoritative. Do not invent a static default when
    // an account exposes no models.
    if (normalized.length > 0 && !state.selectedModel) {
      state.selectedModel = normalized[0].id
    }
  } catch (error) {
    console.error('Failed to load pelican account models:', error)
    state.availableModels = []
  } finally {
    const nextLoading = new Set(accountModelsLoading.value)
    nextLoading.delete(accountID)
    accountModelsLoading.value = nextLoading
    state.modelsLoading = false
  }
}

const hydrateAccountRuntime = async (item: PelicanAccountState) => {
  const state = item as PelicanStateWithModels
  if (!state.availableModels) {
    state.availableModels = accountModelOptions.value.get(state.account.id) || []
  }
  if (!state.availableModels.length && !state.modelsLoading) {
    await ensureAccountModels(state)
  }
  if (!state.selectedModel) state.selectedModel = state.availableModels[0]?.id || ''
}

const isRunning = ref(false)
const showAccountPicker = ref(false)
const accountSearchQuery = ref('')
const promptState = loadPelicanPromptScenarios(
  t('admin.accounts.pelicanPromptScenarioDefault'),
  t('admin.accounts.pelicanPromptDefault')
)
const promptScenarios = ref(promptState.scenarios)
const selectedPromptId = ref(
  promptState.selectedId && promptScenarios.value.some(item => item.id === promptState.selectedId)
    ? promptState.selectedId
    : (promptScenarios.value[0]?.id || '')
)
probeEvaluator.value = promptState.scenarios.find(item => item.id === promptState.selectedId)?.evaluator || 'auto'
probeExpectedAnswer.value = String(promptState.scenarios.find(item => item.id === promptState.selectedId)?.expectedAnswer ?? '')
const customPrompt = ref(promptState.scenarios.find(item => item.id === selectedPromptId.value)?.prompt || 'hi')

const selectedPromptScenario = computed(() =>
  promptScenarios.value.find(item => item.id === selectedPromptId.value)
)
const selectedSharedRule = computed(() => detectionRules.value.find(rule => selectedPromptId.value === `rule-${rule.id}`))
const isKeywordQuestion = computed(() => selectedSharedRule.value?.evaluator === 'contains_keywords')
const activeRuleIds = computed(() => {
  if (isKeywordQuestion.value && selectedSharedRule.value) return [selectedSharedRule.value.id]
  return []
})
const promptDirty = computed(() => {
  if (!selectedPromptScenario.value) return !!customPrompt.value.trim()
  return customPrompt.value !== (selectedPromptScenario.value?.prompt || '')
    || probeEvaluator.value !== (selectedPromptScenario.value?.evaluator || 'auto')
    || String(probeExpectedAnswer.value) !== (selectedPromptScenario.value?.expectedAnswer == null ? '' : String(selectedPromptScenario.value.expectedAnswer))
})

const applySelectedScenario = () => {
  const scenario = selectedPromptScenario.value
  if (!scenario) return
  customPrompt.value = scenario.prompt
  probeEvaluator.value = scenario.evaluator || 'auto'
  probeExpectedAnswer.value = scenario.expectedAnswer == null ? '' : String(scenario.expectedAnswer)
}
watch(selectedPromptId, () => { applySelectedScenario(); persistScenarios() })
const testMode = computed(() => compactTest.value ? 'default' : 'pelican')
watch(testMode, () => { validationError.value = '' })

const savePromptScenario = async () => {
  if (isRunning.value || scenariosBusy.value || isKeywordQuestion.value || (!promptDirty.value && selectedPromptId.value.startsWith('rule-'))) return
  if (!customPrompt.value.trim()) { scenarioError.value = t('admin.accounts.accountTesting.emptyQuestion'); return }
  if (probeEvaluator.value === 'numeric' && (!String(probeExpectedAnswer.value).trim() || !Number.isSafeInteger(Number(probeExpectedAnswer.value)))) {
    validationError.value = t('admin.accounts.probeExpectedAnswer'); return
  }
  const current = selectedPromptScenario.value
  scenariosBusy.value = true
  scenarioError.value = ''
  try {
    const draft = {
      id: current?.id || `prompt-${Date.now()}`,
      prompt: customPrompt.value,
      name: pelicanPromptOptionLabel(customPrompt.value, 40),
      evaluator: probeEvaluator.value as 'auto' | 'html' | 'numeric' | 'reply',
      expectedAnswer: probeEvaluator.value === 'numeric' ? Number(probeExpectedAnswer.value) : undefined
    }
    const req = scenarioRequest(draft)
    const existing = current ? detectionRules.value.find(rule => current.id === `rule-${rule.id}`) : undefined
    const rule = existing ? await adminAPI.degradationRules.update(existing.id, { ...req, enabled: existing.enabled, params: { ...existing.params, ...req.params } })
      : await adminAPI.degradationRules.create(req)
    const index = current ? promptScenarios.value.findIndex(item => item.id === current.id) : -1
    if (index < 0) promptScenarios.value.push(scenarioFromRule(rule))
    else promptScenarios.value.splice(index, 1, scenarioFromRule(rule))
    detectionRules.value = [...detectionRules.value.filter(item => item.id !== rule.id), rule]
    selectedPromptId.value = `rule-${rule.id}`
    persistScenarios()
  } catch (error) { scenarioError.value = extractApiErrorMessage(error) }
  finally { scenariosBusy.value = false }
}

const addPromptScenario = () => {
  if (isRunning.value || scenariosBusy.value) return
  const id = `prompt-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  const prompt = t('admin.accounts.pelicanPromptDefault')
  promptScenarios.value.push({
    id,
    name: pelicanPromptOptionLabel(prompt, 40),
    prompt
  })
  selectedPromptId.value = id
  applySelectedScenario()
  savePelicanPromptScenarios({ scenarios: promptScenarios.value, selectedId: id })
}

const deletePromptScenario = async () => {
  if (isRunning.value || scenariosBusy.value || !selectedPromptId.value) return
  const index = promptScenarios.value.findIndex(item => item.id === selectedPromptId.value)
  if (index < 0) return
  scenariosBusy.value = true
  scenarioError.value = ''
  try {
    const rule = detectionRules.value.find(item => selectedPromptId.value === `rule-${item.id}`)
    const ruleId = rule ? rule.id : (selectedPromptId.value.startsWith('rule-') ? parseInt(selectedPromptId.value.slice(5), 10) : null)
    if (ruleId != null && !isNaN(ruleId)) {
      try {
        await adminAPI.degradationRules.delete(ruleId)
      } catch (err: unknown) {
        console.warn('Failed to delete degradation rule on server:', err)
      }
      detectionRules.value = detectionRules.value.filter(item => item.id !== ruleId)
    }
  } catch (error) {
    scenarioError.value = extractApiErrorMessage(error)
  } finally {
    promptScenarios.value.splice(index, 1)
    if (promptScenarios.value.length > 0) {
      const nextIndex = Math.min(index, promptScenarios.value.length - 1)
      selectedPromptId.value = promptScenarios.value[nextIndex].id
    } else {
      selectedPromptId.value = ''
      customPrompt.value = ''
    }
    persistScenarios()
    scenariosBusy.value = false
  }
}

const showDeleteScenarioDialog = ref(false)

const openDeleteScenarioDialog = () => {
  if (isRunning.value || scenariosBusy.value || !selectedPromptScenario.value || compactTest.value) return
  showDeleteScenarioDialog.value = true
}

const confirmDeletePromptScenario = async () => {
  showDeleteScenarioDialog.value = false
  await deletePromptScenario()
}
const accountStates = ref<PelicanAccountState[]>([])
const selectedAccountIds = ref<Set<number>>(new Set())
const expandedHistoryAccountIds = ref<Set<number>>(new Set())
const showHistoryPanel = ref(false)
const historyPreview = ref<PelicanCachedResult | null>(null)
const mobilePreviewSource = ref(false)
const historyPreviewHtml = computed(() => {
  const entry = historyPreview.value
  return entry?.result?.html?.trim() || extractPelicanPreviewHtml(entry?.output || '')
})

const historyAccountId = ref<number | null>(null)
const allCachedPelicanResults = computed(() =>
  getAllCachedPelicanResults().filter(entry => selectedAccountIds.value.has(entry.account.id))
)
const visibleHistoryResults = computed(() =>
  allCachedPelicanResults.value.filter(entry =>
    historyAccountId.value === null || entry.account.id === historyAccountId.value
  )
)
watch([visibleHistoryResults, historyAccountId], () => {
  if (historyAccountId.value !== null && !selectedAccountIds.value.has(historyAccountId.value)) {
    historyAccountId.value = null
    return
  }
  const current = historyPreview.value
  historyPreview.value = visibleHistoryResults.value.find(entry =>
    entry.account.id === current?.account.id && entry.testedAt === current?.testedAt
  ) || visibleHistoryResults.value[0] || null
})

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

const toCachedResult = (item: PelicanAccountState): PelicanCachedResult => {
  const html = getItemHtml(item)
  return {
    account: {
      id: item.account.id,
      name: item.account.name,
      type: item.account.type
    },
    status: item.status === 'idle' || item.status === 'running' ? 'failed' : item.status,
    result: (item.result || html) ? {
      has_html: !!html,
      html: html || undefined,
      downgraded: item.result?.downgraded ?? (item.status === 'downgraded'),
      reason: item.result?.reason ?? (item.reason || ''),
      response_model: item.result?.response_model ?? item.responseModel
    } : undefined,
    testMode: item.testMode,
    images: item.images,
    logs: item.logs,
    prompt: item.prompt,
    output: item.streamingContent,
    responseModel: item.responseModel,
    reason: item.reason,
    error: item.error,
    elapsedMs: item.elapsedMs,
    testedAt: item.testedAt ?? Date.now()
  }
}

const applyCachedResult = (item: PelicanAccountState, cached: PelicanCachedResult) => {
  item.cancelled = false
  item.testMode = cached.testMode || 'pelican'
  item.images = cached.images
  item.logs = cached.logs
  item.status = cached.status
  item.prompt = cached.prompt
  item.streamingContent = cached.output || cached.result?.html || ''
  item.result = cached.result ? { ...cached.result } : undefined
  const html = item.result?.html?.trim() || extractPelicanPreviewHtml(item.streamingContent)
  if (html) {
    if (!item.result) {
      item.result = { has_html: true, html, downgraded: item.status === 'downgraded', reason: item.reason || '', response_model: cached.responseModel }
    } else {
      item.result.has_html = true
      item.result.html = html
    }
  }
  item.activeTab = (html || cached.images?.length) ? 'preview' : 'output'
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
  if (cached.responseModel) {
    item.selectedModel = cached.responseModel
  }
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
  historyAccountId.value = null
  showHistoryPanel.value = !showHistoryPanel.value
  mobilePreviewSource.value = false
  if (showHistoryPanel.value && !historyPreview.value) {
    historyPreview.value = allCachedPelicanResults.value[0] || null
  }
}

const historyStatusLabel = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') return t(entry.testMode === 'default' || entry.testMode === 'compact' ? 'admin.accounts.detection.connection.success' : 'admin.accounts.pelicanStatusSuccess')
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
  if (entry.status === 'success') return t(entry.testMode === 'default' || entry.testMode === 'compact' ? 'admin.accounts.detection.connection.success' : 'admin.accounts.pelicanStatusSuccess')
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
  historyAccountId.value = item.account.id
  historyPreview.value = entry
  showHistoryPanel.value = true
  mobilePreviewSource.value = false
}

const formatTestedAt = (testedAt: number) => new Date(testedAt).toLocaleString()

const handleClearCache = () => {
  if (isRunning.value) return
  clearPelicanResultCache()
  expandedHistoryAccountIds.value = new Set()
  showHistoryPanel.value = false
  historyPreview.value = null
  for (const item of accountStates.value) {
    item.status = 'idle'
    item.cancelled = false
    item.streamingContent = ''
    item.result = undefined
  item.ruleOutcomes = undefined
    item.responseModel = undefined
    item.reason = undefined
    item.error = undefined
    item.elapsedMs = undefined
    item.testedAt = undefined
    const state = item as PelicanStateWithModels
    state.selectedModel = state.availableModels[0]?.id || ''
  }
}

const toggleAccountSelection = (acc: Account) => {
  if (isRunning.value) return
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
    void hydrateAccountRuntime(item)
    accountStates.value.push(item)
  }
}

const selectAllAccounts = () => {
  if (isRunning.value) return
  for (const acc of filteredCandidateAccounts.value) {
    selectedAccountIds.value.add(acc.id)
  }
  syncAccountStatesFromSelected()
}

const clearAllAccounts = () => {
  if (isRunning.value) return
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
        void hydrateAccountRuntime(item)
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
  if (count <= 1) return 'h-[min(70vh,480px)]'
  if (count === 2) return 'h-[380px]'
  return 'h-72'
})

const accountControllers = new Map<number, AbortController>()

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


const stopSingleTest = (item: PelicanAccountState) => {
  accountControllers.get(item.account.id)?.abort()
}

const stopAllTests = () => {
  for (const controller of accountControllers.values()) controller.abort()
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
      if (!reasoningEffort.value) {
        reasoningEffort.value = 'low'
      }
      if (!isRunning.value) { applySelectedScenario(); void loadDegradationRules() }
      const targetAccounts = accounts || []
      showAccountPicker.value = false

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
          void hydrateAccountRuntime(item)
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
          void hydrateAccountRuntime(item)
          return item
        })
        for (const item of accountStates.value) {
          void hydrateAccountRuntime(item)
        }
      }

      if (targetAccounts.length === 1) {
        const singleAcc = targetAccounts[0]
        const cached = getCachedPelicanResult(singleAcc.id)
        if (cached) {
          if (cached.responseModel) {
            selectedModel.value = cached.responseModel
          }
          if (cached.prompt) {
            const matchedScenario = promptScenarios.value.find(s => s.prompt.trim() === cached.prompt?.trim())
            if (matchedScenario) {
              selectedPromptId.value = matchedScenario.id
              applySelectedScenario()
            } else {
              customPrompt.value = cached.prompt
            }
          }
          if (cached.testMode) {
            compactTest.value = (cached.testMode === 'compact')
          }
        } else {
          if (!selectedModel.value.trim() && pelicanModelHistory.value.length > 0) {
            selectedModel.value = pelicanModelHistory.value[0]
          }
        }
      } else if (targetAccounts.length > 1) {
        if (!selectedModel.value.trim() && pelicanModelHistory.value.length > 0) {
          selectedModel.value = pelicanModelHistory.value[0]
        }
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
  rememberSelectedModel()
  closeLightbox()
  showAccountPicker.value = false
  emit('close')
}

async function testSingleAccount(item: PelicanAccountState, signal: AbortSignal) {
  const overrideModel = selectedModel.value?.trim() || ''
  if (overrideModel) rememberSelectedModel()
  const testModel = overrideModel || item.selectedModel?.trim() || ''
  if (!testModel) {
    item.testMode = testMode.value
    item.result = undefined
    item.reason = undefined
    item.streamingContent = ''
    item.activeTab = 'output'
    item.status = 'failed'
    item.error = t('admin.accounts.pelicanModelsUnavailable')
    return
  }
  const capability = !isImageModel(item) && testMode.value === 'pelican'
  const requestMode = capability ? 'pelican' : compactTest.value && supportsCompact.value ? 'compact' : 'default'
  const sharedRules = capability && isKeywordQuestion.value
  const testPrompt = !capability
    ? (isImageModel(item) ? imagePrompt.value.trim() : '')
    : customPrompt.value.trim() || t('admin.accounts.pelicanPromptDefault')
  const selectedRulePrompt = isKeywordQuestion.value ? selectedSharedRule.value?.prompt : undefined
  const effectiveTestPrompt = selectedRulePrompt || testPrompt
  item.cancelled = false
  item.testMode = requestMode
  item.images = []
  item.logs = []
  item.activeTab = 'output'
  item.prompt = effectiveTestPrompt
  item.status = 'running'
  item.streamingContent = ''
  item.error = undefined
  item.reason = undefined
  item.result = undefined
  item.ruleOutcomes = undefined
  item.responseModel = testModel
  item.testedAt = undefined
  item.elapsedMs = undefined
  const startAt = Date.now()

  try {
    if (sharedRules) {
      const result = await adminAPI.accounts.runQualityRules(item.account.id, testModel, activeRuleIds.value, signal)
      item.streamingContent = result.response_text
      item.responseModel = result.response_model || testModel
      item.ruleOutcomes = result.rule_results
      item.reason = result.downgrade_reason
      item.error = result.error_message || undefined
      item.status = result.status !== 'success' ? 'failed' : result.downgraded ? 'downgraded' : 'success'
      const html = extractPelicanPreviewHtml(result.response_text)
      item.result = {has_html:!!html,html,downgraded:result.downgraded,reason:result.downgrade_reason,response_model:result.response_model}
      return
    }
    const requestBody: {
      model_id: string
      prompt: string
      mode: string
      reasoning_effort?: string
      evaluator?: string
      expected_answer?: number
    } = {
      model_id: testModel,
      prompt: testPrompt,
      mode: requestMode,
      ...(capability ? {reasoning_effort: reasoningEffort.value || 'low', evaluator: probeEvaluator.value, ...(probeEvaluator.value === 'numeric' ? {expected_answer: Number(probeExpectedAnswer.value)} : {})} : {})
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

    const handleLine = (line: string) => {
      if (!line.startsWith('data:')) return
      const jsonStr = line.slice(5).trim()
      if (!jsonStr || jsonStr === '[DONE]') return
      let event
      try { event = JSON.parse(jsonStr) } catch { return }
      if (event.type === 'test_start' && event.model) item.responseModel = event.model
      else if (event.type === 'content' && event.text) item.streamingContent += event.text
      else if (event.type === 'status' && event.text) item.logs!.push(event.text)
      else if (event.type === 'image' && event.image_url) item.images!.push(event.image_url)
      else if (capability && event.type === 'pelican_result' && event.data) {
        item.result = event.data
        item.responseModel = event.data.response_model || item.responseModel
        item.reason = event.data.reason
        item.status = event.data.downgraded ? 'downgraded' : 'success'
        item.activeTab = (event.data.has_html || hasItemHtml(item)) ? 'preview' : 'output'
      } else if (event.type === 'error' || (event.type === 'test_complete' && event.success === false && !item.result)) {
        item.error = event.error || 'Test failed'
        if (!item.result) item.status = 'failed'
      } else if (!capability && event.type === 'test_complete' && event.success === true && !item.error) {
        item.status = 'success'
        if (item.images?.length || hasItemHtml(item)) item.activeTab = 'preview'
      }
    }
    try {
      while (true) {
        const { done, value } = await reader.read()
        if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''
        for (const line of lines) handleLine(line)
      }
      buffer += decoder.decode()
      if (buffer.trim()) handleLine(buffer)
    } finally {
      reader.releaseLock?.()
    }

    if (item.status === 'running') {
      item.status = 'failed'
      item.error = 'No test result returned'
    }
  } catch (err: unknown) {
    if (signal.aborted) {
      item.status = 'idle'
      return
    }
    // pelican_result 到达后流尾部再断开（如末尾大事件传输被切断）不应推翻已判定的结论。
    if (item.status === 'running') {
      item.status = 'failed'
      item.error = extractApiErrorMessage(err, 'Test failed')
    }
  } finally {
    item.elapsedMs = Date.now() - startAt
    if (signal.aborted) {
      item.cancelled = true
      item.status = 'idle'
    }
    if (item.status !== 'idle') {
      if (item.result?.has_html) item.activeTab = 'preview'
      if (accountStates.value.length === 1 && item.status === 'success') emit('test-succeeded', item.account)
      else if (accountStates.value.length === 1 && item.status === 'failed') emit('test-failed', item.account, item.error || 'Test failed')
      item.testedAt = Date.now()
      setCachedPelicanResult({
        ...toCachedResult(item),
      prompt: effectiveTestPrompt
      })
      if (capability && (item.status === 'success' || item.status === 'downgraded')) try {
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
  if (scenariosBusy.value || accountControllers.has(item.account.id)) return
  if (testMode.value === 'pelican' && probeEvaluator.value === 'numeric'
    && (!String(probeExpectedAnswer.value).trim() || !Number.isSafeInteger(Number(probeExpectedAnswer.value)))) {
    validationError.value = t('admin.accounts.probeExpectedAnswer')
    return
  }
  validationError.value = ''
  if (item.modelsLoading && !selectedModel.value.trim() && !item.selectedModel?.trim()) {
    await ensureAccountModels(item as PelicanStateWithModels)
  }
  if (!selectedModel.value.trim() && !item.selectedModel?.trim()) return
  const controller = new AbortController()
  accountControllers.set(item.account.id, controller)
  isRunning.value = true
  emit('running-change', true)
  try {
    await testSingleAccount(item, controller.signal)
  } finally {
    accountControllers.delete(item.account.id)
    isRunning.value = accountControllers.size > 0
    emit('running-change', isRunning.value)
  }
}

async function startAllTests() {
  if (isRunning.value) return
  await Promise.all(accountStates.value.map(item => runSingleTest(item)))
}

onUnmounted(() => {
  stopAllTests()
  closeLightbox()
})
</script>
