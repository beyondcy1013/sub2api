# AGENTS.md — sub2api 项目约束

## 目标

- 维护定制增强版 Sub2API，在上游升级时保留本地功能、历史和未提交工作。
- main/free 共用唯一源码和同一构建产物，同时严格隔离各自的数据、配置及服务生命周期。
- 保证账号调度、故障轮转和降智检测可追踪，并向客户端提供清楚的状态提示，帮助用户核实可能受影响的操作。
- 本文按功能要求、架构与部署、行为保留契约组织；命令、路径、字段和 API 标识保留原样。

## 降智检测与客户端轮转提示（功能要求）

- 仅当本请求实际测智完成并判定疑似降智，且轮转耗尽准备返回错误时，检查被判降智的同账号、同模型在请求前 5 分钟内是否有成功记录。普通 502、连接拒绝、超时及测智异常不得触发降智复核提醒。如果此前成功、现在失败，应提示可能在这段时间内出现降智，期间操作可能混入降智内容，建议反复核实，避免引入 bug。例如：5 分钟前可用，3 分钟前可能已经降智。
- 客户端使用简洁符号表示账号尝试过程：`---` 表示轮转到一个账号，`x` 表示该账号尝试失败，随后再用 `---` 表示轮转到下一个账号。例如 `--- x --- x ---` 表示已尝试两个失败账号，正在轮转到第三个账号。
- 上述规则是功能要求，不代表已实现、已部署或已验收；运维排查须分别核对成功记录、失败原因、提示内容和客户端实际显示。历史成功与当前失败只能支持“疑似降智”的提醒，不能直接判定降智。
- 运维细节统一维护在[降智检测子技能](.agents/skills/sub2api-ops/subskills/degradation-detection/SKILL.md)，由[项目运维技能](.agents/skills/sub2api-ops/SKILL.md)按需加载。

## 本地功能模块归属

- 新增 fork 专属功能应放入 `backend/internal/modules/<feature>` 和 `frontend/src/modules/<feature>`；已有模块负责该行为时，扩展原模块。实现、DTO、API 客户端和定向测试放在一起。迁移清单见 `docs/FORK_DIFF_ARCHITECTURE.md`。
- 核心路由、Wire provider 和上游页面只保留必要的注册与适配。模块实现不得导入旧的 `internal/service`、`internal/handler`、`internal/repository` 或 `internal/server` 包；需要访问核心能力时注入小接口。
- 保留并复用上游共享服务和组件。旧路径兼容适配器可以转发到模块，但不得持有第二套缓存、单例、工作协程或实现副本。
- Mihomo 和 Basis Points 的实现与测试已放入 `backend/internal/modules/{mihomo,basispoints}`，旧 Go 包仅为兼容入口。Mihomo UI/API/类型及本地日志中心页面位于 `frontend/src/modules/{mihomo,logcenter}`；日志中心复用上游 Ops 传输层与共享组件。
- 本批次定向回归使用 `bash scripts/verify-local-modules.sh`。

## 统一源码归属

- `/home/codes/ai/sub2api` 是 `sub2api.service` 和 `sub2freeApi.service` 唯一的源码与构建目录。
- 嵌入前端的服务端只构建一次，将字节完全相同的产物安装到现有两个可执行文件路径。
- `sub2api.service` 必须设置 `DEPLOYMENT_PROFILE=main`；`sub2freeApi.service` 必须设置 `DEPLOYMENT_PROFILE=free`。
- `/home/codes/ai/sub2freeApi` 继续作为 free 的运行与配置目录及历史恢复来源，不得从中构建第二个二进制。
- 统一源码不允许合并数据库、Redis、配置、用户、工作目录、端口或服务生命周期。

## 服务身份

- 项目路径：`/home/codes/ai/sub2api`
- systemd 单元：`sub2api.service`
- 运行用户/用户组：`sub2api:sub2api`
- 工作目录：`/home/codes/ai/sub2api/deploy`
- 环境文件：`/home/codes/ai/sub2api/deploy/sub2api.env`
- 可执行文件路径：`/home/codes/third_party/bin/sub2api/sub2api`
- 可写数据目录：`/home/codes/ai/sub2api/deploy/data`
- HTTP 端口：`18381`

## 二进制备份策略

- `/home/codes/third_party/bin/sub2api/` 随时最多保留 1 份二进制备份，本项目继续遵循每个运行目录一份备份的规则。
- 放入新备份前，将该目录所有现有的 `sub2api.bak.*`、`sub2api.backup*` 和 `sub2api.bak-*` 文件归档到 `/home/codes/third_party/upgrade-backups/`，不得直接删除。
- 每次部署后，每个运行目录恰好保留最新 1 份备份；报告保留文件名、归档数量和删除数量（必须为零）。
- 长期版本历史放在 Git 标签或发布归档中，不放在 bin 目录。

## Android 客户端

- 当前维护的原生 WebView 客户端位于 `android/`，发布入口为 `scripts/build-sub2api-android-apk.sh`。
- `android/VERSION` 是 Android 发布版本的权威来源。版本相对已发布更新清单保持单调递增，不得从独立版本管理的后端服务推导。
- 冷启动必须并发探测 `SourceRegistry.URLS` 的每个地址，并立即加载首个通过有效健康检查的来源。不得在竞速前串行尝试记忆来源、添加固定延迟或显示自动切源 Toast/倒计时。
- 已选来源健康时保持稳定。网络回调在后台验证当前来源；只有确认失败才可静默并发竞速剩余来源。运行时恢复找不到来源时，保留当前 WebView。
- 主框架失败必须匹配所选来源及当前导航代次；来源解析进行中应合并网络回调。
- 原生客户端 UA 包含 `sub2apiAndroid/<version>`。其账号管理页必须使用与桌面一致的密集表格并支持横向滚动。原生 Android 仅固定 36px 选择列，使操作、名称及后续列都能通过滑动进入视口；桌面保留选择/操作/名称固定列。普通移动浏览器仍使用通用卡片布局。
- 更新端点在 `SourceRegistry.UPDATE_URLS` 中按来源索引配对并并发竞速。端口 `18381` 的 Sub2API 未提供 webClx 产物 API，因此在添加并验证代理前，不得替换已验证的 `11111`/`11112` 更新端点。
- 保留现有 `sub2api` 签名密钥，显式启用 APK 签名方案 v1 和 v2。不得为已有应用 ID 生成替代密钥。
- Android 改动后运行 `node --test android/tests/client-contract.test.mjs`。
- 此前在回退无关虚拟鼠标实验时曾误删整个 Android 客户端。回退功能必须限定到文件或差异块，不得仅为移除一种客户端行为而删除 `android/`。
- 上游合并时保留 [docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md) 中的 Android 契约，后续行为变更也记录在那里。

## 数据隔离

本服务必须使用 main 的账号数据库与配置：

- PostgreSQL 数据库：`sub2api`
- Redis DB：`0`
- 配置文件：`/home/codes/ai/sub2api/deploy/data/config.yaml`

不得指向 `sub2freeApi`、Redis DB `1` 或 `/home/codes/third_party/bin/sub2freeApi/sub2freeApi`。

## 升级流程

- 权威分步手册：[docs/UPGRADE_RUNBOOK.md](docs/UPGRADE_RUNBOOK.md)（`/home/codes/ai/sub2api/docs/UPGRADE_RUNBOOK.md`）。
- 行为保留清单：[docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md)。
- 在 fetch、合并、解决冲突、构建、部署或推送升级前，先阅读这两份文档。
- 项目运维入口：[skills/sub2api-ops/SKILL.md](skills/sub2api-ops/SKILL.md)；源码审计与未解决运维风险：[docs/OPS_REVIEW_2026-09-30.md](docs/OPS_REVIEW_2026-09-30.md)。这些文档区分当前行为与拟议修复。
- 只合并 `upstream/main`。`origin/main` 是定制 fork 备份，仅作为推送目标。
- 此定制部署绝不能使用 WebUI 二进制更新器。
- README 有意排除赞助广告。每次合并后运行 `bash scripts/remove-readme-sponsors.sh`，再用 `bash scripts/remove-readme-sponsors.sh --check` 验证；不得恢复 `README.md`、`README_CN.md` 或 `README_JA.md` 的赞助章节。
- 保留三份 README 顶部附近的本地增强版亮点，升级警告继续指向基于源码的运行手册。不得恢复引导定制 fork 运维人员使用 WebUI 二进制更新器的说明。

## 部署检查清单

- 除非用户明确要求不部署，每次代码改动任务都必须在最终回复前自动完成统一双服务构建、部署和线上验证。
- 部署分为两个脚本：`bash scripts/build-unified-release.sh` 只编译和验证，写出 `backend/bin/sub2api-unified.new` 后结束，不操作服务。安装、双服务重启、备份轮转和线上验证由 `bash scripts/deploy-unified-release.sh` 完成（自动读取当前磁盘 SHA 和两个服务 PID 作为前置条件）。构建脚本退出不代表部署完成。
- 默认通过 webClx 部署 API 排队，使编译与安装保持关联并可审计：

```bash
bash /home/root/.zcode/skills/webclx-compile-and-deploy/scripts/request-webclx-deploy-api.sh \
  --cmd bash --arg scripts/build-unified-release.sh \
  --install-cmd bash --install-arg scripts/deploy-unified-release.sh \
  --audit-path /home/codes/third_party/bin/sub2api/sub2api \
  --audit-path /home/codes/third_party/bin/sub2freeApi/sub2freeApi \
  --source-terminal-name '<fresh unique terminal from /api/terminal/sessions?all=true>' \
  --note 'sub2api 例行编译并部署'
```

  `--source-terminal-name` 使用从 `/api/terminal/sessions?all=true` 新获取的唯一终端名称。webClx API 不可用时，按文档回退为手动先构建、再部署的双脚本流程。
- 日常代码改动部署使用不带模式参数的 `bash scripts/build-unified-release.sh`。默认 `quick` 模式复用 Go 测试缓存，跳过重复的全量 Vitest，仍执行精确源码快照的后端测试集、TypeScript 类型检查、Vite 生产构建、嵌入前端的 Go 构建及源码完整性检查。排队前先运行本任务定向后端和前端测试。冻结构建副本使用固定且受 `flock` 保护的 `/data/cargo-target/sub2api-unified-source` 工作区，让未变更 Go 测试结果可跨部署复用；同步前后仍用源码哈希拒绝混杂或并发变化的源码状态。Go 测试还将 webClx 每次运行的 `TMPDIR` 覆盖为固定且受锁保护的 `/data/cargo-target/sub2api-unified-go-test-tmp`；改变此路径会破坏调用 `t.TempDir` 或 `os.TempDir` 的测试结果缓存复用。
- 仅在上游合并、依赖/工具链变更、迁移、安全/认证变更、大范围跨模块重构或无法确定定向回归覆盖时，使用 `bash scripts/build-unified-release.sh --full`。全量模式强制运行不使用缓存的后端测试及完整 Vitest 测试集，不得再次将其设为自动默认模式。
- `bash scripts/build-unified-release.sh --print-plan` 只审查模式，不构建。未知或冲突的构建模式参数必须拒绝执行。

1. 权威前端及带 `-tags embed` 的 Go 二进制恰好构建一次。
2. 将相同字节原子安装到 `/home/codes/third_party/bin/sub2api/sub2api` 和 `/home/codes/third_party/bin/sub2freeApi/sub2freeApi`，各自保留独立备份。
3. main 保持数据库 `sub2api`、Redis DB `0`、端口 `18381`、profile `main`；free 保持数据库 `sub2freeApi`、Redis DB `1`、调度器前缀 `sub2freeApi`、端口 `18382`、profile `free`。
4. 先重启并验证 `sub2api.service`，再处理 `sub2freeApi.service`。不得用一条命令同时重启两者。
5. 确认两个磁盘文件和两个 `/proc/<pid>/exe` 文件的 SHA-256 全部一致。
6. 执行验证：

```bash
systemctl status sub2api.service --no-pager
ss -ltnp | rg ':18381'
curl -fsS http://127.0.0.1:18381/ >/dev/null
systemctl status sub2freeApi.service --no-pager
ss -ltnp | rg ':18382'
curl -fsS http://127.0.0.1:18382/ >/dev/null
```

## 必须保留的本地定制

- 三份 README 均不得含赞助广告，同时继续合并上游功能文档、许可证和项目致谢。
- 管理员账号凭据有意以明文显示 `credentials.api_key`，用于账号编辑和检查。
- 不得将 `api_key` 加回 `SensitiveCredentialKeys`；OAuth token、会话密钥、cookie、AWS 密钥、服务账号 JSON 和私钥仍必须脱敏。
- 旧版或部分前端更新未提交 `api_key` 时，`MergePreservingSensitiveCreds` 必须保留已有值。
- `frontend/src/components/account/EditAccountModal.vue` 应预加载 `credentials.api_key`，账号 API key 输入框使用 `type="text"`。
- 使用自定义 `base_url` 的 OpenAI API Key 账号，遇到上游 401 的 `token_invalidated` 错误码，或中转未带该码但返回标准消息 `authentication token has been invalidated` 时，必须视为中转内部 OAuth 故障：请求切换账号，不改变本地 `status` 或 `schedulable`。此例外不适用于 OAuth 账号、`invalid_api_key`、默认 OpenAI 端点或管理员显式配置的自定义 401 错误码策略。
- **账号管理表格列布局**（`frontend/src/views/admin/AccountsView.vue` 的 `allColumns`）：
  - 开头顺序为 `选择` -> `操作` -> `名称` -> `容量` -> `状态` -> `调度` -> `疑似降智` -> `质量状态` -> `降智次数` -> `用量窗口` -> `平台/类型`。
  - `降智次数`（`degradation_count`）列显式设置 `width: '130px'` 作为最小宽度，表头提供动态时间范围下拉选择（`24h`、`7d` 默认、`30d`、`all`），点击下拉阻止冒泡不触发排序，切换即时查询该时间窗口内各账号降智次数并持久化到 `localStorage`；单元格零值显示 `0`，非零值高亮展示并提供悬浮时间范围与次数说明，支持点击表头在当前页按降智次数升降序排序。
  - `创建时间` 后保持 `今日费用` -> `累计费用` -> `分组`（可见时）-> `余额` -> `5h请求` -> `5h Token` -> `7d请求` -> `7d Token` -> `窗口总费用`。
  - `今日统计` 后保持 `7d(%)` -> `7d`；末尾顺序为 `过期时间` -> `备注` -> `账号ID` -> `上游声明费率` -> `调度倍率` -> `5h(%)` -> `5h`。
  - 利用率表头使用紧凑标签 `5h(%)` 和 `7d(%)`。
  - `名称`（name）列显式设置 `width: '212px'`（比原来的 176px 宽 20%），内部内容上限为 `212px × 32px`。账号名与补充邮箱共用从左到右的文本流，名称在前，长邮箱不得挤掉名称前缀。合并文本自然换行，最多两行，每行 16px，仅在第二行末尾截断，不增加行高或溢出单元格。
  - 去除两端空白并忽略大小写后，若账号名已以相同邮箱结尾，不再追加补充邮箱。
  - 选择列显式设置 `width: '36px'`，`DataTable.vue` 的 `--select-col-width` 保持 `36px`。
  - 账号表的 `状态`（status）列显式设置 `width: '80px'` 作为最小宽度。
  - 表头、标签、排序指示器和桌面单元格内容保持单行且不收缩。
  - `AccountsView.vue` 启用 `DataTable` 的 `single-line-cells` 和 `dynamic-column-widths` 模式。在显式启用的此模式下，`column.width` 为最小宽度，其他内容可撑宽列；名称单元格仍以 `212px` 为上限，必要时表格横向滚动。
  - 桌面横向滚动时，选择、操作、名称列固定在左侧，以声明的 `36px`、`220px`、`212px` 累计偏移。原生 Android 仅固定 `36px` 选择列；操作、名称和后续所有列都必须能滚入视口，不得被比视口更宽的固定区域遮挡。
  - 其他 `DataTable` 使用者保留默认固定宽度行为，将声明宽度同时应用到 `width`、`minWidth` 和 `maxWidth`。
  - 首尾单元格使用 `4px` 外侧内边距，避免多余边缘留白。
  - 非末列在浅色和深色模式下均保留 1px 竖向分隔线。
  - `平台/类型`（platform_type）列显式设置 `width: '170px'` 作为最小宽度。
  - `账号ID`（id）列显式设置 `width: '130px'` 作为最小宽度。
  - `frontend/src/components/common/types.ts` 的 `Column` 接口包含可选属性 `width?: string`。
  - `frontend/src/components/common/DataTable.vue` 的 `DataTable` 默认将 `column.width` 应用为 `width` + `minWidth` + `maxWidth`；启用 `dynamicColumnWidths` 时仅应用为 `minWidth`。
  - `AccountsView.vue` 启用 `DataTable` 的 `compact-rows` 模式。桌面加载态与数据单元格使用 `py-0.5`（上下各 2px 内边距），其他表格保持默认间距。
  - 桌面直接账号操作保持单行 `h-6` 文字按钮，标签可见，内边距为 `px-2`；操作列最小宽度保持 220px。小于 1024px 时，操作列固定 72px，每行提供一个带标签的入口，在 `MobileActionSheet` 中打开 `AccountActionMenu`。所有已有操作和确认均可到达。Android 仍只固定 36px 选择列。
  - 不得将这些列恢复到上游位置或移除宽度属性。
- 升级时保留每个本地提交和未提交文件。遵循 [docs/UPGRADE_RUNBOOK.md](docs/UPGRADE_RUNBOOK.md)，审查 [docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md)；此定制构建不得使用 WebUI 二进制更新器。
- **必须保留 OpenAI 粘性会话并发溢出切换**：
  - 仅当绑定账号能获取真实 `account.Concurrency` 槽位时，才沿用历史 `session_hash` 绑定。
  - 账号满载时，即使历史会话仍绑定该账号，也应通过正常同组选择流程，将当前连接路由到其他可用账号。
  - 单连接溢出不得改写历史粘性绑定；所有合格账号都满载时，保留正常的有界等待计划兜底。
  - 即使关闭 TTFT/错误健康逃逸，也必须执行并发满载溢出切换。严格且不可迁移的 `previous_response_id` 亲和性保持不变。
  - 保留 `openai_gateway_scheduling.go`、`openai_account_scheduler.go` 的实现及对应 `*_test.go` 中的溢出回归测试。
- **必须保留最低成本调度与费率同步**：
  - 全局调度仅有 `default` 和 `lowest_cost`；旧 `extra.super_priority` 及旧模式值不得影响路由或状态显示。
  - `accounts.rate_multiplier` 是最低成本排序的唯一依据。自动探测成功时只将 `resolved_rate_multiplier` 写入该字段；峰值/有效值快照不参与请求排序。
  - `extra.scheduling_rate_sync_mode` 为 `auto_overwrite`（默认）或 `manual_lock`。仅在新字段不存在时，才将旧 upstream/manual 来源值映射为这些模式。
  - `lowest_cost` 下，有并发上限的恢复探测不得测试健康 active 账号或手动暂停账号，只重试 `status=error` 的 API Key 账号。真实上游请求成功后，清理可恢复错误、限流及运行时状态，并恢复 `schedulable=true`。OAuth 和其他账号类型仍排除。
  - 为配置兼容继续接受旧 `super_priority.liveness_include_unschedulable`，但不得因此扩大恢复探测到健康或手动暂停账号。
  - 周期性上游账单/费率探测默认关闭，避免向健康账号发送后台上游请求。需要自动同步费率时，管理员可在调度规则弹窗中显式开启。
  - 调度规则弹窗右上角显示服务端持有的存活探测运行状态：运行情况、预计下批时间，以及最近检查/成功/失败/跳过结果。数据来自后端执行器，不得只使用浏览器计时计划。
  - 账号表仅为可调度且在至少一个调度组中并列最低 `rate_multiplier` 的账号，以金色标记当前最优调度费率。并列最低全部标记；未分组账号仅与同平台未分组账号比较。计算使用完整 active 账号池，不限当前页；这只是管理提示，不保证每个特定模型请求都选中该账号。
- 上游升级后执行验证：

```bash
cd /home/codes/ai/sub2api/backend && go test -tags unit ./internal/handler/dto ./internal/service
cd /home/codes/ai/sub2api/frontend && pnpm vitest run src/components/account/__tests__/EditAccountModal.spec.ts
```

## 账号用量窗口表格契约

- `UsageProgressBar.vue` 保持紧凑，每行仅显示用量窗口标签、进度/利用率和重置状态。不得把请求数、token、`A` 或 `U` 总量放回进度条内部。
- `AccountsView.vue` 分别提供 `5h请求`、`5h Token`、`7d请求`、`7d Token`、`5h(%)`、`5h重置`、`7d(%)`、`7d重置` 和 `窗口总费用` 列。费用列分 5h/7d 两行，在有数据时显示上游（`A`）和用户（`U`）值。
- 窗口包含有效 `window_stats` 时，请求数/token/费用零值显示为 `0`；只有窗口或统计确实缺失时才显示 `-`。新账号或轻度使用账号加载用量后，必须与老账号具有同样完整的字段结构。
- 利用率和重置列保留本地排序；无论升降序，未加载用量的账号均排在有数据账号之后。
- OpenAI 空闲窗口 `0%` 显示 `现在`；利用率大于零且已过期的窗口显示 `待刷新`；未来重置时间显示倒计时。
- `AccountUsageCell.vue` 发出 `usage-loaded`，其载荷是父表请求数/token/利用率/重置/费用列的唯一数据来源。自动加载仅针对 active 账号；显式单行或批量刷新可查询 inactive/error 账号。
- `AccountBulkActionsBar.vue` 中，`批量更新额度` 紧挨在 `批量更新` 前面：
  - 有选中账号时只查询所选账号；未选择时只查询当前页已加载账号，不得扩展到后续筛选分页。
  - 只查询提供单行主动 `查询` 操作的账号类型（OpenAI OAuth 和 Anthropic OAuth/Setup Token）。
  - 以 `source=active&force=true` 调用 `/usage`，最大并发为 4；单账号失败后继续，报告成功/失败数量。
  - 每个成功响应立即更新用量单元格和所有独立窗口列；执行期间禁用两个主要批量按钮。
- 定向回归验证：

```bash
cd /home/codes/ai/sub2api/frontend
pnpm vitest run \
  src/components/account/__tests__/UsageProgressBar.spec.ts \
  src/components/account/__tests__/AccountUsageCell.spec.ts \
  src/components/admin/account/__tests__/AccountBulkActionsBar.usageRefresh.spec.ts \
  src/utils/__tests__/usageWindowDisplay.spec.ts \
  src/utils/__tests__/batchAccountUsageRefresh.spec.ts \
  src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts \
  src/views/admin/__tests__/AccountsView.bulkUsageRefresh.spec.ts
```

## 增强导入混合消息契约

- 粘贴文本增强导入支持单个 JSON 值、JSON 数组，以及包含多个完整 JSON 值的聊天/转发/Markdown 混合文本。
- 提取必须使用能识别字符串的对象/数组括号平衡扫描器。嵌套值、字符串内括号、转义引号和转义反斜杠不得拆断 JSON 值，不得替换为贪婪正则。
- 每个提取值独立规范化与校验。片段错误使用从 1 开始的来源标签 `pasted JSON #N`，所有有效片段按来源顺序合并为一次导入 API 请求。
- 拒绝没有完整 JSON 值的文本和外层截断的 JSON；不得从不完整外层值中导入内部数组/对象。
- 保持纯 JSON 与多文件模式兼容，保留导入路由默认值（最后一个代理、第一个分组）、操作者覆盖及 `skip_default_group_bind: true`。
- 文本模式 UI 保留双语使用说明和提取摘要。除导入请求外，不得记录日志或持久化粘贴的凭据/token。
- 账号工具菜单在 `增强导入` 正下方保留 `批量清除导入账号`。该入口以独立清除模式打开增强解析器，不显示导入路由控件；普通增强导入弹窗不显示破坏性操作。清除模式复用文件/文本解析器，确认前仅将规范化账号数据提交给只读数据库匹配预览，不删除导入的代理。确认时同时报告解析数与匹配数。默认匹配普通和归档账号，将匹配项移入可恢复回收站。默认不勾选的 `彻底删除` 复选框将匹配范围扩大到回收站，并永久删除所有匹配项，不论其当前生命周期层级。不可逆操作确认必须说明账号无法恢复。后端按平台/类型与稳定凭据身份匹配，仅在唯一时允许按名称兜底，永久清除使用受保护的彻底删除服务；不得猜测缺失或歧义匹配。
- 定向回归验证：

```bash
cd /home/codes/ai/sub2api/frontend
pnpm vitest run \
  src/components/admin/account/__tests__/enhancedImport.spec.ts \
  src/components/admin/account/__tests__/EnhancedImportDataModal.spec.ts
```

## 账号归档与可恢复回收站

- **批量账号编辑直接可用并跟踪变更**：
  - `BulkEditAccountModal.vue` 中的值控件在字段复选框未勾选前仍可使用。
  - 修改值时仅自动勾选对应字段用于提交；未修改字段的默认值不得进入批量更新载荷。
  - 保留字段复选框，以便显式执行目标值恰好等于表单默认值的操作，包括清空代理、分组、映射或其他现有值。
- **账号归档**（原称 recycle）通过 `extra.recycled = true` 作为额外筛选条件，不使用软删除（`deleted_at`）。`AccountTableActions.vue` 的筛选切换按钮标为“归档”，使用 `archive` 图标。
- `account_repo.go` 的 `accountListFilteredQuery` 分别接受 `recycled` 和 `deleted` 模式，两者互斥；普通列表排除这两种生命周期状态。
- 所有 `ListAccounts` 调用方都必须传入两个生命周期布尔值。`ListWithFilters` 和 `ListAllWithFilters` 继续接受 `recycled`，始终排除回收站记录。
- 后端归档路由：`POST /api/v1/admin/accounts/:id/recycle` 和 `POST /api/v1/admin/accounts/:id/restore`。
- 前端：`AccountTableActions.vue` 提供归档切换按钮（归档图标）；`AccountsView.vue` 按模式在操作列显示归档/取消归档按钮。
- active 账号行直接操作顺序保持 `编辑` -> `测试连接` -> `归档` -> `更多`；`AccountActionMenu.vue` 内不重复显示 `测试连接`。
- **可恢复回收站**是第二层生命周期，通过 `extra.deleted = true` 标记；新的删除操作不得设置 `deleted_at`：
  - `DELETE /api/v1/admin/accounts/:id` 将记录移入回收站，移除 `extra.recycled`，保留凭据、分组、用量/历史和定时配置。
  - 普通和归档列表排除回收站记录。`GET /api/v1/admin/accounts?deleted=1` 仅列出回收站记录；同时使用 `deleted=1` 和 `recycled=1` 必须拒绝。
  - `AccountTableActions.vue` 的垃圾桶图标切换回收站表格。行操作保留 `编辑`、`测试连接`、`恢复` 和 `更多`；恢复使用 `POST /api/v1/admin/accounts/:id/restore-from-trash`。
  - 批量选择控件在 `本页全选` / `全选所有结果` / `清除选择` 旁保留紧凑筛选输入框，绑定服务端账号搜索，在普通、归档、回收站视图均可用。搜索覆盖账号 ID、名称、备注、平台、类型、状态和错误消息；凭据及 token 不得放进 GET 搜索查询。
  - 回收站账号绝不能成为请求路由候选，必须排除于 OAuth 刷新、定时测试、存活探测、上游费率/账单探测、自动用量/余额刷新、到期自动暂停、调度评分池和模型可用性扫描。显式管理操作仍可用。
  - 普通编辑必须保留服务端拥有的 `deleted` 标记。复制回收站/归档账号时，不得将任一生命周期标记复制给新账号。
  - 迁移 `191_convert_account_soft_delete_to_deleted_staging.sql` 恢复旧软删除记录及保存的分组绑定，清空 `deleted_at`，标记 `extra.deleted=true`。
  - 彻底删除仓储、服务和 API 保持保护，仅接受回收站或旧软删除记录。回收站账号管理视图在 `AccountActionMenu.vue` 中通过 `permanent-delete` 事件提供需要显式确认的单行 `彻底删除`，也为选中记录（含全结果选择）提供批量操作；普通和第一层归档视图不得显示。
- 管理员 `components/admin/account/AccountTestModal.vue` 打开时加载本地模型目录，仅在操作者点击开始后才启动连接测试。保留可停止的 AbortController 流程和允许自建选项的模型选择器，不得恢复已移除的自动开始偏好。见 `docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md` 对应章节。
- 直接连接测试成功后，若账号状态不是 `active` 或调度已暂停，恢复前提示操作者确认。确认后完整恢复运行时状态，必要时激活账号并启用调度；已 active 且可调度的账号不提示。
- 直接连接测试明确失败后，若账号尚非 `error`，提示是否标记失败。确认后使用专用 `mark-failed` API 设置 `status=error`、保存测试错误消息并禁用调度；已 error 账号仅刷新。自动失败窗口观察保持原样，不得在配置阈值前写入 `status=error`。
- 账号名称保持在固定宽高名称单元格内，两行截断并裁剪溢出。不得恢复通过传送渲染将内容放到单元格外的名称悬浮提示。
- sub2freeApi 的 `accounts.ts` 比 sub2api 多一个 `clone` 函数，同步文件时需补回。
- 状态/分组/容量单元格使用纯文本（仅文字颜色类），不得使用徽章/卡片样式。见 `references/account-table-column-layout.md`。
- **用量自动加载跳过非 active 账号**：`AccountUsageCell.vue` 的 `shouldAutoLoadUsageOnMount` 必须以 `props.account.status === 'active'` 为条件。`inactive` 或 `error` 账号页面挂载时不得自动请求 `/usage`，避免向已知不可用账号发送无效上游请求。通过 `usageManualRefreshToken` 的手动刷新对所有状态均不受影响。
- **OpenAI 额度自动暂停显示为额度限流**：管理员账号响应暴露与调度相同服务逻辑在请求时计算的 `quota_rate_limit`。`AccountStatusIndicator.vue` 显示 `额度限流` 及重置时间；账号被排除时，已启用的调度开关显示琥珀色受限状态。不得将此派生状态持久化到 `status`、`schedulable` 或 `rate_limit_reset_at`；窗口恢复必须自动进行。

## 账号操作与活跃粘性会话迁移

- 新账号表单初始化和重置使用 `concurrency: 10`。增强 CLIProxyAPI 导入也默认新账号为 `concurrency: 10`；原生备份导入保留各账号导出的并发值。
- 新账号表单初始化和重置时选择最后一个可用代理及第一个可用分组。若代理/分组属性在弹窗打开后才到达，仅填补仍为空的选择，不覆盖操作者已有选择。候选列表为空时保持未分配。
- 进入 OAuth 授权步骤后，`CreateAccountModal.vue` 在第二步顶部继续显示第一步输入的账号名。操作者完成 OpenAI Auth 期间，作为账号名的邮箱地址也必须保持可见。
- `AccountActionMenu.vue` 对每个账号始终显示 `恢复状态`。
- `AccountActionMenu.vue` 底部低频操作顺序保持 `查看统计` -> `创建 Spark 影子账号`（可用时）-> `设置隐私`（可用时）。不得移回恢复、调度、额度重置或删除操作上方。
- 操作菜单保留 `w-[7.8rem] max-h-[calc(100vh-1rem)] overflow-y-auto`；`AccountsView.vue` 使用 `125` px 预估宽度和 `320` px 预估高度进行视口定位。
- `迁入粘性会话` 对 active 且可调度的 OpenAI 目标账号显示，默认选择最近 5 分钟活跃绑定。允许窗口为 1、5、15、60 分钟。
- 公共设置加载期间，前端能力标记按显式关闭处理：明确的 `sticky_session_reassignment_enabled: false` 仍会在 free profile 隐藏入口，但字段暂缺不得使 main profile 的入口消失。
- 弹窗显示近期/全部数量及匿名近期后缀。最多迁移 100 个，按最新优先，后端重新校验活跃窗口。
- 活跃时长由配置的粘性 TTL 减去 Redis PTTL 推导。迁移使用比较并设置及 `SET ... KEEPTTL`。
- 仅迁移当前 16 位小写十六进制 `session_hash` 键。忽略旧 64 位副本，不得迁移 `response:` / `previous_response_id` 续接绑定。
- `Concurrency limit exceeded for user` 是本地 sub2api 调用方并发等待超时，并非上游消息。粘性集中可能在请求等待同一上游账号时占满调用方槽位，因此将近期绑定迁到空闲容量可能有帮助。不得把此操作指向 `用户管理`。
- 保留 `.codex/skills/sub2api-account-modal-enhancer/references/sticky-session-reassignment.md` 记录的 API 路由和测试。

## 定时账号状态操作

- `账号管理 -> 更多` 始终显示 `定时启用并恢复` 和 `定时暂停调度`。
- 弹窗接受整数小时加 `0..59` 分钟，合计延迟最少 1 分钟、最多 365 天，保存前始终显示计算出的本地执行时间。
- 任务持久化到 `scheduled_account_actions`，不得依赖浏览器持续打开、内存计时器、`temp_unschedulable_until` 或定时账号测试表。
- 每个账号最多一个当前定时操作。再次保存替换原任务；待执行任务可在同一弹窗取消。
- `enable_and_recover` 先以 `InvalidateToken: true` 调用 `RateLimitService.RecoverAccountState`，再调用 `AdminService.SetAccountSchedulable(true)`，保留完整错误/限流/过载/临时/模型/运行时阻断清理。
- `pause` 调用 `AdminService.SetAccountSchedulable(false)`，不修改账号持久化状态。
- 执行器通过数据库租约领取到期记录。成功任务标记完成；失败任务保留 `last_error` 并回到待执行状态，一分钟后重试。服务中断后可重新领取过期处理租约。
- 保留迁移 `185_add_scheduled_account_actions.sql`、账号级 `GET|PUT|DELETE /api/v1/admin/accounts/:id/scheduled-action` 路由、执行器启动/清理接线及对应前后端测试。

## 定时测试自动恢复调度

- `scheduled_test_plans.auto_recover_schedulable`（迁移 `186`）是每个计划独立的开关，在定时测试成功证明账号健康后重新启用调度（`schedulable=true`）。
- 此开关独立于清理错误/限流状态的 `auto_recover`。`auto_recover_schedulable` 面向通过 `schedulable=false` 手动暂停的账号，执行器在测试通过后重新启用。
- `ScheduledTestRunnerService.tryReEnableScheduling` 检查 `account.Schedulable`，仅在当前暂停时调用 `SetSchedulable(true)`。
- 前端 `ScheduledTestsPanel.vue` 在新增与编辑计划表单中均提供开关，标签为 `自动启用调度`。
- 新增与编辑表单保留原始 5 字段 Cron 输入及帮助，同时提供鼠标操作的可视化调度器，支持分钟间隔、每小时、每日和每周计划。高级表达式仍可在自定义模式编辑，不得自动改写。
- 保留迁移 `186_add_scheduled_test_auto_recover_schedulable.sql`、`ScheduledTestPlan` 结构体的 `AutoRecoverSchedulable` 字段、处理器创建/更新请求结构及前端开关。

## 统一自动检测策略（2026-10-18）

- `账号测试 -> 自动 -> 自动策略` 的状态恢复/周期测智支持动态 `all` 和显式 `selected` 账号范围；平台、目标状态与范围共同生效。暂停调度必须显式纳入，归档/回收站/软删除/过期账号排除。
- 新策略不依赖 lowest_cost；旧 liveness 的既有资格限制不变。自动执行时旧 liveness/独立 Cron 计划对新策略覆盖账号让位，避免重复探针。显式手动检测保留。
- 连接通过不恢复质量异常；周期测智只认可当前指定标准试题，不允许无标准启用。成功清理、激活、启用调度分别选择；连续测智失败可按阈值暂停。状态/测智默认关闭，部署不得自行开启。
- 策略与执行进度持久化到本服务 settings；进程重启继续遵守逐账号间隔，数据库锁及运行数据隔离 main/free。恢复前重新校验策略和账号，最终比较更新必须与 scheduler_outbox 原子写入并同步共享快照。
- 费率探测同入口支持 all/selected，缺省 scope 兼容旧逐账号开关；保留手动锁定与既有执行器。旧调度设置保存时不得擦除新增范围字段。实现和回归见 `docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md` 的同名章节。

## 检测来源与使用记录

- 自动连接检测、定时测智、业务请求触发测智须区分来源；检测结果保存实际 main/free、来源账号、Cron 计划（适用时）及自动选模型信息，不用当前配置伪造旧记录来源。
- 使用日志的检测历史入口与账号测试、日志中心复用服务端历史。检测记录不能冒充业务成功或进入近期成功提醒证据；未采集的 Token/费用不能当成零。
- API Key 跨实例检测携带 User-Agent 来源标识，接收端在模型下显示为客户端声明。来源标识不改变鉴权、计费、调度或测智行为；OAuth 出站身份保持原有约束。
