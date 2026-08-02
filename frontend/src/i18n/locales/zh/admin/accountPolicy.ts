export default {
  accountPolicy: {
    title: '账号策略设置',
    description: '集中管理账号状态、额度、调度倍率与本地容错策略',
    searchPlaceholder: '搜索账号名称或备注',
    accountCount: '共 {count} 个账号',
    empty: '没有符合条件的账号',
    selectAccount: '选择左侧账号后管理策略',
    loadFailed: '账号策略加载失败',
    save: '保存策略',
    saved: '账号策略已保存',
    saveFailed: '账号策略保存失败',
    state: {
      title: '账号状态',
      status: '当前状态',
      scheduling: '参与调度',
      enabled: '已启用',
      paused: '已暂停',
      recover: '恢复状态',
      recovered: '账号状态已恢复',
      recoverFailed: '恢复账号状态失败',
      schedulingUpdated: '调度状态已更新',
      schedulingFailed: '更新调度状态失败',
      rateLimitUntil: '限流至 {time}',
      overloadUntil: '过载冷却至 {time}',
      temporaryUntil: '临时暂停至 {time}',
      quotaLimited: '{window} 额度限流：{utilization}%，重置时间 {time}',
      noRuntimeBlock: '当前没有运行时阻断',
      errorMessage: '最近错误'
    },
    relay: {
      title: '中继失败预算',
      unavailable: '当前账号不支持此策略',
      enabled: '启用失败预算',
      windowMinutes: '统计窗口（分钟）',
      thresholdPercent: '失败比例阈值（%）',
      minRequests: '最少请求数',
      consecutiveFailures: '连续失败阈值',
      cooldownMinutes: '冷却时间（分钟）'
    },
    quota: {
      title: '账号额度限制',
      unavailable: '仅 API Key、Bedrock 且非影子账号支持额度限制',
      total: '总额度',
      daily: '每日额度',
      weekly: '每周额度',
      used: '已用 {value}',
      unlimited: '0 表示不限额'
    },
    rate: {
      title: '调度与计费倍率',
      multiplier: '账号倍率',
      autoOverwrite: '自动同步',
      autoOverwriteHint: '成功探测上游声明倍率后自动覆盖',
      manualLock: '手动锁定',
      manualLockHint: '保留当前手动倍率，不被自动探测覆盖'
    },
    validation: {
      relay: '失败预算参数超出允许范围',
      quota: '额度必须是大于或等于 0 的数字',
      rate: '账号倍率必须是大于或等于 0 的数字'
    }
  }
}
