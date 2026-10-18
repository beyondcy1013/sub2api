import { adminAPI } from '@/api/admin'
import type { RequestQualityProbeSettings } from '@/types'
let pendingRead: Promise<RequestQualityProbeSettings> | undefined
// All rows on a page share one in-flight settings read.
export function loadQualitySettings() {
  if (!pendingRead) pendingRead = adminAPI.accounts.getRequestQualityProbeSettings().finally(() => { pendingRead = undefined })
  return pendingRead
}
let pendingWrite: Promise<unknown> = Promise.resolve()
export function saveAccountQuality(id: number, enabled: boolean): Promise<RequestQualityProbeSettings> {
  const write = pendingWrite.then(async () => {
    const current = await adminAPI.accounts.getRequestQualityProbeSettings()
    return adminAPI.accounts.updateRequestQualityProbeSettings({ ...current, account_overrides: { ...current.account_overrides, [id]: enabled } })
  })
  pendingWrite = write.catch(() => {})
  return write
}
