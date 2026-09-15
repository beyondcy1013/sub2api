import { describe, expect, it } from 'vitest'

import accountsViewSource from '../AccountsView.vue?raw'

describe('AccountsView downgraded column', () => {
  it('places persisted pelican downgrade immediately after status with red warning', () => {
    const statusIndex = accountsViewSource.indexOf("{ key: 'status', label: t('admin.accounts.columns.status')")
    const downgradedIndex = accountsViewSource.indexOf("{ key: 'downgraded', label: t('admin.accounts.columns.downgraded')")
    const schedulableIndex = accountsViewSource.indexOf("{ key: 'schedulable'")

    expect(statusIndex).toBeGreaterThan(-1)
    expect(downgradedIndex).toBeGreaterThan(statusIndex)
    expect(schedulableIndex).toBeGreaterThan(downgradedIndex)
    expect(accountsViewSource).toContain('row.extra?.pelican_downgraded === true')
    expect(accountsViewSource).toContain('bg-red-100')
    expect(accountsViewSource).toContain('admin.accounts.pelicanDowngraded')
  })
})
