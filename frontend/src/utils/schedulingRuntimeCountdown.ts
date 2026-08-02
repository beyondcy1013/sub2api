export interface SchedulingRuntimeCountdown {
  duration: string
  remainingSeconds: number
  progressPercent: number | null
  isDue: boolean
}

function parseEveryIntervalMs(expression: string | undefined): number | null {
  const match = /^@every\s+(.+)$/i.exec(expression?.trim() ?? '')
  if (!match) return null

  let value = match[1].trim()
  let totalMs = 0
  while (value) {
    const token = /^(\d+(?:\.\d+)?)(ms|s|m|h)/i.exec(value)
    if (!token) return null
    const amount = Number(token[1])
    const unitMs = { ms: 1, s: 1000, m: 60_000, h: 3_600_000 }[token[2].toLowerCase() as 'ms' | 's' | 'm' | 'h']
    if (!Number.isFinite(amount) || !unitMs) return null
    totalMs += amount * unitMs
    value = value.slice(token[0].length).trimStart()
  }

  return totalMs > 0 ? totalMs : null
}

export function formatSchedulingCountdown(seconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const remainder = totalSeconds % 60
  return [hours, minutes, remainder].map((value) => String(value).padStart(2, '0')).join(':')
}

export function getSchedulingRuntimeCountdown(
  nextRunAt: string | undefined,
  nowMs: number,
  intervalExpression?: string
): SchedulingRuntimeCountdown | null {
  if (!nextRunAt) return null
  const nextRunMs = new Date(nextRunAt).getTime()
  if (!Number.isFinite(nextRunMs)) return null

  const remainingMs = nextRunMs - nowMs
  const remainingSeconds = Math.max(0, Math.ceil(remainingMs / 1000))
  const intervalMs = parseEveryIntervalMs(intervalExpression)
  const progressPercent = intervalMs
    ? Math.min(100, Math.max(0, Math.round(((nowMs - (nextRunMs - intervalMs)) / intervalMs) * 100)))
    : null

  return {
    duration: formatSchedulingCountdown(remainingSeconds),
    remainingSeconds,
    progressPercent,
    isDue: remainingMs <= 0
  }
}
