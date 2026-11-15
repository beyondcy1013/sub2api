<template>
  <section class="min-w-0 space-y-2 rounded-lg border border-gray-200 p-3 text-xs dark:border-dark-600" data-test="test-stream-output">
    <div class="flex flex-wrap items-center justify-between gap-2 text-gray-600 dark:text-gray-300">
      <span role="status">{{ t(`admin.accounts.accountTesting.${running ? 'streaming' : incomplete ? 'interrupted' : 'finished'}`) }} · {{ t('admin.accounts.accountTesting.characters', { reasoning: reasoning.length, answer: content.length }) }}</span>
      <div class="flex gap-3">
        <button type="button" class="text-primary-600 dark:text-primary-400" @click="follow = true; scrollToEnd()">{{ t('admin.accounts.accountTesting.follow') }}</button>
        <button type="button" class="text-primary-600 dark:text-primary-400" @click="copyOutput">{{ t('admin.accounts.copyOutput') }}</button>
      </div>
    </div>
    <div ref="viewport" class="max-h-[420px] overflow-y-auto overscroll-contain rounded-lg bg-gray-900 p-3 font-mono" @scroll="onScroll">
      <details open class="mb-3 text-gray-400">
        <summary class="cursor-pointer">{{ t('admin.accounts.accountTesting.reasoning') }}</summary>
        <pre data-test="test-stream-reasoning" class="mt-2 whitespace-pre-wrap break-words">{{ reasoning || t('admin.accounts.accountTesting.waitingReasoning') }}</pre>
      </details>
      <div class="mb-2 text-gray-400">{{ t('admin.accounts.accountTesting.answer') }}</div>
      <pre data-test="test-stream-answer" class="whitespace-pre-wrap break-words text-green-300">{{ content || t('admin.accounts.accountTesting.waitingAnswer') }}<span v-if="running" aria-hidden="true" class="animate-pulse">▌</span></pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
const props = defineProps<{ reasoning: string; content: string; running: boolean; incomplete?: boolean }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
function copyOutput() {
  copyToClipboard([
    t('admin.accounts.accountTesting.reasoning'), props.reasoning,
    t('admin.accounts.accountTesting.answer'), props.content
  ].join('\n'), t('admin.accounts.outputCopied'))
}
const viewport = ref<HTMLElement | null>(null)
const follow = ref(true)
function onScroll() {
  const el = viewport.value
  if (el) follow.value = el.scrollHeight - el.scrollTop - el.clientHeight < 32
}
async function scrollToEnd() {
  await nextTick()
  if (follow.value && viewport.value) viewport.value.scrollTop = viewport.value.scrollHeight
}
watch(() => [props.reasoning, props.content], scrollToEnd, { immediate: true })
</script>
