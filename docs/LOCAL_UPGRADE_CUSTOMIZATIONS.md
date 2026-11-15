# Unified Local Upgrade Customizations

## Explicit Account Test Model Selection (2026-10-13)

- Regular connectivity tests preserve a nonempty caller-selected model even
  when it is outside the account whitelist. Only an empty selection uses the
  account's default model. An unsupported Sol test must not silently use Luna.
- Explicit operator mappings still apply in normal mode; this change removes
  the implicit whitelist fallback, preserving Pelican's exact-model behavior.
- Regression: `TestAccountTestExplicitModelNeverFallsBackToWhitelist`.
  Verified and deployed to main/free by webClx request
  `103752-18dd824fca025187` on 2026-10-13; both running binaries match
  SHA-256 `09ab741fda793aea24b118abfe25c5905a51e7c91542407968ee2972d3b94dc6`.

## Current Codex Official Login Management

- The first toolbar button syncs the complete official ChatGPT OAuth login from
  `/home/root/.codex`; third-party providers and API keys are rejected.
- Accounts are stored independently, created inactive and unschedulable without
  default groups. Repeated sync pauses matching accounts before updating them.
- CLI configuration changes never delete saved accounts. OAuth expiration and
  revocation still apply; caching does not guarantee indefinite token validity.
- Saved accounts offer a confirmed restore action that backs up both CLI files,
  restores OAuth login and the official OpenAI provider for new sessions without
  changing account scheduling. Deployment installs a restricted root-owned bridge.

This repository is the canonical source for both `sub2api.service` and
`sub2freeApi.service`. The two processes run the same embedded build artifact
with different deployment profiles and isolated runtime state.

The end-to-end upgrade and deployment procedure is
[UPGRADE_RUNBOOK.md](UPGRADE_RUNBOOK.md). This document is the behavior audit
checklist; it is not a substitute for the runbook.

The maintained operations entry point is [sub2api-ops](../skills/sub2api-ops/SKILL.md). The [2026-09-30 source audit](OPS_REVIEW_2026-09-30.md) records current operational gaps; its proposed fixes are not deployed behavior.

## Runtime Boundaries

| Profile | Service | Port | PostgreSQL | Redis | Runtime directory |
| --- | --- | --- | --- | --- | --- |
| `main` | `sub2api.service` | `18381` | `sub2api` | DB `0` | `/home/codes/ai/sub2api/deploy` |
| `free` | `sub2freeApi.service` | `18382` | `sub2freeApi` | DB `1`, scheduler prefix `sub2freeApi` | `/home/codes/ai/sub2freeApi/deploy` |

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

`android/VERSION` is the authoritative client release version. It is intentionally
independent from `backend/cmd/server/VERSION`: the backend remained at `0.1.179`
after Android `0.1.180` had already been published, which caused the first rebuild
attempt to produce a downgrade artifact and fail the required-artifact gate. Every
Android publication must advance this client version monotonically relative to the
current update manifest.

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
- native refresh/settings actions stay hidden behind a small floating handle at
  the right edge, centered about one third of the way down the screen. Tapping
  the handle expands both actions; either action collapses them again, and the
  WebView keeps its full layout without a reserved native toolbar;
- update checks concurrently race the corresponding webClx artifact endpoints.
  Read-only verification on 2026-08-23 found the update API returns `404` on
  both Sub2API `:18381` service origins and `200` on the paired `:11111` and
  `:11112` artifact origins, so update routing intentionally retains the
  separate paired endpoint list.

The Android WebView appends `sub2apiAndroid/<version>` to its user agent. The
account-management page uses that explicit identity to force `DataTable`'s
desktop-style table layout even below the normal 768px breakpoint. The native
layout fixes only the 36px selection column, allowing operation, name, and every
later column to move through the viewport during horizontal swipes; desktop keeps
the fixed selection/action/name columns. This override is intentionally scoped to
the native client; ordinary mobile browsers continue to use the shared card layout.

As of 2026-09-17, the native client's account-management toolbar compresses the
rows above the table to give the dense table more vertical space:

- the `循环测试` liveness runtime summary block (`AccountSchedulingRuntimeSummary`)
  is kept mounted but visually hidden; when a scheduled liveness run finishes, the
  result is reported through a success/error Toast instead of occupying a fixed
  toolbar row. The desktop inline summary block is unchanged;

As of the responsive toolbar rework, the account-management toolbar is
width-responsive (CSS `lg` / 1024px breakpoint, not user-agent sniffing) and
applies to every client including the native Android WebView:

- below `lg` the toolbar keeps only the create, refresh, and filter entries plus
  a labeled `功能`/`Functions` trigger (`data-test="account-functions-trigger"`);
  scheduling rules, table pelican test, archived/deleted view toggles, the
  scheduled-tests entry, the auto-refresh dropdown, and the more-tools dropdown
  stay in the DOM but are hidden below `lg`. The `功能` trigger opens a
  Teleport bottom sheet (`MobileActionSheet.vue`) with categorized actions:
  view switching (normal/archived/deleted), account operations (scheduling
  rules, table pelican test, global scheduled tests), auto-refresh
  toggle+interval with live state, zero-selection batch entries (select page,
  select all results, refresh usage, bulk edit filtered, single-selection
  sticky-session reassignment), all data/import/export/sync tools, error
  passthrough, TLS fingerprint profiles, balance-check settings, and column
  visibility toggles;
- a compact search input bound to the existing server-side search stays
  visible below `lg` and reuses the same debounced reload path (no duplicate
  API calls). The full filter panel remains reachable through the filter
  toggle;
- the bulk selection bar (`AccountBulkActionsBar`) renders only while
  `selectedIds.length > 0` below `lg`, as its own full-width row separate from
  the toolbar on every client including Android (`bare` mode keeps the
  `#afterCreate` slot contract). It shows the selected count, a `批量操作`
  trigger, and `清除选择`; the trigger opens the same `MobileActionSheet`
  reusing the mobile quick proxy/group selectors and selected-scope actions
  with scope labels. Selected actions close the sheet before launching while
  quick selectors stay open; dangerous deletes live in a separate section and
  keep their existing confirmations. Zero-selection operations (refresh
  usage, bulk edit filtered, select page/all results) remain reachable from
  the Functions sheet;
- the wide (`lg` and up) desktop toolbar, dropdown menus, and bulk bar are
  unchanged.

Also as of 2026-09-17, the native Android client pins only the 36px selection
column during horizontal scrolling (`accountStickyLeftColumnKeys = ['select']`);
the operation, name, and later columns scroll into view while swiping. The
previous code pinned selection + actions + name (a 468px sticky region wider
than the phone viewport), which made swiping appear to do nothing. Desktop
keeps the selection/action/name sticky columns unchanged.

Regression verification:

```bash
node --test android/tests/client-contract.test.mjs
```

## Shared Account-Management Contract

Both profiles must preserve all of the following:

- The account 测试连接 model picker (`GET /api/v1/admin/accounts/:id/models`)
  only offers models the account can actually serve. `service
  .AccountTestPickerModelIDs` filters every picker catalog (OpenAI upstream
  discovery, Gemini/Google One curated lists, Antigravity defaults, Grok/Claude
  defaults) through the account model whitelist semantics of
  `IsModelSupported`, and appends concrete whitelist aliases missing from the
  catalog. Server-forced Antigravity gemini passthroughs stay servable;
  OpenAI passthrough accounts and accounts without a whitelist keep the
  catalog minus models routing would never serve. 测试连接不得出现模型白名单
  之外的模型。
- Pelican model catalogs are account-owned. The UI calls each account's
  `/models` picker endpoint, shows that account's options, and defaults to its
  first catalog model whose ID starts with `gpt-6`; global dropdown and per-account
  suggestions exclude all other prefixes. Free-form model inputs still accept
  any model ID, including non-`gpt-6` models. It must not use stale frontend model presets. An explicit
  global override may force one model across the batch, but the default empty
  value preserves each account's own choice. Requests still bypass account
  `model_mapping` (including wildcard aliases), while ordinary connection tests
  keep their existing model-mapping behavior.
  Starting a test clears the previous result's model/timestamp/duration and
  displays the selected model until the current upstream reports its model.
- Pelican cards expose independent test/stop actions, each with its own abort
  controller even during batch tests. Test all runs the current selected list;
  stopping one account never aborts others or records a false failed result.
- Pelican history reactively refreshes after each run and retains its original
  prompt, raw output, HTML, error, and timestamp. Selecting a history entry
  exposes the full prompt/output and sandboxed animation preview; manual ratings
  must not replace the historical prompt with the current editor content.
- Pelican history and animation previews must remain usable on phones and the
  native Android WebView: history rows collapse to a card layout below `lg`,
  animation stays in an opaque-origin `allow-scripts` iframe, the generated SVG
  fills the available width while preserving aspect ratio, and mobile offers an
  explicit animation/source toggle so large generated source cannot crowd out
  the preview. Desktop retains the side-by-side history table/detail layout.
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
- The `余额` column refreshes balances on demand: every eligible `apikey`
  row shows an inline refresh button in the cell, and the bulk action bar
  exposes `批量刷新余额` for the selected scope. Both reuse
  `POST /admin/accounts/:id/balance-query`. The cell prefers
  `extra.balance_query.last_result` (unit-aware: USD `$`, CNY `¥`, quota raw,
  unlimited `∞`) over `extra.balance` whenever its `queried_at` is not older
  than `extra.balance_updated_at`; the periodic balance check writes
  `balance_updated_at` alongside `extra.balance` so the freshest source wins.
  Do not regress the column back to reading only `extra.balance`.
- `apikey` accounts also render the balance directly inside the `用量窗口`
  (usage window) cell in `AccountUsageCell.vue`: a compact `余额 <value>` chip
  plus an inline refresh button that calls the same
  `POST /admin/accounts/:id/balance-query` endpoint and re-emits the patched
  account via `account-updated` (wired to `patchAccountInList` in
  `AccountsView.vue`). Failed or unsupported queries render the chip as
  `余额 --` with an error tooltip instead of breaking the cell. The shared
  `formatAccountBalanceCell` helper supplies the value, so the usage-window
  chip and the `余额` column always agree; eligible apikey rows no longer
  render a bare `-` placeholder when no usage data exists.
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
- When an account currently holds concurrency slots (current_concurrency > 0,
  shown amber in the capacity column), its name in the name column also turns
  amber (text-amber-600 dark:text-amber-400) so the in-use account is easy to
  spot.
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
- On desktop, account actions remain single-line `24px` text buttons with
  visible labels and `px-2` padding inside a `220px` minimum operation column.
  Below 1024px the column is fixed at `72px` and each row exposes a single
  labeled actions trigger that opens `AccountActionMenu` inside
  `MobileActionSheet`. Rows can expose harvest, pelican-test, and state
  protection actions according to account capabilities; see `#cell-actions` in
  `AccountsView.vue`. Do not confuse the responsive top toolbar with this row
  action area; the dated audit
  [OPS_REVIEW_2026-09-30.md](OPS_REVIEW_2026-09-30.md) records the pre-change
  behavior and remaining verification risks.

  The mobile row-action follow-up uses a fixed 72px actions column below 1024px
  and a single labeled trigger. AccountActionMenu renders through
  MobileActionSheet on narrow screens, includes edit/test/harvest/lifecycle
  primary actions and every existing secondary action, and closes before
  dispatching downstream actions. Desktop direct actions remain unchanged;
  native Android still pins only selection. The actions column opts into
  Column.fixedWidth only below lg; captured page/sheet scroll events do not
  dismiss a mobile row-action sheet. The deployment helper now archives prior backups and creates one
  independent verified rollback per service; it does not delete prior rollback
  files.
- On desktop, the selection, actions, and name columns stay fixed on the left
  during horizontal scrolling. Their declared `36px`, `220px`, and `212px`
  widths provide cumulative offsets. Because dynamic columns can exceed those
  declared minima, verify actual widths and sticky overlap in the browser after
  action changes. The native Android table pins only the 36px selection column.
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
  declared rate -> scheduling rate -> 5h utilization (`5h(%)`) -> 5h reset. On desktop, the account table
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
  expensive binding cannot override a cheaper eligible account. This applies to
  both the advanced scheduler and the legacy load-batched sticky layer: a sticky
  account is reused only while its persisted rate ties the cheapest eligible
  account. Once a zero-rate account becomes eligible, the next request rebinds
  to it without changing the status of the previous account. Strict non-movable
  `previous_response_id` affinity remains intact because a response chain cannot
  safely change accounts.
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
cd /home/codes/ai/sub2api/backend && go test -tags unit ./internal/service -run 'Test(OpenAI.*LowestCost|FilterByAccountSchedulingPreference|OrderAccountsBySchedulingPreference|BuildOpenAISelectionOrder|SchedulingLiveness|SchedulingRate)' -count=1
cd /home/codes/ai/sub2api/frontend && pnpm vitest run src/components/account/__tests__/SchedulingRulesModal.spec.ts src/components/account/__tests__/SchedulingRateModal.spec.ts src/components/account/__tests__/SchedulingRateCell.spec.ts src/components/admin/account/__tests__/AccountTableActions.schedulingRules.spec.ts
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

### 测智历史动画预览

- 测智历史用结果区上方的单个下拉列表展示（`pelican-history-select`），不再有独立历史面板、详情栏或历史/自动分页签；选中一条记录会把该次结果原样还原进结果卡片，与刚测完时的卡片完全一致（状态徽章、结果摘要、规则证据、预览/源码/输出/明细标签、耗时与缓存时间）。
- 历史渲染复用实时结果的 `pelicanPreviewDocument` 与 `sandbox="allow-scripts"` 隔离；不得直接把生成 HTML 插入管理页 DOM。
- 普通文本/失败详情在卡片的输出/明细标签查看。切换历史记录重新加载对应预览，不发起上游请求；卡片底部的「用户评价」按账号 + 测试时间写回该条缓存。
- 测智过程中的流式输出、失败原因、提示词、历史原文和 HTML 源码完整换行显示，不使用 `...`、`line-clamp` 或固定最大高度截断。预览 iframe 仍保持视口高度。
- 旧缓存缺少 `has_html` 或结果结构时，从已有 HTML/SVG 原始输出（包括 Markdown 代码块）恢复预览；保留原始输出和提示词。
- `srcdoc` iframe 会继承宿主文档的 CSP。服务端 CSP 为 `script-src 'self' 'nonce-…'`（无 `unsafe-inline`），因此 `pelicanPreviewDocument` 必须把宿主 nonce（`script[nonce]` 的 `.nonce`）注入生成内容与自带适配脚本的每个 `<script>`；否则模型的内联动画脚本被静默拦截，预览只剩静态首帧。CSP 关闭时 nonce 为空，保持不注入。
- 本机缓存历史条目的状态以能力判定为准：`result.has_html && !downgraded` 即通过，旧缓存或流尾异常写入的 `failed` 在读取时纠正，避免历史列表把“未降智”标成红色失败。
- 临时不可调度账号的状态单元格直接显示三行堆叠详情：状态文字（可点击）/「剩余时间 约 N 分钟 · 解除时间 YYYY/MM/DD HH:mm」/错误详情，后两行单行截断并用 `title` 提供全文；点击状态文字仍打开完整弹窗（触发时间、错误码、关键词、规则序号、恢复状态）。外层容器必须用 `inline-grid`（不能用 `flex-col`），因为 `DataTable` 的 `single-line-cells` 会把 `[class~='flex-col']` 强制为 `flex-direction: row`，否则详情会被压成一行并把状态列撑到 600px 以上。
- 服务端检测历史（`account_detection_records`，含手动/定时/请求触发）必须能按生成时的样子回看：`DetectionHistory` 每条记录提供「查看结果」，展开 `DetectionRecordViewer` 的「动画预览 / HTML 源码 / 最终结果（原文）」三标签，复用 `extractPelicanPreviewHtml` + `pelicanPreviewDocument` 的 allow-scripts sandbox iframe；无 HTML 的记录只有原文标签。默认折叠，点击后才创建 iframe，避免一次渲染上百条动画。
- 回归：`frontend/src/components/admin/account/__tests__/AccountPelicanModal.spec.ts`、`frontend/src/utils/__tests__/pelicanPreviewDocument.spec.ts`、`frontend/src/modules/qualityevidence/__tests__/DetectionHistory.spec.ts`、`frontend/src/modules/qualityevidence/__tests__/DetectionRecordViewer.spec.ts`。

### 测智模型目录本地加载

- 鹈鹕测智的“测试大模型”下拉与账号模型目录只请求 `GET /admin/accounts/:id/models?catalog=local`：服务端跳过 OpenAI 上游模型发现（Codex manifest / `{base_url}/models`），直接返回本地目录（`model_mapping` 白名单或写死的默认模型集），单账号一次上游往返最慢会顶满 15s 超时。同样的 `catalog=local` 也用于测试连接弹窗（管理员 `admin/account/AccountTestModal.vue` 与用户侧 `account/AccountTestModal.vue`）和账号调度面板的模型选项（`AccountsView.vue` `handleSchedule`），避免弹窗打开时被上游目录请求阻塞；显式“同步上游支持的模型”仍走 `models/sync-upstream` 不变。
- 不带 `catalog` 参数的默认行为保持上游实时发现，供连接测试等弹窗继续使用；上游失败时仍回退本地目录。
- 批量测智只保留一个“测试大模型”组合控件：下拉选择本地目录或历史值，旁边输入框可手动输入任意模型 ID；非空输入会保存到 `sub2api:account-pelican-model-history:v1`，供后续下拉和输入建议列表使用。留空时内部按各账号本地目录的首个模型测试，不再在结果卡片中重复显示可编辑的“账号模型”控件。
- 历史模型保存到 `sub2api:account-pelican-model-history:v1`（最多 30 条，新的在前）：实际测过的模型（含留空时各账号自身目录选中的模型）都会写入，并在“测试大模型”下拉与输入建议列表中同时出现。
- 自由输入框使用页面内主题化的建议列表（`AccountTestModelPicker` 的 `pelican-model-suggestions`），不再使用原生 `<datalist>`：深色模式下浏览器原生 datalist 弹窗是系统黑底，与主题下拉不一致。支持聚焦列出、按输入过滤、方向键选择与回车填入；输入值本身始终可作为模型 ID 提交。
- “测试大模型”右侧的「清除历史模型」只清空本机这条历史（`clearPelicanModelHistory`），不动账号模型目录、不触发上游请求；没有历史时按钮不显示。
- `utils/pelicanModelCatalog.ts` 在模块级缓存每个账号已成功加载的本地目录：账号状态重建、弹窗重开和单次目录请求失败都不得让“测试大模型”下拉变空；失败时沿用上次目录，先用缓存渲染再由本次会话刷新。
- 回归：`backend/internal/handler/admin/account_handler_available_models_test.go`（LocalCatalogSkipsUpstream）、`frontend/src/components/admin/account/__tests__/AccountPelicanModal.spec.ts`。

### 测试连接手动触发

- 管理员测试连接弹窗（`admin/account/AccountTestModal.vue`）打开时只加载本地模型目录，不自动开始测试；移除了“自动测试”勾选框与 `sub2api.account-test.auto-start` localStorage 偏好，仅在点击“开始测试”按钮后发起探测。不要恢复弹窗打开即自动测试的行为。
- 测试进行中该按钮变为可点击的“停止测试”（`admin.accounts.stopTest`，红色 + x 图标），`stopTest` 通过 AbortController 中断流式探测并回到 idle，终端追加“测试已停止”（`admin.accounts.testStopped`）；用户侧 `account/AccountTestModal.vue` 同样如此。不要把连接中的按钮改回禁用态。
- 测试模型选择器开启 `searchable` + `creatable`：本地目录之外的模型 ID 可在搜索框输入后以“搜索 "xxx"”项选中，手动输入值直接进入 `model_id` 请求体。
- 管理员测试连接弹窗的鹈鹕测智模式复用批量鹈鹕测智的共享预设方案存储（`sub2api:account-pelican-prompt-scenarios:v1`）：打开弹窗时加载全部预设并默认选中上次使用的方案，「提示词方案」下拉选中后实时填充下方提示词文本框，并把选择同步回共享存储；手动编辑文本框即切换为「自定义」选项，按编辑后的内容直接发起测试，不要求先保存方案。不要把该下拉还原成写死的默认提示词。
- 回归：`frontend/src/components/admin/account/__tests__/AccountTestModal.spec.ts`（打开不自动测试、点击停止、测智预设选择与自定义输入用例）。

### 测智历史账号隔离

- 历史下拉只列出当前测试卡片里的账号记录，不按 OpenAI/API Key 类型、名称或模型合并账号；同名账号仍独立。
- 每条历史的选项文案是“测试题目前 24 个字符 · 返回模型 · 判定结果 · 测试时间”，不显示账号名（账号已在对应结果卡片里）；选中后只还原对应账号的卡片，其他账号保持不变。
- 取消选择账号后，该账号的记录同步从下拉移除，空选择不显示其他缓存；保留全部原始缓存及动画预览，切换不删除历史、不触发上游请求。

### 账号检测中心与测智检测记录

- 账号管理页提供统一的「账号检测」入口，弹窗内保留「立即检测」「定时检测」「请求降智检测」「检测记录」「测智规则」页签。立即检测复用原有连接测试和鹈鹕批量测智弹窗；定时检测与规则库复用 `ScheduledTestsPanel`；请求降智检测复用共享账号、模型选择控件，自动请求只选择一道测智规则。不要把这几套能力重新拆成互不相关的入口。
- 自动测智配置明确区分 `model_scope=selected` 和 `all`；新配置默认全部模型，显式 selected 的空列表不检测任何模型。每个命中请求受 `reuse_seconds` 复用窗口约束：窗口内同账号、同模型、同一道题已通过则直接放行，失败/异常/取消不复用；`reuse_seconds=0` 恢复每次请求独立测智。`allow_business_success` / `recent_minutes` 仅兼容旧客户端。详见后文「每次请求并发降智检测」。
- 测智检测记录与历史证明按「账号 ID + 模型 ID + 规则内容指纹」隔离并持久化到 `account_detection_records`；规则提示词、判定器或参数变更后旧证明自动失效。同一账号、模型和规则的新失败记录会使较早的成功证明失效，且查询和证明匹配均忽略模型 ID 大小写；这些历史证明不参与新请求的放行。
- 连接/传输失败、用户停止和超时不等于测智不通过：这些结果分别保存为 `failed`、`cancelled`，不写入 `pelican_downgraded`，也不清除既有降智标记。只有探测请求完整返回且规则判定失败时才标记降智；定时任务对非 success 结果不再改写降智状态或触发自动恢复。
- 每个账号最多保留 200 条检测记录，超出后自动清理最旧记录。管理端检测记录接口按账号、来源和模型过滤，并用 `before_id` 游标分页；历史中的 HTML 动画必须先提取再进入 sandbox iframe，不允许把普通文本或半截 HTML 当作动画预览。
- 回归：`scripts/verify-account-detection.sh`，其中包含 repository 的临时 PostgreSQL 集成测试，覆盖证明隔离、最近失败覆盖旧成功、记录分页和 200 条保留上限。

## OpenAI Basis Points Plugin (installed 2026-09-28)

- The `.s2plugin` source module is
  `github.com/Wei-Shaw/sub2api-oai-basispoints` (built against
  `github.com/Wei-Shaw/sub2api` via a local `../../backend` replace); the
  corresponding GitHub repository is not publicly reachable as of
  2026-09-28, so treat the signed package as a vendor-distributed artifact.
  The in-repo `backend/internal/modules/basispoints` package is a separate
  source port of the same protocol from https://github.com/hloolx/codex2api;
  `JaxsonWang/cpa-plugin-oai-basispoints` is only a review reference and uses
  the incompatible CPA plugin ABI.
- `local.oai-basispoints` v0.6.12 is installed on the main profile
  (upgraded 2026-10-05 from v0.6.9; same plugin row, signature
  `trusted`, artifact SHA-256 `76bca1d51ccb23ea8fef2cea3668a9bb85e25c66d8f10bd8db9bcfdd56da76ae`).
  The plugin is currently **disabled**: the admin disabled it on 2026-10-01
  (audit `admin.plugins.disable.create`), so the `openai.oauth.outbound_transport.v1`
  binding (platform `openai`, account_type `oauth`) is off and OAuth traffic
  bypasses Basis Points. Re-enabling requires
  `POST /api/v1/admin/plugins/5/enable` with `{"accept_untested":true,"rollout_percent":100}`
  because the manifest does not declare host 0.2.13 as tested (required range
  `>=0.2.8 <0.3.0`). Upstream releases live at
  `zhu961212/sub2api-oai-basispoints` (module `wangyunjeff/...`). The earlier
  `sub2api.openai.basispoints` v0.6.0 and `local.flownode.state-reuse` remain
  installed but disabled. The host allows only one transport plugin at a
  time — enabling one while another holds the route fails, and two enabled
  transport bindings poison the OAuth route. Do not re-enable state-reuse or
  the v0.6.0 package while `local.oai-basispoints` is active.
  Upgrading an already-enabled same-key plugin first requires disabling it
  (`PluginManager.Install` refuses otherwise); config survives the upgrade
  (`config_encrypted` is not rewritten), but bindings reset to disabled and
  must be re-enabled.
- v0.6.9 is a different plugin_key than v0.6.0 (`local.oai-basispoints` vs
  `sub2api.openai.basispoints`), so upgrades install as a separate plugin row
  with fresh default config; per-plugin config does not carry over.
- The `local.oai-basispoints` packages (v0.6.9 and v0.6.12) are signed by the
  official publisher key
  `oai-basispoints-v1`; its public key is registered in
  `deploy/data/config.yaml` as
  `plugins.trusted_publishers."oai-basispoints-v1"`, alongside the local key
  `local-oai-basispoints-20260928` used to sign the unsigned v0.6.0 vendor
  package. `plugins.allow_unsigned` stays `false`. Keep both entries when
  editing or regenerating the deploy config; config is read only at process
  start, so changes require a service restart.
- The local signing private key lives at
  `/root/sub2api-plugin-keys/oai-basispoints-20260928.private` (mode 600) for
  re-signing packages that ship unsigned; prefer upstream's officially signed
  releases when available.
- v0.6.9 defaults forward only `gpt-6-astra` and `gpt-5.6-sol` through Basis
  Points; other OAuth models pass through direct until selected in the
  plugin config UI (`enabled_models`). It also exposes per-account BPS
  toggles, auto-degradation detection tasks, and attachment handling — all
  manageable from the plugin page UI.
- Plugin egress follows each account's bound proxy. Most OpenAI accounts are
  bound to the LAN mihomo gateway `192.168.3.2:7890` (this host). On
  2026-09-29 the `1024-GPT` load-balance chain
  (`B2Proxy -> us.1024proxy.io`) intermittently rejected OpenAI traffic, so
  `/etc/mihomo/config.yaml` openai/chatgpt rules were repointed to the
  `Codex-GPT` select group (currently `日本-静态住宅-B2Proxy`); the group's
  health-check URL was also switched to `https://api.openai.com/v1/models` so
  OpenAI-dead nodes are excluded. To restore the 1024 exits, select
  `1024-GPT` inside `Codex-GPT` via the mihomo API (port 9091; obtain authentication from the protected runtime configuration, never from documentation) — no config edit needed. Backups:
  `config.yaml.bak-20260929-openai-healthcheck` (original),
  `config.yaml.bak-20260929-openai-rules-codexgpt`.

### 自定义测智提示词判定（2026-10-15）

- 管理员测试连接的自定义普通问答不附加 HTML 指令，不按 HTML/SVG 缺失判定降智。自动模式仅对文档/鹈鹕动画提示词使用原 HTML 判定；保留显式 HTML、完整回复与数字答案选项。
- 完整回复通过必须说明尚未校验答案正确性；数字答案必须由操作者填写预期整数，不能暗用定时糖果探针的 21。默认/定时 HTML 测试保持原契约。
- 回归：`account_test_service_custom_prompt_test.go`、`AccountTestModal.spec.ts`。

### Claude 测智流完整性（2026-10-15）

- Claude 自定义测智请求使用 16384 输出 token 预算，普通连接测试保留原预算。
- Claude 测智流累积文本并发送共享判定的 `pelican_result`，不再只发送连接完成事件；保留实际返回模型。
- `max_tokens` 截断、异常停止及缺少完成信号的断流显示具体错误并保留输出，能力未判定，不标记降智。处理 EOF 同时返回的最后一行。
- 回归：`TestClaudeProbeStreamCompletion`、`TestClaudeProbeOutputBudget`。

### 暂停账号立即中止在途转发（2026-10-15）

- 管理员单行暂停、批量暂停和编辑为不可调度状态，在持久化成功后取消该账号的在途推理转发；恢复调度允许新请求进入，不复活已取消的请求。运行时取消注册表按服务进程隔离。
- HTTP、流式和 OpenAI WebSocket 转发使用账号尝试级取消上下文；客户端原始上下文保持有效，流式脱离客户端取消时仍必须响应管理员暂停。
- 尚未输出正文的失败交还现有 failover 排除账号并按原策略重选，不做同账号重试，不改写原有严格 previous_response_id 亲和策略。已输出正文时结束响应，不能拼接另一账号的内容。
- 人工暂停不算账号健康失败；注册与暂停互斥，避免已经选中旧账号的并发请求在暂停后继续发送。回归：`TestAccountPauseCancelsOnlySelectedAttemptAndDetachedUpstream`、`TestAccountPauseInterruptsWaitingHTTPAndRestoresClientContext`。

### 测智结果预览与源码标签（2026-10-15）

- 测智结果卡片在内容上方固定显示「预览」「HTML 源码」两个标签，切到长源码后仍能从顶部返回预览；不得恢复底部单按钮切换。
- 历史结果与放大预览也保留两个明确入口；历史记录切换时默认返回动画预览，普通文本结果继续完整显示。
- 回归：`AccountPelicanModal.spec.ts` 的完成后反复切换用例。

### 管理员日志中心（2026-10-15）

- 侧边栏「日志中心」直达 `/admin/logs`，提供「模型失败日志」与「系统日志」页签；复用现有受管理员权限保护的 ops 接口与详情组件，不要求操作者查询数据库。
- 模型失败默认展示最近 24 小时下游请求失败，使用 `view=all` 保留参数错误、认证失败和限流等统计中排除的业务限制；可切换到上游失败尝试，追溯故障转移前的失败账号。模型（精确 ID）、账号 ID、状态码、关键词与分页筛选均传到后端，上游列表同样应用模型筛选。
- 失败列表保留账号名、模型映射、状态码、请求 ID 和错误详情；「关联系统日志」按服务请求 ID（缺失时按客户端请求 ID）及失败时间前后 5 分钟查询，历史失败不受系统列表默认最近 1 小时限制，时间转换保留原时区含义。
- 未开启采集时明确提示；接口读取失败显示错误，不伪装成空列表。主/免费服务分别读取各自日志数据，原运维监控与日志配置入口保留。
- 回归：`scripts/verify-log-center.sh`，使用内存 mock，不依赖 Docker 或在线数据库。

### 路由失败的账号归属与选号诊断（2026-10-18）

- `account_model` 复合路由的 OpenAI 模型可用性诊断必须遵守调度器相同的精确公开模型归属检查。通配符、空映射和透传的通用支持不能代替精确归属；已配置的精确归属账号处于临时冷却时仍保留模型支持，不能误报 404。显式路由继续使用原来的模型支持语义。
- 账号为空的路由失败显示“未选定账号”并提示查看详情或关联系统日志；其他缺失归属显示“未记录账号”。已有账号 ID 的 503 必须继续显示对应账号，不按状态码抹除归属，也不为历史空值推测账号。
- 调度器通过显式 `RoutingFailureSummary` 错误接口提供候选池大小与筛除原因计数。`routingdiagnostics` 模块只将此摘要附加到日志保存副本的 `error_body.routing_diagnostics`，不向调用客户端发送，不冒充上游 HTTP 错误，不存储任意错误串。管理端详情将“选号诊断”和客户端响应分开展示。历史记录继续通过原请求 ID 关联已有系统日志，不回写虚构信息。
- 定向回归：`bash scripts/verify-routing-diagnostics.sh`，覆盖模型归属、冷却、显式路由、选号错误传播、保存副本与客户端响应隔离，以及日志列表与详情展示。

## 统一账号测试工作台（2026-10-15）

- 账号行和菜单只保留一个“账号测试”按钮，顶部和批量入口同样统一为“账号测试”；移除单独的鹈鹕入口。所有入口挂载同一个工作台（`AccountPelicanModal.vue`），单账号与批量共用模型配置、请求执行、停止、结果及历史。
- 不区分连接／能力标签；统一选择测试题目、判定方式和思考程度，共用执行与结果。连接题、数字题、鹈鹕 HTML 动画和自定义题目只是不同题目。题库通过服务端 degradation-rules CRUD 保存，同一服务的安卓与电脑共用；旧浏览器本地题目首次打开自动迁移，缓存的服务端 ID 不重新上传，避免删除后复活。主／免费服务数据库仍隔离。新增手动题目默认不启用自动探测。
- 连接测试保留 OpenAI Compact 与 OpenAI/Gemini 图片能力；只根据明确的 test_complete 成功事件判定连接成功。流尾无换行的事件仍处理；无完成事件的 EOF 判为失败。普通连接测试不写入降智标记。
- 结果保留预览、HTML 源码、原始输出、详情标签，按响应内容展示；HTML iframe 继续使用 allow-scripts sandbox。历史新增 testMode 元数据；旧历史按能力测智解释，连接历史显示连接成功而非未降智。
- 单账号仍通过确认流程恢复状态或标记失败；批量结果提供每账号明确操作，避免并发完成时相互覆盖确认弹窗。批量降智标记收进更多菜单。
- 默认打开不发请求，不隐式选中第一个账号；模型目录只读取本地配置。运行中锁定配置和账号选择，单账号停止互不影响，关闭弹窗后保留既有后台执行行为。
- 回归位于 AccountPelicanModal.spec.ts、AccountActionMenu.mobile.spec.ts、AccountsView.testRecovery.spec.ts、AccountsView.recycleDelete.spec.ts、AccountsView.lite.spec.ts、AccountsView.mobileToolbar.spec.ts；缓存旧格式仍兼容。

### 每次请求并发降智检测（2026-10-16）

- 使用日志的“请求时测智参考”必须按每行账号、模型和记录时间独立查询 `finished_at <= created_at` 的最新能力检测，后续检测不得改变旧请求参考。相同账号/模型的不同请求时间不得合并；没有请求时间或当时没有检测时不展示最新检测兜底。检测仅作为参考，请求成功/失败以 HTTP 状态列为准；账号管理的降智状态仍显示最新近期检测。
- 账号表“降智状态”支持按最新能力检测结果进行服务端全结果集排序，遵循配置的近期窗口；升序依次为疑似降智、检测失败、取消、跳过、通过，降序反转，无记录/过期/未来记录始终排后。同时间检测按更大的记录 ID 取最新，普通连接测试不参与；保留原有固定账号和置顶规则。
- 质量门丢弃的每个业务账号尝试必须立即作为 `request_quality_probe_failed` 错误出现在使用日志（全部/错误筛选），保留请求 ID、账号、模型、端点和本次拒绝原因；后续换号成功不得丢失前面的记录。复用使用日志现有错误数据源，拒绝尝试不调用计费/额度扣减，也不重复计为独立客户端 SLA 失败；客户端取消不伪装成质量失败。最终请求结果仍单独记录。
- 客户端在响应返回前断开（访问日志 499）的请求仍会排空上游并按上游用量计费，但使用日志状态列必须显示 `499 客户端断开`，不得回退成 `200 成功`。终态由 `usage_logs.client_disconnected`（迁移 246）持久化：OpenAI `/v1/responses` 流在检测到下游写入失败后把 `ClientDisconnect` 带回转发结果，记账时写入该列；DTO 暴露 `client_disconnected` 并把状态映射为 `client_disconnected` / 499。该行仍是计费行（`isRowError` 为假，Token 与费用照常显示），也不写入质量失败错误行（客户端取消不伪装成质量失败）。此时同请求的测智记录通常是 `cancelled`（检测取消，未判定）。
- 账号管理顶部仅保留「全局降智设置」入口；总开关位于设置弹窗内，桌面与手机均可操作。读取失败不得伪装成关闭。调度规则仅提供设置跳转，不再读取或顺带覆盖测智配置。完整设置统一位于「账号检测 → 请求降智检测」，检测记录查看请求触发来源。
- 默认关闭，默认检测范围为全部账号、全部模型；旧的显式范围保留。启用后，命中范围的业务账号尝试同时启动业务与一道独立测智请求，使用同一账号和模型，遵循账号原有上游地址、代理和模型映射。自动检测最多选一道规则（未选用内置数学题，旧多选配置取第一道）。同一账号、模型和规则内容在 `reuse_seconds`（默认 300 秒，0 为关闭）内已有通过记录时，本次请求直接放行且不发送探测；只复用通过结果，失败、异常、取消以及规则内容变更均不复用，仍逐个窗口独立探测，也不复用并发请求的探测结果。业务成功和旧免测字段仍不参与放行。
- HTTP 非流式及 SSE 原响应的正文和响应头均在本次测智通过前暂存；通过后放行已暂存内容并继续正常流式返回。内存最多暂存 256 KiB，较大响应写入 0600 且创建后立即 unlink 的临时文件，结束时关闭，不截断内容。暂存阶段的 Flush 不阻塞原业务请求发往上游。检测失败丢弃该账号原响应，交由原有有界 failover；只有规则完整判定失败才写 `pelican_downgraded=true`，网络/取消/超时不写降智状态。
- OpenAI WebSocket 的 ctx_pool/http_bridge/passthrough 入口逐轮独立检测，所有业务事件在该轮测智通过后才写给客户端；同一轮多帧共享判定。连接的 previous_response_id 亲和与既有重连/故障转移边界保持原约束，不将失败账号正文拼接给客户端。
- 关闭后，新请求直接使用原 writer/forwarder，不启动探测或增加响应等待；已启动的账号尝试完成自身判定。该开关不启动后台全账号扫描，也不改变既有手动/定时检测任务和历史降智标记。
- 复用窗口按成功记录的 `rule_fingerprint` 和 `finished_at` 判定，读取失败时回退为探测而不是放行；窗口内出现更新失败记录立即恢复探测。前端 `RequestQualityProbeForm` 暴露秒数（0–3600），账号/模型/规则范围只决定哪些请求参与，不缩短窗口。
- 回归：`TestRequestQuality*`（handler/service，含 `TestRequestQualityProbeReuse*`）、`TestAccountHandler.*Quality`、WebSocket ingress/passthrough 回归；`RequestQualityControl.spec.ts`、`AccountDetectionCenter.spec.ts`、`SchedulingRulesModal.spec.ts`、账号页与移动入口回归。

### 账号行测试入口回退（2026-10-17）

- 操作列显示“账号测试”，点击仅打开统一工作台，需手动开始。移除 autoStart 属性及打开后自动执行定时器；存在历史缓存也不得自动调用上游。

### 手动测试 UI 整合（2026-10-17）

- 批量操作只保留“账号测试”，移除绕过工作台直接执行的“测试并标记”按钮；单行、顶部、批量、检测中心均打开同一工作台。
- 通过试题模板（连接问答、数字答案、HTML/SVG 动画、图片生成、自定义）填入同一题目编辑器，统一手动开始、停止、结果和历史；模板切换不发上游请求。数字模板同时填入期望答案，图片请求采用编辑后的同一提示词。Compact 仅在高级设置保留协议检查。

### 上游账号错误后继续切换（2026-10-17）

- OpenAI 兼容推理上游的 HTTP 404（模型或端点不存在，包括纯文本 `404 page not found`）进入既有有界账号切换流程；API Key Responses 透传采用相同策略。上下文超限、安全策略和普通请求参数错误仍保留原分类。
- Responses、Messages 和 Chat Completions 路径在账号模型映射后发现 GPT 工具调用不兼容当前 Chat-only 账号时，受调度器管理的调用返回账号尝试失败信号，不先提交 400 响应；原循环排除当前账号并选择下一候选。独立单账号调用仍直接返回原有错误，显式允许 Chat 工具的中转账号仍可使用。
- 保留候选分组、模型归属、限流、严格 continuation 亲和与切换次数限制；已输出正文后不得拼接另一账号的响应。没有符合条件的候选时正常结束并报告错误。
- 回归：`TestAccountErrorFailover*`、`TestShouldFailoverUpstreamError*`、`TestGPT61Sol*`、`TestOpenAIStreamingResponseFailed*`、`TestNonStreamingSSEToJSON*`。

### 2026-10-17 手动测试整合验收证据

- 回退直接测智：51 项定向回归通过，部署 `004804-18debef697f8f6fe` 完成。
- 统一工作台：80 项定向前端回归通过（`005330-18debef697f8f701`）；数字、连接、动画模板按题目/判定方式发送，图片使用相同编辑器。后续故障转移合同及49项工作台回归经 `005950-18debef697f8f708` 复验通过。
- 首次部署 `005355-18debef697f8f702` 在后端测试阶段停止，未安装。修正映射推理工具测试需显式 high 才命中能力检查的夹具，并将 OAuth 搜索404旧透传预期对齐通用故障转移合同；未跳过检查。
- 最终部署 `010332-18debef697f8f70b` 01:09:14完成；构建固定快照 `3087484a0453e766fdab4feb5ae56baf386bd002a1f48f153f31da6f68a57870`，工作区期间另有推进，不将新工作区整体宣称已发布。
- 集中日志 `/home/bin/webclx/compile/runs/20261017T010337-548365-25958/build-1.log`，审计同目录 `install-report-1.json`。完整后端套件、TypeScript与Vite生产构建通过；主站/免费站实际HTTP账号模块 `AccountsView-b4Xw5-qy.js` 和中英文模块核验：试题模板/高级设置存在、重复批量按钮/autoStart/直接测智不存在。
- 两端磁盘及运行文件SHA-256均为 `f3a37014ff43e3e2adceb0293bee48b07480b9127e24ad8c7193b88f8a130c0f`，main/free端口、数据库和Redis隔离验证通过。没有发送真实模型测试请求。
- 备份各保留1个：main `sub2api.bak.20261017-010905-642982`，free `sub2freeApi.bak.20261017-010905-642982`；旧备份各归档1个，删除0个。

### 使用记录降智证据（2026-10-17）

- 管理端使用记录默认显示“降智证据”列，按账号与转发模型（与质量门禁使用的渠道解析后、账号映射前模型一致）精确匹配最新能力检测，显示检测时间、检测序号和结果；检测中心记录展示相同序号。普通连接测试不充当降智证据；超时、取消、跳过为未判定。
- 开关与适用范围按当前配置独立计算：关闭且命中范围显示“降智检测关闭”，开启但不命中显示“不需要”，两者均不满足显示“关闭且不需要”。历史检测仍可同时展示，不声称历史业务请求经过该次检测。开启且暂无记录显示“暂无检测记录”；读取异常明确显示失败。
- 后端 `internal/modules/qualityevidence` 批量查询当前页去重的账号/模型组合；前端 `src/modules/qualityevidence` 管理显示契约。不得用其他账号、其他模型或单纯业务成功代替证据。

### 账号降智检测开关（2026-10-17）

- 账号表格原“降智”列更名“降智检测”，保留原列 key 以兼容列偏好，使用与调度列一致的单账号开关，控制账号参与检测而非历史降智结论。
- 单账号设置保存至全局配置 `account_overrides`，覆盖账号范围（全部／异常／指定），不改变总开关或模型范围。全局关闭时明确标注；账号开关不能绕过总开关和模型范围。
- 全局降智设置显示单独设置的账号，可切换或恢复跟随账号范围。多个账号同时保存串行读取最新配置，保留其他账号设置；同页读取合并，避免每行独立请求。
- 原顶部总开关移除，总开关统一在全局降智设置内保存；使用记录降智证据复用相同范围判定。
- 降智设置保存路径对超出当前上限的历史值做收敛（`setting_request_quality_probe.go` 的 `normalizeRequestQualityProbeSettings`：`timeout_seconds`、`recent_minutes`、`status_recent_minutes`、`reuse_seconds`）。此前库中存在 `timeout_seconds=600`（上限 120），导致每次 PUT 都被校验以 400 拒绝、账号降智开关无法切换；读取路径同样归一化后回显值合法，切换恢复。写入校验仍拒绝越界的新值。
- 账号编辑弹窗（`EditAccountModal.vue` 状态区块下方）提供同一个降智检测开关（`frontend/src/modules/qualitycontrol/AccountQualitySwitch.vue`）：读取当前有效参与状态，切换即写入 `account_overrides` 并立即生效，不走账号保存流程；存在覆盖时提供“跟随账号范围”清除覆盖。读取失败显示读取失败且禁用开关，保存失败提示错误并保留原状态。

### 测智管理入口归一化（2026-10-17）

- 顶部保留“账号测试”“测智管理”“调度规则”；移除重复“全局降智设置”和全局“定时测试管理”按钮，移动功能菜单使用同一入口。
- 测智管理由 `frontend/src/modules/qualitycontrol/QualityManagementDialog.vue` 管理，旧 AccountDetectionCenter 为兼容适配器。仅保留手动检测与检测情况。自动功能使用独立“自动检测”按钮和 `AutomaticDetectionDialog.vue` 对话框；桌面工具栏及移动功能菜单均有入口。
- 手动检测复用统一工作台，打开和选账号均不自动发送上游请求；行内和批量账号测试快捷入口保留。
- 独立自动检测对话框包含自动策略（状态恢复、周期测智、费率）、每次请求、定时执行、共享试题。账号选择独立加载全部普通账号，不受当前页及表格筛选限制。请求总开关、账号/模型范围、超时、单账号覆盖继续存原设置；定时任务继续存原计划，不互相覆盖。切换自动子标签保留尚未保存的请求配置。
- 单账号菜单定时测试也打开同一管理窗口的定时区域，保留账号和本地模型目录上下文。流量调度保持独立，不改变实际调度和任务执行逻辑。
- 检测情况展示账号参与配置与历史降智标记，明确其非所有模型的即时判定；历史保留时间、序号、来源、模型和结果。加载失败明确显示错误，不伪装成空记录或已关闭。

## Instance Appearance (2026-10-17)

- Site settings expose independently persisted `instance_name` and `instance_color`.
  They belong to each runtime's existing settings database and are never bundled
  into the shared release binary. Partial/older settings writes preserve them;
  explicit empty values restore automatic identity by access address/profile.
- The `instanceappearance` feature modules own normalization, palette, preview,
  and reusable badges. Colors are fixed palette keys; names are limited to 48
  Unicode characters. Preserve the adapters in the settings DTO/public/embedded
  configuration pipelines across upstream upgrades.
- Identity appears as a fixed 5px top strip, tinted sidebar header, labeled badge
  (including mobile), document-title prefix, and matching local SVG favicon.
  Existing site logos, version badge, status colors and table layout are retained.
- Settings location: global settings → site settings → instance appearance.
  Runtime main/free and AWS should each retain their own labels/colors on upgrade.

账号管理的“降智检测”为账号参与开关，紧邻的“降智状态”列显示下述真实检测证据，不再将人工能力标记作为该列依据。关闭检测不清除历史记录。全局开关明确仅控制业务请求触发检测，仍需账号参与和模型范围同时命中；不控制手动与定时检测。

“降智状态”以真实能力检测记录为准，按当前页账号批量查询最新记录（包括手动、定时和请求触发；排除纯连接测试），不使用人工 `pelican_downgraded` 标记替代检测证据。`request_quality_probe_settings.status_recent_minutes` 默认 5 分钟、允许 1–1440；这是独立的展示窗口，不复用已废弃的免测字段，不影响每请求独立检测。窗口字段在全局开关关闭时仍可设置。状态区分近期无测试、正常、疑似降智、失败、中止、跳过及读取失败；超时不能显示未降智。悬停显示最新记录序号、时间、模型、来源、试题及回答摘要；新规则检测保存实际试题摘要，旧记录缺失摘要时明确说明，不用当前规则内容冒充历史试题。页面每 30 秒批量刷新证据并更新时效，不额外访问上游。

使用记录“降智证据”与账号“降智状态”复用 qualityevidence 模块的能力检测筛选条件和最新排序（finished_at DESC、id DESC），前端复用同一记录类型及 qualityVerdict 判定。相同检测序号的成功、疑似降智、失败、中止和跳过结论必须一致；失败不能因 downgraded=true 被误判为降智。取值范围保留差异：使用记录按账号及请求模型取最新，账号状态按账号跨模型取最新并应用近期窗口；使用记录不因窗口过期丢弃证据。

### 轮转进度与近期成功提醒（2026-10-17）

- 最终失败保留自定义错误码：本轮仅发生质量门拒绝时返回 `quality_check_failed`；质量拒绝与其他上游账号错误并存时返回 `account_attempts_exhausted`，消息分别统计两类账号数（同账号重试不重复计数）。两者均使用 HTTP 502；已提交 SSE 时使用合法的流内终止事件。普通上游失败维持原有错误映射，客户端取消和请求参数错误不改成质量失败。质量检测未通过可包含检测异常，不等同于业务回答已被判错，具体原因仍查逐账号 `request_quality_probe_failed` 记录。
- Responses 的 `error.code`、Chat/Messages 的 `error.type` 与 `error.code`、Ops 最终错误分类应保留上述标识，不能再统一显示为 `Upstream service temporarily unavailable` / `api_error`。本地质量门产生的 502 不得作为真实上游 HTTP 状态填入日志；真实上游错误仍保留。近期成功提醒追加在具体原因之后，不覆盖原因，不改写历史记录。

- 单个下游请求只遍历一轮候选账号。质量拒绝禁止同账号重试；候选耗尽后不因503清空失败账号集合重开一轮。保留取消终止与当前尝试内的其他有界瞬态重试；日志应区分独立HTTP请求、独立测智与最终汇总。

- 管理端账号测试须将上游 `: x` / `: ---` 转为独立 `routing_progress` 测试事件，并在统一测试面板和单账号测试弹窗显示独立进度区；成功与失败后保留进度。不得混入模型答案、判题输入或工具参数。普通上游失败进入换号时也逐账号写 `upstream_account_attempt_failed` 使用记录；质量门已记录的拒绝不重复写。

- OpenAI Responses、Messages、兼容 Responses/Chat Completions 的流式账号切换发送 SSE 注释 `: x`（当前账号尝试失败）和 `: ---`（下一不同账号已获得执行机会）。同账号内部重试不冒充账号切换，选不出下一账号不提前发送 `---`，客户端取消停止通知。标准 SSE 解析器会忽略注释，客户端如需界面可见进度应单独解析这些注释；符号不注入模型正文、工具参数或计费 Token。
- 进度不阻止后续账号切换；发出进度后终止失败必须使用协议对应的流内错误，Responses 发 `response.failed`，禁止在已提交 SSE 后拼接 JSON 错误。使用日志继续按最终失败而非传输层已提交的 HTTP 200 记账。
- 非流式接口保持单个 JSON 响应，不为符号擅自改成流式。最终服务器失败时，必须先有本请求实际测智完成（status=success 且 downgraded=true）的证据，再仅查询这些被判降智账号及其转发模型在请求开始前五分钟内的业务成功记录；仅匹配时附加复核建议。普通 502、连接拒绝、超时、测智异常或其他账号的成功记录均不触发提醒。请求开始后的并发成功、其他账号/模型、普通连接测试不能触发提醒；数据库失败不推测成功。提醒不绕过质量门，不确定降智发生时间，也不声称历史操作已确认降智。
- 实现位于 `internal/modules/{rotationnotice,qualityevidence}`，handler 和 repository 仅注册/适配。定向回归覆盖流式全失败/后续工具调用成功、取消、非流式 JSON、历史窗口与消息幂等。

### 账号测智流式思考与最终结果（2026-10-17）

- 统一账号测试工作台运行时默认进入“流式输出”，独立展示上游明确返回的思考内容和回答原文、字符数及输出状态；不根据回答伪造思考过程。流式区域保持有界滚动，向上查看时暂停跟随，可显式恢复跟随并复制两部分。
- “最终结果（原文）”仅在输出结束后展示完成回答；停止、异常与无完成事件保留片段在流式页并提示未完成。HTML/SVG 预览、源码与放大继续复用隔离 iframe，不从思考内容提取预览。历史缓存兼容旧记录，新增 reasoning 独立字段。
- Chat Completions reasoning_content/reasoning、Responses reasoning_summary_text/reasoning_text delta、Claude thinking delta、Gemini thought parts 以 reasoning 事件发送，content 只携带回答。思考事件不进入历史回答提取或数字答案正文。
- 回归：TestAccountTestReasoningStreamSeparation、AccountPelicanModal.spec.ts 的流式中间态、最终原文、历史及异常片段用例。
- 验收：定向后端分流测试通过；前端 60 项回归及类型检查通过（webClx `175117-18df47e79b347ba2`）。统一双服务构建部署 `175232-18df485fb17d6a5d` 成功，main/free HTTP 200，两个磁盘文件和运行进程 SHA-256 均为 `22858a427812fcbd6ea317d2f355bee3a875c19da929745404235097ddc72be8`。浏览器验收 `180224-18df48cff52da375` 在两服务实际前端资源上通过流式中间态、最终结果、HTML/源码及手机布局，所有 API 使用 mock，不调用真实模型。每服务保留备份 1 份，共归档 2 份、删除 0 份。

### 账号测试移除方案模板（2026-10-17）

- 移除“方案模板”下拉框、“选择模板填入提示词”及其整行说明；不再维护模板填入处理器。测试题目仍通过共享题库和统一编辑器选择、编辑，判定方式与手动开始流程保持可用。此项替代上述手动测试整合中的模板入口约定。
- 移除模型选择下方的覆盖说明，将腾出的空间用于题目编辑器：默认五行、最小高度 112px，允许纵向调整至 240px。

### 使用记录账号筛选与图表精简（2026-10-17）

- 编辑账号搜索文字立即解除旧账号 ID，重新选择后以所选账号 ID 筛选；候选搜索、聚焦加载及路由名称回填使用请求代次校验，迟到响应不得覆盖当前输入或候选。选择、清除、外部回填和卸载均取消待执行搜索。
- 使用记录页隐藏分组使用分布与端点分布，模型使用分布与 Token 用量趋势在桌面同一行、窄屏上下排列；不再请求隐藏图表的分组统计。
- 回归：UsageFilters.spec.ts、UsageView.spec.ts。
- 状态为全部且按账号等维度精确统计时，错误记录独立 COUNT 的 SQL 参数从 `$1` 重新编号，不复用 UNION 分支偏移。列表请求失败后清空旧明细并提示错误，避免将旧账号结果误认为新筛选结果；取消的旧请求不得清除新结果。回归包含 `TestListCombinedUsageLogsAccountCountRestartsPlaceholders`。


## 指定自动测智试题作为唯一降智标准（2026-10-18）

- 账号测试对话框提供“设为自动测智试题”，保存 `request_quality_probe_settings.rule_ids`
  的唯一试题，不自动开启业务请求检测。先保存题目，临时改题不得指定或复用。
- 账号当前降智状态、全局排序、使用记录历史关联与定时测试自动标记/暂停，仅认可指定
  试题身份及实际题面、判定器、答案/关键词一致的结果。手动执行同一已保存试题也认可；
  其他试题仅保留测试历史，不得覆盖标准结果。全局检测开关不影响证据认定。
- 结果保存 `quality_question` 快照；没有快照的旧记录不猜测认定，不删除历史。
  更换标准或修改题面/答案后，旧内容不能作为当前标准的证据。
- 未指定试题时，自动请求检测跳过，不拦截业务，也不得隐式回退到内置糖果题。每次请求质量门仍执行本次
  独立测试，不以手动历史结果免除本次测试；账号/使用记录两处统一复用标准结果。
- 回归：`TestAutomaticQualityQuestionOnly`、`TestAccountStatusSortPostgres`（CTE 只读夹具），
  `AccountPelicanModal.spec.ts` 的试题指定用例，以及 qualityevidence 模块测试。

## 统一自动检测策略与所有/指定账号范围（2026-10-18）

- `账号测试 -> 自动 -> 自动策略` 统一提供状态检测恢复、周期测智及费率探测配置；原有按账号 Cron 计划、业务请求测智与试题库继续可达。
- 状态/测智策略支持 `account_scope=all|selected`。all 每轮从完整普通账号池动态匹配，覆盖后续新增账号；selected 严格使用 `account_ids`。账号范围、平台与目标状态共同生效。状态筛选支持 active/inactive/error 以及暂停调度、限流/临时阻断、疑似降智（任一匹配）。暂停调度另需显式 `include_paused`，默认排除；归档、回收站、旧软删除和过期账号始终排除。
- 成功后的清理错误/运行时阻断、激活停用账号及启用调度分别配置。连接成功不得解除疑似降智；测智恢复及连续失败暂停仅接受当前指定标准试题的有效结果。周期测智仅支持 OpenAI，未指定标准试题拒绝启用；不更改业务请求测智开关。改变标准题面后重新累计连续失败。
- 策略独立于最低成本/default 调度模式。旧 liveness 的 error/API Key/lowest_cost 资格保持原样；自动周期执行时让位于新策略覆盖的账号。旧按账号计划对被新策略覆盖的账号仅推进时间，不重复发送探针；策略范围外按原计划执行。显式手动刷新仍可调用原检测能力。
- 后端所有权为 `internal/modules/autodetection`，前端扩展现有 `src/modules/qualitycontrol`。核心 service/repository/handler 仅提供测试、恢复、持久化与接线适配。管理端 `GET|PUT /api/v1/admin/accounts/automatic-detection` 沿用管理员鉴权。
- 配置持久化到本服务 settings 的 `account_auto_detection_policies`；逐账号下次执行依据、连续失败、最近 20 轮汇总和最近 100 条结果保存在 `account_auto_detection_runtime`。复用定时执行器生命周期，每分钟检查、并发 4、每轮最多 20 个账号，按最久未检测优先分批；同账号同轮不并发运行两种策略。PostgreSQL 会话级 advisory lock 按数据库隔离，进程退出自动释放。每次探针有超时，执行轮有整体上限；重启不丢间隔状态。
- 探针完成后重新读取策略修订号和账号状态/调度开关，变化时跳过自动动作。最终状态/调度写入按 `updated_at` 比较更新，与 scheduler_outbox 事件同语句原子提交，并同步共享调度快照；不得以无缓存同步的裸 SQL 或整个账号覆盖替代。
- 费率配置增量支持 `account_scope=all|selected` 及 `account_ids`。缺省 scope 继续沿用旧逐账号探测开关；显式范围动态覆盖这些探测开关，只针对合格 active API Key 账号，SQL 在 LIMIT 前完成筛选。费率探测仍复用原执行器/快照与手动锁定规则，不承担状态恢复。原调度规则弹窗保存其他设置时保留新增 scope 字段。
- 新状态/测智策略默认关闭，部署本身不创建真实上游探测任务、不改已有账号状态。main/free 的配置、运行记录及执行锁均不共享。
- 定向回归：`go test -race ./internal/modules/autodetection`；`TestAutomaticDetectionStateCASAndOutbox`、`TestUpstreamBillingProbePolicyScope`、`TestAccountRepositoryListDueScopedUpstreamBillingProbeAccounts`；`AutomaticDetectionPolicies.spec.ts`、`QualityManagementDialog.spec.ts` 与 `SchedulingRulesModal.spec.ts`。

- 交付验收（2026-10-18）：webClx 部署请求 `103800-18df48cff52da3f7` 完成，全量前端 434 文件 / 3328 测试通过（1 跳过），后端全量及定向检查通过。main/free 首页均 HTTP 200，自动策略管理接口未登录均 HTTP 401；两个磁盘文件和两个运行进程 SHA-256 均为 `c9f808251679cff3c38fa3a40d40c5a1ee2fe66bb49065f71ea1d99e49ba392a`。各保留 `sub2api.bak.20261018-104511-208049` / `sub2freeApi.bak.20261018-104511-208049` 一份，旧备份共归档 2 份、删除 0 份。
- 浏览器验收请求 `104734-18df48cff52da408` 通过：对已部署 main/free 资源分别验证 1440px 中文、390px 中文、320px 英文，共六组；覆盖所有/指定账号、选择校验、恢复选项保存、无标准试题禁止开启周期测智、费率范围保存和视口布局。管理 API 全模拟，未借验收启用真实生产策略或修改账号。首次失败来自新手引导遮罩；测试夹具已按现有用户引导已读键消除干扰，保留普通点击与布局断言。

## Pool relay transport restart tolerance (2026-10-18)

- OpenAI pool-mode accounts default to at least three persistent network failures
  spanning 15 seconds before the existing ten-minute transport cooldown applies.
  A gap over 60 seconds starts a new streak; a successful real upstream request
  clears all preceding evidence. Counters are process-local and bounded.
- Confirmation uses subsequent real requests, with ordinary failover on each
  failure; it does not sleep, replay requests, or generate billable background
  probes. Explicit proxy authentication rejection still pauses immediately.
- The dedicated guard owns these network failures; the rolling HTTP health
  breaker must not count the synthetic 502 and bypass confirmation. Other HTTP
  errors retain their existing configured policies. Pool retry count remains a
  per-request limit, independent of this confirmation window.
- Implementation: `backend/internal/modules/transportguard` with service adapters
  in `openai_upstream_transport_error.go` and `openai_account_scheduler.go`.
  Preserve module timing/concurrency/reset tests and handler integration tests.

- Account create/edit dialogs expose the confirmation switch and count/span/gap
  under the Temp Unschedulable heading, independent of HTTP keyword rules. The
  values persist in `credentials.pool_transport_guard` (`enabled`, `min_failures`,
  `span_seconds`, `max_gap_seconds`). Defaults are true/3/15/60. Disabling
  confirmation restores immediate durable-network cooldown; success still resets
  evidence. Changes discard pending evidence on the next failure. UI code lives
  in `frontend/src/modules/transportguard`; malformed backend values fall back
  to defaults. The existing ten-minute cooldown duration remains unchanged.

## Pool mode consecutive error cooldown (2026-10-28)

- Pool-mode accounts still skip account-level state on individual upstream
  errors, but the skip now feeds a Redis consecutive-error counter
  (pool_mode_error_count:account:<id>, 30-minute window). At 5 consecutive
  errors the account enters a 10-minute temporary cooldown
  (temp_unschedulable) instead of staying a permanently failing first pick
  under lowest_cost scheduling. The counter resets on any successful real
  upstream request, on ClearRateLimit/recover-state, and when the cooldown is
  applied. Counter or Redis failures preserve the legacy pool-mode skip.
- Explicit temp-unschedulable keyword rules still take precedence, and custom
  error code accounts keep their configured policies. Permanent status=error
  remains off-limits for pool mode: upstream errors do not prove local
  credential invalidity.
- Implementation: service/pool_mode_error_counter.go,
  repository/pool_mode_error_counter_cache.go, the pool-mode branch of
  RateLimitService.HandleUpstreamError, and success reset in
  openai_gateway_usage.go. Unit tests:
  ratelimit_service_pool_mode_error_counter_test.go.

## Server detection history in account tests and logs (2026-10-19)

- Account-test History includes persisted manual, scheduled and request-triggered
  records using the same DetectionHistory component as the Logs detection tab.
  Browser output cache is separately labelled and clearing it never removes
  server records. History shows evidence ID, source, finish time, model, reason,
  question snapshot when available and raw response, with cursor pagination.
- Probe transport failure is inconclusive capability evidence; a later passing
  manual test does not erase earlier failed or downgraded attempts.
- Automatic retention only trims passing records. Failed, cancelled, skipped
  and downgraded records must not be evicted by later successful probes.
  Records removed by the previous retention policy cannot be reconstructed from
  current state; no synthetic historical successes/failures are inserted.
- Regression: DetectionHistory.spec.ts, AccountPelicanModal.spec.ts,
  LogsView.spec.ts and TestAccountDetectionRepositorySaveBoundsHistoryPerAccount.

## Probe billing failure classification (2026-10-19)

- Preserve HTTP probe failures as typed status/body evidence through in-memory
  test SSE capture. A probe HTTP 402 enters the existing RateLimitService billing
  branch immediately, retaining provider-specific account blocking and cache
  invalidation; do not infer billing failure from a model answer containing 402.
- Stop remaining test questions after payment-required, persist a failed,
  inconclusive detection record, and carry 402/body through the response gate to
  normal account failover and attempt logging. Do not wrap it as quality 502,
  set a downgrade flag, or trigger a recent-success downgrade warning.
- Regression: TestAccountDetectionPaymentRequiredPreservesBillingFailure,
  TestRequestQualityProbePaymentRequiredIsNotQualityRejection and
  TestRequestQualityGatePreservesPaymentRequired.

## Upstream forbidden details and probe classification (2026-10-19)

- Preserve typed probe HTTP 403 through detection history, quality gate and normal
  account rotation logging, as with HTTP 402. Stop remaining probe questions;
  permission denial is inconclusive capability evidence, not a quality rejection
  or downgrade. A model-specific 403 must not enter the billing handler.
- At failover exhaustion, ordinary upstream 403 returns permission_error with
  `Upstream HTTP 403` and the extracted, sanitized upstream message (including
  model/group restrictions). Missing structured messages use a fixed fallback;
  raw HTML or response bodies are not returned. Existing explicit passthrough
  rules and credential-specific safe responses retain precedence.
- Apply to Responses, Messages and Chat protocol adapters. Once SSE headers have
  committed, retain HTTP 200 and send the final protocol error in-stream; record
  intended 403 rather than pretending the committed HTTP status can be changed.
- Regression: TestAccountDetectionForbiddenPreservesPermissionFailure,
  TestRequestQualityProbeForbiddenIsNotQualityRejection,
  TestRequestQualityGatePreservesForbidden and TestUpstreamForbidden*.

## Detection provenance in usage and history (2026-10-20)

- Admin usage includes a separate server detection-history tab, reusing the same
  component/API as account tests and Logs. Local probes do not become business
  successes, user charges, or recent-success evidence merely to display them.
  Token/cost data not collected by probes is unknown, not zero.
- New result JSON stores `origin`: trigger (`manual_connection`, `manual_quality`,
  `scheduled_connection`, `scheduled_quality`, `request_quality`), main/free profile,
  source account ID, Cron plan ID when applicable, and automatic/explicit model
  choice for policy/plan execution. Broad manual/scheduled/request filters remain
  compatible. Legacy records explicitly lack detailed origin; do not infer it
  from current settings or rewrite historical evidence.
- API-key account tests append `Sub2API-Detection/1` provenance to their outgoing
  User-Agent; existing client identity is retained. OAuth identities are unchanged.
  A receiving Sub2API instance displays the client-declared source beneath the
  model in usage and error tables using the already persisted User-Agent. This
  hint never controls authorization, billing, probe participation or scheduling.
  Incoming cross-instance probes remain normal billable API calls on that instance.
- Regression: `TestDetectionOrigin*`, `DetectionOriginLabel.spec.ts`,
  `DetectionHistory.spec.ts`, and admin `UsageView.spec.ts`.

## 账号测试题目下拉单行摘要（2026-10-26）

- 统一账号测试的题目下拉复用共享 `Select`，选项使用空白归一化、最多 28 字符的题干摘要，超长省略并保持单行；Android 不再使用会展开完整题干的原生选项列表。
- 选项值仍为原题目 ID，选中后的题目编辑区和实际测试请求保留完整题干。
