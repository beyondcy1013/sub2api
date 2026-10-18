import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountDetectionCenter from '../AccountDetectionCenter.vue'
const api=vi.hoisted(()=>({list:vi.fn(),rules:vi.fn(),settings:vi.fn(),records:vi.fn(),save:vi.fn()}))
vi.mock('@/api/admin',()=>({adminAPI:{accounts:{list:api.list,getRequestQualityProbeSettings:api.settings,updateRequestQualityProbeSettings:api.save,listDetectionRecords:api.records},degradationRules:{list:api.rules}}}))
vi.mock('vue-i18n',async()=>({...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),useI18n:()=>({t:(key:string)=>key})}))
const mountCenter=()=>mount(AccountDetectionCenter,{props:{show:true,initialTab:'manual'},global:{plugins:[createPinia()],stubs:{BaseDialog:{template:'<div><slot /></div>'},ScheduledTestsPanel:true,Select:true}}})
describe('AccountDetectionCenter',()=>{
 beforeEach(()=>{vi.clearAllMocks();api.list.mockResolvedValue({items:[{id:1,name:'A',platform:'openai'}],total:1});api.rules.mockResolvedValue([]);api.settings.mockResolvedValue({enabled:false,models:[],recent_minutes:10,account_scope:'selected',account_ids:[]});api.records.mockResolvedValue([])})
 it('opens one unified test only after selecting an account and clicking',async()=>{
  const wrapper=mountCenter();await flushPromises()
  expect(wrapper.emitted('test')).toBeUndefined()
  await wrapper.get('input[type="checkbox"]').setValue(true)
  const launch=wrapper.findAll('button').find(b=>b.text().includes('openTest'))!
  await launch.trigger('click');expect(wrapper.emitted('test')?.[0]?.[0]).toEqual([expect.objectContaining({id:1})])
  wrapper.unmount()
 })
 it('loads history only on the history tab',async()=>{
  const wrapper=mountCenter();await flushPromises();expect(api.records).not.toHaveBeenCalled()
  await wrapper.get('[data-test="detection-tab-records"]').trigger('click');await flushPromises()
  expect(api.records).toHaveBeenCalledWith({account_id:undefined,source:undefined,model_id:undefined,before_id:undefined})
  wrapper.unmount()
 })
 it('loads every account page',async()=>{
  api.list.mockResolvedValueOnce({items:[{id:1,name:'A',platform:'openai'}],total:2}).mockResolvedValueOnce({items:[{id:2,name:'B',platform:'openai'}],total:2})
  const wrapper=mountCenter();await flushPromises();expect(api.list).toHaveBeenCalledTimes(2);expect(wrapper.text()).toContain('#2');wrapper.unmount()
 })
})

describe('manual quality center',()=>{
 it('opens the history view for non-manual legacy tabs',async()=>{
  api.list.mockResolvedValue({items:[],total:0});api.rules.mockResolvedValue([])
  api.settings.mockResolvedValue({enabled:true,model_scope:'all',models:[],recent_minutes:10,account_scope:'all',account_ids:[],rule_ids:[],timeout_seconds:35})
  api.save.mockImplementation(async value=>value)
  const wrapper=mount(AccountDetectionCenter,{props:{show:true,initialTab:'records'},global:{plugins:[createPinia()],stubs:{BaseDialog:{template:'<div><slot /></div>'},Select:true}}})
  await flushPromises()
  expect(wrapper.get('[data-test="detection-tab-records"]').attributes('aria-selected')).toBe('true')
  expect(wrapper.emitted('settings-saved')).toBeUndefined()
  wrapper.unmount()
 })
})
