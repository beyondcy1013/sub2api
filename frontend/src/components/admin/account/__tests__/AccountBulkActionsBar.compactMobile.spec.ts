import { describe, expect, it, vi } from 'vitest'

import accountBulkActionsBarSource from '../AccountBulkActionsBar.vue?raw'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('AccountBulkActionsBar compact mobile menu', () => {
  it('keeps the popup and action rows compact on small screens', () => {
    expect(accountBulkActionsBarSource).toContain('w-[min(calc(100vw-2rem),16rem)]')
    expect(accountBulkActionsBarSource).toContain('padding: 0.375rem 0.5rem;')
  })
})
