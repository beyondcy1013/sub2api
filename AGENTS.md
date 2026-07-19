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

## Deploy Checklist

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

- Admin account credentials intentionally display `credentials.api_key` in plaintext for account edit/inspection workflows.
- Do not add `api_key` back to `SensitiveCredentialKeys`; OAuth tokens, session keys, cookies, AWS secrets, service account JSON, and private keys must remain redacted.
- `MergePreservingSensitiveCreds` must preserve an existing `api_key` when an older or partial frontend update omits it.
- `frontend/src/components/account/EditAccountModal.vue` should preload `credentials.api_key` and render API key inputs as `type="text"` for account API key fields.
- **Account management table column layout** (`frontend/src/views/admin/AccountsView.vue` `allColumns`):
  - The leading order is `选择` -> `名称` -> `容量` -> `状态` -> `调度` -> `用量窗口` -> `平台/类型`.
  - After `创建时间`, keep `今日费用` -> `分组` (when visible) -> `余额` -> `5h请求` -> `5h Token` -> `7d请求` -> `7d Token` -> `窗口总费用`.
  - The ending order is `过期时间` -> `备注` -> `账号ID` -> `上游声明费率` -> `操作`.
  - `名称` (name) column has explicit `width: '126px'`.
  - The selection column has explicit `width: '36px'`, and `DataTable.vue` keeps `--select-col-width` at `36px`.
  - `状态` (status) has explicit `width: '80px'`.
  - Table headers, labels, and sort indicators remain single-line and non-shrinking.
  - Fixed-width columns apply `width`, `minWidth`, and `maxWidth`.
  - The first and last table cells use `4px` outer padding so the table has no unnecessary edge whitespace.
  - Non-final columns retain 1px vertical separators in light and dark mode.
  - `平台/类型` (platform_type) column has explicit `width: '170px'` (absorbs the freed width).
  - `账号ID` (id) column has explicit `width: '130px'`.
  - The `Column` interface in `frontend/src/components/common/types.ts` has an optional `width?: string` property.
  - The `DataTable` component in `frontend/src/components/common/DataTable.vue` applies `column.width` as `width` + `minWidth` inline style on `<th>` and `<td>` elements.
  - Do NOT revert these columns to their upstream positions or remove the width properties.
- Preserve every local commit and dirty file during upgrades. Follow [docs/UPGRADE_RUNBOOK.md](docs/UPGRADE_RUNBOOK.md) and audit [docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md](docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md); do not use the WebUI binary updater for this customized build.
- **OpenAI sticky-session concurrency spillover must be preserved**:
  - A historical `session_hash` binding is honored only while the bound account can acquire a real `account.Concurrency` slot.
  - When that account is full, route the current connection through the normal same-group selection path to another available account, even if the session remains historically bound to the full account.
  - Do not rewrite the historical sticky binding for a one-connection overflow; if every eligible account is full, retain the normal bounded wait-plan fallback.
  - Concurrency-full spillover is mandatory even when TTFT/error health escape is disabled. Strict, non-movable `previous_response_id` affinity remains unchanged.
  - Preserve the implementation in `openai_gateway_scheduling.go` and `openai_account_scheduler.go` plus the spillover regression tests in their corresponding `*_test.go` files.
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
  `7d请求`, `7d Token`, `5h使用比例`, `5h重置`, `7d使用比例`, `7d重置`, and
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
- Focused regression verification:

```bash
cd /home/third_party/sub2api/frontend
pnpm vitest run \
  src/components/admin/account/__tests__/enhancedImport.spec.ts \
  src/components/admin/account/__tests__/EnhancedImportDataModal.spec.ts
```

## Account Recycle / Trash Feature

- **Bulk account editing is direct and change-tracked**:
  - Value controls in `BulkEditAccountModal.vue` remain usable before their field checkbox is selected.
  - Changing a value automatically selects only that field for submission; untouched field defaults must not enter the bulk-update payload.
  - Field checkboxes remain available for explicit operations whose desired value equals the form default, including clearing a proxy, groups, mappings, or other existing account values.
- Accounts can be "recycled" (moved to trash) via `extra.recycled = true` in the JSONB `extra` field. This does NOT use soft-delete (`deleted_at`).
- `accountListFilteredQuery` in `account_repo.go` accepts a `recycled bool` param: `true` shows only recycled accounts, `false` (default) excludes them.
- All callers of `ListAccounts`, `ListWithFilters`, `ListAllWithFilters` must pass the `recycled bool` as the final parameter.
- Backend routes: `POST /api/v1/admin/accounts/:id/recycle` and `POST /api/v1/admin/accounts/:id/restore`.
- Frontend: `AccountTableActions.vue` has a trash toggle button; `AccountsView.vue` shows Recycle/Restore buttons in the action column depending on mode.
- sub2freeApi has an additional `clone` function in `accounts.ts` that sub2api does not — re-add it when syncing files.
- Status/Groups/Capacity table cells use plain text (text-color classes only), NOT badge/card styling. See `references/account-table-column-layout.md`.
- **Usage auto-load skips non-active accounts**: `AccountUsageCell.vue` `shouldAutoLoadUsageOnMount` must gate on `props.account.status === 'active'`. Accounts with status `inactive` or `error` do NOT auto-fetch `/usage` on page mount, avoiding useless upstream queries against known-unavailable accounts. Manual refresh (via `usageManualRefreshToken`) remains unaffected for all statuses.
- **OpenAI quota auto-pause is displayed as quota limiting**: admin account responses expose the request-time `quota_rate_limit` decision from the same service logic used by scheduling. `AccountStatusIndicator.vue` shows `额度限流` and its reset time, and an enabled scheduling toggle uses an amber limited state while the account is excluded. Do not persist this derived state into `status`, `schedulable`, or `rate_limit_reset_at`; window recovery must remain automatic.

## Account Actions And Active Sticky-Session Reassignment

- New/imported account form initialization and reset use `concurrency: 4`; do not restore upstream `10`.
- New-account form initialization and reset select the last available proxy and the first available group. If proxy/group props arrive after the modal opens, fill only still-empty selections; never overwrite an operator's existing choice. Empty candidate lists remain unassigned.
- `AccountActionMenu.vue` always displays `恢复状态` for every account.
- The action menu keeps `w-[7.8rem] max-h-[calc(100vh-1rem)] overflow-y-auto`; `AccountsView.vue` keeps a `125` px width estimate and a `320` px height estimate for viewport positioning.
- `迁入粘性会话` appears for active, schedulable OpenAI target accounts and defaults to bindings active in the last 5 minutes. Allowed windows are 1, 5, 15, and 60 minutes.
- The dialog shows recent/all counts and anonymous recent suffixes. Move at most 100, newest first, and revalidate the activity window on the backend.
- Activity age is derived from configured sticky TTL minus Redis PTTL. Migrations use compare-and-set plus `SET ... KEEPTTL`.
- Only current 16-character lowercase-hex `session_hash` keys move. Ignore legacy 64-character copies and never move `response:` / `previous_response_id` continuation bindings.
- `Concurrency limit exceeded for user` is a local sub2api caller-concurrency timeout, not an upstream provider message. Sticky concentration can consume caller slots while requests wait on one upstream account, so moving recent bindings to spare capacity can help. Do not direct this operation to `用户管理`.
- Preserve the API routes and tests documented in `.codex/skills/sub2api-account-modal-enhancer/references/sticky-session-reassignment.md`.
