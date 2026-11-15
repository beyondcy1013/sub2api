import { describe, expect, it } from 'vitest'

import accountsViewSource from '../AccountsView.vue?raw'

describe('AccountsView downgraded column', () => {
  it('places the account quality status after schedulable and keeps it visible by default', () => {
    const statusIndex = accountsViewSource.indexOf("{ key: 'status', label: t('admin.accounts.columns.status')")
    const schedulableIndex = accountsViewSource.indexOf("{ key: 'schedulable'")
    const downgradedIndex = accountsViewSource.indexOf("{ key: 'downgraded', label: t('admin.accounts.columns.downgraded')")

    expect(statusIndex).toBeGreaterThan(-1)
    expect(schedulableIndex).toBeGreaterThan(statusIndex)
    expect(downgradedIndex).toBeGreaterThan(schedulableIndex)
    expect(accountsViewSource).toContain('<AccountQualityToggle :account="row"')
    expect(accountsViewSource).toContain("DEFAULT_HIDDEN_COLUMNS = ['today_stats', 'proxy', 'notes', 'scheduler_score', 'rate_multiplier']")
  })
})
