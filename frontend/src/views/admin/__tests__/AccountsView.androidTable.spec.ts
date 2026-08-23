import { describe, expect, it } from 'vitest'

import accountsViewSource from '../AccountsView.vue?raw'

describe('AccountsView Android table layout', () => {
  it('forces the table but pins only the selection column in the native Android client', () => {
    expect(accountsViewSource).toContain(':force-table-layout="useAndroidTableLayout"')
    expect(accountsViewSource).toContain(':sticky-left-column-keys="accountStickyLeftColumnKeys"')
    expect(accountsViewSource).toContain("const useAndroidTableLayout = isSub2ApiAndroidClient()")
    expect(accountsViewSource).toContain(
      "const accountStickyLeftColumnKeys = useAndroidTableLayout\n  ? ['select']\n  : ACCOUNT_STICKY_LEFT_COLUMN_KEYS"
    )
    expect(accountsViewSource).toContain("import { isSub2ApiAndroidClient } from '@/utils/device'")
  })
})
