# sub2api 上游升级运行手册

本文件是 `/home/third_party/sub2api` 的唯一标准升级流程。定制行为的逐项清单见 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md)，项目身份和长期约束见 [../AGENTS.md](../AGENTS.md)。

## 1. 固定边界

| 项目 | 固定值 |
| --- | --- |
| 源码目录 | `/home/third_party/sub2api` |
| 上游合并源 | `upstream/main` (`Wei-Shaw/sub2api`) |
| fork 推送目标 | `origin/main` (`beyondcy1013/sub2api`) |
| systemd 服务 | `sub2api.service` |
| 运行二进制 | `/home/third_party/bin/sub2api/sub2api` |
| HTTP 端口 | `18381` |
| PostgreSQL 数据库 | `sub2api` |
| Redis DB | `0` |
| 配置文件 | `/home/third_party/sub2api/deploy/data/config.yaml` |

必须遵守：

- 只合并 `upstream/main`；`origin/main` 是本地定制 fork 的备份目标，不是上游源码。
- 保留全部本地提交、暂存区、未暂存改动、未跟踪文件以及被忽略的 `AGENTS.md`、`.codex/skills`。
- 禁止 `git reset --hard`、`git clean`、强制推送和丢弃本地提交的 rebase。
- 禁止使用 WebUI 一键二进制升级；官方二进制不包含本地定制。
- 一次只能有一个操作者进行 merge、build、deploy 或 push。发现其他升级会话时先协调所有权。
- 升级失败时保留恢复分支和备份目录，不要为了得到“干净状态”删除证据。

## 2. 流程总览

严格按以下顺序执行，每一阶段成功后才能进入下一阶段：

1. 检查远端、版本、工作区和当前运行态。
2. 创建恢复分支、Git bundle、补丁和本地文件归档。
3. 暂存 tracked 改动，合并 `upstream/main`，再恢复原工作区。
4. 审计并修复本地定制；不得整文件采用 ours/theirs。
5. 完成后端、前端和定制专项测试。
6. 用 `-tags embed` 构建，备份并原子替换二进制。
7. 只重启 `sub2api.service`，验证版本、端口、数据隔离和关键行为。
8. 确认 fork 无并发更新后普通推送到 `origin/main`，禁止 force。
9. 全部验证通过后删除本次临时恢复资料；保留旧运行二进制备份。

## 3. 升级前检查

GitHub 访问必须避开错误的 `127.0.0.1:12111` 环境代理，使用本机 GitHub 代理 `127.0.0.1:7890`：

```bash
export http_proxy="" https_proxy="" all_proxy="" ALL_PROXY=""
export HTTP_PROXY="http://127.0.0.1:7890"
export HTTPS_PROXY="http://127.0.0.1:7890"

cd /home/third_party/sub2api
git remote -v
git status --short
git branch --show-current
git log -5 --oneline --decorate
git fetch upstream --tags

CURRENT_VER=$(cat backend/cmd/server/VERSION)
UPSTREAM_VER=$(git show upstream/main:backend/cmd/server/VERSION)
printf 'current=%s upstream=%s\n' "$CURRENT_VER" "$UPSTREAM_VER"
git rev-list --left-right --count HEAD...upstream/main

systemctl show sub2api.service \
  -p ActiveState -p MainPID -p ExecStart -p WorkingDirectory -p EnvironmentFiles
ss -ltnp | rg ':18381\b'
curl --noproxy '*' -fsS http://127.0.0.1:18381/ >/dev/null
```

记录 `git status --short` 的原始输出。现有 dirty/untracked 文件属于用户，后面必须原样恢复。

## 4. 创建完整恢复点

不要只建一个分支：分支无法保存 dirty 状态和未跟踪文件。四类恢复资料必须同时存在。

```bash
cd /home/third_party/sub2api
TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/home/third_party/upgrade-backups/sub2api-$TS
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

# 保存所有普通 untracked 文件，并显式包含被忽略的本地规则/技能。
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

把 `TS`、`BACKUP`、`BACKUP_BRANCH` 保留在当前 shell 中，直到清理阶段。

## 5. 合并上游并恢复原工作区

只 stash tracked 改动，不包含 untracked 文件：

```bash
cd /home/third_party/sub2api
STASH_CREATED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  git stash push -m "pre-upgrade-$TS"
  STASH_CREATED=1
fi

git fetch upstream --tags
git merge --no-ff upstream/main
```

若 merge 冲突：

1. 用 `git status --short` 列出每一个冲突文件。
2. 对照 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md) 合并双方行为。
3. 不得对整文件使用 `git checkout --ours` 或 `--theirs` 后直接结束。
4. 解决后运行下面的冲突标记检查，再 `git add` 和 `git commit`。

```bash
rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend/src || true
git diff --check
git status --short
```

merge 完成后恢复升级前的 tracked 状态：

```bash
if [ "$STASH_CREATED" -eq 1 ]; then
  git stash pop --index
fi

git status --short
git log --oneline --graph --decorate -20
git log --reverse --oneline upstream/main..HEAD
git diff --stat upstream/main...HEAD
```

`stash pop` 若冲突，必须结合 `$BACKUP/worktree.patch`、`$BACKUP/index.patch` 和备份分支恢复，不能丢弃 stash。

## 6. 本地定制审计

完整合同在 [LOCAL_UPGRADE_CUSTOMIZATIONS.md](LOCAL_UPGRADE_CUSTOMIZATIONS.md)。至少逐项确认：

- 管理端 API Key 明文查看/编辑仍存在；`api_key` 不在 `SensitiveCredentialKeys`，遗漏字段更新仍保留旧 key。
- 账号表列顺序、固定宽度、单行表头、边距、分隔线、回收站和非 active 账号 usage guard 仍存在。
- 新建/导入账号默认并发为 `4`；账号菜单始终有 `恢复状态`。
- `迁入粘性会话` 的 1/5/15/60 分钟窗口、CAS + `KEEPTTL`、16 位 session hash 和 continuation 排除规则仍存在。
- 历史粘性账号满并发时可溢出到同组空闲账号，且不改写历史绑定；`previous_response_id` 仍严格固定。
- OpenAI 全局 5h/7d quota auto-pause 默认阈值仍为 `0/0`，派生的 `quota_rate_limit` 不写回持久状态。
- rate-limit DB 持久化使用 detached context；成功恢复账号时即使 DB 已干净也会清 runtime block。

常用审计命令：

```bash
rg -n 'SensitiveCredentialKeys|PreserveOnMissingCredentialKeys|api_key' \
  backend/internal/service backend/internal/handler/dto frontend/src/components/account
rg -n 'preserveStickyBinding|sticky_escape_enabled|previous_response_id|sticky-sessions|KEEPTTL' \
  backend/internal/service backend/internal/handler frontend/src
rg -n 'rateLimitPersistTimeout|WithoutCancel|notifyAccountSchedulingBlockCleared' \
  backend/internal/service/ratelimit_service.go
rg -n 'shouldAutoLoadUsageOnMount|quota_rate_limit|default_threshold_(5h|7d)' \
  backend/internal frontend/src
rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend/src || true
git diff --check
```

## 7. 测试和前端构建

`frontend/pnpm-lock.yaml` 是 lockfile v9，必须使用 pnpm 9；pnpm 8 会误报 lockfile 或缺依赖。

```bash
cd /home/third_party/sub2api/backend

# 本地 unit-tag 定制测试。
go test -tags unit ./internal/handler/dto ./internal/service \
  -run 'TestRedactCredentials|TestAccountFromServiceShallow|TestMergePreservingSensitiveCreds|TestIsSensitiveCredentialKey' \
  -count=1

# 调度、回收站、限流恢复等专项。
go test ./internal/repository ./internal/handler/admin ./internal/service -count=1

# 完整后端回归。
go test ./...

cd /home/third_party/sub2api/frontend

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
  src/components/account/__tests__/EditAccountModal.spec.ts \
  src/components/account/__tests__/AccountUsageCell.spec.ts \
  src/components/account/__tests__/CreateAccountModal.spec.ts \
  src/components/admin/account/__tests__/StickySessionReassignModal.spec.ts \
  src/components/admin/account/__tests__/AccountActionMenu.spark_shadow.spec.ts \
  src/components/common/__tests__/DataTable.spec.ts \
  src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
"${PNPM[@]}" vitest run
"${PNPM[@]}" typecheck
"${PNPM[@]}" build
```

任何失败都先判断是否由本次 merge 引入。未证明通过前禁止构建部署。

## 8. 构建和原子部署

必须先完成前端 `pnpm build`，然后用 `-tags embed` 把 `frontend/dist` 嵌入 Go 二进制。

```bash
REPO=/home/third_party/sub2api
DEPLOY_DIR=/home/third_party/bin/sub2api
BIN=$DEPLOY_DIR/sub2api
BUILD_OUT=$REPO/backend/bin/sub2api.new
SWAP=$DEPLOY_DIR/.sub2api.swap-$TS
OLD_BIN=$DEPLOY_DIR/sub2api.backup-$TS
SRC_VER=$(cat "$REPO/backend/cmd/server/VERSION")

cd "$REPO/backend"
CGO_ENABLED=0 go build -tags embed -o "$BUILD_OUT" ./cmd/server/
go version -m "$BUILD_OUT" | rg 'tags=embed|CGO_ENABLED=0'
rg -a -q -F "$SRC_VER" "$BUILD_OUT"

OLD_PID=$(systemctl show -p MainPID --value sub2api.service)
OLD_SHA=$(sha256sum "$BIN" | awk '{print $1}')
printf 'old_pid=%s old_sha=%s target_version=%s\n' "$OLD_PID" "$OLD_SHA" "$SRC_VER"

cp -p "$BIN" "$OLD_BIN"
install -m 0755 "$BUILD_OUT" "$SWAP"
chown --reference="$BIN" "$SWAP"
mv -f "$SWAP" "$BIN"
sha256sum "$BIN" "$OLD_BIN"

systemctl restart sub2api.service
```

`install` 只写临时文件，最后同目录 `mv` 才切换正式路径。不要直接把编译输出覆盖正在运行的二进制。

## 9. 运行态验证

```bash
systemctl is-active sub2api.service
systemctl status sub2api.service --no-pager
PID=$(systemctl show -p MainPID --value sub2api.service)
ss -ltnp | rg ':18381\b'
curl --noproxy '*' -fsS http://127.0.0.1:18381/ >/dev/null
DEPLOYED_SHA=$(sha256sum "$BIN" | awk '{print $1}')
RUNNING_SHA=$(sha256sum "/proc/$PID/exe" | awk '{print $1}')
test "$DEPLOYED_SHA" = "$RUNNING_SHA"

# 只显示非敏感隔离项，确认新进程没有串到 free 服务。
PROCESS_ENV=$(tr '\0' '\n' < "/proc/$PID/environ" | \
  rg '^(DATABASE_DBNAME|REDIS_DB|REDIS_SCHEDULER_KEY_PREFIX|SERVER_PORT|DATA_DIR)=' | sort)
printf '%s\n' "$PROCESS_ENV"
printf '%s\n' "$PROCESS_ENV" | rg -q '^DATABASE_DBNAME=sub2api$'
printf '%s\n' "$PROCESS_ENV" | rg -q '^REDIS_DB=0$'
printf '%s\n' "$PROCESS_ENV" | rg -q '^SERVER_PORT=18381$'
```

预期至少包含 `DATABASE_DBNAME=sub2api`、`REDIS_DB=0`、`SERVER_PORT=18381`。

通过认证接口核对 live 版本，变量不得回显：

```bash
set -a
. /home/third_party/sub2api/deploy/sub2api.env
set +a
LOGIN_JSON=$(jq -nc --arg email "$ADMIN_EMAIL" --arg password "$ADMIN_PASSWORD" \
  '{email:$email,password:$password}')
TOKEN=$(curl --noproxy '*' -fsS -X POST http://127.0.0.1:18381/api/v1/auth/login \
  -H 'Content-Type: application/json' --data "$LOGIN_JSON" | jq -er '.data.access_token')
LIVE_VER=$(curl --noproxy '*' -fsS \
  -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:18381/api/v1/admin/system/version | jq -er '.data.version')
test "$LIVE_VER" = "$SRC_VER"

QUOTA_DEFAULTS=$(PGPASSWORD="$DATABASE_PASSWORD" psql \
  -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DATABASE_DBNAME" \
  -At -F '|' -c "SELECT value::jsonb #>> '{openai_account_quota_auto_pause,default_threshold_5h}', value::jsonb #>> '{openai_account_quota_auto_pause,default_threshold_7d}' FROM settings WHERE key = 'ops_advanced_settings'")
test "$QUOTA_DEFAULTS" = '0|0'
unset TOKEN LOGIN_JSON ADMIN_PASSWORD DATABASE_PASSWORD REDIS_PASSWORD JWT_SECRET
```

最后检查新 PID 启动后的日志，没有 panic、fatal 或持续数据库/Redis 错误。

若任一 live 检查失败，立即按“回滚”章节恢复旧二进制，不要 push。

## 10. 推送 fork

只允许普通 fast-forward push：

```bash
cd /home/third_party/sub2api
git fetch origin main

# origin 有别人提交而当前 HEAD 不包含它时必须停止协调，禁止 force。
git merge-base --is-ancestor origin/main HEAD
git status --short
git push origin HEAD:main

LOCAL_SHA=$(git rev-parse HEAD)
REMOTE_SHA=$(git ls-remote origin refs/heads/main | awk '{print $1}')
test "$LOCAL_SHA" = "$REMOTE_SHA"
git rev-list --left-right --count HEAD...origin/main
```

不得把临时备份分支、构建产物、凭据或原有 untracked 文件加入提交。

## 11. 成功后清理

只有测试、构建、部署、live 验证和远端 SHA 全部通过后才能执行：

```bash
rm -f "$BUILD_OUT" "$SWAP"
case "$BACKUP" in
  /home/third_party/upgrade-backups/sub2api-*) ;;
  *) echo "FATAL: unexpected backup path: $BACKUP"; exit 1 ;;
esac
git branch -d "$BACKUP_BRANCH"
rm -rf -- "$BACKUP"
git status --short
```

保留 `$OLD_BIN`，至少到确认新版本稳定运行后再单独决定是否删除。

## 12. 回滚

### 尚未完成 merge

有 merge 冲突且决定停止时使用 `git merge --abort`。不要 reset。若已经 stash，确认原 stash 仍存在，再恢复并对照 `$BACKUP/status.txt`。

### 已部署但运行失败

```bash
ROLLBACK_SWAP=$DEPLOY_DIR/.sub2api.rollback-$TS
install -m 0755 "$OLD_BIN" "$ROLLBACK_SWAP"
chown --reference="$BIN" "$ROLLBACK_SWAP"
mv -f "$ROLLBACK_SWAP" "$BIN"
systemctl restart sub2api.service
systemctl is-active sub2api.service
ss -ltnp | rg ':18381\b'
curl --noproxy '*' -fsS http://127.0.0.1:18381/ >/dev/null
```

二进制回滚不应修改数据库、Redis 或 free 服务。源代码恢复需要从 `$BACKUP_BRANCH`、bundle 和补丁中有选择地恢复，禁止用破坏性命令覆盖升级后的调查现场。

## 13. 完成标准

只有同时满足以下条件才可报告升级完成：

- 上游版本和 merge commit 已记录，原 dirty/untracked 状态没有丢失。
- 定制清单逐项审计通过，无冲突标记和 `git diff --check` 错误。
- 后端完整测试、定制专项、前端完整测试、typecheck、build 全部退出码为 0。
- live API 版本等于源码版本；新 PID 监听 18381，HTTP 200。
- 新进程使用数据库 `sub2api`、Redis DB 0，未触碰 `sub2freeApi.service`。
- `origin/main` SHA 与本地 HEAD 一致且未 force push。
- 临时恢复资料只在上述全部通过后清理，旧二进制备份仍保留。
- 提醒浏览器使用 `Ctrl+Shift+R`（macOS 为 `Cmd+Shift+R`）强制刷新缓存。
