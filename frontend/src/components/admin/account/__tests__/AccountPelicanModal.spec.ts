import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountPelicanModal from '../AccountPelicanModal.vue'

const { copyToClipboard } = vi.hoisted(() => ({
  copyToClipboard: vi.fn()
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.pelicanAccountsCount' && params) {
          return `selected-${params.selected}-total-${params.total}`
        }
        return key
      }
    })
  }
})

function createStreamResponse(lines: string[]) {
  const encoder = new TextEncoder()
  const chunks = lines.map((line) => encoder.encode(line))
  let index = 0

  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn().mockImplementation(async () => {
          if (index < chunks.length) {
            return { done: false, value: chunks[index++] }
          }
          return { done: true, value: undefined }
        })
      })
    }
  } as Response
}

function mountModal(
  accounts = [
    { id: 1, name: 'OpenAI Account 1', platform: 'openai', type: 'oauth', status: 'active' },
    { id: 2, name: 'OpenAI Account 2', platform: 'openai', type: 'apikey', status: 'active' }
  ],
  autoStart = false
) {
  return mount(AccountPelicanModal, {
    props: {
      show: true,
      accounts: accounts as any,
      autoStart
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

describe('AccountPelicanModal', () => {
  beforeEach(() => {
    copyToClipboard.mockReset()
    global.fetch = vi.fn().mockImplementation(() =>
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.6-sol"}\n',
        'data: {"type":"pelican_result","text":"测智通过","data":{"has_html":true,"html":"<!doctype html><html><body><svg></svg></body></html>","downgraded":false,"reason":"测智通过","response_model":"gpt-5.6-sol"}}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('渲染所选账号列表与初始状态', async () => {
    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI Account 1')
    expect(wrapper.text()).toContain('OpenAI Account 2')
    expect((wrapper.vm as any).accountStates).toHaveLength(2)
    expect((wrapper.vm as any).accountStates[0].status).toBe('idle')
  })

  it('点击开始测智并发向后端请求 pelican 模式测试并展示预览', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await (wrapper.vm as any).startAllTests()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(2)
    const [, req1] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(req1.body)).toMatchObject({
      mode: 'pelican'
    })

    const state1 = (wrapper.vm as any).accountStates[0]
    expect(state1.status).toBe('success')
    expect(state1.result.has_html).toBe(true)
    expect(state1.responseModel).toBe('gpt-5.6-sol')

    // iframe preview should be rendered
    const iframe = wrapper.find('iframe')
    expect(iframe.exists()).toBe(true)
    expect(iframe.attributes('srcdoc')).toContain('<svg></svg>')
  })

  it('正确识别并标注疑似降智账号', async () => {
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-4o-mini"}\n',
        'data: {"type":"pelican_result","text":"模型未返回包含 SVG 动画的 HTML 文档","data":{"has_html":false,"html":"","downgraded":true,"reason":"模型未返回包含 SVG 动画的 HTML 文档","response_model":"gpt-4o-mini"}}\n',
        'data: {"type":"test_complete","success":false}\n'
      ])
    ) as any

    const wrapper = mountModal([
      { id: 1, name: 'Downgraded Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    await (wrapper.vm as any).startAllTests()
    await flushPromises()

    const state = (wrapper.vm as any).accountStates[0]
    expect(state.status).toBe('downgraded')
    expect(state.reason).toContain('未返回')
  })

  it('支持单账号重新测智', async () => {
    const wrapper = mountModal([
      { id: 10, name: 'Single Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    const state = (wrapper.vm as any).accountStates[0]
    await (wrapper.vm as any).runSingleTest(state)
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    expect(state.status).toBe('success')
  })

  it('并发数默认值必须为 1', async () => {
    const wrapper = mountModal()
    await flushPromises()

    expect((wrapper.vm as any).concurrency).toBe(1)
  })

  it('支持在弹窗内选择和切换账号', async () => {
    const wrapper = mount(AccountPelicanModal, {
      props: {
        show: true,
        accounts: [{ id: 1, name: 'Acc 1', platform: 'openai', type: 'oauth', status: 'active' }] as any,
        allAccounts: [
          { id: 1, name: 'Acc 1', platform: 'openai', type: 'oauth', status: 'active' },
          { id: 2, name: 'Acc 2', platform: 'openai', type: 'apikey', status: 'active' }
        ] as any
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          TextArea: true,
          Icon: true
        }
      }
    })
    await flushPromises()

    expect((wrapper.vm as any).accountStates).toHaveLength(1)

    // 添加第 2 个账号
    ;(wrapper.vm as any).toggleAccountSelection({ id: 2, name: 'Acc 2', platform: 'openai', type: 'apikey', status: 'active' })
    await flushPromises()

    expect((wrapper.vm as any).accountStates).toHaveLength(2)
  })

  it('支持打开和关闭全屏大图预览 Lightbox', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const state = (wrapper.vm as any).accountStates[0]
    state.result = { has_html: true, html: '<svg></svg>', downgraded: false, reason: '' }

    ;(wrapper.vm as any).openLightbox(state)
    expect((wrapper.vm as any).lightboxItem).toBe(state)

    ;(wrapper.vm as any).closeLightbox()
    expect((wrapper.vm as any).lightboxItem).toBeNull()
  })

  it('默认情况下打开弹窗自动启动测智', async () => {
    const wrapper = mountModal(
      [{ id: 99, name: 'Auto Start Account', platform: 'openai', type: 'oauth', status: 'active' }],
      true
    )
    await flushPromises()
    await new Promise(r => setTimeout(r, 120))
    await flushPromises()

    expect(global.fetch).toHaveBeenCalled()
    expect((wrapper.vm as any).accountStates[0].status).toBe('success')
  })
})
