# 统一源码双服务升级运行手册

本仓库 `/home/third_party/sub2api` 是 `sub2api.service` 与
`sub2freeApi.service` 的唯一源码和构建来源。两个进程运行同一份嵌入式
二进制字节，通过 `DEPLOYMENT_PROFILE=main|free` 选择能力；数据库、Redis、
配置、工作目录、运行用户和 systemd 生命周期仍完全隔离。

行为审计清单见 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md)，
部署矩阵见 [UNIFIED_DUAL_SERVICE_DEPLOYMENT.md](UNIFIED_DUAL_SERVICE_DEPLOYMENT.md)。

## 固定边界

| 项目 | main | free |
| --- | --- | --- |
| systemd | `sub2api.service` | `sub2freeApi.service` |
| profile | `main` | `free` |
| 端口 | `18381` | `18382` |
| PostgreSQL | `sub2api` | `sub2freeApi` |
| Redis | DB `0` | DB `1`, prefix `sub2freeApi` |
| 工作目录 | `/home/third_party/sub2api/deploy` | `/home/third_party/sub2freeApi/deploy` |
| 环境文件 | `sub2api/deploy/sub2api.env` | `sub2freeApi/deploy/sub2api.env` |
| 二进制路径 | `/home/third_party/bin/sub2api/sub2api` | `/home/third_party/bin/sub2freeApi/sub2freeApi` |

`/home/third_party/sub2freeApi` 只保留 free 运行配置、历史和灾难恢复资料，不再
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

## 1. 升级前快照

同时保存 canonical 和 legacy free 仓库的提交历史与工作区：

```bash
for repo in /home/third_party/sub2api /home/third_party/sub2freeApi; do
  name=$(basename "$repo")
  ts=$(date +%Y%m%d-%H%M%S)
  backup="/home/third_party/upgrade-backups/${name}-${ts}"
  mkdir -p "$backup"
  git -C "$repo" branch "backup-pre-upgrade-${ts}" HEAD
  git -C "$repo" bundle create "$backup/repository.bundle" --all
  git bundle verify "$backup/repository.bundle"
  git -C "$repo" status --porcelain=v1 > "$backup/status.txt"
  git -C "$repo" diff --binary > "$backup/worktree.patch"
  git -C "$repo" diff --cached --binary > "$backup/index.patch"
  git -C "$repo" ls-files --others --exclude-standard -z > "$backup/untracked.list"
done
```

另行记录两个 service 的 PID、运行二进制 SHA、端口和活动状态。备份在全部上线
验证完成前不得删除。

## 2. 合并上游

```bash
cd /home/third_party/sub2api
git fetch upstream --tags
git merge --no-ff --no-commit upstream/main
```

逐文件组合解决冲突，禁止整文件盲选 ours/theirs。对照本地定制清单验证主服务
粘性调度/迁移、free 余额检测/错误品牌、账号 API key、回收站、clone、表格布局、
限流恢复和 Redis prefix。保留操作前所有 staged、unstaged、untracked 和 ignored
本地策略文件。

合并完成后立即删除上游 sponsor 区段；该脚本只处理广告区段，不冻结 README
其他内容：

```bash
bash scripts/remove-readme-sponsors.sh
bash scripts/remove-readme-sponsors.sh --check
```

```bash
git diff --check
rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend/src docs README* || true
```

## 3. 验证

必须使用 pnpm 9。当前固定入口：

```bash
PNPM9=/home/root/.npm/_npx/8959f4e966f464e2/node_modules/pnpm/bin/pnpm.cjs
```

```bash
cd /home/third_party/sub2api/backend
go test -tags unit ./internal/config ./internal/pkg/clienterror \
  ./internal/handler/dto ./internal/handler/admin ./internal/handler \
  ./internal/server/middleware ./internal/server/routes ./internal/service \
  ./internal/repository -count=1
go test -tags unit ./... -count=1

cd /home/third_party/sub2api/frontend
node "$PNPM9" vitest run
node "$PNPM9" typecheck
node "$PNPM9" build
```

重点检查 profile/capability、错误品牌、凭据、sticky spillover/reassignment、balance
worker/settings、回收站、clone、账号表、active-only usage 和 rate-limit 恢复测试。

## 4. 一次构建

必须先生成 `frontend/dist`，再只构建一次：

```bash
cd /home/third_party/sub2api/backend
BUILD_OUT=/home/third_party/sub2api/backend/bin/sub2api-unified.new
CGO_ENABLED=0 go build -tags embed -o "$BUILD_OUT" ./cmd/server/
go version -m "$BUILD_OUT" | rg 'tags=embed|CGO_ENABLED=0'
sha256sum "$BUILD_OUT"
```

## 5. 原子部署两次

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
  /home/third_party/bin/sub2api/sub2api \
  /home/third_party/bin/sub2freeApi/sub2freeApi \
  "/proc/$MAIN_PID/exe" \
  "/proc/$FREE_PID/exe"
```

四个 SHA 必须相同。还需验证：main 公开设置为 `deployment_profile=main`、只开放
sticky reassignment；free 为 `deployment_profile=free`、只开放 balance check；
两个数据库名、Redis DB/prefix、工作目录、运行用户和端口均保持固定矩阵。

最后恢复升级前 canonical 用户工作区并与快照清单逐项核对。除非明确授权，不 push。
