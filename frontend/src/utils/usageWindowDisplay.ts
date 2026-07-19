interface UsageWindowResetOptions {
  utilization: number
  resetsAt?: string | null
  now: number
  labels: {
    now: string
    pending: string
  }
  showNowWhenIdle?: boolean
}

export function formatUsageWindowUtilization(utilization: number): string {
  if (!Number.isFinite(utilization)) return '-'
  const percent = Math.round(utilization)
  return percent > 999 ? '>999%' : `${percent}%`
}

export function formatUsageWindowReset({
  utilization,
  resetsAt,
  now,
  labels,
  showNowWhenIdle = false
}: UsageWindowResetOptions): string {
  if (showNowWhenIdle && utilization <= 0) return labels.now
  if (!resetsAt) return '-'

  const resetAt = Date.parse(resetsAt)
  if (!Number.isFinite(resetAt)) return '-'

  const diffMs = resetAt - now
  if (diffMs <= 0) return utilization > 0 ? labels.pending : labels.now

  const diffHours = Math.floor(diffMs / (60 * 60 * 1000))
  const diffMinutes = Math.floor((diffMs % (60 * 60 * 1000)) / (60 * 1000))
  if (diffHours >= 24) return `${Math.floor(diffHours / 24)}d ${diffHours % 24}h`
  if (diffHours > 0) return `${diffHours}h ${diffMinutes}m`
  return `${diffMinutes}m`
}
