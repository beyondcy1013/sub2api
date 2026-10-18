import type { Account, RequestQualityProbeSettings } from '@/types'

// This is account participation; the master switch and model scope still apply.
export function accountQualityEnabled(settings: RequestQualityProbeSettings, account: Account): boolean {
  const override = settings.account_overrides?.[account.id]
  if (typeof override === 'boolean') return override
  if (settings.account_scope === 'selected') return settings.account_ids.includes(account.id) || account.extra?.test_on_request_enabled === true
  if (settings.account_scope === 'error_or_downgraded') return account.status === 'error' || account.extra?.pelican_downgraded === true || !!account.error_message
  return true
}
