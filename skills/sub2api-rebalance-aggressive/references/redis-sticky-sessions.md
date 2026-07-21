# Redis Sticky Session Keys

The gateway stores short- and long-term sticky bindings in Redis under predictable namespaces. This document enumerates the keys the aggressive rebalance plan touches.

## Key Layout

```
sticky_session:<group_id>:<platform>:<session_hash>          → account_id (string)
sticky_session:<group_id>:<platform>:response:<response_id>   → account_id (string)
sched:meta:<account_id>                                      → JSON snapshot of Account
sched:acc:<account_id>                                       → runtime acc stats
concurrency:account:<account_id>                             → ZSET of in-flight request IDs
wait:account:<account_id>                                    → INT pending wait count
```

`<session_hash>` is the FNV-1a hash of the request's session hint (or empty for unbound). `<response_id>` is the OpenAI `previous_response_id` chain token.

## Layer 2 / Layer 1 Lookup

Layer 2 (`session_hash` stickiness) reads:

```go
val := redis.Get(ctx, fmt.Sprintf("sticky_session:%d:%s:%s", groupID, platform, sessionHash))
```

Layer 1 (`previous_response_id` stickiness) reads:

```go
val := redis.Get(ctx, fmt.Sprintf("sticky_session:%d:%s:response:%s", groupID, platform, responseID))
```

If both lookups miss, the request falls through to Layer 3 (5-factor scoring or LRU).

## What the Aggressive Plan Does

For `group=2, platform=openai`:

1. `KEYS sticky_session:2:openai:*` (Layer 2 + Layer 1) — both the bare session binding and the `response:` sub-namespace.
2. `KEYS sched:meta:12 sched:meta:20` — the cached `Account` snapshot used by the runtime scorer. After deletion the next read recomputes from Postgres.
3. `KEYS sched:acc:12 sched:acc:20` — the runtime acc stats. Recomputed on next read.
4. (Optional) `concurrency:account:<id>` and `wait:account:<id>` for the active account — clears the in-flight bookkeeping so the active account starts the rebalance from a clean slate.

## Why This Is Fast

After deletion, every in-flight request misses Layer 1 and Layer 2 and reaches Layer 3. With the safe fixes already applied, the idle account scores higher and wins the next bind for the next ~5 minutes of traffic. The 380 existing bindings naturally re-distribute because the 5-factor scorer picks the previously-idle account for any new (or re-bound) session.

## Why This Is Disruptive

- Any client holding a `previous_response_id` from before the wipe will not be able to continue that chain. The new bound session is a fresh start.
- The runtime `sched:meta:*` cache is briefly cold after deletion; the first request after the wipe may pay a small Postgres round-trip penalty.

## Recovery If Something Goes Wrong

There is no rollback for the Redis wipe — once the keys are gone, the binding decisions are gone. If the rebalance produces unexpected distribution, the simplest recovery is:

```bash
# Re-run the aggressive plan; it is safe to re-run.
bash scripts/rebalance-aggressive.sh --group 2 --platform openai --apply

# Or just wait 1-2 days for natural drift (sub2api-rebalance-safe's path).
```

If you need to fully restore a prior binding map, the only option is to disable both accounts and re-enable them in the desired order — the new traffic will then bind to whichever the scheduler picks first.
