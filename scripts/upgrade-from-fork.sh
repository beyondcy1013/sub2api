#!/usr/bin/env bash
set -euo pipefail

REPO="${1:?Usage: $0 <repo-path>}"
cd "$REPO"

TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/home/third_party/upgrade-backups/$(basename "$REPO")-fork-$TS
BACKUP_BRANCH=backup-fork-$TS
mkdir -p "$BACKUP"

echo "=== [1/7] Snapshot ==="
git branch "$BACKUP_BRANCH" HEAD
git bundle create "$BACKUP/repository.bundle" --all 2>/dev/null
git log --reverse --oneline upstream/main..HEAD > "$BACKUP/local-commits.txt"
git status --porcelain=v1 > "$BACKUP/status.txt"

echo "=== [2/7] Stash dirty changes ==="
STASH_CREATED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  git stash push -m "pre-fork-upgrade-$TS"
  STASH_CREATED=1
fi

echo "=== [3/7] Fetch upstream ==="
http_proxy="" https_proxy="" HTTP_PROXY="" HTTPS_PROXY="" all_proxy="" ALL_PROXY="" \
  git fetch upstream --no-tags

echo "=== [4/7] Merge upstream/main ==="
git merge --no-ff upstream/main || {
  echo "!!! Conflicts detected. Resolve manually."
  git diff --name-only --diff-filter=U
  exit 1
}

echo "=== [5/7] Restore stash ==="
if [ "$STASH_CREATED" -eq 1 ]; then
  git stash pop --index || {
    echo "!!! Stash pop conflict. Resolve manually."
    exit 1
  }
fi

echo "=== [6/7] Push to fork ==="
git push origin main || echo "WARNING: push to fork failed"

echo "=== [7/7] Done ==="
echo "Backup: $BACKUP  Branch: $BACKUP_BRANCH"
echo "Next: verify customizations, run tests, build, deploy."
