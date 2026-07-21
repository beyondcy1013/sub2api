---
name: sub2api-rebalance-redirect
description: Reassign existing sticky_session bindings in Redis from one account to another, optionally in batches with delay to avoid overwhelming the target. Use when you want to forcibly migrate live sessions from a saturated account to an idle one without destroying the bindings (which would force fresh previous_response_id chains). The script writes only the values — the keys themselves are preserved. Idempotent: re-running is a no-op once the target account value is already in place. Always run sub2api-rebalance-safe first. Supports --count N to migrate only the N most recently bound (highest TTL) sessions — useful when an account is saturated and you want to drain a few specific slots, not all 380.
---

# Sub2API Rebalance (Redirect Plan)

Reassign existing `sticky_session:<group>:<platform>:*` bindings from a source account to a target account, optionally in batches with a delay to avoid overwhelming the target.

## Most Common Use Case: Drain a Few Active Sessions

When account A is at `concurrency` cap and you want to free up slots by moving the **N most recently bound** sessions to account B:

```bash
# Pick the 5 most-recently-bound sessions bound to account 12, redirect to account 20
bash scripts/rebalance-redirect.sh \
    --group 2 --platform openai \
    --from-account 12 --to-account 20 \
    --only-from 12 \
    --count 5 \
    --active-source \
    --apply
```

`--active-source` makes the script refuse to run if account 12 currently has zero in-flight / wait — preventing the operator from accidentally moving traffic off a healthy account. `--count 5` keeps it surgical: only the 5 most-recently-bound sessions are rewritten.

## Background

Read [references/rebalance.md](references/rebalance.md) (shared with the safe / fast / aggressive plans) for the data fixes that all plans share. This plan adds a fourth option for breaking the sticky-session lock: **rewrite the binding value** rather than delete the key.

Compared to the other plans:

| Plan | What it does to bindings | Breaks `previous_response_id` chain? |
|------|--------------------------|--------------------------------------|
| safe (A) | Leaves them, drifts naturally | No |
| fast (B) | Leaves them, drift during disable window | No |
| aggressive (C) | Deletes them, re-bind on next request | **Yes** (fresh chain) |
| **redirect (D)** | **Rewrites the value, key stays** | **Yes** (new account has no chain context) |

`redirect` is the most controlled option — the next request from the same session lands on the target account without any re-evaluation delay. The trade-off is the same as `aggressive` regarding `previous_response_id`: the OpenAI Responses API will not find the prior chain on the new account, so any in-flight response chain breaks at the next request.

## When to Use

- `sub2api-rebalance-safe` has been run, and
- You want to **forcibly migrate** the live traffic, not wait for drift, and
- You accept that `previous_response_id` chains will break, and
- The target account has enough `concurrency` headroom to absorb the redirect (otherwise throttle with `--batch-size` / `--batch-delay`).

**Do NOT use** when the target account is itself saturated — even briefly, all redirected sessions will queue on the target's `wait:account:<id>` and time out. Run the safe fixes and let the target drain first.

## Usage

```bash
# Dry-run: list the keys that would be rewritten, grouped by source account
bash scripts/rebalance-redirect.sh --group 2 --platform openai

# Rewrite all 380 bindings from account 12 → account 20, in one batch
bash scripts/rebalance-redirect.sh \
    --group 2 --platform openai \
    --from-account 12 --to-account 20 \
    --apply

# Rewrite in batches of 50 with 30s between batches (recommended for hot accounts)
bash scripts/rebalance-redirect.sh \
    --group 2 --platform openai \
    --from-account 12 --to-account 20 \
    --batch-size 50 --batch-delay 30s \
    --apply

# Only redirect bindings that are still pointing at 12 (idempotent re-runs skip the rest)
bash scripts/rebalance-redirect.sh \
    --group 2 --platform openai \
    --from-account 12 --to-account 20 \
    --only-from 12 \
    --apply
```

Arguments:
- `--group` — `account_groups.group_id`
- `--platform` — `accounts.platform`, e.g. `openai`
- `--from-account` — the account to migrate bindings away from (only the keys whose value matches this id get rewritten, unless `--only-from` is omitted, in which case all keys in the group are rewritten)
- `--to-account` — the account to migrate bindings to
- `--only-from ID` — optional; only rewrite keys whose current value is exactly `ID`. Use this to leave other accounts' bindings alone.
- `--count N` — only rewrite the N most-recently-bound (highest TTL) keys. Default 0 (rewrite all). Combine with `--active-source` for the "drain a few slots" use case.
- `--random` — pick N random keys from the matching set instead of by TTL. Useful when you don't want a deterministic selection.
- `--active-source` — refuse to run if the source account's `concurrency` ZSET is empty AND `wait` counter is 0. Use this guard to avoid accidentally moving traffic off a healthy account.
- `--batch-size N` — rewrite at most N keys per batch (default 100, 0 = no batching)
- `--batch-delay D` — sleep `D` between batches (default 0; durations like `30s`, `2m`)
- `--apply` — required to write

## What It Writes

For each matching `sticky_session:<group>:<platform>:*` key (both bare and `response:*` namespaces), the script does:

```bash
SET sticky_session:2:openai:d8ba45...   "20"   # was "12"
SET sticky_session:2:openai:response:cb196...  "20"   # was "12"
```

The TTL on the keys is preserved (read with `TTL`, `SET` with `KEEPTTL` via `XX`-free `SET ... KEEPTTL`).

## Safety

- Default mode is dry-run. `--apply` is required.
- The script refuses to run if the target account still has `load_factor IS NOT NULL` or its `base_url` does not end in `/v1`.
- The script refuses to run if the target account is `schedulable=false` or `status != 'active'`.
- The script refuses to run if `from-account == to-account`.
- The script prints a warning if the target's current `concurrency` is less than the number of bindings being redirected in one batch — and recommends `--batch-size`.
- The script logs every key it rewrites in apply mode (truncated to 100 lines).
- Re-runs are safe: a key whose value is already `to-account` is left alone.

## Verification

After `--apply`:

```bash
# 1) The bindings are now pointing at the target account
redis-cli -p 6379 EVAL "
  local k = redis.call('KEYS', 'sticky_session:2:openai:*')
  local dist = {}
  for i=1, #k do
    local v = redis.call('GET', k[i])
    dist[v] = (dist[v] or 0) + 1
  end
  return dist
" 0

# 2) The target account's wait queue is filling up
redis-cli -p 6379 GET wait:account:20

# 3) Live traffic distribution (1-2 minutes after the redirect)
PGPASSWORD="..." psql -c "
  SELECT account_id, COUNT(*) AS reqs
  FROM usage_logs
  WHERE created_at > NOW() - INTERVAL '5 minutes'
    AND account_id IN (12, 20)
  GROUP BY account_id;
"
```

## Related Skills

- `sub2api-rebalance-safe` — required prerequisite
- `sub2api-rebalance-fast` — alternative that uses a `schedulable=false` window
- `sub2api-rebalance-aggressive` — alternative that deletes the keys outright
