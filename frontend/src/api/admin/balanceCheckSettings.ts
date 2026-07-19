import { apiClient } from '../client'

export interface BalanceCheckConfig {
  enabled: boolean
  interval: string
  balance_url: string
  request_timeout_seconds: number
  max_concurrent_checks: number
  pause_duration_hours: number
  min_decrease: number
  pause_when_current_below: number
  pause_when_drop_percent: number
  stop_when_current_below: number
  resume_when_current_above: number
  require_quota_hourly_limit: boolean
}

export interface BalanceCheckSettingsResponse {
  config: BalanceCheckConfig
  config_path: string
  restart_required: boolean
}

const balanceCheckSettingsAPI = {
  async get(): Promise<BalanceCheckSettingsResponse> {
    const { data } = await apiClient.get<BalanceCheckSettingsResponse>('/admin/settings/balance-check')
    return data
  },

  async update(config: BalanceCheckConfig): Promise<BalanceCheckSettingsResponse> {
    const { data } = await apiClient.put<BalanceCheckSettingsResponse>('/admin/settings/balance-check', config)
    return data
  }
}

export default balanceCheckSettingsAPI
