# Local Upgrade Customizations

This repository carries local `sub2freeApi` behavior that must be preserved when downloading or merging upgraded upstream source.

## Complete State Preservation

The customization list below is a behavioral verification checklist, not a file allowlist. Preserve all local commits and all working-tree state before upgrading:

```bash
TS=$(date +%Y%m%d-%H%M%S)
BACKUP=/home/third_party/upgrade-backups/sub2freeApi-$TS
mkdir -p "$BACKUP"
git branch "backup-pre-upgrade-$TS" HEAD
git bundle create "$BACKUP/repository.bundle" --all
git log --reverse --oneline origin/main..HEAD > "$BACKUP/local-commits.txt"
git status --porcelain=v1 > "$BACKUP/status.txt"
git diff --binary > "$BACKUP/worktree.patch"
git diff --cached --binary > "$BACKUP/index.patch"
{
  git ls-files --others --exclude-standard -z
  [ -f AGENTS.md ] && printf 'AGENTS.md\0'
  [ -d .codex/skills ] && find .codex/skills -type f -print0
} | sort -zu > "$BACKUP/local-files.list"
if [ -s "$BACKUP/local-files.list" ]; then
  tar --null -czf "$BACKUP/local-files.tar.gz" -T "$BACKUP/local-files.list"
fi
STASH_CREATED=0
if ! git diff --quiet || ! git diff --cached --quiet; then
  git stash push -m "pre-upgrade-$TS"
  STASH_CREATED=1
fi
git fetch origin
git merge --no-ff origin/main
if [ "$STASH_CREATED" -eq 1 ]; then
  git stash pop --index
fi
```

Never use `git reset --hard`, `git clean`, rebase away local commits, or the WebUI one-click updater. The WebUI updater installs an official binary and does not preserve source customizations.

## Preserve These Behaviors

- Admin account management shows account identifiers in plaintext.
- Admin account management shows upstream `credentials.api_key` fully in plaintext.
- Backend admin account DTOs do not redact `api_key`, but still redact OAuth tokens, cookies, private keys, refresh tokens, and similar secrets.
- Account updates preserve an existing `api_key` when the update payload omits it.
- Client-facing errors include a source prefix:
  - `【sub2freeApi限制】` for local/auth/quota/concurrency/config limits.
  - `【上游错误】` for upstream-originated failures.
- `/responses` streaming errors emit protocol-compatible `response.failed` events with prefixed messages.
- The free service deploys only as `sub2freeApi.service` on port `18382`.
- Account filters remain hidden by default behind the Filters toggle.
- The sidebar remains 154px expanded and 67px collapsed.
- Account table widths remain: select 36px, name 126px, status 80px, balance 70px, id 130px, platform/type 170px.
- Table headers remain single-line/non-shrinking; fixed widths use width/minWidth/maxWidth.
- Table outer edge padding remains 4px and non-final columns retain vertical separators.

## Files To Check After Upstream Refresh

Account/API Key plaintext:

- `backend/internal/service/account_credentials_redact.go`
- `backend/internal/handler/dto/credentials_redact.go`
- `backend/internal/handler/dto/account_mapper_redact_test.go`
- `frontend/src/components/account/EditAccountModal.vue`
- `frontend/src/views/admin/AccountsView.vue`
- `frontend/src/components/common/DataTable.vue`
- `frontend/src/components/common/__tests__/DataTable.spec.ts`
- `frontend/src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts`
- `frontend/src/components/admin/account/AccountTableActions.vue`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/layout/AppLayout.vue`
- `frontend/src/types/index.ts`
- `frontend/src/i18n/locales/en.ts`
- `frontend/src/i18n/locales/zh.ts`

Error prefixes:

- `backend/internal/pkg/clienterror/`
- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/handler/gateway_handler.go`
- `backend/internal/handler/gateway_handler_responses.go`
- `backend/internal/handler/gateway_handler_chat_completions.go`
- `backend/internal/handler/gemini_v1beta_handler.go`
- `backend/internal/handler/stream_error_event.go`
- `backend/internal/server/middleware/middleware.go`
- `backend/internal/server/middleware/api_key_auth_google.go`
- Direct service error writers under `backend/internal/service/`.

## Useful Searches

```bash
rg -n "SensitiveCredentialKeys|PreserveOnMissingCredentialKeys|api_key|apiKeyPlainVisibleHint|type=\"password\"|editApiKey" backend/internal frontend/src
rg -n "clienterror|sub2freeApi限制|上游错误|writeResponsesFailedSSE|AbortWithError|abortWithGoogleError|closeOpenAIClientWS" backend/internal
```

## Focused Verification

From `backend/`:

```bash
go test ./internal/pkg/clienterror ./internal/server/middleware ./internal/handler ./internal/handler/dto ./internal/service -run 'TestPrefix|TestApiKeyAuth|TestAPIKeyAuth|TestOpenAIHandleStreamingAwareError|TestGatewayHandleStreamingAwareError|TestOpenAIEnsureForwardErrorResponse|TestGatewayEnsureForwardErrorResponse|TestOpenAIRecoverResponsesPanic|TestMapResponsesErrorCode|TestConcurrencyErrorResponse|TestRedactCredentials|TestAccountFromServiceShallow|TestMergePreservingSensitiveCreds|TestIsSensitiveCredentialKey|Test.*ErrorPassthrough|TestGatewayHandleErrorResponse|TestOpenAIHandleErrorResponse|TestGeminiWriteGeminiMappedError'
```

From `frontend/`:

```bash
pnpm vitest run src/components/account/__tests__/EditAccountModal.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
pnpm typecheck
pnpm build
```

## Deploy

```bash
cd /home/third_party/sub2freeApi/backend
CGO_ENABLED=0 go build -tags embed -o /home/third_party/bin/sub2freeApi/sub2freeApi ./cmd/server/
systemctl restart sub2freeApi.service
systemctl status sub2freeApi.service --no-pager
ss -ltnp | rg ':18382'
curl -fsS http://127.0.0.1:18382/ >/dev/null
curl -sS -X POST http://127.0.0.1:18382/v1/responses -H 'Content-Type: application/json' -d '{"model":"gpt-5","input":"hi"}'
```

The missing-auth response should contain `【sub2freeApi限制】`.
