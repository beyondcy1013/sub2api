# Unified Dual-Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate sub2api and sub2freeApi into `/home/third_party/sub2api` as one source tree and one embedded build artifact while retaining two isolated systemd services with profile-specific behavior.

**Architecture:** The canonical tree contains the union of both repositories. Each process loads `deployment.profile` (`main` or `free`) and derives immutable runtime capabilities from it; public settings inject those capabilities into the shared frontend before mount. One `-tags embed` build is installed byte-for-byte to the two existing binary paths, while systemd users, working directories, PostgreSQL databases, Redis databases/prefixes, ports, and rollback paths remain isolated.

**Tech Stack:** Go 1.26, Gin, Viper, Wire, PostgreSQL migrations, Redis, Vue 3, Pinia, TypeScript, Vitest, pnpm 9, systemd.

## Global Constraints

- Canonical source path is `/home/third_party/sub2api`; `/home/third_party/sub2freeApi` remains the free service runtime/config directory and historical recovery source.
- `sub2api.service` remains on port `18381`, PostgreSQL database `sub2api`, Redis DB `0`, runtime user `sub2api`.
- `sub2freeApi.service` remains on port `18382`, PostgreSQL database `sub2freeApi`, Redis DB `1`, scheduler key prefix `sub2freeApi`, runtime user `sub2freeapi`.
- Preserve every local commit and every staged, unstaged, ignored-policy, and untracked file from both repositories.
- Preserve plaintext admin `credentials.api_key`, sensitive credential redaction, recycle/restore, account table layout, usage auto-load guard, concurrency default `4`, routing defaults, sticky spillover, sticky reassignment, free balance checks, and free error prefixes.
- Do not use the WebUI updater, `git reset --hard`, `git clean`, rebase, force push, or direct-overwrite deployment.
- Build the frontend first, then build the Go server once with `CGO_ENABLED=0` and `-tags embed`.
- Restart and verify one service at a time; a failed service deployment rolls back only that service.

---

### Task 1: Import The Free History Into The Canonical Repository

**Files:**
- Merge: all tracked files reachable from `/home/third_party/sub2freeApi` `main`
- Preserve: both repository backup bundles under `/home/third_party/upgrade-backups`
- Preserve/restore: pre-existing dirty and untracked files recorded by each backup `status.txt`, patches, and tar archive

**Interfaces:**
- Consumes: common upstream base `a2f802d409e0c175dedaf29d67e2f5d0b041e5e9`
- Produces: canonical `main` history containing both current repository heads and a compilable union tree

- [ ] **Step 1: Record clean merge inputs and baseline tests**

Run both repositories' focused credential, scheduling, error-prefix, balance-check, and frontend tests using their current build-tag rules. Record exact failures without changing source.

- [ ] **Step 2: Temporarily preserve the canonical dirty workspace**

Create a uniquely named stash including non-ignored untracked files after the already-verified external bundle. Keep ignored policy files in place.

- [ ] **Step 3: Fetch and merge the local free branch**

```bash
git fetch /home/third_party/sub2freeApi main:refs/remotes/consolidation/free-main
git merge --no-ff refs/remotes/consolidation/free-main
```

Resolve each conflict as a union: retain main sticky scheduling/reassignment and current account UI behavior; retain free balance check, security/ops additions, `clienterror`, scheduler prefix, clone support, and error writers. Never take an entire conflicted file blindly.

- [ ] **Step 4: Verify the union baseline**

Run `git diff --check`, scan for conflict markers, run `go test` on changed backend packages, then run frontend typecheck. Commit the merge only after the union compiles.

### Task 2: Add An Immutable Runtime Deployment Profile

**Files:**
- Create: `backend/internal/config/deployment_profile.go`
- Create: `backend/internal/config/deployment_profile_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/pkg/clienterror/prefix.go`
- Modify: `backend/internal/pkg/clienterror/prefix_test.go`

**Interfaces:**
- Produces: `config.DeploymentConfig`, `Config.DeploymentProfile()`, `Config.RuntimeCapabilities()`, and `clienterror.Configure(profile string)`
- Capability fields: `balance_check`, `sticky_session_reassignment`, and `branded_errors`

- [ ] **Step 1: Write failing profile tests**

Cover an omitted profile defaulting to `main`, `DEPLOYMENT_PROFILE=free`, invalid profile rejection, free/main capability matrices, and profile-dependent default balance-check enablement.

- [ ] **Step 2: Run the profile tests and verify RED**

```bash
go test -tags unit ./internal/config ./internal/pkg/clienterror -run 'TestDeployment|TestClientErrorProfile|TestLoadDefaultBalanceCheck' -count=1
```

- [ ] **Step 3: Implement the profile model**

Use `main` and `free` as the only accepted profile values. Keep the capability derivation in the config package so handlers, services, and UI serialization do not duplicate product-name conditionals.

- [ ] **Step 4: Configure client-visible errors at startup**

The main profile keeps unprefixed messages with source `sub2api`; the free profile keeps `【sub2freeApi限制】` / `【上游错误】` with source `sub2freeApi`. Configure once after validated config load and before application initialization.

- [ ] **Step 5: Run the same tests and verify GREEN**

Commit the validated runtime-profile implementation.

### Task 3: Expose Runtime Capabilities Through Public Settings

**Files:**
- Modify: `backend/internal/service/settings_view.go`
- Modify: `backend/internal/service/setting_public.go`
- Modify: `backend/internal/service/setting_service_public_test.go`
- Modify: `backend/internal/handler/dto/settings.go`
- Modify: `backend/internal/handler/dto/public_settings_injection_schema_test.go`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/stores/__tests__/app.spec.ts`

**Interfaces:**
- Produces public JSON fields: `deployment_profile`, `balance_check_enabled`, `sticky_session_reassignment_enabled`, and `branded_errors_enabled`
- The fields must be identical in `/api/v1/settings/public` and injected `window.__APP_CONFIG__`

- [ ] **Step 1: Write failing service and injection-schema tests**

Assert both profile matrices and exact injection/API key parity.

- [ ] **Step 2: Run the focused tests and verify RED**

```bash
go test -tags unit ./internal/service ./internal/handler/dto -run 'TestSettingService_GetPublicSettings_ExposesDeployment|TestPublicSettingsInjection' -count=1
```

- [ ] **Step 3: Add fields to the service model, injection payload, DTO, and TypeScript type**

Derive values only from the validated config; do not persist them in the settings database.

- [ ] **Step 4: Run the same tests and verify GREEN**

Commit the public capability contract.

### Task 4: Gate Shared Frontend Features By Capability

**Files:**
- Modify: `frontend/src/utils/featureFlags.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/admin/account/AccountActionMenu.vue`
- Modify: `frontend/src/components/admin/account/__tests__/AccountActionMenu.spark_shadow.spec.ts`
- Create/modify: focused router/sidebar capability tests under their existing `__tests__` directories

**Interfaces:**
- Consumes: injected `PublicSettings` capability fields
- Produces: free-only balance-check navigation and main-only sticky-session reassignment action without a second frontend build

- [ ] **Step 1: Write failing frontend capability tests**

Assert balance settings are visible/routable only when `balance_check_enabled` is true and sticky reassignment is visible only when `sticky_session_reassignment_enabled` is true.

- [ ] **Step 2: Run focused Vitest and verify RED**

```bash
pnpm vitest run src/components/admin/account/__tests__/AccountActionMenu.spark_shadow.spec.ts src/router/__tests__/feature-access.spec.ts
```

- [ ] **Step 3: Implement capability gating using the existing public-settings flag registry**

Use opt-in behavior so old/stale injected settings cannot expose the wrong product feature.

- [ ] **Step 4: Run focused Vitest and verify GREEN**

Commit the unified frontend behavior.

### Task 5: Preserve The Full Product Superset

**Files:**
- Verify/resolve: Wire providers and generated wiring under `backend/cmd/server/` and `backend/internal/{handler,repository,service}/wire.go`
- Verify: main sticky-session scheduler/admin files and tests
- Verify: free balance-check handlers/service/page/UI and tests
- Verify: free ops/security migrations `183` and `184`
- Verify: account credential, account table, recycle/restore, clone/duplicate, rate-limit recovery, and usage auto-load files/tests

**Interfaces:**
- Produces: one application graph in which profile-disabled workers are inert and independent profile-enabled features retain their current behavior

- [ ] **Step 1: Regenerate Wire if provider signatures changed**

Run the repository's existing Wire generation command and inspect generated diffs.

- [ ] **Step 2: Run focused main and free behavior suites from the canonical tree**

Use `-tags unit` consistently because the canonical tree retains the sub2api unit-test convention.

- [ ] **Step 3: Run migration and repository tests**

Confirm the union migration set is additive and both existing databases can advance independently.

- [ ] **Step 4: Commit any integration-only fixes**

### Task 6: Document And Configure Two Runtime Services

**Files:**
- Modify: `AGENTS.md`
- Modify: `docs/UPGRADE_RUNBOOK.md`
- Modify: `docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md`
- Create: `docs/UNIFIED_DUAL_SERVICE_DEPLOYMENT.md`
- Modify: `/etc/systemd/system/sub2api.service`
- Modify: `/etc/systemd/system/sub2freeApi.service`
- Modify: `/home/third_party/sub2freeApi/AGENTS.md`

**Interfaces:**
- Main systemd environment: `DEPLOYMENT_PROFILE=main`
- Free systemd environment: `DEPLOYMENT_PROFILE=free`
- Both `ExecStart` paths remain unchanged; their files must have identical SHA after deployment

- [ ] **Step 1: Add profile environment to both units and canonical documentation**

- [ ] **Step 2: Mark the free repository as runtime/config plus historical recovery source**

- [ ] **Step 3: Run `systemd-analyze verify` and `systemctl daemon-reload`**

Do not restart either service yet.

### Task 7: Verify, Build Once, Deploy Twice, And Prove Isolation

**Files:**
- Build: `frontend/dist`
- Build once: `backend/bin/sub2api-unified.new`
- Install atomically: `/home/third_party/bin/sub2api/sub2api`
- Install the same bytes atomically: `/home/third_party/bin/sub2freeApi/sub2freeApi`

**Interfaces:**
- Produces two active services with equal executable SHA and profile-specific runtime behavior

- [ ] **Step 1: Run canonical backend and frontend verification**

Run focused customization suites, complete backend tests, full Vitest, typecheck, and production frontend build. Investigate only failures introduced by this work; document known upstream failures separately.

- [ ] **Step 2: Build one embedded binary and verify metadata**

Confirm `CGO_ENABLED=0`, `tags=embed`, source version `0.1.160`, executable size, and SHA.

- [ ] **Step 3: Deploy and verify `sub2api.service`**

Record baseline PID/SHA, atomically install, restart only `sub2api.service`, then verify port `18381`, HTTP, version, DB `sub2api`, Redis DB `0`, main error semantics, sticky capability, startup logs, and disk/running SHA.

- [ ] **Step 4: Deploy and verify `sub2freeApi.service`**

Recheck the shared artifact SHA, atomically install the same bytes, restart only `sub2freeApi.service`, then verify port `18382`, HTTP, version, DB `sub2freeApi`, Redis DB `1`, scheduler prefix `sub2freeApi`, free error prefix, balance settings API/UI, startup logs, and disk/running SHA.

- [ ] **Step 5: Prove the two installed and running executables are byte-identical**

Compare both disk SHA values and `/proc/<pid>/exe` SHA values. Preserve per-service old-binary backups until post-deploy checks pass.

- [ ] **Step 6: Restore the pre-existing user workspace state**

Apply the canonical stash with index state, reconcile overlapping files by preserving both the user changes and the unified implementation, restore any non-ignored untracked files, and compare against the external backup status/patch inventory.

- [ ] **Step 7: Final audit**

Run `git diff --check`, conflict-marker scans, service isolation checks, and report exact test/deploy evidence. Keep the external backup directories if any verification remains incomplete.
