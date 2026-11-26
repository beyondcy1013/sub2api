import type { Account, RequestQualityProbeSettings } from '@/types'
import { adminAPI } from '@/api/admin'

export const defaultRequestQualitySettings = (): RequestQualityProbeSettings => ({ enabled:false, status_recent_minutes:5, model_scope:'all', models:[], exclude_models:['glm*'], recent_minutes:10, account_scope:'all', account_ids:[], rule_ids:[], allow_business_success:false, timeout_seconds:35, reuse_seconds:300, auto_pause_on_downgrade:true, downgrade_threshold:3 })
export const normalizeRequestQualitySettings = (value: RequestQualityProbeSettings): RequestQualityProbeSettings => ({...defaultRequestQualitySettings(),...value, rule_ids:(value.rule_ids ?? []).slice(0,1), model_scope:value.model_scope ?? (value.models?.length ? 'selected' : 'all'), exclude_models:value.exclude_models ?? ['glm*'], auto_pause_on_downgrade:value.auto_pause_on_downgrade ?? true, downgrade_threshold:Math.min(10, Math.max(1, value.downgrade_threshold ?? 3))})
export function validRequestQualitySettings(value: RequestQualityProbeSettings) {
  const threshold = value.downgrade_threshold ?? 3
  return Number.isInteger(value.status_recent_minutes ?? 5) && (value.status_recent_minutes ?? 5)>=1 && (value.status_recent_minutes ?? 5)<=1440 && (!value.enabled || (Number.isInteger(value.reuse_seconds ?? 300) && (value.reuse_seconds ?? 300)>=0 && (value.reuse_seconds ?? 300)<=3600 && Number.isInteger(value.timeout_seconds) && value.timeout_seconds!>=5 && value.timeout_seconds!<=120 && Number.isInteger(threshold) && threshold>=1 && threshold<=10 && (value.rule_ids?.length ?? 0)<=1 && (value.model_scope === 'all' || value.models.length>0) && (value.account_scope !== 'selected' || value.account_ids.length>0 || Object.values(value.account_overrides ?? {}).some(Boolean))))
}
// Page through the complete management pool; never silently truncate to a single page.
export async function loadDetectionAccounts(): Promise<Account[]> {
  const accounts: Account[]=[]
  for (let page=1;;page++) {
    const data=await adminAPI.accounts.list(page,100,{lite:'1'})
    accounts.push(...data.items)
    if (!data.items.length || accounts.length>=data.total || (data.total == null && data.items.length<100)) return accounts
  }
}
