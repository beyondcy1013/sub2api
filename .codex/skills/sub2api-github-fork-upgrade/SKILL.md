---
name: sub2api-github-fork-upgrade
description: Use when upgrading sub2api from upstream while preserving local behavior and fork history.
---

# sub2api GitHub Fork Upgrade

## Authority

Read these files completely before fetching or merging; they override this skill:

- `/home/third_party/sub2api/AGENTS.md`
- `/home/third_party/sub2api/docs/UPGRADE_RUNBOOK.md`
- `/home/third_party/sub2api/docs/LOCAL_UPGRADE_CUSTOMIZATIONS.md`

Use `/home/third_party/sub2api` as the only source, merge, test, and build repo.
Treat `/home/third_party/sub2freeApi` only as the free runtime/config directory
and a historical recovery source. Never merge or build it during an upgrade.

| Profile | Service | Port | PostgreSQL | Redis | Binary |
| --- | --- | --- | --- | --- | --- |
| main | `sub2api.service` | 18381 | `sub2api` | DB 0 | `/home/third_party/bin/sub2api/sub2api` |
| free | `sub2freeApi.service` | 18382 | `sub2freeApi` | DB 1, prefix `sub2freeApi` | `/home/third_party/bin/sub2freeApi/sub2freeApi` |

## Prepare Safely

Run the deterministic preparation script from the canonical repo:

```bash
cd /home/third_party/sub2api
bash .codex/skills/sub2api-github-fork-upgrade/scripts/upgrade-from-fork.sh
```

The script:

1. creates verified bundles, backup branches, patches, local-file archives, and
   runtime baselines for both repositories;
2. stashes canonical tracked and untracked work, while archiving ignored policy
   and skill files separately;
3. fetches `upstream/main` and tags through the working `127.0.0.1:7890` proxy;
4. starts `git merge --no-ff --no-commit upstream/main`;
5. removes sponsor blocks from all three README files when the merge is clean;
6. stops before commit, worktree restore, tests, deployment, or push.

Never use reset, clean, rebase, force push, the WebUI updater, or an automatic
push-before-tests script.

## Resolve In Two Layers

Resolve only the upstream merge first. Combine each conflicting block; never
select an entire file as ours/theirs. Then run:

```bash
bash scripts/remove-readme-sponsors.sh
bash scripts/remove-readme-sponsors.sh --check
git diff --check
rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend/src docs README* || true
git add <resolved-files>
git commit -m 'merge: upgrade upstream to vX.Y.Z'
```

Only after the merge commit, locate the saved stash by the OID recorded in
`upgrade-state.env`, pop that exact stash with `--index`, and resolve the local
worktree layer. Run the README policy again after restoring it. Compare the
restored status/file list with the pre-upgrade snapshot before continuing.

## Reusable Conflict Rules

- `config.go`: keep both `redis.username` and `redis.scheduler_key_prefix`;
  keep deployment-profile normalization and upstream trusted-proxy/client-IP
  normalization. Register profile-dependent keys in `setDefaults` so upstream's
  environment-reachability audit still passes.
- `scheduler_cache.go`: adopt upstream's independent `LastUsedAt` side key, but
  route account, metadata, side-key, snapshot-prefix, lock, and bucket keys
  through the cache instance prefix. Free keys must remain under
  `sub2freeApi:`. Preserve newer side-key values during snapshot rebuilds.
- `AccountsView.vue`: adopt upstream Teleport/floating-panel positioning while
  retaining enhanced import, free balance settings, local account columns,
  trash/staging actions, direct text actions, and the fixed 176px truncated
  name cell. Do not restore the account-name hover tooltip.
- i18n conflicts: union upstream navigation/accessibility keys with local
  operation keys; do not choose one side wholesale.
- Migration prefixes are not globally unique: the runner keys migrations by
  full filename. Keep both local `185_add_scheduled_account_actions.sql` and
  upstream `185_group_reasoning_effort_policy.sql`; never rename an already
  deployed local migration merely to remove a shared numeric prefix.
- generated wiring: preserve every upstream constructor parameter and both
  deployment-profile dependencies. Regenerate only with the project's existing
  generator if tests prove the generated file stale.

The project customization document is the complete behavior checklist; the
rules above are only high-frequency merge recipes.

## README Advertisement Policy

Keep upstream functional docs, security notices, license text, and project
credits. Exclude only the sponsor advertisement sections in `README.md`,
`README_CN.md`, and `README_JA.md`:

```bash
bash scripts/remove-readme-sponsors.sh
bash scripts/remove-readme-sponsors.sh --check
```

The unified build script also runs `--check`, so a later sponsor update cannot
silently ship.

Preserve the local enhanced-edition highlights near the top of all three
README files. They must continue describing verified lowest-cost/liveness
scheduling, sticky-session concurrency spillover, 5h/7d quota and cost views,
account lifecycle tools, scheduled recovery, and isolated main/free profiles.
Keep the translations aligned and retain the warning that this customized fork
must use the source-based runbook instead of the WebUI binary updater.

## Verify

Use pnpm 9 and run focused tests while resolving. Before deployment, require:

```bash
cd /home/third_party/sub2api/backend
go test -tags unit ./... -count=1

cd /home/third_party/sub2api/frontend
node /home/root/.npm/_npx/8959f4e966f464e2/node_modules/pnpm/bin/pnpm.cjs vitest run
node /home/root/.npm/_npx/8959f4e966f464e2/node_modules/pnpm/bin/pnpm.cjs typecheck
```

Also require the focused commands in `AGENTS.md`, `git diff --check`, the
conflict-marker scan, README `--check`, and an audit of every local contract.
Commit the restored/customized work before building so the deployed snapshot
is reproducible.

## Build And Deploy Through webClx

Read and use `webclx-compile-and-deploy`. Submit one deploy request that runs:

- compile command: `bash scripts/build-unified-release.sh`;
- required artifact: `backend/bin/sub2api-unified.new`;
- install command: `bash scripts/deploy-unified-release.sh` with the recorded
  old shared SHA and both baseline PIDs;
- audit paths: both installed binary paths.

The build script freezes one coherent source snapshot, runs the complete
backend/frontend suites, builds the frontend once, and builds one `-tags embed`
binary. The deploy script installs those exact bytes to main first and free
second, verifies each profile/database/Redis/port boundary, and rolls back only
the affected service on failure.

If webClx returns `queued: true`, wait for its callback; do not poll or submit a
duplicate request.

## Live Proof And Push

After the callback, independently verify both units, ports, HTTP roots, process
profiles, database/Redis environment, startup logs, and these four hashes:

```bash
MAIN_PID=$(systemctl show -p MainPID --value sub2api.service)
FREE_PID=$(systemctl show -p MainPID --value sub2freeApi.service)
sha256sum /home/third_party/bin/sub2api/sub2api \
  /home/third_party/bin/sub2freeApi/sub2freeApi \
  "/proc/$MAIN_PID/exe" "/proc/$FREE_PID/exe"
```

All four hashes must match. Push only canonical `main` after live verification:

```bash
git push origin main
git ls-remote origin refs/heads/main
git rev-parse HEAD
```

The remote SHA must equal local `HEAD`. Do not update the historical
`origin/sub2freeApi` branch from the legacy repository. Keep upgrade bundles,
backup branches, patches, archives, and binary backups until the live and remote
checks pass. Tell the operator to hard-refresh the browser after deployment.
