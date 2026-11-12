import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountPelicanModal from '../AccountPelicanModal.vue'
import accountPelicanModalSource from '../AccountPelicanModal.vue?raw'
import {
  clearPelicanResultCache,
  reloadPelicanResultCacheFromStorage,
  getCachedPelicanResult,
  setCachedPelicanResult
} from '@/utils/pelicanResultCache'

const questionAPI = vi.hoisted(() => ({ list: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), run: vi.fn() }))
let serverQuestions: any[] = []
let nextQuestionID = 1

const { copyToClipboard, updateDowngradedFlag, getAvailableModels } = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
  updateDowngradedFlag: vi.fn().mockResolvedValue({}),
  getAvailableModels: vi.fn().mockImplementation((id: number) => Promise.resolve([
    { id: `gpt-6-account-${id}-model`, type: 'model', display_name: `Account ${id} Model`, created_at: '' }
  ]))
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('@/api', () => ({
  adminAPI: {
    degradationRules: questionAPI,
    accounts: {
      runQualityRules: (...args: any[]) => questionAPI.run(...args),
      updateDowngradedFlag: (...args: any[]) => updateDowngradedFlag(...args),
      getAvailableModels: (...args: any[]) => getAvailableModels(...args)
    }
  }
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
      autoStart,
      initialMode: 'pelican'
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
    serverQuestions = []
    nextQuestionID = 1
    questionAPI.list.mockReset().mockImplementation(async () => structuredClone(serverQuestions))
    questionAPI.create.mockReset().mockImplementation(async (req) => {
      const rule = { ...req, id: nextQuestionID++, created_at: '', updated_at: '' }
      serverQuestions.push(rule)
      return structuredClone(rule)
    })
    questionAPI.update.mockReset().mockImplementation(async (id, req) => {
      const rule = { ...req, id, created_at: '', updated_at: '' }
      serverQuestions = serverQuestions.map(item => item.id === id ? rule : item)
      return structuredClone(rule)
    })
    questionAPI.delete.mockReset().mockImplementation(async id => { serverQuestions = serverQuestions.filter(item => item.id !== id) })
    questionAPI.run.mockReset().mockResolvedValue({status:'success',response_text:'keyword present',downgraded:false})
    clearPelicanResultCache()
    getAvailableModels.mockClear()
    getAvailableModels.mockImplementation((id: number) => Promise.resolve([
      { id: `gpt-6-account-${id}-model`, type: 'model', display_name: `Account ${id} Model`, created_at: '' }
    ]))
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

  it('keeps mobile history readable and falls back to source beside sandboxed animation', () => {
    expect(accountPelicanModalSource).toContain('hidden grid-cols-')
    expect(accountPelicanModalSource).toContain('lg:grid lg:grid-cols-[minmax(72px,84px)')
    expect(accountPelicanModalSource).toContain('data-test="`pelican-history-tab-${tab}`"')
    expect(accountPelicanModalSource).toContain('class="h-[56vh] min-h-[320px] w-full border-0"')
    expect(accountPelicanModalSource).toContain('v-show="!mobilePreviewSource"')
    expect(accountPelicanModalSource).not.toContain('lg:max-h-48')
    expect(accountPelicanModalSource).not.toContain('lg:max-h-64')
    expect(accountPelicanModalSource).not.toContain('line-clamp-4')
    expect(accountPelicanModalSource).toContain('data-test="pelican-streaming-output"')
    expect(accountPelicanModalSource).toContain('whitespace-pre-wrap break-words')
  })
  it('suggests all available models from the account catalog', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.6-sol', display_name: 'Old model' },
      { id: 'gpt-6.1-sol', display_name: 'GPT 6' },
      { id: 'claude-sonnet', display_name: 'Other model' }
    ])
    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.findAll('#pelican-model-presets option').map(option => option.attributes('value'))).toEqual(['claude-sonnet', 'gpt-5.6-sol', 'gpt-6.1-sol'])
    expect((wrapper.vm as any).accountStates[0].selectedModel).toBe('gpt-5.6-sol')
    await wrapper.get('[data-test="pelican-model-input"]').setValue('claude-custom')
    await (wrapper.vm as any).startAllTests()
    expect(JSON.parse((global.fetch as any).mock.calls[0][1].body).model_id).toBe('claude-custom')
    wrapper.unmount()
  })

  it('渲染所选账号列表与初始状态', async () => {
    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI Account 1')
    expect(wrapper.text()).toContain('OpenAI Account 2')
    expect((wrapper.vm as any).accountStates).toHaveLength(2)
    expect((wrapper.vm as any).accountStates[0].status).toBe('idle')
    // 测智模型目录只取本地配置，不触发上游模型发现
    expect(getAvailableModels).toHaveBeenCalledWith(1, { local: true })
    expect(getAvailableModels).toHaveBeenCalledWith(2, { local: true })
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
      model_id: 'gpt-6-account-1-model',
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

  it('完成后预览和源码标签可反复切换', async () => {
    const wrapper = mountModal([
      { id: 1, name: 'Preview Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()
    await (wrapper.vm as any).startAllTests()
    await flushPromises()
    const preview = wrapper.get('[data-test="pelican-card-tab-preview"]')
    const source = wrapper.get('[data-test="pelican-card-tab-source"]')
    for (let i = 0; i < 2; i++) {
      await source.trigger('click')
      expect(source.attributes('aria-selected')).toBe('true')
      expect(wrapper.get('[data-test="pelican-card-source"]').text()).toContain('<svg>')
      expect(wrapper.find('iframe[title="Animation Preview"]').exists()).toBe(false)
      await preview.trigger('click')
      expect(preview.attributes('aria-selected')).toBe('true')
      expect(wrapper.find('iframe[title="Animation Preview"]').exists()).toBe(true)
      expect(wrapper.find('[data-test="pelican-card-source"]').exists()).toBe(false)
    }
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

  it('pelican_result 到达后流尾部断开不推翻已判定的测智结论', async () => {
    const encoder = new TextEncoder()
    const resultChunk = encoder.encode(
      'data: {"type":"pelican_result","text":"测智通过","data":{"has_html":true,"html":"<!doctype html><html><body><svg></svg></body></html>","downgraded":false,"reason":"测智通过","response_model":"gpt-6-luna"}}\n'
    )
    let readCount = 0
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(async () => {
            readCount += 1
            if (readCount === 1) {
              return { done: false, value: resultChunk }
            }
            // 模拟浏览器在结果事件之后读取剩余流时连接被切断
            throw new TypeError('network error')
          })
        })
      }
    } as any) as any

    const wrapper = mountModal([
      { id: 21, name: 'Tail Cut Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    const state = (wrapper.vm as any).accountStates[0]
    await (wrapper.vm as any).runSingleTest(state)
    await flushPromises()

    expect(state.result?.has_html).toBe(true)
    expect(state.status).toBe('success')
    expect(state.error).toBeUndefined()
  })

  it('下拉选择模型后批量测试和重测均使用当前选择，重测清除旧模型与缓存时间', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-test="pelican-model-input"]').setValue('gpt-5.4')
    await (wrapper.vm as any).startAllTests()
    for (const [, request] of vi.mocked(fetch).mock.calls) {
      expect(JSON.parse(request!.body as string).model_id).toBe('gpt-5.4')
    }

    await wrapper.get('[data-test="pelican-model-input"]').setValue('gpt-6-astra')
    const state = (wrapper.vm as any).accountStates[0]
    state.responseModel = 'gpt-5.6-luna'
    state.testedAt = 123
    state.elapsedMs = 300
    vi.mocked(fetch).mockResolvedValueOnce({ ok: false, status: 502 } as Response)
    const retry = (wrapper.vm as any).runSingleTest(state)
    expect(state.responseModel).toBe('gpt-6-astra')
    expect(state.testedAt).toBeUndefined()
    expect(state.elapsedMs).toBeUndefined()
    await retry
    const [, request] = vi.mocked(fetch).mock.calls.at(-1)!
    expect(JSON.parse(request!.body as string).model_id).toBe('gpt-6-astra')
    expect(state.responseModel).toBe('gpt-6-astra')
    expect(state.status).toBe('failed')
    wrapper.unmount()
  })

  it('未指定批量覆盖模型时，每个账号使用自身模型目录中的默认模型', async () => {
    vi.mocked(fetch)
      .mockImplementationOnce(async () => ({
        ok: true,
        data: [
          { id: 'account-one-model', type: 'model', display_name: 'Account One Model', created_at: '' }
        ]
      }) as any)
      .mockImplementationOnce(async () => ({
        ok: true,
        data: [
          { id: 'account-two-model', type: 'model', display_name: 'Account Two Model', created_at: '' }
        ]
      }) as any)
      .mockImplementationOnce(async (_url, init) => {
        expect(JSON.parse(init!.body as string)).toMatchObject({ mode: 'pelican', model_id: 'gpt-6-account-51-model' })
        return createStreamResponse([
        'data: {"type":"test_start","model":"gpt-6-account-51-model"}\n',
          'data: {"type":"test_complete","success":true}\n'
        ]) as Response
      })
      .mockImplementationOnce(async (_url, init) => {
        expect(JSON.parse(init!.body as string)).toMatchObject({ mode: 'pelican', model_id: 'gpt-6-account-52-model' })
        return createStreamResponse([
        'data: {"type":"test_start","model":"gpt-6-account-52-model"}\n',
          'data: {"type":"test_complete","success":true}\n'
        ]) as Response
      })
    const wrapper = mountModal([
      { id: 51, name: 'Account One', platform: 'openai', type: 'oauth', status: 'active' },
      { id: 52, name: 'Account Two', platform: 'anthropic', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    expect(wrapper.find('[data-test="pelican-account-model-51"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="pelican-account-model-52"]').exists()).toBe(false)

    await (wrapper.vm as any).startAllTests()
    await flushPromises()
    const states = (wrapper.vm as any).accountStates
    expect(states.map((item: any) => item.responseModel)).toEqual(['gpt-6-account-51-model', 'gpt-6-account-52-model'])
    wrapper.unmount()
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

  it('单账号可独立开始和停止，批量中停止一个不影响其他账号', async () => {
    const signals: AbortSignal[] = []
    vi.mocked(fetch).mockImplementation((_url, init) => new Promise((_resolve, reject) => {
      const signal = init!.signal as AbortSignal
      signals.push(signal)
      signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
    }))
    const wrapper = mountModal([
      { id: 31, name: 'One', platform: 'openai', type: 'oauth', status: 'active' },
      { id: 32, name: 'Two', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()
    expect(wrapper.text()).toContain('admin.accounts.accountTesting.start')
    await wrapper.get('[data-test="pelican-test-31"]').trigger('click')
    expect(fetch).toHaveBeenCalledTimes(1)
    expect((wrapper.vm as any).accountStates[1].status).toBe('idle')
    await wrapper.get('[data-test="pelican-stop-31"]').trigger('click')
    await flushPromises()
    expect(signals[0].aborted).toBe(true)
    expect(getCachedPelicanResult(31)).toBeUndefined()
    const batch = (wrapper.vm as any).startAllTests()
    await flushPromises()
    await wrapper.get('[data-test="pelican-stop-31"]').trigger('click')
    await flushPromises()
    expect(signals[1].aborted).toBe(true)
    expect(signals[2].aborted).toBe(false)
    expect((wrapper.vm as any).accountStates[1].status).toBe('running')
    expect((wrapper.vm as any).isRunning).toBe(true)
    await wrapper.get('[data-test="pelican-stop-32"]').trigger('click')
    await batch
    expect((wrapper.vm as any).isRunning).toBe(false)
    wrapper.unmount()
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

  it('关闭弹窗后测智请求继续执行并保存结果', async () => {
    let resolveReader = () => {}
    const readGate = new Promise<void>(resolve => {
      resolveReader = resolve
    })
    global.fetch = vi.fn().mockImplementation(async () => {
      await readGate
      return createStreamResponse([
        'data: {"type":"test_start","model":"gpt-6-astra"}\n',
        'data: {"type":"pelican_result","text":"测智通过","data":{"has_html":true,"html":"<!doctype html><html><body><svg></svg></body></html>","downgraded":false,"reason":"测智通过","response_model":"gpt-6-astra"}}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    }) as any

    const wrapper = mountModal([
      { id: 77, name: 'Background Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    const running = (wrapper.vm as any).startAllTests()
    await flushPromises()
    expect((wrapper.vm as any).isRunning).toBe(true)

    await wrapper.setProps({ show: false })
    await flushPromises()
    expect((wrapper.vm as any).isRunning).toBe(true)
    expect(global.fetch).toHaveBeenCalledTimes(1)

    await wrapper.setProps({
      accounts: [{ id: 77, name: 'Background Account', platform: 'openai', type: 'oauth', status: 'active' }]
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect((wrapper.vm as any).accountStates[0].status).toBe('running')

    resolveReader()
    await running
    await flushPromises()

    expect((wrapper.vm as any).isRunning).toBe(false)
    expect((wrapper.vm as any).accountStates[0].status).toBe('success')
    expect(getCachedPelicanResult(77)?.status).toBe('success')
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

  it('显式传入 autoStart 为 true 且存在上一次测智缓存时，使用上一次的测智参数自动开始测智', async () => {
    setCachedPelicanResult({
      account: { id: 101, name: 'Previous Param Account' },
      status: 'success',
      testedAt: Date.now() - 5000,
      responseModel: 'gpt-6.1-sol-previous',
      prompt: '上一次的自定义测智题目',
      testMode: 'pelican'
    })

    const wrapper = mountModal(
      [{ id: 101, name: 'Previous Param Account', platform: 'openai', type: 'oauth', status: 'active' }],
      true
    )
    await flushPromises()
    await new Promise(r => setTimeout(r, 120))
    await flushPromises()

    expect(global.fetch).toHaveBeenCalled()
    const requestCalls = (global.fetch as any).mock.calls
    const testCall = requestCalls.find((call: any[]) => typeof call[0] === 'string' && call[0].includes('/admin/accounts/101/test'))
    expect(testCall).toBeTruthy()
    const body = JSON.parse(testCall[1].body)
    expect(body.model_id).toBe('gpt-6.1-sol-previous')
    expect(body.prompt).toBe('上一次的自定义测智题目')
  })

  it('支持选择或输入大模型并正确传递至后端', async () => {
    const wrapper = mountModal([
      { id: 10, name: 'Model Test Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    // 通过界面输入未预置的大模型 ID 与思考程度。
    await wrapper.get('[data-test="pelican-model-input"]').setValue('gpt-5.6-luna-custom')
    await wrapper.get('[data-test="pelican-model-input"]').trigger('blur')
    expect(JSON.parse(localStorage.getItem('sub2api:account-pelican-model-history:v1') || '[]')).toEqual([
      'gpt-5.6-luna-custom'
    ])
    expect(wrapper.findAll('#pelican-model-presets option').map(option => option.attributes('value'))).toContain('gpt-5.6-luna-custom')
    ;(wrapper.vm as any).reasoningEffort = 'high'
    await flushPromises()

    await (wrapper.vm as any).startAllTests()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalled()
    const [, req] = (global.fetch as any).mock.calls[(global.fetch as any).mock.calls.length - 1]
    const body = JSON.parse(req.body)
    expect(body.model_id).toBe('gpt-5.6-luna-custom')
    expect(body.reasoning_effort).toBe('high')
  })

  it('默认不批量覆盖模型且思考程度默认为 low', async () => {
    const wrapper = mountModal()
    await flushPromises()

    expect((wrapper.vm as any).selectedModel).toBe('')
    expect((wrapper.vm as any).reasoningEffort).toBe('low')
  })

  it('新测试立即刷新历史并展示完整提示词、输出，重开后仍可读取', async () => {
    const wrapper = mountModal()
    const prompt = '完整提示词\n' + '动画要求'.repeat(30)
    ;(wrapper.vm as any).customPrompt = prompt
    await wrapper.get('[data-test="pelican-history-toggle"]').trigger('click')
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]')).toHaveLength(0)
    await (wrapper.vm as any).startAllTests()
    await flushPromises()
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]').length).toBeGreaterThan(0)
    await wrapper.get('[data-test="pelican-global-history-entry"]').trigger('click')
    expect(wrapper.get('[data-test="pelican-history-prompt"]').text()).toBe(prompt)
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toContain('<svg>')
    wrapper.unmount()
    reloadPelicanResultCacheFromStorage()
    const reopened = mountModal()
    await reopened.get('[data-test="pelican-history-toggle"]').trigger('click')
    expect(reopened.get('[data-test="pelican-history-prompt"]').text()).toBe(prompt)
    expect(reopened.get('[data-test="pelican-history-output"]').text()).toContain('<svg>')
    reopened.unmount()
  })

  it.each([
    { result: { has_html: true, html: '<html><body><svg><animate /></svg></body></html>', downgraded: false, reason: '' } },
    { result: { has_html: false, html: '<svg><animate /></svg>', downgraded: true, reason: '' } },
    { output: '动画如下：\n```html\n<html><body><svg><animate /></svg></body></html>\n```' },
    { output: '```svg\n<svg><animate /></svg>\n```' }
  ])('历史默认显示沙箱动画并折叠源码，兼容旧缓存 %#', async (content) => {
    setCachedPelicanResult({ account: { id: 20, name: 'Animation' }, status: 'success', testedAt: 1000, ...content })
    reloadPelicanResultCacheFromStorage()
    const wrapper = mountModal([{ id: 20, name: 'Animation', platform: 'openai', type: 'apikey', status: 'active' }])
    await wrapper.get('[data-test="pelican-history-toggle"]').trigger('click')
    const preview = wrapper.get('[data-test="pelican-history-preview"]')
    expect(preview.attributes('srcdoc')).toContain('<svg><animate /></svg>')
    expect(preview.attributes('srcdoc')).not.toContain('```')
    expect(preview.attributes('sandbox')).toBe('allow-scripts')
    expect(preview.attributes('srcdoc')).toContain("connect-src 'none'")
    expect(wrapper.get('[data-test="pelican-history-source"]').attributes('open')).toBeUndefined()
    setCachedPelicanResult({ account: { id: 20, name: 'Animation' }, status: 'failed', testedAt: 2000, output: '普通文本，无动画' })
    await flushPromises()
    await wrapper.get('[data-test="pelican-global-history-entry"]').trigger('click')
    expect(wrapper.find('[data-test="pelican-history-preview"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="pelican-history-source"]').attributes('open')).toBeDefined()
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('普通文本，无动画')
    wrapper.unmount()
  })

  it('API Key 历史按账号 ID 隔离，单账号详情与选择变化不串号', async () => {
    const accounts = [95, 96, 97].map(id => ({
      id, name: '同名账号', platform: 'openai', type: 'apikey', status: 'active'
    }))
    for (const account of accounts) {
      setCachedPelicanResult({
        account, status: 'success', testedAt: account.id,
        responseModel: 'gpt-6-astra', prompt: `prompt-${account.id}`,
        output: `output-${account.id}`
      })
    }
    const wrapper = mountModal([accounts[0]])
    await wrapper.setProps({ allAccounts: accounts as any })
    await wrapper.get('[data-test="pelican-history-toggle"]').trigger('click')
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]')).toHaveLength(1)
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('output-95')
    const vm = wrapper.vm as any
    await vm.toggleAccountSelection(accounts[1])
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]')).toHaveLength(2)
    await vm.selectHistoryResult(vm.accountStates[1], 0)
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]')).toHaveLength(1)
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('output-96')
    await wrapper.get('[data-test="pelican-history-account"]').setValue('95')
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('output-95')
    await vm.toggleAccountSelection(accounts[0])
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('output-96')
    await vm.toggleAccountSelection(accounts[1])
    expect(wrapper.findAll('[data-test="pelican-global-history-entry"]')).toHaveLength(0)
    expect(wrapper.find('[data-test="pelican-history-output"]').exists()).toBe(false)
    expect(getCachedPelicanResult(97)?.output).toBe('output-97')
    wrapper.unmount()
  })

  it('历史普通文本和失败详情可查看，标记不会改写原始提示词或新增记录', async () => {
    setCachedPelicanResult({
      account: { id: 20, name: 'History Account' }, status: 'downgraded',
      testedAt: 1000, prompt: '原始提示词', output: '纯文本输出\n第二行'
    })
    const wrapper = mountModal([
      { id: 20, name: 'History Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    const vm = wrapper.vm as any
    vm.customPrompt = '另一个提示词'
    await vm.selectHistoryResult(vm.accountStates[0], 0)
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('纯文本输出\n第二行')
    expect(wrapper.get('[data-test="pelican-history-prompt"]').text()).toBe('原始提示词')
    await vm.manualMarkItem(vm.accountStates[0], false)
    expect(vm.historyCount(20)).toBe(1)
    expect(getCachedPelicanResult(20)?.prompt).toBe('原始提示词')
    setCachedPelicanResult({ account: { id: 20, name: 'History Account' }, status: 'failed', testedAt: 2000, error: 'HTTP 404 model missing' })
    await flushPromises()
    await wrapper.get('[data-test="pelican-global-history-entry"]').trigger('click')
    expect(wrapper.get('[data-test="pelican-history-output"]').text()).toBe('HTTP 404 model missing')
    expect(wrapper.get('[data-test="pelican-history-prompt"]').text()).toBe('admin.accounts.pelicanPromptMissing')
    wrapper.unmount()
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
      .toMatchObject({ selectedId: 'rule-1', scenarios: [{ prompt: 'Pelican on spaceship' }] })

    ;(wrapper.vm as any).addPromptScenario()
    await flushPromises()

    const selects = wrapper.findAll('select')
    expect(selects.map(select => (select.element as HTMLSelectElement).value)).toContain((wrapper.vm as any).selectedPromptId)
    expect(selects.length).toBeGreaterThanOrEqual(3)

    expect((wrapper.vm as any).promptScenarios).toHaveLength(2)
    expect((wrapper.vm as any).customPrompt).toContain('admin.accounts.pelicanPromptDefault')
    expect(JSON.parse(localStorage.getItem('sub2api:account-pelican-prompt-scenarios:v1') || 'null').selectedId)
      .not.toBe('default')

    const draftId = (wrapper.vm as any).selectedPromptId
    await wrapper.get('[data-test="account-test-question"]').setValue('rule-1')
    await flushPromises()
    expect((wrapper.vm as any).customPrompt).toBe('Pelican on spaceship')

    await wrapper.get('[data-test="account-test-question"]').setValue(draftId)
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
    expect(wrapper.get('[data-test="account-test-question"]').text()).toContain('Create a long pelican')
    expect(wrapper.get('[data-test="pelican-prompt-save"]').attributes()).toHaveProperty('disabled')
  })

  it('支持在选中账号后直接在“开始测智”后面点击标记降智或满血', async () => {
    updateDowngradedFlag.mockClear()
    const wrapper = mountModal([{ id: 1, name: 'Test Account', platform: 'openai', type: 'oauth', status: 'active' }])
    await flushPromises()

    const item = (wrapper.vm as any).accountStates[0]
    expect(item.status).toBe('idle')

    // 选中账号后，在“开始测智”后面即可看到“标记降智”与“标记满血”按钮
    const markDowngradedBtn = wrapper.get('[data-test="pelican-mark-downgraded-batch"]')
    const markNormalBtn = wrapper.get('[data-test="pelican-mark-normal-batch"]')
    expect(markDowngradedBtn.text()).toContain('admin.accounts.pelicanMarkDowngraded')
    expect(markNormalBtn.text()).toContain('admin.accounts.pelicanMarkNormal')
    expect(markDowngradedBtn.attributes('disabled')).toBeUndefined()
    expect(markNormalBtn.attributes('disabled')).toBeUndefined()

    // 点击“标记降智”
    await markDowngradedBtn.trigger('click')
    await flushPromises()

    expect(updateDowngradedFlag).toHaveBeenCalledWith(1, true)
    expect(item.status).toBe('downgraded')
    expect(item.account.extra.pelican_downgraded).toBe(true)

    // 点击“标记满血”
    await markNormalBtn.trigger('click')
    await flushPromises()

    expect(updateDowngradedFlag).toHaveBeenCalledWith(1, false)
    expect(item.status).toBe('success')
    expect(item.account.extra.pelican_downgraded).toBe(false)
  })
  it('opens the shared workspace without hardcoded undeletable questions and allows deleting all scenarios', async () => {
    const wrapper = mount(AccountPelicanModal, {
      props: { show: true, accounts: [{ id: 1, name: 'One', platform: 'anthropic', type: 'apikey', status: 'active' }] as any },
      global: { stubs: { BaseDialog: {template: '<div><slot /><slot name="footer" /></div>'}, Icon: true } }
    })
    await flushPromises()
    expect(wrapper.find('[data-test="account-test-question"] option[value="connection"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="account-test-question"] option[value="numeric"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="account-test-evaluator"]').exists()).toBe(true)
    expect(fetch).not.toHaveBeenCalled()
    await wrapper.get('[data-test="account-test-evaluator"]').setValue('numeric')
    expect((wrapper.vm as any).probeEvaluator).toBe('numeric')
    await wrapper.get('[data-test="account-test-evaluator"]').setValue('reply')
    expect((wrapper.vm as any).probeEvaluator).toBe('reply')
    // Verify any selected scenario can be deleted with confirmation dialog
    const deleteBtn = wrapper.findAll('button').find(b => b.text().includes('admin.accounts.pelicanPromptDelete'))
    expect(deleteBtn?.exists()).toBe(true)
    expect(deleteBtn?.attributes('disabled')).toBeUndefined()
    // Trigger delete opens confirm dialog
    await deleteBtn?.trigger('click')
    await flushPromises()
    expect((wrapper.vm as any).showDeleteScenarioDialog).toBe(true)
    // Confirm delete and verify it removes the scenario
    await (wrapper.vm as any).confirmDeletePromptScenario()
    await flushPromises()
    expect((wrapper.vm as any).promptScenarios).toHaveLength(0)
    expect((wrapper.vm as any).selectedPromptId).toBe('')
    wrapper.unmount()
  })

  it('allows adding and saving a new question after deleting all scenarios', async () => {
    const wrapper = mountModal([{ id: 1, name: 'One', platform: 'anthropic', type: 'apikey', status: 'active' }])
    await flushPromises()
    const deleteBtn = wrapper.findAll('button').find(b => b.text().includes('admin.accounts.pelicanPromptDelete'))
    await deleteBtn?.trigger('click')
    await flushPromises()
    expect((wrapper.vm as any).showDeleteScenarioDialog).toBe(true)
    await (wrapper.vm as any).confirmDeletePromptScenario()
    await flushPromises()
    expect((wrapper.vm as any).promptScenarios).toHaveLength(0)

    const addBtn = wrapper.findAll('button').find(b => b.text().includes('admin.accounts.pelicanPromptAdd'))
    await addBtn?.trigger('click')
    await flushPromises()
    expect((wrapper.vm as any).promptScenarios).toHaveLength(1)
    expect((wrapper.vm as any).selectedPromptId).toBeTruthy()
    wrapper.unmount()
  })

  it('numeric templates send the exact prompt and expected answer and persist evaluator settings', async () => {
    const wrapper = mountModal([{ id: 1, name: 'One', platform: 'anthropic', type: 'apikey', status: 'active' }])
    await flushPromises()
    await wrapper.get('[data-test="account-test-evaluator"]').setValue('numeric')
    await wrapper.get('[data-test="account-test-expected-answer"]').setValue('17')
    await wrapper.get('textarea.textarea-stub').setValue('Only answer 17')
    await wrapper.get('[data-test="pelican-prompt-save"]').trigger('click')
    await flushPromises()
    await (wrapper.vm as any).startAllTests()
    expect(JSON.parse(vi.mocked(fetch).mock.calls[0][1]!.body as string)).toMatchObject({mode:'pelican',prompt:'Only answer 17',evaluator:'numeric',expected_answer:17})
    const scenarios = JSON.parse(localStorage.getItem('sub2api:account-pelican-prompt-scenarios:v1')!)
    expect(scenarios.scenarios).toContainEqual(expect.objectContaining({evaluator:'numeric',expectedAnswer:17}))
    wrapper.unmount()
  })

  it('numeric validation blocks empty or fractional expected answers for both batch and individual tests', async () => {
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('[data-test="account-test-evaluator"]').setValue('numeric')
    await (wrapper.vm as any).startAllTests()
    expect(fetch).not.toHaveBeenCalled()
    await wrapper.get('[data-test="account-test-expected-answer"]').setValue('1.5')
    await (wrapper.vm as any).runSingleTest((wrapper.vm as any).accountStates[0])
    expect(fetch).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toBe('admin.accounts.probeExpectedAnswer')
    wrapper.unmount()
  })

  it('an incomplete connection response fails without classifying capability or changing its flag', async () => {
    updateDowngradedFlag.mockClear()
    vi.mocked(fetch).mockResolvedValue(createStreamResponse(['data: {"type":"content","text":"partial"}\n']))
    const wrapper = mountModal([{ id: 1, name: 'One', platform: 'openai', type: 'apikey', status: 'active' }])
    await flushPromises()
    await wrapper.get('[data-test="account-test-evaluator"]').setValue('reply')
    await (wrapper.vm as any).startAllTests()
    expect((wrapper.vm as any).accountStates[0]).toMatchObject({status:'failed',streamingContent:'partial'})
    expect(updateDowngradedFlag).not.toHaveBeenCalled()
    expect(wrapper.emitted('test-failed')).toHaveLength(1)
    wrapper.unmount()
  })

  it('preserves Compact and generated image connection tests in the same executor', async () => {
    vi.mocked(fetch).mockImplementation(async () => createStreamResponse([
      'data: {"type":"image","image_url":"data:image/png;base64,aGVsbG8="}\n',
      'data: {"type":"test_complete","success":true}\n'
    ]))
    const wrapper = mountModal([{ id: 1, name: 'One', platform: 'openai', type: 'apikey', status: 'active' }])
    await flushPromises()
    await wrapper.get('[data-test="account-test-evaluator"]').setValue('reply')
    await wrapper.get('[data-test="account-test-compact"]').setValue(true)
    await (wrapper.vm as any).startAllTests()
    expect(JSON.parse(vi.mocked(fetch).mock.calls[0][1]!.body as string).mode).toBe('compact')
    expect(wrapper.find('img').exists()).toBe(true)
    await wrapper.get('[data-test="pelican-card-tab-output"]').trigger('click')
    expect(wrapper.find('img').exists()).toBe(false)
    await wrapper.get('[data-test="pelican-card-tab-preview"]').trigger('click')
    expect(wrapper.find('img').exists()).toBe(true)
    wrapper.unmount()
  })

  it('migrates desktop questions once and loads them on a fresh Android browser', async () => {
    localStorage.setItem('sub2api:account-pelican-prompt-scenarios:v1', JSON.stringify({
      selectedId: 'legacy', scenarios: [{id:'legacy', name:'Zero', prompt:'Only answer 0', evaluator:'numeric', expectedAnswer:0}]
    }))
    const desktop = mountModal()
    await flushPromises()
    expect(questionAPI.create).toHaveBeenCalledOnce()
    expect(serverQuestions[0]).toMatchObject({enabled:false, params:{probe_evaluator:'numeric', expected_answer:0}})
    desktop.unmount()
    localStorage.clear()
    const android = mountModal()
    await flushPromises()
    expect(android.get('[data-test="account-test-question"]').text()).toContain('Only answer 0')
    await android.get('[data-test="account-test-question"]').setValue('rule-1')
    expect((android.vm as any).probeExpectedAnswer).toBe('0')
    expect(questionAPI.create).toHaveBeenCalledOnce()
    android.unmount()
  })

  it('shares edits and deletion on reopen without resurrecting cached server IDs', async () => {
    serverQuestions = [{id:7, name:'Question', prompt:'Say original', evaluator:'refusal_only', enabled:true, params:{}, created_at:'', updated_at:''}]
    const desktop = mountModal()
    await flushPromises()
    await desktop.get('[data-test="account-test-question"]').setValue('rule-7')
    await desktop.get('textarea.textarea-stub').setValue('Say updated')
    await desktop.get('[data-test="pelican-prompt-save"]').trigger('click')
    await flushPromises()
    expect(questionAPI.update).toHaveBeenCalledWith(7, expect.objectContaining({prompt:'Say updated', enabled:true}))
    desktop.unmount()
    const android = mountModal()
    await flushPromises()
    expect((android.vm as any).customPrompt).toBe('Say updated')
    await (android.vm as any).deletePromptScenario()
    expect(questionAPI.delete).toHaveBeenCalledWith(7)
    android.unmount()
    localStorage.setItem('sub2api:account-pelican-prompt-scenarios:v1', JSON.stringify({selectedId:'rule-7',scenarios:[{id:'rule-7',name:'Old',prompt:'Say updated',evaluator:'reply'}]}))
    const staleBrowser = mountModal()
    await flushPromises()
    expect(staleBrowser.get('[data-test="account-test-question"]').text()).not.toContain('Say updated')
    expect(questionAPI.create).not.toHaveBeenCalled()
    staleBrowser.unmount()
  })

  it('retains questions beyond the former twenty-question browser limit', async () => {
    serverQuestions = Array.from({length:25}, (_, i) => ({id:i+1,name:`Q${i}`,prompt:`Question ${i}`,evaluator:'refusal_only',enabled:false,params:{}}))
    const wrapper = mountModal()
    await flushPromises()
    expect(wrapper.get('[data-test="account-test-question"]').findAll('option')).toHaveLength(26)
    expect(wrapper.get('[data-test="account-test-question"]').text()).toContain('Question 24')
    wrapper.unmount()
  })

  it('keeps an edited draft when server saving fails and sends the draft verbatim', async () => {
    serverQuestions = [{id:7,name:'Original',prompt:'Original',evaluator:'refusal_only',enabled:false,params:{}}]
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('[data-test="account-test-question"]').setValue('rule-7')
    await wrapper.get('textarea.textarea-stub').setValue('Edited draft')
    questionAPI.update.mockRejectedValueOnce(new Error('offline'))
    await wrapper.get('[data-test="pelican-prompt-save"]').trigger('click')
    await flushPromises()
    expect((wrapper.vm as any).customPrompt).toBe('Edited draft')
    expect(wrapper.get('[role="alert"]').text()).toContain('offline')
    await (wrapper.vm as any).startAllTests()
    expect(JSON.parse(vi.mocked(fetch).mock.calls[0][1]!.body as string)).toMatchObject({prompt:'Edited draft',evaluator:'reply'})
    expect(questionAPI.run).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps keyword questions on their authoritative shared evaluator', async () => {
    serverQuestions = [{id:7,name:'Keywords',prompt:'Say keyword present',evaluator:'contains_keywords',enabled:false,params:{keywords:['keyword'],keyword_match:'all'}}]
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('[data-test="account-test-question"]').setValue('rule-7')
    await (wrapper.vm as any).startAllTests()
    expect(fetch).not.toHaveBeenCalled()
    expect(questionAPI.run).toHaveBeenCalledWith(1, expect.any(String), [7], expect.any(AbortSignal))
    expect((wrapper.vm as any).accountStates[0].status).toBe('success')
    wrapper.unmount()
  })

  it('does not select the first account implicitly when opening from the toolbar', async () => {
    const wrapper = mountModal([])
    await wrapper.setProps({allAccounts: [{id:9,name:'Candidate',platform:'openai',type:'apikey',status:'active'}] as any})
    await flushPromises()
    expect((wrapper.vm as any).accountStates).toHaveLength(0)
    expect((wrapper.vm as any).showAccountPicker).toBe(false)
    expect(fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('formats object API errors without showing [object Object] and displays near top header', async () => {
    questionAPI.list.mockRejectedValueOnce({ status: 400, message: 'duplicate key error' })
    const wrapper = mountModal()
    await flushPromises()
    const alert = wrapper.get('[role="alert"]')
    expect(alert.text()).toContain('duplicate key error')
    expect(alert.text()).not.toContain('[object Object]')
    expect(wrapper.get('.max-w-2xl').find('[role="alert"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('从该账号设置中的大模型拉取下拉列表且测试题目编辑区保持紧凑行高', async () => {
    localStorage.setItem('sub2api:account-pelican-model-history:v1', JSON.stringify(['history-unrelated-model']))
    getAvailableModels.mockResolvedValue([
      { id: 'gpt-5.6-sol', display_name: 'GPT 5.6 Sol' },
      { id: 'gpt-4o', display_name: 'GPT-4o' }
    ])
    const wrapper = mountModal([
      { id: 42, name: 'Special Account', platform: 'openai', type: 'oauth', status: 'active' }
    ])
    await flushPromises()

    // 测试大模型下拉列表 <select> 应只从该账号设置中拉取，不被外部历史污染
    const selectOptions = wrapper.findAll('select[aria-label="admin.accounts.pelicanModel"] option')
      .map(o => o.attributes('value'))
    expect(selectOptions).toContain('')
    expect(selectOptions).toContain('gpt-4o')
    expect(selectOptions).toContain('gpt-5.6-sol')
    expect(selectOptions).not.toContain('history-unrelated-model')

    // datalist 仍保留输入历史供自由输入自动完成
    const datalistOptions = wrapper.findAll('#pelican-model-presets option')
      .map(o => o.attributes('value'))
    expect(datalistOptions).toContain('history-unrelated-model')

    // 测试题目文本框保持为 2 行且带有紧凑高度限制，防止超出整页翻页
    const textarea = wrapper.find('textarea[placeholder="admin.accounts.pelicanPromptPlaceholder"]')
    expect(textarea.exists()).toBe(true)
    expect(textarea.attributes('rows')).toBe('2')
    expect(textarea.classes()).toContain('max-h-20')

    wrapper.unmount()
  })
})
