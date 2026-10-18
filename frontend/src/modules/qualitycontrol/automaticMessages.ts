export const automaticDetectionMessages = {
 zh: {
  title: '周期任务', connection: '状态检测与恢复', quality: '周期测智', billing: '费率探测', enabled: '启用', all: '所有账号', selected: '指定账号', scope: '账号范围',
  hint: '所有账号在每轮执行时动态匹配，包含后续新增账号；范围与目标状态同时生效。归档、回收站及过期账号不参与。',
  states: '目标状态（任一匹配）', active: '正常', inactive: '停用', error: '错误', paused: '暂停调度', rate_limited: '限流 / 临时阻断', downgraded: '疑似降智',
  includePaused: '纳入暂停调度账号（允许按策略自动恢复）', platforms: '平台（不选表示全部）', interval: '检测间隔（分钟）', model: '模型（留空使用账号本地模型目录）',
  recover: '正常后清理错误、限流及运行时阻断', activate: '正常后激活停用账号', resume: '正常后启用调度',
  standard: '测智使用当前指定的自动测智标准试题，仅支持 OpenAI 平台。连接通过不会解除疑似降智状态。', noStandard: '尚未指定标准试题，请先在试题库中指定后再开启周期测智。',
  pause: '连续测智不通过后暂停调度', threshold: '连续失败阈值', save: '保存策略', saved: '自动检测策略已保存', refresh: '刷新运行情况', running: '正在检测', idle: '等待执行', next: '下次检查', none: '尚无执行记录',
  summary: '检测 {checked} · 正常 {succeeded} · 失败 {failed} · 跳过 {skipped} · 恢复 {recovered}', results: '最近执行与恢复结果', success: '正常', failed: '失败', skipped: '跳过',
  billingHint: '复用现有费率执行器，仅检测支持账单查询的 active API Key 账号。成功同步继续遵守账号的手动锁定设置，不会恢复账号状态。',
  legacyBilling: '当前沿用账号各自的费率探测开关；保存此处后改用所选范围。', billingSave: '保存费率探测', billingSaved: '费率探测已保存', empty: '指定账号时至少选择一个账号。',
  policyTab: '自动策略', scheduled: '独立定时计划', dirty: '有未保存修改；刷新仅更新运行结果。', billingRuntime: '费率运行情况', status: '状态', account: '账号', finished: '完成时间', notEnabled: '未启用'
 },
 en: {
  title: 'Periodic tasks', connection: 'Health detection and recovery', quality: 'Periodic quality tests', billing: 'Rate probes', enabled: 'Enabled', all: 'All accounts', selected: 'Selected accounts', scope: 'Account scope',
  hint: 'All accounts are matched each cycle, including newly added accounts. Scope and target states apply together. Archived, trashed and expired accounts are excluded.',
  states: 'Target states (match any)', active: 'Active', inactive: 'Inactive', error: 'Error', paused: 'Scheduling paused', rate_limited: 'Rate limited / temporarily blocked', downgraded: 'Suspected downgrade',
  includePaused: 'Include paused accounts (allow configured recovery)', platforms: 'Platforms (none selected means all)', interval: 'Interval (minutes)', model: 'Model (empty uses the local account model catalog)',
  recover: 'Clear errors, rate limits and runtime blocks on success', activate: 'Activate inactive accounts on success', resume: 'Enable scheduling on success',
  standard: 'Quality tests use the designated automatic quality question and support OpenAI accounts. Connection success cannot clear quality failures.', noStandard: 'Designate a standard question in the question library before enabling periodic quality tests.',
  pause: 'Pause scheduling after consecutive quality failures', threshold: 'Consecutive failure threshold', save: 'Save policies', saved: 'Automatic detection policies saved', refresh: 'Refresh runtime', running: 'Running', idle: 'Waiting', next: 'Next scan', none: 'No executions yet',
  summary: 'Checked {checked} · Passed {succeeded} · Failed {failed} · Skipped {skipped} · Recovered {recovered}', results: 'Recent executions and recovery results', success: 'Passed', failed: 'Failed', skipped: 'Skipped',
  billingHint: 'Uses the existing rate executor for active API Key accounts supporting billing queries. Rate synchronization respects manual locks and does not restore account health.',
  legacyBilling: 'Currently uses individual account probe switches. Saving here replaces them with the chosen scope.', billingSave: 'Save rate probes', billingSaved: 'Rate probe settings saved', empty: 'Select at least one account.',
  policyTab: 'Automatic policies', scheduled: 'Individual schedules', dirty: 'Unsaved changes; refresh updates runtime only.', billingRuntime: 'Rate runtime', status: 'Status', account: 'Account', finished: 'Finished', notEnabled: 'Disabled'
 }
}

// The application uses the CSP-safe runtime-only i18n build. Supply message
// functions so this module works without a runtime message compiler.
const compiled = (source: Record<string, string>) => Object.fromEntries(Object.entries(source).map(([key, text]) => [key, (ctx: { named: (name: string) => unknown }) => text.replace(/\{(\w+)\}/g, (_, name: string) => String(ctx.named(name)))]))
export default { zh: { automaticDetection: compiled(automaticDetectionMessages.zh) }, en: { automaticDetection: compiled(automaticDetectionMessages.en) } }
