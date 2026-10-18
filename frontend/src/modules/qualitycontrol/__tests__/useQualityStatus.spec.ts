import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useQualityStatus } from '../useQualityStatus'
const api = vi.hoisted(() => ({ settings: vi.fn(), summaries: vi.fn() }))
vi.mock('../api', () => ({ loadQualitySettings: api.settings }))
vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { getQualityStatus: api.summaries } } }))
describe('page quality evidence batches', () => {
 beforeEach(() => { vi.clearAllMocks(); api.settings.mockResolvedValue({ status_recent_minutes: 5 }); api.summaries.mockResolvedValue([]) })
 it('batches a page, replaces records after pagination and keeps failures explicit', async () => {
  const accounts = ref([{ id: 1 }, { id: 2 }]), revision = ref(0)
  let status!: ReturnType<typeof useQualityStatus>
  const wrapper = mount(defineComponent({ setup() { status = useQualityStatus(accounts, revision); return () => null } }))
  await flushPromises()
  expect(api.summaries).toHaveBeenCalledTimes(1)
  expect(api.summaries).toHaveBeenCalledWith([1, 2])
  expect(status.state.value).toBe('ready')
  accounts.value = [{ id: 3 }]; await flushPromises()
  expect(api.summaries).toHaveBeenLastCalledWith([3])
  api.summaries.mockRejectedValue(new Error('offline')); revision.value++; await flushPromises()
  expect(status.state.value).toBe('error')
  wrapper.unmount()
 })
 it('rejects a stale page response and applies the configured window', async () => {
  let finishOld!: (value: unknown[]) => void
  api.summaries.mockReturnValueOnce(new Promise(resolve => { finishOld = resolve }))
  const accounts = ref([{ id: 1 }]), revision = ref(0)
  let status!: ReturnType<typeof useQualityStatus>
  const wrapper = mount(defineComponent({ setup() { status = useQualityStatus(accounts, revision); return () => null } }))
  api.settings.mockResolvedValue({ status_recent_minutes: 12 })
  api.summaries.mockResolvedValue([{ account_id: 2, id: 22 }])
  accounts.value = [{ id: 2 }]; await flushPromises()
  finishOld([{ account_id: 1, id: 11 }]); await flushPromises()
  expect(status.minutes.value).toBe(12)
  expect(Object.keys(status.summaries.value)).toEqual(['2'])
  wrapper.unmount()
 })
})
