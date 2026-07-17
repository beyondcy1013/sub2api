<template>
  <a
    class="inline-flex min-h-6 min-w-0 max-w-full items-center gap-1 text-xs font-normal text-primary-600 transition-colors hover:underline focus:outline-none focus:ring-2 focus:ring-primary-400 focus:ring-offset-2 dark:text-primary-400 dark:focus:ring-offset-dark-800"
    href="https://pay.ldxp.cn/shop/99739964"
    target="_blank"
    rel="noopener noreferrer"
    :title="t('admin.proxies.ad.details')"
    :aria-label="`${t('admin.proxies.ad.inline')} ${t('admin.proxies.ad.details')}`"
  >
    <span class="min-w-0 truncate">{{ t('admin.proxies.ad.inline') }}</span>
    <Icon name="externalLink" size="xs" class="shrink-0" />
  </a>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
</script>
