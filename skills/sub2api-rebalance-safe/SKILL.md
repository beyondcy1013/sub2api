---
name: sub2api-rebalance-safe
description: Safely rebalance OpenAI account traffic by fixing the two known scheduling bugs in sub2api: clearing an inflated load_factor (so the 5-factor scoring no longer overrates the account as "idle") and patching the base_url to include the /v1 suffix (so the upstream no longer returns 404/timeout and the errorRateEWMA stops climbing). Use when an account is "in the same group as another account" yet never receives traffic because it was previously over-weighted or mis-configured. Idempotent, dry-run by default, only writes after --apply. This is the lowest-risk rebalance plan — sticky sessions drift naturally over 1-2 days; it does not force rebalancing.
---

# Sub2API Rebalance (Safe Plan)

Use the bundled bash script. By default it runs in **dry-run** mode and only prints the SQL it would execute. Pass `--apply` to actually write to the database.

## Background

Read [references/rebalance.md](references/rebalance.md) for the full diagnostic and root-cause analysis. The two data fixes below address:

1. `load_factor=100` on the idle account — `EffectiveLoadFactor()` returns 100 instead of the real `concurrency=5`, so `LoadRate = current*100/100 ≈ 0` and the 5-factor scoring falsely reports "always idle" while the account is actually saturated.
2. `base_url` missing `/v1` — the upstream returns 404/timeout, `errorRateEWMA` accumulates (α=0.2) and slowly drives the score down.

After these two fixes the score of the idle account rises above the active one and new sessions start drifting to it within hours/days.

## Usage

```bash
# Dry-run, show what would change
bash scripts/rebalance-safe.sh <account_id>

# Apply for real
bash scripts/rebalance-safe.sh <account_id> --apply

# Custom DB / Redis location
SUB2API_DB_HOST=10.0.0.1 SUB2API_DB_PORT=13307 \
  bash scripts/rebalance-safe.sh 20 --apply
```

The script:
1. Reads connection from `deploy/data/config.yaml` (or `SUB2API_DB_*` / `SUB2API_REDIS_*` env vars).
2. Looks up the account, prints the current `load_factor`, `credentials->>'base_url'`, `priority`, `last_used_at`.
3. Confirms the account is `apikey` / `openai` and `schedulable=true`. Refuses to touch OAuth/setup-token or already-disabled accounts.
4. Computes the patched `base_url` (adds `/v1` if missing).
5. **Dry-run** prints the two SQL statements; **apply** executes them in a single transaction and re-selects the row to confirm.

## Safety

- Default mode is **dry-run**. You must pass `--apply` to write.
- The script is **idempotent**: re-running it after a successful apply is a no-op (no rows updated, exit code 0).
- It refuses to run against any account whose `type` is not `apikey` (the bug only affects apikey accounts).
- It refuses to run against an account whose `schedulable=false` or `status != 'active'` (set those first if you want to fix them too).
- All writes are inside a single SQL transaction. If anything fails, the transaction is rolled back.

## Verification

After `--apply`:

```bash
# 1) The patched row
PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "
  SELECT id, name, load_factor, credentials->>'base_url' AS base_url, last_used_at
  FROM accounts WHERE id = <account_id>;
"

# 2) Wait 1-2 days, then check request distribution
PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "
  SELECT account_id, COUNT(*) AS reqs
  FROM usage_logs
  WHERE created_at > NOW() - INTERVAL '24 hours'
    AND account_id IN (<target_id>, <other_id>)
  GROUP BY account_id;
"
```

## Related Skills

- `sub2api-rebalance-fast` — same fix plus temporarily disabling the other account for 5-10 minutes to force re-binding.
- `sub2api-rebalance-aggressive` — same fix plus clearing all `sticky_session:*` keys for the group.

Both downstream skills depend on this one having been applied first.

## Full Diagnostic

The root-cause analysis that motivated this skill is in [references/rebalance.md](references/rebalance.md). Read it before running the script on a different pair of accounts, since the underlying bugs (`load_factor` overstating idle, `base_url` missing `/v1`, sticky-session lock) only apply to OpenAI `apikey` accounts; other account types use different code paths.
