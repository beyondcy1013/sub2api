# Redirect Plan: Operational Risks

The redirect plan is the most surgical of the four rebalance plans: it does not destroy the binding, it changes its target. The new target account is then used for the next request from that session. This document captures the operational risks that are unique to the redirect plan.

## 1. `previous_response_id` Chain Breakage

The OpenAI Responses API stores conversation state per-account, keyed by the `response_id`. When a session's `sticky_session` value is rewritten from account A to account B:

- The session's next request carries `previous_response_id=<id>` from the old chain.
- The gateway looks up the binding, gets account B, forwards the request with `previous_response_id=<id>`.
- Account B has no record of that `response_id`; the upstream either errors (`not_found`) or silently starts a new chain (depending on the gateway's behavior).

**Mitigation**: warn users that any conversation in progress will break at the redirect point. The redirect is appropriate for stateless, request-by-request workloads, not for long response chains.

## 2. Sudden Load on the Target Account

If 380 bindings all redirect at once, 380 sessions' next requests all hit the target account. With `concurrency=5` (yc), the first 5 requests run; the remaining 375 enqueue on `wait:account:20`. The wait queue is bounded by the gateway's `AcquireAccountSlot` timeout (`concurrency_service.go:165`), after which the requests fail.

**Mitigation**: use `--batch-size 50 --batch-delay 30s` (or smaller / longer) to spread the load. The throttle lets each batch drain before the next lands.

## 3. Bindings Drift Back

The redirected bindings are not "pinned" — they can drift back to the source account if:
- The target account is overloaded (FTFT > 15s triggers sticky escape, the session re-binds to a different account).
- A new request hits a code path that bypasses the binding (e.g. the request comes from a session_hash that was never bound).

**Mitigation**: monitor `usage_logs` 1-2 hours after the redirect. If the source account reclaims traffic, re-run the redirect.

## 4. TTL Mismatch

The original keys have whatever TTL was set when they were created (typically no TTL — most sticky bindings are persistent). The redirect script uses `SET ... KEEPTTL` to preserve the original TTL on each key, so a redirected key does not suddenly gain or lose expiration.

**Mitigation**: this is handled by the script; no user action needed.

## 5. Concurrent Writes

If the gateway writes a new binding to a key at the exact moment the redirect script is rewriting it, the gateway's `SET` may overwrite the redirect's `SET`, or vice versa. The race is small (the script holds a Redis connection but no MULTI/EXEC) but possible.

**Mitigation**: avoid running the redirect during a peak traffic window. Pick a quiet hour.

## Operational Checklist

Before running `--apply`:

- [ ] `sub2api-rebalance-safe` has been applied to the target account (load_factor=NULL, base_url ends in `/v1`).
- [ ] Target account's `concurrency` is known and is large enough to absorb the redirect (or use `--batch-size`).
- [ ] User has been notified that `previous_response_id` chains will break at the redirect point.
- [ ] A quiet hour has been picked (low traffic, off-peak).

After running `--apply`:

- [ ] Re-check the binding distribution with the `EVAL` snippet in `SKILL.md`.
- [ ] Watch `wait:account:<target>` for queue depth.
- [ ] Re-run the script 1-2 hours later if the source account reclaims traffic.
