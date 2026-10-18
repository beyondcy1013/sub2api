<template>
  <div class="flex flex-wrap items-center gap-3 max-lg:min-w-0 max-lg:gap-2">
    <slot name="before"></slot>
    <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary max-lg:min-h-[44px]" :aria-label="t('common.refresh')" :title="t('common.refresh')">
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </button>
    <button
      type="button"
      @click="$emit('pelican-test')"
      class="btn btn-secondary gap-2 px-3 text-cyan-600 hover:text-cyan-700 dark:text-cyan-400 hidden lg:inline-flex"
      :title="t('admin.accounts.accountTesting.title')"
    >
      <Icon name="play" size="sm" />
      <span>{{ t('admin.accounts.accountTesting.title') }}</span>
    </button>
    <button type="button" data-test="account-detection-entry" @click="$emit('detection')" class="btn btn-secondary gap-2 px-3 hidden lg:inline-flex"><Icon name="play" size="sm" /><span>{{ t('admin.accounts.detection.title') }}</span></button>
    <button type="button" data-test="automatic-detection-entry" @click="$emit('automatic-detection')" class="btn btn-secondary gap-2 px-3 hidden lg:inline-flex"><Icon name="refresh" size="sm" /><span>{{ t('automaticDetection.title') }}</span></button>
    <button data-test="account-scheduling-rules" @click="$emit('scheduling-rules')" class="btn btn-secondary gap-2 px-3 hidden lg:inline-flex">
      <Icon name="cog" size="sm" />
      <span>{{ t('admin.accounts.schedulingRules.title') }}</span>
    </button>
    <slot name="after"></slot>
    <slot name="beforeCreate"></slot>
    <button @click="$emit('create')" class="btn btn-primary max-lg:min-h-[44px]">{{ t('admin.accounts.createAccount') }}</button>
    <button
      @click="$emit('toggleRecycled')"
      class="btn px-3 hidden lg:inline-flex"
      :class="{ 'btn-primary': recycled }"
      :title="recycled ? t('admin.accounts.viewNormal') : t('admin.accounts.viewRecycled')"
    >
      <Icon name="archive" size="md" />
    </button>
    <button
      @click="$emit('toggleDeleted')"
      class="btn px-3 hidden lg:inline-flex"
      :class="{ 'btn-primary': deleted }"
      :title="deleted ? t('admin.accounts.viewNormal') : t('admin.accounts.viewDeleted')"
    >
      <Icon name="trash" size="md" />
    </button>
    <slot name="afterCreate"></slot>
    <button
      @click="$emit('toggleFilters')"
      class="btn btn-secondary px-3 max-lg:min-h-[44px]"
      :class="{ 'btn-primary': showFilters }"
      :title="t('admin.accounts.toggleFilters')"
      :aria-label="t('admin.accounts.toggleFilters')"
    >
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M6 12h12m-9 5.25h6" />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps(['loading', 'showFilters', 'recycled', 'deleted'])
defineEmits(['refresh', 'create', 'toggleFilters', 'toggleRecycled', 'toggleDeleted', 'scheduling-rules', 'pelican-test', 'detection', 'automatic-detection'])

const { t } = useI18n()
</script>
