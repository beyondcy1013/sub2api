<template>
  <component
    :is="mobile ? MobileActionSheet : 'div'"
    v-bind="mobile ? { modelValue: show, title: t('admin.accounts.columns.actions') } : {}"
    @update:model-value="onSheetVisibility"
  >
    <Teleport to="body" :disabled="mobile">
      <div v-if="show && (mobile || position)">
        <!-- Backdrop: click anywhere outside to close (desktop only; the sheet has its own overlay) -->
        <div v-if="!mobile" class="fixed inset-0 z-[9998]" @click="emit('close')"></div>
        <div
          :class="mobile
            ? 'account-mobile-actions'
            : 'action-menu-content fixed z-[9999] max-h-[calc(100vh-1rem)] w-[7.8rem] overflow-x-hidden overflow-y-auto rounded-xl bg-white shadow-lg ring-1 ring-black/5 dark:bg-dark-800'"
          :style="!mobile && position ? { top: position.top + 'px', left: position.left + 'px' } : undefined"
        >
          <div class="py-1">
            <template v-if="account">
              <button v-if="account.extra?.codex_local_sync === true && !deleted" @click="runAction('restore-codex', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                恢复到官方 Codex 登录
              </button>
              <template v-if="mobile">
                <div class="mb-2 break-all px-3 text-sm font-medium" data-test="account-row-actions-identity">
                  {{ account.name }} <span class="text-xs text-gray-500">#{{ account.id }}</span>
                </div>
                <button data-test="account-edit-action" @click="runAction('edit', account)" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                  {{ t('common.edit') }}
                </button>
                <button data-test="account-test-action" @click="runAction('test', account)" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                  {{ t('admin.accounts.accountTesting.action') }}
                </button>
                <button v-if="account.platform === 'openai' && !account.parent_account_id && !deleted && !recycled" data-test="account-harvest-action" title="手动打票" @click="runAction('harvest', account)" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-violet-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  打票
                </button>
                <button v-if="deleted" data-test="account-restore-deleted-action" @click="runAction('restore-deleted', account)" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-emerald-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  {{ t('admin.accounts.restoreDeleted') }}
                </button>
                <button v-else-if="recycled" data-test="account-restore-action" @click="runAction('restore', account)" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-emerald-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  {{ t('admin.accounts.restore') }}
                </button>
                <button v-else data-test="account-recycle-action" @click="runAction('recycle', account)" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-amber-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  {{ t('admin.accounts.recycle') }}
                </button>
                <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
              </template>
              <template v-if="!account.extra?.fixed_codex">
                <button v-if="!isPinned" @click="runAction('pin', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-amber-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  <Icon name="arrowUp" size="sm" class="text-amber-500" />
                  {{ t('admin.accounts.pin') }}
                </button>
                <button v-else @click="runAction('unpin', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-amber-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  <Icon name="arrowUp" size="sm" class="text-amber-500" />
                  {{ t('admin.accounts.unpin') }}
                </button>
              </template>
              <button @click="runAction('schedule', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="clock" size="sm" class="text-orange-500" />
                {{ t('admin.scheduledTests.schedule') }}
              </button>
              <button
                v-if="supportsStateProtection"
                data-test="account-state-protection-action"
                @click="runAction('toggle-state-protection', account)"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm"
                :class="isStateProtected ? 'text-emerald-600 hover:bg-gray-100 dark:hover:bg-dark-700' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
                :disabled="stateProtectionBusy === account.id"
              >
                <Icon name="shield" size="sm" :class="isStateProtected ? 'text-emerald-500' : 'text-gray-400'" />
                {{ isStateProtected ? t('admin.accounts.stateProtection.disable') : t('admin.accounts.stateProtection.enable') }}
              </button>
              <button v-if="canDuplicate" @click="runAction('duplicate', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="copy" size="sm" class="text-sky-500" />
                {{ t('admin.accounts.duplicateAccount') }}
              </button>
              <button v-if="account.type === 'apikey' && !isDeletedStaging" @click="runAction('query-balance', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="dollar" size="sm" class="text-emerald-600" />
                {{ t('admin.accounts.balanceQuery.action') }}
              </button>
              <button v-if="canReceiveStickySessions" @click="runAction('sticky-sessions', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="sync" size="sm" class="text-cyan-600" />
                {{ t('admin.accounts.stickySessions.action') }}
              </button>
              <!-- 影子账号不持凭据:重授权/刷新 token 对其无效(后端拒绝),故隐藏(外审 G4)。 -->
              <template v-if="(account.type === 'oauth' || account.type === 'setup-token') && !isShadow">
                <button @click="runAction('reauth', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-blue-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  <Icon name="link" size="sm" />
                  {{ t('admin.accounts.reAuthorize') }}
                </button>
                <button @click="runAction('refresh-token', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-purple-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                  <Icon name="refresh" size="sm" />
                  {{ t('admin.accounts.refreshToken') }}
                </button>
              </template>
              <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
              <button @click="runAction('recover-state', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-emerald-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="sync" size="sm" />
                {{ t('admin.accounts.recoverState') }}
              </button>
              <button @click="runAction('scheduled-action', account, 'enable_and_recover')" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm leading-5 text-emerald-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="clock" size="sm" />
                {{ t('admin.accounts.scheduledAction.enableAndRecover') }}
              </button>
              <button @click="runAction('scheduled-action', account, 'pause')" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm leading-5 text-amber-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="clock" size="sm" />
                {{ t('admin.accounts.scheduledAction.pause') }}
              </button>
              <button v-if="hasQuotaLimit" @click="runAction('reset-quota', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-teal-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="refresh" size="sm" />
                {{ t('admin.accounts.resetQuota') }}
              </button>
              <button v-if="!isDeletedStaging && !account.extra?.fixed_codex" data-test="account-delete-action" @click="runAction('delete', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20">
                <Icon name="trash" size="sm" />
                {{ t('common.delete') }}
              </button>
              <button v-if="isDeletedStaging && !account.extra?.fixed_codex" data-test="account-permanent-delete-action" @click="runAction('permanent-delete', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20">
                <Icon name="trash" size="sm" />
                {{ t('admin.accounts.permanentDelete') }}
              </button>
              <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>
              <button @click="runAction('stats', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="chart" size="sm" class="text-indigo-500" />
                {{ t('admin.accounts.viewStats') }}
              </button>
              <button v-if="isOpenAIOAuthParent" @click="runAction('create-spark-shadow', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-amber-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="sparkles" size="sm" />
                {{ t('admin.accounts.createSparkShadow') }}
              </button>
              <button v-if="supportsPrivacy" @click="runAction('set-privacy', account)" class="flex w-full items-center gap-2 px-4 py-2 text-sm text-emerald-600 hover:bg-gray-100 dark:hover:bg-dark-700">
                <Icon name="shield" size="sm" />
                {{ t('admin.accounts.setPrivacy') }}
              </button>
            </template>
          </div>
        </div>
      </div>
    </Teleport>
  </component>
</template>

<script setup lang="ts">
import { computed, watch, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import MobileActionSheet from '@/components/common/MobileActionSheet.vue'
import type { Account } from '@/types'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const props = withDefaults(defineProps<{
  show: boolean
  account: Account | null
  position: { top: number; left: number } | null
  mobile?: boolean
  recycled?: boolean
  deleted?: boolean
  stateProtectionBusy?: number | null
}>(), {
  mobile: false,
  recycled: false,
  deleted: false,
  stateProtectionBusy: null
})
const emit = defineEmits(['restore-codex', 'close', 'edit', 'test', 'harvest', 'recycle', 'restore', 'restore-deleted', 'stats', 'schedule', 'toggle-state-protection', 'duplicate', 'query-balance', 'sticky-sessions', 'reauth', 'refresh-token', 'recover-state', 'scheduled-action', 'reset-quota', 'set-privacy', 'create-spark-shadow', 'delete', 'permanent-delete', 'pin', 'unpin'])
const { t } = useI18n()

// Mobile sheet closes first so its focus restoration finishes before the
// downstream dialog opens; desktop keeps the historical emit-then-close order.
const runAction = async (name: Parameters<typeof emit>[0], ...args: unknown[]) => {
  if (props.mobile) {
    emit('close')
    await nextTick()
    emit(name, ...args)
  } else {
    emit(name, ...args)
    emit('close')
  }
}

const onSheetVisibility = (visible: boolean) => {
  if (!visible) emit('close')
}

const isPinned = computed(() => Boolean(props.account?.extra?.pinned))
const canDuplicate = computed(() => {
  if (!props.account || props.account.parent_account_id != null) return false
  return ['apikey', 'upstream', 'bedrock', 'service_account'].includes(props.account.type)
})
const isAntigravityOAuth = computed(() => props.account?.platform === 'antigravity' && props.account?.type === 'oauth')
const isOpenAIOAuth = computed(() => props.account?.platform === 'openai' && props.account?.type === 'oauth')
const isOpenAI = computed(() => props.account?.platform === 'openai')
const isDeletedStaging = computed(() => props.account?.extra?.deleted === true)
const canReceiveStickySessions = computed(() =>
	isFeatureFlagEnabled(FeatureFlags.stickySessionReassignment) &&
	isOpenAI.value &&
	!isDeletedStaging.value &&
  props.account?.status === 'active' &&
  props.account?.schedulable === true
)
// 影子账号(链接型,持 parent_account_id)不持凭据、type 不可变,凭据/隐私类操作对其无效。
const isShadow = computed(() => props.account?.parent_account_id != null)
// A "parent" OpenAI OAuth account is one that is NOT itself a shadow (parent_account_id == null)
const isOpenAIOAuthParent = computed(() => isOpenAIOAuth.value && !isShadow.value)
const supportsPrivacy = computed(() => (isAntigravityOAuth.value || isOpenAIOAuth.value) && !isShadow.value)
const supportsStateProtection = computed(() => isOpenAIOAuthParent.value)
const isStateProtected = computed(() => Boolean(props.account?.extra?.state_protection_enabled))
const hasQuotaLimit = computed(() => {
  return (props.account?.type === 'apikey' || props.account?.type === 'bedrock') && (
    (props.account?.quota_limit ?? 0) > 0 ||
    (props.account?.quota_daily_limit ?? 0) > 0 ||
    (props.account?.quota_weekly_limit ?? 0) > 0
  )
})

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') emit('close')
}

// In mobile mode MobileActionSheet owns keydown/overlay handling; this watcher
// only binds the desktop floating menu listener.
watch(
  () => props.show && !props.mobile,
  (visible) => {
    if (visible) {
      window.addEventListener('keydown', handleKeydown)
    } else {
      window.removeEventListener('keydown', handleKeydown)
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.account-mobile-actions button {
  min-height: 44px;
  white-space: normal;
  text-align: left;
}

.account-mobile-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
