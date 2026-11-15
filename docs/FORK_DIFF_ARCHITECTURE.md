# Fork 与上游差异及新增模块边界

本文以 `upstream/main` 为 GitHub 原项目基线，记录当前定制分支的差异范围，并规定后续新增功能的落位方式。它是架构和升级审查文档，不替代 `LOCAL_UPGRADE_CUSTOMIZATIONS.md` 中的行为契约。

## 当前基线

- 上游：`v0.2.16-33-g52b3ca960`
- 本地 `HEAD`：`v0.2.16-337-gd27c5a23e`
- 相对上游的已提交差异：232 个新增文件、325 个修改文件、1 个删除文件。
- 工作区还有未提交文件；它们不属于上面的 Git 差异统计，升级或发布前必须单独审查并提交/归档。
- `upstream` 是原项目，`origin` 是本定制 fork；升级只合并 `upstream/main`，不从 `/home/codes/ai/sub2freeApi` 重新构建。

## 差异分层

### 1. 应继续与核心合并的基础改造

这些改造影响所有请求或部署边界，不能简单复制成旁路插件：

- `config`、deployment profile、主/免费双服务隔离、统一构建和部署。
- OpenAI 调度、最低成本排序、liveness、粘性会话并发溢出和 relay failure budget。
- 账号生命周期：归档、回收站、恢复、定时启停、使用量窗口和额度状态。
- 网关协议兼容、错误来源标记、reasoning replay、Anthropic/OpenAI/Gemini 转换。
- plugin API v2、数据库迁移、Redis key namespace 和调度缓存。
- Android 原生客户端与账号表格契约。

这些文件可以继续修改现有核心实现，但每次上游升级都必须按行为清单和 focused tests 验证。

### 2. 应独立成模块的本地业务

当前工作区已经出现大量新业务文件，后续不要继续把它们堆到 `internal/service` 根目录或大型 handler 中。推荐的目标边界如下：

| 模块 | 后端目标包 | 前端目标目录 | 主要职责 |
| --- | --- | --- | --- |
| Codex 账号采集/票据 | `backend/internal/modules/codexharvest` | `frontend/src/modules/codexharvest` | auth 文件、可用性、采集流程、ticket 生命周期、反馈和调度 |
| 自动检测策略 | `backend/internal/modules/autodetection` | `frontend/src/modules/qualitycontrol` | 所有/指定账号、目标状态、周期连接/标准测智、恢复策略、持久执行进度；核心层仅接线和适配 |
| 请求质量与账号检测 | `backend/internal/modules/requestquality`、`accountdetection` | `frontend/src/modules/request-quality`、`account-detection` | 探测、规则、WebSocket、结果汇总、检测计划 |
| 降级规则 | `backend/internal/modules/degradation` | `frontend/src/modules/degradation` | 规则 CRUD、运行时匹配、探测失败后的降级决策 |
| Excel/BPS 计费 | `backend/internal/modules/basispoints` | `frontend/src/modules/basispoints` | 图片/表格解析、费率计算、结构化结果和 admission |
| OpenAI 代理运行时 | `backend/internal/modules/openai_runtime` | 无独立页面，保留 API 适配器 | fallback chain、quarantine、SSE read pump、turn admission |
| Mihomo 设置 | `backend/internal/modules/mihomo` | `frontend/src/modules/mihomo` | 国家筛选、代理配置和设置页 |
| 运维日志中心 | `backend/internal/modules/logcenter` | `frontend/src/modules/logcenter` | 日志查询、模型过滤、日志页和诊断接口 |

现有文件名已经反映这些边界的，可在下一次小步重构中迁移；仅测试文件不应被误认为独立运行模块。

### 3. 工具和运维，不进入业务核心

`scripts/codex-account-bridge.py`、安装脚本、检测验证脚本、webClx 部署脚本、Android 打包脚本、技能文档和 `docs/` 只通过明确的 API/命令调用业务模块，不应被业务包反向依赖。运行时配置仍分别留在 `deploy/sub2api.env`、`deploy/data` 和免费 profile 中。

## 新增代码规则

1. **新功能先建目录。** 后端放在 `backend/internal/modules/<name>`，按 `handler`、`service`、`repository`、`types`、`transport` 分层；前端放在 `frontend/src/modules/<name>`，内部再分 `api`、`components`、`views`、`utils`。
2. **核心层只做接线。** `internal/server/routes`、现有大型 handler、`wire_gen.go` 和 `AccountsView.vue` 只保留路由、依赖注入、兼容适配和少量状态拼装。新业务逻辑不得直接追加到这些文件。
3. **模块依赖单向。** `transport/handler -> service -> repository`；模块可以依赖公共 domain/interface，但模块之间不能互相引用实现包。跨模块调用使用小接口，接口放在调用方或 `internal/ports`。
4. **API 契约独立。** 新模块的 DTO、错误码、权限和路由集中在模块目录；核心 handler 只转发。前端 API client 与页面组件同模块维护，避免继续扩张 `frontend/src/api/admin/index.ts`。
5. **迁移按功能命名。** 保留已经发布的迁移文件名；新迁移使用下一个编号加功能名，例如 `246_add_account_detection_records.sql`，不要只用模糊编号，也不要重命名历史迁移。
6. **测试跟随模块。** 单元测试与实现同目录；跨层契约测试放模块的 `contract_test.go`；新增 API 至少覆盖权限、profile 隔离和错误路径。
7. **注册点显式且最小。** 每个模块提供一个构造函数或 `Module` 接口，统一在 `wire.go`/路由注册处接入；禁止通过 init() 隐式注册。
8. **上游升级可回放。** 与上游同名文件的改动尽量限制为适配器和调用点；新模块文件优先新增而不是修改上游文件。冲突解决时先合并上游层，再恢复本地模块层。

## 推荐迁移顺序

1. 先冻结当前工作区，提交或归档未跟踪文件，记录每个功能的 API、迁移和测试入口。
2. 先迁移无核心状态的 `mihomo`、`logcenter`、`basispoints`，用适配器保持原路由不变。
3. 再迁移 `degradation`、`requestquality`、`accountdetection`，明确它们与账号状态/调度器的接口。
4. 最后迁移耦合最深的 `codexharvest` 和 `openai_runtime`；迁移期间保留旧包的薄转发文件，完成 focused tests 后再删除旧实现。
5. 每一批迁移后运行对应 Go/Vitest 测试，再执行统一构建；不要在一次上游升级中同时进行大规模目录重排。

## 审查清单

- 新文件是否位于已有模块或新的 `internal/modules/<name>` 下？
- 是否只修改了必要的路由、wire 和公共类型文件？
- 是否新增了独立 DTO、权限校验、错误码和测试？
- 是否会把 `main` profile 的数据库、Redis 或调度 key 泄漏到 `free` profile？
- 是否会被上游升级覆盖，且能通过薄适配器恢复？
- 是否在 `LOCAL_UPGRADE_CUSTOMIZATIONS.md` 中补充了用户可见行为契约？

## 已实施：2026-10-17 首批模块迁移

- `backend/internal/modules/mihomo` 接管原 `internal/mihomo` 的全部实现与测试；原路径只保留类型别名与转发函数，共享唯一运行状态。
- `backend/internal/modules/basispoints` 接管原 `internal/service/basispoints` 的全部协议、图片中继、结构化输出实现与测试及来源声明；原路径只保留兼容入口。核心调用方直接引用新模块。
- `frontend/src/modules/mihomo` 集中组件、类型、独立 API client 与测试；原 settings 路径保留组件/类型兼容导出。
- `frontend/src/modules/logcenter` 集中本地日志页面、API 适配器与测试；路由继续通过原页面薄入口接入。上游 Ops API、权限与共享组件保留其原有所有权，不复制一套后端日志服务。
- 仓库 `AGENTS.md` 已增加新功能模块归属规则；模块 README 记录边界；`scripts/verify-local-modules.sh` 提供聚焦验证入口，日志中心原验证脚本已跟随新测试路径。
- 此批是目录和依赖归属调整，不修改配置、端口、数据、HTTP 路径、调度、票据、图片中继与 UI 行为；不会自动接通目前未注册的 Mihomo 管理接口。
- 迁移前相关源码、index/worktree diff 与状态归档在 `/data/cargo-target/sub2api-module-backups/20261017-033919`，未暂存或提交其他已有改动。
- `degradation`、`requestquality`、`accountdetection`、`codexharvest` 和 `openai_runtime` 仍按上述顺序逐批迁移，不能宣称已完成全部解耦。

### 首次完整验证与旧测试修正

请求 `034134-18debef697f8f711` 的完整后端套件通过，前端为 3277 通过、1 跳过、1 失败。唯一失败来自 `AccountsView.batchTestAndMark.spec.ts`，仍模拟已移除的 `test-and-mark` 事件并期待自动测试/标记。按现行“手动测试 UI 整合”合同，将此测试更新为选择账号后经 `pelican-test` 打开统一工作台，验证账号明细、默认模式、不自动测试/标记，以及无选择时不启动。保留负向断言，未恢复已移除入口，也未跳过测试。

失败发生在安装前，未产生 `install-report-1.json`；main/free PID 与磁盘 SHA 均保持首次请求前状态。

### 首批部署验收通过

- 成功请求：`034937-18debef697f8f712`，2026-10-17 03:56:22 完成。完整后端套件、前端 426 个文件 / 3279 项测试通过（另有 1 项原有跳过）、TypeScript、Vite 及嵌入式 Go 构建通过。
- 构建源码快照：`afb7299e07c4ce78e9f920bf2530c03229b057933d2909a4b79447a1bf9c64b9`。日志：`/home/bin/webclx/compile/runs/20261017T034942-2182085-13658/build-1.log`；审计：同目录 `install-report-1.json`，`modified=2, removed=0, missing=0`。
- 两端磁盘及运行进程 SHA-256 均为 `e261cfa6ee38b333cf423435ab6d3d971b2a1f07b4783d8d01aa935419298573`。main/free 均 active，首页与 `/health` 返回 200，端口、profile、数据库、Redis 与 free scheduler prefix 隔离核验通过。
- main 保留 `sub2api.bak.20261017-035618-2280783`；free 保留 `sub2freeApi.bak.20261017-035618-2280783`。旧备份各归档 1 个，删除 0 个。
- 验收未发送真实上游模型请求，也未声称完成浏览器人工操作验收。后续深度耦合模块仍以本报告的迁移清单为准。

