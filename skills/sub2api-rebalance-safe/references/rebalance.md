# Sub2API Account Rebalance — Diagnostic Reference

This document is shared by all three rebalance skills (`safe`, `fast`, `aggressive`). It captures the root-cause analysis and the data fix that all three plans share.

## Problem

Two accounts in the same group show extreme traffic imbalance:

- **Active account**: 9000+ requests in 14 days, currently saturating its `concurrency` slot
- **Idle account**: 1300+ requests in 14 days, last used 10 days ago, zero in-flight, zero wait

The idle account is `schedulable=true`, `status='active'`, all rate-limit / overload / temp-unschedulable fields are NULL. The database has no flag that would keep it from being picked. Yet the scheduler never picks it.

## Root Causes (verified against the codebase)

### Cause 1: `load_factor=100` overstates "idle"

Field `accounts.load_factor` is a free-form integer (`ent/schema/account.go`). The runtime scorer reads it through `EffectiveLoadFactor()` in `backend/internal/service/account.go:105-116`:

```go
func (a *Account) EffectiveLoadFactor() int {
    if a.LoadFactor != nil && *a.LoadFactor > 0 {
        return *a.LoadFactor       // 100 wins
    }
    if a.Concurrency > 0 {
        return a.Concurrency      // real ceiling is 5
    }
    return 1
}
```

The idle account had `load_factor=100` and `concurrency=5`, so the runtime thinks the cap is 100. The Redis-backed LoadRate computation in `backend/internal/repository/concurrency_cache.go:425-428`:

```go
loadRate = (currentConcurrency + waitingCount) * 100 / maxConcurrency
```

For this account, `loadRate ≈ current*100/100 ≈ 0` even when it is actually saturating its 5-slot ceiling. The 5-factor scoring in `backend/internal/service/openai_account_scheduler.go:754` then computes `loadFactor = 1 - 0/100 = 1.0` and the account looks "always idle" to the scorer. The scorer keeps picking it; the account can't actually serve the load; FTFT climbs past 15s; sticky escape kicks in; the session gets rebound to the other account. That positive feedback is what stranded the idle account.

### Cause 2: `base_url` missing `/v1`

The idle account's `credentials->>'base_url'` is `https://pay.kxaug.xyz` while the active account's is `https://pay.kxaug.xyz/v1`. After a deployment the OpenAI gateway started forcing the `/v1` suffix in `backend/internal/service/openai_gateway_service.go:3463`, so the idle account's requests hit `https://pay.kxaug.xyz/v1/v1/...`. The upstream returns 404/timeout, `errorRateEWMA` accumulates (α=0.2, `openai_account_scheduler.go:183`), and `errorFactor` further depresses the score.

### Cause 3: 380 sticky sessions all point at the active account

`KEYS sticky_session:2:openai:*` returns 380 keys; sampled values are all `"12"`. Layer 2 (`session_hash` stickiness) in `backend/internal/service/gateway_service.go:1714-1783` short-circuits Layer 3 scoring for the same session, so the idle account never gets a chance to be selected for any already-bound session. Only brand-new sessions could pick it, and brand-new sessions are rare 10 days in.

## The Two Data Fixes (shared by all three plans)

```sql
-- Fix 1: clear the inflated load_factor
UPDATE accounts SET load_factor = NULL WHERE id = <account_id>;

-- Fix 2: append /v1 to the base_url (idempotent)
UPDATE accounts
SET credentials = jsonb_set(credentials, '{base_url}', '"https://pay.kxaug.xyz/v1"')
WHERE id = <account_id>;
```

After these two writes the idle account's 5-factor score rises above the active account's. New sessions (and 5-factor tie-breakers) start picking it, and within 1-2 days the sticky bindings drift to a roughly even distribution.

## Why Three Plans?

The two data fixes are the necessary precondition. The three plans differ in **how aggressively** to also break the existing sticky-session lock.

| Plan | Skill | Sticky treatment | Risk | Time to balance |
|------|-------|------------------|------|-----------------|
| A | `sub2api-rebalance-safe` | Leave alone, drift naturally | None | 1-2 days |
| B | `sub2api-rebalance-fast` | Disable active account for 5-10 min | Short user-visible disruption | ~10 min |
| C | `sub2api-rebalance-aggressive` | `DEL` all `sticky_session:*` keys for the group | Breaks `previous_response_id` chains | ~5 min |

All three plans apply the same two SQL fixes first. The difference is whether (and how) they also touch Redis.

## Code Pointers

| Concern | File | Line |
|---------|------|------|
| 5-factor scoring | `backend/internal/service/openai_account_scheduler.go` | 762-766 |
| Top-K + weighted random | `backend/internal/service/openai_account_scheduler.go` | 519-552, 609-662 |
| LRU fallback | `backend/internal/service/gateway_service.go` | 2960-2980 |
| `EffectiveLoadFactor()` | `backend/internal/service/account.go` | 105-116 |
| LoadRate formula | `backend/internal/repository/concurrency_cache.go` | 425-428 |
| Sticky escape threshold (15s FTFT) | `backend/internal/service/openai_account_scheduler.go` | 1379-1408 |
| Sticky session lookup | `backend/internal/service/gateway_service.go` | 1714-1783 |
| base_url construction | `backend/internal/service/openai_gateway_service.go` | 3463 |
