#!/usr/bin/env bash
# rebalance-aggressive.sh — Plan C: safe fixes + delete all sticky_session keys for the group.
#
# Usage:
#   bash rebalance-aggressive.sh --group G --platform P [--active-account ID] [--apply]
#
# Requires: sub2api-rebalance-safe to have been applied to the idle account in the group.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_FILE="$REPO_ROOT/deploy/data/config.yaml"

log()  { printf '\033[1;34m[rebalance-aggressive]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[rebalance-aggressive]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[rebalance-aggressive]\033[0m %s\n' "$*" >&2; }

# ---- arg parsing -------------------------------------------------------------
GROUP_ID=""
PLATFORM=""
ACTIVE_ID=""
MODE="dry-run"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --group)            GROUP_ID="$2"; shift 2 ;;
    --platform)         PLATFORM="$2"; shift 2 ;;
    --active-account)   ACTIVE_ID="$2"; shift 2 ;;
    --apply)            MODE="apply"; shift ;;
    *) err "unknown arg: $1"; exit 64 ;;
  esac
done

if [[ -z "$GROUP_ID" || -z "$PLATFORM" ]]; then
  err "Usage: $0 --group G --platform P [--active-account ID] [--apply]"
  exit 64
fi

# ---- DB connection -----------------------------------------------------------
if ! command -v psql >/dev/null; then err "psql not found"; exit 127; fi
if ! command -v redis-cli >/dev/null; then err "redis-cli not found"; exit 127; fi

DB_HOST="${SUB2API_DB_HOST:-127.0.0.1}"
DB_PORT="${SUB2API_DB_PORT:-13307}"
DB_USER="${SUB2API_DB_USER:-sub2api}"
DB_NAME="${SUB2API_DB_NAME:-sub2api}"

if [[ -f "$CONFIG_FILE" ]] && command -v python3 >/dev/null && python3 -c "import yaml" 2>/dev/null; then
  DB_PASS=$(python3 -c "import yaml; print(yaml.safe_load(open('$CONFIG_FILE'))['database'].get('password',''))")
  eval "$(python3 -c "
import yaml
cfg=yaml.safe_load(open('$CONFIG_FILE'))['database']
print(f'DB_HOST={cfg.get(\"host\",\"127.0.0.1\")}')
print(f'DB_PORT={cfg.get(\"port\",13307)}')
print(f'DB_USER={cfg.get(\"user\",\"sub2api\")}')
print(f'DB_NAME={cfg.get(\"dbname\",\"sub2api\")}')
")"
fi
export PGPASSWORD="${SUB2API_DB_PASS:-$DB_PASS}"
PSQL=(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -X -q -t -A)

REDIS_HOST="${SUB2API_REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${SUB2API_REDIS_PORT:-6379}"
REDIS="redis-cli -h $REDIS_HOST -p $REDIS_PORT"

# ---- preflight: enumerate accounts in this group -----------------------------
log "preflight: enumerating accounts in group $GROUP_ID (platform=$PLATFORM)"
ACCT_ROWS=$("${PSQL[@]}" -c "SELECT a.id || '|' || a.name || '|' ||
                                     COALESCE(a.load_factor::text,'NULL') || '|' ||
                                     COALESCE(a.credentials->>'base_url','') || '|' ||
                                     a.type || '|' || a.platform || '|' ||
                                     (a.schedulable::text) || '|' || a.status
                              FROM account_groups ag
                              JOIN accounts a ON a.id = ag.account_id
                              WHERE ag.group_id = $GROUP_ID
                                AND a.platform = '$PLATFORM'
                                AND a.deleted_at IS NULL;")

if [[ -z "$ACCT_ROWS" ]]; then
  err "no accounts found in group $GROUP_ID with platform=$PLATFORM"
  exit 2
fi

log "accounts in scope:"
IDLE_OK=false
while IFS='|' read -r ID NAME LF BU TYPE PLAT SCHED STATUS; do
  log "  id=$ID name=$NAME type=$TYPE platform=$PLAT schedulable=$SCHED status=$STATUS load_factor=$LF base_url=$BU"
  if [[ "$TYPE" == "apikey" && "$LF" == "NULL" && ( "$BU" == */v1 || "$BU" == */v1/ ) ]]; then
    IDLE_OK=true
  fi
done <<<"$ACCT_ROWS"

if ! $IDLE_OK; then
  err "no account in this group has load_factor=NULL and base_url ending in /v1."
  err "run sub2api-rebalance-safe on the idle account first."
  exit 3
fi

# ---- enumerate Redis keys ----------------------------------------------------
log "enumerating Redis keys to be touched..."

STICKY_KEYS=$($REDIS KEYS "sticky_session:${GROUP_ID}:${PLATFORM}:*" 2>/dev/null || true)
STICKY_COUNT=0
if [[ -n "$STICKY_KEYS" ]]; then STICKY_COUNT=$(echo "$STICKY_KEYS" | wc -l); fi
log "  sticky_session:${GROUP_ID}:${PLATFORM}:*  →  $STICKY_COUNT keys"

# collect all account IDs in the group, then build the sched:meta/acc key list from that
GROUP_ACCT_IDS=$("${PSQL[@]}" -c "SELECT account_id FROM account_groups WHERE group_id = $GROUP_ID;")
META_KEYS=""
ACC_KEYS=""
if [[ -n "$GROUP_ACCT_IDS" ]]; then
  META_KEYS=$(while read -r aid; do
    [[ -n "$aid" ]] && $REDIS KEYS "sched:meta:${aid}" 2>/dev/null
  done <<<"$GROUP_ACCT_IDS")
  ACC_KEYS=$(while read -r aid; do
    [[ -n "$aid" ]] && $REDIS KEYS "sched:acc:${aid}" 2>/dev/null
  done <<<"$GROUP_ACCT_IDS")
fi
META_COUNT=0
if [[ -n "$META_KEYS" ]]; then META_COUNT=$(echo "$META_KEYS" | grep -c .); fi
log "  sched:meta:<acct-in-group>             →  $META_COUNT keys"
ACC_COUNT=0
if [[ -n "$ACC_KEYS" ]]; then ACC_COUNT=$(echo "$ACC_KEYS" | grep -c .); fi
log "  sched:acc:<acct-in-group>              →  $ACC_COUNT keys"

ACTIVE_KEYS=""
if [[ -n "$ACTIVE_ID" ]]; then
  ACTIVE_KEYS="concurrency:account:${ACTIVE_ID}
wait:account:${ACTIVE_ID}"
  log "  concurrency:account:${ACTIVE_ID}        →  1 key"
  log "  wait:account:${ACTIVE_ID}               →  1 key"
fi

TOTAL=$((STICKY_COUNT + META_COUNT + ACC_COUNT))
[[ -n "$ACTIVE_ID" ]] && TOTAL=$((TOTAL + 2))
log "total keys to be touched: $TOTAL"

if [[ "$MODE" != "apply" ]]; then
  log "DRY-RUN — pass --apply to execute. nothing will be deleted."
  exit 0
fi

# ---- apply -------------------------------------------------------------------
log "[1/3] re-applying safe fixes to all idle-eligible accounts in group"
while IFS='|' read -r ID NAME LF BU TYPE PLAT SCHED STATUS; do
  if [[ "$LF" != "NULL" ]]; then
    log "  fixing load_factor on account $ID ($NAME)"
    "${PSQL[@]}" -c "UPDATE accounts SET load_factor = NULL WHERE id = $ID;"
  fi
  if [[ -n "$BU" && "$BU" != */v1 && "$BU" != */v1/ ]]; then
    NEW_BU="${BU%/}/v1"
    log "  fixing base_url on account $ID ($NAME) → $NEW_BU"
    "${PSQL[@]}" -c "UPDATE accounts SET credentials = jsonb_set(credentials, '{base_url}', '\"$NEW_BU\"') WHERE id = $ID;"
  fi
done <<<"$ACCT_ROWS"

log "[2/3] deleting $TOTAL Redis keys"
DELETED=0
LOG_TRUNC=0

delete_keys() {
  local label="$1"; shift
  local keys="$1"
  [[ -z "$keys" ]] && return 0
  while IFS= read -r k; do
    [[ -z "$k" ]] && continue
    if (( LOG_TRUNC < 100 )); then
      log "  DEL $k   ($label)"
      LOG_TRUNC=$((LOG_TRUNC + 1))
    fi
    $REDIS DEL "$k" >/dev/null
    DELETED=$((DELETED + 1))
  done <<<"$keys"
}

delete_keys "sticky_session" "$STICKY_KEYS"
delete_keys "sched:meta"     "$META_KEYS"
delete_keys "sched:acc"      "$ACC_KEYS"
delete_keys "active"         "$ACTIVE_KEYS"

log "[3/3] done. deleted $DELETED keys."
log "next: monitor request distribution; expect rebind within 5 minutes."

log "post-fix verification:"
$REDIS EVAL "
  local k = redis.call('KEYS', 'sticky_session:${GROUP_ID}:${PLATFORM}:*')
  local dist = {}
  for i=1, #k do
    local v = redis.call('GET', k[i])
    dist[v] = (dist[v] or 0) + 1
  end
  return dist
" 0
