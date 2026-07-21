---
name: sub2api-switch-session-account
description: Move one Sub2API Redis sticky_session binding to a specified account. Use when the user asks to switch, move, pin, reroute, or bind a specific session_hash, previous_response_id/response_id, or sticky_session Redis key to a target account, instead of rebalancing an entire group. Supports dry-run first and writes only one Redis key with --apply.
---

# Sub2API Switch Session Account

Move a single sticky binding:

- `sticky_session:<group_id>:<platform>:<session_hash>`
- `sticky_session:<group_id>:<platform>:response:<response_id>`

Use this for surgical fixes such as "把这个会话切到账号 20" or "move this response chain binding to account 12". For bulk traffic movement, use `sub2api-rebalance-redirect` instead.

## Workflow

1. Identify the selector: full Redis key, `session_hash`, or `response_id`.
2. Identify `--group`, `--platform`, and `--to-account`.
3. Run dry-run first.
4. Use `--expect-from <id>` when the current account is known, to prevent racing a changed binding.
5. Add `--apply` only after dry-run shows exactly one intended key.

## Commands

Run from `/home/third_party/sub2api`:

```bash
# By session hash
bash skills/sub2api-switch-session-account/scripts/switch-session-account.sh \
  --group 2 \
  --platform openai \
  --session-hash d8ba45... \
  --to-account 20

# By response id / previous_response_id binding
bash skills/sub2api-switch-session-account/scripts/switch-session-account.sh \
  --group 2 \
  --platform openai \
  --response-id resp_abc... \
  --to-account 20 \
  --expect-from 12

# By full Redis key, apply after dry-run
bash skills/sub2api-switch-session-account/scripts/switch-session-account.sh \
  --group 2 \
  --platform openai \
  --key 'sticky_session:2:openai:response:resp_abc...' \
  --to-account 20 \
  --expect-from 12 \
  --apply
```

## Safety

- Dry-run is the default; `--apply` is required to write.
- The script refuses keys outside `sticky_session:<group>:<platform>:` for the supplied group/platform.
- The script verifies the target account belongs to the group, matches the platform, is not deleted, and has `status='active'`.
- The script refuses `schedulable=false` targets unless `--allow-non-schedulable` is passed.
- The script uses Redis `SET ... KEEPTTL` so the original key TTL is preserved.
- `--expect-from` is recommended for apply mode.

## Caveat

Switching an existing `response:` binding may break a `previous_response_id` chain if the target upstream account does not know that response id. If preserving conversation continuity matters, prefer keeping the session on the original account.
