# Local Upgrade Customizations

This customized source build must be upgraded by merging upstream source, not by installing the official release binary.

## Preserve The Complete Repository State

The behavior checklist below is not a file allowlist. Before upgrading, preserve every local commit, all staged/unstaged tracked changes, and all untracked files:

```bash
TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/home/third_party/upgrade-backups/sub2api-$TS
BACKUP_BRANCH=backup-pre-upgrade-$TS
mkdir -p "$BACKUP"
git branch "$BACKUP_BRANCH" HEAD
git bundle create "$BACKUP/repository.bundle" --all
git log --reverse --oneline origin/main..HEAD > "$BACKUP/local-commits.txt"
git status --porcelain=v1 > "$BACKUP/status.txt"
git diff --binary > "$BACKUP/worktree.patch"
git diff --cached --binary > "$BACKUP/index.patch"
{
  git ls-files --others --exclude-standard -z
  [ -f AGENTS.md ] && printf 'AGENTS.md\0'
  [ -d .codex/skills ] && find .codex/skills -type f -print0
} | sort -zu > "$BACKUP/local-files.list"
if [ -s "$BACKUP/local-files.list" ]; then
  tar --null -czf "$BACKUP/local-files.tar.gz" -T "$BACKUP/local-files.list"
fi
STASH_CREATED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  git stash push -m "pre-upgrade-$TS"
  STASH_CREATED=1
fi
git fetch origin
git merge --no-ff origin/main
if [ "$STASH_CREATED" -eq 1 ]; then
  git stash pop --index
fi
```

Never use `git reset --hard`, `git clean`, rebase away local commits, or the WebUI one-click updater. The WebUI updater installs an official binary and does not merge or reapply local source changes.

## High-Risk Behaviors To Verify

- Admin account identifiers remain visible in plaintext.
- Admin account `credentials.api_key` remains visible/editable in plaintext, while OAuth tokens, cookies, private keys, and similar credentials remain redacted.
- Partial account updates preserve an existing API key when omitted.
- Account filters remain hidden by default behind the Filters toggle.
- The sidebar remains 154px expanded and 67px collapsed.
- Account table widths remain: select 36px, name 126px, status 80px, id 130px, platform/type 170px.
- Table headers remain single-line/non-shrinking; fixed widths use width/minWidth/maxWidth.
- Table outer edge padding remains 4px and non-final columns retain vertical separators.
- `id` and `platform_type` remain near the end before actions.
- New/imported account form initialization and reset keep default concurrency at `4`, not upstream `10`.
- The account `更多` menu always includes `恢复状态`, is vertically scrollable within the viewport, and uses the 320px positioning estimate on short screens.
- Active/schedulable OpenAI targets expose `迁入粘性会话`; the dialog defaults to the last 5 minutes and moves only recent bindings.
- `Concurrency limit exceeded for user` remains correctly described as sub2api's local caller-concurrency timeout. Do not direct operators to user management or claim sticky reassignment cannot help.
- OpenAI historical `session_hash` affinity remains capacity-aware: when the bound account cannot acquire a real concurrency slot, the current connection spills to another eligible account in the same group without rewriting the historical binding.
- Concurrency-full spillover remains independent of the advanced scheduler's TTFT/error health-escape switch. Strict, non-movable `previous_response_id` affinity is not migrated across accounts.
- The service remains isolated on `sub2api.service`, port 18381, database `sub2api`, and Redis DB 0.
- Usage auto-load on page mount skips accounts whose status is not `active`. `AccountUsageCell.vue` `shouldAutoLoadUsageOnMount` must return true only when `props.account.status === 'active'`. Non-active (`inactive`/`error`) accounts must not trigger automatic `/usage` upstream queries on mount; manual refresh via `usageManualRefreshToken` is unaffected.

## OpenAI Sticky Concurrency Spillover

This is a local scheduling customization introduced by commits `22f4263a0` (RED regression tests) and `adae405d0` (implementation). Preserve the behavior when merging upstream scheduler changes.

Behavior contract:

1. Honor a historical `session_hash -> account_id` binding only when the bound account acquires a real `account.Concurrency` slot.
2. If that slot is full, immediately continue through the normal same-group candidate selection and acquire another eligible account instead of returning the bound account's sticky wait plan.
3. Preserve the original Redis sticky binding when a single connection spills over, so later requests can return to the original account after capacity recovers.
4. If no eligible account has capacity, retain the bounded fallback `AccountWaitPlan` behavior.
5. Apply concurrency spillover in both OpenAI paths:
   - Legacy load-aware scheduling in `backend/internal/service/openai_gateway_scheduling.go`.
   - Advanced hard-sticky scheduling in `backend/internal/service/openai_account_scheduler.go`, even when `sticky_escape_enabled` is false for TTFT/error health escape.
6. Keep strict, non-movable `previous_response_id` response-chain affinity on its original account; moving it can break upstream continuation state.

Upgrade review anchors:

- `OpenAIGatewayService.selectAccountWithLoadAwareness` uses `preserveStickyBinding` after a full sticky-account slot and guards subsequent sticky writes.
- `defaultOpenAIAccountScheduler.selectBySessionHash` returns to load balancing whenever slot acquisition reports `Acquired=false`, without requiring the health-escape switch.
- `TestOpenAISelectAccountWithLoadAwareness_StickyFullSpillsToAvailableAccountAndPreservesBinding` covers the locally active legacy path.
- `TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyBusySpillsOverEvenWhenHealthEscapeDisabled` and `TestOpenAIGatewayService_SelectAccountWithScheduler_HealthEscapeDisabledStillSpillsOnConcurrency` cover the advanced path.

## Active Sticky-Session Reassignment

The account action menu provides an administrative redistribution tool for active OpenAI `session_hash` bindings. Preserve these boundaries when upstream changes account management, Redis keys, or scheduling:

1. Routes remain `GET /api/v1/admin/accounts/:id/sticky-sessions` and `POST /api/v1/admin/accounts/:id/sticky-sessions/reassign`.
2. The target must be OpenAI, active, schedulable, and in the selected group; source and target must share platform and group.
3. The UI defaults to 5 minutes and offers exactly 1, 5, 15, and 60 minute windows. It shows recent/all counts and up to 100 anonymized recent session suffixes.
4. Recency is `configured sticky TTL - Redis PTTL`, using `gateway.openai_ws.sticky_session_ttl_seconds`. The backend rechecks the selected window and moves the most recently active candidates first.
5. Scan only the validated `sticky_session:<group>:<platform>:*` namespace. Move only 16-character lowercase-hex current session keys; ignore 64-character compatibility copies.
6. Never move `response:` / `previous_response_id` continuation bindings. Count them separately for the operator.
7. Redis changes use compare-and-set against the source account plus `SET ... KEEPTTL`. A race must not overwrite a newer assignment or extend session lifetime.
8. A single operation accepts 1 through 100 bindings. It affects subsequent requests and does not interrupt an in-flight upstream request.

The full restore contract is in `.codex/skills/sub2api-account-modal-enhancer/references/sticky-session-reassignment.md`. The skill's `apply.sh` is a read-only audit and must never be changed back into a broad `sed -i` source rewriter.

## Verification

```bash
git log --reverse --oneline origin/main..HEAD
git diff --stat origin/main...HEAD

cd /home/third_party/sub2api/backend
go test -tags unit ./internal/handler/dto ./internal/service \
  -run 'TestRedactCredentials|TestAccountFromServiceShallow|TestMergePreservingSensitiveCreds|TestIsSensitiveCredentialKey'

go test ./internal/service \
  -run 'TestOpenAISelectAccountWithLoadAwareness_StickyFullSpillsToAvailableAccountAndPreservesBinding|TestOpenAIGatewayService_SelectAccountWithScheduler_(SessionStickyBusySpillsOverEvenWhenHealthEscapeDisabled|HealthEscapeDisabledStillSpillsOnConcurrency)'

go test ./internal/repository ./internal/handler/admin ./internal/service -count=1

cd /home/third_party/sub2api/frontend
pnpm vitest run \
  src/components/account/__tests__/EditAccountModal.spec.ts \
  src/components/account/__tests__/AccountUsageCell.spec.ts \
  src/components/account/__tests__/CreateAccountModal.spec.ts \
  src/components/admin/account/__tests__/StickySessionReassignModal.spec.ts \
  src/components/admin/account/__tests__/AccountActionMenu.spark_shadow.spec.ts \
  src/components/common/__tests__/DataTable.spec.ts \
  src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
pnpm typecheck
pnpm build
```

Build the deployable binary with `CGO_ENABLED=0 go build -tags embed`, restart only `sub2api.service`, and verify port 18381 plus the embedded frontend assets.

After every test, build, deployment, live version, and behavior check succeeds, delete only this upgrade's temporary recovery data:

```bash
case "$BACKUP" in
  /home/third_party/upgrade-backups/sub2api-*) ;;
  *) echo "FATAL: refusing to delete unexpected backup path: $BACKUP"; exit 1 ;;
esac
git branch -d "$BACKUP_BRANCH" &&
  rm -rf -- "$BACKUP"
```

If any merge, restore, test, deployment, or live check fails, keep the backup directory and branch.
