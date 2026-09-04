#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
PNPM9="/home/root/.npm/_npx/8959f4e966f464e2/node_modules/pnpm/bin/pnpm.cjs"
PNPM_CMD=()
BUILD_OUT="$BACKEND_DIR/bin/sub2api-unified.new"
BUILD_TMP="$BUILD_OUT.tmp.$$"
BUILD_MODE="${SUB2API_BUILD_MODE:-quick}"
MODE_FLAG_SEEN=false
PRINT_PLAN=false
SNAPSHOT_DIR=""
SNAPSHOT_BACKEND_DIR=""
SNAPSHOT_FRONTEND_DIR=""
SNAPSHOT_BUILD_OUT=""
SNAPSHOT_LOCK="/data/cargo-target/sub2api-unified-source.lock"
GO_TEST_TMPDIR="/data/cargo-target/sub2api-unified-go-test-tmp"

usage() {
  cat <<'EOF'
Usage: bash scripts/build-unified-release.sh [--quick|--full] [--print-plan]

  --quick       Default. Reuse Go test cache and skip the duplicate full Vitest suite.
  --full        Force uncached backend tests and the complete frontend Vitest suite.
  --print-plan  Print the resolved verification plan without creating a snapshot or building.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --quick|--full)
      if [ "$MODE_FLAG_SEEN" = true ]; then
        echo 'only one of --quick or --full may be specified' >&2
        exit 2
      fi
      BUILD_MODE="${1#--}"
      MODE_FLAG_SEEN=true
      shift
      ;;
    --print-plan)
      PRINT_PLAN=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$BUILD_MODE" in
  quick|full) ;;
  *)
    echo "SUB2API_BUILD_MODE must be quick or full, got: $BUILD_MODE" >&2
    exit 2
    ;;
esac

print_plan() {
  printf 'mode=%s\n' "$BUILD_MODE"
  if [ "$BUILD_MODE" = full ]; then
    printf '%s\n' 'backend_tests=all_uncached' 'frontend_tests=full_suite'
  else
    printf '%s\n' 'backend_tests=all_cached' 'frontend_tests=skip_full_suite'
  fi
  printf '%s\n' \
    'typecheck=required' \
    'frontend_build=required' \
    'go_embed_build=required' \
    'snapshot_workspace=stable_locked' \
    'go_test_tmp=stable_locked'
}

if [ "$PRINT_PLAN" = true ]; then
  print_plan
  exit 0
fi

SNAPSHOT_DIR="/data/cargo-target/sub2api-unified-source"
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
}
trap cleanup EXIT

run_stage() {
  local label="$1"
  shift
  local started_at started_epoch finished_epoch elapsed
  started_at="$(date '+%Y-%m-%d %H:%M:%S')"
  started_epoch="$(date +%s)"
  echo "==> START $label at $started_at"
  "$@"
  finished_epoch="$(date +%s)"
  elapsed=$((finished_epoch - started_epoch))
  echo "==> DONE  $label elapsed=${elapsed}s"
}

verify_backend() {
  cd "$SNAPSHOT_BACKEND_DIR"
  mkdir -p "$GO_TEST_TMPDIR"
  echo "==> Go test cache=$(go env GOCACHE) tmp=$GO_TEST_TMPDIR"
  if [ "$BUILD_MODE" = full ]; then
    TMPDIR="$GO_TEST_TMPDIR" go test -tags unit ./... -count=1
  else
    TMPDIR="$GO_TEST_TMPDIR" go test -tags unit ./...
  fi
}

verify_frontend_tests() {
  cd "$SNAPSHOT_FRONTEND_DIR"
  "${PNPM_CMD[@]}" vitest run
}

verify_frontend_types() {
  cd "$SNAPSHOT_FRONTEND_DIR"
  "${PNPM_CMD[@]}" typecheck
}

build_frontend() {
  cd "$SNAPSHOT_FRONTEND_DIR"
  "${PNPM_CMD[@]}" exec vite build
}

build_embedded_binary() {
  local version="$1"
  cd "$SNAPSHOT_BACKEND_DIR"
  CGO_ENABLED=0 go build \
    -tags embed \
    -trimpath \
    -ldflags="-s -w -X main.Version=$version" \
    -o "$SNAPSHOT_BUILD_OUT" \
    ./cmd/server/
}

if [ -f "$PNPM9" ]; then
  PNPM_CMD=(node "$PNPM9")
elif command -v corepack >/dev/null 2>&1; then
  PNPM_CMD=(corepack pnpm)
else
  echo "ERROR: pnpm 9 is required but neither $PNPM9 nor corepack is available" >&2
  exit 1
fi
PNPM_VERSION="$("${PNPM_CMD[@]}" --version)"
if [[ "$PNPM_VERSION" != 9.* ]]; then
  echo "ERROR: pnpm 9 is required, got $PNPM_VERSION from ${PNPM_CMD[*]}" >&2
  exit 1
fi
echo "==> pnpm=$PNPM_VERSION command=${PNPM_CMD[*]}"
mkdir -p "$(dirname -- "$SNAPSHOT_LOCK")"
exec 9>"$SNAPSHOT_LOCK"
if ! flock -n 9; then
  echo "ERROR: another unified build owns snapshot workspace $SNAPSHOT_DIR" >&2
  exit 1
fi
mkdir -p "$SNAPSHOT_BACKEND_DIR" "$SNAPSHOT_FRONTEND_DIR" "$SNAPSHOT_DIR/scripts" "$SNAPSHOT_DIR/docs/legal"
bash "$ROOT_DIR/scripts/remove-readme-sponsors.sh" --check
git -C "$ROOT_DIR" diff --check
SOURCE_SHA_BEFORE="$(source_tree_sha "$ROOT_DIR")"

# Freeze one coherent dirty-worktree snapshot. The before/after hashes reject a
# concurrent write during the short copy window; later edits belong to the next
# deployment and cannot mix with this artifact.
rsync -a \
  --delete \
  --exclude '/bin/' \
  --exclude '/internal/web/dist/' \
  "$BACKEND_DIR/" "$SNAPSHOT_BACKEND_DIR/"
rsync -a \
  --delete \
  --exclude '/node_modules/' \
  --exclude '/dist/' \
  --exclude '/coverage/' \
  --exclude '*.tsbuildinfo' \
  "$FRONTEND_DIR/" "$SNAPSHOT_FRONTEND_DIR/"
rsync -a --delete "$ROOT_DIR/scripts/" "$SNAPSHOT_DIR/scripts/"
rsync -a --delete "$ROOT_DIR/docs/legal/" "$SNAPSHOT_DIR/docs/legal/"
if [ -e "$SNAPSHOT_FRONTEND_DIR/node_modules" ] && [ ! -L "$SNAPSHOT_FRONTEND_DIR/node_modules" ]; then
  echo "ERROR: snapshot node_modules path is not the expected symlink" >&2
  exit 1
fi
ln -sfn "$FRONTEND_DIR/node_modules" "$SNAPSHOT_FRONTEND_DIR/node_modules"

SOURCE_SHA_AFTER_COPY="$(source_tree_sha "$ROOT_DIR")"
SNAPSHOT_SOURCE_SHA="$(source_tree_sha "$SNAPSHOT_DIR")"
if [ "$SOURCE_SHA_BEFORE" != "$SOURCE_SHA_AFTER_COPY" ] || \
   [ "$SOURCE_SHA_BEFORE" != "$SNAPSHOT_SOURCE_SHA" ]; then
  echo 'ERROR: source changed while the deployment snapshot was being created' >&2
  exit 1
fi
echo "==> Frozen source snapshot sha256=$SNAPSHOT_SOURCE_SHA"
echo "==> Build verification mode=$BUILD_MODE"
print_plan

if grep -R -n -E '^(<<<<<<< |>>>>>>> |=======$)' --include='*.go' --include='*.ts' --include='*.js' --include='*.vue' --include='*.md' "$SNAPSHOT_BACKEND_DIR" "$SNAPSHOT_FRONTEND_DIR/src" "$SNAPSHOT_DIR/scripts"; then
  echo "ERROR: merge conflict markers found" >&2
  exit 1
fi

run_stage "backend test suite ($BUILD_MODE)" verify_backend
if [ "$BUILD_MODE" = full ]; then
  run_stage 'frontend full test suite' verify_frontend_tests
else
  echo '==> SKIP frontend full test suite in quick mode; task-focused tests are required before queueing'
fi
run_stage 'frontend typecheck' verify_frontend_types
run_stage 'frontend production build' build_frontend
test -s "$SNAPSHOT_BACKEND_DIR/internal/web/dist/index.html"
test "$(source_tree_sha "$SNAPSHOT_DIR")" = "$SNAPSHOT_SOURCE_SHA"

echo "==> Unified embedded binary"
VERSION="$($SNAPSHOT_BACKEND_DIR/scripts/resolve-version.sh)"
mkdir -p "$SNAPSHOT_BACKEND_DIR/bin" "$(dirname -- "$BUILD_OUT")"
run_stage 'unified embedded Go binary' build_embedded_binary "$VERSION"

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
