<template>
  <div class="flex flex-wrap items-center gap-3">
    <slot name="before"></slot>
    <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary">
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </button>
    <button @click="$emit('scheduling-rules')" class="btn btn-secondary gap-2 px-3">
      <Icon name="cog" size="sm" />
      <span>{{ t('admin.accounts.schedulingRules.title') }}</span>
    </button>
    <slot name="after"></slot>
    <slot name="beforeCreate"></slot>
    <button @click="$emit('create')" class="btn btn-primary">{{ t('admin.accounts.createAccount') }}</button>
    <button
      @click="$emit('toggleRecycled')"
      class="btn px-3"
      :class="{ 'btn-primary': recycled }"
      :title="recycled ? t('admin.accounts.viewNormal') : t('admin.accounts.viewRecycled')"
    >
      <Icon name="inbox" size="md" />
    </button>
    <button
      @click="$emit('viewTrash')"
      class="btn btn-secondary px-3"
      :title="t('admin.accounts.trashBin')"
    >
      <Icon name="trash" size="md" />
    </button>
    <slot name="afterCreate"></slot>
    <button
      @click="$emit('toggleFilters')"
      class="btn btn-secondary px-3"
      :class="{ 'btn-primary': showFilters }"
      :title="t('admin.accounts.toggleFilters')"
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

defineProps(['loading', 'showFilters', 'recycled'])
defineEmits(['refresh', 'create', 'toggleFilters', 'toggleRecycled', 'viewTrash', 'scheduling-rules'])

const { t } = useI18n()
</script>
