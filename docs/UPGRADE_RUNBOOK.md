# 统一源码双服务升级运行手册

本仓库 `/home/codes/ai/sub2api` 是 `sub2api.service` 与
`sub2freeApi.service` 的唯一源码和构建来源。两个进程运行同一份嵌入式
二进制字节，通过 `DEPLOYMENT_PROFILE=main|free` 选择能力；数据库、Redis、
配置、工作目录、运行用户和 systemd 生命周期仍完全隔离。

行为审计清单见 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md)，
部署矩阵见 [UNIFIED_DUAL_SERVICE_DEPLOYMENT.md](UNIFIED_DUAL_SERVICE_DEPLOYMENT.md)。

## 上游项目仓库

| 项目 | 仓库 | git remote / 说明 |
| --- | --- | --- |
| 主程序 Sub2API | https://github.com/Wei-Shaw/sub2api | `upstream`，唯一合并来源（`upstream/main`） |
| 定制 fork 备份 | https://github.com/beyondcy1013/sub2api | `origin`，仅作为 push 目标，禁止从中合并 |
| 社区分支参考 | https://github.com/ranxi2001/sub2api | `ranxi`，只读参考，不作为合并来源 |
| OpenAI Basis Points 插件 | `github.com/Wei-Shaw/sub2api-oai-basispoints` | `.s2plugin` 的源 module（经 `replace ../../backend` 复用宿主 plugin API）；对应 GitHub 仓库目前公网不可见，安装包按厂商分发件处理 |

插件协议实现另有两个来源参考（见 `backend/internal/modules/basispoints/NOTICE.md`）：
协议代码移植自 https://github.com/hloolx/codex2api（MIT，公网可访问），复核参考
https://github.com/JaxsonWang/cpa-plugin-oai-basispoints——后者是 CPA 插件 ABI，
与本部署使用的 `.s2plugin` 不兼容，不得混用。

## 固定边界

| 项目 | main | free |
| --- | --- | --- |
| systemd | `sub2api.service` | `sub2freeApi.service` |
| profile | `main` | `free` |
| 端口 | `18381` | `18382` |
| PostgreSQL | `sub2api` | `sub2freeApi` |
| Redis | DB `0` | DB `1`, prefix `sub2freeApi` |
| 工作目录 | `/home/codes/ai/sub2api/deploy` | `/home/codes/ai/sub2freeApi/deploy` |
| 环境文件 | `sub2api/deploy/sub2api.env` | `sub2freeApi/deploy/sub2api.env` |
| 二进制路径 | `/home/codes/third_party/bin/sub2api/sub2api` | `/home/codes/third_party/bin/sub2freeApi/sub2freeApi` |

`/home/codes/ai/sub2freeApi` 只保留 free 运行配置、历史和灾难恢复资料，不再
作为构建源。禁止从两个仓库分别构建。

## 禁止事项

- 只合并 canonical 仓库的 `upstream/main`；`origin/main` 仅作定制 fork 备份目标。
- 禁止 WebUI 二进制更新、`git reset --hard`、`git clean`、丢提交的 rebase、
  force push 和直接覆盖运行二进制。
- 禁止共享数据库、Redis DB/prefix、环境文件、工作目录或运行用户。
- 一次构建后，禁止为第二个服务重新构建或修改产物。
- 未完成测试时不得重启；一个服务失败时只回滚该服务。
- 三种语言 README 的 sponsor 广告区段属于明确排除的上游内容；功能文档、许可证
  和项目致谢仍正常合并。

## 1. 升级前快照与并行写入者

先读 `skills/sub2api-ops/SKILL.md`，核对远程、dirty 文件、运行路径与两服务基线。
检查 `.github/workflows/upstream-sync.yml` 是否仍会定时直接推送 main，记录潜在并行写入者。
不要在分析或文档任务中运行以下准备脚本，它会修改 Git 状态并发起 fetch/merge。

已授权升级时，从 canonical 仓库运行版本化入口：

```bash
bash scripts/upgrade-from-fork.sh
```

它保存两仓库的已验证 bundle、备份分支、index/worktree patch、untracked 与本地策略/技能
归档和运行基线，并记录准确 stash OID 到 `upgrade-state.env`。检查实际归档清单是否包含
所有 ignored 策略和技能目录；Git bundle 本身不包含脏文件。备份包含敏感本地材料，
不得上传到公开仓库。脚本会继续 fetch 并开始不提交的 merge，不是 dry-run。

## 2. 分两层解决合并

先只解决 upstream merge，逐块组合上游与本地功能，不整文件 ours/theirs。
冲突解决后执行 README 清理与检查、冲突标记检查、`git diff --check`，审查后提交 merge。
不要过早 `git add` 冲突文件丢失 index 的 base/ours/theirs 阶段。

再按状态文件记录的 OID 恢复原工作区：

```bash
git stash apply --index <recorded-stash-OID>
```

解决本地工作层的冲突，与升级前清单逐项核对，确认恢复完整后才删除对应 stash。
不得把 `stash@{0}` 当作稳定身份。恢复必须在最终测试、构建和部署之前完成，
否则会把未包含本地定制的版本上线。

重新执行 README 广告清理（保留功能文档、许可证和致谢）：

```bash
bash scripts/remove-readme-sponsors.sh
bash scripts/remove-readme-sponsors.sh --check
git diff --check
```

逐项审查 `LOCAL_UPGRADE_CUSTOMIZATIONS.md`，登记“契约、文件、测试、结果”。
没有文本冲突不等于行为兼容；迁移按完整文件名保留，不能改名已部署的本地迁移。

## 3. 验证

所有测试/编译通过 webClx 提交；不要在本地另跑一遍全套门禁。使用 pnpm 9，
由 `scripts/build-unified-release.sh` 选择已安装的 pnpm、corepack 或固定 pnpm 9 fallback，
不再引用临时 `_npx/<hash>/pnpm.cjs`。局部冲突排查通过 compile-only 请求运行定向测试。
上游合并最终只排队一次完整门禁：

```bash
bash scripts/build-unified-release.sh --full
```

此命令在冻结的源码快照内运行 uncached backend unit suite、完整 Vitest、类型检查、
生产前端与嵌入式 Go 构建。日常小改使用默认 quick；不能把 quick 用于上游升级。

## 4. 一次构建

第三节的完整验证与本节构建属于同一个 webClx 请求，不重复构建。通过部署 API 排队该命令，由脚本先生成 frontend/dist，再构建一次嵌入式二进制：

```bash
cd /home/codes/ai/sub2api
bash scripts/build-unified-release.sh --full
```

日常小范围改动的自动部署不带模式参数，因此默认走 quick；本手册描述的上游升级始终显式
使用 `--full`。构建产物仍为
`/home/codes/ai/sub2api/backend/bin/sub2api-unified.new`，并由脚本验证
`embed`、`CGO_ENABLED=0` 和 SHA-256。

部署脚本通过 scripts/lib/unified-release-backups.sh 为每个服务独立创建回滚副本，旧备份归档到 upgrade-backups；定向验证包含条件调用下的健康检查失败返回。webClx 请求应显式携带当前会话 callback（本轮为 `--callback-session jungle-crawdad`），以及排队前采集的旧 SHA、main/free PID 和两个安装审计路径。不得用历史 PID/SHA 代替本次基线。

## 5. 原子部署两次

上述安装、备份、重启与验证流程已固化为 `bash scripts/deploy-unified-release.sh`：
排队前自动采集当前磁盘 SHA 与两个服务的 PID 作为前置校验；webClx 请求以
`--install-cmd bash --install-arg scripts/deploy-unified-release.sh` 携带它。
`scripts/build-unified-release.sh` 只负责编译验证，单独运行不会部署。

为两个现有二进制分别创建时间戳备份。使用 `install` 写同目录临时文件，再以
`mv` 原子切换。两次安装的输入必须是同一个 `$BUILD_OUT`。

先部署并验证 `sub2api.service`，再部署并验证 `sub2freeApi.service`。每次只重启
目标 unit，并确认端口、HTTP、profile、数据库和 Redis 隔离。失败立即把该服务的
旧二进制原子移回并只重启该服务。

## 6. 最终证明

```bash
MAIN_PID=$(systemctl show -p MainPID --value sub2api.service)
FREE_PID=$(systemctl show -p MainPID --value sub2freeApi.service)
sha256sum \
  /home/codes/third_party/bin/sub2api/sub2api \
  /home/codes/third_party/bin/sub2freeApi/sub2freeApi \
  "/proc/$MAIN_PID/exe" \
  "/proc/$FREE_PID/exe"
```

四个 SHA 必须相同。还需验证：main 公开设置为 `deployment_profile=main`、只开放
sticky reassignment；free 为 `deployment_profile=free`、只开放 balance check；
两个数据库名、Redis DB/prefix、工作目录、运行用户和端口均保持固定矩阵。

原工作区已在第二层合并中恢复，并包含在最终验证的源码内。上线后核对版本/源码快照与构建日志的对应关系；除非明确授权，不 push。推送获授权时仅推 canonical main，远程若已前进先重新评估，不得 force push。
