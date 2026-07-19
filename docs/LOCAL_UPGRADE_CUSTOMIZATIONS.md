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

- Selection column: `36px`.
- Name column: `126px`.
- Status column: `80px`.
- Account ID column: `130px`.
- Platform/type column: `170px`.
- The free-visible balance column: `70px`.
- Fixed-width cells apply `width`, `minWidth`, and `maxWidth`.
- Headers, labels, and sort indicators remain single-line and non-shrinking.
- Custom header slots do not suppress sortable-column indicators.
- First and last cells use `4px` outer padding.
- Non-final columns retain vertical separators in light and dark modes.
- Leading columns keep `schedulable -> usage -> platform/type`. After created
  time, keep today cost -> groups (when visible) -> balance -> 5h/7d
  request/token -> window cost. The ending order is account ID -> upstream
  declared rate -> actions.
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
- DataTable width/header/sort contracts;
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
