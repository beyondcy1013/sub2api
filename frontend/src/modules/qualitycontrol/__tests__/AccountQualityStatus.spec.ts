import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { AccountQualitySummary } from '@/types'
import AccountQualityStatus from '../AccountQualityStatus.vue'
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))
const now = Date.parse('2026-10-17T03:00:00Z')
const record: AccountQualitySummary = { account_id: 1, id: 99, model_id: 'model-a', source: 'request', finished_at: new Date(now - 4 * 60000).toISOString(), status: 'success', downgraded: false, reason: '', response_summary: '42', questions: [{ name: 'math', prompt: '6 times 7?', passed: true, reason: 'correct' }] }
const render = (summary?: AccountQualitySummary, state: 'ready' | 'error' | 'loading' = 'ready') => mount(AccountQualityStatus, { props: { summary, state, now, minutes: 5 } })
describe('recent account degradation evidence', () => {
 it('shows a recent passed test with question, model, record ID and answer', () => {
  const wrapper = render(record)
  expect(wrapper.text()).toContain('recentPassed')
  for (const part of ['6 times 7?', 'correct', 'model-a', '#99', '42']) expect(wrapper.attributes('title')).toContain(part)
 })
 it('shows degradation only when the capability test completed successfully', () => {
  expect(render({ ...record, downgraded: true }).text()).toContain('recentDowngraded')
  expect(render({ ...record, status: 'failed', downgraded: false, reason: 'timeout' }).text()).toContain('testFailed')
 })
 it.each(['cancelled', 'skipped'])('keeps %s separate from a passed test', status => {
  expect(render({ ...record, status }).text()).toContain(status === 'cancelled' ? 'testCancelled' : 'testSkipped')
 })
 it('expires as time advances and honors the configured window', async () => {
  const wrapper = render(record)
  await wrapper.setProps({ now: now + 60001 })
  expect(wrapper.text()).toContain('noRecentTest')
  expect(wrapper.attributes('title')).toContain('#99')
  await wrapper.setProps({ minutes: 10 })
  expect(wrapper.text()).toContain('recentPassed')
 })
 it('includes the exact boundary but excludes future records', () => {
  expect(render({ ...record, finished_at: new Date(now - 5 * 60000).toISOString() }).text()).toContain('recentPassed')
  expect(render({ ...record, finished_at: new Date(now + 1).toISOString() }).text()).toContain('noRecentTest')
 })
 it('never treats missing data or load errors as passed tests', () => {
  expect(render().text()).toContain('noRecentTest')
  expect(render(record, 'error').text()).toContain('statusError')
  expect(render(record, 'loading').text()).toContain('statusLoading')
 })
})
