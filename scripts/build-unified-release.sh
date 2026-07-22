#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
PNPM9="/home/root/.npm/_npx/8959f4e966f464e2/node_modules/pnpm/bin/pnpm.cjs"
BUILD_OUT="$BACKEND_DIR/bin/sub2api-unified.new"
BUILD_TMP="$BUILD_OUT.tmp.$$"
SNAPSHOT_DIR="$(mktemp -d /data/cargo-target/sub2api-unified-source.XXXXXX)"
SNAPSHOT_BACKEND_DIR="$SNAPSHOT_DIR/backend"
SNAPSHOT_FRONTEND_DIR="$SNAPSHOT_DIR/frontend"
SNAPSHOT_BUILD_OUT="$SNAPSHOT_BACKEND_DIR/bin/sub2api-unified.new"

source_tree_sha() {
  local root="$1"
  (
    cd "$root"
    find backend frontend scripts docs/legal \
    \( -path 'backend/bin' -o \
       -path 'backend/internal/web/dist' -o \
       -path 'frontend/node_modules' -o \
       -path 'frontend/dist' -o \
       -path 'frontend/coverage' -o \
       -name '*.tsbuildinfo' \) -prune -o \
    -type f -print0 |
    LC_ALL=C sort -z |
    xargs -0 sha256sum |
    sha256sum |
    awk '{print $1}'
  )
}

cleanup() {
  rm -f -- "$BUILD_TMP"
  rm -rf -- "$SNAPSHOT_DIR"
}
trap cleanup EXIT

test -f "$PNPM9"
bash "$ROOT_DIR/scripts/remove-readme-sponsors.sh" --check
git -C "$ROOT_DIR" diff --check
SOURCE_SHA_BEFORE="$(source_tree_sha "$ROOT_DIR")"

# Freeze one coherent dirty-worktree snapshot. The before/after hashes reject a
# concurrent write during the short copy window; later edits belong to the next
# deployment and cannot mix with this artifact.
rsync -a \
  --exclude '/bin/' \
  --exclude '/internal/web/dist/' \
  "$BACKEND_DIR/" "$SNAPSHOT_BACKEND_DIR/"
rsync -a \
  --exclude '/node_modules/' \
  --exclude '/dist/' \
  --exclude '/coverage/' \
  --exclude '*.tsbuildinfo' \
  "$FRONTEND_DIR/" "$SNAPSHOT_FRONTEND_DIR/"
rsync -a "$ROOT_DIR/scripts/" "$SNAPSHOT_DIR/scripts/"
mkdir -p "$SNAPSHOT_DIR/docs"
rsync -a "$ROOT_DIR/docs/legal/" "$SNAPSHOT_DIR/docs/legal/"
ln -s "$FRONTEND_DIR/node_modules" "$SNAPSHOT_FRONTEND_DIR/node_modules"

SOURCE_SHA_AFTER_COPY="$(source_tree_sha "$ROOT_DIR")"
SNAPSHOT_SOURCE_SHA="$(source_tree_sha "$SNAPSHOT_DIR")"
if [ "$SOURCE_SHA_BEFORE" != "$SOURCE_SHA_AFTER_COPY" ] || \
   [ "$SOURCE_SHA_BEFORE" != "$SNAPSHOT_SOURCE_SHA" ]; then
  echo 'ERROR: source changed while the deployment snapshot was being created' >&2
  exit 1
fi
echo "==> Frozen source snapshot sha256=$SNAPSHOT_SOURCE_SHA"

if grep -R -n -E '^(<<<<<<< |>>>>>>> |=======$)' --include='*.go' --include='*.ts' --include='*.js' --include='*.vue' --include='*.md' "$SNAPSHOT_BACKEND_DIR" "$SNAPSHOT_FRONTEND_DIR/src" "$SNAPSHOT_DIR/scripts"; then
  echo "ERROR: merge conflict markers found" >&2
  exit 1
fi

echo "==> Complete backend test suite"
(
  cd "$SNAPSHOT_BACKEND_DIR"
  go test -tags unit ./... -count=1
)

echo "==> Complete frontend test suite, typecheck, and production build"
(
  cd "$SNAPSHOT_FRONTEND_DIR"
  node "$PNPM9" vitest run
  node "$PNPM9" typecheck
  node "$PNPM9" exec vite build
)
test -s "$SNAPSHOT_BACKEND_DIR/internal/web/dist/index.html"
test "$(source_tree_sha "$SNAPSHOT_DIR")" = "$SNAPSHOT_SOURCE_SHA"

echo "==> Unified embedded binary"
VERSION="$($SNAPSHOT_BACKEND_DIR/scripts/resolve-version.sh)"
mkdir -p "$SNAPSHOT_BACKEND_DIR/bin" "$(dirname -- "$BUILD_OUT")"
(
  cd "$SNAPSHOT_BACKEND_DIR"
  CGO_ENABLED=0 go build \
    -tags embed \
    -trimpath \
    -ldflags="-s -w -X main.Version=$VERSION" \
    -o "$SNAPSHOT_BUILD_OUT" \
    ./cmd/server/
)

test "$(source_tree_sha "$SNAPSHOT_DIR")" = "$SNAPSHOT_SOURCE_SHA"
go version -m "$SNAPSHOT_BUILD_OUT" | grep -Eq 'build[[:space:]]+-tags=embed'
go version -m "$SNAPSHOT_BUILD_OUT" | grep -Eq 'build[[:space:]]+CGO_ENABLED=0'
install -m 0755 "$SNAPSHOT_BUILD_OUT" "$BUILD_TMP"
mv -f -- "$BUILD_TMP" "$BUILD_OUT"
sha256sum "$BUILD_OUT"

LIVE_SOURCE_SHA="$(source_tree_sha "$ROOT_DIR")"
if [ "$LIVE_SOURCE_SHA" != "$SNAPSHOT_SOURCE_SHA" ]; then
  echo "NOTICE: live source advanced during the build; deployed artifact remains snapshot=$SNAPSHOT_SOURCE_SHA"
fi
