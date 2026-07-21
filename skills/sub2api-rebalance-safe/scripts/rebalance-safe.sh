#!/usr/bin/env bash
# rebalance-safe.sh — Plan A: fix load_factor + base_url, let traffic drift naturally.
#
# Usage:
#   bash rebalance-safe.sh <account_id>                # dry-run (default)
#   bash rebalance-safe.sh <account_id> --apply        # actually write
#   bash rebalance-safe.sh <account_id> --rollback     # restore from .bak file
#
# Environment overrides (otherwise read from deploy/data/config.yaml):
#   SUB2API_DB_HOST  (default 127.0.0.1)
#   SUB2API_DB_PORT  (default 13307)
#   SUB2API_DB_USER  (default sub2api)
#   SUB2API_DB_NAME  (default sub2api)
#   SUB2API_DB_PASS  (default empty; if empty, read from config.yaml)
#   SUB2API_REDIS_HOST  (default 127.0.0.1)
#   SUB2API_REDIS_PORT  (default 6379)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_FILE="$REPO_ROOT/deploy/data/config.yaml"

log()  { printf '\033[1;34m[rebalance-safe]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[rebalance-safe]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[rebalance-safe]\033[0m %s\n' "$*" >&2; }

# ---- parse args --------------------------------------------------------------
if [[ $# -lt 1 ]]; then
  err "Usage: $0 <account_id> [--apply|--rollback]"
  exit 64
fi
ACCOUNT_ID="$1"
MODE="${2:-dry-run}"

# ---- resolve DB connection ---------------------------------------------------
resolve_db_from_config() {
  python3 - <<PY
import sys, yaml
with open("$CONFIG_FILE") as f:
    cfg = yaml.safe_load(f)
db = cfg.get("database", {})
print(db.get("host", "127.0.0.1"))
print(db.get("port", 13307))
print(db.get("user", "sub2api"))
print(db.get("dbname", "sub2api"))
print(db.get("password", ""))
PY
}

if [[ -z "${SUB2API_DB_HOST:-}" ]]; then
  if [[ -f "$CONFIG_FILE" ]] && command -v python3 >/dev/null && python3 -c "import yaml" 2>/dev/null; then
    mapfile -t CFG < <(resolve_db_from_config)
    DB_HOST="${SUB2API_DB_HOST:-${CFG[0]}}"
    DB_PORT="${SUB2API_DB_PORT:-${CFG[1]}}"
    DB_USER="${SUB2API_DB_USER:-${CFG[2]}}"
    DB_NAME="${SUB2API_DB_NAME:-${CFG[3]}}"
    DB_PASS="${SUB2API_DB_PASS:-${CFG[4]}}"
  else
    DB_HOST="${SUB2API_DB_HOST:-127.0.0.1}"
    DB_PORT="${SUB2API_DB_PORT:-13307}"
    DB_USER="${SUB2API_DB_USER:-sub2api}"
    DB_NAME="${SUB2API_DB_NAME:-sub2api}"
    DB_PASS="${SUB2API_DB_PASS:-}"
    warn "could not parse $CONFIG_FILE; using defaults (host=$DB_HOST, port=$DB_PORT)"
  fi
else
  DB_HOST="$SUB2API_DB_HOST"
  DB_PORT="${SUB2API_DB_PORT:-13307}"
  DB_USER="${SUB2API_DB_USER:-sub2api}"
  DB_NAME="${SUB2API_DB_NAME:-sub2api}"
  DB_PASS="${SUB2API_DB_PASS:-}"
fi

REDIS_HOST="${SUB2API_REDIS_HOST:-127.0.0.1}"
REDIS_PORT="${SUB2API_REDIS_PORT:-6379}"

export PGPASSWORD="$DB_PASS"
PSQL=(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -X -q -t -A)

# ---- helpers -----------------------------------------------------------------
require_psql() {
  if ! command -v psql >/dev/null; then
    err "psql not found in PATH; install postgresql-client"
    exit 127
  fi
}

require_psql

fetch_account() {
  "${PSQL[@]}" -c "SELECT id || '|' || name || '|' || type || '|' || platform || '|' ||
                         COALESCE(load_factor::text, 'NULL') || '|' ||
                         COALESCE(credentials->>'base_url', '') || '|' ||
                         COALESCE(priority::text, '50') || '|' ||
                         (schedulable::text) || '|' ||
                         status || '|' ||
                         COALESCE(last_used_at::text, 'NULL')
                  FROM accounts WHERE id = $1;"
}

# ---- rollback handling -------------------------------------------------------
BACKUP_FILE="$SCRIPT_DIR/.rebalance-safe.${ACCOUNT_ID}.bak"

if [[ "$MODE" == "--rollback" ]]; then
  if [[ ! -f "$BACKUP_FILE" ]]; then
    err "no backup file at $BACKUP_FILE"
    exit 1
  fi
  log "rolling back account $ACCOUNT_ID from $BACKUP_FILE"
  LOAD_FACTOR_BAK=$(awk -F'|' '{print $1}' "$BACKUP_FILE")
  BASE_URL_BAK=$(awk -F'|' '{print $2}' "$BACKUP_FILE")
  "${PSQL[@]}" <<SQL
BEGIN;
UPDATE accounts
SET load_factor = $LOAD_FACTOR_BAK,
    credentials = jsonb_set(credentials, '{base_url}', '$BASE_URL_BAK')
WHERE id = $ACCOUNT_ID;
COMMIT;
SQL
  rm -f "$BACKUP_FILE"
  log "rollback complete; backup file removed"
  exit 0
fi

# ---- main --------------------------------------------------------------------
log "DB: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
log "looking up account id=$ACCOUNT_ID"
ROW=$(fetch_account "$ACCOUNT_ID")
if [[ -z "$ROW" ]]; then
  err "account $ACCOUNT_ID not found"
  exit 2
fi

IFS='|' read -r ID NAME TYPE PLATFORM LOAD_FACTOR BASE_URL PRIO SCHED STATUS LAST_USED <<<"$ROW"
log "found: id=$ID name=$NAME type=$TYPE platform=$PLATFORM priority=$PRIO schedulable=$SCHED status=$STATUS last_used_at=$LAST_USED"

# normalize schedulable text for downstream comparisons
if [[ "$SCHED" == "true" ]]; then SCHED="t"; fi

# safety gates
if [[ "$TYPE" != "apikey" ]]; then
  err "refusing to touch account with type='$TYPE' (this plan only patches apikey accounts)"
  exit 3
fi
if [[ "$PLATFORM" != "openai" ]]; then
  err "refusing to touch account with platform='$PLATFORM' (this plan only patches openai accounts)"
  exit 3
fi
if [[ "$SCHED" != "t" && "$SCHED" != "true" ]]; then
  err "account is schedulable=$SCHED; enable it first if you really want to patch"
  exit 3
fi
if [[ "$STATUS" != "active" ]]; then
  err "account status='$STATUS'; set it to 'active' first if you really want to patch"
  exit 3
fi

# compute the two patches
NEW_LOAD_FACTOR="NULL"
if [[ -z "$BASE_URL" || "$BASE_URL" == */v1 || "$BASE_URL" == */v1/ ]]; then
  NEW_BASE_URL="$BASE_URL"
else
  NEW_BASE_URL="${BASE_URL%/}/v1"
fi

# detect whether anything actually needs to change
NEEDS_LOAD_FACTOR_PATCH=false
if [[ "$LOAD_FACTOR" != "NULL" && "$LOAD_FACTOR" != "0" ]]; then
  NEEDS_LOAD_FACTOR_PATCH=true
fi
NEEDS_BASE_URL_PATCH=false
if [[ -n "$BASE_URL" && "$BASE_URL" != "$NEW_BASE_URL" ]]; then
  NEEDS_BASE_URL_PATCH=true
fi

log "current load_factor = $LOAD_FACTOR"
log "current base_url    = $BASE_URL"
log "patched load_factor = $NEW_LOAD_FACTOR (needed: $NEEDS_LOAD_FACTOR_PATCH)"
log "patched base_url    = $NEW_BASE_URL (needed: $NEEDS_BASE_URL_PATCH)"

if ! $NEEDS_LOAD_FACTOR_PATCH && ! $NEEDS_BASE_URL_PATCH; then
  log "no changes needed; account is already in the patched state"
  exit 0
fi

if [[ "$MODE" != "--apply" ]]; then
  log "DRY-RUN — pass --apply to execute the following:"
  echo
  echo "  UPDATE accounts SET load_factor = NULL WHERE id = $ACCOUNT_ID;  -- ($NEEDS_LOAD_FACTOR_PATCH)"
  echo "  UPDATE accounts"
  echo "    SET credentials = jsonb_set(credentials, '{base_url}', '\"$NEW_BASE_URL\"')"
  echo "    WHERE id = $ACCOUNT_ID;  -- ($NEEDS_BASE_URL_PATCH)"
  echo
  log "no rows touched. nothing written to Redis."
  exit 0
fi

# apply
log "APPLY: writing patches to account $ACCOUNT_ID"
echo "${LOAD_FACTOR}|${BASE_URL}" > "$BACKUP_FILE"
log "backup saved to $BACKUP_FILE (use --rollback to restore)"

# build SQL dynamically — only update the fields that actually need changing
SQL="BEGIN;"
if $NEEDS_LOAD_FACTOR_PATCH; then
  SQL+="
UPDATE accounts SET load_factor = ${NEW_LOAD_FACTOR}::int WHERE id = $ACCOUNT_ID;"
fi
if $NEEDS_BASE_URL_PATCH; then
  # use a parameterised statement via psql \set to avoid quoting issues
  SQL+="
UPDATE accounts
SET credentials = jsonb_set(credentials, '{base_url}', \$NEWBU\$${NEW_BASE_URL}\$NEWBU\$)
WHERE id = $ACCOUNT_ID;"
fi
SQL+="
COMMIT;"

"${PSQL[@]}" <<<"$SQL"

log "verification:"
fetch_account "$ACCOUNT_ID" | awk -F'|' -v id="$ACCOUNT_ID" '{
  printf "  id=%s name=%s load_factor=%s base_url=%s\n", $1, $2, $5, $6
}'

log "done. Plan A rebalance complete."
log "next: monitor usage distribution for 1-2 days. expect new sessions to drift to this account naturally."
log "if you need faster rebalancing, run sub2api-rebalance-fast."
