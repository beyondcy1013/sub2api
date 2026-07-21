#!/usr/bin/env bash
# rebalance-fast.sh — Plan B: safe fixes + brief schedulable=false on the active account.
#
# Usage:
#   bash rebalance-fast.sh --idle-account ID --active-account ID [--window 8m] [--apply]
#   bash rebalance-fast.sh --active-account ID --rollback
#
# Requires: sub2api-rebalance-safe to have been applied to the idle account.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CONFIG_FILE="$REPO_ROOT/deploy/data/config.yaml"

log()  { printf '\033[1;34m[rebalance-fast]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[rebalance-fast]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[rebalance-fast]\033[0m %s\n' "$*" >&2; }

# ---- arg parsing -------------------------------------------------------------
IDLE_ID=""
ACTIVE_ID=""
WINDOW="8m"
MODE="dry-run"
ROLLBACK=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --idle-account)    IDLE_ID="$2"; shift 2 ;;
    --active-account)  ACTIVE_ID="$2"; shift 2 ;;
    --window)          WINDOW="$2"; shift 2 ;;
    --apply)           MODE="apply"; shift ;;
    --rollback)        ROLLBACK=true; shift ;;
    *) err "unknown arg: $1"; exit 64 ;;
  esac
done

if $ROLLBACK; then
  if [[ -z "$ACTIVE_ID" ]]; then err "--active-account required for --rollback"; exit 64; fi
  BACKUP="$SCRIPT_DIR/.rebalance-fast.${ACTIVE_ID}.bak"
  if [[ ! -f "$BACKUP" ]]; then err "no backup at $BACKUP"; exit 1; fi
  SAVED=$(cat "$BACKUP")
  log "rolling back schedulable of account $ACTIVE_ID to '$SAVED'"
  PGPASSWORD="$(python3 -c "import yaml; print(yaml.safe_load(open('$CONFIG_FILE'))['database']['password'])")" \
    psql -h 127.0.0.1 -p 13307 -U sub2api -d sub2api -X -q \
      -c "UPDATE accounts SET schedulable = $SAVED WHERE id = $ACTIVE_ID;"
  rm -f "$BACKUP"
  log "rollback complete"
  exit 0
fi

if [[ -z "$IDLE_ID" || -z "$ACTIVE_ID" ]]; then
  err "Usage: $0 --idle-account ID --active-account ID [--window 8m] [--apply|--rollback]"
  exit 64
fi

# ---- DB connection -----------------------------------------------------------
if ! command -v psql >/dev/null; then err "psql not found"; exit 127; fi

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

# ---- preflight: safe fixes must already be applied ---------------------------
log "preflight: checking that sub2api-rebalance-safe has been applied to account $IDLE_ID"
IDLE_ROW=$("${PSQL[@]}" -c "SELECT load_factor IS NULL AS lf_null,
                                    credentials->>'base_url' AS bu,
                                    type,
                                    platform,
                                    schedulable,
                                    status
                             FROM accounts WHERE id = $IDLE_ID;")
if [[ -z "$IDLE_ROW" ]]; then err "idle account $IDLE_ID not found"; exit 2; fi
IFS='|' read -r LF_NULL BASE_URL TYPE PLATFORM SCHED STATUS <<<"$IDLE_ROW"
if [[ "$LF_NULL" != "t" ]]; then
  err "load_factor on account $IDLE_ID is not NULL; run sub2api-rebalance-safe first"
  exit 3
fi
if [[ "$BASE_URL" != */v1 && "$BASE_URL" != */v1/ ]]; then
  err "base_url on account $IDLE_ID does not end in /v1 ('$BASE_URL'); run sub2api-rebalance-safe first"
  exit 3
fi
if [[ "$TYPE" != "apikey" || "$PLATFORM" != "openai" || ( "$SCHED" != "t" && "$SCHED" != "true" ) || "$STATUS" != "active" ]]; then
  err "idle account state invalid (type=$TYPE platform=$PLATFORM schedulable=$SCHED status=$STATUS)"
  exit 3
fi

ACTIVE_ROW=$("${PSQL[@]}" -c "SELECT schedulable::text, status FROM accounts WHERE id = $ACTIVE_ID;")
if [[ -z "$ACTIVE_ROW" ]]; then err "active account $ACTIVE_ID not found"; exit 2; fi
IFS='|' read -r ACTIVE_SCHED ACTIVE_STATUS <<<"$ACTIVE_ROW"
if [[ "$ACTIVE_SCHED" != "t" && "$ACTIVE_SCHED" != "true" ]]; then
  err "active account $ACTIVE_ID is already schedulable=$ACTIVE_SCHED; nothing to flip"
  exit 3
fi

log "all preflight checks passed"

# normalize schedulable text for downstream comparisons
if [[ "$SCHED" == "true" ]]; then SCHED="t"; fi
if [[ "$ACTIVE_SCHED" == "true" ]]; then ACTIVE_SCHED="t"; fi

# ---- plan --------------------------------------------------------------------
START_TS=$(date +%s)
END_TS=$((START_TS + $(parse_duration_seconds "$WINDOW")))

log "schedule:"
log "  T+0  apply safe-fix SQL to idle account (idempotent)"
log "  T+0  flip active account $ACTIVE_ID to schedulable=false"
log "  T+${WINDOW}  flip active account $ACTIVE_ID back to schedulable=true"
log "  T+${WINDOW}  exit 0"

if [[ "$MODE" != "apply" ]]; then
  log "DRY-RUN — pass --apply to execute. nothing will be written."
  exit 0
fi

# ---- apply -------------------------------------------------------------------
log "[T+0] applying safe fixes to idle account $IDLE_ID (idempotent)"
"${PSQL[@]}" <<SQL
BEGIN;
UPDATE accounts SET load_factor = NULL WHERE id = $IDLE_ID;
UPDATE accounts
SET credentials = jsonb_set(credentials, '{base_url}',
  COALESCE(credentials->>'base_url', '') || CASE
    WHEN credentials->>'base_url' LIKE '%/v1' OR credentials->>'base_url' LIKE '%/v1/' THEN ''
    ELSE '/v1'
  END)
WHERE id = $IDLE_ID;
COMMIT;
SQL

log "[T+0] flipping active account $ACTIVE_ID to schedulable=false"
echo "t" > "$SCRIPT_DIR/.rebalance-fast.${ACTIVE_ID}.bak"
"${PSQL[@]}" -c "UPDATE accounts SET schedulable = false WHERE id = $ACTIVE_ID;"

# interruptible wait
cleanup() {
  warn "interrupted — restoring active account $ACTIVE_ID to schedulable=true"
  "${PSQL[@]}" -c "UPDATE accounts SET schedulable = true WHERE id = $ACTIVE_ID;" || true
  rm -f "$SCRIPT_DIR/.rebalance-fast.${ACTIVE_ID}.bak"
  exit 130
}
trap cleanup INT TERM

log "[T+0..T+${WINDOW}] waiting. Ctrl-C to abort and auto-restore."
sleep_until "$END_TS"

log "[T+${WINDOW}] flipping active account $ACTIVE_ID back to schedulable=true"
"${PSQL[@]}" -c "UPDATE accounts SET schedulable = true WHERE id = $ACTIVE_ID;"
rm -f "$SCRIPT_DIR/.rebalance-fast.${ACTIVE_ID}.bak"
trap - INT TERM

log "done. both accounts are schedulable. monitor usage distribution for the next 1-2 hours."

# ---- helpers -----------------------------------------------------------------
parse_duration_seconds() {
  local s="$1"
  if [[ "$s" =~ ^([0-9]+)s$ ]]; then echo "${BASH_REMATCH[1]}"; return; fi
  if [[ "$s" =~ ^([0-9]+)m$ ]]; then echo $(( ${BASH_REMATCH[1]} * 60 )); return; fi
  if [[ "$s" =~ ^([0-9]+)h$ ]]; then echo $(( ${BASH_REMATCH[1]} * 3600 )); return; fi
  err "bad duration: $s (use 30s, 5m, 1h)"; exit 64
}

sleep_until() {
  local target="$1"
  while (( $(date +%s) < target )); do
    remaining=$(( target - $(date +%s) ))
    if (( remaining > 60 )); then
      log "  …T-$((remaining/60))m$((remaining%60))s remaining"
      sleep 60
    else
      sleep "$remaining"
      break
    fi
  done
}
