---
name: sub2api-rebalance-fast
description: Faster rebalance by temporarily disabling the active account for 5-10 minutes after the safe fixes are applied. Forces all new sessions to be bound to the previously-idle account. Use when sub2api-rebalance-safe is too slow and you can tolerate a short window of 100% traffic on the idle account. The script blocks for the configured window, then re-enables the active account automatically. Always run sub2api-rebalance-safe first.
---

# Sub2API Rebalance (Fast Plan)

Apply the same two data fixes as `sub2api-rebalance-safe`, then briefly flip the active account to `schedulable=false` for a configurable window. All new sessions during that window are forced to bind to the previously-idle account. When the timer fires, the active account is re-enabled and both accounts share traffic.

## Background

Read [references/rebalance.md](references/rebalance.md) (shared with the safe plan) for the data fixes. This plan adds a short, observable "all traffic goes to one account" window to force the 380 existing sticky sessions to re-bind.

## When to Use

- `sub2api-rebalance-safe` has been run, and
- You can tolerate 5-10 minutes of 100% traffic on the previously-idle account, and
- The active account has finished in-flight requests before you flip it (or you accept that those requests may be canceled).

**Do NOT use** during a peak window. Pick a quiet hour, ideally 02:00-05:00 local time.

## Usage

```bash
# Dry-run, show the schedule without writing
bash scripts/rebalance-fast.sh \
    --idle-account 20 \
    --active-account 12 \
    --window 8m

# Apply: safe-fix + flip active off + wait 8 min + flip active on
bash scripts/rebalance-fast.sh \
    --idle-account 20 \
    --active-account 12 \
    --window 8m \
    --apply
```

Arguments:
- `--idle-account` — the account you want traffic to flow to (must be apikey/openai, currently schedulable)
- `--active-account` — the account currently saturating (will be temporarily disabled)
- `--window` — duration to keep the active account disabled, e.g. `5m`, `10m`, `600s`. Default 8m.
- `--apply` — required to write. Without it the script prints the planned schedule and exits.

## How It Works

1. Refuses to run unless `sub2api-rebalance-safe` has already been applied (it checks `load_factor IS NULL` and the `/v1` suffix on the idle account).
2. Dry-run prints the schedule. With `--apply`:
   1. Apply the safe-fix SQL on the idle account (idempotent — re-applying is a no-op).
   2. `UPDATE accounts SET schedulable=false WHERE id=<active>`.
   3. Sleep for the window.
   4. `UPDATE accounts SET schedulable=true WHERE id=<active>`.
3. A backup file `.rebalance-fast.<active_id>.bak` records the previous `schedulable` value, in case you need to abort mid-window with `--rollback`.

## Abort Mid-Window

If the user wants to stop the sleep early:

```bash
# From another shell
bash scripts/rebalance-fast.sh --active-account 12 --rollback
```

The rollback reads the backup file and sets `schedulable` back to its original value. Safe to re-run.

## Verification

After the window closes and the active account is re-enabled, watch the request distribution:

```sql
SELECT account_id, COUNT(*) AS reqs
FROM usage_logs
WHERE created_at > NOW() - INTERVAL '24 hours'
  AND account_id IN (12, 20)
GROUP BY account_id
ORDER BY account_id;
```

You should see roughly even distribution within 1-2 hours of the window closing.

## Safety

- Dry-run is the default; `--apply` is required to write.
- The script refuses to run if the idle account still has `load_factor IS NOT NULL` or its `base_url` does not end in `/v1`.
- The backup file at `scripts/.rebalance-fast.<active_id>.bak` lets you roll back the `schedulable` flip even if the sleep is interrupted.
- The window is bounded — `sleep` is replaced by an interruptible wait, so Ctrl-C restores the active account.

## Related Skills

- `sub2api-rebalance-safe` — required prerequisite, apply first
- `sub2api-rebalance-aggressive` — same end-state but uses Redis `DEL` instead of a `schedulable=false` window

## Full Diagnostic

The root-cause analysis that motivated this skill is in [references/rebalance.md](references/rebalance.md).
