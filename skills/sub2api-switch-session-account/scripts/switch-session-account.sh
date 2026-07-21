#!/usr/bin/env bash
# switch-session-account.sh - move one Sub2API sticky session binding to a target account.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_FILE="$REPO_ROOT/deploy/data/config.yaml"

log()  { printf '\033[1;34m[switch-session-account]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[switch-session-account]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[switch-session-account]\033[0m %s\n' "$*" >&2; }

usage() {
  cat >&2 <<'USAGE'
Usage:
  switch-session-account.sh --group G --platform P --to-account ID [selector] [--expect-from ID] [--apply]

Selectors, choose exactly one:
  --key REDIS_KEY          Full sticky_session Redis key
  --session-hash HASH      Session hash portion of sticky_session:<group>:<platform>:<hash>
  --response-id ID         Response id portion of sticky_session:<group>:<platform>:response:<id>

Options:
  --expect-from ID         Refuse to write unless the current binding equals ID
  --apply                  Write Redis SET KEEPTTL; default is dry-run
  --allow-non-schedulable  Allow target account with schedulable=false
USAGE
}

GROUP_ID=""
PLATFORM=""
TO_ID=""
KEY=""
SESSION_HASH=""
RESPONSE_ID=""
EXPECT_FROM=""
MODE="dry-run"
ALLOW_NON_SCHEDULABLE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --group) GROUP_ID="$2"; shift 2 ;;
    --platform) PLATFORM="$2"; shift 2 ;;
    --to-account) TO_ID="$2"; shift 2 ;;
    --key) KEY="$2"; shift 2 ;;
    --session-hash) SESSION_HASH="$2"; shift 2 ;;
    --response-id) RESPONSE_ID="$2"; shift 2 ;;
    --expect-from) EXPECT_FROM="$2"; shift 2 ;;
    --apply) MODE="apply"; shift ;;
    --allow-non-schedulable) ALLOW_NON_SCHEDULABLE=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) err "unknown arg: $1"; usage; exit 64 ;;
  esac
done

selector_count=0
[[ -n "$KEY" ]] && selector_count=$((selector_count + 1))
[[ -n "$SESSION_HASH" ]] && selector_count=$((selector_count + 1))
[[ -n "$RESPONSE_ID" ]] && selector_count=$((selector_count + 1))

if [[ -z "$GROUP_ID" || -z "$PLATFORM" || -z "$TO_ID" || "$selector_count" -ne 1 ]]; then
  usage
  exit 64
fi

if [[ ! "$GROUP_ID" =~ ^[0-9]+$ || ! "$TO_ID" =~ ^[0-9]+$ ]]; then
  err "--group and --to-account must be numeric"
  exit 64
fi

if [[ -n "$EXPECT_FROM" && ! "$EXPECT_FROM" =~ ^[0-9]+$ ]]; then
  err "--expect-from must be numeric"
  exit 64
fi

if [[ -z "$KEY" ]]; then
  if [[ -n "$SESSION_HASH" ]]; then
    KEY="sticky_session:${GROUP_ID}:${PLATFORM}:${SESSION_HASH}"
  else
    KEY="sticky_session:${GROUP_ID}:${PLATFORM}:response:${RESPONSE_ID}"
  fi
fi

EXPECTED_PREFIX="sticky_session:${GROUP_ID}:${PLATFORM}:"
if [[ "$KEY" != "$EXPECTED_PREFIX"* ]]; then
  err "--key must start with '$EXPECTED_PREFIX'"
  exit 64
fi

if ! command -v psql >/dev/null; then err "psql not found"; exit 127; fi
if ! command -v redis-cli >/dev/null; then err "redis-cli not found"; exit 127; fi

DB_HOST="${SUB2API_DB_HOST:-127.0.0.1}"
DB_PORT="${SUB2API_DB_PORT:-13307}"
DB_USER="${SUB2API_DB_USER:-sub2api}"
DB_NAME="${SUB2API_DB_NAME:-sub2api}"
DB_PASS="${SUB2API_DB_PASS:-}"

if [[ -f "$CONFIG_FILE" ]] && command -v python3 >/dev/null && python3 -c "import yaml" 2>/dev/null; then
  eval "$(python3 - "$CONFIG_FILE" <<'PY'
import shlex
import sys
import yaml

cfg = yaml.safe_load(open(sys.argv[1]))["database"]
print(f"DB_HOST={shlex.quote(str(cfg.get('host', '127.0.0.1')))}")
print(f"DB_PORT={shlex.quote(str(cfg.get('port', 13307)))}")
print(f"DB_USER={shlex.quote(str(cfg.get('user', 'sub2api')))}")
print(f"DB_NAME={shlex.quote(str(cfg.get('dbname', 'sub2api')))}")
print(f"DB_PASS={shlex.quote(str(cfg.get('password', '')))}")
PY
)"
fi

export PGPASSWORD="${SUB2API_DB_PASS:-$DB_PASS}"
PSQL=(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -X -q -t -A)

REDIS_HOST="${SUB2API_REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${SUB2API_REDIS_PORT:-6379}"
REDIS=(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT")

log "preflight: checking target account $TO_ID in group $GROUP_ID"
TO_ROW=$("${PSQL[@]}" -v ON_ERROR_STOP=1 -c "
  SELECT a.id || '|' || a.name || '|' ||
         a.type || '|' || a.platform || '|' ||
         (a.schedulable::text) || '|' || a.status || '|' ||
         COALESCE(a.concurrency::text,'0')
  FROM accounts a
  JOIN account_groups ag ON ag.account_id = a.id
  WHERE a.id = $TO_ID
    AND ag.group_id = $GROUP_ID
    AND a.deleted_at IS NULL;
")

if [[ -z "$TO_ROW" ]]; then
  err "target account $TO_ID is not in group $GROUP_ID or is deleted"
  exit 2
fi

IFS='|' read -r T_ID T_NAME T_TYPE T_PLAT T_SCHED T_STATUS T_CONC <<<"$TO_ROW"
[[ "$T_SCHED" == "true" ]] && T_SCHED="t"

if [[ "$T_PLAT" != "$PLATFORM" || "$T_STATUS" != "active" ]]; then
  err "target account state invalid (platform=$T_PLAT status=$T_STATUS)"
  exit 3
fi

if [[ "$T_SCHED" != "t" && "$ALLOW_NON_SCHEDULABLE" != "true" ]]; then
  err "target account $TO_ID is schedulable=false; pass --allow-non-schedulable only if intentional"
  exit 3
fi

log "target: id=$T_ID name=$T_NAME type=$T_TYPE platform=$T_PLAT schedulable=$T_SCHED status=$T_STATUS concurrency=$T_CONC"

CURRENT=$("${REDIS[@]}" GET "$KEY" 2>/dev/null || true)
TTL=$("${REDIS[@]}" TTL "$KEY" 2>/dev/null || echo -2)

if [[ -z "$CURRENT" || "$TTL" == "-2" ]]; then
  err "sticky key does not exist or has no value: $KEY"
  exit 4
fi

log "binding: $KEY"
log "current account: $CURRENT"
log "target account : $TO_ID"
log "ttl            : $TTL"

if [[ -n "$EXPECT_FROM" && "$CURRENT" != "$EXPECT_FROM" ]]; then
  err "current account is $CURRENT, expected $EXPECT_FROM; refusing to write"
  exit 5
fi

if [[ "$CURRENT" == "$TO_ID" ]]; then
  log "already bound to target account; nothing to do"
  exit 0
fi

if [[ "$MODE" != "apply" ]]; then
  log "DRY-RUN - pass --apply to execute:"
  log "  SET $KEY $TO_ID KEEPTTL"
  exit 0
fi

warn "rewriting one sticky binding. If this is a response chain, previous_response_id may not exist on the target account."
"${REDIS[@]}" SET "$KEY" "$TO_ID" KEEPTTL >/dev/null

AFTER=$("${REDIS[@]}" GET "$KEY" 2>/dev/null || true)
if [[ "$AFTER" != "$TO_ID" ]]; then
  err "write verification failed: key now points to '$AFTER'"
  exit 6
fi

log "done. $KEY now points to account $TO_ID"
