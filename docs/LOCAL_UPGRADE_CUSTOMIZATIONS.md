# Unified Local Upgrade Customizations

This repository is the canonical source for both `sub2api.service` and
`sub2freeApi.service`. The two processes run the same embedded build artifact
with different deployment profiles and isolated runtime state.

The end-to-end upgrade and deployment procedure is
[UPGRADE_RUNBOOK.md](UPGRADE_RUNBOOK.md). This document is the behavior audit
checklist; it is not a substitute for the runbook.

## Runtime Boundaries

| Profile | Service | Port | PostgreSQL | Redis | Runtime directory |
| --- | --- | --- | --- | --- | --- |
| `main` | `sub2api.service` | `18381` | `sub2api` | DB `0` | `/home/third_party/sub2api/deploy` |
| `free` | `sub2freeApi.service` | `18382` | `sub2freeApi` | DB `1`, scheduler prefix `sub2freeApi` | `/home/third_party/sub2freeApi/deploy` |

Never share a database, Redis DB, scheduler key namespace, writable data
directory, environment file, runtime user, or service restart between the two
profiles. Source consolidation does not authorize data consolidation.

## Complete State Preservation

Before a merge or upstream refresh, preserve every ref, commit, tracked dirty
change, staged change, non-ignored untracked file, and ignored local policy or
skill file for the canonical repository. While the legacy free repository
remains present, snapshot it as an independent recovery source as well.

Required recovery material per repository:

- a backup branch at the pre-operation `HEAD`;
- `git bundle create ... --all` verified by `git bundle verify`;
- `status.txt`, `worktree.patch`, and `index.patch`;
- `local-files.list` plus a null-safe tar archive;
- the pre-operation deployed binary SHA and service PID.

Never use `git reset --hard`, `git clean`, rebase local commits away, force
push, or the WebUI binary updater.

## Shared Account-Management Contract

Both profiles must preserve all of the following:

- Admin account identifiers remain visible in plaintext.
- Admin responses and edit forms expose `credentials.api_key` in plaintext.
- `api_key` stays out of `SensitiveCredentialKeys`.
- OAuth tokens, cookies, session keys, AWS secrets, service account JSON,
  private keys, and `agent_private_key` remain redacted.
- `MergePreservingSensitiveCreds` preserves an existing `api_key` and every
  redacted credential when an older or partial frontend omits it.
- API-key account fields use `type="text"` and preload the current value.
- New-account initialization and reset use `concurrency: 4`.
- A new account selects the last available proxy and first available group.
  Late candidate arrival fills only empty selections and never overwrites an
  operator choice.
- Bulk-edit value controls are usable without first selecting a field checkbox.
  A value change automatically selects that field, and untouched defaults stay
  out of the update payload. Field checkboxes remain as the explicit path for
  applying a value equal to the form default or clearing an existing value.
- Standard and enhanced account imports show proxy/group routing controls with
  both apply checkboxes enabled by default. They select the last available
  proxy and first available group, allow explicit operator overrides, and wait
  for candidates before submitting. Disabling proxy application preserves the
  imported proxy relationship; disabling group application preserves the
  existing no-default-group import behavior.
- Enhanced Import pasted-text mode extracts multiple complete JSON values from
  mixed chat, forwarded, and Markdown text with a string-aware balanced
  object/array scanner. Nested values, quoted braces, escaped quotes, and
  backslashes must not split a segment. Each segment is validated independently
  with a one-based source label, then all segments merge into one import request
  in source order. Pure JSON and multi-file modes remain compatible; incomplete
  enclosing JSON is rejected instead of importing one of its inner values.
- Clone mode preserves the source proxy/group assignments, including explicit
  unassigned values, and never applies new-account routing defaults.
- Account recycle/restore uses `extra.recycled`; it does not use soft delete.
- Normal account lists exclude recycled rows; recycle-bin lists include only
  recycled rows.
- Active rows expose `编辑`, `测试连接`, `回收`, and `更多` directly in that order.
  The more menu does not duplicate `测试连接`.
- The account test dialog defaults `自动测试` to enabled, starts only after a
  default model has loaded, and persists the operator preference in browser
  storage under `sub2api.account-test.auto-start`.
- Account names remain inside the fixed-width name cell with single-line
  truncation and overflow clipping. They do not open a teleported hover
  tooltip.
- Status, groups, and capacity cells use plain text rather than badge/card
  styling.
- Usage auto-load runs only for active accounts; manual refresh remains
  available for every status.
- Usage progress bars remain compact and contain only window label, progress,
  utilization, and reset state. Request/token and `A`/`U` cost totals stay in
  their dedicated account-table columns.
- Valid zero-valued window statistics render as `0`; only genuinely missing
  window data renders as `-`, so newly added and lightly used accounts expose
  the same complete fields after a usage query.
- The parent account table consumes `AccountUsageCell`'s `usage-loaded` payload
  for the 5h/7d request, token, utilization, reset, and cost columns.
- `批量更新额度` remains immediately before `批量更新`. It queries the current
  selection, or only the currently loaded page when nothing is selected; limits
  targets to OpenAI OAuth and Anthropic OAuth/Setup Token; calls active usage
  with `force=true`; runs no more than four calls concurrently; continues after
  individual failures; and applies each successful result immediately.
- `handle429` persists rate-limit state with a detached, bounded context.
- Successful recover-state clears the in-memory scheduling block even when the
  database contains no recoverable state.
- Account action menus expose persistent `定时启用并恢复` and `定时暂停调度`
  tasks. Delay input is whole hours plus `0..59` minutes, with a 1-minute minimum
  and 365-day maximum, and a newly saved task replaces the same account's prior
  task.
- Scheduled account actions survive browser/service restarts in
  `scheduled_account_actions`. Due work is lease-claimed; failures retain their
  error and retry after one minute, while stale leases are reclaimable.
- `enable_and_recover` reuses full `RecoverAccountState(...InvalidateToken:true)`
  semantics before enabling scheduling. `pause` only sets schedulable false; it
  must not rewrite account status or reuse temp-unschedulable/scheduled-test state.

## Shared Account Table Contract

- Selection column minimum: `36px`.
- Name column and its inner content: fixed at `176px`; long names truncate.
- Status column minimum: `80px`.
- Account ID column minimum: `130px`.
- Platform/type column minimum: `170px`.
- The free-visible balance column minimum: `70px`.
- `AccountsView.vue` opts into `DataTable`'s `single-line-cells` and
  `dynamic-column-widths` modes. Desktop headers and all cell content remain on
  one line; stacked/wrapped cell layouts are flattened horizontally.
- In dynamic mode, declared widths apply only as `minWidth` at the table-cell
  level. Other content may expand columns and the table uses horizontal
  scrolling when it exceeds the viewport, while the name slot keeps an inner
  `176px` cap and truncates overflow. Other `DataTable` consumers retain fixed
  `width`/`minWidth`/`maxWidth` behavior.
- Headers, labels, and sort indicators remain single-line and non-shrinking.
- Custom header slots do not suppress sortable-column indicators.
- First and last cells use `4px` outer padding.
- Non-final columns retain vertical separators in light and dark modes.
- The account table enables `compact-rows`; desktop loading and data cells use
  `2px` top/bottom padding without changing the default density of other tables.
- Direct account actions use single-line `24px` icon buttons with accessible
  labels/tooltips so the action column does not force a taller row.
- Leading columns keep `actions -> name -> schedulable -> usage -> platform/type`. After today
  stats, keep 7d utilization (`7d(%)`) -> 7d reset. After created
  time, keep today cost -> groups (when visible) -> balance -> 5h/7d
  request/token -> window cost. The ending order is account ID -> upstream
  declared rate -> 5h utilization (`5h(%)`) -> 5h reset. The account table
  disables sticky positioning because actions precede the name column.
- Filters are hidden by default behind the filters toggle.
- Sidebar width remains `154px` expanded and `67px` collapsed.

## Main Profile Contract

The `main` profile must preserve:

- client error source `sub2api` without free branding prefixes;
- OpenAI sticky-session concurrency spillover when a historical bound account
  is full;
- strict `previous_response_id` affinity;
- historical sticky binding retention during one-connection spillover;
- bounded normal wait-plan fallback when all eligible accounts are full;
- recent sticky-session summary and reassignment APIs;
- the `迁入粘性会话` action for active, schedulable OpenAI targets;
- 1, 5, 15, and 60 minute activity windows, defaulting to 5 minutes;
- newest-first compare-and-set reassignment of at most 100 current 16-character
  lowercase-hex `session_hash` bindings with `SET ... KEEPTTL`;
- exclusion of legacy 64-character keys and every `response:` /
  `previous_response_id` continuation binding.

`Concurrency limit exceeded for user` is a local caller-concurrency wait
timeout, not an upstream provider message. Sticky concentration can consume
caller slots while requests wait on one account, so recent reassignment can
relieve the condition.

## Free Profile Contract

The `free` profile must preserve:

- local/auth/quota/concurrency/config errors prefixed with
  `【sub2freeApi限制】`;
- upstream-originated errors prefixed with `【上游错误】`;
- error source `sub2freeApi`;
- protocol-compatible prefixed `response.failed` streaming events;
- API-key middleware and direct service writers using the same prefix policy;
- balance-check configuration, API, local page, frontend view, scheduler, and
  account pause/resume behavior;
- the configurable balance URL, interval, timeout, concurrency, pause/stop/
  resume thresholds, and quota-hourly-limit requirement;
- per-account hidden balance detector classification in
  `extra.balance_check_type`: `sub2api` uses the account `base_url` normalized
  to `/v1/usage`, while `configured_api` preserves the configured balance API;
  unclassified custom API-key accounts probe sub2api once and then persist the
  successful type together with `extra.balance` in one update;
- sub2api balance parsing accepts wallet `balance`, top-level `remaining`, or
  `quota.remaining`; its HTTP client refuses redirects so an account Bearer key
  cannot be forwarded to a different redirect target;
- Redis scheduler key prefix `sub2freeApi`;
- account clone API support.

The free service must not use the main database, Redis DB `0`, main binary
deployment directory, or `sub2api.service` lifecycle.

## Shared Build And Deployment Contract

1. Build the shared frontend once with pnpm 9.
2. Build the canonical backend once with `CGO_ENABLED=0` and `-tags embed`.
3. Verify the embedded source version and Go build metadata.
4. Install the exact same artifact bytes atomically to both existing binary
   paths, retaining independent timestamped backups.
5. Restart and verify `sub2api.service` first without touching free.
6. Restart and verify `sub2freeApi.service` second without touching main.
7. Confirm both installed binaries and both `/proc/<pid>/exe` files have the
   same SHA-256.
8. Confirm each process profile, port, database, Redis DB/prefix, writable data
   directory, HTTP behavior, live version, and startup logs.

Any failed live check rolls back only the affected service. Keep repository
bundles, backup branches, patches, archives, and old binaries until all tests
and both live matrices pass.

## Super Priority Mode (超级优先)

Global state machine layered on top of the normal `schedulable` account flag.

- Per-account flag lives in the account `extra` JSONB under key `super_priority`
  (bool). Toggling it enqueues a scheduler snapshot update because the flag is
  evaluated at request time by the preference overlay; it must not be added to
  `schedulerNeutralExtraKeys`.
- Global mode + runtime params persist in the `super_priority` section of the
  YAML `config.yaml` (mirrors `balance_check`), backed by
  `config.SuperPriorityConfig` and viper defaults (`mode=normal`,
  `base_strategy=default`, `failure_threshold=2`,
  `check_interval=@every 1m`).
- `SuperPriorityService` (`backend/internal/service/super_priority_service.go`)
  owns the overlay state machine. `Activate` and `Deactivate` only switch the
  request-time preference overlay and never rewrite persisted `schedulable`
  state. `RecordFailure` tracks a rolling 1-minute per-account failure window
  and signals demotion at `failure_threshold`.
- The base strategy is `default` or `lowest_cost`; lowest-cost tiers use the
  account's persisted `rate_multiplier`. When the overlay is active,
  flagged accounts form the first strict preference tier; when that tier is
  unavailable or full, scheduling naturally falls back to ordinary accounts
  under the selected base strategy. Existing priority, load, LRU, compact,
  capability, and bounded-wait rules remain the tie-breakers inside a tier.
- `SuperPriorityRunner` (`super_priority_runner.go`) uses a cron ticker with the
  `@every`/descriptor parser, probes flagged accounts via
  `AccountTestService.RunTestBackground`, and auto-demotes on threshold.
- Both are wired in `backend/internal/service/wire.go` (providers + Start) and
  `backend/cmd/server/wire.go` `provideCleanup` (graceful Stop). Handler setters
  live on `SettingHandler.SetSuperPriorityService` and
  `AccountHandler.SetSuperPriorityService`.
- Routes: `GET|PUT /api/v1/admin/settings/super-priority`,
  `POST /api/v1/admin/settings/super-priority/activate`,
  `POST /api/v1/admin/settings/super-priority/deactivate`,
  `POST /api/v1/admin/accounts/:id/super-priority` (registered in
  `backend/internal/server/routes/admin.go`).
- Frontend: `frontend/src/api/admin/superPriority.ts` + `accounts.setSuperPriority`;
  `AccountActionMenu.vue` exposes the per-account 超级优先 action and
  `AccountsView.vue` keeps the settings entry in the account-tools menu;
  `AccountStatusIndicator.vue` shows the 超级优先 status text/color when
  `extra.super_priority` is set; `SuperPrioritySettingsModal.vue` exposes mode
  toggle, base strategy, threshold/interval, and test-model parameters.

When active, a flagged account's status must display as 超级优先 (fuchsia),
not 正常/暂停. The direct action order stays
`编辑` -> `测试连接` -> `回收` -> `更多`; the 超级优先 action remains in
the more menu so row height and width stay compact.

Focused regression verification:

```bash
cd /home/third_party/sub2api/backend && go test ./internal/service/ -run 'Test(SuperPriority|FilterByAccountSchedulingPreference|OrderAccountsBySchedulingPreference|BuildOpenAISelectionOrder_)' -count=1
cd /home/third_party/sub2api/backend && go test -tags unit ./internal/service/ -run 'TestBalanceCheckService|TestBalanceCheckRuntimeConfig' -count=1
cd /home/third_party/sub2api/frontend && pnpm vitest run src/components/admin/account/__tests__/SuperPrioritySettingsModal.spec.ts src/views/admin/__tests__/AccountsView.recycleDelete.spec.ts
```

## Required Verification

Backend focused tests must cover:

- deployment profile parsing and capability derivation;
- main/free client error policy matrices;
- public-settings API/injection schema parity;
- credential redaction and preserve-on-missing behavior;
- recycle/restore repository filters;
- sticky spillover and sticky reassignment;
- balance-check config/service/handler behavior;
- canceled-context rate-limit persistence and runtime-block recovery.
- scheduled account action validation, replacement/cancellation, leased due
  execution, failure retry, and recover-before-enable ordering;

Frontend focused tests must cover:

- create, clone, and edit account defaults;
- enhanced import mixed-text extraction, source-indexed validation, extraction
  summary, and single-request merging without exposing pasted credentials;
- plaintext API-key editing;
- active-only usage auto-load;
- DataTable width/header/sort/density contracts;
- account table columns and bulk actions;
- compact usage windows, complete zero-valued usage columns, reset labels, and
  bulk active-usage refresh scope/concurrency/partial-failure behavior;
- main-only sticky reassignment visibility;
- scheduled account action menu visibility, hours/minutes validation, target
  time display, save replacement, and pending-task cancellation;
- free-only balance-check navigation and route access.

Before deployment, run the complete canonical backend suite, full Vitest,
TypeScript typecheck, frontend production build, `git diff --check`, and a
conflict-marker scan. A soft browser reload can retain cached version state;
after deployment use `Ctrl+Shift+R` or `Cmd+Shift+R`.
