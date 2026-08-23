# AGENTS.md - sub2api

Local deployment rules for this project.

## Unified Source Ownership

- `/home/third_party/sub2api` is the only source and build directory for both
  `sub2api.service` and `sub2freeApi.service`.
- Build the embedded frontend/server once and install the same artifact bytes
  to both existing executable paths.
- `sub2api.service` must set `DEPLOYMENT_PROFILE=main`;
  `sub2freeApi.service` must set `DEPLOYMENT_PROFILE=free`.
- `/home/third_party/sub2freeApi` remains the free runtime/config directory and
  a historical recovery source. Do not build a second binary from it.
- Source consolidation never permits database, Redis, config, user, working
  directory, port, or service lifecycle consolidation.

## Service Identity

- Project path: `/home/third_party/sub2api`
- Systemd unit: `sub2api.service`
- Runtime user/group: `sub2api:sub2api`
- Working directory: `/home/third_party/sub2api/deploy`
- Environment file: `/home/third_party/sub2api/deploy/sub2api.env`
- Executable path: `/home/third_party/bin/sub2api/sub2api`
- Writable data path: `/home/third_party/sub2api/deploy/data`
- HTTP port: `18381`

## Android Client

- The maintained native WebView client lives in `android/`; its release entrypoint is
  `scripts/build-sub2api-android-apk.sh`.
- `android/VERSION` is the authoritative Android release version. Keep it monotonic
  against the published update manifest; do not derive it from the independently
  versioned backend service.
- Cold start must probe every entry in `SourceRegistry.URLS` concurrently and load the
  first valid health response immediately. Do not serialize a remembered source before
  the race, add a fixed delay, or show automatic source-switch Toasts/countdowns.
- Keep a healthy selected source stable. Network callbacks verify the current source in
  the background; only a confirmed failure may trigger a silent race of the remaining
  sources. If runtime recovery finds no source, preserve the current WebView.
- Main-frame failures must match the selected origin and current navigation generation;
  coalesce network callbacks while resolution is already in flight.
- The native client UA contains `sub2apiAndroid/<version>`. On the account-management
  page this client must use the same dense table layout as desktop, with horizontal
  scrolling and the existing sticky leading columns. Ordinary mobile browsers retain
  the generic card layout.
- Update endpoints remain paired by source index in `SourceRegistry.UPDATE_URLS` and are
  raced concurrently. The Sub2API service on port `18381` does not expose the webClx
  artifact API, so do not replace the verified `11111`/`11112` update endpoints without
  first adding and testing such a proxy.
- Preserve the existing `sub2api` signing key and explicitly enable APK Signature Scheme
  v1 and v2. Never generate a replacement key for an existing application ID.
- Run `node --test android/tests/client-contract.test.mjs` after Android changes.
- The Android client was previously removed wholesale while reverting an unrelated
  virtual-mouse experiment. Revert a feature at file/hunk scope; never delete `android/`
  merely to remove one client behavior.
- Preserve the Android contract documented in
  [docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md) across
  upstream merges and record future behavior changes there.

## Data Isolation

This service must use the main account database/config:

- PostgreSQL database: `sub2api`
- Redis DB: `0`
- Config file: `/home/third_party/sub2api/deploy/data/config.yaml`

Do not point this service at `sub2freeApi`, Redis DB `1`, or `/home/third_party/bin/sub2freeApi/sub2freeApi`.

## Upgrade Procedure

- Canonical step-by-step runbook: [docs/UPGRADE_RUNBOOK.md](docs/UPGRADE_RUNBOOK.md) (`/home/third_party/sub2api/docs/UPGRADE_RUNBOOK.md`).
- Behavioral preservation checklist: [docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md).
- Read both documents before fetching, merging, resolving conflicts, building, deploying, or pushing an upgrade.
- Merge only `upstream/main`. `origin/main` is the customized fork backup and is only a push target.
- Never use the WebUI binary updater for this customized deployment.
- README sponsor advertisements are intentionally excluded. After every merge,
  run `bash scripts/remove-readme-sponsors.sh` and verify with
  `bash scripts/remove-readme-sponsors.sh --check`; do not restore sponsor
  sections in `README.md`, `README_CN.md`, or `README_JA.md`.
- Preserve the local enhanced-edition highlights near the top of all three
  README files and keep their upgrade warning pointed at this source-based
  runbook. Do not restore instructions that tell customized-fork operators to
  use the WebUI binary updater.

## Deploy Checklist

- Unless the user explicitly requests no deployment, finish every code-change task by automatically running the unified dual-service build, deployment, and live verification before the final response.
- Routine code-change deployments use `bash scripts/build-unified-release.sh`
  with no mode flag. Its default `quick` mode reuses Go test-cache results,
  skips the duplicate full Vitest run, and still performs the exact-snapshot
  backend suite, TypeScript typecheck, Vite production build, embedded Go
  build, source-integrity checks, and install audit. Run task-focused backend
  and frontend tests before queueing this fast deployment. The frozen build
  copy uses the stable, `flock`-guarded
  `/data/cargo-target/sub2api-unified-source` workspace so unchanged Go test
  results can be reused across deployments; source hashes before and after
  synchronization still reject mixed or concurrent source state. Go tests also
  override webClx's per-run `TMPDIR` with the stable, lock-protected
  `/data/cargo-target/sub2api-unified-go-test-tmp`; changing that path defeats
  result-cache reuse for tests that call `t.TempDir` or `os.TempDir`.
- Use `bash scripts/build-unified-release.sh --full` only for upstream merges,
  dependency/toolchain changes, migrations, security/auth changes, broad
  cross-module refactors, or when focused regression coverage cannot be
  identified. Full mode forces uncached backend tests and the complete Vitest
  suite. Do not make full mode the automatic default again.
- `bash scripts/build-unified-release.sh --print-plan` is the non-building
  mode audit. Unknown or conflicting build-mode arguments must fail closed.

1. Build the canonical frontend and `-tags embed` Go binary exactly once.
2. Atomically install the same bytes to `/home/third_party/bin/sub2api/sub2api`
   and `/home/third_party/bin/sub2freeApi/sub2freeApi`, retaining independent backups.
3. Keep main on database `sub2api`, Redis DB `0`, port `18381`, profile `main`;
   keep free on database `sub2freeApi`, Redis DB `1`, scheduler prefix
   `sub2freeApi`, port `18382`, profile `free`.
4. Restart and verify `sub2api.service` first, then `sub2freeApi.service`. Never
   restart both in one command.
5. Confirm both disk files and both `/proc/<pid>/exe` files have the same SHA-256.
6. Verify:

```bash
systemctl status sub2api.service --no-pager
ss -ltnp | rg ':18381'
curl -fsS http://127.0.0.1:18381/ >/dev/null
systemctl status sub2freeApi.service --no-pager
ss -ltnp | rg ':18382'
curl -fsS http://127.0.0.1:18382/ >/dev/null
```

## Local Customizations To Preserve

- Keep all three README files free of sponsor advertisements while continuing
  to merge upstream functional documentation, license text, and project credits.
- Admin account credentials intentionally display `credentials.api_key` in plaintext for account edit/inspection workflows.
- Do not add `api_key` back to `SensitiveCredentialKeys`; OAuth tokens, session keys, cookies, AWS secrets, service account JSON, and private keys must remain redacted.
- `MergePreservingSensitiveCreds` must preserve an existing `api_key` when an older or partial frontend update omits it.
- `frontend/src/components/account/EditAccountModal.vue` should preload `credentials.api_key` and render API key inputs as `type="text"` for account API key fields.
- OpenAI API Key accounts with a custom `base_url` must treat an upstream 401
  `token_invalidated` code, or the canonical `authentication token has been
  invalidated` message when the relay omits that code, as the relay's internal
  OAuth failure: fail over the request without changing local `status` or
  `schedulable`. This exception does not apply to OAuth accounts,
  `invalid_api_key`, default OpenAI endpoints, or an administrator's explicit
  custom 401 error-code policy.
- **Account management table column layout** (`frontend/src/views/admin/AccountsView.vue` `allColumns`):
  - The leading order is `选择` -> `操作` -> `名称` -> `容量` -> `状态` -> `调度` -> `用量窗口` -> `平台/类型`.
  - After `创建时间`, keep `今日费用` -> `累计费用` -> `分组` (when visible) -> `余额` -> `5h请求` -> `5h Token` -> `7d请求` -> `7d Token` -> `窗口总费用`.
  - After `今日统计`, keep `7d(%)` -> `7d`; the ending order is `过期时间` -> `备注` -> `账号ID` -> `上游声明费率` -> `调度倍率` -> `5h(%)` -> `5h`.
  - The utilization headers use the compact labels `5h(%)` and `7d(%)`.
  - `名称` (name) column has explicit `width: '212px'` (20% wider than the former 176px); its inner content is capped at `212px × 32px`. The account name and supplemental email share one left-to-right text flow with the name first, so a long email can never squeeze out the name prefix. The combined text wraps naturally to at most two 16px lines and truncates only at the end of the second line without growing the table row or escaping the cell.
  - Do not append the supplemental email when the account name already ends with that same email, comparing case-insensitively after trimming surrounding whitespace.
  - The selection column has explicit `width: '36px'`, and `DataTable.vue` keeps `--select-col-width` at `36px`.
  - `状态` (status) has explicit `width: '80px'` as its minimum width on the account table.
  - Table headers, labels, sort indicators, and desktop cell content remain single-line and non-shrinking.
  - `AccountsView.vue` enables `DataTable`'s `single-line-cells` and `dynamic-column-widths` modes. In this opt-in mode, declared `column.width` values are minimum widths and other content may expand columns, while the name cell keeps its explicit `212px` cap; the table scrolls horizontally when necessary.
  - The selection, operation, and name columns stay fixed on the left while the account table scrolls horizontally, using their declared `36px`, `220px`, and `212px` widths for cumulative offsets.
  - Other `DataTable` consumers retain the default fixed-width behavior where declared widths apply `width`, `minWidth`, and `maxWidth`.
  - The first and last table cells use `4px` outer padding so the table has no unnecessary edge whitespace.
  - Non-final columns retain 1px vertical separators in light and dark mode.
  - `平台/类型` (platform_type) column has explicit `width: '170px'` as its minimum width.
  - `账号ID` (id) column has explicit `width: '130px'` as its minimum width.
  - The `Column` interface in `frontend/src/components/common/types.ts` has an optional `width?: string` property.
  - The `DataTable` component in `frontend/src/components/common/DataTable.vue` applies `column.width` as `width` + `minWidth` + `maxWidth` by default, and as `minWidth` only when `dynamicColumnWidths` is enabled.
  - `AccountsView.vue` enables `DataTable`'s `compact-rows` mode. Desktop loading and data cells use `py-0.5` (2px top/bottom padding); other tables retain the default spacing.
  - Direct account actions remain single-line `h-6` text buttons with visible labels, `px-2` horizontal padding, and an explicit `220px` operation-column minimum so all four actions remain reachable without relying on tooltips.
  - Do NOT revert these columns to their upstream positions or remove the width properties.
- Preserve every local commit and dirty file during upgrades. Follow [docs/UPGRADE_RUNBOOK.md](docs/UPGRADE_RUNBOOK.md) and audit [docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md); do not use the WebUI binary updater for this customized build.
- **OpenAI sticky-session concurrency spillover must be preserved**:
  - A historical `session_hash` binding is honored only while the bound account can acquire a real `account.Concurrency` slot.
  - When that account is full, route the current connection through the normal same-group selection path to another available account, even if the session remains historically bound to the full account.
  - Do not rewrite the historical sticky binding for a one-connection overflow; if every eligible account is full, retain the normal bounded wait-plan fallback.
  - Concurrency-full spillover is mandatory even when TTFT/error health escape is disabled. Strict, non-movable `previous_response_id` affinity remains unchanged.
  - Preserve the implementation in `openai_gateway_scheduling.go` and `openai_account_scheduler.go` plus the spillover regression tests in their corresponding `*_test.go` files.
- **Lowest-cost scheduling and rate synchronization must be preserved**:
  - Global scheduling has only `default` and `lowest_cost`; legacy `extra.super_priority` and mode values never affect routing or status display.
  - `accounts.rate_multiplier` is the sole lowest-cost ranking value. Successful automatic probes copy only `resolved_rate_multiplier` into it; peak/effective snapshots do not rank requests.
  - `extra.scheduling_rate_sync_mode` is `auto_overwrite` (default) or `manual_lock`. Legacy upstream/manual source values map to those modes only when the new field is absent.
  - In `lowest_cost`, the bounded-concurrency recovery probe never tests healthy active accounts or manually paused accounts. It retries only `status=error` API Key accounts; a real successful upstream request clears recoverable error/rate-limit/runtime state and restores `schedulable=true`. OAuth and other account types remain excluded.
  - Legacy `super_priority.liveness_include_unschedulable` remains accepted for configuration compatibility but does not expand the recovery probe to healthy or manually paused accounts.
  - Periodic upstream billing/rate probing defaults to disabled so healthy accounts do not receive background upstream requests. Administrators may explicitly opt in through the scheduling-rules dialog when automatic rate synchronization is required.
  - The scheduling-rules dialog shows server-owned liveness runtime state in its upper-right header: running state, next expected batch time, and the latest checked/succeeded/failed/skipped result. Keep this sourced from the backend runner rather than a browser-only schedule.
  - The account table marks the current optimal scheduling rate in gold only for accounts that are schedulable and tie for the lowest `rate_multiplier` in at least one scheduling group. All tied minima are marked; ungrouped accounts compare only with ungrouped accounts on the same platform. The calculation uses the full active account pool, not the current page, and is an administrative hint rather than a guarantee for every model-specific request.
- After upstream upgrades, verify with:

```bash
cd /home/third_party/sub2api/backend && go test -tags unit ./internal/handler/dto ./internal/service
cd /home/third_party/sub2api/frontend && pnpm vitest run src/components/account/__tests__/EditAccountModal.spec.ts
```

## Account Usage Window Table Contract

- `UsageProgressBar.vue` stays compact: each usage-window row shows only its
  label, progress/utilization, and reset state. Do not put request, token, `A`,
  or `U` totals back inside the progress bar.
- `AccountsView.vue` exposes separate columns for `5h请求`, `5h Token`,
  `7d请求`, `7d Token`, `5h(%)`, `5h重置`, `7d(%)`, `7d重置`, and
  `窗口总费用`. The cost column renders separate 5h/7d lines with upstream
  (`A`) and user (`U`) values when available.
- When a window contains valid `window_stats`, zero requests/tokens/cost are
  rendered as `0`; use `-` only when the window or its statistics are genuinely
  missing. New or lightly used accounts must therefore expose the same complete
  field structure as older accounts after usage has been loaded.
- Utilization and reset columns remain locally sortable. Accounts without
  loaded usage sort after accounts with data, regardless of direction.
- OpenAI idle windows at `0%` display `现在`; an expired positive-utilization
  window displays `待刷新`; a future reset displays a countdown.
- `AccountUsageCell.vue` emits `usage-loaded`, and that payload is the single
  source for the parent table's request/token/utilization/reset/cost columns.
  Usage auto-load remains active-only; explicit single-row or bulk refresh may
  query inactive/error accounts.
- `批量更新额度` stays immediately before `批量更新` in
  `AccountBulkActionsBar.vue`:
  - with selected accounts, query only that selection; with no selection, query
    only the accounts loaded on the current page; never expand a zero-selection
    action to later filtered pages;
  - query only account types that expose the single-row active `查询` action
    (OpenAI OAuth and Anthropic OAuth/Setup Token);
  - call `/usage` with `source=active&force=true`, at a maximum concurrency of
    four; continue after per-account failures and report success/failure counts;
  - apply each successful response immediately to the usage cell and every
    separate usage-window column; while running, disable both primary bulk
    buttons.
- Focused regression verification:

```bash
cd /home/third_party/sub2api/frontend
pnpm vitest run \
  src/components/account/__tests__/UsageProgressBar.spec.ts \
  src/components/account/__tests__/AccountUsageCell.spec.ts \
  src/components/admin/account/__tests__/AccountBulkActionsBar.usageRefresh.spec.ts \
  src/utils/__tests__/usageWindowDisplay.spec.ts \
  src/utils/__tests__/batchAccountUsageRefresh.spec.ts \
  src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts \
  src/views/admin/__tests__/AccountsView.bulkUsageRefresh.spec.ts
```

## Enhanced Import Mixed-Message Contract

- Pasted-text enhanced import accepts a single JSON value, a JSON array, or
  mixed chat/forwarded/Markdown text containing multiple complete JSON values.
- Extraction must use a string-aware balanced object/array scanner. Nested
  values, braces inside quoted strings, escaped quotes, and escaped backslashes
  must not split a JSON value. Do not replace it with a greedy regex.
- Each extracted value is normalized and validated independently. Segment
  errors use the one-based source label `pasted JSON #N`, and all valid
  segments are merged into one import API request in source order.
- Reject text with no complete JSON value and reject truncated outer JSON;
  never import an inner array/object from an incomplete enclosing value.
- Keep pure-JSON and multi-file modes compatible. Preserve import routing
  defaults (last proxy, first group), operator overrides, and
  `skip_default_group_bind: true`.
- The text-mode UI keeps the bilingual usage guide and extraction summary.
  Never log or persist pasted credentials/tokens outside the import request.
- The account tools menu keeps `批量清除导入账号` directly below `增强导入`.
  It opens the enhanced parser in a dedicated clear mode without import routing
  controls; the normal enhanced-import modal does not show the destructive
  action. Clear mode reuses the same file/text parser, sends only normalized
  account data to a read-only database match preview before confirmation, and
  never removes imported proxies. The confirmation reports both parsed and
  matched account counts. By default, matching scans ordinary and staged
  accounts and moves matches to recoverable deleted staging. An unchecked-by-
  default `彻底删除` checkbox expands matching to the deleted-staging repository
  and permanently deletes every match regardless of its current lifecycle
  level. The irreversible confirmation must state that those accounts cannot
  be recovered. The backend matches platform/type plus a
  stable credential identity, permits a name-only fallback only when unique,
  and uses the protected permanent-delete service for irreversible clears.
  Missing or ambiguous matches must not be guessed.
- Focused regression verification:

```bash
cd /home/third_party/sub2api/frontend
pnpm vitest run \
  src/components/admin/account/__tests__/enhancedImport.spec.ts \
  src/components/admin/account/__tests__/EnhancedImportDataModal.spec.ts
```

## Account Staging / Recoverable Deleted Staging

- **Bulk account editing is direct and change-tracked**:
  - Value controls in `BulkEditAccountModal.vue` remain usable before their field checkbox is selected.
  - Changing a value automatically selects only that field for submission; untouched field defaults must not enter the bulk-update payload.
  - Field checkboxes remain available for explicit operations whose desired value equals the form default, including clearing a proxy, groups, mappings, or other existing account values.
- **Account staging** (formerly "recycle") is an extra filter via `extra.recycled = true`. It does NOT use soft-delete (`deleted_at`). The filter toggle is labeled "归档" (Archive) with an `archive` icon in `AccountTableActions.vue`.
- `accountListFilteredQuery` in `account_repo.go` accepts separate `recycled` and `deleted` modes. They are mutually exclusive; ordinary lists exclude both lifecycle states.
- All callers of `ListAccounts` must pass both lifecycle booleans. `ListWithFilters` and `ListAllWithFilters` continue to accept `recycled` and always exclude deleted-staging rows.
- Backend staging routes: `POST /api/v1/admin/accounts/:id/recycle` and `POST /api/v1/admin/accounts/:id/restore`.
- Frontend: `AccountTableActions.vue` has an archive toggle button (archive icon); `AccountsView.vue` shows 归档/取消归档 buttons in the action column depending on mode.
- Active account rows keep the direct action order `编辑` -> `测试连接` -> `归档` -> `更多`; `测试连接` is not duplicated inside `AccountActionMenu.vue`.
- **Recoverable deleted staging** is the second lifecycle level via `extra.deleted = true`; new deletes must not set `deleted_at`:
  - `DELETE /api/v1/admin/accounts/:id` moves the row into deleted staging, removes `extra.recycled`, and preserves credentials, groups, usage/history, and scheduled configuration.
  - Ordinary and recycled lists exclude deleted-staging rows. `GET /api/v1/admin/accounts?deleted=1` lists only deleted-staging rows; `deleted=1` and `recycled=1` together are rejected.
  - The trash icon in `AccountTableActions.vue` toggles the deleted-staging table. Rows retain `编辑`, `测试连接`, `恢复`, and `更多`; restore uses `POST /api/v1/admin/accounts/:id/restore-from-trash`.
  - The bulk-selection controls keep a compact filter input beside `本页全选` / `全选所有结果` / `清除选择`. It binds to the server-side account search and remains available in ordinary, staged, and deleted-staging views. Search covers account ID, name, notes, platform, type, status, and error message; credentials and tokens must not be placed in the GET search query.
  - Deleted-staging rows are never request-routing candidates and must be excluded from OAuth refresh, scheduled tests, liveness, upstream-rate/billing probes, automatic usage/balance refresh, expiration auto-pause, scheduler-score pools, and model-availability scans. Explicit operator management remains available.
  - Ordinary edits must preserve the server-owned `deleted` marker. Duplicating a deleted/staged row must not copy either lifecycle marker to the new account.
  - Migration `191_convert_account_soft_delete_to_deleted_staging.sql` restores legacy soft-deleted rows and saved group bindings, clears `deleted_at`, and marks them `extra.deleted=true`.
  - The permanent-delete repository, service, and API remain protected so they accept only deleted-staging or legacy soft-deleted rows. The deleted-staging (回收站) account-management view exposes `彻底删除` both as an explicitly confirmed per-row action inside `AccountActionMenu.vue` (via the `permanent-delete` event) and as a batch action for selected rows including all-result selections; ordinary and first-level archived views must not expose it.
- `AccountTestModal.vue` defaults `自动测试` to enabled, waits for the default model to load before starting, and persists checkbox changes in browser storage under `sub2api.account-test.auto-start`.
- After a successful direct connection test, an account whose status is not `active` or whose scheduling is paused prompts the operator before recovery. Confirmation performs full runtime-state recovery, activates an inactive account when needed, and enables scheduling; already active and schedulable accounts do not prompt.
- After a definite failed direct connection test, an account that is not already `error` prompts the operator whether to mark it as failed. Confirming uses the dedicated `mark-failed` API to set `status=error`, save the test error message, and disable scheduling; already-error accounts only refresh. Automatic failure-window observations remain unchanged and do not write `status=error` before their configured threshold.
- Account names stay inside the fixed-width, fixed-height name cell with two-line truncation and overflow clipping. Do not restore the name-triggered hover tooltip that teleports content outside the cell.
- sub2freeApi has an additional `clone` function in `accounts.ts` that sub2api does not — re-add it when syncing files.
- Status/Groups/Capacity table cells use plain text (text-color classes only), NOT badge/card styling. See `references/account-table-column-layout.md`.
- **Usage auto-load skips non-active accounts**: `AccountUsageCell.vue` `shouldAutoLoadUsageOnMount` must gate on `props.account.status === 'active'`. Accounts with status `inactive` or `error` do NOT auto-fetch `/usage` on page mount, avoiding useless upstream queries against known-unavailable accounts. Manual refresh (via `usageManualRefreshToken`) remains unaffected for all statuses.
- **OpenAI quota auto-pause is displayed as quota limiting**: admin account responses expose the request-time `quota_rate_limit` decision from the same service logic used by scheduling. `AccountStatusIndicator.vue` shows `额度限流` and its reset time, and an enabled scheduling toggle uses an amber limited state while the account is excluded. Do not persist this derived state into `status`, `schedulable`, or `rate_limit_reset_at`; window recovery must remain automatic.

## Account Actions And Active Sticky-Session Reassignment

- New-account form initialization and reset use `concurrency: 10`. Enhanced CLIProxyAPI imports also default new accounts to `concurrency: 10`; native backup imports preserve each account's exported concurrency.
- New-account form initialization and reset select the last available proxy and the first available group. If proxy/group props arrive after the modal opens, fill only still-empty selections; never overwrite an operator's existing choice. Empty candidate lists remain unassigned.
- After advancing to the OAuth authorization step, `CreateAccountModal.vue` keeps the account name entered on step one visible at the top of step two. Email addresses commonly used as account names must remain visible while the operator completes OpenAI Auth.
- `AccountActionMenu.vue` always displays `恢复状态` for every account.
- The bottom of `AccountActionMenu.vue` keeps the low-frequency actions in this order: `查看统计` -> `创建 Spark 影子账号` (when available) -> `设置隐私` (when available). Do not move them back above the recovery, scheduling, quota-reset, or delete actions.
- The action menu keeps `w-[7.8rem] max-h-[calc(100vh-1rem)] overflow-y-auto`; `AccountsView.vue` keeps a `125` px width estimate and a `320` px height estimate for viewport positioning.
- `迁入粘性会话` appears for active, schedulable OpenAI target accounts and defaults to bindings active in the last 5 minutes. Allowed windows are 1, 5, 15, and 60 minutes.
- Its frontend capability flag is opt-out while public settings are loading: an explicit `sticky_session_reassignment_enabled: false` still hides it for the free profile, but a temporarily missing field must not make the main-profile action disappear.
- The dialog shows recent/all counts and anonymous recent suffixes. Move at most 100, newest first, and revalidate the activity window on the backend.
- Activity age is derived from configured sticky TTL minus Redis PTTL. Migrations use compare-and-set plus `SET ... KEEPTTL`.
- Only current 16-character lowercase-hex `session_hash` keys move. Ignore legacy 64-character copies and never move `response:` / `previous_response_id` continuation bindings.
- `Concurrency limit exceeded for user` is a local sub2api caller-concurrency timeout, not an upstream provider message. Sticky concentration can consume caller slots while requests wait on one upstream account, so moving recent bindings to spare capacity can help. Do not direct this operation to `用户管理`.
- Preserve the API routes and tests documented in `.codex/skills/sub2api-account-modal-enhancer/references/sticky-session-reassignment.md`.

## Scheduled Account State Actions

- `账号管理 -> 更多` always exposes `定时启用并恢复` and `定时暂停调度`.
- The dialog accepts whole hours plus `0..59` minutes. The combined delay must
  be at least 1 minute and at most 365 days, and the calculated local execution
  time remains visible before saving.
- Tasks are persisted in `scheduled_account_actions`; they must not depend on an
  open browser, an in-memory timer, `temp_unschedulable_until`, or the scheduled
  account-test tables.
- Each account has at most one current scheduled action. Saving again replaces
  the prior task; a pending task can be canceled from the same dialog.
- `enable_and_recover` first calls `RateLimitService.RecoverAccountState` with
  `InvalidateToken: true`, then calls `AdminService.SetAccountSchedulable(true)`.
  This preserves full error/rate-limit/overload/temp/model/runtime-block cleanup.
- `pause` calls `AdminService.SetAccountSchedulable(false)` and does not mutate
  the account's persisted status.
- The runner claims due rows with a database lease. Successful tasks complete;
  failed tasks retain `last_error` and return to pending for a one-minute retry.
  Stale processing leases are reclaimable after service interruption.
- Preserve migration `185_add_scheduled_account_actions.sql`, the account-level
  `GET|PUT|DELETE /api/v1/admin/accounts/:id/scheduled-action` routes, runner
  startup/cleanup wiring, and corresponding backend/frontend tests.

## Scheduled Test Auto-Recover Schedulable

- `scheduled_test_plans.auto_recover_schedulable` (migration `186`) is a per-plan
  toggle that re-enables scheduling (`schedulable=true`) for an account after a
  successful scheduled test proves it is healthy.
- This is separate from `auto_recover` (which clears error/rate-limit state).
  `auto_recover_schedulable` targets accounts manually paused via
  `schedulable=false` — the runner re-enables them once the test passes.
- `ScheduledTestRunnerService.tryReEnableScheduling` checks `account.Schedulable`
  and only calls `SetSchedulable(true)` when currently paused.
- Frontend `ScheduledTestsPanel.vue` exposes the toggle in both the add-plan and
  edit-plan forms with the label `自动启用调度`.
- The add-plan and edit-plan forms keep the raw 5-field Cron input and help,
  and also expose a mouse-driven visual scheduler for minute intervals,
  hourly, daily, and weekly plans. Advanced expressions remain editable in
  custom mode without being rewritten.
- Preserve migration `186_add_scheduled_test_auto_recover_schedulable.sql`,
  the `AutoRecoverSchedulable` field in the `ScheduledTestPlan` struct, the
  handler create/update request structs, and the frontend toggle.
