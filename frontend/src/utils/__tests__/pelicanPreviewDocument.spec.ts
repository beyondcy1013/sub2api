import { describe, expect, it, afterEach } from 'vitest'
import { currentCspNonce, pelicanPreviewDocument } from '../pelicanPreviewDocument'

describe('pelicanPreviewDocument', () => {
  it('正确注入 CSP、安全锁定和响应式适配脚本与样式', () => {
    const rawHtml = '<svg width="800" height="600"><circle cx="400" cy="300" r="50"/></svg>'
    const doc = pelicanPreviewDocument(rawHtml)

    // 验证 CSP 与 Referrer
    expect(doc).toContain("Content-Security-Policy")
    expect(doc).toContain("default-src 'none'")
    expect(doc).toContain('name="referrer" content="no-referrer"')
    expect(doc).toContain('name="viewport"')

    // 验证样式重置与居中
    expect(doc).toContain('overflow: hidden !important')
    expect(doc).toContain('object-fit: contain !important')
    expect(doc).toContain('width: 100% !important')
    expect(doc).toContain('height: auto !important')

    // 验证自适应缩放与 viewBox 修复脚本
    expect(doc).toContain('preserveAspectRatio')
    expect(doc).toContain('viewBox')
    expect(doc).toContain('scale(')

    // 原始 HTML 正确拼接在末尾
    expect(doc).toContain(rawHtml)
  })

  it('srcdoc 会继承宿主 CSP，因此所有内联脚本都带上宿主 nonce', () => {
    const rawHtml = '<html><body><svg></svg><script>requestAnimationFrame(() => {})</script></body></html>'
    const doc = pelicanPreviewDocument(rawHtml, 'abc123')

    // 适配与锁定脚本同样需要 nonce，否则在 nonce-only CSP 下会被拦截
    expect(doc).toContain('<script nonce="abc123">')
    expect(doc).toContain('<script nonce="abc123">requestAnimationFrame')
    // 未传入 nonce 时不注入，保持旧行为
    expect(pelicanPreviewDocument(rawHtml, '')).not.toContain('nonce=')
  })

  it('覆盖生成内容自带的 nonce，避免重复属性', () => {
    const doc = pelicanPreviewDocument('<script nonce="stale" src="a.js"></script><script type="module">x()</script>', 'abc123')

    expect(doc).not.toContain('stale')
    expect(doc).toContain('<script src="a.js" nonce="abc123">')
    expect(doc).toContain('<script type="module" nonce="abc123">')
  })

  it('从宿主文档读取 CSP nonce', () => {
    const el = document.createElement('script')
    el.setAttribute('nonce', 'host-nonce')
    document.head.appendChild(el)
    expect(currentCspNonce()).toBe('host-nonce')
    el.remove()
    expect(currentCspNonce()).toBe('')
  })
})
