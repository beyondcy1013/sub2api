import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('account table column locale keys', () => {
  it('exposes the today cost column label at the runtime locale path', () => {
    expect(en.admin.accounts.columns.todayCost).toBe('Today Cost')
    expect(zh.admin.accounts.columns.todayCost).toBe('今日费用')
  })
})
