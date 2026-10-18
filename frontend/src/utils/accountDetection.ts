import type { Account, RequestQualityProbeSettings } from '@/types'
import { adminAPI } from '@/api/admin'

export const defaultRequestQualitySettings = (): RequestQualityProbeSettings => ({ enabled:false, status_recent_minutes:5, model_scope:'all', models:[], recent_minutes:10, account_scope:'all', account_ids:[], rule_ids:[], allow_business_success:false, timeout_seconds:35 })
export const normalizeRequestQualitySettings = (value: RequestQualityProbeSettings): RequestQualityProbeSettings => ({...defaultRequestQualitySettings(),...value, rule_ids:(value.rule_ids ?? []).slice(0,1), model_scope:value.model_scope ?? (value.models?.length ? 'selected' : 'all')})
export function validRequestQualitySettings(value: RequestQualityProbeSettings) {
  return Number.isInteger(value.status_recent_minutes ?? 5) && (value.status_recent_minutes ?? 5)>=1 && (value.status_recent_minutes ?? 5)<=1440 && (!value.enabled || (Number.isInteger(value.timeout_seconds) && value.timeout_seconds!>=5 && value.timeout_seconds!<=120 && (value.rule_ids?.length ?? 0)<=1 && (value.model_scope === 'all' || value.models.length>0) && (value.account_scope !== 'selected' || value.account_ids.length>0 || Object.values(value.account_overrides ?? {}).some(Boolean))))
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
