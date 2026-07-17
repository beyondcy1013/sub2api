import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'admin.proxies.ad.inline': 'Sub2API Enhanced: balanced routing and efficient account operations',
      'admin.proxies.ad.details': 'Enhanced feature details'
    })[key] ?? key
  })
}))

import ProxyAdBanner from '../ProxyAdBanner.vue'

describe('ProxyAdBanner', () => {
  it('opens the configured advertisement page in a new tab', () => {
    const wrapper = mount(ProxyAdBanner, {
      global: {
        stubs: {
          Icon: true
        }
      }
    })
    const link = wrapper.get('a')

    expect(link.text()).toContain('Sub2API Enhanced: balanced routing and efficient account operations')
    expect(link.attributes('href')).toBe('https://pay.ldxp.cn/shop/99739964')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toBe('noopener noreferrer')
    expect(link.attributes('title')).toBe('Enhanced feature details')
    expect(link.attributes('aria-label')).toContain('Enhanced feature details')
  })
})
