import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(testDirectory, '../index.ts'), 'utf8')
const sidebarSource = readFileSync(
  resolve(testDirectory, '../../components/layout/AppSidebar.vue'),
  'utf8'
)

describe('account policy settings navigation', () => {
  it('registers an admin-only route and sidebar entry', () => {
    expect(routerSource).toContain("path: '/admin/account-policy-settings'")
    expect(routerSource).toContain("name: 'AdminAccountPolicySettings'")
    expect(routerSource).toContain("component: () => import('@/views/admin/AccountPolicySettingsView.vue')")
    expect(sidebarSource).toContain("path: '/admin/account-policy-settings'")
    expect(sidebarSource).toContain("label: t('nav.accountPolicySettings')")
  })
})
