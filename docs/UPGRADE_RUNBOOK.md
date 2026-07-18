# sub2freeApi 上游升级运行手册

本文件是 `/home/third_party/sub2freeApi` 的唯一标准升级流程。定制行为的逐项清单见 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md)，项目身份和长期约束见 [../AGENTS.md](../AGENTS.md)。

## 1. 固定边界

| 项目 | 固定值 |
| --- | --- |
| 源码目录 | `/home/third_party/sub2freeApi` |
| 本地开发分支 | `main`（保留完整 merge 历史） |
| 上游合并源 | `upstream/main` (`Wei-Shaw/sub2api`) |
| fork 备份目标 | `origin/sub2freeApi` (`beyondcy1013/sub2api`) |
| fork 本地快照分支 | `sub2freeApi-clean` |
| systemd 服务 | `sub2freeApi.service` |
| 运行二进制 | `/home/third_party/bin/sub2freeApi/sub2freeApi` |
| HTTP 端口 | `18382` |
| PostgreSQL 数据库 | `sub2freeApi` |
| Redis DB / scheduler prefix | `1` / `sub2freeApi` |
| 配置文件 | `/home/third_party/sub2freeApi/deploy/data/config.yaml` |

必须遵守：

- 只合并 `upstream/main`。绝不能把 `origin/main` 合并进 free 项目；fork 的 `main` 属于主项目。
- `origin/sub2freeApi` 是干净源码快照历史，与本地 `main` 的完整历史不同；按本手册的 `git commit-tree` 流程更新，禁止 force。
- 保留全部本地提交、暂存区、未暂存改动、未跟踪文件以及被忽略的 `AGENTS.md`、`.codex/skills`。
- 禁止 `git reset --hard`、`git clean`、丢弃本地提交的 rebase 和 WebUI 一键二进制升级。
- 一次只能有一个操作者进行 merge、build、deploy 或 push。发现其他升级会话时先协调所有权。
- 升级失败时保留恢复分支和备份目录，不要删除失败现场。

## 2. 流程总览

1. 检查远端、版本、工作区、凭据 URL 和当前运行态。
2. 创建恢复分支、Git bundle、tracked 补丁和本地文件归档。
3. stash tracked 改动，合并 `upstream/main`，再恢复原工作区。
4. 对照定制清单逐项审计，组合解决冲突。
5. 完成后端、前端、balance、错误前缀和回收站专项测试。
6. 用 pnpm 9 构建前端，再用 `-tags embed` 构建 Go 二进制。
7. 备份并原子替换二进制，只重启 `sub2freeApi.service`。
8. 验证版本、18382、数据库 `sub2freeApi`、Redis DB 1/prefix 和错误前缀。
9. 将当前源码树追加为 `origin/sub2freeApi` 的新干净快照，核对远端 tree。
10. 全部通过后删除本次临时恢复资料；保留旧运行二进制备份。

## 3. 升级前检查

GitHub 访问应使用当前安全网络配置，禁止把代理凭据或个人代理地址写入仓库：

```bash

cd /home/third_party/sub2freeApi
git remote -v
test "$(git remote get-url origin)" = "https://github.com/beyondcy1013/sub2api.git"
git status --short
git branch --show-current
git log -5 --oneline --decorate
git fetch upstream --tags
git fetch origin sub2freeApi

CURRENT_VER=$(cat backend/cmd/server/VERSION)
UPSTREAM_VER=$(git show upstream/main:backend/cmd/server/VERSION)
printf 'current=%s upstream=%s\n' "$CURRENT_VER" "$UPSTREAM_VER"
git rev-list --left-right --count HEAD...upstream/main

systemctl show sub2freeApi.service \
  -p ActiveState -p MainPID -p ExecStart -p WorkingDirectory -p EnvironmentFiles
ss -ltnp | rg ':18382\b'
curl --noproxy '*' -fsS http://127.0.0.1:18382/ >/dev/null
```

`origin` URL 必须是不含用户名/PAT 的 HTTPS URL。认证由 `gh auth git-credential` helper 提供；发现 URL 内嵌令牌时先移除并轮换令牌，禁止把它写进文档、日志或命令输出。

记录 `git status --short` 的原始输出。现有 dirty/untracked 文件属于用户，后面必须原样恢复。

## 4. 创建完整恢复点

```bash
cd /home/third_party/sub2freeApi
TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/home/third_party/upgrade-backups/sub2freeApi-$TS
BACKUP_BRANCH=backup-pre-upgrade-$TS
mkdir -p "$BACKUP"

# 已提交历史和全部 refs。
git branch "$BACKUP_BRANCH" HEAD
git bundle create "$BACKUP/repository.bundle" --all
git bundle verify "$BACKUP/repository.bundle"
git log --reverse --oneline upstream/main..HEAD > "$BACKUP/local-commits.txt"

# tracked 工作区与暂存区分别保存。
git status --porcelain=v1 > "$BACKUP/status.txt"
git diff --binary > "$BACKUP/worktree.patch"
git diff --cached --binary > "$BACKUP/index.patch"

# 保存普通 untracked 文件，并显式包含被忽略的本地规则/技能。
{
  git ls-files --others --exclude-standard -z
  [ -f AGENTS.md ] && printf 'AGENTS.md\0'
  [ -d .codex/skills ] && find .codex/skills -type f -print0
} | sort -zu > "$BACKUP/local-files.list"
if [ -s "$BACKUP/local-files.list" ]; then
  tar --null -czf "$BACKUP/local-files.tar.gz" -T "$BACKUP/local-files.list"
fi

test -s "$BACKUP/repository.bundle"
test -f "$BACKUP/status.txt"
git show-ref --verify "refs/heads/$BACKUP_BRANCH"
```

不要只依赖备份分支：它不包含 dirty 状态和 untracked 文件。保留 `TS`、`BACKUP`、`BACKUP_BRANCH` 到清理阶段。

## 5. 合并上游并恢复原工作区

```bash
cd /home/third_party/sub2freeApi
STASH_CREATED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  git stash push -m "pre-upgrade-$TS"
  STASH_CREATED=1
fi

git fetch upstream --tags
git merge --no-ff upstream/main
```

若 merge 冲突：

1. `git status --short` 列出全部冲突文件。
2. 对照 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md) 同时保留上游能力和 free 定制。
3. 不得整文件采用 ours/theirs；特别注意 `clienterror`、`accounts.ts`、`AccountsView.vue`、balance service 和 gateway writers。
4. 解决后检查冲突标记和 whitespace，再提交 merge。

```bash
rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend/src || true
git diff --check
git status --short
```

merge 完成后恢复升级前 tracked 状态：

```bash
if [ "$STASH_CREATED" -eq 1 ]; then
  git stash pop --index
fi

git status --short
git log --oneline --graph --decorate -20
git log --reverse --oneline upstream/main..HEAD
git diff --stat upstream/main...HEAD
```

`stash pop` 若冲突，结合 `$BACKUP/worktree.patch`、`$BACKUP/index.patch`、备份分支和 local-files 归档恢复；不能丢弃 stash。

## 6. free 项目定制审计

完整合同在 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md)。至少确认：

- 管理端账号标识和 `credentials.api_key` 仍可明文查看/编辑；其他 OAuth、cookie、private key 等继续脱敏。
- `MergePreservingSensitiveCreds` 在旧前端未提交 `api_key` 时保留已有值。
- `clienterror` 包以及 gateway/middleware/service writers 仍正确区分 `【sub2freeApi限制】` 与 `【上游错误】`，包括 SSE `response.failed`。
- `accounts.ts` 同时保留 free 的 `clone()` 和上游新增能力；回收/恢复 API 与 UI 仍存在。
- 新建账号仍默认选择代理列表最后一个、分组列表第一个；候选项晚到时只补空值，clone 模式保留来源账号的代理/分组（包括空值）。
- 账号表列顺序、固定宽度、余额 70px、单行表头、边距和分隔线没有退化。
- balance-check 配置、页面、服务、定时任务及其测试仍存在。
- 非 active 账号不会自动请求 `/usage`，手动刷新不受影响。
- rate-limit DB 持久化使用 detached context；测试成功恢复账号时，即使 DB 已干净也会清 runtime block。
- Redis scheduler key prefix 仍为 `sub2freeApi`，不能与主服务共享调度 key。

常用审计命令：

```bash
rg -n 'SensitiveCredentialKeys|PreserveOnMissingCredentialKeys|api_key' \
  backend/internal/service backend/internal/handler/dto frontend/src/components/account
rg -n 'clienterror|sub2freeApi限制|上游错误|response.failed|writeResponsesFailedSSE' \
  backend/internal
rg -n 'clone\(|recycleAccount|restoreAccount|toggleRecycled' \
  frontend/src/api frontend/src/components frontend/src/views
rg -n 'balance.check|balance_check|scheduler_key_prefix|shouldAutoLoadUsageOnMount' \
  backend frontend/src deploy/data/config.yaml
rg -n 'rateLimitPersistTimeout|WithoutCancel|notifyAccountSchedulingBlockCleared' \
  backend/internal/service/ratelimit_service.go
rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend/src || true
git diff --check
```

## 7. 测试和前端构建

`frontend/pnpm-lock.yaml` 是 lockfile v9，必须使用 pnpm 9；系统 pnpm 8 不能作为验证工具。

```bash
cd /home/third_party/sub2freeApi/backend

# free 错误前缀、凭据、回收站、balance 和限流恢复专项。
go test ./internal/pkg/clienterror ./internal/server/middleware \
  ./internal/handler ./internal/handler/dto ./internal/repository ./internal/service \
  -count=1

# 完整后端回归。
go test ./...

cd /home/third_party/sub2freeApi/frontend

# 选择 pnpm 9。当前系统全局 pnpm 可能仍是 8.x，不能直接使用。
if [[ "$(pnpm --version 2>/dev/null || true)" == 9.* ]]; then
  PNPM=(pnpm)
else
  find_cached_pnpm9() {
    local candidate version
    while IFS= read -r candidate; do
      version=$(node "$candidate" --version 2>/dev/null || true)
      if [[ "$version" == 9.* ]]; then
        printf '%s\n' "$candidate"
        return 0
      fi
    done < <(find "$HOME/.npm/_npx" -type f \
      -path '*/node_modules/pnpm/bin/pnpm.cjs' -print 2>/dev/null)
    return 1
  }
  PNPM9_CJS=$(find_cached_pnpm9 || true)
  if [ -z "$PNPM9_CJS" ]; then
    npx --yes pnpm@9.15.9 --version
    PNPM9_CJS=$(find_cached_pnpm9)
  fi
  PNPM=(node "$PNPM9_CJS")
fi
"${PNPM[@]}" --version        # 必须输出 9.x
"${PNPM[@]}" install --frozen-lockfile
"${PNPM[@]}" vitest run \
  src/components/account/__tests__/CreateAccountModal.spec.ts \
  src/components/account/__tests__/EditAccountModal.spec.ts \
  src/components/account/__tests__/AccountUsageCell.spec.ts \
  src/components/common/__tests__/DataTable.spec.ts \
  src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
"${PNPM[@]}" vitest run
"${PNPM[@]}" typecheck
"${PNPM[@]}" build
```

任何失败都先确定是否由本次 merge 引入。未获得完整退出码 0 前禁止部署。

## 8. 构建和原子部署

必须先生成 `frontend/dist`，再用 `-tags embed` 构建，否则 18382 的前端路由会返回 404。

```bash
REPO=/home/third_party/sub2freeApi
DEPLOY_DIR=/home/third_party/bin/sub2freeApi
BIN=$DEPLOY_DIR/sub2freeApi
BUILD_OUT=$REPO/backend/bin/sub2freeApi.new
SWAP=$DEPLOY_DIR/.sub2freeApi.swap-$TS
OLD_BIN=$DEPLOY_DIR/sub2freeApi.backup-$TS
SRC_VER=$(cat "$REPO/backend/cmd/server/VERSION")

cd "$REPO/backend"
CGO_ENABLED=0 go build -tags embed -o "$BUILD_OUT" ./cmd/server/
go version -m "$BUILD_OUT" | rg 'tags=embed|CGO_ENABLED=0'
rg -a -q -F "$SRC_VER" "$BUILD_OUT"

OLD_PID=$(systemctl show -p MainPID --value sub2freeApi.service)
OLD_SHA=$(sha256sum "$BIN" | awk '{print $1}')
printf 'old_pid=%s old_sha=%s target_version=%s\n' "$OLD_PID" "$OLD_SHA" "$SRC_VER"

cp -p "$BIN" "$OLD_BIN"
install -m 0755 "$BUILD_OUT" "$SWAP"
chown --reference="$BIN" "$SWAP"
mv -f "$SWAP" "$BIN"
sha256sum "$BIN" "$OLD_BIN"

systemctl restart sub2freeApi.service
```

`install` 只写临时路径，最后同目录 `mv` 才切换正式二进制。绝不能重启 `sub2api.service`。

## 9. 运行态验证

```bash
systemctl is-active sub2freeApi.service
systemctl status sub2freeApi.service --no-pager
PID=$(systemctl show -p MainPID --value sub2freeApi.service)
ss -ltnp | rg ':18382\b'
curl --noproxy '*' -fsS http://127.0.0.1:18382/ >/dev/null
DEPLOYED_SHA=$(sha256sum "$BIN" | awk '{print $1}')
RUNNING_SHA=$(sha256sum "/proc/$PID/exe" | awk '{print $1}')
test "$DEPLOYED_SHA" = "$RUNNING_SHA"

# 仅显示非敏感隔离项。
PROCESS_ENV=$(tr '\0' '\n' < "/proc/$PID/environ" | \
  rg '^(DATABASE_DBNAME|REDIS_DB|REDIS_SCHEDULER_KEY_PREFIX|SERVER_PORT|DATA_DIR)=' | sort)
printf '%s\n' "$PROCESS_ENV"
printf '%s\n' "$PROCESS_ENV" | rg -q '^DATABASE_DBNAME=sub2freeApi$'
printf '%s\n' "$PROCESS_ENV" | rg -q '^REDIS_DB=1$'
printf '%s\n' "$PROCESS_ENV" | rg -q '^REDIS_SCHEDULER_KEY_PREFIX=sub2freeApi$'
printf '%s\n' "$PROCESS_ENV" | rg -q '^SERVER_PORT=18382$'
```

必须看到 `DATABASE_DBNAME=sub2freeApi`、`REDIS_DB=1`、`REDIS_SCHEDULER_KEY_PREFIX=sub2freeApi`、`SERVER_PORT=18382`。

核对 live 版本和 free 错误前缀，变量不得回显：

```bash
set -a
. /home/third_party/sub2freeApi/deploy/sub2api.env
set +a
LOGIN_JSON=$(jq -nc --arg email "$ADMIN_EMAIL" --arg password "$ADMIN_PASSWORD" \
  '{email:$email,password:$password}')
TOKEN=$(curl --noproxy '*' -fsS -X POST http://127.0.0.1:18382/api/v1/auth/login \
  -H 'Content-Type: application/json' --data "$LOGIN_JSON" | jq -er '.data.access_token')
LIVE_VER=$(curl --noproxy '*' -fsS \
  -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:18382/api/v1/admin/system/version | jq -er '.data.version')
test "$LIVE_VER" = "$SRC_VER"

curl --noproxy '*' -sS -X POST http://127.0.0.1:18382/v1/responses \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-5","input":"hi"}' | rg -q '【sub2freeApi限制】'

CONNECTED_DB=$(PGPASSWORD="$DATABASE_PASSWORD" psql \
  -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_DBNAME" \
  -At -c 'SELECT current_database()')
test "$CONNECTED_DB" = 'sub2freeApi'

curl --noproxy '*' -fsS http://127.0.0.1:18382/admin/balance-check-settings >/dev/null
curl --noproxy '*' -fsS -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:18382/api/v1/admin/settings/balance-check | \
  jq -e '.data.config' >/dev/null

REDIS_ARGS=(-h "$REDIS_HOST" -p "$REDIS_PORT" -n "$REDIS_DB" --no-auth-warning)
if [ -n "${REDIS_PASSWORD:-}" ]; then
  export REDISCLI_AUTH="$REDIS_PASSWORD"
fi
test "$(redis-cli "${REDIS_ARGS[@]}" PING)" = 'PONG'
PREFIXED_KEYS=$(redis-cli "${REDIS_ARGS[@]}" --scan \
  --pattern "${REDIS_SCHEDULER_KEY_PREFIX}*" | wc -l)
printf 'scheduler_prefixed_keys=%s\n' "$PREFIXED_KEYS"
unset REDISCLI_AUTH || true
unset TOKEN LOGIN_JSON ADMIN_PASSWORD DATABASE_PASSWORD REDIS_PASSWORD JWT_SECRET
```

检查新 PID 启动后的日志，无 panic、fatal 或持续 DB/Redis 错误。

任一 live 检查失败都立即回滚旧二进制，不得 push。

## 10. 推送 `origin/sub2freeApi` 干净快照

free 的本地 `main` 保留完整合并历史；fork 分支采用连续的干净 tree 快照，避免历史大文件问题。因此不要运行 `git push origin main:sub2freeApi`，也不要 force。

下面用当前远端快照作为 parent，新建一个 tree 等于本地 `HEAD` 的提交。若别人并发更新了远端，普通 push 会安全失败：

```bash
cd /home/third_party/sub2freeApi
git fetch origin sub2freeApi
git status --short

LOCAL_HEAD=$(git rev-parse HEAD)
LOCAL_TREE=$(git rev-parse 'HEAD^{tree}')
REMOTE_PARENT=$(git rev-parse origin/sub2freeApi)
REMOTE_TREE=$(git rev-parse 'origin/sub2freeApi^{tree}')

if [ "$LOCAL_TREE" != "$REMOTE_TREE" ]; then
  SNAPSHOT_COMMIT=$(printf 'snapshot: sub2freeApi tree from %s\n' "$LOCAL_HEAD" | \
    git commit-tree "$LOCAL_TREE" -p "$REMOTE_PARENT")
  git branch -f sub2freeApi-clean "$SNAPSHOT_COMMIT"
  git push origin "$SNAPSHOT_COMMIT:refs/heads/sub2freeApi"
fi

REMOTE_SHA=$(git ls-remote origin refs/heads/sub2freeApi | awk '{print $1}')
git fetch origin sub2freeApi
REMOTE_TREE=$(git rev-parse 'origin/sub2freeApi^{tree}')
test "$LOCAL_TREE" = "$REMOTE_TREE"
test "$REMOTE_SHA" = "$(git rev-parse origin/sub2freeApi)"
printf 'local_head=%s remote_snapshot=%s tree=%s\n' \
  "$LOCAL_HEAD" "$REMOTE_SHA" "$LOCAL_TREE"
```

若 `sub2freeApi-clean` 正被其他 worktree checkout，`git branch -f` 会拒绝；先协调并解除该 worktree 占用，不要删 worktree 或 force。快照提交不改变当前 `main` 工作区。

## 11. 成功后清理

只有测试、构建、部署、live 验证和远端 tree 全部通过后才能执行：

```bash
rm -f "$BUILD_OUT" "$SWAP"
case "$BACKUP" in
  /home/third_party/upgrade-backups/sub2freeApi-*) ;;
  *) echo "FATAL: unexpected backup path: $BACKUP"; exit 1 ;;
esac
git branch -d "$BACKUP_BRANCH"
rm -rf -- "$BACKUP"
git status --short
```

保留 `$OLD_BIN`，至少到确认新版本稳定运行后再单独决定是否删除。

## 12. 回滚

### 尚未完成 merge

有 merge 冲突且决定停止时用 `git merge --abort`。不要 reset。若已 stash，确认 stash 存在，再对照 `$BACKUP/status.txt`、补丁和本地文件归档恢复。

### 已部署但运行失败

```bash
ROLLBACK_SWAP=$DEPLOY_DIR/.sub2freeApi.rollback-$TS
install -m 0755 "$OLD_BIN" "$ROLLBACK_SWAP"
chown --reference="$BIN" "$ROLLBACK_SWAP"
mv -f "$ROLLBACK_SWAP" "$BIN"
systemctl restart sub2freeApi.service
systemctl is-active sub2freeApi.service
ss -ltnp | rg ':18382\b'
curl --noproxy '*' -fsS http://127.0.0.1:18382/ >/dev/null
```

二进制回滚不修改数据库、Redis 或主服务。源代码恢复应使用 `$BACKUP_BRANCH`、bundle 和补丁有选择地处理，禁止覆盖失败现场。

## 13. 完成标准

只有同时满足以下条件才可报告升级完成：

- 上游版本和 merge commit 已记录，原 dirty/untracked 状态无丢失。
- free 定制逐项审计通过，无冲突标记和 `git diff --check` 错误。
- 后端完整测试、free 专项、前端完整测试、typecheck、build 全部退出码为 0。
- live API 版本等于源码版本；新 PID 监听 18382，HTTP 200，错误含 `【sub2freeApi限制】`。
- 新进程使用数据库 `sub2freeApi`、Redis DB 1、scheduler prefix `sub2freeApi`，未触碰主服务。
- `origin/sub2freeApi` 的 tree 与本地 `HEAD` tree 完全一致，且没有 force push。
- origin URL 不含 PAT；历史上暴露过的 PAT 已轮换。
- 临时恢复资料只在全部通过后清理，旧二进制备份保留。
- 提醒浏览器使用 `Ctrl+Shift+R`（macOS 为 `Cmd+Shift+R`）强制刷新缓存。
