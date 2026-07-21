---
name: sub2api-rebalance-aggressive
description: Fastest rebalance by deleting the group's sticky_session keys in Redis after the safe fixes are applied. Causes all bound sessions to re-bind on the next request; balances traffic in ~5 minutes. Use when sub2api-rebalance-safe is too slow and you cannot tolerate the user-visible 5-10 minute window of sub2api-rebalance-fast. Always run sub2api-rebalance-safe first. This is the most disruptive plan: it breaks any active previous_response_id chains because the rebinding produces fresh session IDs.
---

# Sub2API Rebalance (Aggressive Plan)

Apply the same two data fixes as `sub2api-rebalance-safe`, then `DEL` every `sticky_session:<group_id>:<platform>:*` key in Redis. All bound sessions re-bind on the next request, picking whichever account scores higher.

## Background

The aggressive plan shares its data fixes with the safe plan; read [references/rebalance.md](references/rebalance.md) for those. The aggressive plan adds the Redis-side wipe explained in [references/redis-sticky-sessions.md](references/redis-sticky-sessions.md).

## When to Use

- `sub2api-rebalance-safe` has been run, and
- You cannot afford the 5-10 minute user-visible disable window of `sub2api-rebalance-fast`, and
- You accept that any active `previous_response_id` chains will be broken (the OpenAI Responses API treats the new bound session as a fresh conversation).

**Do NOT use** while users are mid-conversation; the new bound session will not be able to continue the old response chain.

## Usage

```bash
# Dry-run: list the keys that would be deleted
bash scripts/rebalance-aggressive.sh --group 2 --platform openai

# Apply: safe fixes + delete sticky keys + clear wait queues for the active account
bash scripts/rebalance-aggressive.sh \
    --group 2 \
    --platform openai \
    --active-account 12 \
    --apply

# Rollback is not supported — once the sticky keys are deleted, the
# binding decisions are gone. Use --apply carefully.
```

Arguments:
- `--group` — `account_groups.group_id` whose sticky sessions should be cleared
- `--platform` — `accounts.platform` for the namespace, e.g. `openai`, `anthropic`, `gemini`
- `--active-account` — optional; if given, also clears `wait:account:<id>` and `concurrency:account:<id>` so the active account starts cold
- `--apply` — required to write

## What Gets Deleted

For `group=2, platform=openai`, the script enumerates and deletes:

- `sticky_session:2:openai:*` (the session_hash → account_id binding, ~380 keys)
- `sticky_session:2:openai:response:*` (the previous_response_id → account_id binding, ~50 keys in this case)
- `sched:meta:12`, `sched:meta:20` (the runtime meta snapshot for each account in the group)
- `sched:acc:12`, `sched:acc:20` (the runtime acc stats)
- if `--active-account` is set: `concurrency:account:<id>` and `wait:account:<id>` for that account

The script prints the count of keys it will touch in dry-run mode.

## Verification

After `--apply`, watch the rebinding in real time:

```bash
# Tail the new bindings
redis-cli -p 6379 EVAL "
  local k = redis.call('KEYS', 'sticky_session:2:openai:*')
  local dist = {}
  for i=1, #k do
    local v = redis.call('GET', k[i])
    dist[v] = (dist[v] or 0) + 1
  end
  return dist
" 0
```

Expect roughly 50/50 distribution after 5-10 minutes of traffic.

## Safety

- Dry-run is the default; `--apply` is required to write.
- The script refuses to run if the idle account still has `load_factor IS NOT NULL` or its `base_url` does not end in `/v1`.
- The deletion is targeted: only `sticky_session:<group>:<platform>:*` and the per-account runtime keys for accounts in that group. Other groups, other platforms, and unrelated sessions are untouched.
- The script logs every key it deletes in apply mode (truncated to 100 lines) so you have a paper trail.

## Related Skills

- `sub2api-rebalance-safe` — required prerequisite, apply first
- `sub2api-rebalance-fast` — alternative that uses a `schedulable=false` window instead of a Redis wipe

## Full Diagnostic

The root-cause analysis that motivated this skill is in [references/rebalance.md](references/rebalance.md).
