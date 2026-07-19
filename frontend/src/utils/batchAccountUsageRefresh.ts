export interface BatchAccountUsageFailure {
  accountId: number
  error: unknown
}

export interface BatchAccountUsageRefreshResult<T> {
  successful: T[]
  failed: BatchAccountUsageFailure[]
}

export async function refreshAccountUsageInBatches<T>(
  accountIds: number[],
  query: (accountId: number) => Promise<T>,
  concurrency = 4
): Promise<BatchAccountUsageRefreshResult<T>> {
  const workerCount = Math.min(Math.max(1, concurrency), accountIds.length)
  const results: Array<
    | { status: 'fulfilled'; value: T }
    | { status: 'rejected'; accountId: number; error: unknown }
    | undefined
  > = new Array(accountIds.length)
  let nextIndex = 0

  const worker = async () => {
    while (true) {
      const index = nextIndex
      nextIndex += 1
      if (index >= accountIds.length) return

      const accountId = accountIds[index]
      try {
        results[index] = { status: 'fulfilled', value: await query(accountId) }
      } catch (error) {
        results[index] = { status: 'rejected', accountId, error }
      }
    }
  }

  await Promise.all(Array.from({ length: workerCount }, () => worker()))

  const successful: T[] = []
  const failed: BatchAccountUsageFailure[] = []
  for (const result of results) {
    if (!result) continue
    if (result.status === 'fulfilled') successful.push(result.value)
    else failed.push({ accountId: result.accountId, error: result.error })
  }

  return { successful, failed }
}
