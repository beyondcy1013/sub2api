import { describe, expect, it } from 'vitest'

import accountBulkActionsBarSource from '../AccountBulkActionsBar.vue?raw'
import mobileActionSheetSource from '@/components/common/MobileActionSheet.vue?raw'

describe('AccountBulkActionsBar compact mobile menu', () => {
  it('renders the selection-only mobile bar and reuses the shared bottom action sheet', () => {
    expect(accountBulkActionsBarSource).toContain('MobileActionSheet')
    expect(accountBulkActionsBarSource).toContain('v-if="selectedIds.length > 0"')
    expect(accountBulkActionsBarSource).toContain('data-test="mobile-bulk-actions-trigger"')
    expect(accountBulkActionsBarSource).toContain('data-test="mobile-bulk-actions-sheet"')
    expect(accountBulkActionsBarSource).toContain('max-lg:hidden')
  })

  it('closes the sheet before emitting selected-scope actions through one helper', () => {
    expect(accountBulkActionsBarSource).toContain('const closeAndRun = async (action: () => void) => {')
    expect(accountBulkActionsBarSource).toContain('await nextTick()')
    expect(accountBulkActionsBarSource).toContain("closeAndRun(() => $emit('test-and-mark'))")
    expect(accountBulkActionsBarSource).toContain("closeAndRun(() => $emit('delete'))")
    expect(accountBulkActionsBarSource).toContain("closeAndRun(() => $emit('permanent-delete'))")
    expect(accountBulkActionsBarSource).toContain("closeAndRun(() => $emit('toggle-schedulable', true))")
  })

  it('auto-closes the sheet when the selection empties', () => {
    expect(accountBulkActionsBarSource).toContain('watch(() => props.selectedIds.length')
    expect(accountBulkActionsBarSource).toContain('if (count === 0) mobileSheetOpen.value = false')
  })

  it('uses a dedicated body scroll lock separate from dialogs and yields keys to Select portals', () => {
    expect(mobileActionSheetSource).toContain("BODY_LOCK_CLASS = 'mobile-action-sheet-open'")
    expect(mobileActionSheetSource).toContain("closest?.('.select-dropdown-portal')")
    expect(mobileActionSheetSource).toContain('prefers-reduced-motion')
  })
})
