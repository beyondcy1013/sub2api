export default {
  accountPolicy: {
    title: 'Account Policy Settings',
    description: 'Manage account state, quotas, scheduling rates, and local resilience policies',
    searchPlaceholder: 'Search account name or notes',
    accountCount: '{count} accounts',
    empty: 'No matching accounts',
    selectAccount: 'Select an account to manage its policies',
    loadFailed: 'Failed to load account policies',
    save: 'Save policies',
    saved: 'Account policies saved',
    saveFailed: 'Failed to save account policies',
    state: {
      title: 'Account state',
      status: 'Current status',
      scheduling: 'Scheduling',
      enabled: 'Enabled',
      paused: 'Paused',
      recover: 'Recover state',
      recovered: 'Account state recovered',
      recoverFailed: 'Failed to recover account state',
      schedulingUpdated: 'Scheduling state updated',
      schedulingFailed: 'Failed to update scheduling state',
      rateLimitUntil: 'Rate limited until {time}',
      overloadUntil: 'Overload cooldown until {time}',
      temporaryUntil: 'Temporarily paused until {time}',
      quotaLimited: '{window} quota limited at {utilization}%; resets {time}',
      noRuntimeBlock: 'No runtime block is active',
      errorMessage: 'Latest error'
    },
    relay: {
      title: 'Relay failure budget',
      unavailable: 'This policy is not supported by the selected account',
      enabled: 'Enable failure budget',
      windowMinutes: 'Window (minutes)',
      thresholdPercent: 'Failure threshold (%)',
      minRequests: 'Minimum requests',
      consecutiveFailures: 'Consecutive failures',
      cooldownMinutes: 'Cooldown (minutes)'
    },
    quota: {
      title: 'Account quota limits',
      unavailable: 'Quota limits require a non-shadow API key or Bedrock account',
      total: 'Total limit',
      daily: 'Daily limit',
      weekly: 'Weekly limit',
      used: '{value} used',
      unlimited: '0 means unlimited'
    },
    rate: {
      title: 'Scheduling and billing rate',
      multiplier: 'Account rate',
      autoOverwrite: 'Auto sync',
      autoOverwriteHint: 'Replace the value after a successful upstream rate probe',
      manualLock: 'Manual lock',
      manualLockHint: 'Keep the manual value when automatic probes run'
    },
    validation: {
      relay: 'Relay failure budget values are outside the allowed range',
      quota: 'Quota limits must be numbers greater than or equal to 0',
      rate: 'Account rate must be a number greater than or equal to 0'
    }
  }
}
