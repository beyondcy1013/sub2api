import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseDialog from '@/components/common/BaseDialog.vue'

describe('BaseDialog title actions', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders optional actions immediately beside the title', () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Scheduling Rules' },
      slots: {
        'title-actions': '<span data-testid="title-help">? Help</span>',
      },
    })

    const title = document.body.querySelector('.modal-title')
    const help = document.body.querySelector('[data-testid="title-help"]')
    expect(title?.parentElement).toBe(help?.parentElement)
    expect(title?.nextElementSibling).toBe(help)

    wrapper.unmount()
  })

  it('renders optional header actions beside the close control', () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Scheduling Rules' },
      slots: {
        'header-actions': '<span data-testid="runtime-status">Next probe in 1m</span>',
      },
    })

    const status = document.body.querySelector('[data-testid="runtime-status"]')
    const close = document.body.querySelector('[aria-label="Close modal"]')
    expect(status?.parentElement).toBe(close?.parentElement)
    expect(status?.nextElementSibling).toBe(close)

    wrapper.unmount()
  })
})
