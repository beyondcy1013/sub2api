import { describe, expect, it } from 'vitest'

import accountsViewSource from '../AccountsView.vue?raw'

describe('AccountsView Android table layout', () => {
  it('forces the desktop-style table only for the native sub2api Android client', () => {
    expect(accountsViewSource).toContain(':force-table-layout="isSub2ApiAndroidClient()"')
    expect(accountsViewSource).toContain("import { isSub2ApiAndroidClient } from '@/utils/device'")
  })
})
