import { apiClient } from '../client'

/**
 * 超级优先模式状态（镜像后端 super_priority_settings_response）。
 * mode: "normal"（通常） / "super_priority"（超级优先激活）。
 * failure_threshold: 一分钟滚动窗口内允许的失败次数，达到即自动降级（默认 2，可配置）。
 * base_strategy: 超级优先外层之下的基础调度策略。
 */
export interface SuperPrioritySettings {
  mode: 'normal' | 'super_priority'
  base_strategy: 'default' | 'lowest_cost'
  failure_threshold: number
  check_interval: string
  liveness_include_unschedulable: boolean
  test_model_id: string
  test_prompt: string
  activated_at: string
  demoted_at: string
  is_active: boolean
  liveness_runtime: SchedulingLivenessRuntimeStatus
}

export interface SuperPriorityRuntimeParams {
  base_strategy: 'default' | 'lowest_cost'
  failure_threshold: number
  check_interval: string
  liveness_include_unschedulable: boolean
  test_model_id: string
  test_prompt: string
}

export interface SuperPriorityActivateResult {
  message: string
}

export interface SchedulingRefreshCounts {
  checked: number
  succeeded: number
  failed: number
  skipped: number
}

export interface SchedulingLivenessRunStatus {
  trigger: 'scheduled' | 'manual'
  started_at: string
  finished_at: string
  result: SchedulingRefreshCounts
  error?: string
}

export interface SchedulingLivenessRuntimeStatus {
  enabled: boolean
  running: boolean
  next_run_at?: string
  last_run?: SchedulingLivenessRunStatus
  history?: SchedulingLivenessRunStatus[]
}

export interface SchedulingRulesRuntimeStatus {
  liveness: SchedulingLivenessRuntimeStatus
  upstream_billing: SchedulingLivenessRuntimeStatus
}

export interface SchedulingRulesRefreshResult {
  liveness: SchedulingRefreshCounts
  upstream_billing: SchedulingRefreshCounts
}

const superPriorityAPI = {
  async get(): Promise<SuperPrioritySettings> {
    const { data } = await apiClient.get<SuperPrioritySettings>('/admin/settings/super-priority')
    return data
  },

  async update(params: SuperPriorityRuntimeParams): Promise<{ message: string; restart_required: boolean }> {
    const { data } = await apiClient.put<{ message: string; restart_required: boolean }>(
      '/admin/settings/super-priority',
      params,
    )
    return data
  },

  async activate(): Promise<SuperPriorityActivateResult> {
    const { data } = await apiClient.post<SuperPriorityActivateResult>(
      '/admin/settings/super-priority/activate',
      {},
    )
    return data
  },

  async deactivate(): Promise<{ message: string }> {
    const { data } = await apiClient.post<{ message: string }>('/admin/settings/super-priority/deactivate')
    return data
  },

  async refresh(): Promise<SchedulingRulesRefreshResult> {
    const { data } = await apiClient.post<SchedulingRulesRefreshResult>(
      '/admin/settings/scheduling-rules/refresh',
      {},
      { timeout: 130000 },
    )
    return data
  },

  async getRuntime(): Promise<SchedulingRulesRuntimeStatus> {
    const { data } = await apiClient.get<SchedulingRulesRuntimeStatus>('/admin/settings/scheduling-rules/runtime')
    return data
  },
}

export default superPriorityAPI
