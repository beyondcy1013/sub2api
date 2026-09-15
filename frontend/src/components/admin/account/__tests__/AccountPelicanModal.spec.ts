import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountPelicanModal from '../AccountPelicanModal.vue'
import {
  clearPelicanResultCache,
  setCachedPelicanResult
} from '@/utils/pelicanResultCache'

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
    localStorage.clear()
    clearPelicanResultCache()
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
    expect(wrapper.text()).toContain('admin.accounts.pelicanPromptLabel')
    expect(wrapper.find('textarea.textarea-stub').exists()).toBe(true)
  })

  it('点击开始测智并发向后端请求 pelican 模式测试并展示预览', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await (wrapper.vm as any).startAllTests()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(2)
    const [, req1] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(req1.body)).toMatchObject({
      mode: 'pelican',
      model_id: 'gpt-6-astra',
      reasoning_effort: 'low'
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

  it('开始测智时所有选中账号同时发起请求', async () => {
    const wrapper = mountModal([
      { id: 31, name: 'Concurrent 1', platform: 'openai', type: 'oauth', status: 'active' },
      { id: 32, name: 'Concurrent 2', platform: 'openai', type: 'oauth', status: 'active' },
      { id: 33, name: 'Concurrent 3', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    await (wrapper.vm as any).startAllTests()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(3)
    expect((wrapper.vm as any).accountStates.map((item: any) => item.status)).toEqual([
      'success',
      'success',
      'success'
    ])
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

  it('默认打开弹窗不直接开始，账号保持待测试 (idle) 状态', async () => {
    const wrapper = mountModal()
    await flushPromises()
    await new Promise(r => setTimeout(r, 120))
    await flushPromises()

    expect(global.fetch).not.toHaveBeenCalled()
    expect((wrapper.vm as any).isRunning).toBe(false)
    expect((wrapper.vm as any).accountStates[0].status).toBe('idle')
    await wrapper.unmount()
  })

  it('显式传入 autoStart 为 true 时自动启动测智', async () => {
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

  it('支持选择或输入大模型并正确传递至后端', async () => {
    const wrapper = mountModal([
      { id: 10, name: 'Model Test Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    // 设置大模型与思考程度
    ;(wrapper.vm as any).selectedModel = 'gpt-5.4'
    ;(wrapper.vm as any).reasoningEffort = 'high'
    await flushPromises()

    await (wrapper.vm as any).startAllTests()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalled()
    const [, req] = (global.fetch as any).mock.calls[(global.fetch as any).mock.calls.length - 1]
    const body = JSON.parse(req.body)
    expect(body.model_id).toBe('gpt-5.4')
    expect(body.reasoning_effort).toBe('high')
  })

  it('默认大模型为 gpt-6-astra 且思考程度默认为 low', async () => {
    const wrapper = mountModal()
    await flushPromises()

    expect((wrapper.vm as any).selectedModel).toBe('gpt-6-astra')
    expect((wrapper.vm as any).reasoningEffort).toBe('low')
  })

  it('点击账号历史按钮可展开并快速浏览历史结果', async () => {
    setCachedPelicanResult({
      account: { id: 20, name: 'History Account', type: 'oauth' },
      status: 'downgraded',
      responseModel: 'old-model',
      reason: 'old failure',
      testedAt: 1000
    })
    setCachedPelicanResult({
      account: { id: 20, name: 'History Account', type: 'oauth' },
      status: 'success',
      result: {
        has_html: true,
        html: '<svg></svg>',
        downgraded: false,
        reason: 'ok',
        response_model: 'new-model'
      },
      responseModel: 'new-model',
      reason: 'ok',
      testedAt: 2000
    })
    const wrapper = mountModal([
      { id: 20, name: 'History Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    expect((wrapper.vm as any).historyCount(20)).toBe(2)
    expect(wrapper.find('iframe').exists()).toBe(true)
    expect((wrapper.vm as any).accountStates[0].responseModel).toBe('new-model')

    await wrapper.get('[data-test="pelican-history-toggle"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="pelican-history-panel"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]')).toHaveLength(2)
    expect(wrapper.text()).toContain('old-model')
    expect(wrapper.findAll('iframe')).toHaveLength(2)

    await wrapper.get('[data-test="pelican-global-history-entry"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('old-model')
    expect(wrapper.text()).toContain('new-model')

    await (wrapper.vm as any).selectHistoryResult((wrapper.vm as any).accountStates[0], 1)
    await flushPromises()
    expect((wrapper.vm as any).accountStates[0].status).toBe('downgraded')
    expect((wrapper.vm as any).accountStates[0].responseModel).toBe('old-model')
  })

  it('使用下拉列表切换提示词方案并持久化选中与内容', async () => {
    const wrapper = mountModal([
      { id: 40, name: 'Prompt Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()
    ;(wrapper.vm as any).customPrompt = 'Pelican on spaceship'
    await flushPromises()
    await wrapper.get('[data-test="pelican-prompt-save"]').trigger('click')
    await flushPromises()
    expect(JSON.parse(localStorage.getItem('sub2api:account-pelican-prompt-scenarios:v1') || 'null'))
      .toMatchObject({ selectedId: 'default', scenarios: [{ prompt: 'Pelican on spaceship' }] })

    ;(wrapper.vm as any).addPromptScenario()
    await flushPromises()

    const selects = wrapper.findAll('select')
    expect(selects.map(select => (select.element as HTMLSelectElement).value)).toContain((wrapper.vm as any).selectedPromptId)
    expect(selects.length).toBeGreaterThanOrEqual(3)

    expect((wrapper.vm as any).promptScenarios).toHaveLength(2)
    expect((wrapper.vm as any).customPrompt).toContain('admin.accounts.pelicanPromptDefault')
    expect(JSON.parse(localStorage.getItem('sub2api:account-pelican-prompt-scenarios:v1') || 'null').selectedId)
      .not.toBe('default')

    await selects.find(select => select.findAll('option').length === 2)!.setValue('default')
    await flushPromises()
    expect((wrapper.vm as any).customPrompt).toBe('Pelican on spaceship')

    await selects.find(select => select.findAll('option').length === 2)!.setValue((wrapper.vm as any).selectedPromptId === 'default'
      ? (wrapper.vm as any).promptScenarios[1].id
      : 'default')
    await flushPromises()
    expect((wrapper.vm as any).customPrompt).toBe('admin.accounts.pelicanPromptDefault')
  })

  it('keeps prompt options previewable and saves edited drafts explicitly', async () => {
    const wrapper = mountModal([
      { id: 41, name: 'Prompt Save Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    expect(wrapper.text()).not.toContain('admin.accounts.pelicanPromptScenario')
    expect(wrapper.text()).not.toContain('admin.accounts.pelicanPromptScenarioName')

    const saveButton = wrapper.get('[data-test="pelican-prompt-save"]')
    expect(saveButton.attributes()).toHaveProperty('disabled')

    await wrapper.get('textarea.textarea-stub').setValue(
      'Create a long pelican SVG animation prompt that should be previewed directly'
    )
    await flushPromises()
    expect((wrapper.vm as any).promptDirty).toBe(true)
    const beforeSave = (wrapper.vm as any).promptScenarios[0].prompt
    expect(beforeSave).toBe('admin.accounts.pelicanPromptDefault')
    expect((wrapper.vm as any).customPrompt).not.toBe(beforeSave)

    await saveButton.trigger('click')
    await flushPromises()

    expect(JSON.parse(localStorage.getItem('sub2api:account-pelican-prompt-scenarios:v1') || 'null').scenarios[0])
      .toMatchObject({ prompt: (wrapper.vm as any).customPrompt })
    expect((wrapper.vm as any).promptScenarios[0].name).toContain('Create a long pelican')
    expect(wrapper.findAll('select')[1]?.text()).toContain('Create a long pelican')
    expect(wrapper.get('[data-test="pelican-prompt-save"]').attributes()).toHaveProperty('disabled')
  })
})
