#!/usr/bin/env bash
set -euo pipefail

CANONICAL_REPO="/home/third_party/sub2api"
LEGACY_REPO="/home/third_party/sub2freeApi"
REQUESTED_REPO="${1:-$CANONICAL_REPO}"

if [ "$(realpath "$REQUESTED_REPO")" != "$CANONICAL_REPO" ]; then
  echo "ERROR: $CANONICAL_REPO is the only source and merge repository" >&2
  exit 2
fi
if [ "$(git -C "$CANONICAL_REPO" branch --show-current)" != "main" ]; then
  echo "ERROR: canonical repository must be on main" >&2
  exit 2
fi
if git -C "$CANONICAL_REPO" rev-parse -q --verify MERGE_HEAD >/dev/null; then
  echo "ERROR: finish the current merge before starting another upgrade" >&2
  exit 2
fi

TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
CANONICAL_BACKUP="/home/third_party/upgrade-backups/sub2api-$TIMESTAMP"
STATE_FILE="$CANONICAL_BACKUP/upgrade-state.env"

snapshot_repo() {
  local repo="$1"
  local name backup branch

  name="$(basename "$repo")"
  backup="/home/third_party/upgrade-backups/${name}-${TIMESTAMP}"
  branch="backup-pre-upgrade-${TIMESTAMP}"
  mkdir -p "$backup"
  git -C "$repo" branch "$branch" HEAD
  git -C "$repo" bundle create "$backup/repository.bundle" --all
  git bundle verify "$backup/repository.bundle" >/dev/null
  git -C "$repo" log --reverse --oneline upstream/main..HEAD > "$backup/local-commits.txt"
  git -C "$repo" status --porcelain=v1 > "$backup/status.txt"
  git -C "$repo" diff --binary > "$backup/worktree.patch"
  git -C "$repo" diff --cached --binary > "$backup/index.patch"

  (
    cd "$repo"
    {
      git ls-files --others --exclude-standard -z
      for policy_file in AGENTS.md CLAUDE.md; do
        if [ -f "$policy_file" ]; then printf '%s\0' "$policy_file"; fi
      done
      for skill_dir in .codex/skills .claude/skills; do
        if [ -d "$skill_dir" ]; then find "$skill_dir" -type f -print0; fi
      done
    } | sort -zu > "$backup/local-files.list"
    if [ -s "$backup/local-files.list" ]; then
      tar --null -czf "$backup/local-files.tar.gz" -T "$backup/local-files.list"
    fi
  )

  printf 'snapshot repo=%s backup=%s branch=%s\n' "$repo" "$backup" "$branch"
}

record_runtime() {
  local main_pid free_pid
  main_pid="$(systemctl show -p MainPID --value sub2api.service)"
  free_pid="$(systemctl show -p MainPID --value sub2freeApi.service)"
  {
    printf 'sub2api_pid=%s\n' "$main_pid"
    printf 'sub2freeApi_pid=%s\n' "$free_pid"
    sha256sum \
      /home/third_party/bin/sub2api/sub2api \
      /home/third_party/bin/sub2freeApi/sub2freeApi \
      "/proc/$main_pid/exe" "/proc/$free_pid/exe"
    systemctl is-active sub2api.service
    systemctl is-active sub2freeApi.service
    ss -ltnp | rg ':18381|:18382'
  } > "$CANONICAL_BACKUP/runtime-baseline.txt"
}

echo "=== [1/6] Snapshot canonical and legacy recovery repositories ==="
snapshot_repo "$CANONICAL_REPO"
snapshot_repo "$LEGACY_REPO"
record_runtime

echo "=== [2/6] Save canonical dirty worktree ==="
STASH_CREATED=0
STASH_OID=""
if [ -n "$(git -C "$CANONICAL_REPO" status --porcelain=v1 --untracked-files=all)" ]; then
  git -C "$CANONICAL_REPO" stash push --include-untracked -m "pre-upgrade-$TIMESTAMP"
  STASH_CREATED=1
  STASH_OID="$(git -C "$CANONICAL_REPO" rev-parse refs/stash)"
fi
{
  printf 'timestamp=%s\n' "$TIMESTAMP"
  printf 'canonical_backup=%s\n' "$CANONICAL_BACKUP"
  printf 'stash_created=%s\n' "$STASH_CREATED"
  printf 'stash_oid=%s\n' "$STASH_OID"
} > "$STATE_FILE"

echo "=== [3/6] Fetch upstream/main and tags ==="
http_proxy="http://127.0.0.1:7890" \
https_proxy="http://127.0.0.1:7890" \
HTTP_PROXY="http://127.0.0.1:7890" \
HTTPS_PROXY="http://127.0.0.1:7890" \
all_proxy="" ALL_PROXY="" \
  git -C "$CANONICAL_REPO" fetch upstream --tags

CURRENT_VERSION="$(git -C "$CANONICAL_REPO" show HEAD:backend/cmd/server/VERSION)"
UPSTREAM_VERSION="$(git -C "$CANONICAL_REPO" show upstream/main:backend/cmd/server/VERSION)"
printf 'version current=%s upstream=%s\n' "$CURRENT_VERSION" "$UPSTREAM_VERSION"

echo "=== [4/6] Merge upstream without committing ==="
if ! git -C "$CANONICAL_REPO" merge --no-ff --no-commit upstream/main; then
  echo "Resolve each conflict by combining upstream and local behavior." >&2
  echo "Do not restore the saved worktree until the merge is committed." >&2
  echo "After conflicts: bash scripts/remove-readme-sponsors.sh" >&2
  git -C "$CANONICAL_REPO" diff --name-only --diff-filter=U >&2
  exit 3
fi

echo "=== [5/6] Reapply README advertisement policy ==="
bash "$CANONICAL_REPO/scripts/remove-readme-sponsors.sh"
bash "$CANONICAL_REPO/scripts/remove-readme-sponsors.sh" --check

echo "=== [6/6] Hand off for review ==="
echo "State: $STATE_FILE"
echo "Inspect and commit the upstream merge before restoring saved work."
if [ "$STASH_CREATED" -eq 1 ]; then
  echo "Saved worktree OID: $STASH_OID"
  echo "After the merge commit, locate that OID in git stash list and pop it with --index."
fi
echo "Run complete verification and unified deployment before pushing origin/main."
