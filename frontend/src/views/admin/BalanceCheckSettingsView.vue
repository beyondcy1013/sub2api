<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <form v-else class="space-y-6" @submit.prevent="saveSettings">
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">余额检测设置</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  控制后台定时余额检测、余额下降暂停和低余额暂停规则。
                </p>
              </div>
              <div class="flex items-center gap-3 rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
                <span class="text-sm text-gray-700 dark:text-gray-300">启用检测</span>
                <Toggle v-model="form.enabled" />
              </div>
            </div>
          </div>

          <div class="space-y-5 p-6">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <label class="input-label" for="balance-check-interval">检测间隔</label>
                <input
                  id="balance-check-interval"
                  v-model.trim="form.interval"
                  class="input w-full"
                  placeholder="@every 5m"
                />
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">示例：@every 5m、@every 1h。</p>
              </div>

              <div>
                <label class="input-label" for="balance-check-url">余额查询地址</label>
                <input
                  id="balance-check-url"
                  v-model.trim="form.balance_url"
                  class="input w-full"
                  placeholder="https://..."
                />
              </div>

              <div>
                <label class="input-label" for="balance-check-timeout">请求超时（秒）</label>
                <input
                  id="balance-check-timeout"
                  v-model.number="form.request_timeout_seconds"
                  type="number"
                  min="1"
                  step="1"
                  class="input w-full"
                />
              </div>

              <div>
                <label class="input-label" for="balance-check-concurrency">最大并发检测数</label>
                <input
                  id="balance-check-concurrency"
                  v-model.number="form.max_concurrent_checks"
                  type="number"
                  min="1"
                  step="1"
                  class="input w-full"
                />
              </div>
            </div>

            <div class="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800/60 dark:bg-blue-900/20">
              <div class="flex items-start gap-3">
                <Icon name="infoCircle" size="md" class="mt-0.5 flex-shrink-0 text-blue-500" />
                <p class="text-sm text-blue-700 dark:text-blue-300">
                  配置保存到运行目录的 config.yaml。保存后需要重启服务，定时检测任务才会读取新配置。
                </p>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">自动停止条件</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              命中后账号会停止调度，不按周期时长自动恢复。数值为 0 的条件表示不启用。
            </p>
          </div>

          <div class="space-y-5 p-6">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <label class="input-label" for="stop-below">余额低于此值停止</label>
                <input
                  id="stop-below"
                  v-model.number="form.stop_when_current_below"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-full"
                />
              </div>

              <div>
                <label class="input-label" for="resume-above">余额恢复到此值自动恢复</label>
                <input
                  id="resume-above"
                  v-model.number="form.resume_when_current_above"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-full"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">周期暂停条件</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              任一条件命中时，账号会暂停指定时长。数值为 0 的条件表示不启用。
            </p>
          </div>

          <div class="space-y-5 p-6">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <label class="input-label" for="pause-duration">暂停时长（小时）</label>
                <input
                  id="pause-duration"
                  v-model.number="form.pause_duration_hours"
                  type="number"
                  min="0.1"
                  step="0.1"
                  class="input w-full"
                />
              </div>

              <div>
                <label class="input-label" for="min-decrease">余额下降阈值</label>
                <input
                  id="min-decrease"
                  v-model.number="form.min_decrease"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-full"
                />
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">当前余额相比上次检测下降达到该值时暂停。</p>
              </div>

              <div>
                <label class="input-label" for="pause-below">当前余额低于</label>
                <input
                  id="pause-below"
                  v-model.number="form.pause_when_current_below"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-full"
                />
              </div>

              <div>
                <label class="input-label" for="drop-percent">余额下降百分比</label>
                <input
                  id="drop-percent"
                  v-model.number="form.pause_when_drop_percent"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-full"
                />
              </div>
            </div>

            <div class="flex items-center justify-between gap-4 rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700">
              <div>
                <div class="text-sm font-medium text-gray-900 dark:text-white">仅检测设置了小时限额的账号</div>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  关闭后，未设置小时限额的账号也会参与余额检测。
                </p>
              </div>
              <Toggle v-model="form.require_quota_hourly_limit" />
            </div>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="min-w-0">
              <div class="text-sm font-medium text-gray-900 dark:text-white">配置文件</div>
              <code class="mt-1 block truncate text-xs text-gray-500 dark:text-gray-400">{{ configPath || '-' }}</code>
              <p v-if="restartRequired" class="mt-2 text-sm text-amber-600 dark:text-amber-400">
                已保存，重启 sub2freeApi.service 后生效。
              </p>
            </div>

            <div class="flex flex-wrap gap-2">
              <button type="button" class="btn btn-secondary" :disabled="loading || saving" @click="loadSettings">
                刷新
              </button>
              <button type="submit" class="btn btn-primary" :disabled="saving">
                {{ saving ? '保存中...' : '保存设置' }}
              </button>
            </div>
          </div>
        </div>
      </form>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'
import balanceCheckSettingsAPI, { type BalanceCheckConfig } from '@/api/admin/balanceCheckSettings'

const appStore = useAppStore()

const defaultForm = (): BalanceCheckConfig => ({
  enabled: true,
  interval: '@every 5m',
  balance_url: 'https://ai.router.team/api/public/cc-switch/balance',
  request_timeout_seconds: 30,
  max_concurrent_checks: 1,
  pause_duration_hours: 5,
  min_decrease: 5,
  pause_when_current_below: 0,
  pause_when_drop_percent: 0,
  stop_when_current_below: 0,
  resume_when_current_above: 0,
  require_quota_hourly_limit: true
})

const form = reactive<BalanceCheckConfig>(defaultForm())
const loading = ref(true)
const saving = ref(false)
const restartRequired = ref(false)
const configPath = ref('')

function applyConfig(config: BalanceCheckConfig) {
  Object.assign(form, config)
}

function validateForm(): string | null {
  if (!form.interval.trim()) return '检测间隔不能为空'
  if (!form.balance_url.trim()) return '余额查询地址不能为空'
  if (form.request_timeout_seconds <= 0) return '请求超时必须大于 0'
  if (form.max_concurrent_checks <= 0) return '最大并发检测数必须大于 0'
  if (form.pause_duration_hours <= 0) return '暂停时长必须大于 0'
  if (form.min_decrease < 0) return '余额下降阈值不能小于 0'
  if (form.pause_when_current_below < 0) return '当前余额阈值不能小于 0'
  if (form.pause_when_drop_percent < 0) return '下降百分比不能小于 0'
  if (form.stop_when_current_below < 0) return '自动停止余额阈值不能小于 0'
  if (form.resume_when_current_above < 0) return '自动恢复余额阈值不能小于 0'
  return null
}

async function loadSettings() {
  loading.value = true
  try {
    const result = await balanceCheckSettingsAPI.get()
    applyConfig(result.config)
    configPath.value = result.config_path
    restartRequired.value = result.restart_required
  } catch (error: any) {
    appStore.showError(error?.message || '余额检测设置加载失败')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  const validationError = validateForm()
  if (validationError) {
    appStore.showError(validationError)
    return
  }

  saving.value = true
  try {
    const result = await balanceCheckSettingsAPI.update({ ...form })
    applyConfig(result.config)
    configPath.value = result.config_path
    restartRequired.value = result.restart_required
    appStore.showSuccess('余额检测设置已保存')
  } catch (error: any) {
    appStore.showError(error?.message || '余额检测设置保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
