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
- The service remains isolated on `sub2api.service`, port 18381, database `sub2api`, and Redis DB 0.

## Verification

```bash
git log --reverse --oneline origin/main..HEAD
git diff --stat origin/main...HEAD

cd /home/third_party/sub2api/backend
go test -tags unit ./internal/handler/dto ./internal/service \
  -run 'TestRedactCredentials|TestAccountFromServiceShallow|TestMergePreservingSensitiveCreds|TestIsSensitiveCredentialKey'

cd /home/third_party/sub2api/frontend
pnpm vitest run \
  src/components/account/__tests__/EditAccountModal.spec.ts \
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
