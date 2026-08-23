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

Migration identity is the complete filename, not only the numeric prefix.
Preserve both `185_add_scheduled_account_actions.sql` and upstream
`185_group_reasoning_effort_policy.sql`; do not rename an already deployed
migration to make prefixes unique.

## README Advertisement Policy

Sponsor advertisement sections are intentionally absent from `README.md`,
`README_CN.md`, and `README_JA.md`. Continue merging upstream functional
documentation, licensing, notices, and project credits, but run
`scripts/remove-readme-sponsors.sh` after every upstream merge and require its
`--check` mode before a unified build. Do not resolve future README conflicts by
restoring the sponsor blocks wholesale.

## README Enhanced Edition Highlights

The top of `README.md`, `README_CN.md`, and `README_JA.md` contains a local
enhanced-edition section that explains the fork's verified operational value:
lowest-cost scheduling and liveness, sticky-session concurrency spillover,
5h/7d quota and cost visibility, account lifecycle tools, scheduled recovery,
and isolated main/free profiles. Preserve this section across upstream merges
and keep the three translations aligned. Do not replace concrete behavior with
unverifiable marketing claims.

The installation upgrade subsection must continue warning customized-fork
operators not to use the WebUI binary updater, because it installs an upstream
binary and discards local enhancements. Link to `docs/UPGRADE_RUNBOOK.md`
instead.

## Android Fastest-Healthy-Source Contract

As of 2026-08-23, the native Android WebView client is maintained in `android/`
and built by `scripts/build-sub2api-android-apk.sh`. It was restored after commit
`f719641b3` reverted the entire client while attempting to remove only the
virtual-mouse experiment introduced by `5ff00db87`. Future feature reverts must
be scoped to the feature; preserve the Android project, tests, signing identity,
and release script.

The client has these runtime guarantees:

- cold start submits every configured Sub2API health probe concurrently and
  immediately loads the first healthy completion;
- a remembered origin never adds a serial wait before that race;
- automatic selection and recovery are silent, with no countdown, switching
  Toast, spinner, or fixed delay;
- network callbacks are coalesced, verify the current source first, and do not
  switch while it remains healthy;
- only a selected-origin main-frame failure or a failed current-source probe
  starts a race excluding that failed source;
- runtime all-failed results preserve the existing WebView; cold-start
  all-failed results alone show the retry view;
- stale or duplicate WebView failure callbacks are guarded by selected origin
  and navigation generation;
- update checks concurrently race the corresponding webClx artifact endpoints.
  Read-only verification on 2026-08-23 found the update API returns `404` on
  both Sub2API `:18381` service origins and `200` on the paired `:11111` and
  `:11112` artifact origins, so update routing intentionally retains the
  separate paired endpoint list.

Regression verification:

```bash
node --test android/tests/client-contract.test.mjs
```

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
- An OpenAI API Key account with a custom `base_url` treats an upstream 401
  `token_invalidated` code, or the canonical `authentication token has been
  invalidated` message when the relay omits that code, as the relay's internal
  OAuth-account failure. The current request still fails over, but local
  `status` and `schedulable` remain unchanged. OAuth accounts, default OpenAI
  endpoints, `invalid_api_key`, and explicit custom 401 error-code policies
  retain their existing behavior.
- Custom OpenAI API-key relays may use the configurable relay failure budget:
  relay-internal invalidations, 408s, eligible 5xx responses, and transient
  transport failures are counted over a rolling window. When the configured
  ratio, minimum-request, or consecutive-failure threshold is reached, apply
  only the finite `openai_relay_failure_budget` runtime cooldown; never write
  a permanent account error. Success after cooldown clears the prior history.
- Local account state, relay failure-budget, quota-limit, and scheduling-rate
  controls live on the dedicated `/admin/account-policy-settings` page. Its
  per-account `GET|PUT /api/v1/admin/accounts/:id/policy-settings` contract
  reads and writes the existing account record in one update; it does not
  create a second source of truth. Runtime recovery and schedulable changes
  continue through their dedicated state APIs so cache invalidation and
  recovery behavior remain intact. Do not put relay failure-budget controls
  or the generic `rate_multiplier` field back into the upstream-heavy
  `EditAccountModal.vue`; unrelated account edits must not overwrite the
  dedicated scheduling-rate mode or value.
- New-account initialization and reset use `concurrency: 10`.
- The OAuth authorization step keeps the account name entered on the first step visible in a read-only summary above the authorization controls. This includes email addresses used as OpenAI account names.
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
- Enhanced CLIProxyAPI imports default newly normalized accounts to
  `concurrency: 10`. Native sub2api backup imports preserve each account's
  explicit exported concurrency; manual new-account initialization is also `10`.
- Their shared routing controls expose an opt-in automatic-save checkbox. When
  enabled, the last submitted proxy/group application flags and selections are
  restored for later standard and enhanced imports. Removed candidate IDs fall
  back to the current last proxy and first group, and pasted/imported secrets
  are never persisted.
- Enhanced Import pasted-text mode extracts multiple complete JSON values from
  mixed chat, forwarded, and Markdown text with a string-aware balanced
  object/array scanner. Nested values, quoted braces, escaped quotes, and
  backslashes must not split a segment. Each segment is validated independently
  with a one-based source label, then all segments merge into one import request
  in source order. Pure JSON and multi-file modes remain compatible; incomplete
  enclosing JSON is rejected instead of importing one of its inner values.
- The account tools menu exposes `批量清除导入账号` directly below Enhanced
  Import. It opens the enhanced parser in a dedicated clear mode without import
  routing controls; the normal import mode does not expose the destructive
  action. Clear mode reuses the exact same file/text normalization, ignores
  proxies, and asks a read-only backend preview to report both parsed and
  database-matched account counts before confirmation. The default action
  matches only ordinary or staged accounts and moves them through the normal
  recoverable deleted-staging service. An unchecked-by-default `彻底删除`
  checkbox additionally scans deleted staging and permanently deletes every
  match, including matches still in the ordinary or staged lifecycle. Its
  confirmation explicitly warns that recovery is impossible. Matching uses
  platform/type plus stable credential identity; name-only fallback is allowed
  only for a unique match. Missing and ambiguous inputs are reported without
  guessing or exposing credential values.
- Clone mode preserves the source proxy/group assignments, including explicit
  unassigned values, and never applies new-account routing defaults.
- Account **staging** (formerly "recycle") uses `extra.recycled`; it does not use soft delete.
- Standard and enhanced file imports tag every imported account with the source file's name part in `extra.import_filename`. The label is derived by stripping the file extension and splitting the stem at its **last** underscore, returning the part before that underscore (e.g. `chatgpt_pro_us_001.json` -> `chatgpt_pro_us`; a name with no underscore keeps the whole stem). It is shown in a dedicated `文件名` (`import_filename`) account-management table column placed after `备注` (notes); accounts with no value show `-`. Pasted-text enhanced import (no source file) leaves the field unset. The helper lives in `frontend/src/utils/importFilename.ts`.
  The filter is labeled "归档" (Archive) with an `archive` icon. It acts as an extra
  filter, not a deletion mechanism.
- Normal account lists exclude staged rows; staged lists include only
  recycled rows.
- **Recoverable deleted staging** is a second lifecycle level represented by
  `extra.deleted = true`; new deletes never set `deleted_at`:
  - `DELETE /admin/accounts/:id` removes `extra.recycled`, marks the account as
    deleted staging, and preserves credentials, account-group bindings, usage
    history, scheduled tests, and other configuration.
  - Ordinary/recycled lists exclude deleted-staging rows. The main account
    table requests `deleted=1` from its trash-icon toggle to display only those
    rows; `deleted=1` and `recycled=1` are mutually exclusive.
  - Deleted-staging rows keep the normal manual-management surface, including
    edit, direct connection test, recovery/state actions, statistics, and
    restore. The `更多` menu exposes a per-row `彻底删除` action (via the
    `permanent-delete` event on `AccountActionMenu.vue`) that opens an explicit
    irreversible-action confirmation. Batch `彻底删除` in the toolbar applies to
    both selected rows and all-page selections.
  - The deleted-staging bulk toolbar shows `彻底删除` only after accounts are
    selected. It processes selected IDs through the protected per-account
    permanent-delete API with bounded concurrency, retains failed IDs selected,
    and reports partial completion without retrying successful IDs.
  - A compact filter input stays beside the page/all-results selection controls
    in every lifecycle view. Server-side search covers account ID, name, notes,
    platform, type, status, and error message. Credential and token values are
    intentionally excluded from the GET query to keep secrets out of URLs,
    browser history, and access logs.
  - `POST /admin/accounts/:id/restore-from-trash` clears the new marker and
    republishes the scheduler snapshot. The same endpoint retains compatibility
    with legacy soft-deleted rows.
  - Routing and automated workers must exclude `extra.deleted=true`, including
    OAuth token refresh, scheduled tests, scheduling liveness, upstream billing
    and rate probes, automatic Ollama/usage/balance refresh, expiry auto-pause,
    scheduler candidate/score queries, and model-availability scans.
  - Full account edits preserve the server-owned lifecycle marker. Duplicates
    discard both `deleted` and `recycled`, so the new paused account appears in
    the normal list.
  - Migration `191_convert_account_soft_delete_to_deleted_staging.sql` restores
    group bindings saved by the former trash flow, clears `deleted_at`, and
    converts every legacy soft-deleted account to `extra.deleted=true`.
  - The permanent-delete route accepts only deleted-staging or legacy
    soft-deleted rows; normal and first-level staged rows remain protected.
- Active rows expose `编辑`, `测试连接`, `归档`, and `更多` directly in that order.
  The more menu does not duplicate `测试连接`.
- Staging (`recycled`) rows expose the same direct `编辑`, `测试连接`, and `更多`
  actions as active rows, with only the toggle action differing: staging rows show
  `取消归档` (`恢复`) where active rows show `归档`. Edit and test-connection must
  stay reachable in staging mode because staged accounts keep full credentials and
  config; the backend does not gate test/edit on `extra.recycled`.
- API-key accounts expose `查询余额` in `更多`. Each account stores its balance
  query scheme and optional same-origin API URL in `extra.balance_query`.
  Automatic probing supports Sub2API, NewAPI, OpenAI-compatible billing, and
  CPA endpoints; a successful fallback persists the detected scheme for later
  fast queries. Requests keep the account proxy, TLS fingerprint, and header
  overrides, and custom endpoints must remain on the account base URL origin.
- When direct HTTP schemes are unsupported, automatic probing falls back to the
  local signIn browser service via `GET /api/sites`,
  `POST /api/sites/:id/refresh-balance`, and `GET /api/jobs`. A successful
  browser refresh persists scheme `signin` plus `sign_in_site_id`, and reports
  the detected endpoint as `signin://<site-id>` for later fast queries.
- signIn matching prefers a remembered site ID, then an exact API-key match,
  then a unique base-URL origin match. Only single-account signIn sites are
  eligible; ambiguous matches fail without starting a browser job. Future
  site-specific browser implementations remain owned by signIn, so adding a
  supported site there does not require another Sub2API probing scheme.
- The signIn service defaults to `http://127.0.0.1:18712`; deployments may
  override it with `SIGNIN_BALANCE_SERVICE_URL`. The override must remain a
  loopback HTTP URL, redirects stay disabled, and request duration and response
  size remain bounded.
- The bottom of the `更多` menu keeps the low-frequency actions ordered as
  `查看统计` -> `创建 Spark 影子账号` (when available) -> `设置隐私` (when
  available), after recovery, scheduled actions, quota reset, and delete.
- The account test dialog defaults `自动测试` to enabled, starts only after a
  default model has loaded, and persists the operator preference in browser
  storage under `sub2api.account-test.auto-start`.
- Automatic connection tests and unconfirmed manual-test outcomes submit their
  observations to the shared per-account failure window and never call
  `SetError` directly before the configured
  `super_priority.failure_threshold` is reached. A definite failed manual test
  may ask the operator whether to mark the account as failed; confirming calls
  `POST /admin/accounts/:id/mark-failed`, which sets `status=error`,
  `error_message`, and `schedulable=false`. This operator-confirmed path is
  distinct from the automatic failure window and is never triggered by a
  background probe or an unconfirmed failure. A successful manual test must
  not implicitly recover account state; recovery remains an explicit operator
  confirmation.
- After a successful direct connection test, accounts that are not active or
  have scheduling paused show a confirmation dialog. Confirming performs full
  runtime-state recovery, activates an inactive account when necessary, and
  enables scheduling. Active, schedulable accounts do not show this prompt.
- After a definite failed direct connection test, an account that is not
  already in `error` status shows a confirmation dialog asking whether to mark
  it as failed. Confirming calls the dedicated `mark-failed` API to set
  `status=error`, store the test error message, and disable scheduling;
  accounts already in `error` status only refresh without prompting.
- Account names remain inside a fixed `212px × 32px` name cell. The name and
  supplemental email use one left-to-right text flow with the name first, so a
  long email cannot squeeze out or hide the name prefix. Combined text wraps
  naturally across at most two 16px lines and truncates only at the second-line
  end, so it does not grow the table row or escape through overflow. Names do
  not open a teleported hover tooltip.
- When the account name already ends with the same supplemental email, the
  email is not appended again. This suffix comparison trims surrounding
  whitespace and ignores case.
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
- The scheduled-test add and edit forms retain the raw 5-field Cron input and
  its help, plus a mouse-driven visual builder for minute intervals, hourly,
  daily, and weekly schedules. Unsupported or advanced expressions stay intact
  in custom mode; the builder changes only `cron_expression` and does not alter
  the backend scheduled-test contract.
- `enable_and_recover` reuses full `RecoverAccountState(...InvalidateToken:true)`
  semantics before enabling scheduling. `pause` only sets schedulable false; it
  must not rewrite account status or reuse temp-unschedulable/scheduled-test state.

## Shared Account Table Contract

- Selection column minimum: `36px`.
- Name column and its inner content: fixed at `212px`; long names preserve their
  beginning and truncate only at the end of the second line.
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
  `212px` cap and truncates overflow. Other `DataTable` consumers retain fixed
  `width`/`minWidth`/`maxWidth` behavior.
- Headers, labels, and sort indicators remain single-line and non-shrinking.
- Custom header slots do not suppress sortable-column indicators.
- First and last cells use `4px` outer padding.
- Non-final columns retain vertical separators in light and dark modes.
- The account table enables `compact-rows`; desktop loading and data cells use
  `2px` top/bottom padding without changing the default density of other tables.
- Direct account actions use single-line `24px` text buttons with visible
  labels and `px-2` horizontal padding. The operation column has a `220px`
  minimum so 编辑 -> 测试连接 -> 回收 -> 更多 stays visible without relying
  on tooltips or icon recognition.
- The selection, actions, and name columns stay fixed on the left during
  horizontal scrolling. Their declared `36px`, `220px`, and `212px` widths
  provide cumulative offsets so the fixed cells never overlap.
- After a successful create-account flow reloads the table, newly visible
  account rows are pinned above the current server-sorted page and use a
  distinct red shadow for the first 10 seconds, then an orange shadow for the
  next 10 seconds. They remain pinned and bold for the full 20 seconds, after
  which the operator's original sorting is restored. If the active sort omits
  the new row from page one, the frontend locates it with a newest-first query
  that preserves the current filters; the operator's chosen sort is not
  overwritten.
  Manual creation, standard import, enhanced import, CRS sync, direct account
  duplication, and Spark shadow creation all enter the same addition-tracking
  flow before they mutate account data; successful additions must not fall
  back to a plain table reload that skips pinning and highlighting.
  Standard and enhanced import responses include each created account's `id`
  and `name`. Their modals pass those identities to the table, which resolves
  and validates the exact account by ID before pinning it. When these explicit
  identities are present, never merge in page-difference or row-position
  guesses: a newly appeared first row under name sorting is not necessarily the
  imported account, and account names are not unique.
  Existing rows must not be highlighted when that reload returns to page one;
  desktop table rows and mobile account cards share the same marker.
- Leading columns keep `actions -> name -> schedulable -> usage -> platform/type`. After today
  stats, keep 7d utilization (`7d(%)`) -> 7d reset. After created
  time, keep today cost -> lifetime cost -> groups (when visible) -> balance -> 5h/7d
  request/token -> window cost. The ending order is account ID -> upstream
  declared rate -> scheduling rate -> 5h utilization (`5h(%)`) -> 5h reset. The account table
  keeps those three leading columns fixed so account identity remains visible.
- The lifetime-cost column is backed by `account_usage_totals`, not by an
  in-memory counter or a direct sum over retention-limited `usage_logs`.
  Migration `192_add_account_usage_totals.sql` seeds surviving history and an
  insert trigger atomically accumulates account, standard, and user costs plus
  request/token totals. Usage-log cleanup must never decrement this ledger.
- Filters are hidden by default behind the filters toggle.
- The account toolbar starts with a compact loop-test runtime summary showing
  the server-provided liveness countdown in `HH:MM:SS`, a cycle progress bar,
  and the latest success/failure/skip result. A due-but-not-started probe keeps
  an explicit `00:00:00` waiting state instead of the ambiguous "starting soon"
  label.
  It updates the countdown locally every second and polls the existing
  scheduling runtime status without reloading the account table.
- Sidebar width remains `154px` expanded and `67px` collapsed.

## Main Profile Contract

The `main` profile must preserve:

- client error source `sub2api` without free branding prefixes; generic
  forward-fallback errors explicitly append `(source: sub2api)` while
  passthrough and established main-profile messages remain unchanged;
- OpenAI sticky-session concurrency spillover when a historical bound account
  is full;
- strict `previous_response_id` affinity;
- historical sticky binding retention during one-connection spillover;
- bounded normal wait-plan fallback when all eligible accounts are full;
- recent sticky-session summary and reassignment APIs;
- the `迁入粘性会话` action for active, schedulable OpenAI targets;
- the action remains visible while public settings are loading and hides only
  when the capability is explicitly `false` (as it is for the free profile);
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

## Account Scheduling Rules

The account-table `调度规则` command exposes exactly two operating modes:
`default` and `lowest_cost`. The legacy endpoints, YAML mode, and
`extra.super_priority` values remain only for backward-compatible reads and
writes; they never affect request routing or account status display.

- `default` keeps the historical scheduling order: eligibility gates, then
  priority/load/LRU and the existing capability rules.
- `lowest_cost` is strict across every normal selection path. Eligible live
  accounts are ordered by `accounts.rate_multiplier`; the scheduler attempts
  the cheapest account first, then the next-cheapest account if the prior one
  is full, excluded, or becomes unavailable. When multiple eligible accounts
  tie for the lowest rate, connections are shared by current load, active
  connection count, and queue depth; exact ties rotate so one account cannot
  monopolize a shared load snapshot. The same equal-rate rotation applies when
  load batching is unavailable. Expensive accounts are not persistently disabled.
- `lowest_cost` does not honor movable `session_hash` affinity, so a historical
  expensive binding cannot override a cheaper eligible account. Strict
  non-movable `previous_response_id` affinity remains intact because a response
  chain cannot safely change accounts.
- `accounts.rate_multiplier` is the single scheduling and upstream-cost
  multiplier. Every account has `extra.scheduling_rate_sync_mode`:
  `auto_overwrite` (default) or `manual_lock`. A successful upstream billing
  probe copies only the stable `resolved_rate_multiplier`, rounded to the
  database's four-decimal precision, into `rate_multiplier` in the same
  transaction as the snapshot and scheduler outbox event. Peak/effective
  point-in-time values never drive scheduling. Failed, unsupported, or stale
  probes never overwrite the persisted multiplier.
- The scheduling-rate dialog treats automatic overwrite and manual editing as
  mutually exclusive. Selecting `auto_overwrite` disables and dims the manual
  multiplier input. Editing the manual multiplier, including copying the
  current upstream value into it, selects `manual_lock` and shows a Toast that
  automatic overwrite was disabled.
- Compatibility: absent sync mode defaults to `auto_overwrite`; legacy
  `scheduling_rate_source=upstream` maps to automatic overwrite and
  `scheduling_rate_source=manual` maps to manual lock.
- Global upstream-probe settings persist independently. The background runner
  scans every minute and probes due eligible accounts at the configured
  5..1440-minute interval. When enabled, the runner covers every active OpenAI
  API Key account; the legacy per-account probe flag is ignored. Automatic
  overwrite takes effect on the next scheduler snapshot refresh.
- Global upstream-probe settings also persist `notify_on_change_only`, which is
  disabled by default. Manual single and batch probes always refresh returned
  snapshots in the account table; when this option is enabled, a successful
  probe with the same effective upstream rate suppresses its completion Toast.
  Failed and unsupported probes remain visible as error or warning Toasts.
- Account balance detection supports a named `nikoapi` algorithm in addition
  to `auto`; persisted legacy `newapi` values normalize to `nikoapi`. Its
  balance query reads `/api/usage/token/` and converts raw quota with
  `/api/status`. Its rate probe reads the newest billed consumption entry from
  `/api/log/token`, preferring `other.user_group_ratio` over
  `other.group_ratio`. Automatic rate probing tries the Sub2API declaration
  contract first and falls back to this NikoAPI algorithm only when that
  contract is unsupported or has an incompatible response.
- While `lowest_cost` is active, the compatibility runner performs recovery
  probes only for `status=error` `api_key` accounts at the configured interval,
  with at most four concurrent connection tests. Healthy active accounts and
  manually paused accounts are never tested. OAuth, setup-token, Bedrock,
  Vertex, and every other non-`api_key` account type are always excluded. The
  persisted `super_priority.liveness_include_unschedulable` field remains
  accepted for compatibility but no longer expands the probe scope. A stable
  concrete model from the account's explicit model
  mapping is preferred. The configured `test_model_id` is only an OpenAI
  fallback and is used only when the account's explicit mapping verifies that
  model. An OpenAI account with neither a concrete mapped model nor a verified
  configured model is skipped instead of receiving an unverifiable request and
  recording a false liveness failure. If the selected account model is later
  rejected by the upstream with a structured model-not-found/not-allowed/not-
  supported response, the background probe is also skipped without updating
  liveness or emitting an account-test error log; authentication, rate-limit,
  timeout, overload, and other upstream failures remain health observations.
  Every account-test entry point applies the same explicit-mapping fallback
  before issuing its request; other platforms otherwise use their own default
  test model. A failed recovery request leaves the account stopped. A real
  successful upstream request invokes the existing successful-test recovery
  path to clear recoverable error, rate-limit, overload, temporary-
  unschedulable, model-rate-limit, and runtime-block state, then restores
  `schedulable=true`. No legacy marker may clear an error without a successful
  upstream request. Its diagnostic
  `extra.scheduling_liveness` snapshot transitions from `alive` to `suspect`,
  then `dead` after the configured consecutive-failure threshold. Only a fresh
  `dead` result is excluded; missing, stale, and `suspect` observations remain
  fallback candidates. Normal active accounts do not depend on a fresh
  liveness snapshot because they are no longer periodically tested.
- Automatic upstream billing/rate probing defaults to disabled so healthy API
  Key accounts receive no background rate-query traffic. The scheduling-rules
  dialog retains an explicit opt-in switch for installations that require
  automatic rate synchronization.
- The account-management scheduling-rules dialog keeps a `? Help` action
  immediately beside its title. Hovering it shows the current eligibility,
  default selection, equal-lowest-rate connection sharing, price-tier fallback,
  liveness recovery, existing-connection, and strict-response-affinity rules.
  Preserve the `BaseDialog` title-actions slot and the below-trigger tooltip
  placement that keep this explanation visible without crowding the form.
  Its upper-right header also shows the runner-owned connection-liveness state:
  running status or the countdown to the next expected batch, plus the latest
  batch time and succeeded/failed/skipped counts. The dialog polls this
  server-side state while open and uses a local one-second clock only to render
  the countdown; browser lifetime is never the source of scheduling truth.
- The account table renders `调度倍率` and `最优` in gold only when an account
  is currently schedulable and ties for the lowest persisted
  `rate_multiplier` in at least one of its scheduling
  groups. Every tied minimum is marked. Ungrouped accounts compare only with
  ungrouped accounts on the same platform. The backend computes this from the
  full active, non-recycled account pool rather than the visible page, and the
  marker is an administrative pool-level hint rather than a promise that every
  model-specific request can use that account.
- The implementation continues to use the existing `super_priority` YAML
  section only as compatibility storage for the durable base strategy,
  liveness interval, failure threshold, and optional test model.

Focused regression verification:

```bash
cd /home/third_party/sub2api/backend && go test -tags unit ./internal/service -run 'Test(OpenAI.*LowestCost|FilterByAccountSchedulingPreference|OrderAccountsBySchedulingPreference|BuildOpenAISelectionOrder|SchedulingLiveness|SchedulingRate)' -count=1
cd /home/third_party/sub2api/frontend && pnpm vitest run src/components/account/__tests__/SchedulingRulesModal.spec.ts src/components/account/__tests__/SchedulingRateModal.spec.ts src/components/account/__tests__/SchedulingRateCell.spec.ts src/components/admin/account/__tests__/AccountTableActions.schedulingRules.spec.ts
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
- dedicated account-policy API/UI loading, atomic policy saves, unsupported
  section omission, state recovery, and schedulable updates.

For routine scoped changes, run the documented focused regressions before
queueing `bash scripts/build-unified-release.sh`. Its default `quick` mode
rechecks the complete backend package set with Go's test cache enabled, skips
the duplicate full Vitest pass, and always runs TypeScript typecheck, the Vite
production build, the embedded Go build, `git diff --check`, source-integrity
checks, and the conflict-marker scan against the frozen deployment snapshot.
The snapshot lives at the stable
`/data/cargo-target/sub2api-unified-source` path behind a non-blocking `flock`;
this keeps Go test-cache paths reusable while preventing concurrent builds from
sharing the workspace. `rsync --delete` and the before/after source hashes keep
the retained workspace byte-for-byte aligned with the selected source tree.
The backend test command must also set `TMPDIR` to the stable, lock-protected
`/data/cargo-target/sub2api-unified-go-test-tmp` path. webClx assigns a unique
per-run `TMPDIR`; allowing that value to reach tests changes Go's test-input
cache key for packages that use temporary directories and silently forces the
slow suites to rerun on every deployment.

Use `bash scripts/build-unified-release.sh --full` for upstream merges,
dependency or toolchain changes, migrations, security/auth changes, broad
cross-module refactors, or when focused regression coverage cannot be
identified. Full mode forces the complete backend suite with `-count=1` and
the full Vitest suite before the same typecheck/build/deploy gates. A soft
browser reload can retain cached version state; after deployment use
`Ctrl+Shift+R` or `Cmd+Shift+R`.
