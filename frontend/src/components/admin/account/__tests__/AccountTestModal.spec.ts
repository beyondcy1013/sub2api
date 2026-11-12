import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountTestModal from '../AccountTestModal.vue'

let storage: Map<string, string>

const { getAvailableModels, copyToClipboard } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  copyToClipboard: vi.fn()
}))

const PELICAN_SCENARIO_STORAGE_KEY = 'sub2api:account-pelican-prompt-scenarios:v1'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.imagePromptDefault': 'Generate a cute orange cat astronaut sticker on a clean pastel background.',
    'admin.accounts.startingTestForAccountWithUpstream': 'Starting test for account: {name} (upstream: {url})',
    'admin.accounts.pelicanPromptDefault': '创建一个 HTML，内容是 SVG 绘制一个鹈鹕骑自行车的 2D 动画，你不需要任何测试，不要有任何限制',
    'admin.accounts.pelicanPromptScenarioDefault': '默认方案',
    'admin.accounts.pelicanPromptCustom': '自定义'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.imageReceived' && params?.count) {
          return `received-${params.count}`
        }
        return Object.entries(params || {}).reduce(
          (message, [name, value]) => message.replace(`{${name}}`, String(value)),
          messages[key] || key
        )
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

function mountModal(account: Record<string, unknown> = {
  id: 42,
  name: 'Gemini Image Test',
  platform: 'gemini',
  type: 'apikey',
  status: 'active'
}) {
  return mount(AccountTestModal, {
    props: {
      show: false,
      account
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: {
          name: 'Select',
          props: {
            modelValue: null,
            options: Array,
            disabled: Boolean,
            searchable: [Boolean, String],
            creatable: Boolean
          },
          template: '<div class="select-stub"></div>'
        },
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

describe('AccountTestModal', () => {
  beforeEach(() => {
    getAvailableModels.mockResolvedValue([
      { id: 'gemini-2.0-flash', display_name: 'Gemini 2.0 Flash' },
      { id: 'gemini-2.5-flash-image', display_name: 'Gemini 2.5 Flash Image' },
      { id: 'gemini-3.1-flash-image', display_name: 'Gemini 3.1 Flash Image' }
    ])
    copyToClipboard.mockReset()
    storage = new Map([['auth_token', 'test-token']])
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => storage.get(key) ?? null),
        setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
        removeItem: vi.fn((key: string) => storage.delete(key)),
        clear: vi.fn(() => storage.clear())
      },
      configurable: true
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-2.5-flash-image"}\n',
        'data: {"type":"image","image_url":"data:image/png;base64,QUJD","mime_type":"image/png"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('打开弹窗不自动测试，点击开始测试才发起请求', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    await flushPromises()

    expect(global.fetch).not.toHaveBeenCalled()

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gemini-3.1-flash-image'
    })
    expect(wrapper.emitted('test-succeeded')?.[0]?.[0]).toMatchObject({ id: 42 })
  })

  it('测试中点击按钮可停止测试并恢复可重试状态', async () => {
    global.fetch = vi.fn().mockImplementation((_url: string, init: any) =>
      Promise.resolve({
        ok: true,
        body: {
          getReader: () => ({
            read: () =>
              new Promise((_resolve, reject) => {
                init.signal?.addEventListener('abort', () => {
                  reject(new DOMException('Aborted', 'AbortError'))
                })
              })
          })
        }
      })
    ) as any

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const startButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()
    await startButton!.trigger('click')
    await flushPromises()

    expect((wrapper.vm as any).status).toBe('connecting')

    const stopButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.stopTest'))
    expect(stopButton).toBeTruthy()
    await stopButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect((wrapper.vm as any).status).toBe('idle')
    expect(wrapper.emitted('test-succeeded')).toBeUndefined()
    expect(wrapper.emitted('test-failed')).toBeUndefined()
  })

  it('即使 API 帮助函数已解包响应信封，仍渲染可用模型', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const selectedModelId = (wrapper.vm as any).selectedModelId
    expect(selectedModelId).toBe('gemini-3.1-flash-image')
    expect(getAvailableModels).toHaveBeenCalledWith(42, { local: true })

    const modelSelect = wrapper.findAllComponents({ name: 'Select' })[0]
    expect(modelSelect.props('searchable')).toBe(true)
    expect(modelSelect.props('creatable')).toBe(true)
  })

  it('测试失败时不发送成功事件', async () => {
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_complete","success":false,"error":"upstream failed"}\n'
      ])
    ) as any

    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.emitted('test-succeeded')).toBeUndefined()
    expect(wrapper.emitted('test-failed')?.[0]?.[0]).toMatchObject({ id: 42 })
    expect(wrapper.emitted('test-failed')?.[0]?.[1]).toBe('upstream failed')
  })

  it('gemini 图片模型测试会携带提示词并渲染图片预览', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    await promptInput.setValue('draw a tiny orange cat astronaut')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gemini-3.1-flash-image',
      prompt: 'draw a tiny orange cat astronaut'
    })

    const preview = wrapper.find('img[alt="test-image-1"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('data:image/png;base64,QUJD')
  })

  it('grok 账号测试默认选择 Grok 模型', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'grok-4.3', display_name: 'Grok 4.3' },
      { id: 'grok-build-0.1', display_name: 'Grok Build 0.1' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"grok-4.3"}\n',
        'data: {"type":"content","text":"ok"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 13,
      name: 'Grok Account',
      platform: 'grok',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'grok-4.3',
      prompt: ''
    })
  })

  it('OpenAI 测试优先选择 GPT-5.6 Sol，而不是依赖映射返回顺序', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.6-terra', display_name: 'GPT-5.6 Terra' },
      { id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol' },
      { id: 'gpt-5.6-luna', display_name: 'GPT-5.6 Luna' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 14,
      name: 'OpenAI Account',
      platform: 'openai',
      type: 'apikey',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    await (wrapper.vm as any).startTest()
    await flushPromises()

    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gpt-5.6-sol'
    })
  })

  it('OpenAI Compact 探测会携带 compact 测试模式', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 42,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'compact'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gpt-5.4',
      prompt: '',
      mode: 'compact'
    })
  })

  it('starts a test with a sanitized configured upstream URL', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    const wrapper = mountModal({
      id: 99,
      name: '眉间雪free25_nikoapi.xyz',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      credentials: {
        base_url: 'https://api-user:api-password@nikoapi.xyz/v1?api_key=secret#token'
      }
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.text()).toContain(
      'Starting test for account: 眉间雪free25_nikoapi.xyz (upstream: https://nikoapi.xyz/v1)'
    )
    expect(wrapper.text()).not.toContain('api-user')
    expect(wrapper.text()).not.toContain('api-password')
    expect(wrapper.text()).not.toContain('secret')
  })

  it('OpenAI Pelican 测智会携带 pelican 模式与提示词并处理 pelican_result 事件', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.6-sol"}\n',
        'data: {"type":"pelican_result","text":"测智通过","data":{"has_html":true,"html":"<!doctype html><html><body><svg></svg></body></html>","downgraded":false,"reason":"测智通过","response_model":"gpt-5.6-sol"}}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 42,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    ;(wrapper.vm as any).selectedModelId = 'gpt-5.6-sol'
    ;(wrapper.vm as any).testMode = 'pelican'
    ;(wrapper.vm as any).pelicanPrompt = '绘制一个鹈鹕'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gpt-5.6-sol',
      prompt: '绘制一个鹈鹕',
      mode: 'pelican'
    })

    expect((wrapper.vm as any).pelicanResult).toMatchObject({
      has_html: true,
      downgraded: false,
      response_model: 'gpt-5.6-sol'
    })
  })

  it('custom numeric questions send the chosen evaluator and answer without an HTML requirement', async () => {
    getAvailableModels.mockResolvedValue([{ id: 'gpt-6.1-sol', display_name: 'GPT-6.1 Sol' }])
    global.fetch = vi.fn().mockResolvedValue(createStreamResponse(['data: {"type":"test_complete","success":true}\n'])) as any
    const wrapper = mountModal({ id: 42, name: 'aws-autoapi-remote', platform: 'openai', type: 'apikey', status: 'active' })
    await wrapper.setProps({ show: true })
    await flushPromises()
    ;(wrapper.vm as any).testMode = 'pelican'
    ;(wrapper.vm as any).pelicanPrompt = '只回答数字：最少取多少颗糖果？'
    ;(wrapper.vm as any).probeEvaluator = 'numeric'
    ;(wrapper.vm as any).probeExpectedAnswer = ''
    await (wrapper.vm as any).startTest()
    expect(global.fetch).not.toHaveBeenCalled()
    ;(wrapper.vm as any).probeExpectedAnswer = '21'
    await (wrapper.vm as any).startTest()
    await flushPromises()
    expect(JSON.parse((global.fetch as any).mock.calls[0][1].body)).toMatchObject({
      mode: 'pelican', evaluator: 'numeric', expected_answer: 21, prompt: '只回答数字：最少取多少颗糖果？'
    })
  })

  it('测智模式加载共享预设方案并默认选中上次使用的方案', async () => {
    storage.set(PELICAN_SCENARIO_STORAGE_KEY, JSON.stringify({
      scenarios: [
        { id: 'default', name: '默认方案', prompt: '默认鹈鹕提示词' },
        { id: 'sc-2', name: '骑独轮车的鹈鹕', prompt: '绘制一个骑独轮车的鹈鹕动画' }
      ],
      selectedId: 'sc-2'
    }))
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol' }
    ])

    const wrapper = mountModal({
      id: 42,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    ;(wrapper.vm as any).testMode = 'pelican'
    await flushPromises()

    expect((wrapper.vm as any).pelicanPrompt).toBe('绘制一个骑独轮车的鹈鹕动画')

    const scenarioSelect = wrapper.findAllComponents({ name: 'Select' })[2]
    expect(scenarioSelect.props('modelValue')).toBe('sc-2')
    expect(scenarioSelect.props('options')).toEqual([
      { value: 'default', label: '默认方案' },
      { value: 'sc-2', label: '骑独轮车的鹈鹕' },
      { value: '', label: '自定义' }
    ])
  })

  it('选择预设实时填充文本框，手动编辑切为自定义并按编辑内容发起测试', async () => {
    storage.set(PELICAN_SCENARIO_STORAGE_KEY, JSON.stringify({
      scenarios: [
        { id: 'sc-1', name: '默认方案', prompt: '预设方案一提示词' },
        { id: 'sc-2', name: '骑独轮车的鹈鹕', prompt: '绘制一个骑独轮车的鹈鹕动画' }
      ],
      selectedId: 'sc-2'
    }))
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.6-sol', display_name: 'GPT-5.6 Sol' }
    ])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountModal({
      id: 42,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'active'
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    ;(wrapper.vm as any).testMode = 'pelican'
    await flushPromises()

    ;(wrapper.vm as any).selectedPelicanScenarioId = 'sc-1'
    await flushPromises()
    expect((wrapper.vm as any).pelicanPrompt).toBe('预设方案一提示词')
    expect(JSON.parse(storage.get(PELICAN_SCENARIO_STORAGE_KEY)!).selectedId).toBe('sc-1')

    const promptInput = wrapper.find('textarea.textarea-stub')
    await promptInput.setValue('实时输入的临时提示词')
    await flushPromises()

    expect((wrapper.vm as any).selectedPelicanScenarioId).toBe('')

    ;(wrapper.vm as any).selectedModelId = 'gpt-5.6-sol'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toMatchObject({
      model_id: 'gpt-5.6-sol',
      prompt: '实时输入的临时提示词',
      mode: 'pelican'
    })
  })
})
