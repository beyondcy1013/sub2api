import { describe, expect, it } from 'vitest'
import { pelicanPreviewDocument } from '../pelicanPreviewDocument'

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

    // 验证自适应缩放与 viewBox 修复脚本
    expect(doc).toContain('preserveAspectRatio')
    expect(doc).toContain('viewBox')
    expect(doc).toContain('scale(')

    // 原始 HTML 正确拼接在末尾
    expect(doc).toContain(rawHtml)
  })
})
