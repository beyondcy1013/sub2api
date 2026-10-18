import { apiClient } from '@/api/client'
export interface DetectionPolicy {
  kind: 'connection' | 'quality'
  enabled: boolean
  account_scope: 'all' | 'selected'
  account_ids: number[]
  platforms: string[]
  statuses: string[]
  include_paused: boolean
  interval_minutes: number
  model_id: string
  recover: boolean
  activate: boolean
  resume: boolean
  pause_on_downgrade: boolean
  failure_threshold: number
}
export interface DetectionRun {
  started_at: string; finished_at: string; checked: number; succeeded: number
  failed: number; skipped: number; recovered: number; error?: string
}
export interface DetectionPolicyResult {
  account_id: number; account_name: string; kind: string
  status: string; message: string; finished_at: string
}
export interface AutomaticDetectionState {
  config: { revision: string; policies: DetectionPolicy[] }
  runtime: { running: boolean; next_run_at?: string | null; last_run?: DetectionRun | null; history: DetectionRun[]; results: DetectionPolicyResult[] }
}
export const automaticDetectionAPI = {
  async get(): Promise<AutomaticDetectionState> {
    return (await apiClient.get<AutomaticDetectionState>('/admin/accounts/automatic-detection')).data
  },
  async save(config: AutomaticDetectionState['config']): Promise<AutomaticDetectionState> {
    return (await apiClient.put<AutomaticDetectionState>('/admin/accounts/automatic-detection', config)).data
  }
}
export function validDetectionPolicy(p: DetectionPolicy) {
  return (!p.enabled || p.account_scope === 'all' || p.account_ids.length > 0) && p.statuses.length > 0 && Number.isInteger(p.interval_minutes) && p.interval_minutes >= 1 && p.interval_minutes <= 1440 && Number.isInteger(p.failure_threshold) && p.failure_threshold >= 1 && p.failure_threshold <= 10
}
