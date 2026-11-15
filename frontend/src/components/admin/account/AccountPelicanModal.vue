<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.accountTesting.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <template #title-actions>
      <div class="relative">
        <button
          type="button"
          data-test="account-test-help"
          class="rounded-lg p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:hover:bg-dark-700 dark:hover:text-gray-300"
          :aria-label="t('admin.accounts.accountTesting.help')"
          :aria-expanded="showAccountTestHelp"
          :title="t('admin.accounts.accountTesting.description')"
          @click.stop="showAccountTestHelp = !showAccountTestHelp"
          @blur="showAccountTestHelp = false"
        >
          <Icon name="questionCircle" size="sm" />
        </button>
        <div
          v-if="showAccountTestHelp"
          role="tooltip"
          class="absolute left-0 top-full z-10 mt-2 w-72 max-w-[calc(100vw-3rem)] rounded-lg border border-gray-200 bg-white p-3 text-xs font-normal leading-relaxed text-gray-600 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
        >
          {{ t('admin.accounts.accountTesting.description') }}
        </div>
      </div>
    </template>
    <div class="space-y-2.5">
      <!-- Description & Config Bar -->
      <div class="rounded-xl border border-gray-200 bg-gray-50/50 p-3 dark:border-dark-700 dark:bg-dark-800/40">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 pb-2 dark:border-dark-700">
          <div class="max-w-2xl">
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
          </div>
        </div>

        <!-- Controls: Model Selection & Reasoning Effort -->
        <div class="mt-2.5 grid grid-cols-1 gap-3 sm:grid-cols-12 items-start">
          <!-- Model preset plus free-form model id input. -->
          <div class="col-span-1 sm:col-span-8 space-y-1">
            <div class="flex items-center justify-between gap-2">
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.accounts.pelicanModel') }}
              </label>
              <button
                v-if="pelicanModelHistory.length > 0"
                type="button"
                data-test="pelican-model-history-clear"
                class="shrink-0 text-[11px] text-gray-400 transition hover:text-gray-600 dark:hover:text-gray-200"
                :disabled="isRunning"
                :title="t('admin.accounts.pelicanModelHistoryClearHint')"
                @click="clearModelHistory"
              >
                {{ t('admin.accounts.pelicanModelHistoryClear') }}
              </button>
            </div>
            <AccountTestModelPicker v-model="selectedModel" :options="modelDatalistOptions" :datalist-options="modelDatalistOptions" :disabled="isRunning" free-input native @update:model-value="rememberSelectedModel" />
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

        <!-- One question editor for every manual test. -->
        <div class="mt-2.5" :class="{'opacity-40 pointer-events-none select-none': compactTest}">
          <div class="pt-2 border-t border-gray-100 dark:border-dark-700">
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div class="flex min-w-0 flex-1 items-center gap-2">
                <label for="account-test-question" class="shrink-0 text-xs font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.pelicanPromptLabel') }}
                </label>
                <Select
                  id="account-test-question"
                  data-test="account-test-question"
                  v-model="selectedPromptId"
                  :aria-label="t('admin.accounts.pelicanPromptScenario')"
                  :disabled="isRunning || scenariosBusy || !promptScenarios.length || compactTest"
                  :options="promptScenarios.map(scenario => ({ value: scenario.id, label: pelicanPromptOptionLabel(scenario.prompt || scenario.name) }))"
                  class="min-w-0 flex-1 max-w-md"
                />
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button type="button" data-test="designate-auto-question" :disabled="isRunning || scenariosBusy || promptDirty || !selectedSharedRule"
                  class="rounded-lg border border-primary-300 px-2.5 py-1.5 text-xs text-primary-600 disabled:opacity-50"
                  @click="designateAutomaticQuestion">
                  {{ t(automaticRuleId === selectedSharedRule?.id ? 'admin.accounts.accountTesting.autoQuestionSelected' : 'admin.accounts.accountTesting.setAutoQuestion') }}
                </button>
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
              :rows="5"
              class="text-xs"
              textarea-class="min-h-[112px] max-h-60 text-xs py-1.5 leading-relaxed"
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

      <details v-if="supportsCompact" class="text-xs" data-test="account-test-advanced">
        <summary class="cursor-pointer text-gray-500">{{ t('admin.accounts.accountTesting.advanced') }}</summary>
        <label class="mt-2 flex items-center gap-2">
          <input v-model="compactTest" data-test="account-test-compact" type="checkbox" :disabled="isRunning" />
          {{ t('admin.accounts.openai.testModeCompact') }}
        </label>
      </details>

      <!-- Results Header Bar: Results label, progress stats, history toggle & clear cache -->
      <div class="flex flex-wrap items-center justify-between gap-2 pt-1">
        <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500">
          <span class="text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('admin.accounts.pelicanResultColumn') }}</span>
          <template v-if="isRunning || completedCount > 0">
            <span>{{ t('admin.accounts.accountTesting.completed') }}: {{ completedCount }} / {{ accountStates.length }}</span>
            <span class="text-green-600 font-medium">{{ t('admin.accounts.accountTesting.passed') }}: {{ normalCount }}</span>
            <span v-if="downgradedCount > 0" class="text-amber-600 font-medium">{{ t('admin.accounts.pelicanStatusDowngraded') }}: {{ downgradedCount }}</span>
            <span v-if="failedCount > 0" class="text-red-600 font-medium">{{ t('admin.accounts.pelicanStatusFailed') }}: {{ failedCount }}</span>
          </template>
        </div>
        <div class="flex items-center gap-2">
          <label class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            <span class="font-medium">{{ t('admin.accounts.pelicanHistory') }}</span>
            <select
              data-test="pelican-history-select"
              :value="selectedHistoryKey"
              :aria-label="t('admin.accounts.pelicanHistory')"
              class="w-72 rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs text-gray-700 focus:border-primary-500 focus:outline-none dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200"
              @change="onHistorySelect"
            >
              <option value="">{{ latestResultOptionLabel }}</option>
              <option v-for="entry in historyOptions" :key="historyEntryKey(entry)" :value="historyEntryKey(entry)">
                {{ historyOptionLabel(entry) }}
              </option>
            </select>
          </label>
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

          </div>

            <DetectionResultSummary v-if="item.status !== 'idle' || item.cancelled" :status="item.cancelled ? 'cancelled' : item.status === 'downgraded' ? 'success' : item.status" :capability="isCapabilityResult(item)" :downgraded="item.status === 'downgraded'" :reason="item.reason" :error="item.error" :model="item.responseModel" :latency-ms="item.elapsedMs" :outcomes="item.ruleOutcomes" class="p-3" />
          <RotationProgress :entries="item.logs || []" />

          <div v-if="item.status !== 'idle' || item.cancelled" role="tablist" :aria-label="t('admin.accounts.pelicanResultColumn')" class="mt-2 flex shrink-0 flex-wrap gap-1 border-b border-gray-200 dark:border-dark-600">
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
            <TestStreamOutput v-if="item.activeTab === 'stream'" :reasoning="item.reasoningContent || ''" :content="item.streamingContent" :running="item.status === 'running'" :incomplete="!!item.cancelled || item.status === 'failed'" />
            <div v-else-if="item.activeTab === 'details'" class="space-y-2 rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-900/50">
              <div>{{ t('admin.accounts.accountTesting.type') }}: {{ t(isCapabilityResult(item) ? 'admin.accounts.accountTesting.capability' : 'admin.accounts.accountTesting.connection') }}</div>
              <div>{{ t('admin.accounts.pelicanModel') }}: {{ item.responseModel || item.selectedModel || '-' }}</div>
              <div>{{ t('admin.accounts.pelicanPromptColumn') }}:</div>
              <pre class="max-h-56 overflow-y-auto whitespace-pre-wrap break-words">{{ item.prompt || '-' }}</pre>
            </div>
            <pre v-else-if="item.activeTab === 'output'" data-test="account-test-output" class="max-h-[420px] overflow-y-auto whitespace-pre-wrap break-words rounded-lg bg-gray-900 p-3 font-mono text-xs text-gray-200">{{ item.status === 'running' ? t('admin.accounts.accountTesting.finalPending') : item.cancelled || item.status === 'failed' ? t('admin.accounts.accountTesting.incomplete') : item.streamingContent || item.result?.html || item.reason || t('admin.accounts.detection.connection.success') }}</pre>
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
                class="max-h-[420px] overflow-y-auto whitespace-pre-wrap break-words rounded-lg bg-gray-900 p-2.5 font-mono text-[11px] text-gray-200"
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

            <pre v-else-if="item.status === 'success'" data-test="account-test-output" class="max-h-[420px] overflow-y-auto whitespace-pre-wrap break-words rounded-lg bg-gray-900 p-3 text-xs text-gray-200">{{ item.streamingContent || item.reason || t('admin.accounts.detection.connection.success') }}</pre>

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
              <template v-if="item.testedAt">
                <span class="ml-1 text-[10px] text-gray-400">{{ t('admin.accounts.pelicanUserRating') }}</span>
                <button
                  v-for="option in (['accurate', 'inaccurate'] as const)"
                  :key="option"
                  type="button"
                  :data-test="`pelican-card-rating-${option}`"
                  class="rounded border px-1.5 py-0.5 text-[10px] font-medium transition"
                  :class="item.userRating === option ? ratingButtonActiveClass(option) : ratingButtonIdleClass(option)"
                  @click="rateItemResult(item, option)"
                >
                  {{ option === 'accurate' ? t('admin.accounts.pelicanRatingAccurate') : t('admin.accounts.pelicanRatingInaccurate') }}
                </button>
              </template>
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
import TestStreamOutput from '@/modules/qualitycontrol/TestStreamOutput.vue'
import AccountTestModelPicker from './AccountTestModelPicker.vue'
import { cachedAccountModels, rememberAccountModels } from '@/utils/pelicanModelCatalog'
import DetectionResultSummary from './DetectionResultSummary.vue'
import type { DegradationRule, RuleOutcome } from '@/types'
import RotationProgress from '@/modules/rotationnotice/RotationProgress.vue'
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
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
import { clearPelicanModelHistory, loadPelicanModelHistory, rememberPelicanModel } from '@/utils/pelicanModelHistory'
import {
  clearPelicanResultCache,
  getAllCachedPelicanResults,
  getCachedPelicanResult,
  setCachedPelicanResult,
  setCachedPelicanUserRating,
  type PelicanCachedResult,
  type PelicanUserRating
} from '@/utils/pelicanResultCache'
import { adminAPI } from '@/api'
import { useAppStore } from '@/stores/app'
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
  }>(),
  {
    allAccounts: () => [],
    initialMode: 'default'
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
const appStore = useAppStore()
const showAccountTestHelp = ref(false)
const suppressResultToasts = ref(false)

interface PelicanAccountState {
  account: Account
  status: 'idle' | 'running' | 'success' | 'downgraded' | 'failed'
  streamingContent: string
  reasoningContent?: string
  prompt?: string
  responseModel?: string
  reason?: string
  error?: string
  elapsedMs?: number
  testedAt?: number
  activeTab: 'preview' | 'source' | 'output' | 'details' | 'stream'
  testMode?: 'default' | 'compact' | 'pelican'
  images?: string[]
  logs?: string[]
  cancelled?: boolean
  selectedModel?: string
  availableModels?: PelicanModel[]
  modelsLoaded?: boolean
  modelsLoading?: boolean
  ruleOutcomes?: RuleOutcome[] | null
  userRating?: PelicanUserRating
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
  modelsLoaded: boolean
  modelsLoading: boolean
}

// 全局值仅在非空时批量覆盖；空值表示每个账号使用自身目录中的默认模型。
const selectedModel = ref('')
const pelicanModelHistory = ref(loadPelicanModelHistory())
const compactTest = ref(false)
const probeEvaluator = ref('auto')
const probeExpectedAnswer = ref('')
const validationError = ref('')
const supportsCompact = computed(() => accountStates.value.length > 0 && accountStates.value.every(item => item.account.platform === 'openai'))
const isImageModel = (item: PelicanAccountState) => {
  const model = (selectedModel.value.trim() || item.selectedModel || '').toLowerCase()
  return (item.account.platform === 'openai' && model.startsWith('gpt-image-'))
    || ((item.account.platform === 'gemini' || (item.account.platform === 'antigravity' && item.account.type === 'apikey')) && model.startsWith('gemini-') && model.includes('-image'))
}
const isCapabilityResult = (item: PelicanAccountState) => item.testMode ? item.testMode === 'pelican' : testMode.value === 'pelican'
const getItemHtml = (item: PelicanAccountState | null | undefined): string => {
  if (!item) return ''
  return item.result?.html?.trim() || extractPelicanPreviewHtml(item.streamingContent || '')
}
const hasItemHtml = (item: PelicanAccountState | null | undefined): boolean => {
  return !!getItemHtml(item)
}
const resultTabs = (item: PelicanAccountState): PelicanAccountState['activeTab'][] => {
  if (item.status === 'failed' || item.cancelled) return ['stream', 'details']
  if (hasItemHtml(item)) return item.status === 'running'
    ? ['stream', 'preview', 'source', 'output', 'details']
    : ['preview', 'output', 'stream', 'source', 'details']
  if (item.images?.length) return item.status === 'running'
    ? ['stream', 'preview', 'output', 'details']
    : ['preview', 'output', 'stream', 'details']
  return item.status === 'running' ? ['stream', 'output', 'details'] : ['output', 'stream', 'details']
}
const resultTabLabel = (tab: PelicanAccountState['activeTab']) => t({stream:'admin.accounts.accountTesting.stream',preview:'admin.accounts.pelicanPreviewTab',source:'admin.accounts.pelicanSourceTab',output:'admin.accounts.accountTesting.output',details:'admin.accounts.accountTesting.details'}[tab])
const REASONING_EFFORT_STORAGE_KEY = 'sub2api:account-pelican-reasoning-effort:v1'
const reasoningEffort = ref(localStorage.getItem(REASONING_EFFORT_STORAGE_KEY) || 'low')
watch(reasoningEffort, val => {
  if (val) localStorage.setItem(REASONING_EFFORT_STORAGE_KEY, val)
})
const detectionRules = ref<DegradationRule[]>([])
const scenariosBusy = ref(false)
const scenarioError = ref('')
const automaticRuleId = ref<number | null>(null)
const designateAutomaticQuestion = async () => {
  const rule = selectedSharedRule.value
  if (!rule || promptDirty.value || scenariosBusy.value) return
  scenariosBusy.value = true
  scenarioError.value = ''
  try {
    const settings = await adminAPI.accounts.getRequestQualityProbeSettings()
    if (!rule.enabled) {
      const updated = await adminAPI.degradationRules.update(rule.id, { name: rule.name, prompt: rule.prompt, evaluator: rule.evaluator, params: { ...rule.params }, enabled: true })
      detectionRules.value = detectionRules.value.map(item => item.id === rule.id ? updated : item)
    }
    await adminAPI.accounts.updateRequestQualityProbeSettings({ ...settings, rule_ids: [rule.id] })
    automaticRuleId.value = rule.id
  } catch (error) { scenarioError.value = extractApiErrorMessage(error) }
  finally { scenariosBusy.value = false }
}
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
    const settings = await adminAPI.accounts.getRequestQualityProbeSettings()
    automaticRuleId.value = settings.rule_ids?.length === 1 ? settings.rule_ids[0]! : null
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
const accountModelsLoading = ref(new Set<number>())

// 账号状态会随表格刷新重建，目录请求也可能偶发失败；候选模型取状态里的目录
// 与模块级缓存目录的并集，保证“测试大模型”下拉不会突然变空。
const accountConfiguredModels = computed<PelicanModel[]>(() => {
  const merged = new Map<string, PelicanModel>()
  const states = accountStates.value.length > 0
    ? accountStates.value
    : candidateAccounts.value.map(acc => ({ account: acc } as PelicanAccountState))

  for (const item of states) {
    const stateModels = (item as PelicanStateWithModels).availableModels || []
    const models = stateModels.length > 0 ? stateModels : cachedAccountModels(item.account.id)
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

// 测过的模型（含各账号自身目录里的模型）都进入下拉历史，便于下次直接选回。
const rememberTestedModel = (model: string) => {
  const normalized = model.trim()
  if (!normalized) return
  pelicanModelHistory.value = rememberPelicanModel(normalized)
}

const clearModelHistory = () => {
  clearPelicanModelHistory()
  pelicanModelHistory.value = []
}

const ensureAccountModels = async (item: PelicanAccountState) => {
  const state = item as PelicanStateWithModels
  if (state.modelsLoaded || state.modelsLoading) return
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
    rememberAccountModels(accountID, normalized)
    state.availableModels = normalized.length > 0 ? normalized : cachedAccountModels(accountID)
    // The backend catalog is authoritative. Do not invent a static default when
    // an account exposes no models.
    if (normalized.length > 0 && !state.selectedModel) {
      state.selectedModel = normalized[0].id
    }
  } catch (error) {
    console.error('Failed to load pelican account models:', error)
    // A failed refresh must not empty a catalog that was already usable.
    state.availableModels = cachedAccountModels(accountID)
  } finally {
    const nextLoading = new Set(accountModelsLoading.value)
    nextLoading.delete(accountID)
    accountModelsLoading.value = nextLoading
    state.modelsLoading = false
    state.modelsLoaded = true
  }
}

const hydrateAccountRuntime = async (item: PelicanAccountState) => {
  const state = item as PelicanStateWithModels
  // 先用缓存目录渲染，再由本次会话刷新；刷新失败时保留缓存而不是清空。
  if (!state.modelsLoaded && !state.modelsLoading) {
    await ensureAccountModels(state)
  }
  if (!state.availableModels?.length) {
    state.availableModels = cachedAccountModels(state.account.id)
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
const selectedHistoryKey = ref('')
const allCachedPelicanResults = computed(() =>
  getAllCachedPelicanResults().filter(entry => selectedAccountIds.value.has(entry.account.id))
)
// 历史只看当前测试卡片里的账号，避免选中记录后无处还原。
const historyOptions = computed(() => {
  const visible = new Set(accountStates.value.map(item => item.account.id))
  return allCachedPelicanResults.value.filter(entry => visible.has(entry.account.id))
})
const historyEntryKey = (entry: PelicanCachedResult) => entry.account.id + ':' + entry.testedAt
const HISTORY_PROMPT_PREVIEW_CHARS = 24
// 历史下拉用题目开头区分记录：账号已在结果卡片里，重复显示账号没有信息量。
const historyPromptPreview = (prompt?: string) => {
  const text = (prompt || '').replace(/\s+/g, ' ').trim()
  if (!text) return t('admin.accounts.pelicanPromptMissing')
  return text.length > HISTORY_PROMPT_PREVIEW_CHARS ? text.slice(0, HISTORY_PROMPT_PREVIEW_CHARS) + '…' : text
}
// “最新结果”也要带时间，否则和历史条目混在同一个下拉里无法比较先后。
const latestResultOptionLabel = computed(() => {
  const times = [
    ...historyOptions.value.map(entry => entry.testedAt),
    ...accountStates.value.map(item => item.testedAt).filter((value): value is number => typeof value === 'number')
  ]
  const latest = times.length ? Math.max(...times) : 0
  return latest
    ? t('admin.accounts.pelicanHistoryLatest') + ' · ' + new Date(latest).toLocaleString()
    : t('admin.accounts.pelicanHistoryLatest')
})
const historyOptionLabel = (entry: PelicanCachedResult) =>
  historyPromptPreview(entry.prompt) + ' · ' + (entry.responseModel || '-') + ' · ' + historyStatusLabel(entry) + ' · ' + formatTestedAt(entry.testedAt)

// 下拉选中一条历史后，直接把它还原进结果卡片，和刚测完时完全一致。
const applyHistorySelection = (key: string) => {
  selectedHistoryKey.value = key
  if (!key) {
    // 回到“最新结果”：把每张卡片还原成该账号最近一次测智。
    for (const item of accountStates.value) {
      if (item.status === 'running') continue
      const latest = getCachedPelicanResult(item.account.id)
      if (latest) applyCachedResult(item, latest)
    }
    return
  }
  const entry = historyOptions.value.find(item => historyEntryKey(item) === key)
  if (!entry) return
  const target = accountStates.value.find(item => item.account.id === entry.account.id)
  if (!target || target.status === 'running') return
  applyCachedResult(target, entry)
}
const onHistorySelect = (event: Event) => applyHistorySelection((event.target as HTMLSelectElement).value)

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
    reasoning: item.reasoningContent,
    ruleOutcomes: item.ruleOutcomes || undefined,
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
  item.reasoningContent = cached.reasoning || ''
  item.ruleOutcomes = cached.ruleOutcomes
  item.userRating = cached.userRating
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
  item.activeTab = item.status === 'failed' ? 'stream' : (html || cached.images?.length) ? 'preview' : 'output'
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

const historyStatusLabel = (entry: PelicanCachedResult) => {
  if (entry.status === 'success') return t(entry.testMode === 'default' || entry.testMode === 'compact' ? 'admin.accounts.detection.connection.success' : 'admin.accounts.pelicanStatusSuccess')
  if (entry.status === 'downgraded') return t('admin.accounts.pelicanStatusDowngraded')
  return t('admin.accounts.pelicanStatusFailed')
}

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

// 评价绑定到该结果的缓存条目（按账号 + testedAt），实时结果和历史结果共用。
const rateItemResult = (item: PelicanAccountState, rating: PelicanUserRating) => {
  if (!item.testedAt) return
  const updated = setCachedPelicanUserRating(item.account.id, item.testedAt, rating)
  item.userRating = updated ? updated.userRating : item.userRating
}

const formatTestedAt = (testedAt: number) => new Date(testedAt).toLocaleString()

const handleClearCache = () => {
  if (isRunning.value) return
  clearPelicanResultCache()
  selectedHistoryKey.value = ''
  for (const item of accountStates.value) {
    item.status = 'idle'
    item.cancelled = false
    item.streamingContent = ''
    item.reasoningContent = ''
    item.result = undefined
    item.ruleOutcomes = undefined
    item.userRating = undefined
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
  showAccountTestHelp.value = false
  rememberSelectedModel()
  closeLightbox()
  showAccountPicker.value = false
  emit('close')
}

async function testSingleAccount(item: PelicanAccountState, signal: AbortSignal) {
  const overrideModel = selectedModel.value?.trim() || ''
  const testModel = overrideModel || item.selectedModel?.trim() || ''
  rememberTestedModel(testModel)
  if (!testModel) {
    item.testMode = testMode.value
    item.result = undefined
    item.reason = undefined
    item.streamingContent = ''
    item.reasoningContent = ''
    item.activeTab = 'stream'
    item.status = 'failed'
    item.error = t('admin.accounts.pelicanModelsUnavailable')
    return
  }
  const capability = !isImageModel(item) && testMode.value === 'pelican'
  const requestMode = capability ? 'pelican' : compactTest.value && supportsCompact.value ? 'compact' : 'default'
  const sharedRules = capability && isKeywordQuestion.value
  const testPrompt = !capability
    ? (isImageModel(item) ? customPrompt.value.trim() : '')
    : customPrompt.value.trim() || t('admin.accounts.pelicanPromptDefault')
  const selectedRulePrompt = isKeywordQuestion.value ? selectedSharedRule.value?.prompt : undefined
  const effectiveTestPrompt = selectedRulePrompt || testPrompt
  item.cancelled = false
  item.testMode = requestMode
  item.images = []
  item.logs = []
  item.activeTab = 'stream'
  item.prompt = effectiveTestPrompt
  item.status = 'running'
  item.streamingContent = ''
  item.reasoningContent = ''
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
      item.activeTab = item.status === 'failed' ? 'stream' : html ? 'preview' : 'output'
      return
    }
    const requestBody: {
      model_id: string
      prompt: string
      mode: string
      reasoning_effort?: string
      evaluator?: string
      expected_answer?: number
      rule_id?: number
    } = {
      model_id: testModel,
      prompt: testPrompt,
      mode: requestMode,
      ...(capability ? {rule_id: !promptDirty.value ? selectedSharedRule.value?.id : undefined, reasoning_effort: reasoningEffort.value || 'low', evaluator: probeEvaluator.value, ...(probeEvaluator.value === 'numeric' ? {expected_answer: Number(probeExpectedAnswer.value)} : {})} : {})
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
      else if (event.type === 'reasoning' && event.text) item.reasoningContent = (item.reasoningContent || '') + event.text
      else if (event.type === 'content' && event.text) item.streamingContent += event.text
      else if ((event.type === 'status' || event.type === 'routing_progress') && event.text) item.logs!.push(event.text)
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
        item.activeTab = (item.images?.length || hasItemHtml(item)) ? 'preview' : 'output'
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
      // The server alone persists eligible quality results. Other questions
      // must never overwrite the account marker through a second UI request.
      if (capability && !promptDirty.value && selectedSharedRule.value?.id === automaticRuleId.value
        && (item.status === 'success' || item.status === 'downgraded')) {
        emit('updated', { id: item.account.id, downgraded: item.status === 'downgraded' })
      }
    }
  }
}

function notifyTestResult(item: PelicanAccountState) {
  const name = item.account.name
  if (item.status === 'success') {
    appStore.showSuccess(t('admin.accounts.accountTesting.notifySuccess', { name }))
  } else if (item.status === 'downgraded') {
    appStore.showWarning(t('admin.accounts.accountTesting.notifyDowngraded', { name }))
  } else if (item.status === 'failed') {
    appStore.showError(t('admin.accounts.accountTesting.notifyFailed', { name }))
  }
}

function notifyBatchResult(items: PelicanAccountState[]) {
  const passed = items.filter(item => item.status === 'success').length
  const downgraded = items.filter(item => item.status === 'downgraded').length
  const failed = items.filter(item => item.status === 'failed').length
  if (passed + downgraded + failed === 0) return
  const message = t('admin.accounts.accountTesting.notifyBatch', {
    total: passed + downgraded + failed,
    passed,
    downgraded,
    failed
  })
  if (failed > 0) appStore.showError(message)
  else if (downgraded > 0) appStore.showWarning(message)
  else appStore.showSuccess(message)
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
  // 新一次测智会刷新结果卡片，历史下拉回到“最新结果”。
  selectedHistoryKey.value = ''
  isRunning.value = true
  emit('running-change', true)
  try {
    await testSingleAccount(item, controller.signal)
  } finally {
    accountControllers.delete(item.account.id)
    isRunning.value = accountControllers.size > 0
    emit('running-change', isRunning.value)
    if (!suppressResultToasts.value) notifyTestResult(item)
  }
}

async function startAllTests() {
  if (isRunning.value) return
  const items = [...accountStates.value]
  if (items.length <= 1) {
    await Promise.all(items.map(item => runSingleTest(item)))
    return
  }
  suppressResultToasts.value = true
  try {
    await Promise.all(items.map(item => runSingleTest(item)))
  } finally {
    suppressResultToasts.value = false
    notifyBatchResult(items)
  }
}

onUnmounted(() => {
  stopAllTests()
  closeLightbox()
})
</script>
