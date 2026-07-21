#!/usr/bin/env bash
# rebalance-redirect.sh — Plan D: rewrite sticky_session values from one account to another.
#
# Usage:
#   bash rebalance-redirect.sh --group G --platform P --from-account FA --to-account TA [--only-from ID] [--batch-size N] [--batch-delay D] [--apply]
#
# Requires: sub2api-rebalance-safe to have been applied to the target account.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_FILE="$REPO_ROOT/deploy/data/config.yaml"

log()  { printf '\033[1;34m[rebalance-redirect]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[rebalance-redirect]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[rebalance-redirect]\033[0m %s\n' "$*" >&2; }

# ---- arg parsing -------------------------------------------------------------
GROUP_ID=""
PLATFORM=""
FROM_ID=""
TO_ID=""
ONLY_FROM=""
COUNT=0
BATCH_SIZE=100
BATCH_DELAY="0s"
ACTIVE_SOURCE=false
RANDOM_PICK=false
MODE="dry-run"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --group)          GROUP_ID="$2"; shift 2 ;;
    --platform)       PLATFORM="$2"; shift 2 ;;
    --from-account)   FROM_ID="$2"; shift 2 ;;
    --to-account)     TO_ID="$2"; shift 2 ;;
    --only-from)      ONLY_FROM="$2"; shift 2 ;;
    --count)          COUNT="$2"; shift 2 ;;
    --batch-size)     BATCH_SIZE="$2"; shift 2 ;;
    --batch-delay)    BATCH_DELAY="$2"; shift 2 ;;
    --active-source)  ACTIVE_SOURCE=true; shift ;;
    --random)         RANDOM_PICK=true; shift ;;
    --apply)          MODE="apply"; shift ;;
    *) err "unknown arg: $1"; exit 64 ;;
  esac
done

if [[ -z "$GROUP_ID" || -z "$PLATFORM" || -z "$FROM_ID" || -z "$TO_ID" ]]; then
  err "Usage: $0 --group G --platform P --from-account FA --to-account TA [--only-from ID] [--batch-size N] [--batch-delay D] [--apply]"
  exit 64
fi

if [[ "$FROM_ID" == "$TO_ID" ]]; then
  err "--from-account and --to-account must be different (got $FROM_ID)"
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

# ---- preflight ---------------------------------------------------------------
log "preflight: checking target account $TO_ID"
TO_ROW=$("${PSQL[@]}" -c "SELECT id || '|' || name || '|' ||
                                 COALESCE(load_factor::text,'NULL') || '|' ||
                                 COALESCE(credentials->>'base_url','') || '|' ||
                                 a.type || '|' || a.platform || '|' ||
                                 (a.schedulable::text) || '|' || a.status || '|' ||
                                 COALESCE(a.concurrency::text,'0')
                          FROM accounts a
                          JOIN account_groups ag ON ag.account_id = a.id
                          WHERE a.id = $TO_ID AND ag.group_id = $GROUP_ID;")
if [[ -z "$TO_ROW" ]]; then
  err "target account $TO_ID is not in group $GROUP_ID"
  exit 2
fi
IFS='|' read -r T_ID T_NAME T_LF T_BU T_TYPE T_PLAT T_SCHED T_STATUS T_CONC <<<"$TO_ROW"
[[ "$T_SCHED" == "true" ]] && T_SCHED="t"

if [[ "$T_TYPE" != "apikey" || "$T_PLAT" != "$PLATFORM" || "$T_SCHED" != "t" || "$T_STATUS" != "active" ]]; then
  err "target account state invalid (type=$T_TYPE platform=$T_PLAT schedulable=$T_SCHED status=$T_STATUS)"
  exit 3
fi
if [[ "$T_LF" != "NULL" ]]; then
  err "target account $TO_ID has load_factor=$T_LF; run sub2api-rebalance-safe first"
  exit 3
fi
if [[ -n "$T_BU" && "$T_BU" != */v1 && "$T_BU" != */v1/ ]]; then
  err "target account $TO_ID has base_url='$T_BU' (not ending in /v1); run sub2api-rebalance-safe first"
  exit 3
fi

# ---- enumerate keys ----------------------------------------------------------
log "enumerating sticky_session:${GROUP_ID}:${PLATFORM}:* keys"
ALL_KEYS=$($REDIS KEYS "sticky_session:${GROUP_ID}:${PLATFORM}:*" 2>/dev/null || true)
if [[ -z "$ALL_KEYS" ]]; then
  log "no keys to redirect; nothing to do"
  exit 0
fi
ALL_COUNT=$(echo "$ALL_KEYS" | wc -l)
log "  found $ALL_COUNT keys"

# --active-source guard: refuse if the source account currently has no live load
if $ACTIVE_SOURCE; then
  log "--active-source: checking live load on account $FROM_ID"
  if [[ -z "$FROM_ID" ]]; then
    err "--active-source requires --from-account (to know which account to check)"
    exit 64
  fi
  INFL=$($REDIS ZCARD "concurrency:account:${FROM_ID}" 2>/dev/null || echo 0)
  WAIT=$($REDIS GET "wait:account:${FROM_ID}" 2>/dev/null || echo 0)
  [[ -z "$INFL" ]] && INFL=0
  [[ -z "$WAIT" ]] && WAIT=0
  log "  in-flight on account $FROM_ID : $INFL"
  log "  wait queue on account $FROM_ID : $WAIT"
  if (( INFL == 0 )) && (( WAIT == 0 )); then
    err "account $FROM_ID has no in-flight and no waiting requests; --active-source refuses to run on a quiet account"
    exit 4
  fi
fi

# build the set of keys whose current value matches the filter
log "fetching current value and TTL of each key (Redis round-trip per key)..."
MATCH_KEYS=""
MATCH_EXISTING=0
MATCH_ALREADY=0
LOG_TRUNC=0
TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

# store: key TAB ttl
while IFS= read -r k; do
  [[ -z "$k" ]] && continue
  cur=$($REDIS GET "$k" 2>/dev/null || echo "")
  if [[ -z "$cur" ]]; then continue; fi
  if [[ -n "$ONLY_FROM" && "$cur" != "$ONLY_FROM" ]]; then continue; fi
  if [[ "$cur" == "$TO_ID" ]]; then
    MATCH_ALREADY=$((MATCH_ALREADY + 1))
    continue
  fi
  MATCH_EXISTING=$((MATCH_EXISTING + 1))
  ttl=$($REDIS TTL "$k" 2>/dev/null || echo -1)
  # treat -1 (no expiry) and -2 (no key — shouldn't happen) as "most recent"
  [[ "$ttl" == "-1" || "$ttl" == "-2" || -z "$ttl" ]] && ttl=999999999
  printf '%s\t%s\n' "$ttl" "$k" >> "$TMPFILE"
done <<<"$ALL_KEYS"

if [[ "$MATCH_EXISTING" -eq 0 ]]; then
  log "no keys need rewriting ($MATCH_ALREADY already point to $TO_ID)"
  exit 0
fi

# sort by TTL desc (most recent first); keep only the key column
if $RANDOM_PICK; then
  log "sorting: random shuffle of $MATCH_EXISTING keys"
  SORTED=$(awk -F'\t' '{print $2}' "$TMPFILE" | shuf)
else
  log "sorting: by TTL descending (most recent first)"
  SORTED=$(sort -t$'\t' -k1,1 -nr "$TMPFILE" | awk -F'\t' '{print $2}')
fi

# apply --count: keep only the first N
if [[ "$COUNT" -gt 0 && "$MATCH_EXISTING" -gt "$COUNT" ]]; then
  log "--count $COUNT: keeping the $COUNT most-recent of $MATCH_EXISTING matching keys"
  SORTED=$(echo "$SORTED" | head -n "$COUNT")
  MATCH_EXISTING="$COUNT"
fi
echo "$SORTED" > "$TMPFILE"

# concurrency warning
if [[ "$BATCH_SIZE" == "0" || "$BATCH_SIZE" -ge "$MATCH_EXISTING" ]]; then
  effective_batch="$MATCH_EXISTING"
else
  effective_batch="$BATCH_SIZE"
fi
if [[ "$effective_batch" -gt "${T_CONC:-0}" && "${T_CONC:-0}" -gt 0 ]]; then
  warn "target account concurrency=${T_CONC} but first batch will be $effective_batch"
  warn "this can saturate the target. consider --batch-size ${T_CONC} --batch-delay 30s"
  if [[ "$MODE" == "apply" ]]; then
    read -r -p "press enter to continue, ctrl-c to abort: "
  fi
fi

log "summary:"
log "  total keys in group       : $ALL_COUNT"
log "  matching the filter       : $MATCH_EXISTING"
log "  already pointing at $TO_ID : $MATCH_ALREADY"
log "  to be rewritten           : $MATCH_EXISTING"
log "  selection order           : $( $RANDOM_PICK && echo random || echo 'TTL desc (most recent first)' )"
log "  batch size                : $BATCH_SIZE"
log "  batch delay               : $BATCH_DELAY"

if [[ "$MODE" != "apply" ]]; then
  log "DRY-RUN — pass --apply to execute. nothing will be written."
  log "first 10 keys that would be rewritten (in selection order):"
  head -10 "$TMPFILE" | while IFS= read -r k; do log "  SET $k = $TO_ID"; done
  exit 0
fi

# ---- apply -------------------------------------------------------------------
log "[1/3] rewriting bindings in batches"
WRITTEN=0
BATCH=0
BATCH_FIRST=1

while IFS= read -r k; do
  [[ -z "$k" ]] && continue

  if [[ "$BATCH_FIRST" -eq 1 ]]; then
    BATCH=$((BATCH + 1))
    log "  batch $BATCH starting at key $k"
    BATCH_FIRST=0
  fi

  if (( LOG_TRUNC < 100 )); then
    log "  SET $k = $TO_ID"
    LOG_TRUNC=$((LOG_TRUNC + 1))
  fi

  # preserve TTL: use SET KEEPTTL (Redis >= 6.0)
  $REDIS SET "$k" "$TO_ID" KEEPTTL >/dev/null
  WRITTEN=$((WRITTEN + 1))

  # batch boundary
  if [[ "$BATCH_SIZE" != "0" ]] && (( WRITTEN % BATCH_SIZE == 0 )); then
    log "  batch $BATCH complete: $WRITTEN / $MATCH_EXISTING rewritten"
    BATCH_FIRST=1
    if [[ "$BATCH_DELAY" != "0s" && "$BATCH_DELAY" != "0" ]]; then
      secs=$(parse_duration_seconds "$BATCH_DELAY")
      log "  sleeping $BATCH_DELAY ($secs s) before next batch…"
      sleep "$secs"
    fi
  fi
done < "$TMPFILE"

log "[2/3] final distribution:"
$REDIS EVAL "
  local k = redis.call('KEYS', 'sticky_session:${GROUP_ID}:${PLATFORM}:*')
  local dist = {}
  for i=1, #k do
    local v = redis.call('GET', k[i])
    dist[v] = (dist[v] or 0) + 1
  end
  return dist
" 0

log "[3/3] done. wrote $WRITTEN keys (skipped $MATCH_ALREADY that were already at target)."
log "next: monitor wait:account:$TO_ID and usage_logs for the next few minutes."

# ---- helpers -----------------------------------------------------------------
parse_duration_seconds() {
  local s="$1"
  if [[ "$s" =~ ^([0-9]+)s$ ]]; then echo "${BASH_REMATCH[1]}"; return; fi
  if [[ "$s" =~ ^([0-9]+)m$ ]]; then echo $(( ${BASH_REMATCH[1]} * 60 )); return; fi
  if [[ "$s" =~ ^([0-9]+)h$ ]]; then echo $(( ${BASH_REMATCH[1]} * 3600 )); return; fi
  if [[ "$s" =~ ^[0-9]+$ ]]; then echo "$s"; return; fi
  err "bad duration: $s (use 30s, 5m, 1h)"; exit 64
}
