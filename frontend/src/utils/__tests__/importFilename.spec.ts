import { describe, it, expect } from 'vitest'
import { extractImportFilenamePart } from '../importFilename'

describe('extractImportFilenamePart', () => {
  it('splits at the last underscore and returns the part before it', () => {
    expect(extractImportFilenamePart('chatgpt_pro_us_001.json')).toBe('chatgpt_pro_us')
  })

  it('returns the whole stem when there is no underscore', () => {
    expect(extractImportFilenamePart('accounts.json')).toBe('accounts')
  })

  it('returns the stem when there is no extension', () => {
    expect(extractImportFilenamePart('chatgpt_pro_us_001')).toBe('chatgpt_pro_us')
  })

  it('handles a single underscore', () => {
    expect(extractImportFilenamePart('us_001.json')).toBe('us')
  })

  it('strips only the last extension for double extensions', () => {
    expect(extractImportFilenamePart('a.b_c.json')).toBe('a.b')
  })

  it('returns empty string for empty input', () => {
    expect(extractImportFilenamePart('')).toBe('')
  })

  it('strips a leading path', () => {
    expect(extractImportFilenamePart('/tmp/foo/chatgpt_pro_us_001.json')).toBe('chatgpt_pro_us')
  })
})
